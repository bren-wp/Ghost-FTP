//! One-click "Add ghostftp-cli to PATH" (Plan 16 Phase 4) — per-user, no admin.
//!
//! Every serious dev tool (`code`, `gh`) offers a one-click "install shell
//! command"; Plan 10's `cli_updater::install_missing` downloads `ghostftp-cli` into
//! an app-owned `bin/` dir but leaves wiring PATH to the user. This closes that
//! gap at the **per-user** level, which needs no elevation:
//!
//! - **Windows:** append the app-owned `bin/` dir to the per-user `Path` under
//!   `HKCU\Environment` — no admin, no UAC. We read/modify/write the *exact*
//!   existing value, preserving its registry type (`REG_EXPAND_SZ` stays
//!   `REG_EXPAND_SZ`, `%VARS%` unexpanded), dedupe so re-adding is a no-op, and
//!   broadcast `WM_SETTINGCHANGE("Environment")` so new terminals pick it up.
//!   **`setx` is deliberately never used** — it truncates at 1024 chars and
//!   flattens the value type, silently eating other entries. The prior value is
//!   snapshotted into `ghostftp.db` before the first write so removal is a faithful
//!   restore.
//! - **Linux:** symlink `ghostftp-cli` into `~/.local/bin` (user-level, no
//!   sudo); if that dir isn't on PATH, append one marker-guarded `export PATH=…`
//!   line to the detected shell profile so removal is exact.
//!
//! Removal only ever touches Ghost FTP's own entry/symlink/marker — never anything
//! else in the user's environment.

use serde::Serialize;
use std::path::PathBuf;
use tauri::{AppHandle, Manager, State};

use crate::AppState;

/// The `ghostftp.db` env-backup key for the Windows per-user `Path`.
#[cfg(windows)]
const WIN_PATH_BACKUP_KEY: &str = "windows_user_path";

/// What the Settings → About PATH row renders. Reports both whether `ghostftp-cli`
/// resolves *anywhere* on PATH (a sidecar or user-installed copy counts) and
/// whether **Ghost FTP's own managed entry** is present, so the UI can show a plain
/// ✓ / "Add to PATH" / "Remove from PATH" / "Install first".
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PathStatus {
    /// The app-owned `bin/` dir Ghost FTP manages (where `install_missing` drops the CLI).
    pub bin_dir: String,
    /// Whether a `ghostftp-cli` binary actually exists in `bin_dir` (so "Add to PATH"
    /// is meaningful — otherwise the user should Install it first).
    pub bin_has_cli: bool,
    /// Whether `ghostftp-cli` resolves on PATH right now (`where`/`which`), and where.
    pub on_path: bool,
    pub cli_location: Option<String>,
    /// Whether Ghost FTP's managed entry is present (the `bin/` dir on the Windows
    /// per-user Path, or the `~/.local/bin` symlink on Unix).
    pub managed: bool,
    /// A short platform note for the UI ("Open a new terminal to pick it up", the
    /// shell profile touched, …).
    pub detail: Option<String>,
}

// ---------------------------------------------------------------------------
// Pure PATH-string helpers (platform-agnostic, unit-tested). Windows separates
// entries with ';'; splitting and re-joining on ';' round-trips byte-for-byte
// (empty entries included), so add = push / remove = retain both preserve every
// other entry exactly — the invariant the plan's "rest of PATH byte-identical"
// verification checks.
// ---------------------------------------------------------------------------

/// Normalize one entry for *comparison only* (never for storage): trim
/// surrounding whitespace, drop a trailing slash/backslash, and lowercase
/// (Windows paths are case-insensitive). Storage always keeps the original bytes.
fn normalize_entry(e: &str) -> String {
    // Windows is case-insensitive. Ghost FTP previously used both "Ghost FTP"
    // and "GhostFTP" for its own app directory. Canonicalize only that product
    // path segment so upgrades stay idempotent without changing unrelated paths.
    let normalized = e
        .trim()
        .trim_end_matches(['\\', '/'])
        .replace('/', "\\")
        .to_ascii_lowercase();

    normalized
        .split('\\')
        .map(|segment| {
            if segment == "ghost ftp" {
                "ghostftp"
            } else {
                segment
            }
        })
        .collect::<Vec<_>>()
        .join("\\")
}

