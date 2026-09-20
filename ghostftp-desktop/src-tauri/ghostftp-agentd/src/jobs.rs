//! Detached background jobs on the daemon (Plan 10 Phase 4, agent arm).
//!
//! A [`JobStore`] spawns a command in a fresh child, streams its stdout/stderr
//! into capped in-memory buffers, and records the exit code when it finishes —
//! the daemon-target analogue of the SSH `~/.ghostftp/jobs/<id>` dir. It retires the
//! `nohup … & ; tail -f log` loop for multi-minute work that would blow the
//! `Exec` timeout: [`Request::ExecStart`](ghostftp_agent_proto::msg::Request) returns
//! at once with a job id, and the controller polls
//! [`ExecPoll`](ghostftp_agent_proto::msg::Request) until it's done.
//!
//! One store is shared across every connection (it lives behind an `Arc` on the
//! [`Daemon`](crate::server::Daemon)), so a job started on one channel is still
//! pollable after the controller re-dials. Jobs live for the daemon's lifetime —
//! a daemon restart forgets in-flight jobs (a later poll gets `not_found`), the
//! one gap versus the SSH arm's on-disk dir. Finished jobs are pruned on a TTL,
//! and the store is bounded, so a long-lived daemon can't accumulate them.

use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};
use tokio::io::AsyncReadExt;
use tokio::sync::oneshot;

/// Keep finished jobs for this long so a poll shortly after completion still
/// sees the exit code + output, then reap them.
const JOB_TTL: Duration = Duration::from_secs(60 * 60);
/// Hard ceiling on tracked jobs; over it, the oldest finished ones are dropped.
const MAX_JOBS: usize = 256;

/// A byte buffer that stops growing at `cap` and remembers it was clipped, so a
/// runaway job can't balloon the daemon's memory (mirrors the exec output cap).
struct CappedBuf {
    data: Vec<u8>,
    cap: usize,
    truncated: bool,
}

impl CappedBuf {
    fn new(cap: usize) -> Self {
        Self {
            data: Vec::new(),
            cap,
            truncated: false,
        }
    }
    fn push(&mut self, bytes: &[u8]) {
        if self.data.len() >= self.cap {
            self.truncated = true;
            return;
        }
        let room = self.cap - self.data.len();
        if bytes.len() > room {
            self.data.extend_from_slice(&bytes[..room]);
            self.truncated = true;
        } else {
            self.data.extend_from_slice(bytes);
        }
    }
}

/// Liveness of a job, updated once by the waiter task when the child exits.
struct JobState {
    running: bool,
    exit_code: Option<i32>,
    finished_at: Option<Instant>,
}

struct Job {
    stdout: Arc<Mutex<CappedBuf>>,
    stderr: Arc<Mutex<CappedBuf>>,
    state: Arc<Mutex<JobState>>,
    /// Taken (once) to signal a kill; `None` after the first kill.
    kill: Mutex<Option<oneshot::Sender<()>>>,
}

/// A status snapshot handed back to a poller.
pub struct JobStatus {
    pub running: bool,
    pub exit_code: Option<i32>,
    pub stdout: String,
    pub stderr: String,
    pub truncated: bool,
}

/// Tracks detached jobs by id. Cheap to `clone` via the `Arc` on `Daemon`.
#[derive(Default)]
pub struct JobStore {
    jobs: Mutex<HashMap<String, Job>>,
}

impl JobStore {
    pub fn new() -> Self {
        Self::default()
    }

    /// Spawn `command` in the daemon's native shell under `job_id`, capturing up
    /// to `max_bytes` of each stream. Returns the spawn error if the child can't
    /// start; otherwise the job runs to completion in the background.
    pub fn start(&self, job_id: &str, command: &str, max_bytes: usize) -> std::io::Result<()> {
        let mut cmd = crate::ops::build_command(command);
        cmd.stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped())
            .stdin(std::process::Stdio::null());
        configure_process_tree(&mut cmd);
        let mut child = cmd.spawn()?;
        let out_pipe = child.stdout.take();
        let err_pipe = child.stderr.take();

        let stdout = Arc::new(Mutex::new(CappedBuf::new(max_bytes)));
        let stderr = Arc::new(Mutex::new(CappedBuf::new(max_bytes)));
        let state = Arc::new(Mutex::new(JobState {
            running: true,
            exit_code: None,
            finished_at: None,
        }));

        // Drain each pipe into its capped buffer; the tasks end when the child
        // closes the stream (i.e. on exit), so joining them means "all output
        // captured".
        let out_task = out_pipe.map(|p| {
            let buf = stdout.clone();
            tokio::spawn(pump(p, buf))
        });
        let err_task = err_pipe.map(|p| {
            let buf = stderr.clone();
            tokio::spawn(pump(p, buf))
        });

        let (kill_tx, kill_rx) = oneshot::channel::<()>();
        let waiter_state = state.clone();
        tokio::spawn(async move {
            let status = tokio::select! {
                s = child.wait() => s,
                _ = kill_rx => {
                    terminate_process_tree(&mut child).await;
                    child.wait().await
                }
            };
            // Flush the last of the output before flipping to finished, so a poll
            // that sees running=false also sees the complete capture.
            if let Some(t) = out_task {
                let _ = t.await;
            }
            if let Some(t) = err_task {
                let _ = t.await;
            }
            let code = status.ok().and_then(|s| s.code());
            let mut st = waiter_state.lock().unwrap();
            st.running = false;
            st.exit_code = code;
            st.finished_at = Some(Instant::now());
        });

