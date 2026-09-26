use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::gdrive::normalize;
use crate::session::hubspot::{FileEntry, HubDbTable, HubSpotSession, NodeMeta};
use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use std::sync::Arc;

/// RemoteFs over three HubSpot surfaces sharing one session. The portal root
/// lists the virtual directories the private app's scopes unlock (probed at
/// connect): `design (draft)` and `design (published)` — labeled so the live
/// one is unmistakable — mapping to the CMS Source Code API v3's two
/// environments (`content` scope), plus `files` for the File Manager (Files
/// API v3, `files` scope) and `hubdb` for HubDB tables as read-only virtual
/// CSV files (`hubdb` scope). Inside the design roots, folders come from the
/// metadata tree (`children` arrays); inside `files`, from the paged
/// folder/file listings (real folders). `/hubdb/` is a flat list of
/// `{table}.csv` files (no subfolders, no writes — HubDB write-back ships in
/// a later phase). No chmod/symlinks; design rename is GET + PUT + DELETE
/// (the API has no atomic rename), File Manager rename is a PATCH (folders go
/// through the async update task). Everything carries size + updated
/// timestamps, so the change signal is `MtimeSize` (no etag exists).
pub struct HubSpotFs {
    session: Arc<HubSpotSession>,
}

impl HubSpotFs {
    pub fn new(session: Arc<HubSpotSession>) -> Self {
        Self { session }
    }

    /// Rename within the File Manager: files via `PATCH /files/v3/files/{id}`,
    /// folders via the async update task (`PATCH …/folders/{id}` + poll).
    async fn files_rename(&self, from_inner: &str, to_inner: &str) -> Result<()> {
        if from_inner.is_empty() || to_inner.is_empty() {
            return Err(anyhow!("the /files root can't be renamed from Ghost FTP"));
        }
        let from_parent = from_inner.rsplit_once('/').map(|(p, _)| p).unwrap_or("");
        let to_parent = to_inner.rsplit_once('/').map(|(p, _)| p).unwrap_or("");
        let to_name = to_inner.rsplit('/').next().unwrap_or(to_inner);
        if from_parent != to_parent {
            return Err(anyhow!(
                "moving across /files/ folders isn't supported — copy + delete instead"
            ));
        }
        let entry = self
            .session
            .files_stat(from_inner)
            .await
            .ok_or_else(|| anyhow!("hubspot /files/{from_inner}: not found"))?;
        if entry.folder {
            self.session.files_rename_folder(&entry.id, to_name).await
        } else {
            self.session.files_rename_file(&entry.id, to_name).await
        }
    }

    /// Delete within the File Manager. Recursive folder delete walks the tree
    /// with an explicit stack (the session's throttle keeps this inside the
    /// rate limit), deleting files, then folders bottom-up.
    async fn files_delete(&self, inner: &str, recursive: bool) -> Result<()> {
        if inner.is_empty() {
            return Err(anyhow!("the /files root can't be deleted from Ghost FTP"));
        }
        let entry = self
            .session
            .files_stat(inner)
            .await
            .ok_or_else(|| anyhow!("hubspot /files/{inner}: not found"))?;
        if !entry.folder {
            return self.session.files_delete_file(&entry.id).await;
        }
        if !recursive {
            return Err(anyhow!(
                "/files/{inner} is a directory — delete it recursively"
            ));
        }
        let mut stack = vec![(inner.to_string(), entry.id.clone())];
        let mut folders: Vec<String> = Vec::new();
        while let Some((path, id)) = stack.pop() {
            folders.push(id);
            for child in self.session.files_list(&path).await? {
                if child.folder {
                    stack.push((child.path.trim_matches('/').to_string(), child.id.clone()));
                } else {
                    self.session.files_delete_file(&child.id).await?;
                }
            }
        }
        for id in folders.iter().rev() {
            self.session.files_delete_folder(id).await?;
        }
        Ok(())
    }
}

/// Hidden placeholder that materializes "empty directories" — the Source Code
/// API has none; a folder exists only while a file sits under it. The `.txt`
/// extension keeps it inside the upload whitelist.
pub const KEEP_NAME: &str = ".ghostftp-keep.txt";

/// Content written for the [`KEEP_NAME`] placeholder file.
const KEEP_CONTENT: &[u8] = b"# ghostftp placeholder - keeps this directory on the portal\n";

/// The virtual root directory labels → Source Code API environments.
pub const DRAFT_DIR: &str = "design (draft)";
pub const PUBLISHED_DIR: &str = "design (published)";

/// The virtual root for the File Manager (Files API v3) — real folders, no
/// extension whitelist, uploads default to `PUBLIC_NOT_INDEXABLE`.
pub const FILES_DIR: &str = "files";

/// The virtual root for HubDB (HubDB API v3): a flat list of
/// `{table-name}.csv` virtual files. READ-ONLY — write-back (CSV draft
/// import + publish, or sparse row PATCH) ships in a later phase, once
/// row-id round-tripping is safe against the import's row-deletion
/// semantics.
pub const HUBDB_DIR: &str = "hubdb";

/// The read-only error every mutation under `/hubdb/` returns, phrased like
/// the HTTP backend's (`HTTP source is read-only (…)`).
fn hubdb_read_only(op: &str) -> anyhow::Error {
    anyhow!("/hubdb/ is read-only ({op} not supported) — HubDB write-back lands in a later phase")
}