/// Is `dir` already present as one of the ';'-separated entries in `value`?
#[cfg(any(windows, test))]
fn contains_dir(value: &str, dir: &str) -> bool {
    let target = normalize_entry(dir);
    !target.is_empty() && value.split(';').any(|e| normalize_entry(e) == target)
}

/// Append `dir` as a new entry unless it's already present. Preserves every
/// existing entry (and any empty ones) verbatim.
#[cfg(any(windows, test))]
fn add_dir(value: &str, dir: &str) -> String {
    if contains_dir(value, dir) {
        return value.to_string();
    }
    if value.is_empty() {
        return dir.to_string();
    }
    let mut entries: Vec<&str> = value.split(';').collect();
    entries.push(dir);
    entries.join(";")
}

/// Remove every entry equal to `dir` (normalized), leaving all others byte-identical.
#[cfg(any(windows, test))]
fn remove_dir(value: &str, dir: &str) -> String {
    let target = normalize_entry(dir);
    value
        .split(';')
        .filter(|e| normalize_entry(e) != target)
        .collect::<Vec<_>>()
        .join(";")
}

// ---------------------------------------------------------------------------
// Shared: locate the app-owned bin dir + the CLI on PATH.
// ---------------------------------------------------------------------------

fn cli_exe_name() -> &'static str {
    if cfg!(windows) {
        "ghostftp-cli.exe"
    } else {
        "ghostftp-cli"
    }
}

/// `<app_data_dir>/bin` — the same dir `cli_updater::install_missing` writes into.
fn app_bin_dir(app: &AppHandle) -> Result<PathBuf, String> {
    Ok(app
        .path()
        .app_data_dir()
        .map_err(|e| format!("resolve app_data_dir: {e}"))?
        .join("bin"))
}

/// First `ghostftp-cli` hit on PATH via `where` (Windows) / `which` (Unix), if any.
fn which_ghostftp_cli() -> Option<PathBuf> {
    let finder = if cfg!(windows) { "where" } else { "which" };
    let out = std::process::Command::new(finder)
        .arg("ghostftp-cli")
        .output()
        .ok()?;
    if !out.status.success() {
        return None;
    }
    String::from_utf8_lossy(&out.stdout)
        .lines()
        .map(|l| l.trim())
        .find(|l| !l.is_empty())
        .map(PathBuf::from)
}

/// Compute the current status without mutating anything.
fn compute_status(app: &AppHandle) -> Result<PathStatus, String> {
    let bin_dir = app_bin_dir(app)?;
    let bin_has_cli = bin_dir.join(cli_exe_name()).is_file();
    let located = which_ghostftp_cli();
    let managed = is_managed(&bin_dir);
    Ok(PathStatus {
        bin_dir: bin_dir.to_string_lossy().to_string(),
        bin_has_cli,
        on_path: located.is_some(),
        cli_location: located.map(|p| p.to_string_lossy().to_string()),
        managed,
        detail: None,
    })
}

// ---------------------------------------------------------------------------
// Windows implementation — HKCU\Environment, no admin.
// ---------------------------------------------------------------------------

#[cfg(windows)]
mod platform {
    use super::*;
    use winreg::enums::*;
    use winreg::types::FromRegValue;
    use winreg::{RegKey, RegValue};

    /// Open HKCU\Environment for read+write (created if somehow absent).
    fn open_env() -> Result<RegKey, String> {
        let hkcu = RegKey::predef(HKEY_CURRENT_USER);
        let (env, _) = hkcu
            .create_subkey("Environment")
            .map_err(|e| format!("open HKCU\\Environment: {e}"))?;
        Ok(env)
    }

