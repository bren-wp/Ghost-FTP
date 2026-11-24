//! Linux Secret Service over the official libsecret secret-tool client.
//!
//! No new Cargo lock dependencies; requires libsecret-tools plus a user-session
//! implementation of org.freedesktop.secrets (GNOME Keyring, KWallet).
//! Credentials are written via a private stdin pipe, NEVER via argv or env.
//! stdout contains a secret only during lookup, kept within Rust; stderr is
//! discarded to avoid echoing provider data into logs.
//! Legacy kernel keyutils entries are still resolved by the credentials layer.
//! This module must never erase the old backend during migration.

use anyhow::{anyhow, Result};
use std::io::{Read, Write};
use std::process::{Child, Command, Output, Stdio};
use std::time::{Duration, Instant};

// GNOME libsecret's secret-tool currently reads at most 8192 bytes from a
// piped password; reject larger values instead of truncating private material.
const MAX_SECRET_BYTES: usize = 8191;
// A locked/unresponsive Secret Service must not indefinitely park a desktop
// command or leave a dangling credential helper on Windows/Linux clients.
const SECRET_TOOL_TIMEOUT: Duration = Duration::from_secs(30);
const CHILD_POLL_MIN_INTERVAL: Duration = Duration::from_millis(1);
const CHILD_POLL_MAX_INTERVAL: Duration = Duration::from_millis(20);

fn poll_delay(interval: Duration, remaining: Duration) -> Duration {
    interval.min(remaining)
}

fn next_poll_interval(interval: Duration) -> Duration {
    (interval * 2).min(CHILD_POLL_MAX_INTERVAL)
}

// Opt-in, test-only aggregate process timing. No credential bytes, identifiers
// or command arguments are recorded. Normal production builds retain no samples.
#[cfg(test)]
std::thread_local! {
    static HELPER_PROFILE: std::cell::RefCell<Option<[Vec<u128>; 3]>> =
        const { std::cell::RefCell::new(None) };
}

#[cfg(test)]
pub(crate) fn start_linux_helper_profile() {
    HELPER_PROFILE.with(|profile| {
        *profile.borrow_mut() = Some(std::array::from_fn(|_| Vec::new()));
    });
}

#[cfg(test)]
pub(crate) fn finish_linux_helper_profile() {
    HELPER_PROFILE.with(|profile| {
        if let Some(mut groups) = profile.borrow_mut().take() {
            for (operation, samples) in ["lookup", "store", "clear"].into_iter().zip(&mut groups) {
                if samples.is_empty() {
                    continue;
                }
                samples.sort_unstable();
                let count = samples.len();
                let total: u128 = samples.iter().sum();
                // Nearest rank, using the actual count and no interpolation.
                let p95 = samples[(count * 95).div_ceil(100) - 1];
                let p99 = samples[(count * 99).div_ceil(100) - 1];
                eprintln!(
                    "recovery_linux_helper operation={operation} calls={count} total_us={total} median_us={} p95_us={p95} p99_us={p99} max_us={}",
                    samples[count / 2],
                    samples[count - 1]
                );
            }
        }
    });
}

fn profile_helper(_operation: usize, run: impl FnOnce() -> Result<Output>) -> Result<Output> {
    #[cfg(test)]
    let started = Instant::now();
    let result = run();
    #[cfg(test)]
    HELPER_PROFILE.with(|profile| {
        if let Some(groups) = profile.borrow_mut().as_mut() {
            groups[_operation].push(started.elapsed().as_micros());
        }
    });
    result
}

// Drain both helper pipes before exit, or the OS pipe buffer can stall the child.
// Refuse an unbounded amount of subprocess output.
const MAX_HELPER_STDERR_BYTES: usize = 4096;

fn read_bounded(mut pipe: impl Read, max_bytes: usize) -> Result<Vec<u8>> {
    // Cap retained bytes, NOT bytes consumed from the pipe. Stopping reads at
    // max_bytes + 1 can close stdout while the helper is still writing, causing
    // an otherwise successful helper to exit with SIGPIPE (intermittent CI
    // failures). Continue draining into a fixed buffer until EOF, while saving
    // only the first max_bytes + 1 bytes for the caller's overflow check.
    let limit = max_bytes.saturating_add(1);
    let mut output = Vec::with_capacity(limit.min(8192));
    let mut buffer = [0u8; 8192];
    loop {
        match pipe.read(&mut buffer) {
            Ok(0) => break,
            Ok(count) => {
                let retain = (limit - output.len()).min(count);
                output.extend_from_slice(&buffer[..retain]);
            }
            Err(error) if error.kind() == std::io::ErrorKind::Interrupted => continue,
            Err(_) => return Err(anyhow!("Linux Secret Service output could not be read")),
        }
    }
    Ok(output)
}

