use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::gdrive::normalize;
use crate::session::shopify::{AssetMeta, ShopifySession, ThemeInfo};
use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use std::sync::Arc;

/// RemoteFs over the Shopify Admin theme Assets API. A theme *is* a remote
/// filesystem: the store root lists themes as virtual directories (`{name}
/// [main]` for the live one), and inside a theme the flat asset list is the
/// tree — directories are inferred from key prefixes (`layout/`,
/// `templates/`, …). No chmod/symlinks; rename is PUT-new + DELETE-old
/// (Shopify has no atomic rename). Files carry `size` + `updated_at`, so the
/// change signal is `MtimeSize` (no etag exists).
pub struct ShopifyFs {
    session: Arc<ShopifySession>,
}

impl ShopifyFs {
    pub fn new(session: Arc<ShopifySession>) -> Self {
        Self { session }
    }
}

/// Hidden placeholder that materializes "empty directories" — Shopify has
/// none; a key prefix exists only while a file sits under it.
pub const KEEP_NAME: &str = ".ghostftp-keep";

/// Content written for the [`KEEP_NAME`] placeholder asset.
const KEEP_CONTENT: &[u8] = b"# ghostftp placeholder - keeps this directory on the store\n";

/// Theme assets that transfer as JSON `value` strings (Liquid/JSON/CSS/JS/SVG
/// & friends); everything else goes as base64 `attachment`.
pub fn is_text_key(key: &str) -> bool {
    let name = key.rsplit('/').next().unwrap_or(key);
    if name == KEEP_NAME {
        return true;
    }
    let ext = name.rsplit('.').next().unwrap_or("").to_ascii_lowercase();
    matches!(
        ext.as_str(),
        "liquid" | "json" | "css" | "js" | "svg" | "txt" | "md" | "html" | "xml" | "csv"
    )
}

/// Split a Ghost FTP path into (theme display name, asset key within the theme).
/// `/Dawn [main]/layout/theme.liquid` → `("Dawn [main]", "layout/theme.liquid")`.
pub fn split_path(ghostftp_path: &str) -> (String, String) {
    let norm = normalize(ghostftp_path);
    let t = norm.trim_start_matches('/');
    match t.split_once('/') {
        Some((theme, rest)) => (theme.to_string(), rest.trim_matches('/').to_string()),
        None => (t.to_string(), String::new()),
    }
}

/// Resolve a Ghost FTP path to (theme, asset key). The key is empty for a theme
/// root. Errors on an unknown theme name.
pub fn resolve(session: &ShopifySession, ghostftp_path: &str) -> Result<(ThemeInfo, String)> {
    let (display, key) = split_path(ghostftp_path);
    let theme = session
        .find_theme(&display)
        .ok_or_else(|| anyhow!("{ghostftp_path}: no such theme"))?;
    Ok((theme, key))
}

/// Read one asset's bytes (text `value` or base64 `attachment`), for the
/// transfer + editor arms.
pub async fn read_asset(session: &ShopifySession, ghostftp_path: &str) -> Result<Vec<u8>> {
    let (theme, key) = resolve(session, ghostftp_path)?;
    if key.is_empty() {
        return Err(anyhow!("{ghostftp_path} is a theme, not a file"));
    }
    session.asset_get(theme.id, &key).await
}

/// Write one asset's bytes, for the transfer + editor arms.
pub async fn write_asset(session: &ShopifySession, ghostftp_path: &str, data: &[u8]) -> Result<()> {
    let (theme, key) = resolve(session, ghostftp_path)?;
    if key.is_empty() {
        return Err(anyhow!("cannot write {ghostftp_path}: it is a theme"));
    }
    session.asset_put(theme.id, &key, data).await
}

/// Cached asset size for transfer planning (0 when unknown).
pub async fn asset_size(session: &ShopifySession, ghostftp_path: &str) -> u64 {
    match resolve(session, ghostftp_path) {
        Ok((theme, key)) if !key.is_empty() => session
            .assets(theme.id)
            .await
            .map(|list| {
                list.iter()
                    .find(|a| a.key == key)
                    .map(|a| a.size)
                    .unwrap_or(0)
            })
            .unwrap_or(0),
        _ => 0,
    }
}

/// Whether a path exists — as an asset key, or as a key prefix (directory).
pub async fn asset_exists(session: &ShopifySession, ghostftp_path: &str) -> bool {
    match resolve(session, ghostftp_path) {
        Ok((_, key)) if key.is_empty() => true, // a known theme
        Ok((theme, key)) => {
            let prefix = format!("{key}/");
            session
                .assets(theme.id)
                .await
                .map(|list| {
                    list.iter()
                        .any(|a| a.key == key || a.key.starts_with(&prefix))
                })
                .unwrap_or(false)
        }
        Err(_) => false,
    }
}

