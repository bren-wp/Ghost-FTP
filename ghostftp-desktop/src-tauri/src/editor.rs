//! Edit-in-place: download a remote file to a temp dir, open it in the
//! user's default editor, watch for saves, and upload back. Modelled on the
//! FileZilla "View/Edit" flow.
//!
//! Lifecycle:
//!   1. Frontend calls `start_edit(session_id, remote_path)`.
//!   2. Backend downloads the file to `tempdir/ghostftp-edits/{uuid}/{basename}`.
//!   3. Backend opens the tempfile with the OS default editor.
//!   4. A `notify` watcher monitors the tempfile's parent dir. On every Modify
//!      event we debounce 500 ms, then upload the tempfile back to
//!      `remote_path` and emit `editor://saved`.
//!   5. Frontend can call `stop_edit(edit_id)` to clean up; or the manager's
//!      Drop tears down the watcher when the AppState goes away.

use crate::session::Session;
use anyhow::{Context, Result};
use notify::{Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::io::{Read, Write};
use std::collections::HashMap;
#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::{Duration, SystemTime};
use tauri::{AppHandle, Emitter};
use tokio::sync::Mutex;
use uuid::Uuid;

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct EditSavedEvent {
    pub edit_id: String,
    pub remote_path: String,
    pub bytes: u64,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct EditStartedEvent {
    pub edit_id: String,
    pub session_id: String,
    pub remote_path: String,
    pub local_temp_path: String,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct EditErrorEvent {
    pub edit_id: String,
    pub remote_path: String,
    pub message: String,
}

const MAX_BUILTIN_EDITOR_BYTES: usize = 2 * 1024 * 1024;
const MAX_EDITOR_FINGERPRINT_BYTES: u64 = 8 * 1024 * 1024;

/// Bounded content fingerprints detect rapid edits even on filesystems whose
/// modification timestamps do not advance for same-size writes. Files larger
/// than the limit retain the existing size+mtime fallback.
#[derive(Clone, Debug, PartialEq, Eq)]
struct EditorFingerprint {
    modified: SystemTime,
    size: u64,
    digest: Option<[u8; 32]>,
}

fn editor_fingerprint(path: &std::path::Path) -> Result<EditorFingerprint> {
    let meta = std::fs::symlink_metadata(path)
        .with_context(|| format!("lstat editor copy {}", path.display()))?;
    anyhow::ensure!(
        meta.file_type().is_file(),
        "editor refuses symbolic links and special files"
    );
    let digest = if meta.len() <= MAX_EDITOR_FINGERPRINT_BYTES {
        let mut file = std::fs::File::open(path).context("read editor copy fingerprint")?;
        let mut hasher = Sha256::new();
        let mut buf = [0u8; 64 * 1024];
        loop {
            let count = file.read(&mut buf).context("hash editor copy")?;
            if count == 0 {
                break;
            }
            hasher.update(&buf[..count]);
        }
        Some(hasher.finalize().into())
    } else {
        None
    };
    Ok(EditorFingerprint {
        modified: meta.modified().context("stat editor modification time")?,
        size: meta.len(),
        digest,
    })
}

/// A failed local write must never truncate the only editable copy.
/// Create a unique sibling, sync its bytes, then atomically replace the real
/// file; a failed rename leaves the original untouched and discards staging.
fn write_editor_copy_atomically(path: &std::path::Path, content: &[u8]) -> Result<()> {
    let meta = std::fs::symlink_metadata(path).context("lstat local edit target")?;
    anyhow::ensure!(meta.file_type().is_file(), "editor target is not a regular file");
    let parent = path.parent().context("editor target has no directory")?;
    let staged = parent.join(format!(".ghostftp-edit-{}.tmp", Uuid::new_v4()));
    let mut options = std::fs::OpenOptions::new();
    options.write(true).create_new(true);
    let result = (|| -> Result<()> {
        let mut temp = options.open(&staged).context("create editor staging copy")?;
        #[cfg(unix)]
        std::fs::set_permissions(&staged, std::fs::Permissions::from_mode(0o600))
            .context("secure editor staging copy")?;
        temp.write_all(content).context("write editor staging copy")?;
        temp.sync_all().context("flush edited data to disk")?;
        // Fail closed if the original became a symlink while staging.
        let current = std::fs::symlink_metadata(path).context("recheck edited target")?;
        anyhow::ensure!(current.file_type().is_file(), "editor target changed file type");
        std::fs::rename(&staged, path).context("commit edited copy")?;
        Ok(())
    })();
    if result.is_err() {
        let _ = std::fs::remove_file(&staged);
    }
    result
}

struct EditState {
    /// Held only to keep the watcher alive. Dropping it stops the watcher.
    _watcher: RecommendedWatcher,
    tempdir: PathBuf,
    local_path: PathBuf,
    write_gate: Arc<Mutex<()>>,
}

pub struct EditManager {
    sessions: Mutex<HashMap<String, EditState>>,
}

impl EditManager {
    pub fn new() -> Self {
        Self {
            sessions: Mutex::new(HashMap::new()),
        }
    }

    pub async fn start(
        &self,
        session: Arc<Session>,
        session_id: String,
        remote_path: String,
        editor: Option<String>,
        builtin: bool,
        app: AppHandle,
    ) -> Result<EditStartedEvent> {
        let edit_id = Uuid::new_v4().to_string();
        let parent = std::env::temp_dir().join("ghostftp-edits");
        std::fs::create_dir_all(&parent).context("create private editor directory")?;
        let meta = std::fs::symlink_metadata(&parent).context("lstat editor directory")?;
        anyhow::ensure!(
            meta.file_type().is_dir(),
            "editor directory is not a real directory"
        );
        // Editor temp copies may contain .env, configuration secrets or source
        // code. The default Unix mkdir umask can allow other users to read
        // them; ensure no group/other traversal before downloading any data.
        #[cfg(unix)]
        std::fs::set_permissions(&parent, std::fs::Permissions::from_mode(0o700))
            .context("secure editor parent directory")?;
        let base = parent.join(&edit_id);
        std::fs::create_dir(&base).context("create unique editor workspace")?;
        #[cfg(unix)]
        std::fs::set_permissions(&base, std::fs::Permissions::from_mode(0o700))
            .context("secure editor workspace")?;

        let name = remote_path
            .rsplit(['/', '\\'])
            .next()
            .filter(|s| !s.is_empty())
            .unwrap_or("file");
        let local_path = base.join(name);

        // Download the file into the tempfile.
        download_to(&session, &remote_path, &local_path)
            .await
            .with_context(|| format!("downloading {remote_path}"))?;

        // A content-aware baseline prevents our initial download from being
        // uploaded again, and catches equal-mtime edits from native editors.
        let initial_fingerprint = editor_fingerprint(&local_path)
            .context("fingerprint remote editor copy")?;

        // A bounded wake-up slot prevents file editors with very noisy
        // filesystem events from allocating unbounded notification memory.
        // Filter sibling .swp/lock files on the notify thread and coalesce
        // writes. A lost watcher event gets a periodic 10s reconciliation.
        let (tx, mut rx) = tokio::sync::mpsc::channel::<()>(1);
        let target_file = local_path
            .file_name()
            .map(|s| s.to_owned())
            .unwrap_or_default();
        let watched_name = target_file.clone();
        let mut watcher: RecommendedWatcher = RecommendedWatcher::new(
            move |res: notify::Result<Event>| match res {
                Ok(event)
                    if is_write_event(&event)
                        && event
                            .paths
                            .iter()
                            .any(|p| p.file_name() == Some(&watched_name)) =>
                {
                    let _ = tx.try_send(());
                }
                Err(error) => {
                    tracing::warn!("remote edit watcher error, scheduling rescan: {error}");
                    let _ = tx.try_send(());
                }
                _ => {}
            },
            notify::Config::default(),
        )
        .context("creating notify watcher")?;
        // Watch the parent dir so atomic-rename saves (write to .swp, rename
        // over) still trigger us.
        watcher
            .watch(&base, RecursiveMode::NonRecursive)
            .context("watching tempdir")?;

        // Spawn the debounce + upload task. Lives as long as the watcher
        // sends events; exits cleanly when the watcher is dropped.
        let upload_session = session.clone();
        let upload_remote = remote_path.clone();
        let upload_local = local_path.clone();
        let edit_id_for_task = edit_id.clone();
        let app_for_task = app.clone();
        tokio::spawn(async move {
            let mut last_uploaded = initial_fingerprint;
            let debounce = Duration::from_millis(500);

            loop {
                // Periodic catch-up also retries an upload after a transient
                // server failure with no further editor save event.
                match tokio::time::timeout(Duration::from_secs(10), rx.recv()).await {
                    Ok(Some(())) => {
                        let started = tokio::time::Instant::now();
                        loop {
                            if started.elapsed() >= Duration::from_secs(5) {
                                break;
                            }
                            match tokio::time::timeout(debounce, rx.recv()).await {
                                Ok(Some(())) => continue,
                                Ok(None) => return,
                                Err(_) => break,
                            }
                        }
                    }
                    Ok(None) => return,
                    Err(_) => {} // idle retry/reconciliation
                }

                // Never auto-upload symbolic links, special nodes or missing
                // targets. A digest catches a same-size edit with unchanged
                // timestamp; a failed remote upload retains the old baseline.
                let fingerprint = match editor_fingerprint(&upload_local) {
                    Ok(value) => value,
                    Err(error) => {
                        tracing::warn!("editor fingerprint unavailable: {error:#}");
                        continue;
                    }
                };
                if fingerprint == last_uploaded {
                    continue;
                }

                match upload_from(
                    &upload_session,
                    upload_local.to_string_lossy().as_ref(),
                    &upload_remote,
                )
                .await
                {
                    Ok(bytes) => {
                        // Only successful uploads acknowledge a local version.
                        // A failed remote write must stay dirty for the next
                        // periodic retry; never claim the save is complete.
                        last_uploaded = fingerprint;
                        let _ = app_for_task.emit(
                            "editor://saved",
                            EditSavedEvent {
                                edit_id: edit_id_for_task.clone(),
                                remote_path: upload_remote.clone(),
                                bytes,
                            },
                        );
                    }
                    Err(e) => {
                        let _ = app_for_task.emit(
                            "editor://error",
                            EditErrorEvent {
                                edit_id: edit_id_for_task.clone(),
                                remote_path: upload_remote.clone(),
                                message: e.to_string(),
                            },
                        );
                    }
                }
            }
        });

        // Spawn the editor — non-blocking; we keep watching regardless of
        // whether the editor process exits. Uses the configured editor command
        // when set, else the OS default app. If launch fails, explicitly drop
        // the watcher and remove the downloaded temp copy; the edit was never
        // registered in `sessions`, so stop() cannot clean it up later.
        if !builtin {
            if let Err(error) = spawn_editor(&local_path, editor.as_deref()) {
                drop(watcher);
                let _ = std::fs::remove_dir_all(&base);
                return Err(error);
            }
        }

        let started = EditStartedEvent {
            edit_id: edit_id.clone(),
            session_id,
            remote_path,
            local_temp_path: local_path.to_string_lossy().into_owned(),
        };

        self.sessions.lock().await.insert(
            edit_id,
            EditState {
                _watcher: watcher,
                tempdir: base,
                local_path,
                write_gate: Arc::new(Mutex::new(())),
            },
        );
        Ok(started)
    }

    /// Only the current edit session's local temp copy may be read through IPC.
    pub async fn read_text(&self, edit_id: &str) -> Result<String> {
        let path = self
            .sessions
            .lock()
            .await
            .get(edit_id)
            .context("editing session is no longer active")?
            .local_path
            .clone();
        let meta = tokio::fs::symlink_metadata(&path)
            .await
            .context("stat edit copy")?;
        anyhow::ensure!(
            meta.file_type().is_file(),
            "edit copy is not a regular file"
        );
        anyhow::ensure!(
            meta.len() <= MAX_BUILTIN_EDITOR_BYTES as u64,
            "file is too large for the built-in editor"
        );
        let bytes = tokio::fs::read(&path).await.context("read edit copy")?;
        anyhow::ensure!(
            bytes.len() <= MAX_BUILTIN_EDITOR_BYTES,
            "file is too large for the built-in editor"
        );
        String::from_utf8(bytes).context("file is not UTF-8 text; use an external editor")
    }

    /// Local write only. The existing watcher publishes remote saved/error
    /// events; never present successful IPC completion as remote-save success.
    pub async fn write_text(&self, edit_id: &str, content: String) -> Result<()> {
        anyhow::ensure!(
            content.len() <= MAX_BUILTIN_EDITOR_BYTES,
            "edited content exceeds the 2 MiB built-in editor limit"
        );
        let (path, gate) = {
            let sessions = self.sessions.lock().await;
            let session = sessions
                .get(edit_id)
                .context("editing session is no longer active")?;
            (session.local_path.clone(), session.write_gate.clone())
        };
        let _guard = gate.lock().await;
        tokio::task::spawn_blocking(move || write_editor_copy_atomically(&path, content.as_bytes()))
            .await
            .context("join atomic editor write")?
    
    }

    pub async fn stop(&self, edit_id: &str) -> Result<()> {
        let removed = self.sessions.lock().await.remove(edit_id);
        if let Some(state) = removed {
            // Dropping `state` drops the watcher, which terminates the
            // debounce task on its next recv. Clean up the temp dir.
            let _ = std::fs::remove_dir_all(&state.tempdir);
        }
        Ok(())
    }
}

impl Default for EditManager {
    fn default() -> Self {
        Self::new()
    }
}

fn is_write_event(event: &Event) -> bool {
    matches!(
        event.kind,
        EventKind::Modify(_) | EventKind::Create(_) | EventKind::Remove(_)
    )
}

#[cfg(target_os = "windows")]
fn spawn_editor(path: &std::path::Path, editor: Option<&str>) -> Result<()> {
    if let Some(cmd) = editor.map(str::trim).filter(|c| !c.is_empty()) {
        // Never pass a configured editor through cmd.exe: Settings is
        // user-controlled and shell metacharacters must not become executable
        // syntax. Direct CreateProcess/PATH lookup keeps the editor value an
        // executable name/path and the file path a separate argument.
        let mut command = std::process::Command::new(cmd);
        crate::windows_process::hide_console(&mut command);
        command
            .arg(path)
            .spawn()
            .with_context(|| format!("spawn configured editor for {}", path.display()))?;
        return Ok(());
    }
    // Use Windows file association directly rather than `cmd /c start`.
    // Apart from avoiding shell parsing of hostile filenames, this also avoids
    // the transient console window a console-subsystem cmd.exe can create.
    let mut command = std::process::Command::new("rundll32.exe");
    crate::windows_process::hide_console(&mut command);
    command
        .arg("url.dll,FileProtocolHandler")
        .arg(path)
        .spawn()
        .with_context(|| format!("spawn editor for {}", path.display()))?;
    Ok(())
}

#[cfg(unix)]
fn spawn_editor(path: &std::path::Path, editor: Option<&str>) -> Result<()> {
    // Explicit setting wins, then $EDITOR, else xdg-open for the desktop.
    let chosen = editor
        .map(str::trim)
        .filter(|c| !c.is_empty())
        .map(str::to_owned)
        .or_else(|| std::env::var("EDITOR").ok().filter(|e| !e.is_empty()));
    if let Some(cmd) = chosen {
        std::process::Command::new(&cmd)
            .arg(path)
            .spawn()
            .with_context(|| format!("spawn editor `{cmd}` for {}", path.display()))?;
        return Ok(());
    }
    std::process::Command::new("xdg-open")
        .arg(path)
        .spawn()
        .with_context(|| format!("xdg-open {}", path.display()))?;
    Ok(())
}

/// Download from any backend to a local path. Mirrors the streaming logic in
/// transfer.rs but writes to a known destination instead of emitting progress.
async fn download_to(
    session: &Session,
    remote_path: &str,
    local_path: &std::path::Path,
) -> Result<()> {
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    if let Some(parent) = local_path.parent() {
        std::fs::create_dir_all(parent).ok();
    }

    match session {
        Session::Ssh(ssh) => {
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            let mut remote = sftp
                .open(remote_path)
                .await
                .with_context(|| format!("open sftp {remote_path}"))?;
            let mut local = tokio::fs::File::create(local_path).await?;
            let mut buf = vec![0u8; 64 * 1024];
            loop {
                let n = remote.read(&mut buf).await?;
                if n == 0 {
                    break;
                }
                local.write_all(&buf[..n]).await?;
            }
            local.flush().await?;
        }
        Session::Ftp(ftp) => {
            let ftp = ftp.clone();
            let remote = remote_path.to_string();
            let dest = local_path.to_path_buf();
            ftp.with_stream(move |stream| {
                let file = std::fs::File::create(&dest)
                    .with_context(|| format!("create {}", dest.display()))?;
                stream.retr_to_writer(&remote, std::io::BufWriter::new(file))?;
                Ok(())
            })
            .await?;
        }
        Session::Object(obj) => {
            use futures::StreamExt;
            let key = remote_path.trim_start_matches('/');
            let p = object_store::path::Path::from(key);
            let get = obj
                .store
                .get(&p)
                .await
                .with_context(|| format!("object get {key}"))?;
            let mut file = tokio::fs::File::create(local_path).await?;
            let mut stream = get.into_stream();
            while let Some(chunk) = stream.next().await {
                let chunk = chunk?;
                file.write_all(&chunk).await?;
            }
            file.flush().await?;
        }
        Session::Webdav(dav) => {
            use futures::StreamExt;
            let url = dav.url_for(remote_path, false);
            let resp = dav
                .request(reqwest::Method::GET, url)
                .send()
                .await
                .with_context(|| format!("webdav get {remote_path}"))?;
            if !resp.status().is_success() {
                return Err(anyhow::anyhow!(
                    "webdav get {remote_path}: HTTP {}",
                    resp.status().as_u16()
                ));
            }
            let mut file = tokio::fs::File::create(local_path).await?;
            let mut stream = resp.bytes_stream();
            while let Some(chunk) = stream.next().await {
                let chunk = chunk?;
                file.write_all(&chunk).await?;
            }
            file.flush().await?;
        }
        Session::Http(http) => {
            use futures::StreamExt;
            let url = http.url_for(remote_path, false);
            let resp = http
                .request(reqwest::Method::GET, url)
                .send()
                .await
                .with_context(|| format!("http get {remote_path}"))?;
            if !resp.status().is_success() {
                return Err(anyhow::anyhow!(
                    "http get {remote_path}: HTTP {}",
                    resp.status().as_u16()
                ));
            }
            let mut file = tokio::fs::File::create(local_path).await?;
            let mut stream = resp.bytes_stream();
            while let Some(chunk) = stream.next().await {
                let chunk = chunk?;
                file.write_all(&chunk).await?;
            }
            file.flush().await?;
        }
        Session::Dropbox(dbx) => {
            use futures::StreamExt;
            let arg = serde_json::json!({
                "path": crate::remotefs::dropbox::dropbox_api_path(remote_path)
            })
            .to_string();
            let resp = dbx.content_get("/2/files/download", &arg).await?;
            let mut file = tokio::fs::File::create(local_path).await?;
            let mut stream = resp.bytes_stream();
            while let Some(chunk) = stream.next().await {
                let chunk = chunk?;
                file.write_all(&chunk).await?;
            }
            file.flush().await?;
        }
        Session::OneDrive(od) => {
            use futures::StreamExt;
            let resp = od
                .get_stream(&crate::remotefs::onedrive::content_ref(remote_path))
                .await?;
            let mut file = tokio::fs::File::create(local_path).await?;
            let mut stream = resp.bytes_stream();
            while let Some(chunk) = stream.next().await {
                let chunk = chunk?;
                file.write_all(&chunk).await?;
            }
            file.flush().await?;
        }
        Session::GDrive(gd) => {
            use futures::StreamExt;
            let (file_id, _) = gd
                .resolve_item(remote_path)
                .await?
                .ok_or_else(|| anyhow::anyhow!("{remote_path}: not found"))?;
            let resp = gd
                .get_stream(&format!("/files/{file_id}?alt=media"))
                .await?;
            let mut file = tokio::fs::File::create(local_path).await?;
            let mut stream = resp.bytes_stream();
            while let Some(chunk) = stream.next().await {
                let chunk = chunk?;
                file.write_all(&chunk).await?;
            }
            file.flush().await?;
        }
        Session::Box(bx) => {
            use futures::StreamExt;
            let (file_id, _) = bx
                .resolve_item(remote_path)
                .await?
                .ok_or_else(|| anyhow::anyhow!("{remote_path}: not found"))?;
            let resp = bx.get_stream(&format!("/files/{file_id}/content")).await?;
            let mut file = tokio::fs::File::create(local_path).await?;
            let mut stream = resp.bytes_stream();
            while let Some(chunk) = stream.next().await {
                let chunk = chunk?;
                file.write_all(&chunk).await?;
            }
            file.flush().await?;
        }
        Session::Shopify(sh) => {
            let data = crate::remotefs::shopify::read_asset(sh, remote_path).await?;
            let mut file = tokio::fs::File::create(local_path).await?;
            file.write_all(&data).await?;
            file.flush().await?;
        }
        Session::HubSpot(hs) => {
            let data = crate::remotefs::hubspot::read_file(hs, remote_path).await?;
            let mut file = tokio::fs::File::create(local_path).await?;
            file.write_all(&data).await?;
            file.flush().await?;
        }
        Session::Dynamics(dynm) => {
            let data = crate::remotefs::dynamics::read_file(dynm, remote_path).await?;
            let mut file = tokio::fs::File::create(local_path).await?;
            file.write_all(&data).await?;
            file.flush().await?;
        }
        Session::Agent(agent) => {
            use base64::Engine as _;
            use ghostftp_agent_proto::msg::{Request, Response};
            let mut file = tokio::fs::File::create(local_path).await?;
            let mut offset: u64 = 0;
            loop {
                let resp = agent
                    .request(Request::ReadChunk {
                        path: remote_path.to_string(),
                        offset,
                        len: 0,
                    })
                    .await?;
                let (data, eof) = match resp {
                    Response::Chunk { data, eof } => (data, eof),
                    Response::Error { message, .. } => {
                        return Err(anyhow::anyhow!("read {remote_path}: {message}"))
                    }
                    other => {
                        return Err(anyhow::anyhow!("read {remote_path}: unexpected {other:?}"))
                    }
                };
                let bytes = base64::engine::general_purpose::STANDARD.decode(&data)?;
                if !bytes.is_empty() {
                    file.write_all(&bytes).await?;
                    offset += bytes.len() as u64;
                }
                if eof {
                    break;
                }
            }
            file.flush().await?;
        }
    }
    Ok(())
}

/// Upload a local file back to `remote_path`, returning the byte count. Writes
/// in place (replacing whatever is at `remote_path`) on every backend — also the
/// engine behind the Agent Bridge's `write` on protocols with no direct write op.
pub(crate) async fn upload_from(
    session: &Session,
    local_path: &str,
    remote_path: &str,
) -> Result<u64> {
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    let local = std::path::PathBuf::from(local_path);
    let metadata = tokio::fs::symlink_metadata(&local)
        .await
        .with_context(|| format!("lstat edited file {}", local.display()))?;
    anyhow::ensure!(
        metadata.file_type().is_file(),
        "edited file is a symbolic link or special node; refusing automatic upload"
    );
    let size = metadata.len();

    match session {
        Session::Ssh(ssh) => {
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            let mut remote = sftp
                .create(remote_path)
                .await
                .with_context(|| format!("create sftp {remote_path}"))?;
            let mut local_f = tokio::fs::File::open(&local).await?;
            let mut buf = vec![0u8; 64 * 1024];
            loop {
                let n = local_f.read(&mut buf).await?;
                if n == 0 {
                    break;
                }
                remote.write_all(&buf[..n]).await?;
            }
            remote.flush().await?;
        }
        Session::Ftp(ftp) => {
            let ftp = ftp.clone();
            let remote = remote_path.to_string();
            let src = local.clone();
            ftp.with_stream(move |stream| {
                let file =
                    std::fs::File::open(&src).with_context(|| format!("open {}", src.display()))?;
                let mut reader = std::io::BufReader::new(file);
                stream.put_from_reader(&remote, &mut reader)?;
                Ok(())
            })
            .await?;
        }
        Session::Object(obj) => {
            let key = remote_path.trim_start_matches('/');
            let p = object_store::path::Path::from(key);
            let mut file = tokio::fs::File::open(&local).await?;
            let mut buf = Vec::with_capacity(size as usize);
            file.read_to_end(&mut buf).await?;
            obj.store
                .put(&p, bytes::Bytes::from(buf).into())
                .await
                .with_context(|| format!("object put {key}"))?;
        }
        Session::Webdav(dav) => {
            use tokio_util::io::ReaderStream;
            let file = tokio::fs::File::open(&local).await?;
            let body = reqwest::Body::wrap_stream(ReaderStream::new(file));
            let url = dav.url_for(remote_path, false);
            let resp = dav
                .request(reqwest::Method::PUT, url)
                .header(reqwest::header::CONTENT_LENGTH, size)
                .body(body)
                .send()
                .await
                .with_context(|| format!("webdav put {remote_path}"))?;
            if !resp.status().is_success() {
                return Err(anyhow::anyhow!(
                    "webdav put {remote_path}: HTTP {}",
                    resp.status().as_u16()
                ));
            }
        }
        Session::Http(_) => {
            return Err(anyhow::anyhow!(
                "HTTP source is read-only — saving edits is not supported"
            ));
        }
        Session::Dropbox(dbx) => {
            use tokio_util::io::ReaderStream;
            let arg = serde_json::json!({
                "path": crate::remotefs::dropbox::dropbox_api_path(remote_path),
                "mode": "overwrite", "autorename": false, "mute": true
            })
            .to_string();
            let token = dbx.access_token().await?;
            let f = tokio::fs::File::open(&local).await?;
            let body = reqwest::Body::wrap_stream(ReaderStream::new(f));
            let resp = dbx
                .client
                .post(format!("{}/2/files/upload", dbx.content_base))
                .bearer_auth(&token)
                .header("Dropbox-API-Arg", &arg)
                .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
                .body(body)
                .send()
                .await
                .with_context(|| format!("dropbox upload {remote_path}"))?;
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow::anyhow!(
                    "dropbox upload {remote_path} ({code}): {text}"
                ));
            }
        }
        Session::OneDrive(od) => {
            use tokio_util::io::ReaderStream;
            let token = od.access_token().await?;
            let f = tokio::fs::File::open(&local).await?;
            let body = reqwest::Body::wrap_stream(ReaderStream::new(f));
            let resp = od
                .client
                .put(format!(
                    "{}{}",
                    od.graph_base,
                    crate::remotefs::onedrive::content_ref(remote_path)
                ))
                .bearer_auth(&token)
                .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
                .body(body)
                .send()
                .await
                .with_context(|| format!("onedrive upload {remote_path}"))?;
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow::anyhow!(
                    "onedrive upload {remote_path} ({code}): {text}"
                ));
            }
        }
        Session::GDrive(gd) => {
            // Edit-in-place always targets an existing file: update its media.
            let (file_id, _) = gd
                .resolve_item(remote_path)
                .await?
                .ok_or_else(|| anyhow::anyhow!("{remote_path}: not found"))?;
            let token = gd.access_token().await?;
            let bytes = tokio::fs::read(&local).await?;
            let resp = gd
                .client
                .patch(format!(
                    "{}/files/{file_id}?uploadType=media",
                    gd.upload_base
                ))
                .bearer_auth(&token)
                .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
                .body(bytes)
                .send()
                .await
                .with_context(|| format!("gdrive update {remote_path}"))?;
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow::anyhow!(
                    "gdrive upload {remote_path} ({code}): {text}"
                ));
            }
        }
        Session::Box(bx) => {
            // Edit-in-place uploads a new version of the existing file.
            let (file_id, _) = bx
                .resolve_item(remote_path)
                .await?
                .ok_or_else(|| anyhow::anyhow!("{remote_path}: not found"))?;
            let token = bx.access_token().await?;
            let bytes = tokio::fs::read(&local).await?;
            let part = reqwest::multipart::Part::bytes(bytes)
                .file_name(crate::session::boxdrive::basename(remote_path).to_string());
            let form = reqwest::multipart::Form::new().part("file", part);
            let resp = bx
                .client
                .post(format!("{}/files/{file_id}/content", bx.upload_base))
                .bearer_auth(&token)
                .multipart(form)
                .send()
                .await
                .with_context(|| format!("box update {remote_path}"))?;
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow::anyhow!("box upload {remote_path} ({code}): {text}"));
            }
        }
        Session::Shopify(sh) => {
            // Create and update are the same PUT in the Assets API.
            let data = tokio::fs::read(&local).await?;
            crate::remotefs::shopify::write_asset(sh, remote_path, &data).await?;
        }
        Session::HubSpot(hs) => {
            // Create and update are the same multipart PUT in the Source Code
            // API.
            let data = tokio::fs::read(&local).await?;
            crate::remotefs::hubspot::write_file(hs, remote_path, &data).await?;
        }
        Session::Dynamics(dynm) => {
            // Create or update by name lookup, then publish — save = deployed.
            let data = tokio::fs::read(&local).await?;
            crate::remotefs::dynamics::write_file(dynm, remote_path, &data).await?;
        }
        Session::Agent(agent) => {
            use base64::Engine as _;
            use ghostftp_agent_proto::msg::{Request, Response};
            let mut file = tokio::fs::File::open(&local).await?;
            let mut buf = vec![0u8; 128 * 1024];
            let mut offset: u64 = 0;
            let mut first = true;
            loop {
                let n = file.read(&mut buf).await?;
                if n == 0 {
                    break;
                }
                let data = base64::engine::general_purpose::STANDARD.encode(&buf[..n]);
                match agent
                    .request(Request::WriteChunk {
                        path: remote_path.to_string(),
                        offset,
                        data,
                        truncate: first,
                        done: false,
                    })
                    .await?
                {
                    Response::Written { .. } => {}
                    Response::Error { message, .. } => {
                        return Err(anyhow::anyhow!("write {remote_path}: {message}"))
                    }
                    other => {
                        return Err(anyhow::anyhow!("write {remote_path}: unexpected {other:?}"))
                    }
                }
                offset += n as u64;
                first = false;
            }
        }
    }
    Ok(size)
}