fn wait_bounded(mut child: Child, timeout: Duration) -> Result<Output> {
    let started = Instant::now();
    std::thread::scope(|scope| {
        let stdout_reader = child
            .stdout
            .take()
            .map(|pipe| scope.spawn(move || read_bounded(pipe, MAX_SECRET_BYTES)));
        let stderr_reader = child
            .stderr
            .take()
            .map(|pipe| scope.spawn(move || read_bounded(pipe, MAX_HELPER_STDERR_BYTES)));

        let mut interval = CHILD_POLL_MIN_INTERVAL;
        let status: Result<std::process::ExitStatus> = loop {
            match child.try_wait() {
                Ok(Some(status)) => break Ok(status),
                Ok(None) if started.elapsed() >= timeout => {
                    let _ = child.kill();
                    let _ = child.wait();
                    break Err(anyhow!(
                        "Linux Secret Service timed out; unlock the keyring"
                    ));
                }
                Ok(None) => {
                    std::thread::sleep(poll_delay(
                        interval,
                        timeout.saturating_sub(started.elapsed()),
                    ));
                    interval = next_poll_interval(interval);
                }
                Err(_) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    break Err(anyhow!("Linux Secret Service process failed"));
                }
            }
        };

        let stdout = match stdout_reader {
            Some(reader) => reader
                .join()
                .map_err(|_| anyhow!("Linux Secret Service output reader failed"))??,
            None => Vec::new(),
        };
        let stderr = match stderr_reader {
            Some(reader) => reader
                .join()
                .map_err(|_| anyhow!("Linux Secret Service error reader failed"))??,
            None => Vec::new(),
        };
        Ok(Output {
            status: status?,
            stdout,
            stderr,
        })
    })
}

// A write to child stdin can block indefinitely before calling wait_bounded.
// Keep the deadline supervisor active while a worker handles the input pipe.
fn write_then_wait_bounded(mut child: Child, secret: &[u8], timeout: Duration) -> Result<Output> {
    let mut stdin = match child.stdin.take() {
        Some(stdin) => stdin,
        None => {
            let _ = child.kill();
            let _ = child.wait();
            return Err(anyhow!("Linux Secret Service pipe unavailable"));
        }
    };
    std::thread::scope(|scope| {
        let writer = scope.spawn(move || stdin.write_all(secret));
        let output = wait_bounded(child, timeout);
        let written = writer
            .join()
            .map_err(|_| anyhow!("Linux Secret Service input writer failed"))?;
        let output = output?;
        written.map_err(|_| anyhow!("Linux Secret Service refused credential input"))?;
        Ok(output)
    })
}

fn command(operation: &str, service: &str, account: &str) -> Command {
    let mut command = Command::new("secret-tool");
    command
        .arg(operation)
        .arg("--") // Prevent untrusted profile ids from becoming CLI options.
        .args([
            "application",
            "ghostftp",
            "service",
            service,
            "account",
            account,
        ])
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    command
}

pub fn lookup(service: &str, account: &str) -> Result<Option<String>> {
    let output = profile_helper(0, || {
        let child = command("lookup", service, account)
            .spawn()
            .map_err(|_| anyhow!("Linux Secret Service client not installed or unavailable"))?;
        wait_bounded(child, SECRET_TOOL_TIMEOUT)
    })?;
    if output.status.success() {
        if output.stdout.len() > MAX_SECRET_BYTES {
            return Err(anyhow!(
                "Linux Secret Service credential exceeds maximum length"
            ));
        }
        return String::from_utf8(output.stdout)
            .map(Some)
            .map_err(|_| anyhow!("Linux Secret Service returned a non-textual credential"));
    }
    // secret-tool exits 1 with empty stderr when no entry exists. All other
    // failure causes (locked vault, D-Bus errors) fail closed, not as NoEntry.
    if output.status.code() == Some(1) && output.stderr.is_empty() {
        return Ok(None);
    }
    Err(anyhow!(
        "Linux Secret Service lookup failed or vault is locked"
    ))
}

pub fn store(service: &str, account: &str, secret: &str) -> Result<()> {
    if secret.is_empty() || secret.len() > MAX_SECRET_BYTES {
        return Err(anyhow!("Invalid Linux Secret Service credential length"));
    }
    let mut command = Command::new("secret-tool");
    command
        .arg("store")
        .arg("--label=Ghost FTP")
        .arg("--")
        .args([
            "application",
            "ghostftp",
            "service",
            service,
            "account",
            account,
        ])
        .stdin(Stdio::piped())
        .stdout(Stdio::null())
        .stderr(Stdio::null());
    let output = profile_helper(1, || {
        let child = command.spawn().map_err(|_| {
            anyhow!("Linux Secret Service client is unavailable; install libsecret-tools")
        })?;
        write_then_wait_bounded(child, secret.as_bytes(), SECRET_TOOL_TIMEOUT)
    })?;
    if !output.status.success() {
        return Err(anyhow!(
            "Linux Secret Service refused to save the credential"
        ));
    }
    match lookup(service, account)? {
        Some(saved) if saved == secret => Ok(()),
        _ => Err(anyhow!("Linux Secret Service write could not be verified")),
    }
}

