//! Fleet Search engine (see `docs/plans/7_fleet-search.md`).
//!
//! Spotlight/Everything for a connection: instant **filename** search and
//! **content grep** across a remote tree — servers, buckets, agents, or local.
//! Like Disk Usage (Plan 4) and Diff (Plan 6) it works over **any** `RemoteFs`,
//! and — where the backend has a shell — an exec fast path (`rg`/`grep`/`find`)
//! makes content search on a big server genuinely fast instead of thousands of
//! byte-reads.
//!
//! Two query kinds live in one [`SearchQuery`]:
//! * [`SearchKind::Name`] — glob/substring against each entry's name,
//! * [`SearchKind::Content`] — literal or regex grep across file contents.
//!
//! Strategy is picked by capability (fastest first), mirroring `diskscan`:
//! 1. **Exec fast path** (SSH / Ghost FTP Agent) — one `rg`/`grep`/`find` instead of
//!    a walk (Phase 2).
//! 2. **Object stores** — name search over a flat key listing; content requires
//!    fetching every object, so it's opt-in ([`SearchQuery::content_remote`]).
//! 3. **Generic walk fallback** — [`crate::scan`] / a `RemoteFs::list_dir` BFS
//!    for names; for content, read candidate files and match client-side, always
//!    bounded (size cap) and cancellable. Always available.
//!
//! The pure matchers ([`ContentMatcher`], [`NameMatcher`], [`Filters`]) are
//! I/O-free and unit-tested; [`run_search`] streams hits into a [`HitSink`]
//! (capped, cancellable) so the CLI, the `ghostftp_search` bridge tool, and the GUI
//! [`SearchManager`] all drive the same engine. Symlinks are skipped, exactly as
//! the walk already skips them.

use std::collections::VecDeque;
use std::time::Duration;

use anyhow::{bail, Context, Result};
use futures::stream::{FuturesUnordered, StreamExt};
use serde::{Deserialize, Serialize};

use crate::remotefs::{FileKind, RemoteFs};
use crate::scan::{self, CancelToken, ScanOptions};
use crate::session::{AgentSession, FtpSession, ObjectSession, Session, SshSession};

/// Output cap on a fast-path exec (`rg`/`grep`/`find`). Beyond this the result
/// would be partial, so the runner bails to the (complete) generic walk.
const MAX_EXEC_BYTES: usize = 64 * 1024 * 1024;
/// Wall-clock budget for one fast-path exec.
const EXEC_TIMEOUT: Duration = Duration::from_secs(120);
const EXEC_TIMEOUT_MS: u64 = 120_000;

/// Default cap on hits returned (a line for content, an entry for name). Keeps a
/// runaway query from ballooning memory / the response; surfaced as `truncated`.
pub const DEFAULT_MAX_RESULTS: usize = 1000;

/// Default per-file byte cap for content search. Files larger than this are
/// skipped (they're usually media/build artifacts, not what a grep wants) so a
/// content walk never streams a multi-GB blob just to grep it.
pub const DEFAULT_MAX_FILE_BYTES: u64 = 8 * 1024 * 1024;

/// Which kind of search this is.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SearchKind {
    /// Match each entry's name (glob when the pattern has `*`/`?`, else substring).
    Name,
    /// Grep file contents (literal by default, regex when `regex` is set).
    Content,
}

/// A search request. `Default` is an empty name search with the standard caps.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchQuery {
    /// What to look for. For name search a `*`/`?` makes it a glob against the
    /// entry name; otherwise it's a substring. For content search it's a literal
    /// unless `regex` is set.
    pub pattern: String,
    pub kind: SearchKind,
    /// Content: treat `pattern` as a regular expression instead of a literal.
    /// (Name search always auto-detects glob vs substring; this flag is ignored.)
    #[serde(default)]
    pub regex: bool,
    #[serde(default)]
    pub case_sensitive: bool,
    /// Only consider files whose name matches one of these globs (empty = all).
    #[serde(default)]
    pub include_globs: Vec<String>,
    /// Skip files whose name matches any of these globs.
    #[serde(default)]
    pub exclude_globs: Vec<String>,
    /// Opt-in: allow content search to DOWNLOAD every candidate file on backends
    /// that have no server-side grep (object stores, FTP, WebDAV, cloud). Off by
    /// default so a content query on S3 doesn't silently pull the whole bucket.
    #[serde(default)]
    pub content_remote: bool,
    #[serde(default = "default_max_results")]
    pub max_results: usize,
    #[serde(default = "default_max_file_bytes")]
    pub max_file_bytes: u64,
}

fn default_max_results() -> usize {
    DEFAULT_MAX_RESULTS
}
fn default_max_file_bytes() -> u64 {
    DEFAULT_MAX_FILE_BYTES
}

impl Default for SearchQuery {
    fn default() -> Self {
        Self {
            pattern: String::new(),
            kind: SearchKind::Name,
            regex: false,
            case_sensitive: false,
            include_globs: Vec::new(),
            exclude_globs: Vec::new(),
            content_remote: false,
            max_results: DEFAULT_MAX_RESULTS,
            max_file_bytes: DEFAULT_MAX_FILE_BYTES,
        }
    }
}

/// One search hit. Name hits carry just the entry; content hits add the matching
/// line's number, the byte column of the match, and a trimmed preview.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchHit {
    /// Absolute path on the backend (what "reveal" / "open" acts on).
    pub path: String,
    /// Path relative to the search root (the stable display key).
    pub relative: String,
    pub is_dir: bool,
    pub size: u64,
    /// Content hits only: 1-based line number.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub line: Option<u64>,
    /// Content hits only: byte column of the match within the line (0-based).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub column: Option<u64>,
    /// Content hits only: the matching line, trimmed and length-capped.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preview: Option<String>,
}

impl SearchHit {
    fn name(path: String, relative: String, is_dir: bool, size: u64) -> Self {
        SearchHit {
            path,
            relative,
            is_dir,
            size,
            line: None,
            column: None,
            preview: None,
        }
    }

    fn content(
        path: String,
        relative: String,
        size: u64,
        line: u64,
        column: u64,
        preview: String,
    ) -> Self {
        SearchHit {
            path,
            relative,
            is_dir: false,
            size,
            line: Some(line),
            column: Some(column),
            preview: Some(preview),
        }
    }
}

/// Which strategy produced a search. `Generic` is the always-available walk; the
/// `Shell` exec fast path and `ObjectFlat` name listing land in Phase 2 and set
/// this so a surface can show which path fired (and why, if it fell back).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SearchStrategy {
    /// Recursive `RemoteFs::list_dir` walk (names) or read-and-grep (content).
    Generic,
    /// One `rg`/`grep`/`find` over the SSH exec channel or a Ghost FTP Agent.
    Shell,
    /// A single flat object listing under the prefix (S3/Azure), name-only.
    ObjectFlat,
}

/// What a run produced beyond the hits themselves.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchStats {
    pub strategy: SearchStrategy,
    /// The hit cap was reached — there may be more matches than returned.
    pub truncated: bool,
    /// Files whose contents were read (content search) or directories listed
    /// (name search) — a rough "how much work did this do" for the surface.
    pub scanned: usize,
    /// A short human note, e.g. why a fast path fell back to the generic walk.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub note: Option<String>,
}

/// The collected result the CLI + bridge render (the GUI streams instead).
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchResult {
    pub root: String,
    pub kind: SearchKind,
    pub hits: Vec<SearchHit>,
    pub stats: SearchStats,
}

// ---------- Hit sink (cap + stream) ----------

/// A cap-and-stream target for hits. Strategies push into it; it enforces
/// `max_results` and remembers whether it truncated. The callback lets the GUI
/// stream hits live while the CLI/bridge just collect them into a `Vec`. The
/// callback is `Send` so a search can run inside a spawned task (the bridge).
pub struct HitSink<'a> {
    on_hit: &'a mut (dyn FnMut(SearchHit) + Send),
    count: usize,
    max: usize,
    truncated: bool,
}