#[cfg(test)]
mod builtin_editor_integrity_tests {
    use super::*;

    #[test]
    fn same_timestamp_and_length_edit_changes_fingerprint() {
        let dir = std::env::temp_dir().join(format!("ghostftp-fingerprint-{}", Uuid::new_v4()));
        std::fs::create_dir(&dir).unwrap();
        let file = dir.join("site.php");
        std::fs::write(&file, b"hello").unwrap();
        let first = editor_fingerprint(&file).unwrap();
        std::fs::write(&file, b"world").unwrap();
        std::fs::File::open(&file).unwrap().set_modified(first.modified).unwrap();
        let second = editor_fingerprint(&file).unwrap();
        assert_eq!(first.modified, second.modified);
        assert_eq!(first.size, second.size);
        assert_ne!(first.digest, second.digest);
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[test]
    fn atomic_editor_save_replaces_content_without_stray_temp_files() {
        let dir = std::env::temp_dir().join(format!("ghostftp-editor-atomic-{}", Uuid::new_v4()));
        std::fs::create_dir(&dir).unwrap();
        let target = dir.join("index.html");
        std::fs::write(&target, b"previous").unwrap();
        write_editor_copy_atomically(&target, b"new content").unwrap();
        assert_eq!(std::fs::read(&target).unwrap(), b"new content");
        assert_eq!(std::fs::read_dir(&dir).unwrap().count(), 1);
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[cfg(unix)]
    #[test]
    fn editor_save_refuses_symbolic_link_target() {
        use std::os::unix::fs::symlink;
        let dir = std::env::temp_dir().join(format!("ghostftp-editor-symlink-{}", Uuid::new_v4()));
        std::fs::create_dir(&dir).unwrap();
        let secret = dir.join("secret");
        std::fs::write(&secret, b"protected").unwrap();
        let target = dir.join("edit.php");
        symlink(&secret, &target).unwrap();
        assert!(editor_fingerprint(&target).is_err());
        assert!(write_editor_copy_atomically(&target, b"overwrite").is_err());
        assert_eq!(std::fs::read(&secret).unwrap(), b"protected");
        std::fs::remove_dir_all(dir).unwrap();
    }
}