    /// Read the per-user `Path` as `(value, type)`. A missing value reads as an
    /// empty `REG_EXPAND_SZ` (the type Windows uses for Path), so a first write
    /// creates it with the right, `%VAR%`-expanding type.
    fn read_path(env: &RegKey) -> (String, RegType) {
        match env.get_raw_value("Path") {
            Ok(rv) => {
                let s = String::from_reg_value(&rv).unwrap_or_default();
                (s, rv.vtype)
            }
            Err(_) => (String::new(), RegType::REG_EXPAND_SZ),
        }
    }

    /// Encode a string back into a raw registry value of the given type
    /// (UTF-16LE + null terminator — what REG_SZ / REG_EXPAND_SZ expect).
    fn make_value(s: &str, vtype: RegType) -> RegValue {
        let mut bytes: Vec<u8> = Vec::with_capacity((s.len() + 1) * 2);
        for u in s.encode_utf16() {
            bytes.extend_from_slice(&u.to_le_bytes());
        }
        bytes.extend_from_slice(&[0, 0]);
        RegValue { bytes, vtype }
    }

    /// winreg `RegType` → its Win32 discriminant, for the ghostftp.db backup record.
    fn vtype_u32(t: &RegType) -> u32 {
        match t {
            RegType::REG_NONE => 0,
            RegType::REG_SZ => 1,
            RegType::REG_EXPAND_SZ => 2,
            RegType::REG_BINARY => 3,
            RegType::REG_DWORD => 4,
            RegType::REG_MULTI_SZ => 7,
            _ => 2, // treat anything unexpected as expand-string (Path's real type)
        }
    }

    pub(super) fn is_managed(bin_dir: &std::path::Path) -> bool {
        let env = match open_env() {
            Ok(e) => e,
            Err(_) => return false,
        };
        let (value, _) = read_path(&env);
        super::contains_dir(&value, &bin_dir.to_string_lossy())
    }

    pub(super) fn add(app: &AppHandle, db: &crate::db::Db) -> Result<PathStatus, String> {
        let bin_dir = app_bin_dir(app)?;
        std::fs::create_dir_all(&bin_dir)
            .map_err(|e| format!("create {}: {e}", bin_dir.display()))?;
        let bin_str = bin_dir.to_string_lossy().to_string();

        let env = open_env()?;
        let (current, vtype) = read_path(&env);

        // Snapshot the true prior value ONCE, before we ever touch it, so a later
        // remove can restore it faithfully.
        db.env_backup_set_once(
            WIN_PATH_BACKUP_KEY,
            if current.is_empty() {
                None
            } else {
                Some(current.as_str())
            },
            vtype_u32(&vtype),
        )
        .map_err(|e| format!("back up prior PATH: {e}"))?;

        if !super::contains_dir(&current, &bin_str) {
            let next = super::add_dir(&current, &bin_str);
            env.set_raw_value("Path", &make_value(&next, vtype))
                .map_err(|e| format!("write HKCU\\Environment\\Path: {e}"))?;
            broadcast_environment_change();
        }

        let mut status = super::compute_status(app)?;
        status.detail = Some(
            "Added to your account's PATH. Open a new terminal for `ghostftp-cli` to \
             resolve (already-open terminals need a restart)."
                .into(),
        );
        Ok(status)
    }

    pub(super) fn remove(app: &AppHandle, _db: &crate::db::Db) -> Result<PathStatus, String> {
        let bin_dir = app_bin_dir(app)?;
        let bin_str = bin_dir.to_string_lossy().to_string();

        let env = open_env()?;
        let (current, vtype) = read_path(&env);
        if super::contains_dir(&current, &bin_str) {
            // Surgically drop only our entry from the CURRENT value (preserving
            // any changes the user made elsewhere and the value's type), rather
            // than blind-restoring the backup.
            let next = super::remove_dir(&current, &bin_str);
            env.set_raw_value("Path", &make_value(&next, vtype))
                .map_err(|e| format!("write HKCU\\Environment\\Path: {e}"))?;
            broadcast_environment_change();
        }

        let mut status = super::compute_status(app)?;
        status.detail = Some("Removed from your account's PATH.".into());
        Ok(status)
    }

