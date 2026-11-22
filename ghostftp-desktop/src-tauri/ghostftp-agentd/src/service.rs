//! Install `ghostftp-agentd` as a background service so a controlled machine keeps
//! serving across reboots without anyone leaving a terminal open.
//!
//! Per platform, using only what ships with the OS (no extra tooling):
//!   * **Linux** — a systemd unit. A system unit when run as root, otherwise a
//!     per-user unit (`systemctl --user`), which needs no sudo. This is the
//!     primary target: the ServerKit/hosting case.
//!   * **Windows** — a per-user logon Scheduled Task (`schtasks`). `ghostftp-agentd`
//!     is a plain console app, not an SCM service, so a Task is the honest fit.
//!
//! The service/task just runs `ghostftp-agentd run`, so it obeys the same
//! identity, pins, and policy as an interactive run — installing changes *when*
//! it runs, not *what* it can do.

use anyhow::{bail, Context, Result};
use std::path::PathBuf;

#[cfg(target_os = "windows")]
const TASK_NAME: &str = "Ghost FTPAgent";

/// Absolute path to the running executable, for the service definition.
fn exe_path() -> Result<PathBuf> {
    std::env::current_exe().context("resolve current executable path")
}

/// Extra args to append to `run` in the service definition (e.g. --read-only,
/// --port). `--config-dir` is passed through so a service uses the same config
/// the admin set up interactively.
pub fn install(extra_args: &[String]) -> Result<()> {
    #[cfg(target_os = "linux")]
    return linux::install(extra_args);
    #[cfg(target_os = "windows")]
    return windows::install(extra_args);
    #[cfg(not(any(target_os = "linux", target_os = "windows")))]
    bail!("service install isn't supported on this platform — run `ghostftp-agentd run` yourself");
}

pub fn uninstall() -> Result<()> {
    #[cfg(target_os = "linux")]
    return linux::uninstall();
    #[cfg(target_os = "windows")]
    return windows::uninstall();
    #[cfg(not(any(target_os = "linux", target_os = "windows")))]
    bail!("service uninstall isn't supported on this platform");
}

/// Inspect the installed service/task without starting it or mutating identity.
pub fn status() -> Result<()> {
    #[cfg(target_os = "linux")]
    return linux::status();
    #[cfg(target_os = "windows")]
    return windows::status();
    #[cfg(not(any(target_os = "linux", target_os = "windows")))]
    bail!("service status isn't supported on this platform");
}

/// Build the `run …` argument line shared by every backend.
fn validate_service_argument(value: &str) -> Result<()> {
    // Unit files are line-based and Windows tasks are command-line based.
    // Never let an argument inject a new service directive or control byte.
    if value.chars().any(char::is_control) {
        bail!("service arguments must not contain control characters");
    }
    Ok(())
}

fn run_args(extra_args: &[String], quote: fn(&str) -> Result<String>) -> Result<String> {
    std::iter::once("run")
        .chain(extra_args.iter().map(String::as_str))
        .map(quote)
        .collect::<Result<Vec<_>>>()
        .map(|parts| parts.join(" "))
}

#[cfg(target_os = "linux")]
mod linux {
    use super::*;
    use std::process::Command;

    pub fn install(extra: &[String]) -> Result<()> {
        let exe = exe_path()?;
        let is_root = unsafe { libc_geteuid() } == 0;
        let unit = unit_file(&exe, extra)?;
        let (path, user_flag) = if is_root {
            (
                PathBuf::from("/etc/systemd/system/ghostftp-agentd.service"),
                false,
            )
        } else {
            let dir = dirs::config_dir()
                .context("no user config dir")?
                .join("systemd/user");
            std::fs::create_dir_all(&dir)
                .with_context(|| format!("create {}", dir.display()))?;
            (dir.join("ghostftp-agentd.service"), true)
        };
        std::fs::write(&path, unit).with_context(|| format!("write {}", path.display()))?;

        let scope = |args: &[&str]| -> Result<()> {
            let mut c = Command::new("systemctl");
            if user_flag {
                c.arg("--user");
            }
            c.args(args);
            let status = c.status().context("run systemctl")?;
            if !status.success() {
                bail!("systemctl {:?} failed", args);
            }
            Ok(())
        };
        scope(&["daemon-reload"])?;
        scope(&["enable", "--now", "ghostftp-agentd.service"])?;

        println!(
            "Installed {} systemd service.",
            if user_flag { "user" } else { "system" }
        );
        println!("  unit  : {}", path.display());
        println!(
            "  status: systemctl {}status ghostftp-agentd",
            if user_flag { "--user " } else { "" }
        );
        if user_flag {
            println!(
                "  note  : a user service stops at logout unless you enable lingering:\n\
                 \x20         sudo loginctl enable-linger $USER"
            );
        }
        Ok(())
    }

