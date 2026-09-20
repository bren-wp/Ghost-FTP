//! Remote image thumbnail previews (Plan 13 Phase 1).
//!
//! Ghost FTP thumbnails local images by slurping the bytes off disk. Remote images
//! can't be free like that — a preview means a download, and a naive
//! implementation that fires one download per row would turn scrolling a
//! 100k-file `wp-content/uploads` into 100k requests. The whole subsystem exists
//! to protect one invariant: **scrolling must never trigger unbounded requests.**
//!
//! The defenses, outermost first:
//! 1. **Opt-in, default off** for remote backends (the frontend setting gate).
//! 2. **Viewport-driven fetching** — the frontend only calls in for rows that
//!    scroll within ~1 screen (IntersectionObserver), and cancels before dispatch
//!    when a queued row scrolls back out.
//! 3. **Per-connection concurrency cap** ([`PER_SESSION_CONCURRENCY`]) — even when
//!    a whole screen scrolls into view at once, only this many fetches touch the
//!    shared session pool concurrently.
//! 4. **Size guard** — files over [`MAX_PREVIEW_FILE_BYTES`] are skipped, and the
//!    bounded read never pulls more than that many bytes off the network.
//! 5. **Disk cache** — a decoded+downscaled PNG per (connection, path, change
//!    signal), so reopening a folder costs zero network. The index lives in
//!    `ghostftp.db`; an LRU keeps the cache dir under [`CACHE_BUDGET_BYTES`].
//!
//! Decode + downscale happen in Rust (the `image` crate) off a blocking task, so
//! the cached artifact is already tiny; the frontend just wraps the bytes in a
//! `Blob`, matching the existing local-preview path.

use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::sync::Arc;

use anyhow::{bail, Context, Result};
use base64::Engine as _;
use futures::StreamExt;
use tokio::sync::{Mutex, Semaphore};

use crate::session::Session;

/// Largest remote file we'll pull for a thumbnail (and the hard ceiling on the
/// bounded read). Bigger files fall back to the type icon. Most web images are
/// well under this; the cap is a backstop, not a target.
pub const MAX_PREVIEW_FILE_BYTES: u64 = 25 * 1024 * 1024; // 25 MiB

/// Disk budget for the on-disk thumbnail cache. The LRU in `ghostftp.db` evicts down
/// to this after each write.
pub const CACHE_BUDGET_BYTES: u64 = 256 * 1024 * 1024; // 256 MiB

/// Concurrent preview fetches allowed *per connection*. The frontend viewport
/// gate already bounds requests to the rows on screen; this second cap keeps a
/// screenful scrolling into view all at once from swamping the session pool.
const PER_SESSION_CONCURRENCY: usize = 4;

/// Longest edge (in px) of the cached thumbnail. The grid renders at 56 CSS px;
/// caching a little larger keeps a Retina grid crisp and lets list/detail rows
/// reuse the same artifact without a re-fetch.
const THUMB_MAX_EDGE: u32 = 160;

/// Owns the thumbnail cache dir and the per-connection fetch gates. One instance
/// lives in `AppState`.
pub struct PreviewManager {
    cache_dir: PathBuf,
    /// One semaphore per session id, created on first use — the per-connection
    /// concurrency cap.
    gates: Mutex<HashMap<String, Arc<Semaphore>>>,
}

impl PreviewManager {
    /// Create the manager, ensuring the cache dir exists.
    pub fn new(cache_dir: PathBuf) -> Self {
        std::fs::create_dir_all(&cache_dir).ok();
        Self {
            cache_dir,
            gates: Mutex::new(HashMap::new()),
        }
    }

    /// The per-connection fetch gate, lazily created.
    async fn gate(&self, session_id: &str) -> Arc<Semaphore> {
        let mut gates = self.gates.lock().await;
        gates
            .entry(session_id.to_string())
            .or_insert_with(|| Arc::new(Semaphore::new(PER_SESSION_CONCURRENCY)))
            .clone()
    }

    fn cache_file(&self, key: &str) -> PathBuf {
        self.cache_dir.join(format!("{key}.png"))
    }

