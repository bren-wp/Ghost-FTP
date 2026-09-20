//! `ghostftp.db` — one shared SQLite database for every subsystem that needs a
//! persisted per-connection index (see `docs/plans/3_scan-index-foundation.md`).
//!
//! The win is deliberately modest: **one** connection + **one** migration runner,
//! with **per-feature tables** in the same file. No grand unified schema. This
//! plan lands the `sync_state` table (folder-sync's per-file memory); Plans 6/7/8
//! add `scan_cache` / `search_index` / `diff_hash` as *additive* migrations —
//! never a rewrite.
//!
//! Migrations are forward-only and versioned via SQLite's `user_version` pragma:
//! append a statement to [`MIGRATIONS`] and bump nothing else. `rusqlite`'s
//! `bundled` feature compiles SQLite from source, so there's no system dependency.
//!
//! The connection is guarded by a `std::sync::Mutex`. Every query runs to
//! completion under the lock and returns owned data — the guard is never held
//! across an `.await`, so callers stay `Send`.

use std::collections::HashMap;
use std::path::Path;
use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};

use anyhow::{Context, Result};
use rusqlite::Connection;
use serde::{Deserialize, Serialize};

/// Milliseconds since the Unix epoch — the timestamp unit every `ghostftp.db` row
/// uses. Monotonic enough for ordering; not security-sensitive.
pub fn now_ms() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as i64)
        .unwrap_or(0)
}

/// Forward-only schema migrations. Index = target `user_version - 1`; appending
/// a statement is the *only* supported schema change. Never edit or reorder an
/// existing entry — a shipped `ghostftp.db` has already run it.
const MIGRATIONS: &[&str] = &[
    // v1 — folder-sync per-file index. Keyed by (pair_id, rel_path). `size`/
    // `mtime` are the *source-side* values captured at last sync; `remote_signal`
    // is the backend's opaque change token (ETag/hash) when it has one, else NULL.
    "CREATE TABLE sync_state (
        pair_id        TEXT    NOT NULL,
        rel_path       TEXT    NOT NULL,
        size           INTEGER NOT NULL,
        mtime          INTEGER NOT NULL,
        remote_signal  TEXT,
        last_synced_ms INTEGER NOT NULL,
        state          TEXT    NOT NULL DEFAULT 'synced',
        PRIMARY KEY (pair_id, rel_path)
    ) WITHOUT ROWID;",
    // v2 — command snippets (Plan 11 Phase 4). The low-friction, single-session
    // counterpart to Fleet Skills: a saved command line with optional
    // `{{variable}}` placeholders, insertable into a live shell with one
    // keystroke. `folder` groups them; `use_count` feeds palette ordering.
    "CREATE TABLE snippets (
        id         TEXT    PRIMARY KEY,
        name       TEXT    NOT NULL,
        body       TEXT    NOT NULL,
        folder     TEXT,
        use_count  INTEGER NOT NULL DEFAULT 0,
        created_ms INTEGER NOT NULL,
        updated_ms INTEGER NOT NULL
    );",
    // v3 — keychain manifest (Plan 12 Phase 1). The OS keychain (Windows DPAPI /
    // Linux Keychain) can't *enumerate* its entries, so Ghost FTP records every
    // service-credential it writes here (service + account). The encrypted
    // backup (Phase 4) reads this to know which secrets to pull. OAuth tokens
    // are derived from profiles instead, so this table only holds the app's own
    // service keys.
    "CREATE TABLE keychain_manifest (
        service TEXT NOT NULL,
        account TEXT NOT NULL,
        PRIMARY KEY (service, account)
    ) WITHOUT ROWID;",
    // v4 — settings (Plan 12 Phase 2). One row per key; value is JSON. The
    // single source of truth for app preferences, replacing the webview-private
    // localStorage blob. `updated_ms` lets a later surface reconcile writes.
    "CREATE TABLE settings (
        key        TEXT PRIMARY KEY,
        value      TEXT NOT NULL,
        updated_ms INTEGER NOT NULL
    ) WITHOUT ROWID;",
    // v5 — remote image thumbnail cache index (Plan 13 Phase 1). One row per
    // cached thumbnail file. `key` is the hex hash of
    // (session_id, path, dim, change-signal) and also names the `<key>.png` file
    // in the thumbnails cache dir; including the change-signal means a re-uploaded
    // file yields a fresh key, so edits invalidate. `bytes` + `used_ms` drive the
    // LRU eviction that keeps the cache under its disk budget. `session_id` lets a
    // disconnected/removed connection's thumbnails be dropped wholesale.
    "CREATE TABLE thumb_cache (
        key        TEXT PRIMARY KEY,
        session_id TEXT NOT NULL,
        bytes      INTEGER NOT NULL,
        created_ms INTEGER NOT NULL,
        used_ms    INTEGER NOT NULL
    ) WITHOUT ROWID;",
    // v6 — environment-value backup (Plan 16 Phase 4, one-click PATH install).
    // Before the *first* time we append Ghost FTP's bin dir to the per-user Windows
    // `Path` (HKCU\Environment), we snapshot the exact prior value here so a later
    // "Remove from PATH" can restore it faithfully — including its registry value
    // type (REG_EXPAND_SZ must stay REG_EXPAND_SZ). `raw_value` is the value as a
    // UTF-8 string (NULL if the value was absent); `vtype` is the winreg RegType
    // discriminant so we rewrite with the original type. One row per key
    // ('windows_user_path' today). Insert-once semantics preserve the true
    // original across repeated add/remove cycles.
    "CREATE TABLE env_backup (
        key          TEXT PRIMARY KEY,
        raw_value    TEXT,
        vtype        INTEGER NOT NULL,
        backed_up_ms INTEGER NOT NULL
    ) WITHOUT ROWID;",
];

