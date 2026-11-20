use std::process::Command;

/// Prevent console-subsystem helpers from briefly creating a visible terminal
/// window when Ghost FTP launches them from the native GUI.
#[cfg(windows)]
pub(crate) fn hide_console(command: &mut Command) {
    use std::os::windows::process::CommandExt;
    const CREATE_NO_WINDOW: u32 = 0x0800_0000;
    command.creation_flags(CREATE_NO_WINDOW);
}

#[cfg(not(windows))]
pub(crate) fn hide_console(_command: &mut Command) {}

/// Ask Windows to open a local document or an already-approved HTTPS URL with
/// its registered association. Do not search PATH or the current directory for
/// rundll32.exe: both may be attacker-controlled on a portable installation.
#[cfg(windows)]
pub(crate) fn open_with_shell(target: &std::ffi::OsStr) -> std::io::Result<()> {
    use std::io::{Error, ErrorKind};
    use std::os::windows::ffi::OsStrExt;
    use windows_sys::Win32::System::Com::{
        CoInitializeEx, CoUninitialize, COINIT_APARTMENTTHREADED, COINIT_DISABLE_OLE1DDE,
    };
    use windows_sys::Win32::UI::Shell::ShellExecuteW;
    use windows_sys::Win32::UI::WindowsAndMessaging::SW_SHOWNORMAL;

    let mut wide: Vec<u16> = target.encode_wide().collect();
    if wide.is_empty() || wide.contains(&0) {
        return Err(Error::new(ErrorKind::InvalidInput, "Invalid shell target"));
    }
    wide.push(0);

    // Shell extensions may require a COM single-threaded apartment. Caller
    // runs this function on a blocking worker, never on Tauri's GUI thread.
    let hr = unsafe {
        CoInitializeEx(
            std::ptr::null(),
            COINIT_APARTMENTTHREADED | COINIT_DISABLE_OLE1DDE,
        )
    };
    if hr < 0 {
        return Err(Error::other(
            "Could not initialize the Windows shell COM apartment",
        ));
    }
    struct ComGuard;
    impl Drop for ComGuard {
        fn drop(&mut self) {
            unsafe { CoUninitialize() };
        }
    }
    let _com = ComGuard;
    let verb: Vec<u16> = "open".encode_utf16().chain(std::iter::once(0)).collect();

    let result = unsafe {
        ShellExecuteW(
            std::ptr::null_mut(),
            verb.as_ptr(),
            wide.as_ptr(),
            std::ptr::null(),
            std::ptr::null(),
            SW_SHOWNORMAL,
        )
    };
    if (result as isize) <= 32 {
        // Numeric status only; do not log user filenames or URLs with tokens.
        return Err(Error::other(format!(
            "Windows shell association could not be opened (code {})",
            result as isize
        )));
    }
    Ok(())
}

#[cfg(all(windows, test))]
mod tests {
    use super::*;
    use std::ffi::OsStr;

    #[test]
    fn associated_shell_target_rejects_nul_and_empty_input() {
        assert!(open_with_shell(OsStr::new("")).is_err());
        assert!(open_with_shell(OsStr::new("C:\\safe\\bad\0name.txt")).is_err());
    }
}
