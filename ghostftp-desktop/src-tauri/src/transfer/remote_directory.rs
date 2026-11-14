//! Verify remote directory creation before queuing folder-upload transfers.
//! FTP/SFTP servers differ in whether mkdir on an existing directory is an
//! error. A readable existing directory is safe; an inaccessible one is not.
use crate::remotefs::RemoteFs;
use anyhow::{Context, Result};

pub(super) async fn ensure_upload_directory(fs: &dyn RemoteFs, path: &str) -> Result<()> {
    match fs.create_dir(path).await {
        Ok(()) => Ok(()),
        Err(create_error) => {
            fs.list_dir(path).await.with_context(|| {
                format!("cannot create or list remote upload directory {path}: {create_error:#}")
            })?;
            Ok(())
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::remotefs::{Capabilities, ChangeSignal, DirEntry};
    use std::sync::atomic::{AtomicUsize, Ordering};

    struct FakeFs {
        create_ok: bool,
        list_ok: bool,
        list_calls: AtomicUsize,
    }

    #[async_trait::async_trait]
    impl RemoteFs for FakeFs {
        async fn list_dir(&self, _: &str) -> Result<Vec<DirEntry>> {
            self.list_calls.fetch_add(1, Ordering::SeqCst);
            anyhow::ensure!(self.list_ok, "permission or network failure");
            Ok(vec![])
        }
        async fn create_dir(&self, _: &str) -> Result<()> {
            anyhow::ensure!(self.create_ok, "already exists or permission denied");
            Ok(())
        }
        async fn rename(&self, _: &str, _: &str) -> Result<()> {
            Ok(())
        }
        async fn delete(&self, _: &str, _: bool) -> Result<()> {
            Ok(())
        }
        async fn chmod(&self, _: &str, _: u32) -> Result<()> {
            Ok(())
        }
        fn capabilities(&self) -> Capabilities {
            Capabilities {
                can_chmod: false,
                can_symlink: false,
                can_rename: false,
                has_directories: true,
                has_shell: false,
                change_signal: ChangeSignal::MtimeSize,
            }
        }
    }

    #[tokio::test]
    async fn upload_directory_creation_is_idempotent_but_never_swallows_errors() {
        for (create_ok, list_ok, expected_ok, expected_lists) in [
            (true, false, true, 0),
            (false, true, true, 1),
            (false, false, false, 1),
        ] {
            let fs = FakeFs {
                create_ok,
                list_ok,
                list_calls: AtomicUsize::new(0),
            };
            assert_eq!(
                ensure_upload_directory(&fs, "/remote/folder").await.is_ok(),
                expected_ok
            );
            assert_eq!(fs.list_calls.load(Ordering::SeqCst), expected_lists);
        }
    }
}