impl<'a> HitSink<'a> {
    pub fn new(max: usize, on_hit: &'a mut (dyn FnMut(SearchHit) + Send)) -> Self {
        Self {
            on_hit,
            count: 0,
            max: max.max(1),
            truncated: false,
        }
    }

    /// Record a hit. Returns `true` while the caller should keep producing hits,
    /// `false` once the cap is hit (and flips `truncated`).
    fn push(&mut self, hit: SearchHit) -> bool {
        if self.count >= self.max {
            self.truncated = true;
            return false;
        }
        (self.on_hit)(hit);
        self.count += 1;
        self.count < self.max
    }

    fn wants_more(&self) -> bool {
        self.count < self.max
    }

    pub fn truncated(&self) -> bool {
        self.truncated
    }

    pub fn count(&self) -> usize {
        self.count
    }
}

// ---------- Compiled matchers (pure) ----------

/// Compiled from a [`SearchQuery`] once, then reused across every entry.
struct Compiled {
    name: Option<NameMatcher>,
    content: Option<ContentMatcher>,
    filters: Filters,
}

impl Compiled {
    fn new(q: &SearchQuery) -> Result<Self> {
        if q.pattern.is_empty() {
            bail!("search pattern is empty");
        }
        let (name, content) = match q.kind {
            SearchKind::Name => (Some(NameMatcher::compile(q)?), None),
            SearchKind::Content => (None, Some(ContentMatcher::compile(q)?)),
        };
        Ok(Self {
            name,
            content,
            filters: Filters::compile(q)?,
        })
    }
}

/// Content grep matcher: a literal substring or a compiled regex.
enum ContentMatcher {
    Literal {
        needle: String,
        case_sensitive: bool,
    },
    Regex(regex::Regex),
}

impl ContentMatcher {
    fn compile(q: &SearchQuery) -> Result<Self> {
        if q.regex {
            let re = regex::RegexBuilder::new(&q.pattern)
                .case_insensitive(!q.case_sensitive)
                .build()
                .with_context(|| format!("invalid regex: {}", q.pattern))?;
            Ok(Self::Regex(re))
        } else if q.case_sensitive {
            Ok(Self::Literal {
                needle: q.pattern.clone(),
                case_sensitive: true,
            })
        } else {
            Ok(Self::Literal {
                needle: q.pattern.to_lowercase(),
                case_sensitive: false,
            })
        }
    }

    /// Byte column of the first match in `line`, or `None`.
    fn find(&self, line: &str) -> Option<usize> {
        match self {
            Self::Regex(re) => re.find(line).map(|m| m.start()),
            Self::Literal {
                needle,
                case_sensitive,
            } => {
                if *case_sensitive {
                    line.find(needle.as_str())
                } else {
                    // Column is against the lowercased copy; close enough for a
                    // jump-to-line (the line number is what matters).
                    line.to_lowercase().find(needle.as_str())
                }
            }
        }
    }
}

/// Name matcher: a glob (pattern has `*`/`?`) or a plain substring.
enum NameMatcher {
    Substring {
        needle: String,
        case_sensitive: bool,
    },
    Glob(regex::Regex),
}

impl NameMatcher {
    fn compile(q: &SearchQuery) -> Result<Self> {
        if q.pattern.contains(['*', '?']) {
            let src = format!("^{}$", glob_body(&q.pattern));
            let re = regex::RegexBuilder::new(&src)
                .case_insensitive(!q.case_sensitive)
                .build()
                .with_context(|| format!("invalid name pattern: {}", q.pattern))?;
            Ok(Self::Glob(re))
        } else if q.case_sensitive {
            Ok(Self::Substring {
                needle: q.pattern.clone(),
                case_sensitive: true,
            })
        } else {
            Ok(Self::Substring {
                needle: q.pattern.to_lowercase(),
                case_sensitive: false,
            })
        }
    }

    fn matches(&self, name: &str) -> bool {
        match self {
            Self::Glob(re) => re.is_match(name),
            Self::Substring {
                needle,
                case_sensitive,
            } => {
                if *case_sensitive {
                    name.contains(needle.as_str())
                } else {
                    name.to_lowercase().contains(needle.as_str())
                }
            }
        }
    }
}

/// Include/exclude globs, compiled to anchored regexes and matched against a
/// file's *name* (so `*.log` / `*.rs` work as expected).
struct Filters {
    include: Vec<regex::Regex>,
    exclude: Vec<regex::Regex>,
}

impl Filters {
    fn compile(q: &SearchQuery) -> Result<Self> {
        let build = |globs: &[String]| -> Result<Vec<regex::Regex>> {
            globs
                .iter()
                .filter(|g| !g.trim().is_empty())
                .map(|g| {
                    let src = format!("^{}$", glob_body(g));
                    regex::RegexBuilder::new(&src)
                        .case_insensitive(!q.case_sensitive)
                        .build()
                        .with_context(|| format!("invalid glob: {g}"))
                })
                .collect()
        };
        Ok(Self {
            include: build(&q.include_globs)?,
            exclude: build(&q.exclude_globs)?,
        })
    }

    /// Whether a file name survives the include/exclude filters. Directories are
    /// never filtered here (include/exclude target file names); descent is
    /// separate from result filtering. Case is baked into the compiled globs.
    fn passes(&self, name: &str) -> bool {
        if !self.include.is_empty() && !self.include.iter().any(|r| r.is_match(name)) {
            return false;
        }
        if self.exclude.iter().any(|r| r.is_match(name)) {
            return false;
        }
        true
    }

    fn active(&self) -> bool {
        !self.include.is_empty() || !self.exclude.is_empty()
    }
}

/// Translate a shell glob into a regex body (no anchors): `*` → `.*`, `?` → `.`,
/// everything else escaped. Deliberately simple — no `**` / char classes — since
/// it only ever matches a single path segment (a file/entry name).
fn glob_body(glob: &str) -> String {
    let mut out = String::with_capacity(glob.len() * 2);
    for ch in glob.chars() {
        match ch {
            '*' => out.push_str(".*"),
            '?' => out.push('.'),
            c => out.push_str(&regex::escape(&c.to_string())),
        }
    }
    out
}

/// Basename of a POSIX-relative path (already `/`-separated by `relative_of`).
fn basename(rel: &str) -> &str {
    rel.rsplit('/').next().unwrap_or(rel)
}

// ---------- Public entry points ----------

/// Walk / grep `root` on `fs`, streaming hits into `sink`. `session` (absent for
/// the local FS) unlocks the protocol fast paths: an `rg`/`grep`/`find` exec on
/// SSH/agent, or a flat object listing for name search on S3/Azure. Every fast
/// path falls back to the always-available generic walk on any failure, recording
/// why in the returned note. `sink` carries the truncation flag + hit count.
pub async fn run_search(
    fs: &dyn RemoteFs,
    session: Option<&Session>,
    root: &str,
    query: &SearchQuery,
    cancel: &CancelToken,
    sink: &mut HitSink<'_>,
) -> Result<SearchStats> {
    let compiled = Compiled::new(query)?;
    match query.kind {
        SearchKind::Name => run_name(fs, session, root, query, &compiled, cancel, sink).await,
        SearchKind::Content => run_content(fs, session, root, query, &compiled, cancel, sink).await,
    }
}

fn stats(
    strategy: SearchStrategy,
    sink: &HitSink<'_>,
    scanned: usize,
    note: Option<String>,
) -> SearchStats {
    SearchStats {
        strategy,
        truncated: sink.truncated,
        scanned,
        note,
    }
}

/// Drain a fully-parsed fast-path result into the sink, stopping at the cap.
fn drain(hits: Vec<SearchHit>, sink: &mut HitSink<'_>) {
    for h in hits {
        if !sink.push(h) {
            break;
        }
    }
}

/// A short note explaining why a fast path fell back to the generic walk.
/// Mirrors `diskscan::fallback_walk`.
fn fallback_note(err: &anyhow::Error) -> String {
    let msg = err.to_string();
    let short = msg.lines().next().unwrap_or(&msg);
    if short.contains("denied") {
        "exec disabled — used walk".to_string()
    } else {
        format!("fast path unavailable — used walk ({short})")
    }
}

