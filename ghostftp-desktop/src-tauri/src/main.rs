#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

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
        " Remove-Item -LiteralPath '{}' -Force -ErrorAction SilentlyContinue;",
        quote(&install_dir)
    ));

    let _ = std::process::Command::new("powershell.exe")
        .args([
            "-NoProfile",
            "-NonInteractive",
            "-WindowStyle",
            "Hidden",
            "-Command",
        ])
        .arg(script)
        .spawn();
    true
}

fn main() {
    #[cfg(windows)]
    if run_uninstaller_if_requested() {
        return;
    }
    ghostftp_lib::run();
}
