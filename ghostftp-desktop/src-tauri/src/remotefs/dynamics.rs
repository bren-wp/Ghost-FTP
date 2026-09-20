use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::dynamics::{DynamicsSession, WebResource, MAX_WRITE_BYTES};
use crate::session::gdrive::normalize;
use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use std::sync::Arc;

/// RemoteFs over the Dataverse Web API's `webresourceset`: web resources are
/// literally files stored in a table — `name` is the path (`new_/js/form.js`,
/// publisher prefix + virtual folders inferred from prefixes, the Shopify
/// trick), `content` moves as base64, and every write needs a `PublishXml`
/// to go live (save = deployed, which the connect dialog says). The whole
/// tree is one paged query, cached ~30s by the session. Managed-solution
/// components are listed read-only (mode `0o444`) and refuse mutations
/// client-side; the OData error is the backstop. No chmod/symlinks; rename is
/// create-new + delete-old (best-effort — Dataverse has no atomic rename).
/// Entries carry `modifiedon`; size needs a content fetch, so listings
/// report the cached size (0 until read) — the change signal is `MtimeSize`
/// (no etag exists; `versionnumber` is org-wide).
pub struct DynamicsFs {
    session: Arc<DynamicsSession>,
}

impl DynamicsFs {
    pub fn new(session: Arc<DynamicsSession>) -> Self {
        Self { session }
    }
}

/// Hidden placeholder that materializes "empty directories" — Dataverse has
/// none; a folder prefix exists only while a file sits under it. The `.js`
/// extension keeps it a valid webresourcetype (3).
pub const KEEP_NAME: &str = ".ghostftp-keep.js";

/// Content written for the [`KEEP_NAME`] placeholder web resource.
const KEEP_CONTENT: &[u8] = b"// ghostftp placeholder - keeps this directory in the environment\n";

/// The webresourcetype for a file extension, or `None` when Dataverse has no
/// type for it: 1 HTML, 2 CSS, 3 JS, 4 XML, 5 PNG, 6 JPG, 7 GIF, 9 XSL,
/// 10 ICO, 11 SVG, 12 RESX (8 XAP is legacy Silverlight — not creatable).
pub fn type_for_extension(name: &str) -> Option<i64> {
    let base = name.rsplit('/').next().unwrap_or(name);
    if base == KEEP_NAME {
        return Some(3);
    }
    let ext = base.rsplit('.').next().unwrap_or("").to_ascii_lowercase();
    match ext.as_str() {
        "html" | "htm" => Some(1),
        "css" => Some(2),
        "js" => Some(3),
        "xml" => Some(4),
        "png" => Some(5),
        "jpg" | "jpeg" => Some(6),
        "gif" => Some(7),
        "xsl" | "xslt" => Some(9),
        "ico" => Some(10),
        "svg" => Some(11),
        "resx" => Some(12),
        _ => None,
    }
}

/// Text vs binary by webresourcetype: 1–4, 9 and 12 are text (edit-in-place);
/// images/XAP/ICO are binary (transfer only).
pub fn is_text_type(res_type: i64) -> bool {
    matches!(res_type, 1..=4 | 9 | 12)
}

/// The user-actionable version of [`type_for_extension`], called before every
/// write — Dataverse 400s on unknown types anyway; we fail earlier and clearer.
fn check_type(name: &str) -> Result<i64> {
    type_for_extension(name).ok_or_else(|| {
        anyhow!(
            "Dynamics web resources only support these extensions: \
             html css js xml png jpg gif xsl ico svg resx — {name} has no webresourcetype"
        )
    })
}

/// The resource name for a Ghost FTP path: `/new_/js/form.js` → `new_/js/form.js`;
/// empty for the root.
pub fn name_of(ghostftp_path: &str) -> String {
    normalize(ghostftp_path).trim_matches('/').to_string()
}

/// Refuse to mutate a managed-solution component, phrased for the user.
fn ensure_writable(r: &WebResource, op: &str) -> Result<()> {
    if r.managed {
        return Err(anyhow!(
            "{} is a managed web resource — it can't be {op} in place. \
             Export the solution or work on an unmanaged copy.",
            r.name
        ));
    }
    Ok(())
}