        let mut jobs = self.jobs.lock().unwrap();
        Self::prune(&mut jobs);
        jobs.insert(
            job_id.to_string(),
            Job {
                stdout,
                stderr,
                state,
                kill: Mutex::new(Some(kill_tx)),
            },
        );
        Ok(())
    }

    /// Snapshot a job's status + captured output. `None` if the id is unknown.
    pub fn poll(&self, job_id: &str) -> Option<JobStatus> {
        let jobs = self.jobs.lock().unwrap();
        let job = jobs.get(job_id)?;
        let st = job.state.lock().unwrap();
        let out = job.stdout.lock().unwrap();
        let err = job.stderr.lock().unwrap();
        Some(JobStatus {
            running: st.running,
            exit_code: st.exit_code,
            stdout: String::from_utf8_lossy(&out.data).to_string(),
            stderr: String::from_utf8_lossy(&err.data).to_string(),
            truncated: out.truncated || err.truncated,
        })
    }

    /// Signal a job to die (best-effort). `false` if the id is unknown.
    pub fn kill(&self, job_id: &str) -> bool {
        let jobs = self.jobs.lock().unwrap();
        let Some(job) = jobs.get(job_id) else {
            return false;
        };
        if let Some(tx) = job.kill.lock().unwrap().take() {
            let _ = tx.send(());
        }
        true
    }

    /// Reap finished jobs older than the TTL, and — if still over the ceiling —
    /// the oldest finished ones. Running jobs are always kept.
    fn prune(jobs: &mut HashMap<String, Job>) {
        let now = Instant::now();
        jobs.retain(|_, job| {
            let st = job.state.lock().unwrap();
            match st.finished_at {
                Some(t) => now.duration_since(t) < JOB_TTL,
                None => true,
            }
        });
        if jobs.len() > MAX_JOBS {
            let mut finished: Vec<(String, Instant)> = jobs
                .iter()
                .filter_map(|(k, j)| j.state.lock().unwrap().finished_at.map(|t| (k.clone(), t)))
                .collect();
            finished.sort_by_key(|(_, t)| *t);
            let excess = jobs.len() - MAX_JOBS;
            for (k, _) in finished.into_iter().take(excess) {
                jobs.remove(&k);
            }
        }
    }
}

#[cfg(unix)]
fn configure_process_tree(cmd: &mut tokio::process::Command) {
    use std::os::unix::process::CommandExt;
    cmd.as_std_mut().process_group(0);
}

#[cfg(windows)]
fn configure_process_tree(_cmd: &mut tokio::process::Command) {}

#[cfg(unix)]
async fn terminate_process_tree(child: &mut tokio::process::Child) {
    if let Some(pid) = child.id() {
        // The child is the leader of the process group created above. Killing
        // the group prevents shell-spawned descendants from keeping pipes open
        // and continuing after the user has cancelled the job.
        let rc = unsafe { libc::kill(-(pid as i32), libc::SIGKILL) };
        if rc == 0 {
            return;
        }
    }
    let _ = child.start_kill();
}

#[cfg(windows)]
async fn terminate_process_tree(child: &mut tokio::process::Child) {
    if let Some(pid) = child.id() {
        let _ = tokio::process::Command::new("taskkill")
            .args(["/PID", &pid.to_string(), "/T", "/F"])
            .stdout(std::process::Stdio::null())
            .stderr(std::process::Stdio::null())
            .status()
            .await;
    }
    let _ = child.start_kill();
}

/// Copy a child pipe into its capped buffer until EOF (the child exits) or a
/// read error.
async fn pump<R: tokio::io::AsyncRead + Unpin>(mut pipe: R, buf: Arc<Mutex<CappedBuf>>) {
    let mut chunk = [0u8; 8192];
    loop {
        match pipe.read(&mut chunk).await {
            Ok(0) | Err(_) => break,
            Ok(n) => buf.lock().unwrap().push(&chunk[..n]),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn small_cap() -> usize {
        64 * 1024
    }

    // A detached job runs to completion and its output + exit code are pollable.
    #[tokio::test]
    async fn start_poll_captures_output_and_exit() {
        let store = JobStore::new();
        #[cfg(windows)]
        let cmd = "Write-Output hello-detached";
        #[cfg(not(windows))]
        let cmd = "echo hello-detached";
        store.start("job-1", cmd, small_cap()).unwrap();

        // Poll until it finishes (bounded so a hang fails the test).
        let mut status = None;
        for _ in 0..100 {
            let s = store.poll("job-1").expect("job exists");
            if !s.running {
                status = Some(s);
                break;
            }
            tokio::time::sleep(Duration::from_millis(50)).await;
        }
        let s = status.expect("job finished");
        assert_eq!(s.exit_code, Some(0));
        assert!(
            s.stdout.contains("hello-detached"),
            "stdout: {:?}",
            s.stdout
        );
    }

    // An unknown id polls as None (the bridge maps this to `not_found`).
    #[tokio::test]
    async fn poll_unknown_is_none() {
        let store = JobStore::new();
        assert!(store.poll("nope").is_none());
        assert!(!store.kill("nope"));
    }

    // A long job can be killed and then reports finished (non-zero / no code).
    #[tokio::test]
    async fn kill_stops_a_running_job() {
        let store = JobStore::new();
        #[cfg(windows)]
        let cmd = "Start-Sleep -Seconds 30";
        #[cfg(not(windows))]
        let cmd = "sleep 30";
        store.start("job-k", cmd, small_cap()).unwrap();
        // It's running.
        assert!(store.poll("job-k").unwrap().running);
        assert!(store.kill("job-k"));
        // It stops promptly.
        let mut stopped = false;
        for _ in 0..100 {
            if !store.poll("job-k").unwrap().running {
                stopped = true;
                break;
            }
            tokio::time::sleep(Duration::from_millis(50)).await;
        }
        assert!(stopped, "killed job never went to finished");
    }
}