    /// Produce a small PNG thumbnail for a remote (or local) image, base64-encoded
    /// for the IPC hop. `session == None` is the local filesystem. `signal` is the
    /// backend's opaque change token (an object-store ETag, or `"size:mtime"`) —
    /// it's folded into the cache key so an edited file re-generates rather than
    /// serving a stale thumbnail. Any error → the caller shows the type icon.
    pub async fn thumbnail(
        &self,
        db: &crate::db::Db,
        session: Option<Arc<Session>>,
        session_id: &str,
        path: &str,
        file_size: u64,
        signal: &str,
    ) -> Result<String> {
        if file_size > MAX_PREVIEW_FILE_BYTES {
            bail!("file too large for preview ({file_size} bytes)");
        }

        let key = cache_key(session_id, path, THUMB_MAX_EDGE, signal);
        let file = self.cache_file(&key);

        // Cache hit: the index knows the key *and* the file is still on disk.
        if db.thumb_touch(&key).unwrap_or(false) {
            if let Ok(bytes) = tokio::fs::read(&file).await {
                return Ok(b64(&bytes));
            }
            // Index/file drift (cache dir cleared out-of-band) — fall through and
            // regenerate; thumb_record below re-asserts the row.
        }

        // Bounded fetch under the per-connection concurrency cap. The permit is
        // held only across the network read, then released before the CPU-bound
        // decode so a slow decode never blocks another row's fetch.
        let raw = {
            let gate = self.gate(session_id).await;
            let _permit = gate.acquire_owned().await.context("preview gate closed")?;
            read_head(session.as_deref(), path, MAX_PREVIEW_FILE_BYTES).await?
        };

        let png = tokio::task::spawn_blocking(move || downscale_png(&raw, THUMB_MAX_EDGE))
            .await
            .context("thumbnail decode task")??;

        // Cache: write atomically, index, then evict LRU down to budget.
        self.write_atomic(&file, &png).await?;
        let evicted = db
            .thumb_record(&key, session_id, png.len() as u64, CACHE_BUDGET_BYTES)
            .unwrap_or_default();
        for k in evicted {
            let _ = tokio::fs::remove_file(self.cache_file(&k)).await;
        }

        Ok(b64(&png))
    }

    /// Drop every cached thumbnail for a session (its connection was removed).
    #[allow(dead_code)]
    pub async fn clear_session(&self, db: &crate::db::Db, session_id: &str) {
        for k in db.thumb_clear_session(session_id).unwrap_or_default() {
            let _ = tokio::fs::remove_file(self.cache_file(&k)).await;
        }
        self.gates.lock().await.remove(session_id);
    }

    /// Write `bytes` to `dest` via a uniquely-named temp file + rename, so two
    /// concurrent fetches of the same key can't half-write each other's file.
    async fn write_atomic(&self, dest: &Path, bytes: &[u8]) -> Result<()> {
        let tmp = self.cache_dir.join(format!("{}.tmp", uuid::Uuid::new_v4()));
        tokio::fs::write(&tmp, bytes)
            .await
            .with_context(|| format!("write thumb temp {}", tmp.display()))?;
        match tokio::fs::rename(&tmp, dest).await {
            Ok(()) => Ok(()),
            Err(e) => {
                let _ = tokio::fs::remove_file(&tmp).await;
                Err(e).with_context(|| format!("commit thumb {}", dest.display()))
            }
        }
    }
}

/// The cache filename stem: a hex SHA-256 over the invalidation-relevant inputs.
/// Hex (not base64) so it's always filesystem-safe.
fn cache_key(session_id: &str, path: &str, dim: u32, signal: &str) -> String {
    use sha2::{Digest, Sha256};
    let mut h = Sha256::new();
    h.update(session_id.as_bytes());
    h.update([0]);
    h.update(path.as_bytes());
    h.update([0]);
    h.update(dim.to_le_bytes());
    h.update([0]);
    h.update(signal.as_bytes());
    let digest = h.finalize();
    let mut s = String::with_capacity(digest.len() * 2);
    for b in digest {
        s.push_str(&format!("{b:02x}"));
    }
    s
}

fn b64(bytes: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(bytes)
}

/// Decode an image from `bytes` and downscale it to fit a `max_edge`-px square,
/// re-encoding as PNG. Aspect ratio is preserved; images already within the box
/// are re-encoded unchanged. Returns an error for undecodable/unsupported bytes,
/// which the caller treats as "no preview".
fn downscale_png(bytes: &[u8], max_edge: u32) -> Result<Vec<u8>> {
    let img = image::load_from_memory(bytes).context("decode image")?;
    let scaled = if img.width().max(img.height()) > max_edge {
        // `thumbnail` is a fast, aspect-preserving box filter — ideal here.
        img.thumbnail(max_edge, max_edge)
    } else {
        img
    };
    let mut out = std::io::Cursor::new(Vec::new());
    scaled
        .write_to(&mut out, image::ImageFormat::Png)
        .context("encode png")?;
    Ok(out.into_inner())
}