/// Read one web resource's bytes, for the transfer + editor + preview arms.
pub async fn read_file(session: &DynamicsSession, ghostftp_path: &str) -> Result<Vec<u8>> {
    let name = name_of(ghostftp_path);
    if name.is_empty() {
        return Err(anyhow!(
            "{ghostftp_path} is the web-resource root, not a file"
        ));
    }
    let r = find(session, &name)
        .await?
        .ok_or_else(|| anyhow!("dynamics {name}: not found"))?;
    session.content_get(&r.id, &r.name).await
}

/// Write one web resource's bytes, for the transfer + editor arms: create or
/// update by a fresh name lookup, then publish — save = deployed.
pub async fn write_file(session: &DynamicsSession, ghostftp_path: &str, data: &[u8]) -> Result<()> {
    let name = name_of(ghostftp_path);
    if name.is_empty() {
        return Err(anyhow!(
            "cannot write {ghostftp_path}: it is the web-resource root"
        ));
    }
    if data.len() > MAX_WRITE_BYTES {
        return Err(anyhow!(
            "{name} is {} bytes — Dataverse web resources are capped at 5 MB",
            data.len()
        ));
    }
    let res_type = check_type(&name)?;
    match session.resource_lookup(&name).await? {
        Some(r) => {
            ensure_writable(&r, "edited")?;
            session.resource_update(&r.id, &r.name, data).await?;
            session.publish(&[r.id]).await?;
        }
        None => {
            let id = session.resource_create(&name, res_type, data).await?;
            session.publish(&[id]).await?;
        }
    }
    session.invalidate();
    Ok(())
}

/// Size for transfer planning: the cached decoded length when a recent
/// read/write set it, else a content fetch (there is no metadata size — 0
/// only when the fetch fails).
pub async fn file_size(session: &DynamicsSession, ghostftp_path: &str) -> u64 {
    let name = name_of(ghostftp_path);
    if name.is_empty() {
        return 0;
    }
    if let Some(size) = session.cached_size(&name) {
        return size;
    }
    match find(session, &name).await {
        Ok(Some(r)) => session
            .content_get(&r.id, &r.name)
            .await
            .map(|d| d.len() as u64)
            .unwrap_or(0),
        _ => 0,
    }
}

/// Whether a path exists — as a web resource, or as a name prefix (directory).
pub async fn file_exists(session: &DynamicsSession, ghostftp_path: &str) -> bool {
    let name = name_of(ghostftp_path);
    if name.is_empty() {
        return true; // the root
    }
    let prefix = format!("{name}/");
    session
        .webresources()
        .await
        .map(|list| {
            list.iter()
                .any(|r| r.name == name || r.name.starts_with(&prefix))
        })
        .unwrap_or(false)
}

/// Find one resource by name in the cached listing (None when absent).
async fn find(session: &DynamicsSession, name: &str) -> Result<Option<WebResource>> {
    Ok(session
        .webresources()
        .await?
        .into_iter()
        .find(|r| r.name == name))
}