/// Name search: object-flat listing (S3/Azure) or a `find` exec (SSH/agent) when
/// available, else the generic BFS. The generic walk matches directory names too.
async fn run_name(
    fs: &dyn RemoteFs,
    session: Option<&Session>,
    root: &str,
    query: &SearchQuery,
    compiled: &Compiled,
    cancel: &CancelToken,
    sink: &mut HitSink<'_>,
) -> Result<SearchStats> {
    let fast = match session {
        Some(Session::Object(obj)) => Some((
            SearchStrategy::ObjectFlat,
            name_object(obj, root, query, compiled, cancel).await,
        )),
        Some(Session::Ssh(ssh)) => Some((
            SearchStrategy::Shell,
            name_exec_ssh(ssh, root, query, compiled).await,
        )),
        Some(Session::Agent(agent)) => Some((
            SearchStrategy::Shell,
            name_exec_agent(agent, root, query, compiled).await,
        )),
        _ => None,
    };
    let mut note = None;
    if let Some((strategy, res)) = fast {
        match res {
            Ok(hits) => {
                let scanned = hits.len();
                drain(hits, sink);
                return Ok(stats(strategy, sink, scanned, None));
            }
            Err(e) => {
                tracing::info!("fleet-search name fast path fell back: {e:#}");
                note = Some(fallback_note(&e));
            }
        }
    }
    let scanned = name_walk(fs, root, compiled, cancel, sink).await;
    Ok(stats(SearchStrategy::Generic, sink, scanned, note))
}

/// Content search: an `rg`/`grep` exec on SSH/agent when available, else the
/// generic read-and-grep walk. The walk needs the download opt-in on backends
/// with no server-side grep (object stores, FTP, WebDAV, cloud).
async fn run_content(
    fs: &dyn RemoteFs,
    session: Option<&Session>,
    root: &str,
    query: &SearchQuery,
    compiled: &Compiled,
    cancel: &CancelToken,
    sink: &mut HitSink<'_>,
) -> Result<SearchStats> {
    let fast = match session {
        Some(Session::Ssh(ssh)) => Some(content_exec_ssh(ssh, root, query, compiled).await),
        Some(Session::Agent(agent)) => Some(content_exec_agent(agent, root, query, compiled).await),
        _ => None,
    };
    let mut note = None;
    if let Some(res) = fast {
        match res {
            Ok(hits) => {
                let scanned = hits.len();
                drain(hits, sink);
                return Ok(stats(SearchStrategy::Shell, sink, scanned, None));
            }
            Err(e) => {
                tracing::info!("fleet-search content fast path fell back: {e:#}");
                note = Some(fallback_note(&e));
            }
        }
    }
    if needs_content_optin(session) && !query.content_remote {
        bail!(
            "content search on {} would download every candidate file — re-run with remote content enabled to allow it",
            proto_label(session)
        );
    }
    let scanned = content_walk(fs, session, root, compiled, query, cancel, sink).await?;
    Ok(stats(SearchStrategy::Generic, sink, scanned, note))
}

/// The collecting convenience the CLI + `ghostftp_search` bridge tool call — runs a
/// search to completion and returns every hit (up to the cap). The GUI drives
/// [`run_search`] directly through [`SearchManager`] for streaming + cancel.
pub async fn search(
    fs: &dyn RemoteFs,
    session: Option<&Session>,
    root: &str,
    query: &SearchQuery,
) -> Result<SearchResult> {
    let cancel = CancelToken::new();
    let mut hits: Vec<SearchHit> = Vec::new();
    let stats;
    {
        let mut push = |h: SearchHit| hits.push(h);
        let mut sink = HitSink::new(query.max_results, &mut push);
        stats = run_search(fs, session, root, query, &cancel, &mut sink).await?;
    }
    Ok(SearchResult {
        root: root.to_string(),
        kind: query.kind,
        hits,
        stats,
    })
}

// ---------- Generic name walk (files + dirs, streaming) ----------

/// Bounded-concurrency BFS that streams matching **names** (files and
/// directories) as it goes. Mirrors `scan::walk`'s pipeline but yields per-entry
/// hits and matches directory names too (which `scan::walk` drops). Stops early
/// once the sink is full or the token is cancelled. Returns the directory count.
async fn name_walk(
    fs: &dyn RemoteFs,
    root: &str,
    compiled: &Compiled,
    cancel: &CancelToken,
    sink: &mut HitSink<'_>,
) -> usize {
    let matcher = compiled
        .name
        .as_ref()
        .expect("name search compiles a name matcher");
    let normalized_root = root.trim_end_matches('/').to_string();
    let limit = scan::DEFAULT_CONCURRENCY;

    let mut queue: VecDeque<String> = VecDeque::new();
    queue.push_back(root.to_string());
    let mut inflight = FuturesUnordered::new();
    let mut dirs = 0usize;

    loop {
        if cancel.is_cancelled() || !sink.wants_more() {
            break;
        }
        while inflight.len() < limit {
            match queue.pop_front() {
                Some(dir) => inflight.push(async move { fs.list_dir(&dir).await }),
                None => break,
            }
        }
        let Some(res) = inflight.next().await else {
            break;
        };
        dirs += 1;
        let Ok(entries) = res else { continue };
        for entry in entries {
            let is_dir = matches!(entry.kind, FileKind::Directory);
            if is_dir {
                queue.push_back(entry.path.clone());
            }
            // Skip symlinks/other for matching too — same as the walk.
            if !is_dir && !matches!(entry.kind, FileKind::File) {
                continue;
            }
            if !matcher.matches(&entry.name) {
                continue;
            }
            // include/exclude filter file names; a matching directory bypasses it.
            if !is_dir && !compiled.filters.passes(&entry.name) {
                continue;
            }
            let rel = scan::relative_of(&normalized_root, &entry.path);
            if rel.is_empty() {
                continue;
            }
            if !sink.push(SearchHit::name(entry.path.clone(), rel, is_dir, entry.size)) {
                break;
            }
        }
    }
    dirs
}

// ---------- Generic content walk (read + grep) ----------

/// Enumerate files under `root` (via the shared scan walk), then read and grep
/// each candidate client-side. Bounded by the per-file size cap and fully
/// cancellable — the "always works" content fallback for backends without a
/// server-side grep. Returns the number of files actually read.
async fn content_walk(
    fs: &dyn RemoteFs,
    session: Option<&Session>,
    root: &str,
    compiled: &Compiled,
    query: &SearchQuery,
    cancel: &CancelToken,
    sink: &mut HitSink<'_>,
) -> Result<usize> {
    let matcher = compiled
        .content
        .as_ref()
        .expect("content search compiles a content matcher");
    let opts = ScanOptions {
        concurrency: scan::DEFAULT_CONCURRENCY,
        cancel: cancel.clone(),
    };
    let tree = scan::walk(fs, root, &opts, |_| {})
        .await
        .with_context(|| format!("walking {root}"))?;

    let mut scanned = 0usize;
    for (rel, entry) in &tree.files {
        if cancel.is_cancelled() || !sink.wants_more() {
            break;
        }
        let name = basename(rel);
        if compiled.filters.active() && !compiled.filters.passes(name) {
            continue;
        }
        if entry.size > query.max_file_bytes {
            continue;
        }
        let bytes = match read_capped(session, &entry.absolute, query.max_file_bytes).await {
            Ok(b) => b,
            Err(_) => continue, // unreadable / unsupported: skip, don't fail the search
        };
        let Some(text) = decode_text(&bytes) else {
            continue; // binary
        };
        scanned += 1;
        if !grep_text(&text, matcher, &entry.absolute, rel, entry.size, sink) {
            break;
        }
    }
    Ok(scanned)
}