/// Read up to `max` bytes from the head of a file, using each backend's native
/// API. This is the bounded-read primitive the whole plan hinges on — it mirrors
/// `transfer::remote_size`'s per-`Session` dispatch.
///
/// Also the engine behind the Agent Bridge's `read_file` on backends with no
/// in-place read op (`bridge::op_read_file_fetch`), which is why the match is
/// exhaustive over `Session`: a new backend must say how to read its bytes, not
/// silently fall into a "not supported" arm.
///
/// `session == None` is the local filesystem. Because the caller has already
/// size-gated to `MAX_PREVIEW_FILE_BYTES`, a `max` of that value reads the whole
/// (small enough) file — enough to decode — while still capping the transfer.
pub(crate) async fn read_head(session: Option<&Session>, path: &str, max: u64) -> Result<Vec<u8>> {
    match session {
        None => read_local(path, max).await,
        Some(Session::Ssh(ssh)) => {
            use tokio::io::AsyncReadExt;
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            let remote = sftp
                .open(path)
                .await
                .with_context(|| format!("open {path} over SFTP"))?;
            let mut buf = Vec::new();
            remote.take(max).read_to_end(&mut buf).await?;
            Ok(buf)
        }
        Some(Session::Object(obj)) => {
            let key = path.trim_start_matches('/');
            let p = object_store::path::Path::from(key);
            let get = obj
                .store
                .get(&p)
                .await
                .with_context(|| format!("get {key}"))?;
            collect_capped(
                get.into_stream().map(|r| r.map_err(anyhow::Error::from)),
                max,
            )
            .await
        }
        Some(Session::Ftp(ftp)) => {
            let path = path.to_string();
            let cap = max as usize;
            ftp.with_stream(move |stream| {
                let mut sink = CapSink {
                    buf: Vec::new(),
                    max: cap,
                };
                stream.retr_to_writer(&path, &mut sink)?;
                Ok(sink.buf)
            })
            .await
        }
        Some(Session::Webdav(dav)) => {
            let url = dav.url_for(path, false);
            let resp = dav
                .request(reqwest::Method::GET, url)
                .header(reqwest::header::RANGE, range_header(max))
                .send()
                .await
                .with_context(|| format!("GET {path} over WebDAV"))?
                .error_for_status()
                .with_context(|| format!("GET {path} over WebDAV"))?;
            collect_capped(
                resp.bytes_stream().map(|r| r.map_err(anyhow::Error::from)),
                max,
            )
            .await
        }
        Some(Session::Http(http)) => {
            let url = http.url_for(path, false);
            let resp = http
                .request(reqwest::Method::GET, url)
                .header(reqwest::header::RANGE, range_header(max))
                .send()
                .await
                .with_context(|| format!("GET {path}"))?
                .error_for_status()
                .with_context(|| format!("GET {path}"))?;
            collect_capped(
                resp.bytes_stream().map(|r| r.map_err(anyhow::Error::from)),
                max,
            )
            .await
        }
        Some(Session::Agent(agent)) => read_agent(agent, path, max).await,
        Some(Session::Shopify(sh)) => {
            // No ranged read in the Assets API — the file comes back whole.
            let data = crate::remotefs::shopify::read_asset(sh, path).await?;
            Ok(data.into_iter().take(max as usize).collect())
        }
        Some(Session::HubSpot(hs)) => {
            // No ranged read in the Source Code API — the file comes back whole.
            let data = crate::remotefs::hubspot::read_file(hs, path).await?;
            Ok(data.into_iter().take(max as usize).collect())
        }
        Some(Session::Dynamics(dynm)) => {
            // No ranged read in the Web API — the file comes back whole.
            let data = crate::remotefs::dynamics::read_file(dynm, path).await?;
            Ok(data.into_iter().take(max as usize).collect())
        }
        // The consumer-cloud four stream the whole body (their download
        // endpoints ignore Range), so `collect_capped` stops reading and drops
        // the rest once it has `max` bytes.
        Some(Session::Dropbox(dbx)) => {
            let arg = serde_json::json!({
                "path": crate::remotefs::dropbox::dropbox_api_path(path)
            })
            .to_string();
            let resp = dbx.content_get("/2/files/download", &arg).await?;
            collect_capped(
                resp.bytes_stream().map(|r| r.map_err(anyhow::Error::from)),
                max,
            )
            .await
        }
        Some(Session::OneDrive(od)) => {
            let resp = od
                .get_stream(&crate::remotefs::onedrive::content_ref(path))
                .await?;
            collect_capped(
                resp.bytes_stream().map(|r| r.map_err(anyhow::Error::from)),
                max,
            )
            .await
        }
        Some(Session::GDrive(gd)) => {
            let (file_id, _) = gd
                .resolve_item(path)
                .await?
                .ok_or_else(|| anyhow::anyhow!("{path}: not found"))?;
            let resp = gd
                .get_stream(&format!("/files/{file_id}?alt=media"))
                .await?;
            collect_capped(
                resp.bytes_stream().map(|r| r.map_err(anyhow::Error::from)),
                max,
            )
            .await
        }
        Some(Session::Box(bx)) => {
            let (file_id, _) = bx
                .resolve_item(path)
                .await?
                .ok_or_else(|| anyhow::anyhow!("{path}: not found"))?;
            let resp = bx.get_stream(&format!("/files/{file_id}/content")).await?;
            collect_capped(
                resp.bytes_stream().map(|r| r.map_err(anyhow::Error::from)),
                max,
            )
            .await
        }
    }
}