/// Entering a root the private app's scopes don't cover. Connect hides these
/// roots, but a stale tree node or a deep link can still land here.
fn missing_scope(root: &str, scope: &str) -> anyhow::Error {
    anyhow!(
        "{root} needs the `{scope}` scope — enable it in HubSpot → Settings → Integrations → \
         Private Apps → your app → Scopes, then reconnect"
    )
}

/// The inner path under `/hubdb/` ("" for the hubdb root itself), or `None`
/// for other roots. The root is flat — a `/` in the inner path never
/// resolves to a table.
pub fn hubdb_inner(norm_path: &str) -> Option<String> {
    let t = norm_path.trim_start_matches('/');
    if t == HUBDB_DIR {
        return Some(String::new());
    }
    t.strip_prefix(&format!("{HUBDB_DIR}/"))
        .map(|rest| rest.trim_matches('/').to_string())
}

/// The inner path under `/files/` ("" for the files root itself), or `None`
/// when the path belongs to a design environment. This prefix check is the
/// split point: design-env paths go to the Source Code API, `/files/` paths
/// to the Files API.
pub fn files_inner(norm_path: &str) -> Option<String> {
    let t = norm_path.trim_start_matches('/');
    if t == FILES_DIR {
        return Some(String::new());
    }
    t.strip_prefix(&format!("{FILES_DIR}/"))
        .map(|rest| rest.trim_matches('/').to_string())
}

/// Split a Ghost FTP path into (Source Code environment, path within it).
/// `/design (draft)/themes/x/main.css` → `("draft", "themes/x/main.css")`.
/// The inner path is empty for an environment root. Errors on anything
/// outside the two virtual roots.
pub fn split_path(ghostftp_path: &str) -> Result<(&'static str, String)> {
    let norm = normalize(ghostftp_path);
    let t = norm.trim_start_matches('/');
    let (root, rest) = match t.split_once('/') {
        Some((root, rest)) => (root, rest.trim_matches('/').to_string()),
        None => (t, String::new()),
    };
    let env = match root {
        DRAFT_DIR => "draft",
        PUBLISHED_DIR => "published",
        _ => {
            return Err(anyhow!(
                "{ghostftp_path}: unknown HubSpot root — expected /{DRAFT_DIR}, /{PUBLISHED_DIR}, /{FILES_DIR} or /{HUBDB_DIR}"
            ))
        }
    };
    Ok((env, rest))
}

/// Read one file's bytes, for the transfer + editor + preview arms.
pub async fn read_file(session: &HubSpotSession, ghostftp_path: &str) -> Result<Vec<u8>> {
    if let Some(inner) = hubdb_inner(&normalize(ghostftp_path)) {
        if inner.is_empty() {
            return Err(anyhow!("{ghostftp_path} is a directory, not a file"));
        }
        return session.hubdb_read_csv(&inner).await;
    }
    if let Some(inner) = files_inner(&normalize(ghostftp_path)) {
        if inner.is_empty() {
            return Err(anyhow!("{ghostftp_path} is a directory, not a file"));
        }
        return session.files_read(&inner).await;
    }
    let (env, path) = split_path(ghostftp_path)?;
    if path.is_empty() {
        return Err(anyhow!("{ghostftp_path} is a design environment, not a file"));
    }
    session.content_get(env, &path).await
}

/// Write one file's bytes, for the transfer + editor arms. `/files/` uploads
/// skip the Design Manager extension whitelist (the Files API accepts
/// arbitrary types) and default to `PUBLIC_NOT_INDEXABLE`. `/hubdb/` is
/// read-only.
pub async fn write_file(session: &HubSpotSession, ghostftp_path: &str, data: &[u8]) -> Result<()> {
    if hubdb_inner(&normalize(ghostftp_path)).is_some() {
        return Err(hubdb_read_only("write"));
    }
    if let Some(inner) = files_inner(&normalize(ghostftp_path)) {
        if inner.is_empty() {
            return Err(anyhow!("cannot write {ghostftp_path}: it is a directory"));
        }
        return session.files_write(&inner, data).await;
    }
    let (env, path) = split_path(ghostftp_path)?;
    if path.is_empty() {
        return Err(anyhow!(
            "cannot write {ghostftp_path}: it is a design environment"
        ));
    }
    session.content_put(env, &path, data).await
}

/// Metadata size for transfer planning (0 when unknown).
pub async fn file_size(session: &HubSpotSession, ghostftp_path: &str) -> u64 {
    if hubdb_inner(&normalize(ghostftp_path)).is_some() {
        // The CSV byte size is unknown without fetching every row — the
        // honest answer is 0, like any other unknown size.
        return 0;
    }
    if let Some(inner) = files_inner(&normalize(ghostftp_path)) {
        if inner.is_empty() {
            return 0;
        }
        return session
            .files_stat(&inner)
            .await
            .map(|e| if e.folder { 0 } else { e.size })
            .unwrap_or(0);
    }
    match split_path(ghostftp_path) {
        Ok((env, path)) if !path.is_empty() => session
            .stat(env, &path)
            .await
            .map(|n| if n.folder { 0 } else { n.size })
            .unwrap_or(0),
        _ => 0,
    }
}