/// Push a content hit for every matching line in `text`. Returns `false` once the
/// sink is full so the caller stops enumerating files.
fn grep_text(
    text: &str,
    matcher: &ContentMatcher,
    abs: &str,
    rel: &str,
    size: u64,
    sink: &mut HitSink<'_>,
) -> bool {
    for (idx, raw) in text.split('\n').enumerate() {
        let line = raw.strip_suffix('\r').unwrap_or(raw);
        if let Some(col) = matcher.find(line) {
            let hit = SearchHit::content(
                abs.to_string(),
                rel.to_string(),
                size,
                idx as u64 + 1,
                col as u64,
                preview(line),
            );
            if !sink.push(hit) {
                return false;
            }
        }
    }
    true
}

/// Trim and length-cap a matching line for display.
fn preview(line: &str) -> String {
    const MAX: usize = 300;
    let trimmed = line.trim();
    if trimmed.chars().count() > MAX {
        let capped: String = trimmed.chars().take(MAX).collect();
        format!("{capped}…")
    } else {
        trimmed.to_string()
    }
}

/// Decode read bytes as text, or `None` if they look binary (a NUL byte in the
/// first 8 KiB — the same heuristic `grep`/`ripgrep` use).
fn decode_text(bytes: &[u8]) -> Option<String> {
    if bytes.iter().take(8192).any(|&b| b == 0) {
        return None;
    }
    Some(String::from_utf8_lossy(bytes).into_owned())
}

/// Backends with no server-side grep, where content search must download every
/// file — gated behind [`SearchQuery::content_remote`]. Local, SSH, and the Ghost FTP
/// Agent are cheap (local read or an exec fast path) and never need the opt-in.
fn needs_content_optin(session: Option<&Session>) -> bool {
    matches!(
        session,
        Some(
            Session::Object(_)
                | Session::Ftp(_)
                | Session::Webdav(_)
                | Session::Http(_)
                | Session::Dropbox(_)
                | Session::OneDrive(_)
                | Session::GDrive(_)
                | Session::Box(_)
                | Session::Shopify(_)
                | Session::HubSpot(_)
                | Session::Dynamics(_)
        )
    )
}

fn proto_label(session: Option<&Session>) -> String {
    session
        .map(|s| s.protocol().to_string())
        .unwrap_or_else(|| "local".to_string())
}

// ---------- Object-store flat name search ----------

/// Name search over one flat object listing under the prefix (S3/Azure/GCS). No
/// recursion — the store returns every key at once, exactly like `diskscan`'s
/// object fast path — so we just name-match each key. Content search over objects
/// takes the (opt-in) generic read walk instead; there's no server-side grep.
async fn name_object(
    obj: &ObjectSession,
    root: &str,
    query: &SearchQuery,
    compiled: &Compiled,
    cancel: &CancelToken,
) -> Result<Vec<SearchHit>> {
    let matcher = compiled
        .name
        .as_ref()
        .expect("name search compiles a name matcher");
    let prefix_raw = root.trim().trim_matches('/');
    let prefix = if prefix_raw.is_empty() || prefix_raw == "." {
        String::new()
    } else {
        prefix_raw.to_string()
    };
    let prefix_path = (!prefix.is_empty()).then(|| object_store::path::Path::from(prefix.as_str()));

    let mut stream = obj.store.list(prefix_path.as_ref());
    let mut hits = Vec::new();
    while let Some(meta) = stream.next().await {
        if cancel.is_cancelled() || hits.len() >= query.max_results {
            break;
        }
        let meta = meta.context("list objects")?;
        let key = meta.location.as_ref().to_string();
        let rel = if prefix.is_empty() {
            key.clone()
        } else {
            key.strip_prefix(&prefix)
                .unwrap_or(&key)
                .trim_start_matches('/')
                .to_string()
        };
        if rel.is_empty() {
            continue;
        }
        let name = basename(&rel);
        if !matcher.matches(name) || !compiled.filters.passes(name) {
            continue;
        }
        hits.push(SearchHit::name(
            format!("/{key}"),
            rel,
            false,
            meta.size as u64,
        ));
    }
    Ok(hits)
}

// ---------- Exec fast path (rg / grep / find) ----------

/// POSIX single-quote a string so it survives spaces / shell metacharacters when
/// interpolated into a remote command.
fn sh_quote(s: &str) -> String {
    format!("'{}'", s.replace('\'', "'\\''"))
}

/// The glob `find -iname` should match. A pattern with `*`/`?` is used verbatim;
/// a plain substring becomes `*substr*` (matching the generic substring rule).
fn find_name_glob(query: &SearchQuery) -> String {
    if query.pattern.contains(['*', '?']) {
        query.pattern.clone()
    } else {
        format!("*{}*", query.pattern)
    }
}

/// `find <root> \( -type f -o -type d \) -iname '<glob>' -printf '%y\t%s\t%p\n'`
/// — one line per matching file/dir: type char, byte size, absolute path.
fn find_name_command(root: &str, query: &SearchQuery) -> String {
    let name_flag = if query.case_sensitive {
        "-name"
    } else {
        "-iname"
    };
    format!(
        "find {} \\( -type f -o -type d \\) {} {} -printf '%y\\t%s\\t%p\\n' 2>/dev/null",
        sh_quote(root),
        name_flag,
        sh_quote(&find_name_glob(query)),
    )
}

/// Parse `find … -printf '%y\t%s\t%p\n'` into name hits. Only files (`f`) and
/// directories (`d`) are kept — symlinks/other are skipped, as the walk skips
/// them. Include/exclude globs are re-applied client-side to file names.
fn parse_find_names(root: &str, out: &str, compiled: &Compiled, max: usize) -> Vec<SearchHit> {
    let normalized_root = root.trim_end_matches('/');
    let mut hits = Vec::new();
    for line in out.lines() {
        if hits.len() >= max {
            break;
        }
        let line = line.trim_end_matches('\r');
        if line.is_empty() {
            continue;
        }
        let mut parts = line.splitn(3, '\t');
        let (Some(ty), Some(size_s), Some(path)) = (parts.next(), parts.next(), parts.next())
        else {
            continue;
        };
        let is_dir = ty == "d";
        if !is_dir && ty != "f" {
            continue;
        }
        let rel = scan::relative_of(normalized_root, path);
        if rel.is_empty() {
            continue;
        }
        let name = basename(&rel);
        if !is_dir && !compiled.filters.passes(name) {
            continue;
        }
        let size = size_s.trim().parse::<u64>().unwrap_or(0);
        hits.push(SearchHit::name(path.to_string(), rel, is_dir, size));
    }
    hits
}

/// `rg --json` command line. `--no-ignore --hidden` searches everything (like the
/// generic walk, which honours no ignore rules); `--max-filesize` mirrors the
/// walk's per-file cap; `-F` is literal, `-i` case-insensitive; `-g`/`-g !` carry
/// the include/exclude globs to the server.
fn rg_command(root: &str, query: &SearchQuery) -> String {
    let mut cmd = format!(
        "rg --json --no-ignore --hidden --max-filesize {}",
        query.max_file_bytes
    );
    if !query.case_sensitive {
        cmd.push_str(" -i");
    }
    if !query.regex {
        cmd.push_str(" -F");
    }
    for g in query.include_globs.iter().filter(|g| !g.trim().is_empty()) {
        cmd.push_str(&format!(" -g {}", sh_quote(g)));
    }
    for g in query.exclude_globs.iter().filter(|g| !g.trim().is_empty()) {
        cmd.push_str(&format!(" -g {}", sh_quote(&format!("!{g}"))));
    }
    cmd.push_str(&format!(
        " -e {} -- {}",
        sh_quote(&query.pattern),
        sh_quote(root)
    ));
    cmd
}

/// Parse `rg --json` output (one JSON object per line) into content hits, keeping
/// only `type == "match"` records. rg reports the byte column of the first
/// submatch, so jump-to-line lands on the match.
fn parse_rg_json(root: &str, out: &str, max: usize) -> Vec<SearchHit> {
    let normalized_root = root.trim_end_matches('/');
    let mut hits = Vec::new();
    for line in out.lines() {
        if hits.len() >= max {
            break;
        }
        let Ok(v) = serde_json::from_str::<serde_json::Value>(line) else {
            continue;
        };
        if v.get("type").and_then(|t| t.as_str()) != Some("match") {
            continue;
        }
        let data = &v["data"];
        let Some(path) = data["path"]["text"].as_str().filter(|p| !p.is_empty()) else {
            continue;
        };
        let lineno = data["line_number"].as_u64().unwrap_or(0);
        let text = data["lines"]["text"].as_str().unwrap_or("");
        let col = data["submatches"][0]["start"].as_u64().unwrap_or(0);
        let rel = scan::relative_of(normalized_root, path);
        hits.push(SearchHit::content(
            path.to_string(),
            rel,
            0,
            lineno,
            col,
            preview(text),
        ));
    }
    hits
}