/// A persisted per-file sync record — what the source looked like the last time
/// this pair synced it. The reconciler diffs the *current* source against this to
/// catch same-size edits the live size+mtime comparison would miss.
#[derive(Debug, Clone)]
pub struct SyncStateRow {
    pub size: u64,
    pub mtime: i64,
    pub remote_signal: Option<String>,
    pub last_synced_ms: i64,
}

/// A saved command snippet (Plan 11 Phase 4). `body` may contain `{{variable}}`
/// placeholders the UI resolves before inserting into a shell. Serialized to the
/// frontend as-is; `id` is a client-minted UUID.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Snippet {
    pub id: String,
    pub name: String,
    pub body: String,
    #[serde(default)]
    pub folder: Option<String>,
    #[serde(default)]
    pub use_count: i64,
    #[serde(default)]
    pub created_ms: i64,
    #[serde(default)]
    pub updated_ms: i64,
}

pub struct Db {
    conn: Mutex<Connection>,
}

impl Db {
    /// Open (creating if absent) `ghostftp.db` at `path` and run pending migrations.
    pub fn open(path: &Path) -> Result<Self> {
        let conn = Connection::open(path)
            .with_context(|| format!("open ghostftp.db at {}", path.display()))?;
        // WAL keeps readers from blocking the single writer — this is a desktop
        // app with occasional background writes, not a hot OLTP path.
        conn.pragma_update(None, "journal_mode", "WAL").ok();
        conn.pragma_update(None, "foreign_keys", "ON").ok();
        migrate(&conn).context("running ghostftp.db migrations")?;
        Ok(Self { conn: Mutex::new(conn) })
    }

    /// Open an in-memory database (tests).
    #[cfg(test)]
    pub fn open_in_memory() -> Result<Self> {
        let conn = Connection::open_in_memory()?;
        migrate(&conn)?;
        Ok(Self { conn: Mutex::new(conn) })
    }