    /// Tell already-running shells (via Explorer) to reload the environment block
    /// so terminals spawned afterwards see the new PATH without a logoff.
    fn broadcast_environment_change() {
        use windows_sys::Win32::UI::WindowsAndMessaging::{
            SendMessageTimeoutW, HWND_BROADCAST, SMTO_ABORTIFHUNG, WM_SETTINGCHANGE,
        };
        let param: Vec<u16> = "Environment\0".encode_utf16().collect();
        unsafe {
            SendMessageTimeoutW(
                HWND_BROADCAST,
                WM_SETTINGCHANGE,
                0,
                param.as_ptr() as isize,
                SMTO_ABORTIFHUNG,
                5000,
                std::ptr::null_mut(),
            );
        }
    }

    #[cfg(test)]
    mod win_tests {
        // `super::*` already re-exports platform's winreg imports (RegKey,
        // RegType, HKEY_CURRENT_USER, …).
        use super::*;

        /// Real registry round-trip against a **scratch** key (never the user's
        /// real Path): proves winreg preserves `REG_EXPAND_SZ` and leaves `%VARS%`
        /// unexpanded through our encode/decode + add/remove, and that removal is
        /// byte-identical. Safe and self-cleaning.
        #[test]
        fn scratch_key_preserves_type_and_round_trips() {
            let hkcu = RegKey::predef(HKEY_CURRENT_USER);
            let sub = "Software\\Ghost FTP\\PathIntegrationTest";
            let _ = hkcu.delete_subkey_all(sub); // clean any prior run
            let (key, _) = hkcu.create_subkey(sub).unwrap();

            // Seed an expand-string value with an unexpanded %VAR% (like a real Path).
            let original = r"C:\Windows;%JAVA_HOME%\bin";
            key.set_raw_value("Path", &make_value(original, RegType::REG_EXPAND_SZ))
                .unwrap();

            // Read it back through our decoder: value + type must survive.
            let (read, vtype) = read_path(&key);
            assert_eq!(read, original);
            assert_eq!(
                vtype,
                RegType::REG_EXPAND_SZ,
                "type must stay REG_EXPAND_SZ"
            );

            // Add our bin dir, write with the SAME type, read back.
            let bin = r"C:\Ghost FTP\bin";
            let added = super::super::add_dir(&read, bin);
            key.set_raw_value("Path", &make_value(&added, vtype))
                .unwrap();
            let (after_add, vtype2) = read_path(&key);
            assert_eq!(after_add, format!("{original};{bin}"));
            assert_eq!(vtype2, RegType::REG_EXPAND_SZ, "type preserved after add");
            assert!(after_add.contains("%JAVA_HOME%"), "%VARS% stay unexpanded");

            // Re-add is a no-op (no duplicate).
            assert_eq!(super::super::add_dir(&after_add, bin), after_add);

            // Remove restores the value byte-for-byte, still expand-string.
            let removed = super::super::remove_dir(&after_add, bin);
            key.set_raw_value("Path", &make_value(&removed, vtype2))
                .unwrap();
            let (after_remove, vtype3) = read_path(&key);
            assert_eq!(after_remove, original, "remove is byte-identical");
            assert_eq!(vtype3, RegType::REG_EXPAND_SZ);

            hkcu.delete_subkey_all(sub).unwrap();
        }

