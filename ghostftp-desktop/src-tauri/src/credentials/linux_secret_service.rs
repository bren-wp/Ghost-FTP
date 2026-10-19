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
use std::process::{Command, Stdio};

const MAX_SECRET_BYTES: usize = 4 * 1024 * 1024;

fn command(operation: &str, service: &str, account: &str) -> Command {
    let mut command = Command::new("secret-tool");
    command
        .arg(operation)
        .arg("--") // Prevent untrusted profile ids from becoming CLI options.
        .args(["application", "ghostftp", "service", service, "account", account])
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    command
}

pub fn lookup(service: &str, account: &str) -> Result<Option<String>> {
    let output = command("lookup", service, account)
        .output()
        .map_err(|_| anyhow!("Linux Secret Service client not installed or unavailable"))?;
    if output.status.success() {
        if output.stdout.len() > MAX_SECRET_BYTES {
            return Err(anyhow!("Linux Secret Service credential exceeds maximum length"));
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
    Err(anyhow!("Linux Secret Service lookup failed or vault is locked"))
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
        .args(["application", "ghostftp", "service", service, "account", account])
        .stdin(Stdio::piped())
        .stdout(Stdio::null())
        .stderr(Stdio::null());
    let mut child = command
        .spawn()
        .map_err(|_| anyhow!("Linux Secret Service client is unavailable; install libsecret-tools"))?;
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
    if !child
        .wait()
        .map_err(|_| anyhow!("Linux Secret Service write interrupted"))?
        .success()
    {
        return Err(anyhow!("Linux Secret Service refused to save the credential"));
    }
    match lookup(service, account)? {
        Some(saved) if saved == secret => Ok(()),
        _ => Err(anyhow!("Linux Secret Service write could not be verified")),
    }
}

pub fn remove(service: &str, account: &str) -> Result<()> {
    let output = command("clear", service, account)
        .output()
        .map_err(|_| anyhow!("Linux Secret Service client unavailable during deletion"))?;
    if !output.status.success() {
        return Err(anyhow!("Linux Secret Service refused credential deletion"));
    }
    if lookup(service, account)?.is_some() {
        return Err(anyhow!("Linux Secret Service credential still exists after deletion"));
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
    }
}
