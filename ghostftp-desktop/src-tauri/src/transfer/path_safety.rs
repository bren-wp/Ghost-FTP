//! Path and directory-inventory safety used by all download backends.
//! Keep these checks independent of transfer scheduling and UI state.
use anyhow::{Context, Result};
use std::path::{Path, PathBuf};

pub(super) const MAX_DIRECTORY_UPLOAD_ENTRIES: usize = 100_000;
pub(super) const MAX_DIRECTORY_UPLOAD_DEPTH: usize = 64;
pub(super) const MAX_DIRECTORY_UPLOAD_PATH_BYTES: usize = 16 * 1024 * 1024;

pub(super) fn validate_directory_upload_budget(
    entries: usize,
    depth: usize,
    bytes: usize,
) -> Result<()> {
    anyhow::ensure!(
        entries <= MAX_DIRECTORY_UPLOAD_ENTRIES,
        "directory upload entry limit exceeded"
    );
    anyhow::ensure!(
        depth <= MAX_DIRECTORY_UPLOAD_DEPTH,
        "directory upload depth limit exceeded"
    );
    anyhow::ensure!(
        bytes <= MAX_DIRECTORY_UPLOAD_PATH_BYTES,
        "directory upload path byte limit exceeded"
    );
    Ok(())
}

pub(super) fn safe_upload_name(name: &str) -> bool {
    !name.is_empty()
        && !matches!(name, "." | "..")
        && !name
            .chars()
            .any(|ch| matches!(ch, '/' | '\\' | '\0' | '\r' | '\n'))
}

/// Never materialize an untrusted remote leaf as a Windows device name,
/// alternate data stream, path separator or control-character filename.
/// Apply the same conservative policy on Linux for portable downloads.
pub(super) fn safe_local_download_leaf(name: &str) -> bool {
    if !safe_upload_name(name)
        || name.ends_with('.')
        || name.ends_with(' ')
        || name
            .chars()
            .any(|ch| ch.is_control() || matches!(ch, ':' | '*' | '?' | '"' | '<' | '>' | '|'))
    {
        return false;
    }
    let stem = name.split('.').next().unwrap_or("");
    let upper = stem.to_ascii_uppercase();
    if matches!(
        upper.as_str(),
        "CON" | "PRN" | "AUX" | "NUL" | "CONIN$" | "CONOUT$"
    ) {
        return false;
    }
    let mut chars = upper.chars();
    let prefix = chars.by_ref().take(3).collect::<String>();
    let suffix = chars.collect::<String>();
    // Windows reserves COM1–COM9 and LPT1–LPT9, including superscript
    // ¹/²/³ variants. Return the inverse directly; keep the same validation.
    let reserved_serial_device = matches!(prefix.as_str(), "COM" | "LPT")
        && suffix.chars().count() == 1
        && suffix
            .chars()
            .next()
            .is_some_and(|digit| matches!(digit, '1'..='9' | '¹' | '²' | '³'));
    !reserved_serial_device
}

// Treat every server listing as untrusted. A sibling name, parent segment
// or drive/ADS separator must never escape the user-selected download root.
pub(super) fn checked_remote_download_relative(root: &str, child: &str) -> Result<PathBuf> {
    anyhow::ensure!(
        root.contains('\\') || !child.contains('\\'),
        "ambiguous server path separators are unsupported"
    );
    let normalized_root = root.replace('\\', "/");
    let normalized_child = child.replace('\\', "/");
    let prefix = normalized_root.trim_end_matches('/');
    // "/" is a valid remote directory root: the empty prefix still requires
    // an absolute child path and every component is checked below. Reject
    // empty and repeated-slash roots instead of disabling boundary checks.
    anyhow::ensure!(
        !prefix.is_empty() || normalized_root == "/",
        "download root is invalid"
    );
    let remainder = normalized_child
        .strip_prefix(prefix)
        .filter(|suffix| suffix.starts_with('/'))
        .context("server returned a path outside the selected download directory")?;
    let rel = remainder
        .strip_prefix('/')
        .context("invalid child path separator")?;
    anyhow::ensure!(!rel.is_empty(), "server returned the root as a child file");
    let mut relative = PathBuf::new();
    for component in rel.split('/') {
        anyhow::ensure!(
            safe_local_download_leaf(component),
            "server returned an unsafe download filename"
        );
        relative.push(component);
    }
    Ok(relative)
}

/// Refuse pre-existing symlinked directory components while preparing a
/// download tree. Never let create_dir_all follow a local link into files
/// outside the destination chosen by the user.
pub(super) async fn ensure_download_subdirectory(root: &Path, dir: &Path) -> Result<()> {
    let rel = dir
        .strip_prefix(root)
        .context("download directory escaped the selected root")?;
    let meta = tokio::fs::symlink_metadata(root)
        .await
        .context("inspect download root")?;
    anyhow::ensure!(
        meta.file_type().is_dir(),
        "download destination root must be a real directory"
    );
    let mut current = root.to_path_buf();
    for part in rel.components() {
        anyhow::ensure!(
            matches!(part, std::path::Component::Normal(_)),
            "download destination contains a non-normal path component"
        );
        current.push(part.as_os_str());
        match tokio::fs::symlink_metadata(&current).await {
            Ok(meta) => {
                anyhow::ensure!(
                    meta.file_type().is_dir(),
                    "download destination component is not a real directory"
                );
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                match tokio::fs::create_dir(&current).await {
                    Ok(()) => {}
                    // Concurrent downloads can both observe a missing directory.
                    // Accept the second creation only after verifying the winner
                    // made a real directory, never a link or a regular file.
                    Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                        let raced =
                            tokio::fs::symlink_metadata(&current)
                                .await
                                .with_context(|| {
                                    format!("recheck raced directory {}", current.display())
                                })?;
                        anyhow::ensure!(
                            raced.file_type().is_dir(),
                            "download destination component is not a real directory"
                        );
                    }
                    Err(error) => {
                        return Err(error).with_context(|| {
                            format!("create download directory {}", current.display())
                        });
                    }
                }
            }
            Err(e) => {
                return Err(e)
                    .with_context(|| format!("inspect download directory {}", current.display()));
            }
        }
    }
    Ok(())
}

