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
/// Maximum concurrent active jobs owned by any one paired controller.
const MAX_JOBS_PER_PEER: usize = 32;

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

    /// Admit the job while the store mutex is held, *before* spawning a process.
    /// The lock spans capacity checks, OS spawn and insertion: concurrent clients
    /// cannot oversubscribe the daemon or replace an existing process handle.
    fn admit(jobs: &mut HashMap<String, Job>, job_id: &str) -> std::io::Result<()> {
        use std::io::{Error, ErrorKind};
        if jobs.contains_key(job_id) {
            return Err(Error::new(ErrorKind::AlreadyExists, "job id is already tracked"));
        }
        fn namespace(id: &str) -> &str {
            id.split_once(':').map_or("", |(peer, _)| peer)
        }
        let peer = namespace(job_id);
        let active_for_peer = jobs
            .iter()
            .filter(|(id, job)| {
                namespace(id) == peer
                    && job
                        .state
                        .lock()
                        .unwrap_or_else(std::sync::PoisonError::into_inner)
                        .running
            })
            .count();
        if active_for_peer >= MAX_JOBS_PER_PEER {
            return Err(Error::new(ErrorKind::WouldBlock, "controller's active job quota reached"));
        }
        if jobs.len() >= MAX_JOBS {
            // A completed job may be retired early to make room; never evict
            // or overwrite a running process, even when the store is full.
            let oldest_finished = jobs
                .iter()
                .filter_map(|(id, job)| {
                    job.state
                        .lock()
                        .unwrap_or_else(std::sync::PoisonError::into_inner)
                        .finished_at
                        .map(|at| (id.clone(), at))
                })
                .min_by_key(|(_, at)| *at)
                .map(|(id, _)| id);
            if let Some(id) = oldest_finished {
                jobs.remove(&id);
            } else {
                return Err(Error::new(ErrorKind::WouldBlock, "daemon's active job capacity reached"));
            }
        }
        Ok(())
    }

    /// Spawn `command` in the daemon's native shell under `job_id`, capturing up
    /// to `max_bytes` of each stream. Returns the spawn error if the child can't
    /// start; otherwise the job runs to completion in the background.
    pub fn start(&self, job_id: &str, command: &str, max_bytes: usize) -> std::io::Result<()> {
        let mut jobs = self.jobs.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        Self::prune(&mut jobs);
        Self::admit(&mut jobs, job_id)?;
        // Keep the admission lock through spawn and insertion. If spawn fails,
        // no slot was inserted, and the guard releases the capacity.
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

    /// Terminate only detached processes started by a specific pinned peer.
    /// Revocation may not kill unrelated controllers' jobs.
    pub fn kill_namespace(&self, peer_key: &str) -> usize {
        let prefix = format!("{peer_key}:");
        let matching: Vec<String> = {
            let jobs = self
                .jobs
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner);
            jobs.keys()
                .filter(|key| key.starts_with(&prefix))
                .cloned()
                .collect()
        };
        for id in &matching {
            let _ = self.kill(id);
        }
        matching.len()
    }

    /// Machine-wide kill switch when the owner disables remote execution.
    pub fn kill_all(&self) -> usize {
        let keys: Vec<String> = {
            let jobs = self
                .jobs
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner);
            jobs.keys().cloned().collect()
        };
        for id in &keys {
            let _ = self.kill(id);
        }
        keys.len()
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
fn configure_process_tree(cmd: &mut tokio::process::Command) {
    use std::os::windows::process::CommandExt;
    const CREATE_NO_WINDOW: u32 = 0x0800_0000;
    cmd.as_std_mut().creation_flags(CREATE_NO_WINDOW);
}

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
        let mut command = tokio::process::Command::new("taskkill");
        configure_process_tree(&mut command);
        let _ = command
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

        // Windows PowerShell cold start can contend with the concurrently
        // running Rust workspace tests on hosted CI machines. Keep a hard
        // deadline, but allow a realistic startup margin without weakening the
        // eventual exit status or output assertions. Linux stays fast.
        let startup_budget = if cfg!(windows) { 60 } else { 20 };
        let s = tokio::time::timeout(Duration::from_secs(startup_budget), async {
            loop {
                let status = store.poll("job-1").expect("job exists");
                if !status.running {
                    break status;
                }
                tokio::time::sleep(Duration::from_millis(50)).await;
            }
        })
        .await
        .expect("detached job must finish within the platform-specific startup budget");
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

    fn placeholder_job(running: bool, finished_at: Option<Instant>) -> Job {
        Job {
            stdout: Arc::new(Mutex::new(CappedBuf::new(0))),
            stderr: Arc::new(Mutex::new(CappedBuf::new(0))),
            state: Arc::new(Mutex::new(JobState {
                running,
                exit_code: if running { None } else { Some(0) },
                finished_at,
            })),
            kill: Mutex::new(None),
        }
    }

    #[test]
    fn admission_rejects_duplicate_and_per_controller_overload() {
        let mut jobs = HashMap::new();
        for n in 0..MAX_JOBS_PER_PEER {
            jobs.insert(format!("peer-a:{n}"), placeholder_job(true, None));
        }
        assert_eq!(
            JobStore::admit(&mut jobs, "peer-a:new").unwrap_err().kind(),
            std::io::ErrorKind::WouldBlock
        );
        assert_eq!(
            JobStore::admit(&mut jobs, "peer-a:0").unwrap_err().kind(),
            std::io::ErrorKind::AlreadyExists
        );
        assert!(JobStore::admit(&mut jobs, "peer-b:new").is_ok());
        assert_eq!(jobs.len(), MAX_JOBS_PER_PEER);
    }

    #[test]
    fn admission_never_evicts_running_jobs_but_reaps_finished_when_full() {
        let mut jobs = HashMap::new();
        for n in 0..MAX_JOBS {
            jobs.insert(format!("peer-{n}:job"), placeholder_job(true, None));
        }
        assert_eq!(
            JobStore::admit(&mut jobs, "new-peer:new").unwrap_err().kind(),
            std::io::ErrorKind::WouldBlock
        );
        assert_eq!(jobs.len(), MAX_JOBS);
        let completed = "peer-0:job";
        let state = &jobs.get(completed).unwrap().state;
        let mut state = state.lock().unwrap();
        state.running = false;
        state.finished_at = Some(Instant::now());
        drop(state);
        assert!(JobStore::admit(&mut jobs, "new-peer:new").is_ok());
        assert!(!jobs.contains_key(completed));
        assert_eq!(jobs.len(), MAX_JOBS - 1);
    }

    #[tokio::test(flavor = "multi_thread", worker_threads = 2)]
    async fn simultaneous_starts_cannot_overwrite_same_job_id() {
        let store = Arc::new(JobStore::new());
        let mut attempts = Vec::new();
        for _ in 0..12 {
            let store = store.clone();
            attempts.push(tokio::task::spawn_blocking(move || {
                #[cfg(windows)]
                let command = "Write-Output admitted";
                #[cfg(not(windows))]
                let command = "echo admitted";
                store.start("peer:one-id", command, small_cap())
            }));
        }
        let mut accepted = 0;
        for result in attempts {
            match result.await.unwrap() {
                Ok(()) => accepted += 1,
                Err(err) => assert_eq!(err.kind(), std::io::ErrorKind::AlreadyExists),
            }
        }
        assert_eq!(accepted, 1);
        assert!(store.poll("peer:one-id").is_some());
        store.kill("peer:one-id");
    }
}