/// Whether a path exists — as a file, or as a folder.
pub async fn file_exists(session: &HubSpotSession, ghostftp_path: &str) -> bool {
    if let Some(inner) = hubdb_inner(&normalize(ghostftp_path)) {
        return inner.is_empty() || session.hubdb_stat(&inner).await.is_some();
    }
    if let Some(inner) = files_inner(&normalize(ghostftp_path)) {
        return inner.is_empty() || session.files_stat(&inner).await.is_some();
    }
    match split_path(ghostftp_path) {
        Ok((_, path)) if path.is_empty() => true, // a known environment
        Ok((env, path)) => session.stat(env, &path).await.is_some(),
        Err(_) => false,
    }
}

/// Every file path under a folder, recursively fetching subfolder metadata
/// (the session's throttle keeps this inside the rate limit).
async fn collect_files(session: &HubSpotSession, env: &str, root: &str) -> Result<Vec<String>> {
    let mut out = Vec::new();
    let mut stack = vec![root.to_string()];
    while let Some(dir) = stack.pop() {
        let node = session.metadata(env, &dir).await?;
        for child in &node.children {
            let path = format!("{dir}/{}", child.name);
            if child.folder {
                stack.push(path);
            } else {
                out.push(path);
            }
        }
    }
    Ok(out)
}