    pub fn status() -> Result<()> {
        let user_flag = unsafe { libc_geteuid() } != 0;
        let mut command = Command::new("systemctl");
        if user_flag {
            command.arg("--user");
        }
        let output = command
            .args([
                "show",
                "ghostftp-agentd.service",
                "--no-pager",
                "--property=LoadState,ActiveState,SubState,UnitFileState",
            ])
            .output()
            .context("query ghostftp-agentd systemd service")?;
        if !output.status.success() {
            bail!(
                "systemctl show failed: {}",
                String::from_utf8_lossy(&output.stderr).trim()
            );
        }
        print!("{}", String::from_utf8_lossy(&output.stdout));
        Ok(())
    }

    pub fn uninstall() -> Result<()> {
        let is_root = unsafe { libc_geteuid() } == 0;
        let (path, user_flag) = if is_root {
            (
                PathBuf::from("/etc/systemd/system/ghostftp-agentd.service"),
                false,
            )
        } else {
            (
                dirs::config_dir()
                    .context("no user config dir")?
                    .join("systemd/user/ghostftp-agentd.service"),
                true,
            )
        };
        if !path.is_file() {
            bail!("Ghost FTP Agent systemd service is not installed at {}", path.display());
        }
        let mut c = Command::new("systemctl");
        if user_flag {
            c.arg("--user");
        }
        let stopped = c
            .args(["disable", "--now", "ghostftp-agentd.service"])
            .status()
            .context("disable and stop Ghost FTP Agent systemd service")?;
        if !stopped.success() {
            bail!("systemctl failed to disable and stop Ghost FTP Agent; service file left intact");
        }
        std::fs::remove_file(&path).with_context(|| format!("remove {}", path.display()))?;
        let mut c = Command::new("systemctl");
        if user_flag {
            c.arg("--user");
        }
        let reload = c
            .arg("daemon-reload")
            .status()
            .context("reload systemd after service removal")?;
        if !reload.success() {
            bail!("Ghost FTP Agent service removed but systemd daemon-reload failed");
        }
        println!("Removed and stopped ghostftp-agentd systemd service.");
        Ok(())
    }

    pub(super) fn quote_systemd_argument(value: &str) -> Result<String> {
        validate_service_argument(value)?;
        let mut escaped = String::with_capacity(value.len() + 2);
        escaped.push('"');
        for c in value.chars() {
            match c {
                '\\' => escaped.push_str("\\\\"),
                '"' => escaped.push_str("\\\""),
                '%' => escaped.push_str("%%"),
                '$' => escaped.push_str("$$"),
                _ => escaped.push(c),
            }
        }
        escaped.push('"');
        Ok(escaped)
    }

    pub(super) fn unit_file(exe: &std::path::Path, extra: &[String]) -> Result<String> {
        let exe = exe
            .to_str()
            .context("service executable path must be valid UTF-8")?;
        Ok(format!(
            "[Unit]\n\
             Description=Ghost FTP Agent daemon (control this machine from Ghost FTP)\n\
             After=network-online.target\n\
             Wants=network-online.target\n\n\
             [Service]\n\
             Type=simple\n\
             ExecStart={exe} {args}\n\
             Restart=on-failure\n\
             RestartSec=5\n\n\
             [Install]\n\
             WantedBy=default.target\n",
            exe = quote_systemd_argument(exe)?,
            args = run_args(extra, quote_systemd_argument)?,
        ))
    }

    // Avoid a whole `libc` dependency for one call.
    extern "C" {
        fn geteuid() -> u32;
    }
    unsafe fn libc_geteuid() -> u32 {
        geteuid()
    }
}

#[cfg(target_os = "windows")]
mod windows {
    use super::*;
    use std::process::Command;

    pub(super) fn quote_windows_argument(value: &str) -> Result<String> {
        validate_service_argument(value)?;
        // Match the Microsoft CRT / CommandLineToArgvW backslash-quote rules.
        let mut quoted = String::from("\"");
        let mut pending_backslashes = 0;
        for c in value.chars() {
            match c {
                '\\' => pending_backslashes += 1,
                '"' => {
                    quoted.push_str(&"\\".repeat(2 * pending_backslashes + 1));
                    quoted.push('"');
                    pending_backslashes = 0;
                }
                _ => {
                    quoted.push_str(&"\\".repeat(pending_backslashes));
                    quoted.push(c);
                    pending_backslashes = 0;
                }
            }
        }
        quoted.push_str(&"\\".repeat(2 * pending_backslashes));
        quoted.push('"');
        Ok(quoted)
    }

    pub(super) fn task_run_command(exe: &std::path::Path, extra: &[String]) -> Result<String> {
        let exe = exe
            .to_str()
            .context("task executable path must be valid UTF-8")?;
        Ok(format!(
            "{} {}",
            quote_windows_argument(exe)?,
            run_args(extra, quote_windows_argument)?,
        ))
    }