#[async_trait]
impl RemoteFs for DynamicsFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let norm = normalize(path);
        let name = name_of(&norm);
        let resources = self.session.webresources().await?;
        if resources.iter().any(|r| r.name == name) {
            return Err(anyhow!("{path} is a file, not a directory"));
        }
        let prefix = if name.is_empty() {
            String::new()
        } else {
            format!("{name}/")
        };
        Ok(children(
            norm.trim_end_matches('/'),
            &prefix,
            &resources,
            &|n| self.session.cached_size(n),
        ))
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let from_name = name_of(from);
        let to_name = name_of(to);
        if from_name.is_empty() || to_name.is_empty() {
            return Err(anyhow!(
                "the web-resource root can't be renamed from Ghost FTP"
            ));
        }
        let resources = self.session.webresources().await?;
        if let Some(r) = resources.iter().find(|r| r.name == from_name) {
            // No atomic rename in the Web API: create the new row, DELETE the
            // old one (best-effort — a crash between the two leaves a copy).
            ensure_writable(r, "renamed")?;
            let res_type = check_type(&to_name)?;
            let data = self.session.content_get(&r.id, &r.name).await?;
            let id = self
                .session
                .resource_create(&to_name, res_type, &data)
                .await?;
            self.session.publish(&[id]).await?;
            self.session
                .resource_delete(&r.id, &r.name)
                .await
                .with_context(|| {
                    format!("rename {from}: new resource written, old resource delete failed")
                })?;
            self.session.invalidate();
            return Ok(());
        }
        // Directory rename: every resource under the prefix, one at a time
        // (the session's throttle keeps this inside the rate limit).
        let prefix = format!("{from_name}/");
        let under: Vec<WebResource> = resources
            .iter()
            .filter(|r| r.name.starts_with(&prefix))
            .cloned()
            .collect();
        if under.is_empty() {
            return Err(anyhow!("{from}: not found"));
        }
        for r in &under {
            ensure_writable(r, "renamed")?;
        }
        for r in under {
            let data = self.session.content_get(&r.id, &r.name).await?;
            let new_name = format!("{to_name}/{}", &r.name[prefix.len()..]);
            let id = self
                .session
                .resource_create(&new_name, r.res_type, &data)
                .await?;
            self.session.publish(&[id]).await?;
            self.session.resource_delete(&r.id, &r.name).await?;
        }
        self.session.invalidate();
        Ok(())
    }

    async fn delete(&self, path: &str, recursive: bool) -> Result<()> {
        let name = name_of(path);
        if name.is_empty() {
            return Err(anyhow!(
                "the web-resource root can't be deleted from Ghost FTP"
            ));
        }
        let resources = self.session.webresources().await?;
        if let Some(r) = resources.iter().find(|r| r.name == name) {
            ensure_writable(r, "deleted")?;
            self.session.resource_delete(&r.id, &r.name).await?;
            self.session.invalidate();
            return Ok(());
        }
        // Recursive delete iterates the cached names under the prefix.
        let prefix = format!("{name}/");
        let under: Vec<WebResource> = resources
            .iter()
            .filter(|r| r.name.starts_with(&prefix))
            .cloned()
            .collect();
        if under.is_empty() {
            return Err(anyhow!("{path}: not found"));
        }
        if !recursive {
            return Err(anyhow!("{path} is a directory — delete it recursively"));
        }
        for r in &under {
            ensure_writable(r, "deleted")?;
        }
        for r in under {
            self.session.resource_delete(&r.id, &r.name).await?;
        }
        self.session.invalidate();
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        let name = name_of(path);
        if name.is_empty() {
            return Err(anyhow!("the web-resource root already exists"));
        }
        // Materialize the prefix with a hidden placeholder web resource (type
        // 3 = JS, so it carries a valid webresourcetype), then publish it.
        let keep = format!("{name}/{KEEP_NAME}");
        let id = self.session.resource_create(&keep, 3, KEEP_CONTENT).await?;
        self.session.publish(&[id]).await?;
        self.session.invalidate();
        Ok(())
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(anyhow!("not supported"))
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

/// Direct children of `prefix` ("" for the root) derived from the flat
/// resource list: synthesized directories (deduped) plus files with metadata.
/// The [`KEEP_NAME`] placeholder is hidden from listings; managed resources
/// are flagged read-only via mode `0o444`.
fn children(
    base: &str,
    prefix: &str,
    resources: &[WebResource],
    size_of: &dyn Fn(&str) -> Option<u64>,
) -> Vec<DirEntry> {
    let mut out: Vec<DirEntry> = Vec::new();
    for r in resources {
        let Some(rest) = r.name.strip_prefix(prefix) else {
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
                let name = r.name.rsplit('/').next().unwrap_or(&r.name).to_string();
                out.push(DirEntry {
                    path: format!("{base}/{name}"),
                    name,
                    kind: FileKind::File,
                    // No metadata size exists — report the cached decoded
                    // length (0 until the file is read), honestly.
                    size: size_of(&r.name).unwrap_or(0),
                    modified: r.modified,
                    // The read-only flag the UI can render for managed rows.
                    mode: r.managed.then_some(0o444),
                    etag: None,
                });
            }
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::session::dynamics::{publish_xml, resource_from_json};

    fn res(name: &str, res_type: i64, managed: bool) -> WebResource {
        WebResource {
            id: format!("id-{name}"),
            name: name.to_string(),
            res_type,
            modified: Some(1_721_035_800),
            managed,
        }
    }

    #[test]
    fn path_mapping() {
        assert_eq!(name_of("/"), String::new());
        assert_eq!(name_of("/new_"), "new_");
        assert_eq!(name_of("/new_/js/form.js"), "new_/js/form.js");
        assert_eq!(name_of("/new_/js/"), "new_/js");
        assert_eq!(name_of("new_/css/main.css"), "new_/css/main.css");
    }

    #[test]
    fn type_extension_classification() {
        assert_eq!(type_for_extension("new_/page.html"), Some(1));
        assert_eq!(type_for_extension("new_/css/main.css"), Some(2));
        assert_eq!(type_for_extension("new_/js/form.js"), Some(3));
        assert_eq!(type_for_extension("new_/data/config.xml"), Some(4));
        assert_eq!(type_for_extension("new_/img/logo.PNG"), Some(5)); // case-insensitive
        assert_eq!(type_for_extension("new_/img/photo.jpeg"), Some(6));
        assert_eq!(type_for_extension("new_/img/anim.gif"), Some(7));
        assert_eq!(type_for_extension("new_/xsl/transform.xsl"), Some(9));
        assert_eq!(type_for_extension("new_/favicon.ico"), Some(10));
        assert_eq!(type_for_extension("new_/img/icon.svg"), Some(11));
        assert_eq!(type_for_extension("new_/res/strings.resx"), Some(12));
        assert_eq!(type_for_extension(&format!("dir/{KEEP_NAME}")), Some(3));
        assert_eq!(type_for_extension("new_/notes.txt"), None);
        assert_eq!(type_for_extension("new_/archive.zip"), None);
        assert_eq!(type_for_extension("Makefile"), None);

        // Text (edit-in-place) vs binary (transfer only).
        for t in [1, 2, 3, 4, 9, 12] {
            assert!(is_text_type(t), "type {t} is text");
        }
        for t in [5, 6, 7, 8, 10, 11] {
            assert!(!is_text_type(t), "type {t} is binary");
        }
    }

    #[test]
    fn base64_round_trip() {
        use base64::Engine as _;
        let data: Vec<u8> = (0u8..=255).collect();
        let b64 = base64::engine::general_purpose::STANDARD.encode(&data);
        let back = base64::engine::general_purpose::STANDARD
            .decode(&b64)
            .expect("decode");
        assert_eq!(back, data);
    }

    #[test]
    fn publish_xml_payload() {
        assert_eq!(
            publish_xml(&["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".to_string()]),
            "<importexportxml><webresources>\
             <webresource>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</webresource>\
             </webresources></importexportxml>"
        );
        let batch = publish_xml(&["id1".to_string(), "id2".to_string()]);
        assert!(batch.contains("<webresource>id1</webresource>"));
        assert!(batch.contains("<webresource>id2</webresource>"));
        assert!(batch.starts_with("<importexportxml>"));
        assert_eq!(
            publish_xml(&[]),
            "<importexportxml><webresources></webresources></importexportxml>"
        );
    }

    #[test]
    fn managed_refusal() {
        let managed = res("amp_/lib/managed.js", 3, true);
        let unmanaged = res("new_/js/form.js", 3, false);
        let err = ensure_writable(&managed, "edited").unwrap_err();
        assert!(err.to_string().contains("managed web resource"));
        assert!(ensure_writable(&unmanaged, "edited").is_ok());
        assert!(ensure_writable(&unmanaged, "deleted").is_ok());
    }

    #[test]
    fn resource_json_parse() {
        let v = serde_json::json!({
            "webresourceid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
            "name": "new_/js/form.js",
            "webresourcetype": 3,
            "modifiedon": "2024-07-15T09:30:00Z",
            "ismanaged": false
        });
        let r = resource_from_json(&v).expect("parse");
        assert_eq!(r.id, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee");
        assert_eq!(r.name, "new_/js/form.js");
        assert_eq!(r.res_type, 3);
        assert_eq!(r.modified, Some(1_721_035_800));
        assert!(!r.managed);
        // Missing fields are skipped, not fatal.
        assert!(resource_from_json(&serde_json::json!({"name": "x"})).is_none());
    }

    #[test]
    fn prefix_tree_children() {
        let resources = vec![
            res("new_/js/form.js", 3, false),
            res("new_/js/lib/util.js", 3, false),
            res("new_/css/main.css", 2, false),
            res("amp_/lib/managed.js", 3, true),
            res(&format!("new_/img/{KEEP_NAME}"), 3, false),
        ];
        let no_sizes = &|_: &str| None;
        let root = children("", "", &resources, no_sizes);
        assert_eq!(root.len(), 2);
        assert!(root.iter().all(|e| e.kind == FileKind::Directory));
        assert!(root.iter().any(|e| e.name == "new_"));
        assert!(root.iter().any(|e| e.name == "amp_"));

        let js = children("/new_", "new_/", &resources, no_sizes);
        assert_eq!(js.len(), 3);
        assert!(js
            .iter()
            .any(|e| e.name == "js" && e.kind == FileKind::Directory));
        assert!(js
            .iter()
            .any(|e| e.name == "css" && e.kind == FileKind::Directory));

        // Managed rows are listed with the read-only mode flag.
        let lib = children("/amp_", "amp_/", &resources, no_sizes);
        let managed = lib.iter().find(|e| e.name == "lib").expect("lib dir");
        assert_eq!(managed.kind, FileKind::Directory);
        let rows = children("/amp_/lib", "amp_/lib/", &resources, no_sizes);
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0].mode, Some(0o444));
        assert_eq!(rows[0].modified, Some(1_721_035_800));

        // The placeholder alone still materializes its directory, but is
        // hidden when listing that directory.
        assert!(js
            .iter()
            .any(|e| e.name == "img" && e.kind == FileKind::Directory));
        let img = children("/new_/img", "new_/img/", &resources, no_sizes);
        assert!(img.is_empty());

        // Cached sizes flow into entries.
        let with_sizes = &|n: &str| (n == "new_/js/form.js").then_some(15);
        let files = children("/new_/js", "new_/js/", &resources, with_sizes);
        let form = files.iter().find(|e| e.name == "form.js").expect("form.js");
        assert_eq!(form.size, 15);
        assert_eq!(form.path, "/new_/js/form.js");
        assert_eq!(form.mode, None);
    }

    /// End-to-end against the Dynamics mock (`tests/dynamics_mock.py`).
    /// Skipped unless GHOSTFTP_DYNAMICS_MOCK_URL is set; run with
    /// `cargo test -p ghostftp -- --ignored dynamics_roundtrip` after starting it.
    /// Exercises connect (client-credentials exchange + WhoAmI label), the
    /// paged tree listing, upload through the 429 retry hook, read-back,
    /// publish recording, rename (create+delete), recursive delete, the
    /// managed-write refusal, and the throttle pacing.
    #[tokio::test]
    #[ignore = "requires the Dynamics mock (GHOSTFTP_DYNAMICS_MOCK_URL)"]
    async fn live_dynamics_roundtrip() {
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::dynamics::{credential_key, dynamics_connect};

        let Ok(mock) = std::env::var("GHOSTFTP_DYNAMICS_MOCK_URL") else {
            eprintln!("skip: GHOSTFTP_DYNAMICS_MOCK_URL unset");
            return;
        };
        std::env::set_var(
            "GHOSTFTP_DYNAMICS_API_BASE",
            format!("{mock}/api/data/v9.2"),
        );
        std::env::set_var(
            "GHOSTFTP_DYNAMICS_TOKEN_URL",
            format!("{mock}/mock-tenant/oauth2/v2.0/token"),
        );

        let profile = ConnectionProfile {
            icon: None,
            id: "dynamics-mock-test".into(),
            name: "org".into(),
            protocol: "dynamics".into(),
            host: "mockorg.crm.dynamics.com".into(),
            port: 443,
            username: String::new(),
            auth: AuthMethod::Password {
                password: String::new(),
            },
            default_remote_path: None,
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

        // Client-credentials blob: tenant:client_id:client_secret.
        let pid = profile.id.clone();
        crate::credentials::set_secret(
            &credential_key(&pid),
            "mock-tenant:test-client-id:test-secret",
        )
        .expect("seed secret");
        let session = Arc::new(dynamics_connect(&profile).await.expect("connect"));
        assert_eq!(
            session.account_label().await.expect("label"),
            "mockorg.crm.dynamics.com · user 11111111-2222-3333-4444-555555555555"
        );

        // The mock records every PublishXml batch; the test reads them back
        // straight off the mock (unauthenticated test hook).
        let publishes = || async {
            let v: serde_json::Value = reqwest::get(format!("{mock}/__publishes"))
                .await
                .expect("publishes fetch")
                .json()
                .await
                .expect("publishes json");
            v.get("publishes")
                .and_then(|p| p.as_array())
                .cloned()
                .unwrap_or_default()
                .len()
        };
        let baseline = publishes().await;

        let fs = DynamicsFs::new(session.clone());

        // The paged tree (the mock pages at 2 rows, forcing nextLink loops).
        let root = fs.list_dir("/").await.expect("root");
        assert!(root
            .iter()
            .any(|e| e.name == "new_" && e.kind == FileKind::Directory));
        assert!(root
            .iter()
            .any(|e| e.name == "amp_" && e.kind == FileKind::Directory));
        let js = fs.list_dir("/new_/js").await.expect("js dir");
        assert!(js
            .iter()
            .any(|e| e.name == "form.js" && e.kind == FileKind::File));
        assert!(fs.list_dir("/new_/js/form.js").await.is_err()); // a file

        // Managed rows are listed read-only.
        let lib = fs.list_dir("/amp_/lib").await.expect("managed dir");
        let managed = lib
            .iter()
            .find(|e| e.name == "managed.js")
            .expect("managed.js listed");
        assert_eq!(managed.mode, Some(0o444));

        // Upload through the 429-injection hook (first POST 429s, retry wins),
        // then read back and check the publish fired.
        write_file(&session, "/new_/js/rl429.js", b"console.log(1);")
            .await
            .expect("create");
        assert_eq!(publishes().await, baseline + 1, "create published");
        let data = read_file(&session, "/new_/js/rl429.js")
            .await
            .expect("read back");
        assert_eq!(data, b"console.log(1);");
        assert_eq!(file_size(&session, "/new_/js/rl429.js").await, 15);
        assert!(file_exists(&session, "/new_/js/rl429.js").await);
        assert!(file_exists(&session, "/new_/js").await);

        // Update the same file: PATCH + publish.
        write_file(&session, "/new_/js/rl429.js", b"console.log(2);")
            .await
            .expect("update");
        assert_eq!(publishes().await, baseline + 2, "update published");
        assert_eq!(
            read_file(&session, "/new_/js/rl429.js")
                .await
                .expect("re-read"),
            b"console.log(2);"
        );

        // Unsupported extensions fail client-side before any POST.
        assert!(write_file(&session, "/new_/js/notes.txt", b"x")
            .await
            .is_err());

        // Rename = create-new + publish + delete-old.
        fs.rename("/new_/js/rl429.js", "/new_/js/moved.js")
            .await
            .expect("rename");
        assert!(file_exists(&session, "/new_/js/moved.js").await);
        assert!(!file_exists(&session, "/new_/js/rl429.js").await);

        fs.delete("/new_/js/moved.js", false).await.expect("delete");
        assert!(!file_exists(&session, "/new_/js/moved.js").await);

        // mkdir materializes a hidden placeholder; recursive delete walks it.
        fs.create_dir("/new_/js/ghostftp-dir").await.expect("mkdir");
        let js2 = fs.list_dir("/new_/js").await.expect("relist");
        assert!(js2.iter().any(|e| e.name == "ghostftp-dir"));
        let empty = fs
            .list_dir("/new_/js/ghostftp-dir")
            .await
            .expect("list keep dir");
        assert!(empty.is_empty(), "placeholder hidden from listing");
        fs.delete("/new_/js/ghostftp-dir", true)
            .await
            .expect("rmdir");
        assert!(!file_exists(&session, "/new_/js/ghostftp-dir").await);

        // Managed resources refuse every mutation, client-side.
        for err in [
            write_file(&session, "/amp_/lib/managed.js", b"x")
                .await
                .unwrap_err(),
            fs.delete("/amp_/lib/managed.js", false).await.unwrap_err(),
            fs.rename("/amp_/lib/managed.js", "/amp_/lib/x.js")
                .await
                .unwrap_err(),
        ] {
            assert!(err.to_string().contains("managed web resource"), "{err}");
        }

        // chmod is unsupported.
        assert!(fs.chmod("/new_/js/form.js", 0o644).await.is_err());

        // Throttle honored: a burst of writes is paced at ≥100 ms apart —
        // 4 sequential ops can't finish in under ~300 ms of pacing.
        let start = std::time::Instant::now();
        for i in 0..4 {
            write_file(&session, &format!("/new_/js/pace{i}.js"), b"x")
                .await
                .expect("paced write");
        }
        assert!(
            start.elapsed() >= std::time::Duration::from_millis(300),
            "throttle should pace the writes, took {:?}",
            start.elapsed()
        );
        fs.delete("/new_/js", true).await.expect("cleanup");

        crate::credentials::delete_secret(&credential_key(&pid));
        eprintln!("live_dynamics_roundtrip: OK");
    }
}
