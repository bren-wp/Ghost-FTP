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
use std::io::Write;
use std::process::{Child, Command, Output, Stdio};
use std::time::{Duration, Instant};

// GNOME libsecret's secret-tool currently reads at most 8192 bytes from a
// piped password; reject larger values instead of truncating private material.
const MAX_SECRET_BYTES: usize = 8191;
// A locked/unresponsive Secret Service must not indefinitely park a desktop
// command or leave a dangling credential helper on Windows/Linux clients.
const SECRET_TOOL_TIMEOUT: Duration = Duration::from_secs(30);
const CHILD_POLL_INTERVAL: Duration = Duration::from_millis(20);

fn wait_bounded(mut child: Child, timeout: Duration) -> Result<Output> {
    let started = Instant::now();
    loop {
        match child.try_wait() {
            Ok(Some(_)) => {
                return child
                    .wait_with_output()
                    .map_err(|_| anyhow!("Linux Secret Service response could not be read"));
            }
            Ok(None) if started.elapsed() >= timeout => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(anyhow!("Linux Secret Service timed out; unlock the keyring"));
            }
            Ok(None) => std::thread::sleep(CHILD_POLL_INTERVAL),
            Err(_) => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(anyhow!("Linux Secret Service process failed"));
            }
        }
    }
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
    let mut child = command.spawn().map_err(|_| {
        anyhow!("Linux Secret Service client is unavailable; install libsecret-tools")
    })?;
    let written = child
        .stdin
        .take()
        .ok_or_else(|| anyhow!("Linux Secret Service pipe unavailable"))?
        .write_all(secret.as_bytes());
    if written.is_err() {
        let _ = child.kill();
        let _ = child.wait();
        return Err(anyhow!("Linux Secret Service refused the credential input"));
    }
    if !wait_bounded(child, SECRET_TOOL_TIMEOUT)?.status.success() {
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
    fn completed_credential_helper_retains_exact_stdout_bytes() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "printf \'secret-with-newline\\n\'"])
            .stdout(Stdio::piped())
            .stderr(Stdio::null());
        let child = helper.spawn().expect("launch local output fixture");
        let output = wait_bounded(child, Duration::from_secs(2)).unwrap();
        assert!(output.status.success());
        assert_eq!(output.stdout, b"secret-with-newline\\n");
    }
}