#[async_trait]
impl RemoteFs for ShopifyFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let norm = normalize(path);
        if norm == "/" {
            // The store root: one virtual directory per theme.
            return Ok(self
                .session
                .themes()
                .into_iter()
                .map(|t| DirEntry {
                    name: t.display_name(),
                    path: format!("/{}", t.display_name()),
                    kind: FileKind::Directory,
                    size: 0,
                    modified: None,
                    mode: None,
                    etag: None,
                })
                .collect());
        }
        let (theme, key) = resolve(&self.session, &norm)?;
        let prefix = if key.is_empty() {
            String::new()
        } else {
            format!("{key}/")
        };
        let assets = self.session.assets(theme.id).await?;
        if !key.is_empty() && assets.iter().any(|a| a.key == key) {
            return Err(anyhow!("{path} is a file, not a directory"));
        }
        Ok(children(norm.trim_end_matches('/'), &prefix, &assets))
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let (theme, from_key) = resolve(&self.session, from)?;
        let (to_theme, to_key) = resolve(&self.session, to)?;
        if theme.id != to_theme.id {
            return Err(anyhow!(
                "Shopify assets can't move across themes — copy + delete instead"
            ));
        }
        if from_key.is_empty() || to_key.is_empty() {
            return Err(anyhow!("themes can't be renamed from Ghost FTP"));
        }
        // No atomic rename in the Assets API: PUT the new key, DELETE the old
        // one (best-effort — a crash between the two leaves a copy).
        let assets = self.session.assets(theme.id).await?;
        if assets.iter().any(|a| a.key == from_key) {
            let data = self.session.asset_get(theme.id, &from_key).await?;
            self.session.asset_put(theme.id, &to_key, &data).await?;
            self.session
                .asset_delete(theme.id, &from_key)
                .await
                .with_context(|| {
                    format!("rename {from}: new key written, old key delete failed")
                })?;
            return Ok(());
        }
        // Directory rename: every key under the prefix.
        let prefix = format!("{from_key}/");
        let keys: Vec<String> = assets
            .iter()
            .filter(|a| a.key.starts_with(&prefix))
            .map(|a| a.key.clone())
            .collect();
        if keys.is_empty() {
            return Err(anyhow!("{from}: not found"));
        }
        for k in keys {
            let data = self.session.asset_get(theme.id, &k).await?;
            let new_key = format!("{to_key}/{}", &k[prefix.len()..]);
            self.session.asset_put(theme.id, &new_key, &data).await?;
            self.session.asset_delete(theme.id, &k).await?;
        }
        Ok(())
    }

    async fn delete(&self, path: &str, recursive: bool) -> Result<()> {
        let (theme, key) = resolve(&self.session, path)?;
        if key.is_empty() {
            return Err(anyhow!("themes can't be deleted from Ghost FTP"));
        }
        let assets = self.session.assets(theme.id).await?;
        if assets.iter().any(|a| a.key == key) {
            return self.session.asset_delete(theme.id, &key).await;
        }
        // Directory delete: iterate the keys under the prefix (the session's
        // throttle keeps this inside the rate limit).
        let prefix = format!("{key}/");
        let keys: Vec<String> = assets
            .iter()
            .filter(|a| a.key.starts_with(&prefix))
            .map(|a| a.key.clone())
            .collect();
        if keys.is_empty() {
            return Err(anyhow!("{path}: not found"));
        }
        if !recursive {
            return Err(anyhow!("{path} is a directory — delete it recursively"));
        }
        for k in keys {
            self.session.asset_delete(theme.id, &k).await?;
        }
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        let (theme, key) = resolve(&self.session, path)?;
        if key.is_empty() {
            return Err(anyhow!("themes are created in the Shopify admin, not here"));
        }
        // Materialize the prefix with a hidden placeholder asset.
        self.session
            .asset_put(theme.id, &format!("{key}/{KEEP_NAME}"), KEEP_CONTENT)
            .await
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(anyhow!("Shopify has no POSIX permissions"))
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: false,
            can_symlink: false,
            can_rename: true,
            has_directories: true,
            has_shell: false,
            change_signal: ChangeSignal::MtimeSize,
        }
    }
}

/// Direct children of `prefix` ("" for a theme root) derived from the flat
/// asset list: synthesized directories (deduped) plus files with metadata.
/// The [`KEEP_NAME`] placeholder is hidden from listings.
fn children(base: &str, prefix: &str, assets: &[AssetMeta]) -> Vec<DirEntry> {
    let mut out: Vec<DirEntry> = Vec::new();
    for a in assets {
        let Some(rest) = a.key.strip_prefix(prefix) else {
            continue;
        };
        if rest.is_empty() {
            continue;
        }
        match rest.split_once('/') {
            Some((dir, _)) => {
                if !out
                    .iter()
                    .any(|e| e.kind == FileKind::Directory && e.name == dir)
                {
                    out.push(DirEntry {
                        name: dir.to_string(),
                        path: format!("{base}/{dir}"),
                        kind: FileKind::Directory,
                        size: 0,
                        modified: None,
                        mode: None,
                        etag: None,
                    });
                }
            }
            None => {
                if rest == KEEP_NAME {
                    continue;
                }
                out.push(file_entry(a, base));
            }
        }
    }
    out
}

