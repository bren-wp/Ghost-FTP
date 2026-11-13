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
const CHILD_POLL_INTERVAL: Duration = Duration::from_millis(20);

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
                Ok(None) => std::thread::sleep(CHILD_POLL_INTERVAL),
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
    let child = command("lookup", service, account)
        .spawn()
        .map_err(|_| anyhow!("Linux Secret Service client not installed or unavailable"))?;
    let output = wait_bounded(child, SECRET_TOOL_TIMEOUT)?;
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
    let child = command.spawn().map_err(|_| {
        anyhow!("Linux Secret Service client is unavailable; install libsecret-tools")
    })?;
    if !write_then_wait_bounded(child, secret.as_bytes(), SECRET_TOOL_TIMEOUT)?
        .status
        .success()
    {
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
    let child = command("clear", service, account)
        .spawn()
        .map_err(|_| anyhow!("Linux Secret Service client unavailable during deletion"))?;
    let output = wait_bounded(child, SECRET_TOOL_TIMEOUT)?;
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