        /// The genuine end-to-end against the **real** `HKCU\Environment\Path`,
        /// using a synthetic (harmless, non-existent) marker dir so nothing real
        /// is affected. Ignored by default — run with
        /// `cargo test -p ghostftp real_environment_add_remove -- --ignored --nocapture`.
        /// Adds the marker, verifies exactly one entry appears and the type is
        /// unchanged, then removes it and verifies the whole value is byte-identical.
        #[test]
        #[ignore]
        fn real_environment_add_remove() {
            let marker = r"C:\__ghostftp_path_integration_test__";
            let env = open_env().unwrap();
            let (before, vtype_before) = read_path(&env);
            assert!(
                !super::super::contains_dir(&before, marker),
                "marker unexpectedly already present — aborting to protect PATH"
            );

            // --- add ---
            let added = super::super::add_dir(&before, marker);
            env.set_raw_value("Path", &make_value(&added, vtype_before.clone()))
                .unwrap();
            broadcast_environment_change();
            let (after_add, vtype_add) = read_path(&env);
            eprintln!("after add: {after_add}");
            assert!(super::super::contains_dir(&after_add, marker));
            assert_eq!(
                after_add.matches(marker).count(),
                1,
                "exactly one entry added"
            );
            assert_eq!(vtype_add, vtype_before, "value type unchanged by add");
            // A second add must not duplicate.
            assert_eq!(super::super::add_dir(&after_add, marker), after_add);

            // --- remove ---
            let removed = super::super::remove_dir(&after_add, marker);
            env.set_raw_value("Path", &make_value(&removed, vtype_add))
                .unwrap();
            broadcast_environment_change();
            let (after_remove, _) = read_path(&env);
            eprintln!("after remove: {after_remove}");
            assert_eq!(
                after_remove, before,
                "the rest of PATH is byte-identical after remove"
            );
        }
    }
}

// ---------------------------------------------------------------------------
// Linux implementation — ~/.local/bin symlink + marker-guarded profile.
// ---------------------------------------------------------------------------

/// The marker comments that fence Ghost FTP's line in a shell profile, so removal is
/// exact. Everything between (inclusive) is ours to delete; nothing else is.
#[cfg(unix)]
const PROFILE_BEGIN: &str = "# >>> ghostftp-cli PATH >>>";
#[cfg(unix)]
const PROFILE_END: &str = "# <<< ghostftp-cli PATH <<<";

/// Insert the marker-fenced block (idempotent — replaces an existing block).
/// Pure string surgery so it's unit-tested on any platform (used for real only on
/// Unix, but the tests exercise it everywhere).
#[cfg(any(unix, test))]
fn insert_marker_block(content: &str, begin: &str, end: &str, body: &str) -> String {
    let stripped = remove_marker_block(content, begin, end);
    let mut out = stripped.trim_end().to_string();
    if !out.is_empty() {
        out.push('\n');
    }
    out.push_str(begin);
    out.push('\n');
    out.push_str(body);
    out.push('\n');
    out.push_str(end);
    out.push('\n');
    out
}

/// Strip a marker-fenced block (and only that block) if present, leaving the rest
/// untouched. If markers are absent, returns the content unchanged.
#[cfg(any(unix, test))]
fn remove_marker_block(content: &str, begin: &str, end: &str) -> String {
    let mut out: Vec<&str> = Vec::new();
    let mut skipping = false;
    for line in content.lines() {
        if line.trim() == begin {
            skipping = true;
            continue;
        }
        if skipping {
            if line.trim() == end {
                skipping = false;
            }
            continue;
        }
        out.push(line);
    }
    let mut joined = out.join("\n");
    if content.ends_with('\n') && !joined.is_empty() {
        joined.push('\n');
    }
    joined
}

#[cfg(unix)]
mod platform {
    use super::*;

    fn local_bin() -> Result<PathBuf, String> {
        let home = dirs::home_dir().ok_or_else(|| "no home directory".to_string())?;
        Ok(home.join(".local").join("bin"))
    }

    fn symlink_path() -> Result<PathBuf, String> {
        Ok(local_bin()?.join("ghostftp-cli"))
    }

    /// The `ghostftp-cli` we should link to: a copy on PATH first, else the app-owned
    /// `bin/ghostftp-cli` that `install_missing` writes.
    fn link_target(app: &AppHandle) -> Option<PathBuf> {
        which_ghostftp_cli().or_else(|| {
            let p = app_bin_dir(app).ok()?.join("ghostftp-cli");
            p.is_file().then_some(p)
        })
    }