/// `grep -rn` command line (the fallback when `rg` is absent). `-I` skips binary
/// files (matching the walk's binary skip); `-i`/`-E`/`-F` mirror the query.
fn grep_command(root: &str, query: &SearchQuery) -> String {
    let mut flags = String::from("-rnI");
    if !query.case_sensitive {
        flags.push('i');
    }
    flags.push(if query.regex { 'E' } else { 'F' });
    let mut cmd = format!("grep {flags}");
    for g in query.include_globs.iter().filter(|g| !g.trim().is_empty()) {
        cmd.push_str(&format!(" --include={}", sh_quote(g)));
    }
    for g in query.exclude_globs.iter().filter(|g| !g.trim().is_empty()) {
        cmd.push_str(&format!(" --exclude={}", sh_quote(g)));
    }
    cmd.push_str(&format!(
        " -e {} -- {} 2>/dev/null",
        sh_quote(&query.pattern),
        sh_quote(root)
    ));
    cmd
}

/// Parse `grep -rn` output (`path:lineno:text`) into content hits. The line-number
/// parse doubles as a guard: a line where the second field isn't a number (a path
/// containing `:`) is skipped rather than mis-parsed. The column is recomputed
/// client-side since grep doesn't report one.
fn parse_grep(root: &str, out: &str, compiled: &Compiled, max: usize) -> Vec<SearchHit> {
    let normalized_root = root.trim_end_matches('/');
    let matcher = compiled.content.as_ref();
    let mut hits = Vec::new();
    for line in out.lines() {
        if hits.len() >= max {
            break;
        }
        let line = line.trim_end_matches('\r');
        let mut it = line.splitn(3, ':');
        let (Some(path), Some(num), Some(text)) = (it.next(), it.next(), it.next()) else {
            continue;
        };
        let Ok(lineno) = num.parse::<u64>() else {
            continue;
        };
        let rel = scan::relative_of(normalized_root, path);
        let col = matcher.and_then(|m| m.find(text)).unwrap_or(0) as u64;
        hits.push(SearchHit::content(
            path.to_string(),
            rel,
            0,
            lineno,
            col,
            preview(text),
        ));
    }
    hits
}

/// SSH content: prefer `rg --json`, fall back to `grep -rn`. rg exit 0 (matches)
/// and 1 (no matches) are both a clean run; anything else (2/127) means rg is
/// missing or errored, so try grep. If grep is missing too, bail to the generic
/// SFTP read walk.
async fn content_exec_ssh(
    ssh: &SshSession,
    root: &str,
    query: &SearchQuery,
    compiled: &Compiled,
) -> Result<Vec<SearchHit>> {
    if let Ok(out) = ssh
        .exec_bounded(&rg_command(root, query), MAX_EXEC_BYTES, EXEC_TIMEOUT, None)
        .await
    {
        if !out.truncated && matches!(out.exit_code, Some(0) | Some(1)) {
            return Ok(parse_rg_json(root, &out.stdout, query.max_results));
        }
    }
    let out = ssh
        .exec_bounded(
            &grep_command(root, query),
            MAX_EXEC_BYTES,
            EXEC_TIMEOUT,
            None,
        )
        .await
        .context("run grep over SSH")?;
    if out.truncated {
        bail!("grep output exceeded the cap");
    }
    match out.exit_code {
        Some(0) | Some(1) => Ok(parse_grep(root, &out.stdout, compiled, query.max_results)),
        code => bail!("grep unavailable (exit {code:?})"),
    }
}

/// Ghost FTP Agent content: same rg→grep ladder, gated by the daemon's exec policy.
async fn content_exec_agent(
    agent: &AgentSession,
    root: &str,
    query: &SearchQuery,
    compiled: &Compiled,
) -> Result<Vec<SearchHit>> {
    if let Ok(out) = agent
        .exec(&rg_command(root, query), MAX_EXEC_BYTES, EXEC_TIMEOUT_MS)
        .await
    {
        if !out.truncated && !out.timed_out && matches!(out.exit_code, Some(0) | Some(1)) {
            return Ok(parse_rg_json(root, &out.stdout, query.max_results));
        }
    }
    let out = agent
        .exec(&grep_command(root, query), MAX_EXEC_BYTES, EXEC_TIMEOUT_MS)
        .await?;
    if out.truncated || out.timed_out {
        bail!("grep output too large or timed out");
    }
    match out.exit_code {
        Some(0) | Some(1) => Ok(parse_grep(root, &out.stdout, compiled, query.max_results)),
        code => bail!("grep unavailable (exit {code:?})"),
    }
}

/// SSH name search via `find -iname`. Accepts a non-empty result even on a
/// non-zero exit (find returns non-zero when *some* subdir was unreadable yet
/// still lists the rest); only a truly empty non-zero run bails to the walk.
async fn name_exec_ssh(
    ssh: &SshSession,
    root: &str,
    query: &SearchQuery,
    compiled: &Compiled,
) -> Result<Vec<SearchHit>> {
    let out = ssh
        .exec_bounded(
            &find_name_command(root, query),
            MAX_EXEC_BYTES,
            EXEC_TIMEOUT,
            None,
        )
        .await
        .context("run find over SSH")?;
    if out.truncated {
        bail!("find output exceeded the cap");
    }
    let hits = parse_find_names(root, &out.stdout, compiled, query.max_results);
    if hits.is_empty() && out.exit_code != Some(0) {
        bail!("find failed (exit {:?})", out.exit_code);
    }
    Ok(hits)
}

/// Ghost FTP Agent name search via `find -iname`, gated by the daemon's exec policy.
async fn name_exec_agent(
    agent: &AgentSession,
    root: &str,
    query: &SearchQuery,
    compiled: &Compiled,
) -> Result<Vec<SearchHit>> {
    let out = agent
        .exec(
            &find_name_command(root, query),
            MAX_EXEC_BYTES,
            EXEC_TIMEOUT_MS,
        )
        .await?;
    if out.truncated || out.timed_out {
        bail!("find output too large or timed out");
    }
    let hits = parse_find_names(root, &out.stdout, compiled, query.max_results);
    if hits.is_empty() && out.exit_code != Some(0) {
        bail!("find failed (exit {:?})", out.exit_code);
    }
    Ok(hits)
}

// ---------- Capped byte reads for the content walk ----------

