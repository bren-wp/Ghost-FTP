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

/// Build the `run …` argument line shared by every backend.
fn run_args(extra_args: &[String]) -> String {
    let mut parts = vec!["run".to_string()];
    parts.extend(extra_args.iter().cloned());
    parts.join(" ")
}

#[cfg(target_os = "linux")]
mod linux {
    use super::*;
    use std::process::Command;

    pub fn install(extra: &[String]) -> Result<()> {
        let exe = exe_path()?;
        let is_root = unsafe { libc_geteuid() } == 0;
        let unit = unit_file(&exe, extra);
        let (path, user_flag) = if is_root {
            (PathBuf::from("/etc/systemd/system/ghostftp-agentd.service"), false)
        } else {
            let dir = dirs::config_dir()
                .context("no user config dir")?
                .join("systemd/user");
            std::fs::create_dir_all(&dir).ok();
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

        println!("Installed {} systemd service.", if user_flag { "user" } else { "system" });
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

    pub fn uninstall() -> Result<()> {
        let is_root = unsafe { libc_geteuid() } == 0;
        let (path, user_flag) = if is_root {
            (PathBuf::from("/etc/systemd/system/ghostftp-agentd.service"), false)
        } else {
            (
                dirs::config_dir()
                    .context("no user config dir")?
                    .join("systemd/user/ghostftp-agentd.service"),
                true,
            )
        };
        let mut c = Command::new("systemctl");
        if user_flag {
            c.arg("--user");
        }
        let _ = c.args(["disable", "--now", "ghostftp-agentd.service"]).status();
        std::fs::remove_file(&path).ok();
        let mut c = Command::new("systemctl");
        if user_flag {
            c.arg("--user");
        }
        let _ = c.arg("daemon-reload").status();
        println!("Removed ghostftp-agentd systemd service.");
        Ok(())
    }

    fn unit_file(exe: &std::path::Path, extra: &[String]) -> String {
        format!(
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
            exe = exe.display(),
            args = run_args(extra),
        )
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

    pub fn install(extra: &[String]) -> Result<()> {
        let exe = exe_path()?;
        // A logon Scheduled Task: ghostftp-agentd is a console app, not an SCM
        // service, so `sc create` would be killed for not reporting to the SCM.
        let tr = format!("\"{}\" {}", exe.display(), run_args(extra));
        let status = Command::new("schtasks")
            .args([
                "/create", "/f", "/tn", TASK_NAME, "/sc", "onlogon", "/rl", "limited", "/tr", &tr,
            ])
            .status()
            .context("run schtasks (is it on PATH?)")?;
        if !status.success() {
            bail!("schtasks failed to create the task");
        }
        // Start it now so the user doesn't have to log out/in first.
        let _ = Command::new("schtasks").args(["/run", "/tn", TASK_NAME]).status();
        println!("Installed logon task '{TASK_NAME}'. It starts ghostftp-agentd at every sign-in.");
        println!("  remove: ghostftp-agentd uninstall");
        Ok(())
    }

    pub fn uninstall() -> Result<()> {
        let _ = Command::new("schtasks").args(["/end", "/tn", TASK_NAME]).status();
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

    #[test]
    fn run_args_joins_extra() {
        assert_eq!(run_args(&[]), "run");
        assert_eq!(
            run_args(&["--read-only".into(), "--port".into(), "9000".into()]),
            "run --read-only --port 9000"
        );
    }
}