    /// Load every remembered file for a pair, keyed by relative path.
    pub fn load_sync_state(&self, pair_id: &str) -> Result<HashMap<String, SyncStateRow>> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare(
            "SELECT rel_path, size, mtime, remote_signal, last_synced_ms
               FROM sync_state WHERE pair_id = ?1",
        )?;
        let rows = stmt.query_map([pair_id], |r| {
            Ok((
                r.get::<_, String>(0)?,
                SyncStateRow {
                    size: r.get::<_, i64>(1)? as u64,
                    mtime: r.get(2)?,
                    remote_signal: r.get(3)?,
                    last_synced_ms: r.get(4)?,
                },
            ))
        })?;
        let mut out = HashMap::new();
        for row in rows {
            let (k, v) = row?;
            out.insert(k, v);
        }
        Ok(out)
    }

    /// Record (insert or replace) one file's synced state.
    pub fn upsert_sync_state(
        &self,
        pair_id: &str,
        rel_path: &str,
        size: u64,
        mtime: i64,
        remote_signal: Option<&str>,
        last_synced_ms: i64,
    ) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "INSERT INTO sync_state
                 (pair_id, rel_path, size, mtime, remote_signal, last_synced_ms, state)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, 'synced')
             ON CONFLICT(pair_id, rel_path) DO UPDATE SET
                 size = excluded.size,
                 mtime = excluded.mtime,
                 remote_signal = excluded.remote_signal,
                 last_synced_ms = excluded.last_synced_ms,
                 state = 'synced'",
            rusqlite::params![pair_id, rel_path, size as i64, mtime, remote_signal, last_synced_ms],
        )?;
        Ok(())
    }

    /// Forget one file (it's gone from the source and has been mirrored away).
    pub fn delete_sync_state(&self, pair_id: &str, rel_path: &str) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "DELETE FROM sync_state WHERE pair_id = ?1 AND rel_path = ?2",
            rusqlite::params![pair_id, rel_path],
        )?;
        Ok(())
    }

    /// Drop a pair's entire index (the pair was deleted).
    pub fn clear_pair(&self, pair_id: &str) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute("DELETE FROM sync_state WHERE pair_id = ?1", [pair_id])?;
        Ok(())
    }

    // ---- Command snippets (Plan 11 Phase 4) ----

    /// Every snippet, most-used first (ties broken by most-recently-edited).
    /// The palette and panel both consume this ordering directly.
    pub fn list_snippets(&self) -> Result<Vec<Snippet>> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare(
            "SELECT id, name, body, folder, use_count, created_ms, updated_ms
               FROM snippets
              ORDER BY use_count DESC, updated_ms DESC",
        )?;
        let rows = stmt.query_map([], |r| {
            Ok(Snippet {
                id: r.get(0)?,
                name: r.get(1)?,
                body: r.get(2)?,
                folder: r.get(3)?,
                use_count: r.get(4)?,
                created_ms: r.get(5)?,
                updated_ms: r.get(6)?,
            })
        })?;
        rows.collect::<rusqlite::Result<Vec<_>>>().map_err(Into::into)
    }

    /// Insert or update a snippet. An update preserves `use_count`/`created_ms`
    /// (they're the snippet's history, not part of the edit) and refreshes
    /// `updated_ms`. Returns the full, re-ordered list so callers stay in sync.
    pub fn upsert_snippet(
        &self,
        id: &str,
        name: &str,
        body: &str,
        folder: Option<&str>,
    ) -> Result<Vec<Snippet>> {
        let now = now_ms();
        {
            let conn = self.conn.lock().unwrap();
            conn.execute(
                "INSERT INTO snippets (id, name, body, folder, use_count, created_ms, updated_ms)
                 VALUES (?1, ?2, ?3, ?4, 0, ?5, ?5)
                 ON CONFLICT(id) DO UPDATE SET
                     name = excluded.name,
                     body = excluded.body,
                     folder = excluded.folder,
                     updated_ms = excluded.updated_ms",
                rusqlite::params![id, name, body, folder, now],
            )?;
        }
        self.list_snippets()
    }

    /// Delete one snippet; returns the surviving list.
    pub fn delete_snippet(&self, id: &str) -> Result<Vec<Snippet>> {
        {
            let conn = self.conn.lock().unwrap();
            conn.execute("DELETE FROM snippets WHERE id = ?1", [id])?;
        }
        self.list_snippets()
    }

    /// Record one use of a snippet (its text was inserted into a shell) so it
    /// floats up the ordering. Returns the re-ordered list.
    pub fn touch_snippet(&self, id: &str) -> Result<Vec<Snippet>> {
        {
            let conn = self.conn.lock().unwrap();
            conn.execute(
                "UPDATE snippets SET use_count = use_count + 1 WHERE id = ?1",
                [id],
            )?;
        }
        self.list_snippets()
    }

    // ---- Keychain manifest (Plan 12 Phase 1) ----

    /// Remember that a `(service, account)` keychain entry exists (idempotent).
    pub fn record_keychain(&self, service: &str, account: &str) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "INSERT OR IGNORE INTO keychain_manifest (service, account) VALUES (?1, ?2)",
            rusqlite::params![service, account],
        )?;
        Ok(())
    }

    /// Forget a `(service, account)` entry (its secret was cleared).
    pub fn forget_keychain(&self, service: &str, account: &str) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "DELETE FROM keychain_manifest WHERE service = ?1 AND account = ?2",
            rusqlite::params![service, account],
        )?;
        Ok(())
    }

    /// Every recorded `(service, account)` — the list of app service-credentials
    /// the backup should pull from the keychain.
    pub fn list_keychain(&self) -> Result<Vec<(String, String)>> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn
            .prepare("SELECT service, account FROM keychain_manifest ORDER BY service, account")?;
        let rows = stmt.query_map([], |r| Ok((r.get(0)?, r.get(1)?)))?;
        rows.collect::<rusqlite::Result<Vec<_>>>().map_err(Into::into)
    }

    // ---- Settings (Plan 12 Phase 2) ----

    /// Every setting as `key -> raw JSON value`. Seeds the frontend store on boot.
    pub fn settings_get_all(&self) -> Result<HashMap<String, String>> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare("SELECT key, value FROM settings")?;
        let rows = stmt.query_map([], |r| Ok((r.get::<_, String>(0)?, r.get::<_, String>(1)?)))?;
        let mut out = HashMap::new();
        for row in rows {
            let (k, v) = row?;
            out.insert(k, v);
        }
        Ok(out)
    }

    /// One setting's raw JSON value, or `None`. Used by the pre-paint injector
    /// to read theme/accent before the window's first paint.
    pub fn settings_get(&self, key: &str) -> Result<Option<String>> {
        let conn = self.conn.lock().unwrap();
        let v = conn
            .query_row("SELECT value FROM settings WHERE key = ?1", [key], |r| {
                r.get::<_, String>(0)
            })
            .ok();
        Ok(v)
    }

    /// Upsert one setting (`value` is a raw JSON string).
    pub fn settings_set(&self, key: &str, value: &str) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "INSERT INTO settings (key, value, updated_ms) VALUES (?1, ?2, ?3)
             ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_ms = excluded.updated_ms",
            rusqlite::params![key, value, now_ms()],
        )?;
        Ok(())
    }

    /// Delete one setting row. Used by "reset to default" flows (e.g. clearing a
    /// keyboard-shortcut override so the built-in binding applies again). Removing
    /// the row is what makes "default" distinct from an explicit stored value.
    pub fn settings_delete(&self, key: &str) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute("DELETE FROM settings WHERE key = ?1", [key])?;
        Ok(())
    }

    // ---- Environment-value backup (Plan 16 Phase 4, PATH install) ----

    /// Snapshot the prior environment value **once**, before Ghost FTP's first write.
    /// Idempotent: a second call with the same `key` is a no-op, so the stored
    /// value always reflects the true pre-Ghost FTP state across add/remove cycles.
    /// `vtype` is the winreg `RegType` discriminant; `raw_value` is `None` when the
    /// value didn't exist yet.
    pub fn env_backup_set_once(
        &self,
        key: &str,
        raw_value: Option<&str>,
        vtype: u32,
    ) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "INSERT OR IGNORE INTO env_backup (key, raw_value, vtype, backed_up_ms)
             VALUES (?1, ?2, ?3, ?4)",
            rusqlite::params![key, raw_value, vtype, now_ms()],
        )?;
        Ok(())
    }

    /// The backed-up `(raw_value, vtype)` for `key`, or `None` if never captured.
    pub fn env_backup_get(&self, key: &str) -> Result<Option<(Option<String>, u32)>> {
        let conn = self.conn.lock().unwrap();
        let row = conn
            .query_row(
                "SELECT raw_value, vtype FROM env_backup WHERE key = ?1",
                [key],
                |r| Ok((r.get::<_, Option<String>>(0)?, r.get::<_, i64>(1)? as u32)),
            )
            .ok();
        Ok(row)
    }

    /// Write a consistent, fully-checkpointed snapshot of the database to
    /// `dest` (Plan 12 Phase 4 backup). `VACUUM INTO` folds any WAL into the
    /// copy, so the snapshot reflects all committed data even while the live DB
    /// is open in WAL mode. `dest` must not already exist.
    pub fn snapshot_to(&self, dest: &std::path::Path) -> Result<()> {
        let _ = std::fs::remove_file(dest);
        let escaped = dest.to_string_lossy().replace('\'', "''");
        let conn = self.conn.lock().unwrap();
        conn.execute_batch(&format!("VACUUM INTO '{escaped}'"))
            .with_context(|| format!("VACUUM INTO {}", dest.display()))?;
        Ok(())
    }

    // ---- Remote thumbnail cache index (Plan 13 Phase 1) ----

    /// True if a live cache row exists for `key`, bumping its `used_ms` so the LRU
    /// keeps recently-viewed thumbnails. The caller still confirms the `<key>.png`
    /// file is on disk before trusting a `true` (index and files can drift if the
    /// cache dir is cleared out-of-band).
    pub fn thumb_touch(&self, key: &str) -> Result<bool> {
        let conn = self.conn.lock().unwrap();
        let n = conn.execute(
            "UPDATE thumb_cache SET used_ms = ?2 WHERE key = ?1",
            rusqlite::params![key, now_ms()],
        )?;
        Ok(n > 0)
    }

    /// Record a freshly-written thumbnail, then evict least-recently-used rows
    /// until the total cached bytes fall under `budget_bytes`. Returns the `key`s
    /// whose files the caller must delete from disk (the index row is already
    /// gone). The just-inserted row is never evicted — it's the most-recently-used.
    pub fn thumb_record(
        &self,
        key: &str,
        session_id: &str,
        bytes: u64,
        budget_bytes: u64,
    ) -> Result<Vec<String>> {
        let now = now_ms();
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "INSERT INTO thumb_cache (key, session_id, bytes, created_ms, used_ms)
             VALUES (?1, ?2, ?3, ?4, ?4)
             ON CONFLICT(key) DO UPDATE SET bytes = excluded.bytes, used_ms = excluded.used_ms",
            rusqlite::params![key, session_id, bytes as i64, now],
        )?;

        let total: i64 = conn
            .query_row("SELECT COALESCE(SUM(bytes), 0) FROM thumb_cache", [], |r| r.get(0))?;
        if (total as u64) <= budget_bytes {
            return Ok(Vec::new());
        }

        // Evict oldest-used first, but never the row we just wrote.
        let mut over = total as u64 - budget_bytes;
        let mut evicted = Vec::new();
        let mut stmt = conn.prepare(
            "SELECT key, bytes FROM thumb_cache WHERE key != ?1 ORDER BY used_ms ASC",
        )?;
        let victims: Vec<(String, i64)> = stmt
            .query_map([key], |r| Ok((r.get::<_, String>(0)?, r.get::<_, i64>(1)?)))?
            .collect::<rusqlite::Result<Vec<_>>>()?;
        drop(stmt);
        for (k, b) in victims {
            if over == 0 {
                break;
            }
            conn.execute("DELETE FROM thumb_cache WHERE key = ?1", [&k])?;
            over = over.saturating_sub(b as u64);
            evicted.push(k);
        }
        Ok(evicted)
    }

    /// Drop the index rows for a session's thumbnails, returning their `key`s so
    /// the caller can delete the files. Used when a connection is removed.
    pub fn thumb_clear_session(&self, session_id: &str) -> Result<Vec<String>> {
        let conn = self.conn.lock().unwrap();
        let keys: Vec<String> = {
            let mut stmt =
                conn.prepare("SELECT key FROM thumb_cache WHERE session_id = ?1")?;
            let rows = stmt
                .query_map([session_id], |r| r.get::<_, String>(0))?
                .collect::<rusqlite::Result<Vec<_>>>()?;
            rows
        };
        conn.execute("DELETE FROM thumb_cache WHERE session_id = ?1", [session_id])?;
        Ok(keys)
    }

    /// Bulk-upsert settings in one transaction (the one-time localStorage import).
    pub fn settings_set_many(&self, entries: &[(String, String)]) -> Result<()> {
        let mut conn = self.conn.lock().unwrap();
        let now = now_ms();
        let tx = conn.transaction()?;
        for (k, v) in entries {
            tx.execute(
                "INSERT INTO settings (key, value, updated_ms) VALUES (?1, ?2, ?3)
                 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_ms = excluded.updated_ms",
                rusqlite::params![k, v, now],
            )?;
        }
        tx.commit()?;
        Ok(())
    }
}