/// Read up to `max` bytes of a file for grepping. `session == None` is the local
/// filesystem. Backends without a byte-read path here (WebDAV/HTTP/cloud/agent in
/// Phase 1) return an error the caller treats as "skip this file". Mirrors the
/// per-session dispatch `diff.rs` uses for `--hash`.
async fn read_capped(session: Option<&Session>, path: &str, max: u64) -> Result<Vec<u8>> {
    match session {
        None => read_local(path, max).await,
        Some(Session::Ssh(ssh)) => read_ssh(ssh, path, max).await,
        Some(Session::Object(obj)) => read_object(obj, path, max).await,
        Some(Session::Ftp(ftp)) => read_ftp(ftp, path, max).await,
        Some(other) => bail!("content read not supported for {} yet", other.protocol()),
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

async fn read_ssh(ssh: &SshSession, path: &str, max: u64) -> Result<Vec<u8>> {
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

async fn read_object(obj: &ObjectSession, path: &str, max: u64) -> Result<Vec<u8>> {
    let key = path.trim_start_matches('/');
    let p = object_store::path::Path::from(key);
    let get = obj
        .store
        .get(&p)
        .await
        .with_context(|| format!("get {key}"))?;
    let mut stream = get.into_stream();
    let mut buf = Vec::new();
    while let Some(chunk) = stream.next().await {
        buf.extend_from_slice(&chunk?);
        if buf.len() as u64 >= max {
            break;
        }
    }
    Ok(buf)
}

async fn read_ftp(ftp: &FtpSession, path: &str, max: u64) -> Result<Vec<u8>> {
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
    let path = path.to_string();
    let max = max as usize;
    ftp.with_stream(move |stream| {
        let mut sink = CapSink {
            buf: Vec::new(),
            max,
        };
        stream.retr_to_writer(&path, &mut sink)?;
        Ok(sink.buf)
    })
    .await
}

// ---------- GUI search runner (Plan 7 Phase 3) ----------
//
// The GUI search panel drives its own cancellable run — streaming hits in
// batches + live progress, stop mid-search — and reuses the pure engine above.
// Shaped exactly like `diff::DiffManager` / `diskscan::ScanManager`: an
// `Arc<SearchManager>` in `AppState`, one run per `Mutex<HashMap<id, …>>`,
// progress over `search://…` events, cancel via a shared [`CancelToken`], hits
// streamed as `search://hit` batches, the full list fetched once on completion.

use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::{Arc, Mutex as StdMutex};
use std::time::{Instant, SystemTime, UNIX_EPOCH};

use tauri::{AppHandle, Emitter, State};
use tokio::sync::Mutex;
use uuid::Uuid;

use crate::AppState;

/// Hits per streamed `search://hit` batch (also flushed on a ~120 ms timer).
const HIT_BATCH: usize = 50;

#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum SearchRunState {
    Searching,
    Done,
    Error,
    Canceled,
}

/// A snapshot of a search for the frontend: live counts while `Searching`, the
/// full hit list once `Done` (fetched via `search_result`).
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchSnapshot {
    pub id: String,
    pub session_id: String,
    pub root: String,
    pub kind: SearchKind,
    pub state: SearchRunState,
    pub strategy: SearchStrategy,
    pub files_scanned: usize,
    pub hit_count: usize,
    pub truncated: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub note: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hits: Option<Vec<SearchHit>>,
    pub started_at: i64,
}

/// The lightweight body streamed over `search://progress`.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct SearchProgressEvent {
    id: String,
    strategy: SearchStrategy,
    files_scanned: usize,
    hit_count: usize,
}

/// A batch of newly-found hits streamed over `search://hit`.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct SearchHitEvent {
    id: String,
    hits: Vec<SearchHit>,
}

enum SearchOutcome {
    Running,
    Done,
    Error(String),
    Canceled,
}

struct SearchRun {
    id: String,
    session_id: String,
    root: String,
    kind: SearchKind,
    strategy: StdMutex<SearchStrategy>,
    note: StdMutex<Option<String>>,
    files_scanned: AtomicUsize,
    hit_count: AtomicUsize,
    truncated: AtomicBool,
    cancel: CancelToken,
    started_at: i64,
    outcome: StdMutex<SearchOutcome>,
    /// The full accumulated hit list, cloned into a snapshot on completion.
    hits: StdMutex<Vec<SearchHit>>,
}

impl SearchRun {
    fn snapshot(&self, with_hits: bool) -> SearchSnapshot {
        let (state, error) = match &*self.outcome.lock().unwrap() {
            SearchOutcome::Running => (SearchRunState::Searching, None),
            SearchOutcome::Done => (SearchRunState::Done, None),
            SearchOutcome::Error(e) => (SearchRunState::Error, Some(e.clone())),
            SearchOutcome::Canceled => (SearchRunState::Canceled, None),
        };
        let hits =
            (with_hits && state == SearchRunState::Done).then(|| self.hits.lock().unwrap().clone());
        SearchSnapshot {
            id: self.id.clone(),
            session_id: self.session_id.clone(),
            root: self.root.clone(),
            kind: self.kind,
            state,
            strategy: *self.strategy.lock().unwrap(),
            files_scanned: self.files_scanned.load(Ordering::Relaxed),
            hit_count: self.hit_count.load(Ordering::Relaxed),
            truncated: self.truncated.load(Ordering::Relaxed),
            note: self.note.lock().unwrap().clone(),
            error,
            hits,
            started_at: self.started_at,
        }
    }
}

struct SearchHandle {
    info: Arc<SearchRun>,
    task: tauri::async_runtime::JoinHandle<()>,
}

pub struct SearchManager {
    runs: Mutex<HashMap<String, SearchHandle>>,
}

impl Default for SearchManager {
    fn default() -> Self {
        Self::new()
    }
}

fn now_ts() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

impl SearchManager {
    pub fn new() -> Self {
        Self {
            runs: Mutex::new(HashMap::new()),
        }
    }

    /// Kick off a search of `root` on `fs`. `session` (absent for the local FS)
    /// unlocks the exec fast paths. Returns the run id immediately; the work
    /// streams `search://progress` + `search://hit` and settles on one of
    /// `search://done` / `search://error` / `search://canceled`.
    pub async fn start(
        self: &Arc<Self>,
        session_id: String,
        root: String,
        query: SearchQuery,
        fs: Box<dyn RemoteFs>,
        session: Option<Arc<Session>>,
        app: AppHandle,
    ) -> String {
        let id = Uuid::new_v4().to_string();
        let info = Arc::new(SearchRun {
            id: id.clone(),
            session_id,
            root,
            kind: query.kind,
            strategy: StdMutex::new(SearchStrategy::Generic),
            note: StdMutex::new(None),
            files_scanned: AtomicUsize::new(0),
            hit_count: AtomicUsize::new(0),
            truncated: AtomicBool::new(false),
            cancel: CancelToken::new(),
            started_at: now_ts(),
            outcome: StdMutex::new(SearchOutcome::Running),
            hits: StdMutex::new(Vec::new()),
        });

        let task_info = Arc::clone(&info);
        let task = tauri::async_runtime::spawn(async move {
            run_search_task(task_info, query, fs, session, app).await;
        });

        self.runs
            .lock()
            .await
            .insert(id.clone(), SearchHandle { info, task });
        id
    }

    pub async fn snapshot(&self, id: &str, with_hits: bool) -> Option<SearchSnapshot> {
        self.runs
            .lock()
            .await
            .get(id)
            .map(|h| h.info.snapshot(with_hits))
    }

    pub async fn cancel(&self, id: &str) {
        if let Some(h) = self.runs.lock().await.get(id) {
            h.info.cancel.cancel();
        }
    }

    /// Drop a finished (or abandoned) run — the panel was closed.
    pub async fn forget(&self, id: &str) {
        if let Some(h) = self.runs.lock().await.remove(id) {
            h.info.cancel.cancel();
            h.task.abort();
        }
    }
}

fn emit_search_progress(info: &SearchRun, app: &AppHandle) {
    let _ = app.emit(
        "search://progress",
        SearchProgressEvent {
            id: info.id.clone(),
            strategy: *info.strategy.lock().unwrap(),
            files_scanned: info.files_scanned.load(Ordering::Relaxed),
            hit_count: info.hit_count.load(Ordering::Relaxed),
        },
    );
}

