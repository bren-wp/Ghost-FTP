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
use std::collections::HashMap;
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

struct EditState {
    /// Held only to keep the watcher alive. Dropping it stops the watcher.
    _watcher: RecommendedWatcher,
    tempdir: PathBuf,
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
        app: AppHandle,
    ) -> Result<EditStartedEvent> {
        let edit_id = Uuid::new_v4().to_string();
        let base = std::env::temp_dir().join("ghostftp-edits").join(&edit_id);
        std::fs::create_dir_all(&base).context("create temp dir for edit")?;

        let name = remote_path
            .rsplit(|c| c == '/' || c == '\\')
            .next()
            .filter(|s| !s.is_empty())
            .unwrap_or("file");
        let local_path = base.join(name);

        // Download the file into the tempfile.
        download_to(&session, &remote_path, &local_path)
            .await
            .with_context(|| format!("downloading {remote_path}"))?;

        // Record the mtime so the first watcher event triggered by *us* writing
        // the file doesn't immediately re-upload it.
        let initial_mtime = std::fs::metadata(&local_path)
            .ok()
            .and_then(|m| m.modified().ok())
            .unwrap_or(SystemTime::UNIX_EPOCH);

        // Channel from the sync notify thread → our async debounce task.
        let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel::<Event>();
        let mut watcher: RecommendedWatcher = RecommendedWatcher::new(
            move |res: notify::Result<Event>| {
                if let Ok(event) = res {
                    let _ = tx.send(event);
                }
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
        let target_file = local_path
            .file_name()
            .map(|s| s.to_owned())
            .unwrap_or_default();

        tokio::spawn(async move {
            let mut last_mtime = initial_mtime;
            let debounce = Duration::from_millis(500);

            loop {
                let event = match rx.recv().await {
                    Some(e) => e,
                    None => return, // watcher dropped
                };

                if !is_write_event(&event) {
                    continue;
                }
                // Only react to events touching our specific file, in case
                // the editor wrote sibling .swp / lockfiles.
                if !event.paths.iter().any(|p| p.file_name() == Some(&target_file)) {
                    continue;
                }

                // Debounce: collect further events for `debounce` ms before
                // acting. Editor saves can fan out into 3+ events in a row.
                loop {
                    match tokio::time::timeout(debounce, rx.recv()).await {
                        Ok(Some(_)) => continue, // another event arrived — keep waiting
                        Ok(None) => return,
                        Err(_) => break, // debounce window expired
                    }
                }

                // Skip if the mtime hasn't actually moved past last_mtime.
                let meta = match std::fs::metadata(&upload_local) {
                    Ok(m) => m,
                    Err(_) => continue,
                };
                let mtime = meta.modified().unwrap_or(SystemTime::UNIX_EPOCH);
                if mtime <= last_mtime {
                    continue;
                }
                last_mtime = mtime;

                match upload_from(
                    &upload_session,
                    upload_local.to_string_lossy().as_ref(),
                    &upload_remote,
                )
                .await
                {
                    Ok(bytes) => {
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
        // when set, else the OS default app.
        spawn_editor(&local_path, editor.as_deref())?;

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
            },
        );
        Ok(started)
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
        // Route through `cmd /c` so PATH shims like `code` (code.cmd) resolve;
        // a bare CreateProcess won't find a .cmd by name.
        std::process::Command::new("cmd")
            .args(["/c", cmd])
            .arg(path)
            .spawn()
            .with_context(|| format!("spawn editor `{cmd}` for {}", path.display()))?;
        return Ok(());
    }
    // `cmd /c start "" <path>` opens the file with its associated app.
    // The empty `""` is the window title — required because `start` interprets
    // a single quoted arg as the title, not the target.
    std::process::Command::new("cmd")
        .args(["/c", "start", ""])
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
            let resp = gd.get_stream(&format!("/files/{file_id}?alt=media")).await?;
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
                    .request(Request::ReadChunk { path: remote_path.to_string(), offset, len: 0 })
                    .await?;
                let (data, eof) = match resp {
                    Response::Chunk { data, eof } => (data, eof),
                    Response::Error { message, .. } => {
                        return Err(anyhow::anyhow!("read {remote_path}: {message}"))
                    }
                    other => return Err(anyhow::anyhow!("read {remote_path}: unexpected {other:?}")),
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
    let size = tokio::fs::metadata(&local)
        .await
        .with_context(|| format!("stat {}", local.display()))?
        .len();

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
                let file = std::fs::File::open(&src)
                    .with_context(|| format!("open {}", src.display()))?;
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
                return Err(anyhow::anyhow!("dropbox upload {remote_path} ({code}): {text}"));
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
                return Err(anyhow::anyhow!("onedrive upload {remote_path} ({code}): {text}"));
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
                .patch(format!("{}/files/{file_id}?uploadType=media", gd.upload_base))
                .bearer_auth(&token)
                .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
                .body(bytes)
                .send()
                .await
                .with_context(|| format!("gdrive update {remote_path}"))?;
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow::anyhow!("gdrive upload {remote_path} ({code}): {text}"));
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
                    other => return Err(anyhow::anyhow!("write {remote_path}: unexpected {other:?}")),
                }
                offset += n as u64;
                first = false;
            }
        }
    }
    Ok(size)
}