/// Apply every migration whose index is `>= user_version`, bumping the pragma in
/// lockstep so a half-applied run resumes cleanly on the next open.
fn migrate(conn: &Connection) -> Result<()> {
    let version: i64 = conn.query_row("PRAGMA user_version", [], |r| r.get(0))?;
    for (i, sql) in MIGRATIONS.iter().enumerate().skip(version as usize) {
        conn.execute_batch(sql)
            .with_context(|| format!("apply migration v{}", i + 1))?;
        // `user_version` can't be a bound parameter — the value is our own index.
        conn.pragma_update(None, "user_version", (i + 1) as i64)?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn migrates_and_round_trips_sync_state() {
        let db = Db::open_in_memory().unwrap();
        assert!(db.load_sync_state("p1").unwrap().is_empty());

        db.upsert_sync_state("p1", "a/b.txt", 42, 1000, Some("etag-1"), 5000)
            .unwrap();
        db.upsert_sync_state("p1", "c.txt", 7, 2000, None, 5001).unwrap();
        db.upsert_sync_state("p2", "a/b.txt", 99, 3000, None, 5002)
            .unwrap();

        let p1 = db.load_sync_state("p1").unwrap();
        assert_eq!(p1.len(), 2);
        let row = &p1["a/b.txt"];
        assert_eq!(row.size, 42);
        assert_eq!(row.mtime, 1000);
        assert_eq!(row.remote_signal.as_deref(), Some("etag-1"));
        assert_eq!(row.last_synced_ms, 5000);

        // upsert replaces in place
        db.upsert_sync_state("p1", "a/b.txt", 43, 1500, None, 6000)
            .unwrap();
        let p1 = db.load_sync_state("p1").unwrap();
        assert_eq!(p1.len(), 2);
        assert_eq!(p1["a/b.txt"].size, 43);
        assert_eq!(p1["a/b.txt"].remote_signal, None);

        db.delete_sync_state("p1", "a/b.txt").unwrap();
        assert_eq!(db.load_sync_state("p1").unwrap().len(), 1);

        // p2 is untouched by p1 operations
        assert_eq!(db.load_sync_state("p2").unwrap().len(), 1);
        db.clear_pair("p2").unwrap();
        assert!(db.load_sync_state("p2").unwrap().is_empty());
    }

    #[test]
    fn snippets_crud_and_ordering() {
        let db = Db::open_in_memory().unwrap();
        assert!(db.list_snippets().unwrap().is_empty());

        db.upsert_snippet("a", "List sites", "ls {{dir}}", Some("wp"))
            .unwrap();
        let list = db
            .upsert_snippet("b", "Restart nginx", "sudo systemctl restart nginx", None)
            .unwrap();
        assert_eq!(list.len(), 2);

        // Using `a` twice floats it above `b` — use_count is the primary sort key.
        db.touch_snippet("a").unwrap();
        let list = db.touch_snippet("a").unwrap();
        assert_eq!(list[0].id, "a");
        assert_eq!(list[0].use_count, 2);

        // Editing preserves use_count and created_ms, refreshes body.
        let created = list[0].created_ms;
        let list = db
            .upsert_snippet("a", "List a dir", "ls -la {{dir}}", Some("wp"))
            .unwrap();
        let a = list.iter().find(|s| s.id == "a").unwrap();
        assert_eq!(a.use_count, 2, "edit must not reset usage");
        assert_eq!(a.created_ms, created, "edit must not reset created_ms");
        assert_eq!(a.body, "ls -la {{dir}}");
        assert_eq!(a.name, "List a dir");

        let list = db.delete_snippet("b").unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].id, "a");
    }

    #[test]
    fn open_creates_file_and_persists_across_reopen() {
        // Exercises the exact code path startup uses: Db::open on a real path,
        // WAL pragma, migrations, and durability across a reopen.
        let mut path = std::env::temp_dir();
        path.push(format!("ghostftp_db_open_test_{}.db", std::process::id()));
        let _ = std::fs::remove_file(&path);

        {
            let db = Db::open(&path).unwrap();
            db.upsert_sync_state("p", "f.txt", 1, 2, None, 3).unwrap();
        }
        assert!(path.exists(), "Db::open must create the file");

        // Reopen: migrations don't re-run destructively and data survives.
        let db = Db::open(&path).unwrap();
        let rows = db.load_sync_state("p").unwrap();
        assert_eq!(rows.len(), 1);
        assert_eq!(rows["f.txt"].size, 1);

        drop(db);
        let _ = std::fs::remove_file(&path);
        let _ = std::fs::remove_file(path.with_extension("db-wal"));
        let _ = std::fs::remove_file(path.with_extension("db-shm"));
    }

    #[test]
    fn keychain_manifest_round_trips() {
        let db = Db::open_in_memory().unwrap();
        assert!(db.list_keychain().unwrap().is_empty());

        db.record_keychain("com.ghostftp.credentials", "anthropic-api-key")
            .unwrap();
        // Idempotent: a second record of the same pair is a no-op.
        db.record_keychain("com.ghostftp.credentials", "anthropic-api-key")
            .unwrap();
        db.record_keychain("com.ghostftp.credentials", "openai-api-key")
            .unwrap();
        let list = db.list_keychain().unwrap();
        assert_eq!(list.len(), 2);
        assert!(list.contains(&(
            "com.ghostftp.credentials".to_string(),
            "anthropic-api-key".to_string()
        )));

        db.forget_keychain("com.ghostftp.credentials", "anthropic-api-key")
            .unwrap();
        let list = db.list_keychain().unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].1, "openai-api-key");
    }

    #[test]
    fn settings_round_trip_and_bulk_import() {
        let db = Db::open_in_memory().unwrap();
        assert!(db.settings_get_all().unwrap().is_empty());
        assert!(db.settings_get("appTheme").unwrap().is_none());

        db.settings_set("appTheme", "\"dracula\"").unwrap();
        db.settings_set("terminalFontSize", "13").unwrap();
        assert_eq!(db.settings_get("appTheme").unwrap().as_deref(), Some("\"dracula\""));

        // Upsert replaces in place.
        db.settings_set("appTheme", "\"nord\"").unwrap();
        assert_eq!(db.settings_get("appTheme").unwrap().as_deref(), Some("\"nord\""));

        // Bulk import merges/overwrites.
        db.settings_set_many(&[
            ("appTheme".into(), "\"tokyo\"".into()),
            ("accentColor".into(), "\"#ff0000\"".into()),
        ])
        .unwrap();
        let all = db.settings_get_all().unwrap();
        assert_eq!(all.len(), 3);
        assert_eq!(all["appTheme"], "\"tokyo\"");
        assert_eq!(all["accentColor"], "\"#ff0000\"");
        assert_eq!(all["terminalFontSize"], "13");
    }

    #[test]
    fn thumb_cache_touch_record_and_lru_evict() {
        // `used_ms` has millisecond resolution; space the ordered operations out
        // so the LRU tiebreak is deterministic (real views are seconds apart).
        let gap = || std::thread::sleep(std::time::Duration::from_millis(2));
        let db = Db::open_in_memory().unwrap();
        // Nothing cached yet.
        assert!(!db.thumb_touch("k1").unwrap());

        // Budget of 100 bytes. Three 40-byte thumbnails overflow it.
        assert!(db.thumb_record("k1", "s1", 40, 100).unwrap().is_empty());
        gap();
        assert!(db.thumb_record("k2", "s1", 40, 100).unwrap().is_empty());
        gap();

        // Touch k1 so it's more-recently-used than k2 — k2 becomes the LRU victim.
        assert!(db.thumb_touch("k1").unwrap());
        gap();
        let evicted = db.thumb_record("k3", "s1", 40, 100).unwrap();
        assert_eq!(evicted, vec!["k2".to_string()], "least-recently-used is evicted first");
        // The just-written row is never a victim.
        assert!(db.thumb_touch("k3").unwrap());
        assert!(db.thumb_touch("k1").unwrap());
        assert!(!db.thumb_touch("k2").unwrap());

        // Clearing a session returns its surviving keys and empties the index.
        let mut cleared = db.thumb_clear_session("s1").unwrap();
        cleared.sort();
        assert_eq!(cleared, vec!["k1".to_string(), "k3".to_string()]);
        assert!(!db.thumb_touch("k1").unwrap());
    }

    #[test]
    fn env_backup_captures_once() {
        let db = Db::open_in_memory().unwrap();
        assert!(db.env_backup_get("windows_user_path").unwrap().is_none());

        // First capture wins and sticks.
        db.env_backup_set_once("windows_user_path", Some("C:\\A;C:\\B"), 2)
            .unwrap();
        // A second capture (after Ghost FTP already modified PATH) must NOT overwrite —
        // the backup has to stay the true pre-Ghost FTP value.
        db.env_backup_set_once("windows_user_path", Some("C:\\A;C:\\B;C:\\Ghost FTP\\bin"), 2)
            .unwrap();

        let (val, vtype) = db.env_backup_get("windows_user_path").unwrap().unwrap();
        assert_eq!(val.as_deref(), Some("C:\\A;C:\\B"));
        assert_eq!(vtype, 2);

        // A NULL prior value (the value didn't exist) round-trips as None.
        db.env_backup_set_once("other_key", None, 1).unwrap();
        let (val, vtype) = db.env_backup_get("other_key").unwrap().unwrap();
        assert_eq!(val, None);
        assert_eq!(vtype, 1);
    }

    #[test]
    fn migrate_is_idempotent_across_opens() {
        // Running migrate twice on the same connection must not error or duplicate.
        let conn = Connection::open_in_memory().unwrap();
        migrate(&conn).unwrap();
        migrate(&conn).unwrap();
        let v: i64 = conn.query_row("PRAGMA user_version", [], |r| r.get(0)).unwrap();
        assert_eq!(v as usize, MIGRATIONS.len());
    }
}