pub(super) const MAX_DIRECTORY_DOWNLOAD_ENTRIES: usize = 100_000;
pub(super) const MAX_DIRECTORY_DOWNLOAD_DEPTH: usize = 64;
pub(super) const MAX_DIRECTORY_DOWNLOAD_PATH_BYTES: usize = 16 * 1024 * 1024;

pub(super) fn validate_directory_download_budget(
    entries: usize,
    depth: usize,
    path_bytes: usize,
) -> Result<()> {
    anyhow::ensure!(
        entries <= MAX_DIRECTORY_DOWNLOAD_ENTRIES,
        "directory download entry budget exceeded"
    );
    anyhow::ensure!(
        depth <= MAX_DIRECTORY_DOWNLOAD_DEPTH,
        "directory download depth budget exceeded"
    );
    anyhow::ensure!(
        path_bytes <= MAX_DIRECTORY_DOWNLOAD_PATH_BYTES,
        "directory download path budget exceeded"
    );
    Ok(())
}

pub(crate) fn require_directory_upload_metadata(
    path: &Path,
    metadata: std::io::Result<std::fs::Metadata>,
) -> Result<std::fs::Metadata> {
    metadata.with_context(|| {
        format!(
            "stat directory upload entry {}; refusing to continue with an incomplete source tree",
            path.display()
        )
    })
}

/// Never follow directory-upload source links. DirEntry::metadata follows
/// symlinks on supported platforms and could turn an out-of-tree target into
/// an apparently ordinary file or directory. Fail before any remote writes.
pub(super) async fn inspect_directory_upload_entry(path: &Path) -> Result<std::fs::Metadata> {
    let metadata =
        require_directory_upload_metadata(path, tokio::fs::symlink_metadata(path).await)?;
    anyhow::ensure!(
        metadata.file_type().is_dir() || metadata.file_type().is_file(),
        "directory upload refused: symbolic link or special entry {}",
        path.display()
    );
    Ok(metadata)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn directory_upload_entry_accepts_only_regular_files_and_directories() {
        let root =
            std::env::temp_dir().join(format!("ghostftp-upload-entry-{}", uuid::Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let file = root.join("normal.txt");
        tokio::fs::write(&file, b"upload these bytes")
            .await
            .unwrap();
        let subdir = root.join("subdirectory");
        tokio::fs::create_dir(&subdir).await.unwrap();
        assert!(inspect_directory_upload_entry(&file)
            .await
            .unwrap()
            .file_type()
            .is_file());
        assert!(inspect_directory_upload_entry(&subdir)
            .await
            .unwrap()
            .file_type()
            .is_dir());
        assert!(inspect_directory_upload_entry(&root.join("missing"))
            .await
            .is_err());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn directory_upload_does_not_follow_links_outside_selected_tree() {
        use std::os::unix::fs::symlink;
        let root =
            std::env::temp_dir().join(format!("ghostftp-upload-link-{}", uuid::Uuid::new_v4()));
        let chosen = root.join("chosen");
        let outside = root.join("private");
        tokio::fs::create_dir_all(&chosen).await.unwrap();
        tokio::fs::create_dir_all(&outside).await.unwrap();
        let secret = outside.join("secret.txt");
        tokio::fs::write(&secret, b"do not upload").await.unwrap();
        let file_link = chosen.join("secret.txt");
        let dir_link = chosen.join("private-folder");
        symlink(&secret, &file_link).unwrap();
        symlink(&outside, &dir_link).unwrap();
        assert!(inspect_directory_upload_entry(&file_link).await.is_err());
        assert!(inspect_directory_upload_entry(&dir_link).await.is_err());
        assert_eq!(tokio::fs::read(&secret).await.unwrap(), b"do not upload");
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[tokio::test]
    async fn parallel_directory_materialization_is_idempotent() {
        let root = std::env::temp_dir().join(format!(
            "ghostftp-concurrent-directory-{}",
            uuid::Uuid::new_v4()
        ));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let target = root.join("nested").join("reports");
        let mut tasks = Vec::new();
        for _ in 0..24 {
            let root = root.clone();
            let target = target.clone();
            tasks.push(tokio::spawn(async move {
                ensure_download_subdirectory(&root, &target).await
            }));
        }
        for task in tasks {
            task.await.unwrap().unwrap();
        }
        assert!(tokio::fs::symlink_metadata(&target)
            .await
            .unwrap()
            .file_type()
            .is_dir());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn download_subdirectory_never_accepts_existing_symlink() {
        use std::os::unix::fs::symlink;
        let root =
            std::env::temp_dir().join(format!("ghostftp-concurrent-link-{}", uuid::Uuid::new_v4()));
        let outside = root.join("outside");
        let selected = root.join("selected");
        tokio::fs::create_dir_all(&selected).await.unwrap();
        tokio::fs::create_dir_all(&outside).await.unwrap();
        symlink(&outside, selected.join("nested")).unwrap();

        assert!(
            ensure_download_subdirectory(&selected, &selected.join("nested").join("file"))
                .await
                .is_err()
        );
        assert!(!outside.join("file").exists());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }
}