    pub fn install(extra: &[String]) -> Result<()> {
        let exe = exe_path()?;
        // A logon Scheduled Task: ghostftp-agentd is a console app, not an SCM
        // service, so `sc create` would be killed for not reporting to the SCM.
        let tr = task_run_command(&exe, extra)?;
        let status = Command::new("schtasks")
            .args([
                "/create", "/f", "/tn", TASK_NAME, "/sc", "onlogon", "/rl", "limited", "/tr", &tr,
            ])
            .status()
            .context("run schtasks (is it on PATH?)")?;
        if !status.success() {
            bail!("schtasks failed to create the task");
        }
        // Report a failed initial launch; a scheduled but not running agent
        // must not be presented as a successfully started remote service.
        let started = Command::new("schtasks")
            .args(["/run", "/tn", TASK_NAME])
            .status()
            .context("start Ghost FTP Agent logon task")?;
        if !started.success() {
            bail!("Ghost FTP Agent task was created but could not start; inspect 'service-status'");
        }
        println!("Installed and started logon task '{TASK_NAME}'. It starts at every sign-in.");
        println!("  remove: ghostftp-agentd uninstall");
        Ok(())
    }

    pub fn status() -> Result<()> {
        let output = Command::new("schtasks")
            .args(["/query", "/tn", TASK_NAME, "/fo", "LIST"])
            .output()
            .context("query Ghost FTP Agent logon task")?;
        if !output.status.success() {
            bail!(
                "Ghost FTP Agent logon task not found or unavailable: {}",
                String::from_utf8_lossy(&output.stderr).trim()
            );
        }
        print!("{}", String::from_utf8_lossy(&output.stdout));
        Ok(())
    }

    pub fn uninstall() -> Result<()> {
        let _ = Command::new("schtasks")
            .args(["/end", "/tn", TASK_NAME])
            .status();
        let status = Command::new("schtasks")
            .args(["/delete", "/f", "/tn", TASK_NAME])
            .status()
            .context("run schtasks")?;
        if !status.success() {
            bail!("schtasks failed to delete the task (was it installed?)");
        }
        println!("Removed logon task '{TASK_NAME}'.");
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::Path;

    #[test]
    fn run_args_keeps_each_argument_separate() {
        let argv = vec!["--config-dir".to_string(), "dir with spaces".to_string()];
        let result = run_args(&argv, |value| {
            validate_service_argument(value)?;
            Ok(format!("<{value}>"))
        })
        .unwrap();
        assert_eq!(result, "<run> <--config-dir> <dir with spaces>");
        assert!(run_args(&["malicious\nExecStart=/bin/sh".into()], |value| {
            validate_service_argument(value)?;
            Ok(value.to_string())
        })
        .is_err());
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn systemd_unit_quotes_paths_and_rejects_directive_injection() {
        let args = vec![
            "--config-dir".to_string(),
            "/home/test user/%p/$VAULT".to_string(),
        ];
        let unit = linux::unit_file(Path::new("/opt/Ghost FTP/agentd"), &args).unwrap();
        assert!(unit.contains(
            "ExecStart=\"/opt/Ghost FTP/agentd\" \"run\" \"--config-dir\" \"/home/test user/%%p/$$VAULT\""
        ));
        let special = linux::quote_systemd_argument("quote\" and \\ backslash").unwrap();
        assert_eq!(special, "\"quote\\\" and \\\\ backslash\"");
        assert!(linux::unit_file(
            Path::new("/usr/bin/agentd"),
            &["evil\n[Service]\nExecStart=/bin/sh".into()]
        )
        .is_err());
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn systemd_analyze_accepts_generated_unit_when_installed() {
        if std::process::Command::new("systemd-analyze")
            .arg("--version")
            .output()
            .is_err()
        {
            return;
        }
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("ghostftp-agentd.service");
        let args = vec![
            "--config-dir".to_string(),
            "/tmp/Ghost FTP/%p/$HOME".to_string(),
        ];
        std::fs::write(
            &path,
            linux::unit_file(Path::new("/usr/bin/true"), &args).unwrap(),
        )
        .unwrap();
        let output = std::process::Command::new("systemd-analyze")
            .arg("verify")
            .arg(&path)
            .output()
            .unwrap();
        assert!(
            output.status.success(),
            "systemd unit rejected: {}",
            String::from_utf8_lossy(&output.stderr),
        );
    }

    #[cfg(target_os = "windows")]
    #[test]
    fn task_quotes_executable_and_arguments() {
        let args = vec![
            "--config-dir".to_string(),
            r"C:\Users\Test User\Ghost FTP".to_string(),
        ];
        let task =
            windows::task_run_command(Path::new(r"C:\Program Files\Ghost FTP\agentd.exe"), &args)
                .unwrap();
        assert_eq!(
            task,
            r#""C:\Program Files\Ghost FTP\agentd.exe" "run" "--config-dir" "C:\Users\Test User\Ghost FTP""#
        );
        assert_eq!(
            windows::quote_windows_argument("has \"quotes\"").unwrap(),
            r#""has \"quotes\"""#
        );
        assert_eq!(
            windows::quote_windows_argument("ends\\").unwrap(),
            r#""ends\\""#,
        );
        assert_eq!(
            windows::quote_windows_argument("München Ž").unwrap(),
            r#""München Ž""#,
        );
        assert!(windows::quote_windows_argument("bad\r\nline").is_err());
    }
}