#[async_trait]
impl RemoteFs for HubSpotFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let norm = normalize(path);
        if norm == "/" {
            // The portal root: one virtual directory per surface the app's
            // scopes unlock (detected at connect).
            let s = self.session.surfaces;
            let mut dirs = Vec::new();
            if s.design {
                dirs.push(DRAFT_DIR);
                dirs.push(PUBLISHED_DIR);
            }
            if s.files {
                dirs.push(FILES_DIR);
            }
            if s.hubdb {
                dirs.push(HUBDB_DIR);
            }
            return Ok(dirs
                .into_iter()
                .map(|d| DirEntry {
                    name: d.to_string(),
                    path: format!("/{d}"),
                    kind: FileKind::Directory,
                    size: 0,
                    modified: None,
                    mode: None,
                    etag: None,
                })
                .collect());
        }
        if let Some(inner) = hubdb_inner(&norm) {
            if !self.session.surfaces.hubdb {
                return Err(missing_scope("/hubdb/", "hubdb"));
            }
            if !inner.is_empty() {
                // Flat root: anything deeper is a .csv file, not a directory.
                return Err(anyhow!("{path} is a file, not a directory"));
            }
            let tables = self.session.hubdb_list().await?;
            return Ok(hubdb_entries(norm.trim_end_matches('/'), &tables));
        }
        if let Some(inner) = files_inner(&norm) {
            if !self.session.surfaces.files {
                return Err(missing_scope("/files/", "files"));
            }
            let entries = self.session.files_list(&inner).await?;
            return Ok(file_entries(norm.trim_end_matches('/'), &entries));
        }
        let (env, inner) = split_path(&norm)?;
        if !self.session.surfaces.design {
            return Err(missing_scope("the design roots", "content"));
        }
        let node = self.session.metadata(env, &inner).await?;
        if !node.folder {
            return Err(anyhow!("{path} is a file, not a directory"));
        }
        Ok(children(norm.trim_end_matches('/'), &node.children))
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        if hubdb_inner(&normalize(from)).is_some() || hubdb_inner(&normalize(to)).is_some() {
            return Err(hubdb_read_only("rename"));
        }
        let from_files = files_inner(&normalize(from));
        let to_files = files_inner(&normalize(to));
        match (from_files, to_files) {
            (Some(from_inner), Some(to_inner)) => {
                return self.files_rename(&from_inner, &to_inner).await;
            }
            (None, None) => {}
            _ => {
                return Err(anyhow!(
                    "HubSpot files can't move between the File Manager and the design roots"
                ))
            }
        }
        let (env, from_path) = split_path(from)?;
        let (to_env, to_path) = split_path(to)?;
        if env != to_env {
            return Err(anyhow!(
                "HubSpot files can't move across draft/published — copy + delete instead"
            ));
        }
        if from_path.is_empty() || to_path.is_empty() {
            return Err(anyhow!("design environments can't be renamed from Ghost FTP"));
        }
        let node = self.session.metadata(env, &from_path).await?;
        if !node.folder {
            // No atomic rename in the Source Code API: PUT the new path,
            // DELETE the old one (best-effort — a crash between the two
            // leaves a copy).
            let data = self.session.content_get(env, &from_path).await?;
            self.session.content_put(env, &to_path, &data).await?;
            self.session
                .content_delete(env, &from_path)
                .await
                .with_context(|| {
                    format!("rename {from}: new path written, old path delete failed")
                })?;
            return Ok(());
        }
        // Directory rename: every file under the folder, recursively.
        let files = collect_files(&self.session, env, &from_path).await?;
        for f in files {
            let data = self.session.content_get(env, &f).await?;
            let new_path = format!("{to_path}/{}", &f[from_path.len() + 1..]);
            self.session.content_put(env, &new_path, &data).await?;
            self.session.content_delete(env, &f).await?;
        }
        Ok(())
    }

    async fn delete(&self, path: &str, recursive: bool) -> Result<()> {
        if hubdb_inner(&normalize(path)).is_some() {
            return Err(hubdb_read_only("delete"));
        }
        if let Some(inner) = files_inner(&normalize(path)) {
            return self.files_delete(&inner, recursive).await;
        }
        let (env, inner) = split_path(path)?;
        if inner.is_empty() {
            return Err(anyhow!("design environments can't be deleted from Ghost FTP"));
        }
        let node = self.session.metadata(env, &inner).await?;
        if !node.folder {
            return self.session.content_delete(env, &inner).await;
        }
        if !recursive {
            return Err(anyhow!("{path} is a directory — delete it recursively"));
        }
        // Walk the metadata tree, deleting every file (the session's throttle
        // keeps this inside the rate limit).
        for f in collect_files(&self.session, env, &inner).await? {
            self.session.content_delete(env, &f).await?;
        }
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        if hubdb_inner(&normalize(path)).is_some() {
            return Err(hubdb_read_only("create dir"));
        }
        if let Some(inner) = files_inner(&normalize(path)) {
            // Real folders — no `.ghostftp-keep` placeholder needed under /files/.
            return self.session.files_create_folder(&inner).await;
        }
        let (env, inner) = split_path(path)?;
        if inner.is_empty() {
            return Err(anyhow!(
                "design environments are created by HubSpot, not here"
            ));
        }
        // Materialize the folder with a hidden placeholder file.
        self.session
            .content_put(env, &format!("{inner}/{KEEP_NAME}"), KEEP_CONTENT)
            .await
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

/// Direct children of a folder as DirEntries. The [`KEEP_NAME`] placeholder
/// is hidden from listings.
fn children(base: &str, nodes: &[NodeMeta]) -> Vec<DirEntry> {
    nodes
        .iter()
        .filter(|n| n.folder || n.name != KEEP_NAME)
        .map(|n| DirEntry {
            name: n.name.clone(),
            path: format!("{base}/{}", n.name),
            kind: if n.folder {
                FileKind::Directory
            } else {
                FileKind::File
            },
            size: if n.folder { 0 } else { n.size },
            modified: n.updated_at,
            mode: None,
            etag: None,
        })
        .collect()
}

/// A Files API listing as DirEntries (`/files/` root contents — real
/// folders, no placeholder filtering).
fn file_entries(base: &str, entries: &[FileEntry]) -> Vec<DirEntry> {
    entries
        .iter()
        .map(|e| DirEntry {
            name: e.name.clone(),
            path: format!("{base}/{}", e.name),
            kind: if e.folder {
                FileKind::Directory
            } else {
                FileKind::File
            },
            size: if e.folder { 0 } else { e.size },
            modified: e.updated_at,
            mode: None,
            etag: None,
        })
        .collect()
}

/// The HubDB table listing as DirEntries: one virtual `{name}.csv` file per
/// table (the table `name`, not the label — already lowercase/underscore
/// safe). Size is 0 because the CSV byte size is unknowable without fetching
/// every row (the honest answer — a row count would masquerade as bytes);
/// `modified` carries the table's `updatedAt` for the `MtimeSize` signal.
fn hubdb_entries(base: &str, tables: &[HubDbTable]) -> Vec<DirEntry> {
    tables
        .iter()
        .map(|t| DirEntry {
            name: format!("{}.csv", t.name),
            path: format!("{base}/{}.csv", t.name),
            kind: FileKind::File,
            size: 0,
            modified: t.updated_at,
            mode: None,
            etag: None,
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::session::hubspot::{
        file_entry_from_json, files_name_stem, node_from_json, parse_timestamp, upload_allowed,
    };

    #[test]
    fn path_mapping() {
        assert_eq!(
            split_path("/design (draft)").expect("env root"),
            ("draft", String::new())
        );
        assert_eq!(
            split_path("/design (draft)/themes/x/main.css").expect("file"),
            ("draft", "themes/x/main.css".to_string())
        );
        assert_eq!(
            split_path("/design (published)/templates/").expect("trailing slash"),
            ("published", "templates".to_string())
        );
        assert!(split_path("/").is_err());
        assert!(split_path("/files/library").is_err()); // design-only split
    }

    #[test]
    fn files_root_split() {
        assert_eq!(files_inner("/files"), Some(String::new()));
        assert_eq!(
            files_inner("/files/library/logo.png"),
            Some("library/logo.png".to_string())
        );
        assert_eq!(files_inner("/files/"), Some(String::new()));
        assert_eq!(files_inner("/design (draft)/x"), None);
        assert_eq!(files_inner("/files2/x"), None); // prefix boundary
        assert_eq!(files_inner("/"), None);
    }

    #[test]
    fn hubdb_root_split() {
        assert_eq!(hubdb_inner("/hubdb"), Some(String::new()));
        assert_eq!(
            hubdb_inner("/hubdb/pricing.csv"),
            Some("pricing.csv".to_string())
        );
        assert_eq!(
            hubdb_inner("/hubdb/sub/pricing.csv"),
            Some("sub/pricing.csv".to_string()) // rejected downstream: flat root
        );
        assert_eq!(hubdb_inner("/hubdb/"), Some(String::new()));
        assert_eq!(hubdb_inner("/design (draft)/x"), None);
        assert_eq!(hubdb_inner("/hubdb2/x"), None); // prefix boundary
        assert_eq!(hubdb_inner("/"), None);
    }

    #[test]
    fn hubdb_csv_serialization() {
        use crate::session::hubspot::{hubdb_to_csv, HubDbRow};
        let columns: Vec<String> = ["plan", "price", "notes"].iter().map(|s| s.to_string()).collect();
        let row = |id: &str, values: serde_json::Map<String, serde_json::Value>| HubDbRow {
            id: id.to_string(),
            values,
        };

        // Quoting: commas, double quotes and embedded newlines; nulls empty.
        let mut values = serde_json::Map::new();
        values.insert("plan".into(), serde_json::Value::from("Pro, \"annual\"\nplan"));
        values.insert("price".into(), serde_json::json!(49.5));
        values.insert("notes".into(), serde_json::Value::Null);
        // A row with missing keys renders empty cells.
        let rows = vec![row("102", values), row("103", serde_json::Map::new())];
        let csv = String::from_utf8(hubdb_to_csv(&columns, &rows)).expect("utf-8");
        assert_eq!(
            csv,
            "id,plan,price,notes\r\n102,\"Pro, \"\"annual\"\"\nplan\",49.5,\r\n103,,,\r\n"
        );

        // Unicode passes through unquoted; booleans render plain.
        let mut values = serde_json::Map::new();
        values.insert("plan".into(), serde_json::Value::from("café ☕"));
        values.insert("price".into(), serde_json::json!(true));
        let csv = String::from_utf8(hubdb_to_csv(&columns, &[row("9", values)])).expect("utf-8");
        assert_eq!(csv, "id,plan,price,notes\r\n9,café ☕,true,\r\n");

        // Structured values (multi-select arrays) render as compact JSON,
        // quoted because of the commas/quotes inside.
        let mut values = serde_json::Map::new();
        values.insert("notes".into(), serde_json::json!(["a", "b"]));
        let csv = String::from_utf8(hubdb_to_csv(&columns, &[row("7", values)])).expect("utf-8");
        assert_eq!(csv, "id,plan,price,notes\r\n7,,,\"[\"\"a\"\",\"\"b\"\"]\"\r\n");

        // An empty table is header-only.
        let csv = String::from_utf8(hubdb_to_csv(&columns, &[])).expect("utf-8");
        assert_eq!(csv, "id,plan,price,notes\r\n");
    }

    #[test]
    fn hubdb_table_json_to_dir_entry() {
        use crate::session::hubspot::{hubdb_row_from_json, hubdb_table_from_json};
        let table = hubdb_table_from_json(&serde_json::json!({
            "id": 7001,
            "name": "pricing",
            "label": "Pricing",
            "updatedAt": "2024-07-15T09:30:00Z",
            "columns": [{"name": "plan", "type": "TEXT"}, {"name": "price", "type": "NUMBER"}]
        }))
        .expect("table");
        assert_eq!(table.id, "7001");
        assert_eq!(table.name, "pricing");
        assert_eq!(table.columns, vec!["plan", "price"]);
        assert_eq!(table.updated_at, Some(1_721_035_800));

        // Summaries carry no columns; malformed entries are skipped.
        let summary = hubdb_table_from_json(&serde_json::json!({
            "id": "7002", "name": "team", "updatedAt": 1721035800000i64
        }))
        .expect("summary");
        assert!(summary.columns.is_empty());
        assert_eq!(summary.updated_at, Some(1_721_035_800));
        assert!(hubdb_table_from_json(&serde_json::json!({"id": 1})).is_none());

        // Rows tolerate numeric ids and missing values maps.
        let row = hubdb_row_from_json(&serde_json::json!({"id": 101, "values": {"plan": "Pro"}}))
            .expect("row");
        assert_eq!(row.id, "101");
        assert_eq!(row.values.len(), 1);
        assert!(hubdb_row_from_json(&serde_json::json!({"values": {}})).is_none());

        let entries = hubdb_entries("/hubdb", &[table, summary]);
        assert_eq!(entries.len(), 2);
        assert_eq!(entries[0].name, "pricing.csv");
        assert_eq!(entries[0].path, "/hubdb/pricing.csv");
        assert_eq!(entries[0].kind, FileKind::File);
        assert_eq!(entries[0].size, 0); // unknown until rows are fetched
        assert_eq!(entries[0].modified, Some(1_721_035_800));
        assert_eq!(entries[1].name, "team.csv");
    }

    #[test]
    fn files_api_json_to_entry() {
        // A folder result (v3 shape: ISO-8601 timestamps, string id).
        let folder = file_entry_from_json(
            &serde_json::json!({
                "id": "100",
                "name": "library",
                "path": "/library",
                "createdAt": "2024-07-15T09:30:00Z",
                "updatedAt": "2024-07-15T09:30:00Z"
            }),
            true,
        )
        .expect("folder");
        assert!(folder.folder);
        assert_eq!(folder.name, "library");
        assert_eq!(folder.updated_at, Some(1_721_035_800));

        // A file result: `name` excludes the extension — the display name
        // comes from `path`; numeric ids are tolerated.
        let file = file_entry_from_json(
            &serde_json::json!({
                "id": 200,
                "name": "logo",
                "path": "/library/logo.png",
                "size": 24574,
                "access": "PRIVATE",
                "defaultHostingUrl": "https://cdn.example/library/logo.png",
                "updatedAt": "2024-07-15T09:30:00Z"
            }),
            false,
        )
        .expect("file");
        assert!(!file.folder);
        assert_eq!(file.id, "200");
        assert_eq!(file.name, "logo.png");
        assert_eq!(file.size, 24574);
        assert_eq!(file.access, "PRIVATE");
        assert_eq!(
            file.url.as_deref(),
            Some("https://cdn.example/library/logo.png")
        );

        // Malformed entries are skipped, not fatal.
        assert!(file_entry_from_json(&serde_json::json!({"name": "x"}), false).is_none());

        let root = file_entries("/files", std::slice::from_ref(&folder));
        assert_eq!(root.len(), 1);
        assert_eq!(root[0].kind, FileKind::Directory);
        assert_eq!(root[0].path, "/files/library");
        let library = file_entries("/files/library", std::slice::from_ref(&file));
        assert_eq!(library[0].kind, FileKind::File);
        assert_eq!(library[0].size, 24574);
        assert_eq!(library[0].path, "/files/library/logo.png");
    }

    #[test]
    fn rename_sends_name_stem() {
        assert_eq!(files_name_stem("logo.png"), "logo");
        assert_eq!(files_name_stem("archive.tar.gz"), "archive.tar");
        assert_eq!(files_name_stem("Makefile"), "Makefile");
        assert_eq!(files_name_stem(".gitkeep"), ".gitkeep");
    }

    #[test]
    fn upload_extension_whitelist() {
        assert!(upload_allowed("themes/x/main.css"));
        assert!(upload_allowed("themes/x/module.module/module.html"));
        assert!(upload_allowed("assets/font.WOFF2")); // case-insensitive
        assert!(upload_allowed(&format!("dir/{KEEP_NAME}")));
        assert!(!upload_allowed("evil.exe"));
        assert!(!upload_allowed("archive.tar.gz"));
        assert!(!upload_allowed("Makefile"));
    }

    #[test]
    fn metadata_json_to_dir_entry() {
        let v = serde_json::json!({
            "name": "themes",
            "folder": true,
            "createdAt": 1721035800000i64,
            "updatedAt": 1721035800000i64,
            "children": [
                {"name": "example", "folder": true, "updatedAt": 1721035800000i64, "children": []},
                {"name": "main.css", "folder": false, "size": 12, "updatedAt": 1721035800000i64}
            ]
        });
        let node = node_from_json(&v).expect("parse");
        assert!(node.folder);
        assert_eq!(node.name, "themes");
        assert_eq!(node.children.len(), 2);

        let entries = children("/design (draft)", &node.children);
        assert_eq!(entries.len(), 2);
        let dir = entries.iter().find(|e| e.name == "example").expect("dir");
        assert_eq!(dir.kind, FileKind::Directory);
        assert_eq!(dir.path, "/design (draft)/example");
        let css = entries.iter().find(|e| e.name == "main.css").expect("css");
        assert_eq!(css.kind, FileKind::File);
        assert_eq!(css.size, 12);
        assert_eq!(css.path, "/design (draft)/main.css");
        // Epoch-millis timestamps land in seconds.
        assert_eq!(css.modified, Some(1_721_035_800));
        assert_eq!(
            parse_timestamp(&serde_json::json!(1_721_035_800_000i64)),
            Some(1_721_035_800)
        );
        assert_eq!(
            parse_timestamp(&serde_json::json!("2024-07-15T09:30:00Z")),
            Some(1_721_035_800)
        );
    }

    #[test]
    fn keep_placeholder_filtered() {
        let v = serde_json::json!({
            "name": "newdir",
            "folder": true,
            "children": [
                {"name": KEEP_NAME, "folder": false, "size": 57},
                {"name": "real.css", "folder": false, "size": 4}
            ]
        });
        let node = node_from_json(&v).expect("parse");
        let entries = children("/design (draft)/newdir", &node.children);
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].name, "real.css");
    }

    /// End-to-end against the HubSpot mock (`tests/hubspot_mock.py`). Skipped
    /// unless GHOSTFTP_HUBSPOT_MOCK_URL is set; run with
    /// `cargo test -p ghostftp -- --ignored hubspot_roundtrip` after starting it.
    /// Exercises connect (token probe + portal label), both design
    /// environments, the metadata walk, upload through the 429 retry hook,
    /// read-back, rename (GET+PUT+DELETE), placeholder mkdir, recursive
    /// delete, and the extension whitelist — plus the `/files/` root: paged
    /// listing, mkdir, upload (no whitelist), public CDN read, private
    /// signed-url read, file PATCH rename, async folder rename, and
    /// recursive delete — plus the `/hubdb/` root: paged table listing, CSV
    /// reads (header order, escaping, nulls, empty tables), and the
    /// read-only errors on every mutation.
    #[tokio::test]
    #[ignore = "requires the HubSpot mock (GHOSTFTP_HUBSPOT_MOCK_URL)"]
    async fn live_hubspot_roundtrip() {
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::hubspot::{credential_key, hubspot_connect};

        let Ok(mock) = std::env::var("GHOSTFTP_HUBSPOT_MOCK_URL") else {
            eprintln!("skip: GHOSTFTP_HUBSPOT_MOCK_URL unset");
            return;
        };
        std::env::set_var("GHOSTFTP_HUBSPOT_API_BASE", &mock);

        let profile = ConnectionProfile {
            icon: None,
            id: "hubspot-mock-test".into(),
            name: "portal".into(),
            protocol: "hubspot".into(),
            host: String::new(),
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

        let pid = profile.id.clone();
        crate::credentials::set_secret(&credential_key(&pid), "pat-test").expect("seed secret");
        let session = Arc::new(hubspot_connect(&profile).await.expect("connect"));
        assert_eq!(
            session.account_label().await.expect("label"),
            "HubSpot portal 987654321"
        );

        let fs = HubSpotFs::new(session.clone());
        let roots = fs.list_dir("/").await.expect("roots");
        assert_eq!(roots.len(), 4);
        assert!(roots.iter().any(|e| e.name == DRAFT_DIR));
        assert!(roots.iter().any(|e| e.name == PUBLISHED_DIR));
        assert!(roots.iter().any(|e| e.name == FILES_DIR));
        assert!(roots.iter().any(|e| e.name == HUBDB_DIR));

        // Walk the draft tree.
        let env_root = fs.list_dir("/design (draft)").await.expect("draft root");
        assert!(env_root
            .iter()
            .any(|e| e.name == "themes" && e.kind == FileKind::Directory));
        let theme = fs
            .list_dir("/design (draft)/themes/example")
            .await
            .expect("theme");
        assert!(theme
            .iter()
            .any(|e| e.name == "main.css" && e.kind == FileKind::File));

        // Upload through the 429-injection hook (first PUT 429s, retry wins).
        let js = "/design (draft)/themes/example/assets/rl429.js";
        write_file(&session, js, b"console.log(1);")
            .await
            .expect("put");
        let data = read_file(&session, js).await.expect("get");
        assert_eq!(data, b"console.log(1);");
        assert_eq!(file_size(&session, js).await, 15);

        // The extension whitelist rejects client-side before any PUT.
        assert!(write_file(&session, "/design (draft)/evil.exe", b"x")
            .await
            .is_err());

        // Rename = GET + PUT new path + DELETE old path.
        let moved = "/design (draft)/themes/example/assets/moved.js";
        fs.rename(js, moved).await.expect("rename");
        assert!(file_exists(&session, moved).await);
        assert!(!file_exists(&session, js).await);

        fs.delete(moved, false).await.expect("delete");
        assert!(!file_exists(&session, moved).await);

        // mkdir materializes a hidden placeholder; recursive delete walks it.
        fs.create_dir("/design (draft)/themes/example/ghostftp-dir")
            .await
            .expect("mkdir");
        let theme2 = fs
            .list_dir("/design (draft)/themes/example")
            .await
            .expect("relist");
        assert!(theme2.iter().any(|e| e.name == "ghostftp-dir"));
        let empty = fs
            .list_dir("/design (draft)/themes/example/ghostftp-dir")
            .await
            .expect("list placeholder dir");
        assert!(empty.is_empty(), "placeholder hidden from listing");
        fs.delete("/design (draft)/themes/example/ghostftp-dir", true)
            .await
            .expect("rmdir");
        assert!(!file_exists(&session, "/design (draft)/themes/example/ghostftp-dir").await);

        // ---- /files/ root (Files API v3) ----

        // Paged listing (the mock pages at 2 entries, forcing cursor loops):
        // root folders + files, then a nested folder.
        let files_root = fs.list_dir("/files").await.expect("files root");
        assert!(files_root
            .iter()
            .any(|e| e.name == "library" && e.kind == FileKind::Directory));
        assert!(files_root
            .iter()
            .any(|e| e.name == "root-note.txt" && e.kind == FileKind::File));
        let library = fs.list_dir("/files/library").await.expect("library");
        assert!(library
            .iter()
            .any(|e| e.name == "logo.png" && e.kind == FileKind::File));
        assert!(library
            .iter()
            .any(|e| e.name == "docs" && e.kind == FileKind::Directory));

        // Public file reads straight off its CDN URL; the PRIVATE one goes
        // through the signed-url flow.
        let logo = read_file(&session, "/files/library/logo.png")
            .await
            .expect("public read");
        assert_eq!(logo, b"png-bytes");
        let secret = read_file(&session, "/files/library/docs/secret.pdf")
            .await
            .expect("signed-url read");
        assert_eq!(secret, b"pdf-bytes");

        // mkdir is a real folder (no placeholder) — listed back immediately.
        fs.create_dir("/files/ghostftp-qa").await.expect("mkdir");
        let files_root2 = fs.list_dir("/files").await.expect("relist root");
        assert!(files_root2
            .iter()
            .any(|e| e.name == "ghostftp-qa" && e.kind == FileKind::Directory));

        // Upload any file type (no Design Manager whitelist under /files/).
        let report = "/files/ghostftp-qa/report.exe";
        write_file(&session, report, b"quarterly numbers")
            .await
            .expect("upload");
        assert_eq!(read_file(&session, report).await.expect("read back"),
            b"quarterly numbers");
        assert_eq!(file_size(&session, report).await, 17);
        assert!(file_exists(&session, report).await);

        // File rename = PATCH name (same folder only).
        let renamed = "/files/ghostftp-qa/report-final.exe";
        fs.rename(report, renamed).await.expect("file rename");
        assert!(file_exists(&session, renamed).await);
        assert!(!file_exists(&session, report).await);
        assert!(fs.rename(renamed, "/files/library/x.exe").await.is_err());

        // Folder rename = PATCH folder + async task poll; the tree refresh
        // sees the moved file under the new folder name.
        fs.rename("/files/ghostftp-qa", "/files/ghostftp-qa2")
            .await
            .expect("folder rename");
        assert!(file_exists(&session, "/files/ghostftp-qa2/report-final.exe").await);
        assert!(!file_exists(&session, "/files/ghostftp-qa").await);

        // Recursive delete walks files, then folders bottom-up.
        fs.delete("/files/ghostftp-qa2", true)
            .await
            .expect("recursive delete");
        assert!(!file_exists(&session, "/files/ghostftp-qa2").await);

        // ---- /hubdb/ root (HubDB API v3, read-only virtual CSVs) ----

        // Paged table listing (the mock pages at 2 entries, forcing the
        // cursor loop across all 3 tables): one .csv file per table, size 0
        // (unknowable without fetching rows), modified from updatedAt.
        let hubdb_root = fs.list_dir("/hubdb").await.expect("hubdb root");
        assert_eq!(hubdb_root.len(), 3);
        let pricing = hubdb_root
            .iter()
            .find(|e| e.name == "pricing.csv")
            .expect("pricing.csv");
        assert_eq!(pricing.kind, FileKind::File);
        assert_eq!(pricing.path, "/hubdb/pricing.csv");
        assert_eq!(pricing.size, 0);
        assert_eq!(pricing.modified, Some(1_721_035_800));
        assert!(hubdb_root.iter().any(|e| e.name == "team.csv"));
        assert!(hubdb_root.iter().any(|e| e.name == "archive.csv"));
        assert!(fs.list_dir("/hubdb/pricing.csv").await.is_err()); // not a dir
        assert!(file_exists(&session, "/hubdb").await);
        assert!(file_exists(&session, "/hubdb/pricing.csv").await);
        assert!(!file_exists(&session, "/hubdb/nope.csv").await);

        // CSV read: `id` first, then columns in schema order; RFC-4180
        // quoting (comma, quotes, embedded newline), null + missing values
        // empty, unicode unquoted, `\r\n` line endings. Rows page at 2.
        let csv = read_file(&session, "/hubdb/pricing.csv")
            .await
            .expect("pricing csv");
        let text = String::from_utf8(csv).expect("utf-8");
        assert_eq!(
            text,
            "id,plan,price,notes\r\n\
             101,Starter,0,\r\n\
             102,\"Pro, annual\",49.5,\"said \"\"hi\"\"\nthen left\"\r\n\
             103,Café plan ☕,500,\r\n"
        );
        assert_eq!(file_size(&session, "/hubdb/pricing.csv").await, 0);

        // An empty table is header-only; unknown tables 404.
        let csv = read_file(&session, "/hubdb/archive.csv")
            .await
            .expect("empty table");
        assert_eq!(String::from_utf8(csv).expect("utf-8"), "id,archived\r\n");
        assert!(read_file(&session, "/hubdb/nope.csv").await.is_err());

        // Every mutation is refused client-side with the read-only error.
        for err in [
            write_file(&session, "/hubdb/pricing.csv", b"id,plan\n")
                .await
                .unwrap_err(),
            fs.rename("/hubdb/pricing.csv", "/hubdb/x.csv")
                .await
                .unwrap_err(),
            fs.delete("/hubdb/pricing.csv", false).await.unwrap_err(),
            fs.create_dir("/hubdb/sub").await.unwrap_err(),
        ] {
            assert!(err.to_string().contains("read-only"), "{err}");
        }

        // The design roots are untouched by all of the above.
        let live = read_file(
            &session,
            "/design (published)/themes/example/templates/index.html",
        )
        .await
        .expect("published read");
        assert_eq!(live, b"<html>live</html>");

        crate::credentials::delete_secret(&credential_key(&pid));
        eprintln!("live_hubspot_roundtrip: OK");
    }

    /// A token whose app lacks the `content` scope still connects: the
    /// design roots drop out of the portal root, entering one names the
    /// missing scope, and the remaining surfaces work. Same mock run as
    /// `live_hubspot_roundtrip` (`GHOSTFTP_HUBSPOT_MOCK_URL`).
    #[tokio::test]
    #[ignore = "requires the HubSpot mock (GHOSTFTP_HUBSPOT_MOCK_URL)"]
    async fn hubspot_missing_scope_degrades() {
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::hubspot::{credential_key, hubspot_connect};

        let Ok(mock) = std::env::var("GHOSTFTP_HUBSPOT_MOCK_URL") else {
            eprintln!("skip: GHOSTFTP_HUBSPOT_MOCK_URL unset");
            return;
        };
        std::env::set_var("GHOSTFTP_HUBSPOT_API_BASE", &mock);

        let profile = ConnectionProfile {
            icon: None,
            id: "hubspot-no-content-test".into(),
            name: "portal".into(),
            protocol: "hubspot".into(),
            host: String::new(),
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

        let pid = profile.id.clone();
        crate::credentials::set_secret(&credential_key(&pid), "pat-no-content")
            .expect("seed secret");
        let session = Arc::new(hubspot_connect(&profile).await.expect("connect"));
        assert!(!session.surfaces.design);
        assert!(session.surfaces.files);
        assert!(session.surfaces.hubdb);

        let fs = HubSpotFs::new(session);
        let roots = fs.list_dir("/").await.expect("roots");
        assert_eq!(roots.len(), 3);
        assert!(!roots.iter().any(|e| e.name == DRAFT_DIR));
        assert!(!roots.iter().any(|e| e.name == PUBLISHED_DIR));
        assert!(roots.iter().any(|e| e.name == FILES_DIR));

        let err = fs
            .list_dir("/design (draft)")
            .await
            .expect_err("design root without content scope");
        assert!(err.to_string().contains("`content`"), "{err}");

        // The remaining surfaces are unaffected.
        let files_root = fs.list_dir("/files").await.expect("files root");
        assert!(files_root.iter().any(|e| e.name == "library"));
        assert!(fs.list_dir("/hubdb").await.expect("hubdb root").len() == 3);

        crate::credentials::delete_secret(&credential_key(&pid));
        eprintln!("hubspot_missing_scope_degrades: OK");
    }
}