async fn run_search_task(
    info: Arc<SearchRun>,
    query: SearchQuery,
    fs: Box<dyn RemoteFs>,
    session: Option<Arc<Session>>,
    app: AppHandle,
) {
    // Stream hits in throttled batches while accumulating the full list on `info`.
    let pending: Arc<StdMutex<Vec<SearchHit>>> = Arc::new(StdMutex::new(Vec::new()));
    let last_emit = Arc::new(StdMutex::new(Instant::now()));

    let result = {
        let mut on_hit = {
            let info = Arc::clone(&info);
            let pending = Arc::clone(&pending);
            let last = Arc::clone(&last_emit);
            let app = app.clone();
            let id = info.id.clone();
            move |h: SearchHit| {
                info.hits.lock().unwrap().push(h.clone());
                info.hit_count.fetch_add(1, Ordering::Relaxed);
                let mut p = pending.lock().unwrap();
                p.push(h);
                let due = p.len() >= HIT_BATCH
                    || last.lock().unwrap().elapsed() >= Duration::from_millis(120);
                if due {
                    let batch = std::mem::take(&mut *p);
                    *last.lock().unwrap() = Instant::now();
                    drop(p);
                    let _ = app.emit(
                        "search://hit",
                        SearchHitEvent {
                            id: id.clone(),
                            hits: batch,
                        },
                    );
                    emit_search_progress(&info, &app);
                }
            }
        };
        let mut sink = HitSink::new(query.max_results, &mut on_hit);
        run_search(
            fs.as_ref(),
            session.as_deref(),
            &info.root,
            &query,
            &info.cancel,
            &mut sink,
        )
        .await
    };

    // Flush any hits left in the last (sub-threshold) batch.
    let leftover = std::mem::take(&mut *pending.lock().unwrap());
    if !leftover.is_empty() {
        let _ = app.emit(
            "search://hit",
            SearchHitEvent {
                id: info.id.clone(),
                hits: leftover,
            },
        );
    }

    // Cancellation wins even if a strategy returned a partial Ok.
    if info.cancel.is_cancelled() {
        *info.outcome.lock().unwrap() = SearchOutcome::Canceled;
        let _ = app.emit("search://canceled", info.snapshot(false));
        return;
    }

    match result {
        Ok(s) => {
            *info.strategy.lock().unwrap() = s.strategy;
            *info.note.lock().unwrap() = s.note;
            info.truncated.store(s.truncated, Ordering::Relaxed);
            info.files_scanned.store(s.scanned, Ordering::Relaxed);
            *info.outcome.lock().unwrap() = SearchOutcome::Done;
            let _ = app.emit("search://done", info.snapshot(false));
        }
        Err(e) => {
            *info.outcome.lock().unwrap() = SearchOutcome::Error(format!("{e:#}"));
            let _ = app.emit("search://error", info.snapshot(false));
        }
    }
}

// ---------- Tauri commands ----------

/// Start a search of `path` on `session_id`. Returns the run id; the frontend
/// then listens for `search://…` and fetches the hits on `done`.
#[tauri::command]
pub async fn search_start(
    session_id: String,
    path: String,
    query: SearchQuery,
    app: AppHandle,
    state: State<'_, AppState>,
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
    let fs = crate::commands::fs_for_public(&session_id, &state).await?;
    let mgr = Arc::clone(&state.search);
    Ok(mgr.start(session_id, path, query, fs, session, app).await)
}

#[tauri::command]
pub async fn search_status(
    search_id: String,
    state: State<'_, AppState>,
) -> Result<SearchSnapshot, String> {
    state
        .search
        .snapshot(&search_id, false)
        .await
        .ok_or_else(|| format!("search {search_id} not found"))
}

/// Full snapshot including the `hits` (present once the search is done).
#[tauri::command]
pub async fn search_result(
    search_id: String,
    state: State<'_, AppState>,
) -> Result<SearchSnapshot, String> {
    state
        .search
        .snapshot(&search_id, true)
        .await
        .ok_or_else(|| format!("search {search_id} not found"))
}

#[tauri::command]
pub async fn search_cancel(search_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state.search.cancel(&search_id).await;
    Ok(())
}