    /// Detect the user's shell profile to append the PATH export to.
    fn shell_profile() -> Option<PathBuf> {
        let home = dirs::home_dir()?;
        let shell = std::env::var("SHELL").unwrap_or_default();
        let name = if shell.contains("zsh") {
            ".zshrc"
        } else if shell.contains("bash") {
            ".bashrc"
        } else {
            ".profile"
        };
        Some(home.join(name))
    }

    fn local_bin_on_path() -> bool {
        let lb = match local_bin() {
            Ok(p) => p,
            Err(_) => return false,
        };
        std::env::var("PATH")
            .unwrap_or_default()
            .split(':')
            .any(|e| super::normalize_entry(e) == super::normalize_entry(&lb.to_string_lossy()))
    }

    pub(super) fn is_managed(_bin_dir: &std::path::Path) -> bool {
        symlink_path()
            .map(|p| p.symlink_metadata().is_ok())
            .unwrap_or(false)
    }

    pub(super) fn add(app: &AppHandle, _db: &crate::db::Db) -> Result<PathStatus, String> {
        let target = link_target(app)
            .ok_or_else(|| "ghostftp-cli isn't installed yet — install it first".to_string())?;
        let bin = local_bin()?;
        std::fs::create_dir_all(&bin).map_err(|e| format!("create {}: {e}", bin.display()))?;
        let link = symlink_path()?;
        // Replace any stale symlink so re-adding is idempotent.
        let _ = std::fs::remove_file(&link);
        std::os::unix::fs::symlink(&target, &link)
            .map_err(|e| format!("symlink {}: {e}", link.display()))?;

        let mut detail = format!("Linked ghostftp-cli into {}.", bin.display());
        if !local_bin_on_path() {
            if let Some(profile) = shell_profile() {
                let existing = std::fs::read_to_string(&profile).unwrap_or_default();
                let body = format!("export PATH=\"{}:$PATH\"", bin.display());
                let next = super::insert_marker_block(
                    &existing,
                    super::PROFILE_BEGIN,
                    super::PROFILE_END,
                    &body,
                );
                std::fs::write(&profile, next)
                    .map_err(|e| format!("write {}: {e}", profile.display()))?;
                detail = format!(
                    "Linked ghostftp-cli into {} and added it to {}. Open a new terminal \
                     (or `source` your profile) to pick it up.",
                    bin.display(),
                    profile.display()
                );
            }
        }

        let mut status = super::compute_status(app)?;
        status.detail = Some(detail);
        Ok(status)
    }

    pub(super) fn remove(app: &AppHandle, _db: &crate::db::Db) -> Result<PathStatus, String> {
        if let Ok(link) = symlink_path() {
            let _ = std::fs::remove_file(&link);
        }
        if let Some(profile) = shell_profile() {
            if let Ok(existing) = std::fs::read_to_string(&profile) {
                let next =
                    super::remove_marker_block(&existing, super::PROFILE_BEGIN, super::PROFILE_END);
                if next != existing {
                    std::fs::write(&profile, next)
                        .map_err(|e| format!("write {}: {e}", profile.display()))?;
                }
            }
        }
        let mut status = super::compute_status(app)?;
        status.detail = Some("Removed the ghostftp-cli symlink and profile line.".into());
        Ok(status)
    }
}

/// Platform dispatch for whether Ghost FTP's managed entry is present.
fn is_managed(bin_dir: &std::path::Path) -> bool {
    #[cfg(any(windows, unix))]
    {
        platform::is_managed(bin_dir)
    }
    #[cfg(not(any(windows, unix)))]
    {
        let _ = bin_dir;
        false
    }
}

// ---------------------------------------------------------------------------
// Tauri commands (Settings → About).
// ---------------------------------------------------------------------------

#[tauri::command]
pub async fn path_status(app: AppHandle) -> Result<PathStatus, String> {
    compute_status(&app)
}

#[tauri::command]
pub async fn path_add(app: AppHandle, state: State<'_, AppState>) -> Result<PathStatus, String> {
    #[cfg(any(windows, unix))]
    {
        platform::add(&app, &state.db)
    }
    #[cfg(not(any(windows, unix)))]
    {
        let _ = (&app, &state);
        Err("PATH integration isn't supported on this platform".into())
    }
}

