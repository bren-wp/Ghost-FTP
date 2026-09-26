//! Inert on-demand provider for every build that isn't Windows-with-`virtualfs`.
//!
//! Keeps the subsystem cross-platform: [`super::VirtualFs`] compiles and runs
//! everywhere, and an on-demand pair configured on an unsupported build degrades
//! to a clear "not available" (via `supported() == false`) instead of a compile
//! error. Mirrors the API surface of `windows.rs` exactly.

use std::path::Path;
use std::sync::Arc;

use anyhow::Result;
use tauri::AppHandle;

use super::Hydrator;

/// On-demand virtual folders are never available on this build.
pub fn supported() -> bool {
    false
}

/// A no-op provider handle. Never actually constructed (`Provider::start`
/// errors first), but the type must exist so `VirtualFs::running` type-checks.
pub struct Provider;

impl Provider {
    pub async fn start(
        _pair_id: String,
        _local_root: std::path::PathBuf,
        _remote_root: String,
        _hydrator: Arc<dyn Hydrator>,
        _app: AppHandle,
    ) -> Result<Provider> {
        anyhow::bail!("on-demand virtual folders need a Windows build with --features virtualfs")
    }

    pub async fn free_up_space(&self) -> Result<u32> {
        Ok(0)
    }

    pub fn stop(self) {}
}

/// Nothing was ever registered with the OS, so unregister is a no-op.
pub fn unregister_root(_pair_id: &str, _local_root: Option<&Path>) -> Result<()> {
    Ok(())
}