/// A `DirEntry` for one asset listed directly under `base`.
fn file_entry(a: &AssetMeta, base: &str) -> DirEntry {
    let name = a.key.rsplit('/').next().unwrap_or(&a.key).to_string();
    DirEntry {
        path: format!("{base}/{name}"),
        name,
        kind: FileKind::File,
        size: a.size,
        modified: a.updated_at,
        mode: None,
        etag: None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::session::shopify::{asset_from_json, parse_time};

    fn meta(key: &str, size: u64) -> AssetMeta {
        AssetMeta {
            key: key.to_string(),
            size,
            updated_at: Some(1_721_035_800),
            content_type: None,
        }
    }

    #[test]
    fn path_mapping() {
        assert_eq!(split_path("/"), (String::new(), String::new()));
        assert_eq!(
            split_path("/Dawn [main]"),
            ("Dawn [main]".to_string(), String::new())
        );
        assert_eq!(
            split_path("/Dawn/layout/theme.liquid"),
            ("Dawn".to_string(), "layout/theme.liquid".to_string())
        );
        assert_eq!(
            split_path("/Dawn/snippets/icons/"),
            ("Dawn".to_string(), "snippets/icons".to_string())
        );
    }

    #[test]
    fn theme_display_names() {
        let main = ThemeInfo {
            id: 1,
            name: "Dawn".into(),
            role: "main".into(),
        };
        let draft = ThemeInfo {
            id: 2,
            name: "Dawn copy".into(),
            role: "unpublished".into(),
        };
        assert_eq!(main.display_name(), "Dawn [main]");
        assert_eq!(draft.display_name(), "Dawn copy");
    }

    #[test]
    fn text_binary_classification() {
        assert!(is_text_key("layout/theme.liquid"));
        assert!(is_text_key("templates/index.json"));
        assert!(is_text_key("assets/base.css"));
        assert!(is_text_key("assets/app.js"));
        assert!(is_text_key("assets/icon.svg"));
        assert!(!is_text_key("assets/logo.png"));
        assert!(!is_text_key("assets/font.woff2"));
    }

    #[test]
    fn asset_json_to_dir_entry() {
        let a = serde_json::json!({
            "key": "layout/theme.liquid",
            "size": 1234,
            "updated_at": "2024-07-15T09:30:00-04:00",
            "content_type": "text/x-liquid",
        });
        let m = asset_from_json(&a).expect("parse");
        assert_eq!(m.key, "layout/theme.liquid");
        assert_eq!(m.size, 1234);
        // 09:30 at -04:00 is 13:30 UTC.
        assert_eq!(m.updated_at, Some(1_721_035_800 + 4 * 3600));
        let e = file_entry(&m, "/Dawn/layout");
        assert_eq!(e.name, "theme.liquid");
        assert_eq!(e.path, "/Dawn/layout/theme.liquid");
        assert_eq!(e.kind, FileKind::File);
        assert_eq!(e.size, 1234);
        assert_eq!(parse_time("2024-07-15T09:30:00Z"), Some(1_721_035_800));
    }

    #[test]
    fn prefix_tree_children() {
        let assets = vec![
            meta("layout/theme.liquid", 10),
            meta("layout/password.liquid", 20),
            meta("snippets/icon.liquid", 5),
            meta("snippets/nested/deep.liquid", 7),
            meta(&format!("templates/{KEEP_NAME}"), 1),
        ];
        let root = children("/Dawn", "", &assets);
        assert_eq!(root.len(), 3);
        assert!(root.iter().all(|e| e.kind == FileKind::Directory));
        // A placeholder alone still materializes its directory.
        assert!(root.iter().any(|e| e.name == "templates"));

        let snip = children("/Dawn/snippets", "snippets/", &assets);
        assert_eq!(snip.iter().filter(|e| e.kind == FileKind::File).count(), 1);
        assert!(snip
            .iter()
            .any(|e| e.name == "nested" && e.kind == FileKind::Directory));

        // The placeholder itself is hidden when listing its own directory.
        let tpl = children("/Dawn/templates", "templates/", &assets);
        assert!(tpl.is_empty());
    }

    /// End-to-end against the Shopify mock (`tests/shopify_mock.py`). Skipped
    /// unless GHOSTFTP_SHOPIFY_MOCK_URL is set; run with
    /// `cargo test -p ghostftp -- --ignored shopify_roundtrip` after starting it.
    /// Exercises connect (client-credentials exchange + static token), theme
    /// listing, asset list/upload/read, the 429 retry hook, rename, and delete.
    #[tokio::test]
    #[ignore = "requires the Shopify mock (GHOSTFTP_SHOPIFY_MOCK_URL)"]
    async fn live_shopify_roundtrip() {
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::shopify::{credential_key, shopify_connect};

        let Ok(mock) = std::env::var("GHOSTFTP_SHOPIFY_MOCK_URL") else {
            eprintln!("skip: GHOSTFTP_SHOPIFY_MOCK_URL unset");
            return;
        };
        std::env::set_var("GHOSTFTP_SHOPIFY_API_BASE", &mock);
        std::env::set_var(
            "GHOSTFTP_SHOPIFY_TOKEN_URL",
            format!("{mock}/admin/oauth/access_token"),
        );

        let mk_profile = |pid: &str| ConnectionProfile {
            icon: None,
            id: pid.into(),
            name: "shop".into(),
            protocol: "shopify".into(),
            host: "test-shop.myshopify.com".into(),
            port: 443,
            username: String::new(),
            auth: AuthMethod::Password {
                password: String::new(),
            },
            default_remote_path: None,
            description: None,
            color: None,
            auto_connect: None,
            bucket: None,
            region: None,
            endpoint: None,
            account: None,
            agent_key: None,
            group: None,
            favorite: None,
            bookmarked: None,
            tags: None,
            last_used: None,
            sort_order: None,
            jump_host: None,
            jump_port: None,
            jump_username: None,
        };

        // Client-credentials flavor: token exchange happens on first request.
        let pid = "shopify-mock-test";
        crate::credentials::set_secret(&credential_key(pid), "test-client:test-secret")
            .expect("seed secret");
        let session = Arc::new(shopify_connect(&mk_profile(pid)).await.expect("connect"));
        assert_eq!(
            session.account_label().await.expect("label"),
            "test-shop.myshopify.com"
        );
        assert_eq!(session.token().await.expect("token"), "CC-TOKEN");

        let fs = ShopifyFs::new(session.clone());
        let themes = fs.list_dir("/").await.expect("themes");
        assert!(themes.iter().any(|e| e.name == "Dawn [main]"));
        assert!(themes.iter().any(|e| e.name == "Draft"));

        let theme_root = fs.list_dir("/Draft").await.expect("list theme");
        assert!(theme_root
            .iter()
            .any(|e| e.name == "layout" && e.kind == FileKind::Directory));

        // Upload through the 429-injection hook (first PUT 429s, retry wins).
        write_asset(&session, "/Draft/assets/rl429.txt", b"hi shop")
            .await
            .expect("put");
        let data = read_asset(&session, "/Draft/assets/rl429.txt")
            .await
            .expect("get");
        assert_eq!(data, b"hi shop");
        assert_eq!(asset_size(&session, "/Draft/assets/rl429.txt").await, 7);

        // Rename = PUT new + DELETE old.
        fs.rename("/Draft/assets/rl429.txt", "/Draft/assets/moved.txt")
            .await
            .expect("rename");
        assert!(asset_exists(&session, "/Draft/assets/moved.txt").await);
        assert!(!asset_exists(&session, "/Draft/assets/rl429.txt").await);

        fs.delete("/Draft/assets/moved.txt", false)
            .await
            .expect("delete");
        assert!(!asset_exists(&session, "/Draft/assets/moved.txt").await);

        // mkdir materializes a hidden placeholder; recursive delete walks it.
        fs.create_dir("/Draft/snippets/ghostftp-dir")
            .await
            .expect("mkdir");
        let snip = fs.list_dir("/Draft/snippets").await.expect("list snippets");
        assert!(snip.iter().any(|e| e.name == "ghostftp-dir"));
        let empty = fs
            .list_dir("/Draft/snippets/ghostftp-dir")
            .await
            .expect("list placeholder dir");
        assert!(empty.is_empty(), "placeholder hidden from listing");
        fs.delete("/Draft/snippets/ghostftp-dir", true)
            .await
            .expect("rmdir");
        assert!(!asset_exists(&session, "/Draft/snippets/ghostftp-dir").await);

        // Static-token flavor: the secret passes through untouched.
        let pid2 = "shopify-mock-static";
        crate::credentials::set_secret(&credential_key(pid2), "shpat_test").expect("seed static");
        let static_session = shopify_connect(&mk_profile(pid2))
            .await
            .expect("connect static");
        assert_eq!(
            static_session.token().await.expect("static token"),
            "shpat_test"
        );

        crate::credentials::delete_secret(&credential_key(pid));
        crate::credentials::delete_secret(&credential_key(pid2));
        eprintln!("live_shopify_roundtrip: OK");
    }
}