async fn read_local(path: &str, max: u64) -> Result<Vec<u8>> {
    use tokio::io::AsyncReadExt;
    let file = tokio::fs::File::open(path)
        .await
        .with_context(|| format!("open {path}"))?;
    let mut buf = Vec::new();
    file.take(max).read_to_end(&mut buf).await?;
    Ok(buf)
}

/// Read the head of a file from a Ghost FTP Agent daemon over the `ReadChunk` op,
/// stopping once we have `max` bytes or hit EOF.
async fn read_agent(
    session: &Arc<crate::session::AgentSession>,
    path: &str,
    max: u64,
) -> Result<Vec<u8>> {
    use ghostftp_agent_proto::msg::{Request, Response};
    let mut buf = Vec::new();
    let mut offset: u64 = 0;
    loop {
        let resp = session
            .request(Request::ReadChunk {
                path: path.to_string(),
                offset,
                len: 0, // server default chunk size
            })
            .await?;
        let (data_b64, eof) = match resp {
            Response::Chunk { data, eof } => (data, eof),
            Response::Error { message, .. } => bail!("read {path}: {message}"),
            other => bail!("read {path}: unexpected {other:?}"),
        };
        let bytes = base64::engine::general_purpose::STANDARD
            .decode(&data_b64)
            .context("decode chunk")?;
        if !bytes.is_empty() {
            offset += bytes.len() as u64;
            buf.extend_from_slice(&bytes);
        }
        if eof || buf.len() as u64 >= max {
            break;
        }
    }
    buf.truncate(max as usize);
    Ok(buf)
}

/// Drain a byte stream into a buffer, stopping once `max` bytes are collected
/// (the server may honor our Range header, or ignore it and stream the whole
/// body — either way we cap what we keep and drop the rest).
async fn collect_capped<S>(mut stream: S, max: u64) -> Result<Vec<u8>>
where
    S: futures::Stream<Item = Result<bytes::Bytes>> + Unpin,
{
    let mut buf = Vec::new();
    while let Some(chunk) = stream.next().await {
        buf.extend_from_slice(&chunk?);
        if buf.len() as u64 >= max {
            buf.truncate(max as usize);
            break;
        }
    }
    Ok(buf)
}

/// A byte range request for the first `max` bytes.
fn range_header(max: u64) -> String {
    format!("bytes=0-{}", max.saturating_sub(1))
}

/// A `std::io::Write` sink that keeps at most `max` bytes (for FTP's blocking
/// `retr_to_writer`), swallowing the rest so the transfer still completes.
struct CapSink {
    buf: Vec<u8>,
    max: usize,
}

impl std::io::Write for CapSink {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        if self.buf.len() < self.max {
            let room = self.max - self.buf.len();
            self.buf.extend_from_slice(&data[..room.min(data.len())]);
        }
        Ok(data.len())
    }
    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

// ---------- Command ----------