#[tauri::command]
pub async fn path_remove(app: AppHandle, state: State<'_, AppState>) -> Result<PathStatus, String> {
    #[cfg(any(windows, unix))]
    {
        platform::remove(&app, &state.db)
    }
    #[cfg(not(any(windows, unix)))]
    {
        let _ = (&app, &state);
        Err("PATH integration isn't supported on this platform".into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn contains_is_case_and_slash_insensitive() {
        let v = r"C:\Windows;C:\Users\me\AppData\Local\Ghost FTP\bin";
        assert!(contains_dir(v, r"c:\users\me\appdata\local\ghostftp\bin"));
        assert!(contains_dir(v, r"C:\Users\me\AppData\Local\Ghost FTP\bin\")); // trailing slash
        assert!(!contains_dir(
            v,
            r"C:\Users\me\AppData\Local\Ghost FTP\bin2"
        ));
        assert!(!contains_dir("", r"C:\x"));
    }

    #[test]
    fn add_is_idempotent_and_preserves_entries() {
        let v = r"C:\Windows;C:\Windows\System32";
        let added = add_dir(v, r"C:\Ghost FTP\bin");
        assert_eq!(added, r"C:\Windows;C:\Windows\System32;C:\Ghost FTP\bin");
        // Re-adding is a no-op (even with different case / trailing slash).
        assert_eq!(add_dir(&added, r"c:\ghostftp\bin\"), added);
    }

    #[test]
    fn add_into_empty_value() {
        assert_eq!(add_dir("", r"C:\Ghost FTP\bin"), r"C:\Ghost FTP\bin");
    }

    #[test]
    fn remove_is_surgical_and_byte_identical_round_trip() {
        let original = r"C:\Windows;C:\Windows\System32;%JAVA_HOME%\bin";
        let bin = r"C:\Ghost FTP\bin";
        let added = add_dir(original, bin);
        // Removing our entry restores the original value byte-for-byte.
        assert_eq!(remove_dir(&added, bin), original);
        // Removing when absent leaves the value untouched.
        assert_eq!(remove_dir(original, bin), original);
        // %VARS% are preserved unexpanded throughout.
        assert!(remove_dir(&added, bin).contains("%JAVA_HOME%"));
    }

    #[test]
    fn remove_preserves_empty_trailing_entry() {
        // A trailing ';' (empty entry) is preserved through an add/remove cycle.
        let original = "C:\\A;C:\\B;";
        let bin = "C:\\Ghost FTP\\bin";
        let added = add_dir(original, bin);
        assert_eq!(added, "C:\\A;C:\\B;;C:\\Ghost FTP\\bin");
        assert_eq!(remove_dir(&added, bin), original);
    }

    #[test]
    fn marker_block_insert_and_remove_round_trip() {
        let begin = "# >>> ghostftp-cli PATH >>>";
        let end = "# <<< ghostftp-cli PATH <<<";
        let profile = "export EDITOR=vim\nalias ll='ls -la'\n";
        let body = "export PATH=\"$HOME/.local/bin:$PATH\"";

        let with = insert_marker_block(profile, begin, end, body);
        assert!(with.contains(begin) && with.contains(end) && with.contains(body));
        assert!(with.contains("export EDITOR=vim"));

        // Inserting again replaces (doesn't duplicate) the block.
        let with2 = insert_marker_block(&with, begin, end, body);
        assert_eq!(with2.matches(begin).count(), 1);

        // Removing restores the original untouched content.
        let without = remove_marker_block(&with, begin, end);
        assert_eq!(without, profile);
    }

    #[test]
    fn remove_marker_block_absent_is_noop() {
        let begin = "# >>> ghostftp-cli PATH >>>";
        let end = "# <<< ghostftp-cli PATH <<<";
        let profile = "export EDITOR=vim\n";
        assert_eq!(remove_marker_block(profile, begin, end), profile);
    }
}