#[tauri::command]
pub async fn search_forget(search_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state.search.forget(&search_id).await;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn q(pattern: &str, kind: SearchKind) -> SearchQuery {
        SearchQuery {
            pattern: pattern.to_string(),
            kind,
            ..Default::default()
        }
    }

    #[test]
    fn name_substring_is_case_insensitive_by_default() {
        let m = NameMatcher::compile(&q("log", SearchKind::Name)).unwrap();
        assert!(m.matches("Error.LOG"));
        assert!(m.matches("catalog.txt"));
        assert!(!m.matches("data.bin"));
    }

    #[test]
    fn name_glob_matches_extension() {
        let m = NameMatcher::compile(&q("*.log", SearchKind::Name)).unwrap();
        assert!(m.matches("nginx.log"));
        assert!(m.matches("NGINX.LOG")); // case-insensitive
        assert!(!m.matches("nginx.log.1"));
        let single = NameMatcher::compile(&q("err?r.txt", SearchKind::Name)).unwrap();
        assert!(single.matches("error.txt"));
        assert!(!single.matches("erroor.txt"));
    }

    #[test]
    fn name_case_sensitive_honoured() {
        let mut query = q("Log", SearchKind::Name);
        query.case_sensitive = true;
        let m = NameMatcher::compile(&query).unwrap();
        assert!(m.matches("Login"));
        assert!(!m.matches("catalog"));
    }

    #[test]
    fn content_literal_finds_column() {
        let m = ContentMatcher::compile(&q("panic", SearchKind::Content)).unwrap();
        assert_eq!(m.find("thread 'main' panicked"), Some(14));
        assert_eq!(m.find("all good here"), None);
        // Case-insensitive by default.
        assert_eq!(m.find("PANIC at the disco"), Some(0));
    }

    #[test]
    fn content_regex_compiles_and_matches() {
        let mut query = q(r"TODO|FIXME", SearchKind::Content);
        query.regex = true;
        let m = ContentMatcher::compile(&query).unwrap();
        assert!(m.find("// FIXME: later").is_some());
        assert!(m.find("nothing here").is_none());
    }

    #[test]
    fn bad_regex_is_an_error() {
        let mut query = q(r"(unclosed", SearchKind::Content);
        query.regex = true;
        assert!(ContentMatcher::compile(&query).is_err());
    }

    #[test]
    fn filters_include_and_exclude() {
        let mut query = q("x", SearchKind::Content);
        query.include_globs = vec!["*.rs".into()];
        query.exclude_globs = vec!["*_test.rs".into()];
        let f = Filters::compile(&query).unwrap();
        assert!(f.passes("main.rs"));
        assert!(!f.passes("main_test.rs")); // excluded
        assert!(!f.passes("readme.md")); // not included
        assert!(f.active());
    }

    #[test]
    fn empty_pattern_rejected() {
        assert!(Compiled::new(&q("", SearchKind::Name)).is_err());
    }

    #[test]
    fn preview_trims_and_caps() {
        assert_eq!(preview("   hello world  "), "hello world");
        let long = "a".repeat(400);
        let p = preview(&long);
        assert_eq!(p.chars().count(), 301); // 300 + ellipsis
        assert!(p.ends_with('…'));
    }

    #[test]
    fn decode_text_rejects_binary() {
        assert!(decode_text(b"plain text\nsecond line").is_some());
        assert!(decode_text(&[0x00, 0x01, 0x02]).is_none());
    }

    #[test]
    fn grep_text_reports_every_matching_line() {
        let m = ContentMatcher::compile(&q("err", SearchKind::Content)).unwrap();
        let text = "line one\nERR here\nclean\nanother err\n";
        let mut hits: Vec<SearchHit> = Vec::new();
        {
            let mut push = |h: SearchHit| hits.push(h);
            let mut sink = HitSink::new(100, &mut push);
            grep_text(text, &m, "/a/f.txt", "f.txt", 30, &mut sink);
        }
        assert_eq!(hits.len(), 2);
        assert_eq!(hits[0].line, Some(2));
        assert_eq!(hits[0].preview.as_deref(), Some("ERR here"));
        assert_eq!(hits[1].line, Some(4));
    }

    #[test]
    fn hit_sink_caps_and_flags_truncation() {
        let mut hits: Vec<SearchHit> = Vec::new();
        let truncated;
        {
            let mut push = |h: SearchHit| hits.push(h);
            let mut sink = HitSink::new(2, &mut push);
            assert!(sink.push(SearchHit::name("/a".into(), "a".into(), false, 1))); // wants more
            assert!(!sink.push(SearchHit::name("/b".into(), "b".into(), false, 1))); // hit cap
            assert!(!sink.push(SearchHit::name("/c".into(), "c".into(), false, 1))); // rejected
            truncated = sink.truncated();
        }
        assert_eq!(hits.len(), 2);
        assert!(truncated);
    }

    // End-to-end name search over a real temp tree via LocalFs — the runtime
    // observation the generic strategy leans on (every backend falls back to it).
    #[tokio::test]
    async fn name_search_over_a_real_directory() {
        use crate::remotefs::local::LocalFs;
        let base =
            std::env::temp_dir().join(format!("ghostftp_search_name_{}", std::process::id()));
        let sub = base.join("logs");
        let _ = std::fs::remove_dir_all(&base);
        std::fs::create_dir_all(&sub).unwrap();
        std::fs::write(base.join("readme.md"), "x").unwrap();
        std::fs::write(sub.join("nginx.log"), "x").unwrap();
        std::fs::write(sub.join("access.log"), "x").unwrap();

        let root = base.to_string_lossy().to_string();
        let result = search(&LocalFs, None, &root, &q("*.log", SearchKind::Name))
            .await
            .unwrap();
        let mut rels: Vec<String> = result.hits.iter().map(|h| h.relative.clone()).collect();
        rels.sort();
        assert_eq!(rels, vec!["logs/access.log", "logs/nginx.log"]);
        assert!(result.hits.iter().all(|h| !h.is_dir));

        std::fs::remove_dir_all(&base).ok();
    }

    // End-to-end content grep over a real temp tree: finds the matching line and
    // its number, and the size cap skips an over-large file.
    #[tokio::test]
    async fn content_search_over_a_real_directory() {
        use crate::remotefs::local::LocalFs;
        let base =
            std::env::temp_dir().join(format!("ghostftp_search_content_{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&base);
        std::fs::create_dir_all(&base).unwrap();
        std::fs::write(base.join("a.txt"), "hello\nneedle here\nbye\n").unwrap();
        std::fs::write(base.join("b.txt"), "nothing to see").unwrap();

        let root = base.to_string_lossy().to_string();
        let result = search(&LocalFs, None, &root, &q("needle", SearchKind::Content))
            .await
            .unwrap();
        assert_eq!(result.hits.len(), 1);
        assert_eq!(result.hits[0].relative, "a.txt");
        assert_eq!(result.hits[0].line, Some(2));
        assert!(result.stats.scanned >= 1);

        std::fs::remove_dir_all(&base).ok();
    }

    // The exact engine path the GUI SearchManager drives: run_search streaming
    // into a HitSink, plus cooperative cancel. (search() is the collecting
    // wrapper; the manager calls run_search directly with a streaming closure.)
    #[tokio::test]
    async fn run_search_streams_and_honors_cancel() {
        use crate::remotefs::local::LocalFs;
        let base =
            std::env::temp_dir().join(format!("ghostftp_search_stream_{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&base);
        std::fs::create_dir_all(&base).unwrap();
        for i in 0..5 {
            std::fs::write(base.join(format!("f{i}.log")), "x").unwrap();
        }
        let root = base.to_string_lossy().to_string();

        // Streaming: run_search pushes each hit into the sink as it's found.
        let cancel = CancelToken::new();
        let mut streamed: Vec<String> = Vec::new();
        let stats = {
            let mut push = |h: SearchHit| streamed.push(h.relative.clone());
            let mut sink = HitSink::new(1000, &mut push);
            run_search(
                &LocalFs,
                None,
                &root,
                &q("*.log", SearchKind::Name),
                &cancel,
                &mut sink,
            )
            .await
            .unwrap()
        };
        assert_eq!(streamed.len(), 5);
        assert_eq!(stats.strategy, SearchStrategy::Generic);
        assert!(!stats.truncated);

        // A pre-cancelled token stops the walk before it yields anything.
        let cancelled = CancelToken::new();
        cancelled.cancel();
        let mut none: Vec<SearchHit> = Vec::new();
        {
            let mut push = |h: SearchHit| none.push(h);
            let mut sink = HitSink::new(1000, &mut push);
            let _ = run_search(
                &LocalFs,
                None,
                &root,
                &q("*.log", SearchKind::Name),
                &cancelled,
                &mut sink,
            )
            .await;
        }
        assert!(none.is_empty());

        std::fs::remove_dir_all(&base).ok();
    }

    // ---- exec fast-path parsers (Phase 2) ----

    #[test]
    fn parse_find_output_keeps_files_and_dirs_skips_symlinks() {
        let compiled = Compiled::new(&q("x", SearchKind::Name)).unwrap();
        let out = "f\t120\t/var/log/nginx.log\r\nd\t4096\t/var/log/nginx\nl\t0\t/var/log/sym\n";
        let hits = parse_find_names("/var/log", out, &compiled, 100);
        assert_eq!(hits.len(), 2); // symlink skipped
        assert_eq!(hits[0].relative, "nginx.log");
        assert!(!hits[0].is_dir);
        assert_eq!(hits[0].size, 120);
        assert_eq!(hits[1].relative, "nginx");
        assert!(hits[1].is_dir);
    }

    #[test]
    fn parse_find_output_applies_exclude_to_files() {
        let mut query = q("x", SearchKind::Name);
        query.exclude_globs = vec!["*.tmp".into()];
        let compiled = Compiled::new(&query).unwrap();
        let out = "f\t1\t/r/keep.log\nf\t2\t/r/skip.tmp\n";
        let hits = parse_find_names("/r", out, &compiled, 100);
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].relative, "keep.log");
    }

    #[test]
    fn parse_rg_json_extracts_match_records() {
        let out = concat!(
            r#"{"type":"begin","data":{"path":{"text":"/srv/app/main.rs"}}}"#,
            "\n",
            r#"{"type":"match","data":{"path":{"text":"/srv/app/main.rs"},"lines":{"text":"    panic!(\"boom\");\n"},"line_number":42,"submatches":[{"match":{"text":"panic"},"start":4,"end":9}]}}"#,
            "\n",
            r#"{"type":"end","data":{"path":{"text":"/srv/app/main.rs"}}}"#,
            "\n",
        );
        let hits = parse_rg_json("/srv/app", out, 100);
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].relative, "main.rs");
        assert_eq!(hits[0].line, Some(42));
        assert_eq!(hits[0].column, Some(4));
        assert_eq!(hits[0].preview.as_deref(), Some("panic!(\"boom\");"));
    }

    #[test]
    fn parse_grep_output_recomputes_column_and_guards_colons() {
        let compiled = Compiled::new(&q("panic", SearchKind::Content)).unwrap();
        // Second line's "field 2" isn't a number → skipped (a path with a colon).
        let out = "/srv/app/main.rs:42:    panic!(\"boom\");\n/srv/app/notes.txt:oops:foo\n";
        let hits = parse_grep("/srv/app", out, &compiled, 100);
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].relative, "main.rs");
        assert_eq!(hits[0].line, Some(42));
        assert_eq!(hits[0].column, Some(4)); // "panic" at col 4
    }

    #[test]
    fn command_builders_quote_and_carry_flags() {
        let mut query = q("pan ic", SearchKind::Content);
        query.include_globs = vec!["*.rs".into()];
        query.exclude_globs = vec!["*.min.js".into()];
        let rg = rg_command("/a b", &query);
        assert!(rg.contains("--json"));
        assert!(rg.contains(" -F")); // literal
        assert!(rg.contains(" -i")); // case-insensitive default
        assert!(rg.contains("-g '*.rs'"));
        assert!(rg.contains("-g '!*.min.js'"));
        assert!(rg.contains("-e 'pan ic'"));
        assert!(rg.contains("-- '/a b'"));

        let grep = grep_command("/a b", &query);
        assert!(grep.contains("grep -rnIiF"));
        assert!(grep.contains("--include='*.rs'"));
        assert!(grep.contains("--exclude='*.min.js'"));

        // Name search: a plain substring becomes *substr*; a glob is used as-is.
        assert_eq!(find_name_glob(&q("log", SearchKind::Name)), "*log*");
        assert_eq!(find_name_glob(&q("*.log", SearchKind::Name)), "*.log");
        let find = find_name_command("/var/log", &q("app", SearchKind::Name));
        assert!(find.contains("-iname '*app*'"));
        assert!(find.contains(r"-type f -o -type d"));
    }
}