/// Fetch (or cache-serve) a small PNG thumbnail for a remote/local image,
/// returned base64-encoded. The frontend adapter gates this on the
/// `remoteImagePreviews` setting + a viewport observer + a concurrency limiter;
/// the backend adds a disk cache, a per-connection concurrency cap, and a hard
/// size/byte bound. Errors surface as a rejected promise → the pane shows the
/// type icon.
#[tauri::command]
pub async fn preview_thumbnail(
    session_id: String,
    path: String,
    file_size: u64,
    signal: String,
    state: tauri::State<'_, crate::AppState>,
) -> Result<String, String> {
    let session = if session_id == crate::commands::LOCAL_SESSION {
        None
    } else {
        Some(
            state
                .sessions
                .get(&session_id)
                .await
                .ok_or_else(|| format!("session {session_id} not found"))?,
        )
    };
    state
        .preview
        .thumbnail(&state.db, session, &session_id, &path, file_size, &signal)
        .await
        .map_err(|e| e.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A 2×3 red PNG, encoded once so the decode/downscale path has real bytes.
    fn tiny_png() -> Vec<u8> {
        use image::{ImageBuffer, Rgba};
        let img: ImageBuffer<Rgba<u8>, Vec<u8>> =
            ImageBuffer::from_pixel(2, 3, Rgba([255, 0, 0, 255]));
        let mut out = std::io::Cursor::new(Vec::new());
        image::DynamicImage::ImageRgba8(img)
            .write_to(&mut out, image::ImageFormat::Png)
            .unwrap();
        out.into_inner()
    }

    #[test]
    fn downscale_produces_valid_bounded_png() {
        // A big source downsizes to fit the box; a small one passes through.
        use image::{ImageBuffer, Rgba};
        let big: ImageBuffer<Rgba<u8>, Vec<u8>> =
            ImageBuffer::from_pixel(1000, 400, Rgba([0, 128, 255, 255]));
        let mut buf = std::io::Cursor::new(Vec::new());
        image::DynamicImage::ImageRgba8(big)
            .write_to(&mut buf, image::ImageFormat::Png)
            .unwrap();

        let out = downscale_png(&buf.into_inner(), 160).unwrap();
        let decoded = image::load_from_memory(&out).unwrap();
        assert!(decoded.width().max(decoded.height()) <= 160);
        // Aspect ratio (2.5:1) is preserved within rounding.
        assert_eq!(decoded.width(), 160);
        assert_eq!(decoded.height(), 64);

        // A smaller-than-box image is re-encoded unchanged in dimensions.
        let small = downscale_png(&tiny_png(), 160).unwrap();
        let d = image::load_from_memory(&small).unwrap();
        assert_eq!((d.width(), d.height()), (2, 3));
    }

    #[test]
    fn downscale_rejects_non_image_bytes() {
        assert!(downscale_png(b"not an image at all", 160).is_err());
    }

    #[test]
    fn cache_key_changes_with_signal_and_path() {
        let a = cache_key("s1", "/a.png", 160, "etag-1");
        let b = cache_key("s1", "/a.png", 160, "etag-2");
        let c = cache_key("s1", "/b.png", 160, "etag-1");
        assert_ne!(a, b, "a changed etag must re-key");
        assert_ne!(a, c, "a different path must re-key");
        assert_eq!(
            a,
            cache_key("s1", "/a.png", 160, "etag-1"),
            "stable inputs → stable key"
        );
        assert_eq!(a.len(), 64, "hex sha-256");
    }

    #[tokio::test]
    async fn local_thumbnail_round_trips_through_cache() {
        let dir = std::env::temp_dir().join(format!("ghostftp_prev_{}", std::process::id()));
        let cache = dir.join("cache");
        let mgr = PreviewManager::new(cache.clone());
        let db = crate::db::Db::open_in_memory().unwrap();

        // Write a real PNG to a local file.
        let img = dir.join("pic.png");
        std::fs::create_dir_all(&dir).unwrap();
        std::fs::write(&img, tiny_png()).unwrap();
        let path = img.to_string_lossy().to_string();

        let b64 = mgr
            .thumbnail(&db, None, "local", &path, tiny_png().len() as u64, "sig-1")
            .await
            .unwrap();
        let bytes = base64::engine::general_purpose::STANDARD
            .decode(&b64)
            .unwrap();
        assert!(
            image::load_from_memory(&bytes).is_ok(),
            "returns a decodable PNG"
        );

        // Second call is a cache hit (index touched, file present) and matches.
        let key = cache_key("local", &path, THUMB_MAX_EDGE, "sig-1");
        assert!(db.thumb_touch(&key).unwrap());
        let again = mgr
            .thumbnail(&db, None, "local", &path, tiny_png().len() as u64, "sig-1")
            .await
            .unwrap();
        assert_eq!(again, b64, "cache hit returns identical bytes");

        // Oversized files are refused before any read.
        assert!(mgr
            .thumbnail(
                &db,
                None,
                "local",
                &path,
                MAX_PREVIEW_FILE_BYTES + 1,
                "sig-1"
            )
            .await
            .is_err());

        let _ = std::fs::remove_dir_all(&dir);
    }
}
