#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[cfg(windows)]
fn show_uninstall_error(error: &std::io::Error) {
    use std::ffi::OsStr;
    use std::os::windows::ffi::OsStrExt;
    use windows_sys::Win32::UI::WindowsAndMessaging::{MessageBoxW, MB_ICONERROR, MB_OK};

    let title: Vec<u16> = OsStr::new("Ghost FTP Uninstall")
        .encode_wide()
        .chain(std::iter::once(0))
        .collect();
    let message = format!(
        "Ghost FTP could not start the uninstall helper. No application files were removed.\n\n{error}"
    );
    let message: Vec<u16> = OsStr::new(&message)
        .encode_wide()
        .chain(std::iter::once(0))
        .collect();

    unsafe {
        MessageBoxW(
            std::ptr::null_mut(),
            message.as_ptr(),
            title.as_ptr(),
            MB_OK | MB_ICONERROR,
        );
    }
}

#[cfg(windows)]
fn spawn_uninstall_helper(script: String) -> std::io::Result<()> {
    let windows_dir = std::env::var_os("SystemRoot")
        .or_else(|| std::env::var_os("WINDIR"))
        .ok_or_else(|| {
            std::io::Error::new(
                std::io::ErrorKind::NotFound,
                "Windows system directory is unavailable",
            )
        })?;
    let powershell = std::path::PathBuf::from(windows_dir)
        .join("System32")
        .join("WindowsPowerShell")
        .join("v1.0")
        .join("powershell.exe");

    if !powershell.is_file() {
        return Err(std::io::Error::new(
            std::io::ErrorKind::NotFound,
            format!("system PowerShell was not found at {}", powershell.display()),
        ));
    }

    std::process::Command::new(powershell)
        .args([
            "-NoProfile",
            "-NonInteractive",
            "-WindowStyle",
            "Hidden",
            "-Command",
        ])
        .arg(script)
        .spawn()
        .map(|_| ())
}

#[cfg(windows)]
fn run_uninstaller_if_requested() -> bool {
    if !std::env::args_os().skip(1).any(|arg| arg == "--uninstall") {
        return false;
    }

    let Ok(exe) = std::env::current_exe() else {
        return true;
    };
    let Some(install_dir) = exe.parent().map(std::path::Path::to_path_buf) else {
        return true;
    };
    // Never recursively delete from the running process. A detached PowerShell
    // helper waits for this PID, removes only the known per-user product files,
    // then removes the installation directory if it is empty.
    let pid = std::process::id();
    let desktop = std::env::var_os("USERPROFILE")
        .map(std::path::PathBuf::from)
        .map(|p| p.join("Desktop").join("Ghost FTP.lnk"));
    let start_menu = std::env::var_os("APPDATA")
        .map(std::path::PathBuf::from)
        .map(|p| {
            p.join("Microsoft")
                .join("Windows")
                .join("Start Menu")
                .join("Programs")
        });
    let quote = |s: &std::path::Path| s.to_string_lossy().replace('\'', "''");
    let mut script = format!(
        "$ErrorActionPreference='SilentlyContinue'; Wait-Process -Id {pid}; Remove-Item -LiteralPath '{}' -Force;",
        quote(&exe)
    );
    if let Some(path) = desktop.as_deref() {
        script.push_str(&format!(
            " Remove-Item -LiteralPath '{}' -Force;",
            quote(path)
        ));
    }
    if let Some(dir) = start_menu.as_deref() {
        script.push_str(&format!(
            " Remove-Item -LiteralPath '{}' -Force; Remove-Item -LiteralPath '{}' -Force;",
            quote(&dir.join("Ghost FTP.lnk")),
            quote(&dir.join("Uninstall Ghost FTP.lnk"))
        ));
    }
    script.push_str(
        r" Remove-Item -LiteralPath 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP' -Recurse -Force;",
    );
    script.push_str(&format!(
        " if ((Test-Path -LiteralPath '{}') -and -not (Get-ChildItem -LiteralPath '{}' -Force | Select-Object -First 1)) {{ Remove-Item -LiteralPath '{}' -Force -Confirm:$false -ErrorAction SilentlyContinue; }}",
        quote(&install_dir),
        quote(&install_dir),
        quote(&install_dir)
    ));

    if let Err(error) = spawn_uninstall_helper(script) {
        show_uninstall_error(&error);
    }
    true
}

fn main() {
    #[cfg(windows)]
    if run_uninstaller_if_requested() {
        return;
    }
    ghostftp_lib::run();
}