pub fn remove(service: &str, account: &str) -> Result<()> {
    let output = profile_helper(2, || {
        let child = command("clear", service, account)
            .spawn()
            .map_err(|_| anyhow!("Linux Secret Service client unavailable during deletion"))?;
        wait_bounded(child, SECRET_TOOL_TIMEOUT)
    })?;
    if !output.status.success() {
        return Err(anyhow!("Linux Secret Service refused credential deletion"));
    }
    if lookup(service, account)?.is_some() {
        return Err(anyhow!(
            "Linux Secret Service credential still exists after deletion"
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn adaptive_polling_is_capped_and_respects_remaining_deadline() {
        let mut interval = CHILD_POLL_MIN_INTERVAL;
        for expected in [1, 2, 4, 8, 16, 20, 20] {
            assert_eq!(interval, Duration::from_millis(expected));
            interval = next_poll_interval(interval);
        }
        assert_eq!(
            poll_delay(interval, Duration::from_millis(3)),
            Duration::from_millis(3)
        );
        assert_eq!(poll_delay(interval, Duration::ZERO), Duration::ZERO);
    }

    #[test]
    fn invalid_secret_inputs_are_rejected_before_launching_any_process() {
        assert!(store("service", "account", "").is_err());
        assert!(store("service", "account", &"x".repeat(MAX_SECRET_BYTES + 1)).is_err());
        assert!(store("service", "account", &"x".repeat(MAX_SECRET_BYTES + 200)).is_err());
    }

    #[test]
    fn blocked_credential_helper_times_out_without_hanging() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "exec sleep 2"])
            .stdout(Stdio::null())
            .stderr(Stdio::null());
        let started = Instant::now();
        let child = helper.spawn().expect("launch temporary blocked command");
        let err = wait_bounded(child, Duration::from_millis(80)).unwrap_err();
        assert!(err.to_string().contains("timed out"));
        assert!(started.elapsed() < Duration::from_secs(2));
    }

    #[test]
    fn blocked_stdin_write_is_interrupted_by_the_process_deadline() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "exec sleep 2"])
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(Stdio::null());
        let started = Instant::now();
        let child = helper.spawn().expect("launch unresponsive helper");
        // Larger than a typical Linux pipe buffer: the timeout must still fire.
        let input = vec![b'x'; 512 * 1024];
        let err = write_then_wait_bounded(child, &input, Duration::from_millis(80))
            .expect_err("hung helper must time out");
        assert!(err.to_string().contains("timed out"));
        assert!(started.elapsed() < Duration::from_secs(2));
    }

    #[test]
    fn credential_input_writer_closes_stdin_after_success() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "cat >/dev/null"])
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(Stdio::null());
        let child = helper.spawn().expect("launch input consumer");
        let output = write_then_wait_bounded(child, b"test-secret", Duration::from_secs(2))
            .expect("finish writing helper input");
        assert!(output.status.success());
    }

    #[test]
    fn oversized_helper_output_is_drained_without_unbounded_allocation() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "head -c 8193 /dev/zero"])
            .stdout(Stdio::piped())
            .stderr(Stdio::null());
        let child = helper.spawn().expect("launch output fixture");
        let output =
            wait_bounded(child, Duration::from_secs(2)).expect("drain a large credential response");
        assert!(output.status.success());
        assert_eq!(output.stdout.len(), MAX_SECRET_BYTES + 1);
    }

    #[test]
    fn large_helper_output_is_fully_drained_while_memory_stays_bounded() {
        // A producer must not see a prematurely closed stdout/stderr pipe,
        // even after both credential-sized capture limits have been reached.
        // This also confirms the discarded data never accumulates in memory.
        let mut helper = Command::new("sh");
        helper
            .args([
                "-c",
                "head -c 1048576 /dev/zero; head -c 1048576 /dev/zero >&2",
            ])
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        let child = helper.spawn().expect("launch large output fixture");
        let output = wait_bounded(child, Duration::from_secs(10)).expect("drain both output pipes");
        assert!(output.status.success());
        assert_eq!(output.stdout.len(), MAX_SECRET_BYTES + 1);
        assert_eq!(output.stderr.len(), MAX_HELPER_STDERR_BYTES + 1);
    }

    #[test]
    fn completed_credential_helper_retains_exact_stdout_bytes() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "printf 'secret-with-newline\\n'"])
            .stdout(Stdio::piped())
            .stderr(Stdio::null());
        let child = helper.spawn().expect("launch local output fixture");
        let output = wait_bounded(child, Duration::from_secs(2)).unwrap();
        assert!(output.status.success());
        assert_eq!(output.stdout, b"secret-with-newline\n");
    }
}
