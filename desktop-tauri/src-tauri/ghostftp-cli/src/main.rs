// Ghost FTP command-line client. Reuses ghostftp_lib's RemoteFs / Session layers
// (the same code the GUI runs against) so any backend the GUI supports —
// SFTP, FTP, FTPS, S3, R2, B2, Azure Blob — works here too.
//
// Path syntax:
//   ./local/path             local
//   /etc/hosts               local (absolute)
//   C:\Users\me              local (Windows)
//   prod:/var/log            remote, using saved profile "prod"
//
// Saved profiles are read from the same on-disk store the GUI uses, so
// connections you set up in the app are immediately scriptable.

use anyhow::{anyhow, bail, Context, Result};
use async_trait::async_trait;
use clap::{Parser, Subcommand};
use ghostftp_lib::profiles::{ConnectionProfile, ProfileStore};
use ghostftp_lib::remotefs::{DirEntry, FileKind, RemoteFs};
use ghostftp_lib::session::{
    open_session as ghostftp_open_session, FtpSession, HostDecision, HostKeyVerifier,
    HostPromptKind, ObjectSession, Session,
};
use futures::StreamExt;
use indicatif::{ProgressBar, ProgressStyle};
use std::io::{self, Write};
use std::path::PathBuf;
use std::sync::Arc;

#[derive(Parser)]
#[command(
    name = "ghostftp-cli",
    about = "Ghost FTP CLI — SFTP / FTP / S3 / R2 / B2 / Azure Blob from the terminal",
    version
)]
struct Cli {
    #[command(subcommand)]
    command: Cmd,
}

#[derive(Subcommand)]
enum Cmd {
    /// List a directory.
    Ls {
        /// Path. Use `profile:/path` for a saved profile.
        target: String,
        /// Show file sizes in bytes (default: human-readable).
        #[arg(long)]
        bytes: bool,
    },

    /// Run a shell command on a saved SSH profile's server.
    ///
    /// Connects with the profile's stored credentials and runs the command
    /// non-interactively, the same as typing it into that server's shell.
    /// stdout/stderr are passed through and the command's exit code becomes
    /// this process's exit code.
    Exec {
        /// Saved profile name or id (must be SSH/SFTP).
        profile: String,
        /// The command to run. Quote it, or pass it as trailing arguments.
        #[arg(trailing_var_arg = true, required = true)]
        command: Vec<String>,
    },

    /// Copy a file from one location to another.
    Cp {
        source: String,
        destination: String,
    },

    /// Rename / move a file (must be within a single backend).
    Mv { source: String, destination: String },

    /// Remove a file. Use --recursive for directories.
    Rm {
        target: String,
        #[arg(short, long)]
        recursive: bool,
    },

    /// Create a directory.
    Mkdir { target: String },

    /// Mirror or copy a directory tree between local and remote.
    Sync {
        /// Local directory.
        local: String,
        /// Remote directory (must be `profile:/path`).
        remote: String,
        /// Direction. Defaults to push (local to remote).
        #[arg(long, value_enum, default_value = "push")]
        direction: Dir,
        /// Mirror mode also deletes destination files that don't exist on the
        /// source. Off by default — the planner is additive.
        #[arg(long)]
        mirror: bool,
        /// Show the plan without executing.
        #[arg(long)]
        dry_run: bool,
    },

    /// Compare two directory trees and report what differs.
    ///
    /// Each side is `profile:/path` (a saved profile), `local:/path`, or a plain
    /// local path — so **remote↔remote** works too (staging vs prod, two servers,
    /// two buckets), which no local diff tool can do. By default files are
    /// compared by size; `--hash` confirms same-size files by content (sha256).
    /// Exits 0 when the trees are identical, 1 when they differ.
    Diff {
        /// Side A: `profile:/path`, `local:/path`, or a local path.
        a: String,
        /// Side B: `profile:/path`, `local:/path`, or a local path.
        b: String,
        /// Confirm same-size files by content hash (sha256). Opt-in — it reads
        /// every same-size file (server-side over SSH where possible).
        #[arg(long)]
        hash: bool,
        /// Emit the full result as JSON (every entry, including unchanged).
        #[arg(long)]
        json: bool,
        /// Also list unchanged files in the human output (JSON always includes them).
        #[arg(long)]
        all: bool,
    },

    /// Find duplicate files in one tree — the `name_1.ext` copies Ghost FTP's
    /// rename-on-conflict creates, or (with --hash) exact content duplicates.
    ///
    /// The target is `profile:/path`, `local:/path`, or a plain local path.
    /// Read-only unless --delete is passed (keeps the unsuffixed/oldest file
    /// per group, deletes the rest — requires --yes).
    Dedupe {
        /// Target: `profile:/path`, `local:/path`, or a local path.
        target: String,
        /// Group by content hash (sha256) instead of name — catches duplicates
        /// with unrelated names, anywhere in the tree. Slower: reads files.
        #[arg(long)]
        hash: bool,
        /// Emit the raw JSON result.
        #[arg(long)]
        json: bool,
        /// Delete the duplicates (every non-keeper in each group).
        #[arg(long)]
        delete: bool,
        /// Confirm the deletion (required together with --delete).
        #[arg(long)]
        yes: bool,
    },

    /// Search a directory tree by file name or by content (grep).
    ///
    /// The target is `profile:/path` (a saved profile) or a local path. Name
    /// search matches each entry's name (a `*`/`?` makes it a glob, else a
    /// substring); `--content` greps file contents (literal, or `--regex`).
    /// On SSH / Ghost FTP Agent servers content search runs `rg`/`grep` server-side;
    /// object stores name-match a flat key listing. Exits 0 when something
    /// matched, 1 when nothing did.
    Search {
        /// Target: `profile:/path` or a local path.
        target: String,
        /// What to look for (a name glob/substring, or a content pattern).
        pattern: String,
        /// Grep file contents instead of matching names.
        #[arg(long)]
        content: bool,
        /// Treat the content pattern as a regular expression (implies --content).
        #[arg(long)]
        regex: bool,
        /// Case-sensitive matching (default: insensitive).
        #[arg(long)]
        case_sensitive: bool,
        /// Only consider files whose name matches this glob (repeatable).
        #[arg(long)]
        include: Vec<String>,
        /// Skip files whose name matches this glob (repeatable).
        #[arg(long)]
        exclude: Vec<String>,
        /// Allow content search to DOWNLOAD every file on backends with no
        /// server-side grep (object stores, FTP, WebDAV, cloud). Off by default.
        #[arg(long)]
        content_remote: bool,
        /// Cap on returned hits (default 1000).
        #[arg(long)]
        max: Option<usize>,
        /// Emit the raw JSON result.
        #[arg(long)]
        json: bool,
    },

    /// Manage saved profiles.
    Profiles {
        #[command(subcommand)]
        action: ProfileCmd,
    },

    /// Operate a server through Ghost FTP's running Agent Bridge.
    ///
    /// Unlike the other subcommands (which open their own connection), `agent`
    /// talks to the Bridge you already have open in the Ghost FTP app — so commands
    /// go through Ghost FTP's per-command approval and show up in its live console.
    /// The bridge URL + token are read from Ghost FTP's local discovery file, so you
    /// only pass a server name; you never handle a URL or token.
    Agent {
        #[command(subcommand)]
        action: AgentCmd,
    },

    /// Run saved Skills (fleet automations) through Ghost FTP's Agent Bridge.
    ///
    /// A Skill is a named, parameterized, multi-step workflow that fans shell
    /// commands across one or many connected servers. Like `agent`, this talks
    /// to the Bridge open in the Ghost FTP app, so a run goes through Ghost FTP's approval
    /// and shows up in its live console.
    Skill {
        #[command(subcommand)]
        action: SkillCmd,
    },

    /// Update this ghostftp-cli binary to the latest release (or a specific --tag).
    ///
    /// Downloads the matching release asset from ghostftp.com (the same source the
    /// agent installer uses) and swaps it in place. ghostftp-cli and the Ghost FTP app
    /// ship as separate downloads, so the CLI can lag the app after an app
    /// update — this catches it up. Use --check to only compare versions.
    SelfUpdate {
        /// Update to a specific release tag (e.g. v1.4.0) instead of the latest.
        #[arg(long)]
        tag: Option<String>,
        /// Only report current vs latest version; don't download or replace anything.
        #[arg(long)]
        check: bool,
    },

    /// Fetch a URL behind HTTP Basic Auth using a saved HTTP(S) profile's creds.
    ///
    /// Reuses the stored username/password of a saved HTTP(S) connection (Plan 5
    /// Phase 4's HttpFs) to GET an auth-walled page — e.g. a rendered page on a
    /// staging site — and writes the body to stdout. The profile is matched by the
    /// URL's host; pass --profile to choose explicitly. Never echoes credentials.
    Fetch {
        /// The URL to GET (http/https).
        url: String,
        /// Saved HTTP(S) profile to use (by name/id). Omit to match on the host.
        #[arg(long)]
        profile: Option<String>,
    },

    /// Export or restore an encrypted backup of profiles, secrets, and state.
    ///
    /// The backup is a single password-protected file (Argon2id + AES-256-GCM)
    /// carrying every connection profile, its keychain credentials, app settings,
    /// and sync pairs — enough to move Ghost FTP to a new machine. Provide the password
    /// with --password or the GHOSTFTP_BACKUP_PASSWORD environment variable.
    Backup {
        #[command(subcommand)]
        action: BackupCmd,
    },
}

#[derive(Subcommand)]
enum BackupCmd {
    /// Write an encrypted backup to a file.
    Export {
        /// Destination path (e.g. ghostftp-backup.ghostftpbak).
        path: String,
        /// Encryption password. Falls back to $GHOSTFTP_BACKUP_PASSWORD.
        #[arg(long)]
        password: Option<String>,
    },
    /// Restore an encrypted backup, overwriting local profiles/settings/secrets.
    Import {
        /// Backup file to restore.
        path: String,
        /// Decryption password. Falls back to $GHOSTFTP_BACKUP_PASSWORD.
        #[arg(long)]
        password: Option<String>,
    },
}

#[derive(Subcommand)]
enum ProfileCmd {
    /// List every saved profile.
    List,
    /// Show details for one profile by name or id.
    Show { name: String },
}

#[derive(Subcommand)]
enum SkillCmd {
    /// List the saved Skills (name, status, params, step count, default targets).
    List {
        /// Emit the raw JSON result.
        #[arg(long)]
        json: bool,
    },
    /// Run a Skill across its targets.
    Run {
        /// The Skill's name (as shown by `skill list`).
        name: String,
        /// Server(s) to run on (name or id). Repeatable. Pass `all` for every
        /// exec-capable connection. Omit to use the Skill's default targets.
        #[arg(long)]
        target: Vec<String>,
        /// Parameter value as `key=value`. Repeatable.
        #[arg(long)]
        param: Vec<String>,
        /// Show the resolved commands per target without running anything.
        #[arg(long)]
        dry_run: bool,
        /// Emit the raw JSON result.
        #[arg(long)]
        json: bool,
    },
}

#[derive(Subcommand)]
enum AgentCmd {
    /// Show agent context: bridge state, policy, enabled sessions, saved commands.
    Context,
    /// List the servers the Agent Bridge can currently reach.
    Sessions,
    /// Run a shell command on a server (SSH or Ghost FTP Agent). Exits with its exit code.
    ///
    /// For a multi-line program (heredocs, nested quotes) pass `--file <path>` or
    /// `--stdin` instead of a command: the script bytes are read locally and run
    /// verbatim on the server, so nothing gets re-parsed at a shell boundary. See
    /// also `agent script`.
    Exec {
        /// Saved server name (as shown in Ghost FTP) or its session id.
        server: String,
        /// Show what would run and whether it would need approval, without executing.
        #[arg(long)]
        dry_run: bool,
        /// Timeout in milliseconds (default 60000; clamped to 1000–900000).
        #[arg(long)]
        timeout_ms: Option<u64>,
        /// Read the program to run from this local file, verbatim (no shell
        /// re-parsing). Mutually exclusive with a trailing command / --stdin.
        #[arg(long, conflicts_with = "stdin")]
        file: Option<String>,
        /// Read the program to run from stdin, verbatim.
        #[arg(long)]
        stdin: bool,
        /// Launch as a background job and return a job id immediately (poll with
        /// `agent job`). For multi-minute work that would time out. Works on SSH
        /// servers and paired Ghost FTP Agent machines (agent needs ghostftp-agentd from
        /// Plan 10+). Can't be combined with --file/--stdin yet.
        #[arg(long)]
        detach: bool,
        /// The command to run. Quote it, or pass it as trailing arguments. Omit
        /// when using --file / --stdin.
        #[arg(trailing_var_arg = true)]
        command: Vec<String>,
    },
    /// Run a whole local script file on a server (SSH or Ghost FTP Agent), verbatim.
    ///
    /// Reads the file's bytes locally and ships them as an opaque payload, so
    /// heredocs, nested quotes and newlines survive intact — no base64 or quoting
    /// gymnastics. Shorthand for `agent exec <server> --file <script>`.
    Script {
        /// Saved server name (as shown in Ghost FTP) or its session id.
        server: String,
        /// Local path to the script to run.
        file: String,
        /// Show what would run and whether it would need approval, without executing.
        #[arg(long)]
        dry_run: bool,
        /// Timeout in milliseconds (default 60000; clamped to 1000–900000).
        #[arg(long)]
        timeout_ms: Option<u64>,
    },
    /// Poll a background job started with `agent exec --detach` (SSH server or
    /// paired Ghost FTP Agent).
    Job {
        /// Saved server name (as shown in Ghost FTP) or its session id.
        server: String,
        /// The job id returned by `agent exec --detach`.
        job_id: String,
    },
    /// List background jobs on a server (SSH only).
    Jobs {
        /// Saved server name (as shown in Ghost FTP) or its session id.
        server: String,
    },
    /// List a remote directory.
    Ls {
        server: String,
        /// Remote directory (default: ".").
        path: Option<String>,
    },
    /// Read a remote text file (any protocol, capped at 256 KiB).
    Read { server: String, path: String },
    /// Read several remote text files at once (any protocol, total capped at 1 MiB).
    ReadBatch {
        server: String,
        /// Remote file paths. Pass multiple times or space-separated.
        #[arg(required = true)]
        paths: Vec<String>,
    },
    /// Find files whose name matches a glob pattern, recursively (SSH only).
    Glob {
        server: String,
        /// Root directory to search under (default: ".").
        path: Option<String>,
        /// Glob pattern, e.g. '*.log'.
        pattern: String,
    },
    /// Search a connected server by file name or by content (grep).
    ///
    /// Name search (default) matches entry names (a `*`/`?` makes it a glob).
    /// `--content` greps file contents — server-side via `rg`/`grep` on SSH /
    /// Ghost FTP Agent servers. Read-only; goes through Ghost FTP's approval + console.
    Search {
        server: String,
        /// What to look for (a name glob/substring, or a content pattern).
        query: String,
        /// Root directory to search under (default: ".").
        path: Option<String>,
        /// Grep file contents instead of matching names.
        #[arg(long)]
        content: bool,
        /// Treat the content pattern as a regex (implies --content).
        #[arg(long)]
        regex: bool,
        /// Case-sensitive matching (default: insensitive).
        #[arg(long)]
        case_sensitive: bool,
        /// Only consider files whose name matches this glob (repeatable).
        #[arg(long)]
        include: Vec<String>,
        /// Skip files whose name matches this glob (repeatable).
        #[arg(long)]
        exclude: Vec<String>,
        /// Allow content search to download files on grep-less backends.
        #[arg(long)]
        content_remote: bool,
    },
    /// Write text straight into a remote file — no local staging file, no upload.
    ///
    /// Drops a debug script, a config snippet or a one-file patch directly on the
    /// server (SSH via SFTP, a Ghost FTP Agent via a ranged write), bypassing the
    /// mangling-prone upload path. Provide the content with exactly one of
    /// --from-file / --stdin / --content. Refuses to overwrite unless --overwrite.
    Write {
        /// Saved server name (as shown in Ghost FTP) or its session id.
        server: String,
        /// Absolute remote file path to write.
        remote_path: String,
        /// Read the content from this local file.
        #[arg(long, conflicts_with_all = ["stdin", "content"])]
        from_file: Option<String>,
        /// Read the content from stdin.
        #[arg(long, conflicts_with = "content")]
        stdin: bool,
        /// Inline content to write.
        #[arg(long)]
        content: Option<String>,
        /// Replace the file if it already exists (default: refuse).
        #[arg(long)]
        overwrite: bool,
    },
    /// Download a remote file to a local dir (default: your Downloads).
    Download {
        server: String,
        remote_path: String,
        local_dir: Option<String>,
    },
    /// Upload a local file into a remote directory.
    ///
    /// A name collision is RENAMED by default (uploading `x.php` next to an
    /// existing `x.php` lands as `x_1.php`, leaving the original in place). To
    /// REPLACE the remote file, pass `--overwrite` — or use `agent write` for
    /// text you can pass inline.
    Upload {
        server: String,
        local_path: String,
        remote_dir: String,
        /// Replace an existing remote file of the same name (default: rename).
        #[arg(long)]
        overwrite: bool,
    },
    /// Delete a remote file or directory (any protocol).
    Rm {
        server: String,
        path: String,
        /// Required to delete a directory: removes everything inside it.
        #[arg(long, short)]
        recursive: bool,
    },
    /// Rename or move a remote path, server-side (any protocol that supports it).
    Mv {
        server: String,
        from: String,
        to: String,
    },
    /// Create a remote directory (one level; the parent must exist).
    Mkdir { server: String, path: String },
    /// Upload a whole local directory tree into a remote directory. One
    /// approval covers the tree; the prompt shows file/byte counts.
    UploadDir {
        server: String,
        /// Local directory to upload (recreated INSIDE the remote directory).
        local_dir: String,
        /// Remote destination directory.
        remote_dir: String,
        /// Replace existing remote files (default: rename colliding uploads).
        #[arg(long)]
        overwrite: bool,
    },
    /// One-way sync between a local directory and a remote directory. Use
    /// --dry-run first: it prints the plan without changing anything.
    Sync {
        server: String,
        /// Local directory (source for push, destination for pull).
        local_dir: String,
        /// Remote directory (destination for push, source for pull).
        remote_dir: String,
        /// Direction. Defaults to push (local to remote).
        #[arg(long, value_enum, default_value = "push")]
        direction: Dir,
        /// Mirror mode ALSO DELETES destination files missing from the source.
        /// The approval prompt in Ghost FTP shows the exact delete count.
        #[arg(long)]
        mirror: bool,
        /// Show the plan without executing.
        #[arg(long)]
        dry_run: bool,
    },
    /// Diff two directory trees through the Agent Bridge — any two connected
    /// servers (remote↔remote), or a server vs local. Read-only.
    ///
    /// Each side is `server:/path` (a connected server), `local:/path`, or a
    /// plain path (local). By default files compare by size; `--hash` confirms
    /// same-size files by content.
    Diff {
        /// Side A: `server:/path`, `local:/path`, or a local path.
        a: String,
        /// Side B: `server:/path`, `local:/path`, or a local path.
        b: String,
        /// Confirm same-size files by content hash (sha256).
        #[arg(long)]
        hash: bool,
        /// Emit the raw JSON result.
        #[arg(long)]
        json: bool,
    },
    /// Check a transfer started via `agent download` / `agent upload`.
    Transfer { transfer_id: String },
    /// Tail a remote log file for up to 30 seconds (SSH only).
    Tail {
        server: String,
        /// Remote log file path.
        path: String,
        /// Number of trailing lines to start with (default 50).
        #[arg(long, default_value_t = 50)]
        lines: usize,
    },
    /// Show context about a server (protocol, host, port, …).
    Info { server: String },
    /// Recent Agent Bridge activity (what's already run), newest first.
    History {
        /// Limit to one server (name or session id).
        #[arg(long)]
        server: Option<String>,
        /// Max entries (default 50, max 200).
        #[arg(long, default_value_t = 50)]
        limit: usize,
    },
    /// Run one of your SAVED, pre-approved commands by name (SSH only). These run
    /// with no approval prompt because you wrote and vetted them in Ghost FTP.
    Run {
        /// Saved server name (as shown in Ghost FTP) or its session id.
        server: String,
        /// Show the saved command without running it.
        #[arg(long)]
        dry_run: bool,
        /// The saved command's name (see `agent commands`).
        name: String,
    },
    /// List your saved, pre-approved commands.
    Commands,
}

#[derive(Clone, Copy, clap::ValueEnum)]
enum Dir {
    /// Local → remote.
    Push,
    /// Remote → local.
    Pull,
}

// clap's derive builds the whole subcommand graph (every `Cmd`/`AgentCmd`
// variant) via nested generic calls; in debug builds that recursion now
// overflows the Windows main thread's default 1 MiB stack, so run on a thread
// with headroom.
fn main() {
    std::thread::Builder::new()
        .name("ghostftp-cli-main".into())
        .stack_size(32 * 1024 * 1024)
        .spawn(async_main)
        .expect("spawn ghostftp-cli main thread")
        .join()
        .expect("join ghostftp-cli main thread");
}

#[tokio::main]
async fn async_main() {
    let cli = Cli::parse();
    if let Err(e) = run(cli).await {
        eprintln!("\x1b[31merror:\x1b[0m {e:#}");
        std::process::exit(1);
    }
}

async fn run(cli: Cli) -> Result<()> {
    let store = open_store()?;

    match cli.command {
        Cmd::Ls { target, bytes } => cmd_ls(&store, &target, bytes).await,
        Cmd::Exec { profile, command } => cmd_exec(&store, &profile, &command).await,
        Cmd::Cp { source, destination } => cmd_cp(&store, &source, &destination).await,
        Cmd::Mv { source, destination } => cmd_mv(&store, &source, &destination).await,
        Cmd::Rm { target, recursive } => cmd_rm(&store, &target, recursive).await,
        Cmd::Mkdir { target } => cmd_mkdir(&store, &target).await,
        Cmd::Sync {
            local,
            remote,
            direction,
            mirror,
            dry_run,
        } => cmd_sync(&store, &local, &remote, direction, mirror, dry_run).await,
        Cmd::Diff { a, b, hash, json, all } => cmd_diff(&store, &a, &b, hash, json, all).await,
        Cmd::Dedupe { target, hash, json, delete, yes } => {
            cmd_dedupe(&store, &target, hash, json, delete, yes).await
        }
        Cmd::Search {
            target,
            pattern,
            content,
            regex,
            case_sensitive,
            include,
            exclude,
            content_remote,
            max,
            json,
        } => {
            cmd_search(
                &store, &target, &pattern, content, regex, case_sensitive, include, exclude,
                content_remote, max, json,
            )
            .await
        }
        Cmd::Profiles { action } => match action {
            ProfileCmd::List => cmd_profiles_list(&store).await,
            ProfileCmd::Show { name } => cmd_profiles_show(&store, &name).await,
        },
        // The agent subcommands talk to the running bridge over HTTP (blocking
        // ureq); they need no profile store and don't await.
        Cmd::Agent { action } => cmd_agent(action),
        // Skills likewise run through the bridge.
        Cmd::Skill { action } => cmd_skill(action),
        // Self-update fetches a release asset from ghostftp.com over HTTPS.
        Cmd::SelfUpdate { tag, check } => cmd_self_update(tag, check),
        // Authenticated GET through a saved HTTP profile's creds.
        Cmd::Fetch { url, profile } => cmd_fetch(&store, &url, profile).await,
        // Encrypted backup / restore — sync, no bridge or session needed.
        Cmd::Backup { action } => cmd_backup(action),
    }
}

// ---- Backup / restore --------------------------------------------------

fn resolve_backup_password(flag: Option<String>) -> Result<String> {
    if let Some(p) = flag {
        if !p.is_empty() {
            return Ok(p);
        }
    }
    if let Ok(p) = std::env::var("GHOSTFTP_BACKUP_PASSWORD") {
        if !p.is_empty() {
            return Ok(p);
        }
    }
    anyhow::bail!("provide a password with --password or the GHOSTFTP_BACKUP_PASSWORD env var")
}

fn cmd_backup(action: BackupCmd) -> Result<()> {
    let dir = default_data_dir()?;
    match action {
        BackupCmd::Export { path, password } => {
            let pw = resolve_backup_password(password)?;
            let db = ghostftp_lib::db::Db::open(&dir.join("ghostftp.db"))?;
            let s = ghostftp_lib::backup::export(&dir, &db, &pw, std::path::Path::new(&path))?;
            println!(
                "Exported {} profile(s), {} credential(s), and {} KiB of app state to {path}",
                s.profiles,
                s.credentials,
                s.db_bytes / 1024
            );
            Ok(())
        }
        BackupCmd::Import { path, password } => {
            let pw = resolve_backup_password(password)?;
            // defer = false: no running GUI to coordinate with, apply now. (If the
            // GUI happens to hold ghostftp.db open, the DB swap is staged and applies
            // on its next launch.)
            let s = ghostftp_lib::backup::import(&dir, &pw, std::path::Path::new(&path), false)?;
            println!(
                "Restored {} profile(s) and {} credential(s). Restart Ghost FTP if it's running.",
                s.profiles, s.credentials
            );
            Ok(())
        }
    }
}

// ---- Profile store -----------------------------------------------------

/// Resolve the same on-disk path the Tauri GUI uses for its data dir, so a
/// CLI invocation sees the profiles the user set up in the app. `GHOSTFTP_DATA_DIR`
/// overrides it — handy for scripted migrations and for pointing the CLI at a
/// non-default data dir.
fn default_data_dir() -> Result<PathBuf> {
    if let Ok(dir) = std::env::var("GHOSTFTP_DATA_DIR") {
        if !dir.is_empty() {
            return Ok(PathBuf::from(dir));
        }
    }
    let base = dirs::data_dir().ok_or_else(|| anyhow!("could not resolve a data dir"))?;
    Ok(base.join("com.juandenis.ghostftp"))
}

fn open_store() -> Result<ProfileStore> {
    let dir = default_data_dir()?;
    ProfileStore::from_dir(&dir)
}

// ---- Path parsing ------------------------------------------------------

/// Targets passed on the CLI are either local paths or remote paths prefixed
/// with a saved profile name and a colon. We don't support inline URLs
/// (`sftp://user@host/...`) yet — those need secrets we'd have to prompt for,
/// and the saved-profile path covers the daily-use case.
enum Target {
    Local(String),
    Remote { profile_name: String, path: String },
}

fn parse_target(raw: &str) -> Target {
    // Windows drive letters (`C:\`, `D:\`) look like profiles syntactically.
    // Disambiguate by checking whether the char before the colon is a single
    // letter — drive letters are always `[A-Za-z]:`.
    if raw.len() >= 2 {
        let bytes = raw.as_bytes();
        if bytes[1] == b':' && bytes[0].is_ascii_alphabetic() {
            return Target::Local(raw.to_string());
        }
    }
    match raw.split_once(':') {
        Some((name, path)) if !name.is_empty() && !name.contains('/') && !name.contains('\\') => {
            Target::Remote {
                profile_name: name.to_string(),
                path: path.to_string(),
            }
        }
        _ => Target::Local(raw.to_string()),
    }
}

async fn find_profile(store: &ProfileStore, name: &str) -> Result<ConnectionProfile> {
    let profiles = store.list().await?;
    let lowered = name.to_ascii_lowercase();
    if let Some(p) = profiles
        .iter()
        .find(|p| p.name.to_ascii_lowercase() == lowered || p.id == name)
    {
        return Ok(p.clone());
    }
    bail!(
        "no profile named `{name}`. List available with `ghostftp-cli profiles list`."
    );
}

// ---- Session opening ---------------------------------------------------

/// Build a session for a remote target. Each invocation opens a fresh
/// connection — short-lived CLI scripts don't need a long-lived manager.
async fn open_session(profile: &ConnectionProfile) -> Result<Session> {
    let verifier: Arc<dyn HostKeyVerifier> = Arc::new(CliHostKeyVerifier);
    ghostftp_open_session(profile, verifier).await
}

fn fs_for(session: &Session) -> Box<dyn RemoteFs> {
    match session {
        Session::Ssh(ssh) => Box::new(ghostftp_lib::remotefs::sftp::SftpFs::new(ssh.clone())),
        Session::Ftp(ftp) => Box::new(ghostftp_lib::remotefs::ftp::FtpFs::new(ftp.clone())),
        Session::Object(obj) => {
            Box::new(ghostftp_lib::remotefs::object::ObjectFs::new(obj.clone()))
        }
        Session::Webdav(dav) => Box::new(ghostftp_lib::remotefs::webdav::WebdavFs::new(dav.clone())),
        Session::Http(http) => Box::new(ghostftp_lib::remotefs::http::HttpFs::new(http.clone())),
        Session::Dropbox(dbx) => Box::new(ghostftp_lib::remotefs::dropbox::DropboxFs::new(dbx.clone())),
        Session::OneDrive(od) => Box::new(ghostftp_lib::remotefs::onedrive::OneDriveFs::new(od.clone())),
        Session::GDrive(gd) => Box::new(ghostftp_lib::remotefs::gdrive::GDriveFs::new(gd.clone())),
        Session::Box(bx) => Box::new(ghostftp_lib::remotefs::boxdrive::BoxFs::new(bx.clone())),
        Session::Shopify(sh) => Box::new(ghostftp_lib::remotefs::shopify::ShopifyFs::new(sh.clone())),
        Session::HubSpot(hs) => Box::new(ghostftp_lib::remotefs::hubspot::HubSpotFs::new(hs.clone())),
        Session::Dynamics(dynm) => {
            Box::new(ghostftp_lib::remotefs::dynamics::DynamicsFs::new(dynm.clone()))
        }
        Session::Agent(agent) => Box::new(ghostftp_lib::remotefs::agent::AgentFs::new(agent.clone())),
    }
}

// ---- Host-key verifier for the CLI -------------------------------------

struct CliHostKeyVerifier;

#[async_trait]
impl HostKeyVerifier for CliHostKeyVerifier {
    async fn decide(
        &self,
        host: &str,
        port: u16,
        key_type: &str,
        fingerprint: &str,
        stored_fingerprint: Option<&str>,
        kind: HostPromptKind,
    ) -> Result<HostDecision, russh::Error> {
        match kind {
            HostPromptKind::Unknown => {
                eprintln!("\n{}", warn("Unknown host key:"));
                eprintln!("  host        {host}:{port}");
                eprintln!("  fingerprint {fingerprint} ({key_type})");
            }
            HostPromptKind::Mismatch => {
                eprintln!("\n{}", warn("Host key changed (possible MITM):"));
                eprintln!("  host        {host}:{port}");
                eprintln!("  server      {fingerprint} ({key_type})");
                if let Some(saved) = stored_fingerprint {
                    eprintln!("  saved       {saved}");
                }
            }
        }
        eprint!("Accept this key? [t]rust+save / [a]ccept once / [r]eject> ");
        let _ = io::stderr().flush();
        let mut line = String::new();
        if io::stdin().read_line(&mut line).is_err() {
            return Ok(HostDecision::Reject);
        }
        match line.trim().to_ascii_lowercase().as_str() {
            "t" | "trust" | "y" | "yes" => Ok(HostDecision::Trust),
            "a" | "accept" | "o" | "once" => Ok(HostDecision::Accept),
            _ => Ok(HostDecision::Reject),
        }
    }
}

fn warn(s: &str) -> String {
    format!("\x1b[33m⚠ {s}\x1b[0m")
}

fn dim(s: &str) -> String {
    format!("\x1b[2m{s}\x1b[0m")
}

// ---- Subcommands -------------------------------------------------------

async fn cmd_ls(store: &ProfileStore, raw: &str, bytes: bool) -> Result<()> {
    let entries = match parse_target(raw) {
        Target::Local(p) => ghostftp_lib::remotefs::local::LocalFs.list_dir(&p).await?,
        Target::Remote { profile_name, path } => {
            let profile = find_profile(store, &profile_name).await?;
            let session = open_session(&profile).await?;
            let fs = fs_for(&session);
            fs.list_dir(&path).await?
        }
    };

    let mut entries = entries;
    entries.sort_by(|a, b| match (a.kind, b.kind) {
        (FileKind::Directory, FileKind::Directory) | (_, _) if a.kind == b.kind => {
            a.name.cmp(&b.name)
        }
        (FileKind::Directory, _) => std::cmp::Ordering::Less,
        (_, FileKind::Directory) => std::cmp::Ordering::Greater,
        _ => a.name.cmp(&b.name),
    });

    for e in entries {
        print_entry(&e, bytes);
    }
    Ok(())
}

fn print_entry(e: &DirEntry, bytes: bool) {
    let kind = match e.kind {
        FileKind::Directory => "d",
        FileKind::Symlink => "l",
        FileKind::File => "-",
        FileKind::Other => "?",
    };
    let size = if e.kind == FileKind::Directory {
        String::new()
    } else if bytes {
        format!("{}", e.size)
    } else {
        fmt_bytes(e.size)
    };
    let mode = e
        .mode
        .map(|m| format!("{:o}", m & 0o777))
        .unwrap_or_else(|| "---".into());
    println!(
        "{kind} {mode:>4}  {size:>10}  {name}",
        name = e.name
    );
    let _ = e.modified;
}

async fn cmd_exec(store: &ProfileStore, profile_name: &str, command: &[String]) -> Result<()> {
    let profile = find_profile(store, profile_name).await?;
    let session = open_session(&profile).await?;
    let Session::Ssh(ssh) = session else {
        bail!(
            "`exec` needs an SSH/SFTP profile; `{}` is {}",
            profile.name,
            profile.protocol
        );
    };
    let cmd = command.join(" ");
    let out = ssh
        .exec(&cmd)
        .await
        .with_context(|| format!("exec `{cmd}`"))?;

    let mut so = io::stdout();
    so.write_all(out.stdout.as_bytes()).ok();
    so.flush().ok();
    if !out.stderr.is_empty() {
        let mut se = io::stderr();
        se.write_all(out.stderr.as_bytes()).ok();
        se.flush().ok();
    }
    // Propagate the remote command's exit code as our own.
    std::process::exit(out.exit_code.unwrap_or(0))
}

async fn cmd_cp(store: &ProfileStore, src: &str, dst: &str) -> Result<()> {
    let src_t = parse_target(src);
    let dst_t = parse_target(dst);

    match (src_t, dst_t) {
        // local → local
        (Target::Local(s), Target::Local(d)) => {
            let bytes = tokio::fs::copy(&s, &d)
                .await
                .with_context(|| format!("copy {s} -> {d}"))?;
            println!("Copied {} ({})", d, fmt_bytes(bytes));
        }
        // upload: local → remote
        (Target::Local(local_path), Target::Remote { profile_name, path: dest }) => {
            let profile = find_profile(store, &profile_name).await?;
            let session = open_session(&profile).await?;
            let dest_parent = parent_of(&dest);
            let local_name = basename(&local_path);
            upload_file(&session, &local_path, &dest_parent, &local_name).await?;
        }
        // download: remote → local
        (Target::Remote { profile_name, path: src }, Target::Local(local_dir)) => {
            let profile = find_profile(store, &profile_name).await?;
            let session = open_session(&profile).await?;
            download_file(&session, &src, &local_dir).await?;
        }
        // remote → remote: not supported across profiles in v1.1.
        (Target::Remote { .. }, Target::Remote { .. }) => {
            bail!("remote-to-remote copy is not yet supported; download then upload");
        }
    }
    Ok(())
}

async fn cmd_mv(store: &ProfileStore, src: &str, dst: &str) -> Result<()> {
    let src_t = parse_target(src);
    let dst_t = parse_target(dst);
    match (src_t, dst_t) {
        (Target::Local(s), Target::Local(d)) => {
            tokio::fs::rename(&s, &d)
                .await
                .with_context(|| format!("rename {s} -> {d}"))?;
        }
        (
            Target::Remote { profile_name: s_p, path: s_path },
            Target::Remote { profile_name: d_p, path: d_path },
        ) if s_p == d_p => {
            let profile = find_profile(store, &s_p).await?;
            let session = open_session(&profile).await?;
            fs_for(&session).rename(&s_path, &d_path).await?;
        }
        _ => bail!("mv must stay within a single profile or be fully local"),
    }
    Ok(())
}

async fn cmd_rm(store: &ProfileStore, raw: &str, recursive: bool) -> Result<()> {
    match parse_target(raw) {
        Target::Local(p) => {
            ghostftp_lib::remotefs::local::LocalFs.delete(&p, recursive).await?;
        }
        Target::Remote { profile_name, path } => {
            let profile = find_profile(store, &profile_name).await?;
            let session = open_session(&profile).await?;
            fs_for(&session).delete(&path, recursive).await?;
        }
    }
    Ok(())
}

async fn cmd_mkdir(store: &ProfileStore, raw: &str) -> Result<()> {
    match parse_target(raw) {
        Target::Local(p) => {
            ghostftp_lib::remotefs::local::LocalFs.create_dir(&p).await?;
        }
        Target::Remote { profile_name, path } => {
            let profile = find_profile(store, &profile_name).await?;
            let session = open_session(&profile).await?;
            fs_for(&session).create_dir(&path).await?;
        }
    }
    Ok(())
}

async fn cmd_sync(
    store: &ProfileStore,
    local: &str,
    remote: &str,
    direction: Dir,
    mirror: bool,
    dry_run: bool,
) -> Result<()> {
    let local_path = match parse_target(local) {
        Target::Local(p) => p,
        Target::Remote { .. } => bail!("sync's first argument must be a local path"),
    };
    let (profile_name, remote_path) = match parse_target(remote) {
        Target::Remote { profile_name, path } => (profile_name, path),
        Target::Local(_) => bail!("sync's second argument must be `profile:/path`"),
    };
    let profile = find_profile(store, &profile_name).await?;
    let session = open_session(&profile).await?;

    let local_fs: Box<dyn RemoteFs> = Box::new(ghostftp_lib::remotefs::local::LocalFs);
    let remote_fs = fs_for(&session);
    let dir = match direction {
        Dir::Push => ghostftp_lib::sync::SyncDirection::LocalToRemote,
        Dir::Pull => ghostftp_lib::sync::SyncDirection::RemoteToLocal,
    };
    let strategy = if mirror {
        ghostftp_lib::sync::SyncStrategy::Mirror
    } else {
        ghostftp_lib::sync::SyncStrategy::Additive
    };

    let plan = ghostftp_lib::sync::plan(
        local_fs.as_ref(),
        remote_fs.as_ref(),
        &local_path,
        &remote_path,
        dir,
        strategy,
    )
    .await?;

    eprintln!(
        "\n{}\n  copies   {}\n  deletes  {}\n  bytes    {}",
        dim(&format!(
            "Plan: {} → {} ({})",
            local_path,
            remote_path,
            if mirror { "mirror" } else { "additive" }
        )),
        plan.copies.len(),
        plan.deletes.len(),
        fmt_bytes(plan.total_bytes),
    );

    if dry_run {
        for c in plan.copies.iter().take(50) {
            eprintln!(
                "  {}  {}  {}",
                reason_label(&c.reason),
                fmt_bytes(c.size),
                c.relative
            );
        }
        if plan.copies.len() > 50 {
            eprintln!("  …and {} more", plan.copies.len() - 50);
        }
        return Ok(());
    }

    if plan.copies.is_empty() && plan.deletes.is_empty() {
        eprintln!("Already in sync.");
        return Ok(());
    }

    let bar = ProgressBar::new(plan.copies.len() as u64);
    bar.set_style(
        ProgressStyle::with_template("{bar:30.cyan/blue} {pos:>3}/{len:3} {msg}")
            .unwrap(),
    );
    for c in &plan.copies {
        bar.set_message(c.relative.clone());
        match direction {
            Dir::Push => {
                let dest_parent = parent_of(&c.destination_path);
                upload_file(&session, &c.source_path, &dest_parent, &basename(&c.source_path))
                    .await?;
            }
            Dir::Pull => {
                let dest_parent = parent_of(&c.destination_path);
                tokio::fs::create_dir_all(&dest_parent).await.ok();
                download_file(&session, &c.source_path, &dest_parent).await?;
            }
        }
        bar.inc(1);
    }
    bar.finish_and_clear();

    for d in &plan.deletes {
        let fs: &dyn RemoteFs = match direction {
            Dir::Push => remote_fs.as_ref(),
            Dir::Pull => local_fs.as_ref(),
        };
        if let Err(e) = fs.delete(&d.path, false).await {
            eprintln!("{} {}: {e}", warn("delete failed"), d.path);
        }
    }
    eprintln!("Sync complete.");
    Ok(())
}

// ---- Directory diff ----------------------------------------------------

/// One side of a diff: a local path, or an opened remote session + path. The
/// session is kept alive because `--hash` reads bytes through it after the walk.
enum DiffSide {
    Local(String),
    Remote { session: Session, path: String },
}

/// Parse a diff side. Same rules as `parse_target`, plus an explicit `local:`
/// prefix (the plan's documented syntax) that forces the local filesystem even
/// where a same-named profile exists.
fn parse_diff_target(raw: &str) -> Target {
    if let Some((name, path)) = raw.split_once(':') {
        if name.eq_ignore_ascii_case("local") {
            return Target::Local(path.to_string());
        }
    }
    parse_target(raw)
}

async fn resolve_diff_side(store: &ProfileStore, raw: &str) -> Result<DiffSide> {
    match parse_diff_target(raw) {
        Target::Local(p) => Ok(DiffSide::Local(p)),
        Target::Remote { profile_name, path } => {
            let profile = find_profile(store, &profile_name).await?;
            let session = open_session(&profile).await?;
            Ok(DiffSide::Remote { session, path })
        }
    }
}

async fn cmd_diff(
    store: &ProfileStore,
    a: &str,
    b: &str,
    hash: bool,
    json: bool,
    all: bool,
) -> Result<()> {
    let side_a = resolve_diff_side(store, a).await?;
    let side_b = resolve_diff_side(store, b).await?;

    let fs_a: Box<dyn RemoteFs> = match &side_a {
        DiffSide::Local(_) => Box::new(ghostftp_lib::remotefs::local::LocalFs),
        DiffSide::Remote { session, .. } => fs_for(session),
    };
    let fs_b: Box<dyn RemoteFs> = match &side_b {
        DiffSide::Local(_) => Box::new(ghostftp_lib::remotefs::local::LocalFs),
        DiffSide::Remote { session, .. } => fs_for(session),
    };

    let (root_a, sess_a) = match &side_a {
        DiffSide::Local(p) => (p.as_str(), None),
        DiffSide::Remote { session, path } => (path.as_str(), Some(session)),
    };
    let (root_b, sess_b) = match &side_b {
        DiffSide::Local(p) => (p.as_str(), None),
        DiffSide::Remote { session, path } => (path.as_str(), Some(session)),
    };

    if !json {
        eprintln!(
            "{}",
            dim(&format!(
                "Diffing {} ↔ {}{}",
                root_a,
                root_b,
                if hash { " (hashing content)" } else { "" }
            ))
        );
    }

    let result = ghostftp_lib::diff::diff(
        fs_a.as_ref(),
        root_a,
        sess_a,
        fs_b.as_ref(),
        root_b,
        sess_b,
        hash,
    )
    .await?;

    if json {
        println!("{}", serde_json::to_string_pretty(&result)?);
    } else {
        print_diff_human(&result, all);
    }

    // Conventional diff exit code: 0 when identical, 1 when the trees differ.
    let s = &result.summary;
    let differ = s.only_in_a + s.only_in_b + s.different;
    std::process::exit(if differ == 0 { 0 } else { 1 })
}

fn print_diff_human(result: &ghostftp_lib::diff::DiffResult, all: bool) {
    use ghostftp_lib::diff::{DiffClass, DiffReason};

    let mut shown = 0usize;
    for e in &result.entries {
        if e.class == DiffClass::Same && !all {
            continue;
        }
        let (marker, detail) = match e.class {
            DiffClass::OnlyInA => (
                "\x1b[36mA only\x1b[0m",
                e.a_size.map(fmt_bytes).unwrap_or_default(),
            ),
            DiffClass::OnlyInB => (
                "\x1b[35mB only\x1b[0m",
                e.b_size.map(fmt_bytes).unwrap_or_default(),
            ),
            DiffClass::Different => {
                let why = match e.reason {
                    Some(DiffReason::Size) => format!(
                        "size {} → {}",
                        e.a_size.map(fmt_bytes).unwrap_or_default(),
                        e.b_size.map(fmt_bytes).unwrap_or_default()
                    ),
                    Some(DiffReason::Content) => "content".to_string(),
                    None => String::new(),
                };
                ("\x1b[33mdiffer\x1b[0m", why)
            }
            DiffClass::Same => ("\x1b[2msame\x1b[0m  ", String::new()),
        };
        let hash_note = if e.hash_error.is_some() {
            dim("  [hash unavailable]")
        } else {
            String::new()
        };
        if detail.is_empty() {
            println!("{marker}  {}{hash_note}", e.relative);
        } else {
            println!("{marker}  {}  {}{hash_note}", e.relative, dim(&detail));
        }
        shown += 1;
    }

    if shown == 0 {
        eprintln!("Trees are identical.");
    }

    let s = &result.summary;
    eprintln!(
        "\n{}",
        dim(&format!(
            "{} only in A · {} only in B · {} differ · {} same{}",
            s.only_in_a,
            s.only_in_b,
            s.different,
            s.same,
            if result.hashed { " (hashed)" } else { "" }
        ))
    );
}

// ---- Duplicate finder --------------------------------------------------

async fn cmd_dedupe(
    store: &ProfileStore,
    target: &str,
    hash: bool,
    json: bool,
    delete: bool,
    yes: bool,
) -> Result<()> {
    use ghostftp_lib::dedupe::{self, DedupeMode};

    if delete && !yes {
        anyhow::bail!("--delete permanently removes files; pass --yes to confirm");
    }

    // Resolve the target to a RemoteFs + optional live session (local = None).
    // parse_diff_target also honors the explicit `local:` prefix.
    let (fs, session, root): (Box<dyn RemoteFs>, Option<Session>, String) = match parse_diff_target(target) {
        Target::Local(p) => (Box::new(ghostftp_lib::remotefs::local::LocalFs), None, p),
        Target::Remote { profile_name, path } => {
            let profile = find_profile(store, &profile_name).await?;
            let session = open_session(&profile).await?;
            let root = path.clone();
            (fs_for(&session), Some(session), root)
        }
    };

    let mode = if hash { DedupeMode::Hash } else { DedupeMode::Name };
    if !json {
        eprintln!(
            "{}",
            dim(&format!(
                "Scanning {} for duplicates{}…",
                root,
                if hash { " (hashing content)" } else { "" }
            ))
        );
    }

    let result = dedupe::find_duplicates(fs.as_ref(), &root, session.as_ref(), mode).await?;

    if json {
        println!("{}", serde_json::to_string_pretty(&result)?);
    } else {
        print_dedupe_human(&result);
    }

    if delete {
        let paths = dedupe::duplicate_paths(&result);
        if paths.is_empty() {
            eprintln!("Nothing to delete.");
            return Ok(());
        }
        let mut failed = 0usize;
        for p in &paths {
            match fs.delete(p, false).await {
                Ok(()) => {
                    if !json {
                        println!("{}", dim(&format!("deleted  {p}")));
                    }
                }
                Err(e) => {
                    failed += 1;
                    eprintln!("failed   {p}: {e}");
                }
            }
        }
        eprintln!(
            "{}",
            dim(&format!(
                "Deleted {} of {} duplicates{}.",
                paths.len() - failed,
                paths.len(),
                if failed > 0 { format!(" ({failed} failed)") } else { String::new() }
            ))
        );
        if failed > 0 {
            std::process::exit(1);
        }
    }

    Ok(())
}

fn print_dedupe_human(result: &ghostftp_lib::dedupe::DedupeResult) {
    if result.groups.is_empty() {
        eprintln!("No duplicates found.");
    }
    for g in &result.groups {
        let wasted = g.size * (g.files.len() - 1) as u64;
        println!(
            "\x1b[33mgroup\x1b[0m  {}  {}",
            g.key,
            dim(&format!(
                "({} files · {} each · {} wasted)",
                g.files.len(),
                fmt_bytes(g.size),
                fmt_bytes(wasted)
            ))
        );
        for (i, f) in g.files.iter().enumerate() {
            if i == g.keep {
                println!("\x1b[32m  keep\x1b[0m  {}", f.path);
            } else {
                println!("  \x1b[31mdup\x1b[0m   {}", f.path);
            }
        }
    }
    let s = &result.summary;
    eprintln!(
        "\n{}",
        dim(&format!(
            "{} groups · {} duplicates · {} wasted · {} files scanned{}{}",
            s.groups,
            s.duplicate_files,
            fmt_bytes(s.wasted_bytes),
            s.files_scanned,
            if result.mode == ghostftp_lib::dedupe::DedupeMode::Hash { " · hashed" } else { "" },
            if s.hash_errors > 0 { format!(" · {} hash errors", s.hash_errors) } else { String::new() }
        ))
    );
}

// ---- Fleet search ------------------------------------------------------

#[allow(clippy::too_many_arguments)]
async fn cmd_search(
    store: &ProfileStore,
    target: &str,
    pattern: &str,
    content: bool,
    regex: bool,
    case_sensitive: bool,
    include: Vec<String>,
    exclude: Vec<String>,
    content_remote: bool,
    max: Option<usize>,
    json: bool,
) -> Result<()> {
    use ghostftp_lib::search::{SearchKind, SearchQuery, DEFAULT_MAX_FILE_BYTES, DEFAULT_MAX_RESULTS};

    // Resolve the target to a RemoteFs + optional live session (local = None).
    let (fs, session, root): (Box<dyn RemoteFs>, Option<Session>, String) = match parse_target(target) {
        Target::Local(p) => (Box::new(ghostftp_lib::remotefs::local::LocalFs), None, p),
        Target::Remote { profile_name, path } => {
            let profile = find_profile(store, &profile_name).await?;
            let session = open_session(&profile).await?;
            let fs = fs_for(&session);
            (fs, Some(session), path)
        }
    };
    let root = if root.trim().is_empty() { ".".to_string() } else { root };

    // `--regex` implies content search (a regex over names is unusual; the plan
    // scopes regex to content grep).
    let content = content || regex;
    let query = SearchQuery {
        pattern: pattern.to_string(),
        kind: if content { SearchKind::Content } else { SearchKind::Name },
        regex,
        case_sensitive,
        include_globs: include,
        exclude_globs: exclude,
        content_remote,
        max_results: max.unwrap_or(DEFAULT_MAX_RESULTS),
        max_file_bytes: DEFAULT_MAX_FILE_BYTES,
    };

    if !json {
        eprintln!(
            "{}",
            dim(&format!(
                "Searching {root} for {pattern:?}{}",
                if content { " (content)" } else { "" }
            ))
        );
    }

    let result = ghostftp_lib::search::search(fs.as_ref(), session.as_ref(), &root, &query).await?;

    if json {
        println!("{}", serde_json::to_string_pretty(&result)?);
    } else {
        print_search_human(&result);
    }

    // grep convention: 0 when something matched, 1 when nothing did.
    std::process::exit(if result.hits.is_empty() { 1 } else { 0 })
}

fn print_search_human(result: &ghostftp_lib::search::SearchResult) {
    use ghostftp_lib::search::{SearchKind, SearchStrategy};

    for h in &result.hits {
        match result.kind {
            SearchKind::Name => {
                if h.is_dir {
                    println!("\x1b[34m{}/\x1b[0m", h.relative);
                } else {
                    println!("{}  {}", h.relative, dim(&fmt_bytes(h.size)));
                }
            }
            SearchKind::Content => {
                let line = h.line.unwrap_or(0);
                let preview = h.preview.as_deref().unwrap_or("");
                println!(
                    "\x1b[36m{}\x1b[0m:\x1b[33m{}\x1b[0m: {}",
                    h.relative, line, preview
                );
            }
        }
    }

    if result.hits.is_empty() {
        eprintln!("No matches.");
    }

    let s = &result.stats;
    let strat = match s.strategy {
        SearchStrategy::Generic => "walk",
        SearchStrategy::Shell => "exec",
        SearchStrategy::ObjectFlat => "object",
    };
    let mut summary = format!("{} hits · {strat}", result.hits.len());
    if s.truncated {
        summary.push_str(" · capped (more matches exist)");
    }
    if let Some(note) = &s.note {
        summary.push_str(&format!(" · {note}"));
    }
    eprintln!("\n{}", dim(&summary));
}

// ---- Authenticated fetch (Plan 10 Phase 5) -----------------------------

/// GET `url_str` through a saved HTTP(S) profile's stored Basic-Auth creds and
/// write the body to stdout. The profile is chosen explicitly (`--profile`) or
/// matched to the URL's host. The Authorization header is applied by the session
/// and never printed.
async fn cmd_fetch(store: &ProfileStore, url_str: &str, profile_opt: Option<String>) -> Result<()> {
    let url = reqwest::Url::parse(url_str).with_context(|| format!("parse URL {url_str}"))?;
    if !matches!(url.scheme(), "http" | "https") {
        bail!("fetch needs an http/https URL (got {})", url.scheme());
    }
    let host = url.host_str().ok_or_else(|| anyhow!("URL has no host"))?;

    let profile = match profile_opt {
        Some(name) => {
            let p = find_profile(store, &name).await?;
            if p.protocol != "http" && p.protocol != "https" {
                bail!("`{}` is not an HTTP/HTTPS profile", p.name);
            }
            p
        }
        None => {
            let profiles = store.list().await?;
            let matches: Vec<ConnectionProfile> = profiles
                .into_iter()
                .filter(|p| {
                    (p.protocol == "http" || p.protocol == "https")
                        && profile_host(p).as_deref().map(|h| h.eq_ignore_ascii_case(host))
                            .unwrap_or(false)
                })
                .collect();
            match matches.len() {
                1 => matches.into_iter().next().unwrap(),
                0 => bail!(
                    "no saved HTTP profile matches host `{host}`. Add one in Ghost FTP, or pass \
                     --profile <name>."
                ),
                _ => bail!(
                    "more than one saved HTTP profile matches host `{host}`; pass --profile <name>."
                ),
            }
        }
    };

    let session = open_session(&profile).await?;
    let Session::Http(http) = session else {
        bail!("`{}` is not an HTTP/HTTPS profile", profile.name);
    };
    let resp = http
        .request(reqwest::Method::GET, url)
        .send()
        .await
        .context("GET failed")?;
    let status = resp.status();
    let body = resp.bytes().await.context("read response body")?;
    let mut so = io::stdout();
    so.write_all(&body).ok();
    so.flush().ok();
    if !status.is_success() {
        eprintln!("\n{}", warn(&format!("HTTP {}", status.as_u16())));
        std::process::exit(1);
    }
    Ok(())
}

/// The host of an HTTP(S) profile's URL (stored in `endpoint`), for matching a
/// `fetch` URL to a saved profile.
fn profile_host(p: &ConnectionProfile) -> Option<String> {
    let raw = p.endpoint.as_deref()?.trim();
    let with_scheme = if raw.contains("://") {
        raw.to_string()
    } else {
        format!("https://{raw}")
    };
    reqwest::Url::parse(&with_scheme).ok()?.host_str().map(|h| h.to_string())
}

async fn cmd_profiles_list(store: &ProfileStore) -> Result<()> {
    let profiles = store.list().await?;
    if profiles.is_empty() {
        eprintln!("No profiles saved. Open the GUI to add one.");
        return Ok(());
    }
    for p in profiles {
        let suffix = match p.protocol.as_str() {
            "s3" | "azure" => format!(
                "{}://{}",
                p.protocol,
                p.bucket.as_deref().unwrap_or("?")
            ),
            _ => format!("{}@{}:{}", p.username, p.host, p.port),
        };
        println!("{:<24} {:<6} {}", p.name, p.protocol, suffix);
    }
    Ok(())
}

async fn cmd_profiles_show(store: &ProfileStore, name: &str) -> Result<()> {
    let p = find_profile(store, name).await?;
    let mut json = serde_json::to_value(&p)?;
    // Redact the password / passphrase before printing.
    if let Some(obj) = json.as_object_mut() {
        if let Some(auth) = obj.get_mut("auth") {
            if let Some(auth_obj) = auth.as_object_mut() {
                if auth_obj.contains_key("password") {
                    auth_obj.insert("password".into(), serde_json::json!("<redacted>"));
                }
                if auth_obj.contains_key("passphrase") {
                    auth_obj.insert("passphrase".into(), serde_json::json!("<redacted>"));
                }
            }
        }
    }
    println!("{}", serde_json::to_string_pretty(&json)?);
    Ok(())
}

// ---- self-update (Plan 10 Phase 0b) ------------------------------------

/// Ghost FTP release service base URL (same as scripts/install-agentd.sh).
const RELEASE_BASE: &str = "https://ghostftp.com";

/// The release asset name matching this OS/arch — mirrors the `ghostftp-cli` matrix
/// in `the Ghost FTP release pipeline` and the daemon installer's naming.
fn target_asset_name() -> Result<&'static str> {
    Ok(match (std::env::consts::OS, std::env::consts::ARCH) {
        ("windows", "x86_64") => "ghostftp-cli-windows-x86_64.exe",
        ("linux", "x86_64") => "ghostftp-cli-linux-x86_64",
        (os, arch) => bail!(
            "no prebuilt ghostftp-cli for {os}/{arch}; build from source (cargo build -p ghostftp-cli)"
        ),
    })
}

/// Ask the Ghost FTP release service for the latest release tag, normalised to a bare version
/// (`v1.4.0` → `1.4.0`).
fn latest_release_tag() -> Result<String> {
    let agent = ureq::AgentBuilder::new()
        .timeout(std::time::Duration::from_secs(30))
        .build();
    let url = format!("{RELEASE_BASE}/api/releases/latest.json");
    let resp = agent
        .get(&url)
        .set("User-Agent", "ghostftp-cli-self-update")
        .set("Accept", "application/json")
        .call()
        .with_context(|| format!("query {url}"))?;
    let v: serde_json::Value = resp.into_json().context("parse Ghost FTP release JSON")?;
    let tag = v
        .get("tag_name")
        .and_then(|t| t.as_str())
        .ok_or_else(|| anyhow!("the Ghost FTP release response had no tag_name"))?;
    Ok(tag.trim_start_matches('v').to_string())
}

/// Release download URL for `asset`, at a specific tag or `latest`.
fn asset_url(tag: Option<&str>, asset: &str) -> String {
    match tag {
        Some(t) => format!("{RELEASE_BASE}/downloads/{t}/{asset}"),
        None => format!("{RELEASE_BASE}/downloads/{asset}"),
    }
}

fn download_bytes(url: &str) -> Result<Vec<u8>> {
    use std::io::Read as _;
    let agent = ureq::AgentBuilder::new()
        .timeout(std::time::Duration::from_secs(300))
        .build();
    let resp = agent
        .get(url)
        .set("User-Agent", "ghostftp-cli-self-update")
        .call()
        .with_context(|| format!("download {url}"))?;
    let mut buf = Vec::new();
    resp.into_reader()
        .read_to_end(&mut buf)
        .context("read download body")?;
    Ok(buf)
}

/// Replace the binary at `target` with `new_bytes`. A running exe can't be
/// overwritten in place on Windows (and to stay crash-safe on Unix), so write the
/// new bytes to a sibling temp file on the same volume, then rename: on Unix an
/// atomic rename over the target (the running process keeps the old inode); on
/// Windows, move the current exe aside first (allowed while running), move the
/// new one in, and leave the `.old` for the OS to reap. Split out from
/// `current_exe()` so the swap mechanism is unit-testable on a temp file.
fn swap_binary_at(target: &std::path::Path, new_bytes: &[u8]) -> Result<()> {
    let dir = target
        .parent()
        .ok_or_else(|| anyhow!("target has no parent directory"))?;
    let file_name = target
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("ghostftp-cli");
    let new_path = dir.join(format!("{file_name}.new"));
    std::fs::write(&new_path, new_bytes)
        .with_context(|| format!("write {}", new_path.display()))?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&new_path, std::fs::Permissions::from_mode(0o755))
            .context("chmod +x the new binary")?;
        std::fs::rename(&new_path, target)
            .with_context(|| format!("replace {}", target.display()))?;
    }
    #[cfg(windows)]
    {
        let old_path = dir.join(format!("{file_name}.old"));
        let _ = std::fs::remove_file(&old_path); // clear a stale one
        std::fs::rename(target, &old_path).context("rename current exe aside")?;
        if let Err(e) = std::fs::rename(&new_path, target) {
            // Roll back so the user isn't left without a binary.
            let _ = std::fs::rename(&old_path, target);
            let _ = std::fs::remove_file(&new_path);
            return Err(anyhow::Error::new(e).context("move the new exe into place"));
        }
        let _ = std::fs::remove_file(&old_path); // often locked while running; ignore
    }
    Ok(())
}

fn cmd_self_update(tag: Option<String>, check: bool) -> Result<()> {
    let current = cli_version();
    let asset = target_asset_name()?;

    // Target version for reporting: the explicit tag, else the Ghost FTP release service's latest.
    let target_ver = match &tag {
        Some(t) => t.trim_start_matches('v').to_string(),
        None => latest_release_tag()?,
    };

    if check {
        println!("current: v{current}");
        println!("latest:  v{target_ver}");
        match (parse_semver(current), parse_semver(&target_ver)) {
            (Some(c), Some(l)) if c < l => {
                println!("→ an update is available — run `ghostftp-cli self-update`");
            }
            (Some(_), Some(_)) => println!("→ already up to date"),
            _ => println!("→ could not compare versions"),
        }
        return Ok(());
    }

    // Skip a no-op update to latest when we're already current.
    if tag.is_none() {
        if let (Some(c), Some(l)) = (parse_semver(current), parse_semver(&target_ver)) {
            if c >= l {
                println!("ghostftp-cli v{current} is already up to date.");
                return Ok(());
            }
        }
    }

    let url = asset_url(tag.as_deref(), asset);
    eprintln!("Downloading {asset} (v{target_ver})…");
    let bytes = download_bytes(&url)?;
    // A real ghostftp-cli is multi-MB; a tiny body is an HTML error page (bad tag) or
    // a redirect that wasn't followed — refuse to install it over the binary.
    if bytes.len() < 200_000 {
        bail!(
            "downloaded asset is only {} bytes — that doesn't look like ghostftp-cli (wrong --tag?)",
            bytes.len()
        );
    }
    let exe = std::env::current_exe().context("resolve the running ghostftp-cli path")?;
    swap_binary_at(&exe, &bytes)?;
    println!("Updated ghostftp-cli: v{current} → v{target_ver}");
    #[cfg(windows)]
    println!(
        "(this process keeps running the old code; the next `ghostftp-cli` call uses the new binary)"
    );
    Ok(())
}

// ---- Agent Bridge client -----------------------------------------------
//
// `ghostftp-cli agent …` talks to Ghost FTP's RUNNING Agent Bridge over localhost HTTP
// rather than opening its own connection. It reads the bridge URL + bearer
// token from the discovery file Ghost FTP writes while the bridge is on, so neither
// the user nor an AI agent ever handles a URL or token — and every call still
// goes through Ghost FTP's per-command approval and live console.

struct Endpoint {
    url: String,
    token: String,
    /// The running Ghost FTP app's version, published in the discovery file. Used to
    /// warn when this CLI binary is older than the app (Plan 10 Phase 0a).
    app_version: Option<String>,
}

/// Read the bridge URL + token from Ghost FTP's discovery file (same data dir the
/// GUI uses). A missing file means the bridge isn't running.
fn read_endpoint() -> Result<Endpoint> {
    let path = default_data_dir()?.join("agent-endpoint.json");
    let bytes = std::fs::read(&path).map_err(|_| {
        anyhow!(
            "Ghost FTP's Agent Bridge isn't running. Open Ghost FTP and turn on the Agent \
             Bridge (the master switch at the top of the Bridge panel)."
        )
    })?;
    let v: serde_json::Value = serde_json::from_slice(&bytes)
        .context("agent-endpoint.json is corrupt; toggle the Agent Bridge off and on in Ghost FTP")?;
    let url = v
        .get("url")
        .and_then(|x| x.as_str())
        .ok_or_else(|| anyhow!("agent-endpoint.json is missing `url`"))?
        .to_string();
    let token = v
        .get("token")
        .and_then(|x| x.as_str())
        .ok_or_else(|| anyhow!("agent-endpoint.json is missing `token`"))?
        .to_string();
    let app_version = v
        .get("version")
        .and_then(|x| x.as_str())
        .map(String::from);
    Ok(Endpoint { url, token, app_version })
}

/// This CLI binary's build version.
fn cli_version() -> &'static str {
    env!("CARGO_PKG_VERSION")
}

/// Parse a `MAJOR.MINOR.PATCH` version into a comparable tuple, ignoring any
/// pre-release/build metadata (`-rc1`, `+meta`). Returns None if it doesn't look
/// like a semver, so an unparseable value simply suppresses the staleness check
/// rather than firing a bogus warning.
fn parse_semver(s: &str) -> Option<(u64, u64, u64)> {
    let core = s.trim().split(['-', '+']).next().unwrap_or(s);
    let mut it = core.split('.');
    let major = it.next()?.parse().ok()?;
    let minor = it.next().unwrap_or("0").parse().ok()?;
    let patch = it.next().unwrap_or("0").parse().ok()?;
    Some((major, minor, patch))
}

/// Print a one-line stderr warning when this CLI is OLDER than the running Ghost FTP
/// app (Plan 10 Phase 0a). The app and `ghostftp-cli` ship as separate downloads, so
/// the CLI silently lags after an app update — this turns a cryptic
/// `unexpected argument '--flag'` into a clear "your CLI is stale." Best-effort:
/// if either version is absent or unparseable, or the CLI is same/newer, it says
/// nothing.
fn warn_if_stale(ep: &Endpoint) {
    let Some(app) = ep.app_version.as_deref().and_then(parse_semver) else {
        return;
    };
    let Some(cli) = parse_semver(cli_version()) else {
        return;
    };
    if cli < app {
        eprintln!(
            "{}",
            warn(&format!(
                "ghostftp-cli v{} is older than the running Ghost FTP app v{}; some commands/flags \
                 may be missing — update with `ghostftp-cli self-update` or from Ghost FTP → Settings.",
                cli_version(),
                ep.app_version.as_deref().unwrap_or("?"),
            ))
        );
    }
}

/// POST a JSON body to a bridge route. The 180s timeout comfortably exceeds the
/// bridge's 120s approval window, so a call that's blocked waiting for the user
/// to click Approve in Ghost FTP doesn't time out on our side first.
fn http_post(ep: &Endpoint, route: &str, body: serde_json::Value) -> Result<serde_json::Value> {
    let agent = ureq::AgentBuilder::new()
        .timeout(std::time::Duration::from_secs(180))
        .build();
    match agent
        .post(&format!("{}{}", ep.url, route))
        .set("Authorization", &format!("Bearer {}", ep.token))
        .set("Content-Type", "application/json")
        .send_json(body)
    {
        Ok(r) => Ok(r.into_json::<serde_json::Value>()?),
        Err(ureq::Error::Status(code, r)) => bail!("Agent Bridge error ({code}): {}", err_text(r, code)),
        Err(e) => bail!(
            "couldn't reach Ghost FTP's Agent Bridge at {} ({e}). Is Ghost FTP still running with the bridge on?",
            ep.url
        ),
    }
}

fn http_get(ep: &Endpoint, route: &str) -> Result<serde_json::Value> {
    let agent = ureq::AgentBuilder::new()
        .timeout(std::time::Duration::from_secs(15))
        .build();
    match agent
        .get(&format!("{}{}", ep.url, route))
        .set("Authorization", &format!("Bearer {}", ep.token))
        .call()
    {
        Ok(r) => Ok(r.into_json::<serde_json::Value>()?),
        Err(ureq::Error::Status(code, r)) => bail!("Agent Bridge error ({code}): {}", err_text(r, code)),
        Err(e) => bail!(
            "couldn't reach Ghost FTP's Agent Bridge at {} ({e}). Is Ghost FTP still running with the bridge on?",
            ep.url
        ),
    }
}

/// Pull the bridge's `{error}` text out of a non-2xx response (falls back to the
/// status code).
fn err_text(r: ureq::Response, code: u16) -> String {
    r.into_json::<serde_json::Value>()
        .ok()
        .and_then(|v| v.get("error").and_then(|e| e.as_str()).map(String::from))
        .unwrap_or_else(|| format!("HTTP {code}"))
}

/// Map a friendly server name to a bridge session id (the REST routes need the
/// exact id). Mirrors the bridge's own MCP name resolution: exact id wins, then
/// a unique case-insensitive name; an ambiguous name errors rather than guesses.
fn resolve_server(ep: &Endpoint, name: &str) -> Result<String> {
    let body = http_get(ep, "/sessions")?;
    let sessions = body
        .get("sessions")
        .and_then(|v| v.as_array())
        .cloned()
        .unwrap_or_default();
    if sessions.is_empty() {
        bail!("no server has granted agent access yet — enable one in Ghost FTP's Agent Bridge panel");
    }
    if sessions
        .iter()
        .any(|s| s.get("id").and_then(|v| v.as_str()) == Some(name))
    {
        return Ok(name.to_string());
    }
    let matches: Vec<&serde_json::Value> = sessions
        .iter()
        .filter(|s| {
            s.get("name")
                .and_then(|v| v.as_str())
                .map(|n| n.eq_ignore_ascii_case(name))
                .unwrap_or(false)
        })
        .collect();
    match matches.len() {
        1 => Ok(matches[0]
            .get("id")
            .and_then(|v| v.as_str())
            .unwrap_or_default()
            .to_string()),
        0 => {
            let names: Vec<&str> = sessions
                .iter()
                .filter_map(|s| s.get("name").and_then(|v| v.as_str()))
                .collect();
            bail!("no enabled server matches \"{name}\". Available: {}", names.join(", "))
        }
        _ => bail!(
            "\"{name}\" matches more than one connected server; pass the exact session id (see `ghostftp-cli agent sessions`)"
        ),
    }
}

/// True when `p` starts with a Windows drive prefix (`C:\` or `C:/`).
fn is_windows_drive_path(p: &str) -> bool {
    let b = p.as_bytes();
    b.len() >= 3 && b[0].is_ascii_alphabetic() && b[1] == b':' && (b[2] == b'/' || b[2] == b'\\')
}

/// Best-effort: is the connected server a Windows machine? Reads `/info`'s
/// `remote.os` (cached at connect for Ghost FTP Agents; `uname -s` for SSH). Any
/// failure or unknown OS → false, so the mangling guard errs toward firing.
fn server_is_windows(ep: &Endpoint, session_id: &str) -> bool {
    let Ok(body) = http_post(ep, "/info", serde_json::json!({ "sessionId": session_id })) else {
        return false;
    };
    body.get("remote")
        .and_then(|r| r.get("os"))
        .and_then(|v| v.as_str())
        .map(|os| os.eq_ignore_ascii_case("windows"))
        .unwrap_or(false)
}

/// Reject a remote-path argument that Git Bash / MSYS almost certainly mangled.
/// When a remote path arrives Windows-drive-prefixed (`C:\…` / `C:/…`) but the
/// target isn't actually a Windows machine, MSYS rewrote a POSIX path like
/// `/var/www` into `C:/Program Files/Git/var/www` *before* ghostftp-cli saw it —
/// uploading there would silently land in a nonsense directory. Turn that into a
/// clear, actionable error (Plan 10 Phase 3). The `/info` round-trip only happens
/// on the drive-prefixed error path, so the common good path pays nothing.
fn guard_mangled_remote_path(ep: &Endpoint, session_id: &str, remote_path: &str) -> Result<()> {
    // Only pay for the OS lookup when the path actually looks mangled.
    let is_windows_target = is_windows_drive_path(remote_path) && server_is_windows(ep, session_id);
    check_mangled_remote_path(remote_path, is_windows_target)
}

/// True when `p` is a drive letter whose separator went missing — `C:UsersUser`.
/// That is what an unquoted `C:\Users\User` becomes in bash/zsh, where each
/// backslash escapes the character after it. Windows reads the result as a
/// *drive-relative* path, so it resolves against the daemon's working directory
/// instead of failing — the upload lands somewhere real but wrong, silently.
fn is_collapsed_windows_path(p: &str) -> bool {
    let b = p.as_bytes();
    b.len() >= 3 && b[0].is_ascii_alphabetic() && b[1] == b':' && b[2] != b'/' && b[2] != b'\\'
}

/// Pure decision behind [`guard_mangled_remote_path`], split out so the
/// reject/allow rule and its message are unit-testable without a live bridge.
fn check_mangled_remote_path(remote_path: &str, is_windows_target: bool) -> Result<()> {
    if is_collapsed_windows_path(remote_path) {
        bail!(
            "`{remote_path}` is missing its path separators — your shell ate the backslashes \
             (in bash, `C:\\Users\\User` collapses to `C:UsersUser`). Windows would resolve that \
             against the server's current directory and put the file somewhere you didn't mean. \
             Use forward slashes (`C:/Users/User`) or single-quote the path ('C:\\Users\\User')."
        );
    }
    if !is_windows_drive_path(remote_path) || is_windows_target {
        return Ok(());
    }
    bail!(
        "that remote path looks like Git Bash rewrote it (MSYS path conversion turned a \
         POSIX path such as /var/www into `{remote_path}`). Re-run with `MSYS_NO_PATHCONV=1`, \
         prefix the path with `//` (e.g. `//var/www`), or drop text straight in with \
         `ghostftp-cli agent write`."
    )
}

/// Read a script's bytes from a local file or stdin, returning the bytes plus a
/// friendly label (the filename, or "stdin") for Ghost FTP's console/audit. Used by
/// `agent exec --file/--stdin` and `agent script` (Plan 10 Phase 1).
fn read_script_source(file: Option<&str>, stdin: bool) -> Result<(Vec<u8>, String)> {
    if let Some(path) = file {
        let bytes = std::fs::read(path).with_context(|| format!("read script {path}"))?;
        Ok((bytes, format!("script {}", basename(path))))
    } else if stdin {
        use std::io::Read as _;
        let mut buf = Vec::new();
        io::stdin()
            .read_to_end(&mut buf)
            .context("read script from stdin")?;
        Ok((buf, "script (stdin)".to_string()))
    } else {
        bail!("no script source (expected --file or --stdin)")
    }
}

/// Ship a script to the bridge's `/exec_script` route (base64 on the wire so any
/// bytes survive) and print its output. The user never sees the base64 — that's
/// the whole point (Plan 10 Phase 1).
fn run_agent_script(
    ep: &Endpoint,
    session_id: &str,
    script: &[u8],
    label: &str,
    dry_run: bool,
    timeout_ms: Option<u64>,
) -> Result<()> {
    use base64::Engine as _;
    let b64 = base64::engine::general_purpose::STANDARD.encode(script);
    let mut req = serde_json::json!({
        "sessionId": session_id,
        "script": b64,
        "label": label,
        "dryRun": dry_run,
    });
    if let Some(t) = timeout_ms {
        req["timeoutMs"] = serde_json::json!(t);
    }
    let body = http_post(ep, "/exec_script", req)?;
    if dry_run {
        println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
        return Ok(());
    }
    print_exec_output_and_exit(&body)
}

/// Print an exec/script result (stdout, stderr, timeout note) and exit this
/// process with the remote command's exit code. Shared by `agent exec`,
/// `agent script` and `agent exec --file/--stdin`.
fn print_exec_output_and_exit(body: &serde_json::Value) -> Result<()> {
    if let Some(out) = body.get("stdout").and_then(|v| v.as_str()) {
        let mut so = io::stdout();
        so.write_all(out.as_bytes()).ok();
        so.flush().ok();
    }
    if let Some(e) = body.get("stderr").and_then(|v| v.as_str()) {
        if !e.is_empty() {
            let mut se = io::stderr();
            se.write_all(e.as_bytes()).ok();
            se.flush().ok();
        }
    }
    if body.get("timedOut").and_then(|v| v.as_bool()).unwrap_or(false) {
        eprintln!("{}", warn("command timed out before finishing"));
    }
    // exitCode is a number or null on the wire; null => 0, like cmd_exec.
    let code = body.get("exitCode").and_then(|v| v.as_i64()).unwrap_or(0);
    std::process::exit(code as i32)
}

fn cmd_agent(action: AgentCmd) -> Result<()> {
    let ep = read_endpoint()?;
    warn_if_stale(&ep);
    match action {
        AgentCmd::Context => {
            let body = http_get(&ep, "/context")?;
            println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
            Ok(())
        }
        AgentCmd::Sessions => {
            let body = http_get(&ep, "/sessions")?;
            let sessions = body
                .get("sessions")
                .and_then(|v| v.as_array())
                .cloned()
                .unwrap_or_default();
            if sessions.is_empty() {
                eprintln!(
                    "No servers have granted agent access. Enable one in Ghost FTP's Agent Bridge panel."
                );
            }
            for s in &sessions {
                let name = s.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                let proto = s.get("protocol").and_then(|v| v.as_str()).unwrap_or("?");
                let host = s.get("host").and_then(|v| v.as_str()).unwrap_or("");
                let can_exec = s.get("canExec").and_then(|v| v.as_bool()).unwrap_or(false);
                println!("{name:<24} {proto:<6} {host:<28} exec={can_exec}");
            }
            Ok(())
        }
        AgentCmd::Exec { server, dry_run, timeout_ms, file, stdin, detach, command } => {
            let id = resolve_server(&ep, &server)?;
            // --file / --stdin ship a script verbatim; otherwise run the joined
            // command line. Exactly one source must be given.
            if file.is_some() || stdin {
                if !command.is_empty() {
                    bail!("pass a command OR --file/--stdin, not both");
                }
                if detach {
                    bail!("--detach can't be combined with --file/--stdin yet");
                }
                let (bytes, label) = read_script_source(file.as_deref(), stdin)?;
                return run_agent_script(&ep, &id, &bytes, &label, dry_run, timeout_ms);
            }
            if command.is_empty() {
                bail!("nothing to run — give a command, or use --file <path> / --stdin");
            }
            let mut req = serde_json::json!({
                "sessionId": id,
                "command": command.join(" "),
                "dryRun": dry_run,
                "detach": detach,
            });
            if let Some(t) = timeout_ms {
                req["timeoutMs"] = serde_json::json!(t);
            }
            let body = http_post(&ep, "/exec", req)?;
            if detach && !dry_run {
                let job = body.get("jobId").and_then(|v| v.as_str()).unwrap_or("?");
                println!("started background job: {job}");
                println!("poll with: ghostftp-cli agent job {server} {job}");
                return Ok(());
            }
            if dry_run {
                println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
                return Ok(());
            }
            print_exec_output_and_exit(&body)
        }
        AgentCmd::Job { server, job_id } => {
            let id = resolve_server(&ep, &server)?;
            let body = http_post(&ep, "/job", serde_json::json!({ "sessionId": id, "jobId": job_id }))?;
            print_job(&body);
            let running = body.get("running").and_then(|v| v.as_bool()).unwrap_or(false);
            // Mirror the job's exit code once it's finished, like a foreground exec.
            if !running {
                if let Some(code) = body.get("exitCode").and_then(|v| v.as_i64()) {
                    std::process::exit(code as i32);
                }
            }
            Ok(())
        }
        AgentCmd::Jobs { server } => {
            let id = resolve_server(&ep, &server)?;
            let body = http_post(&ep, "/jobs", serde_json::json!({ "sessionId": id }))?;
            let jobs = body.get("jobs").and_then(|v| v.as_array()).cloned().unwrap_or_default();
            if jobs.is_empty() {
                eprintln!("No background jobs on {server}.");
            }
            for j in &jobs {
                let jid = j.get("jobId").and_then(|v| v.as_str()).unwrap_or("?");
                let running = j.get("running").and_then(|v| v.as_bool()).unwrap_or(false);
                let code = j.get("exitCode").and_then(|v| v.as_i64());
                let state = if running {
                    "running".to_string()
                } else {
                    format!("done (exit {})", code.map(|c| c.to_string()).unwrap_or_else(|| "?".into()))
                };
                println!("{jid}  {state}");
            }
            Ok(())
        }
        AgentCmd::Script { server, file, dry_run, timeout_ms } => {
            let id = resolve_server(&ep, &server)?;
            let (bytes, label) = read_script_source(Some(&file), false)?;
            run_agent_script(&ep, &id, &bytes, &label, dry_run, timeout_ms)
        }
        AgentCmd::Ls { server, path } => {
            let id = resolve_server(&ep, &server)?;
            let mut req = serde_json::json!({ "sessionId": id });
            if let Some(p) = path {
                req["path"] = serde_json::Value::String(p);
            }
            let body = http_post(&ep, "/list", req)?;
            print_agent_entries(&body);
            Ok(())
        }
        AgentCmd::Read { server, path } => {
            let id = resolve_server(&ep, &server)?;
            let body = http_post(&ep, "/read", serde_json::json!({ "sessionId": id, "path": path }))?;
            if let Some(content) = body.get("content").and_then(|v| v.as_str()) {
                let mut so = io::stdout();
                so.write_all(content.as_bytes()).ok();
                so.flush().ok();
            }
            if body.get("truncated").and_then(|v| v.as_bool()).unwrap_or(false) {
                eprintln!("\n{}", warn("output truncated at 256 KiB"));
            }
            Ok(())
        }
        AgentCmd::ReadBatch { server, paths } => {
            let id = resolve_server(&ep, &server)?;
            let body = http_post(
                &ep,
                "/read_batch",
                serde_json::json!({ "sessionId": id, "paths": paths }),
            )?;
            println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
            Ok(())
        }
        AgentCmd::Glob { server, path, pattern } => {
            let id = resolve_server(&ep, &server)?;
            let mut req = serde_json::json!({ "sessionId": id, "pattern": pattern });
            if let Some(p) = path {
                req["path"] = serde_json::Value::String(p);
            }
            let body = http_post(&ep, "/glob", req)?;
            if let Some(matches) = body.get("matches").and_then(|v| v.as_array()) {
                for m in matches {
                    if let Some(p) = m.as_str() {
                        println!("{p}");
                    }
                }
            }
            Ok(())
        }
        AgentCmd::Search {
            server,
            query,
            path,
            content,
            regex,
            case_sensitive,
            include,
            exclude,
            content_remote,
        } => {
            let id = resolve_server(&ep, &server)?;
            let mut req = serde_json::json!({
                "sessionId": id,
                "query": query,
                "content": content || regex,
                "regex": regex,
                "caseSensitive": case_sensitive,
                "include": include,
                "exclude": exclude,
                "contentRemote": content_remote,
            });
            if let Some(p) = path {
                req["path"] = serde_json::Value::String(p);
            }
            let body = http_post(&ep, "/search", req)?;
            let is_content = body.get("kind").and_then(|v| v.as_str()) == Some("content");
            if let Some(matches) = body.get("matches").and_then(|v| v.as_array()) {
                for m in matches {
                    let path = m.get("path").and_then(|v| v.as_str()).unwrap_or("");
                    if is_content {
                        let line = m.get("line").and_then(|v| v.as_u64()).unwrap_or(0);
                        let preview = m.get("preview").and_then(|v| v.as_str()).unwrap_or("");
                        println!(
                            "\x1b[36m{path}\x1b[0m:\x1b[33m{line}\x1b[0m: {preview}"
                        );
                    } else if m.get("isDir").and_then(|v| v.as_bool()).unwrap_or(false) {
                        println!("{path}/");
                    } else {
                        println!("{path}");
                    }
                }
            }
            if let Some(note) = body.get("note").and_then(|v| v.as_str()) {
                eprintln!("{}", dim(note));
            }
            if body.get("truncated").and_then(|v| v.as_bool()).unwrap_or(false) {
                eprintln!("{}", warn("results truncated (cap reached)"));
            }
            Ok(())
        }
        AgentCmd::Download { server, remote_path, local_dir } => {
            let id = resolve_server(&ep, &server)?;
            guard_mangled_remote_path(&ep, &id, &remote_path)?;
            let mut req = serde_json::json!({ "sessionId": id, "path": remote_path });
            if let Some(d) = local_dir {
                req["localDir"] = serde_json::Value::String(d);
            }
            let body = http_post(&ep, "/download", req)?;
            print_transfer_started(&body);
            Ok(())
        }
        AgentCmd::Write { server, remote_path, from_file, stdin, content, overwrite } => {
            let id = resolve_server(&ep, &server)?;
            guard_mangled_remote_path(&ep, &id, &remote_path)?;
            // Exactly one content source.
            let bytes: Vec<u8> = match (from_file, stdin, content) {
                (Some(path), false, None) => {
                    std::fs::read(&path).with_context(|| format!("read {path}"))?
                }
                (None, true, None) => {
                    use std::io::Read as _;
                    let mut buf = Vec::new();
                    io::stdin().read_to_end(&mut buf).context("read content from stdin")?;
                    buf
                }
                (None, false, Some(text)) => text.into_bytes(),
                (None, false, None) => {
                    bail!("no content — give one of --from-file <path>, --stdin, or --content <text>")
                }
                _ => bail!("give exactly one of --from-file / --stdin / --content"),
            };
            use base64::Engine as _;
            let b64 = base64::engine::general_purpose::STANDARD.encode(&bytes);
            let body = http_post(
                &ep,
                "/write",
                serde_json::json!({
                    "sessionId": id,
                    "path": &remote_path,
                    "content": b64,
                    "overwrite": overwrite,
                }),
            )?;
            let n = body.get("bytes").and_then(|v| v.as_u64()).unwrap_or(bytes.len() as u64);
            let p = body.get("path").and_then(|v| v.as_str()).unwrap_or(&remote_path);
            println!("wrote {} to {p}", fmt_bytes(n));
            Ok(())
        }
        AgentCmd::Upload { server, local_path, remote_dir, overwrite } => {
            let id = resolve_server(&ep, &server)?;
            guard_mangled_remote_path(&ep, &id, &remote_dir)?;
            let body = http_post(
                &ep,
                "/upload",
                serde_json::json!({
                    "sessionId": id,
                    "localPath": local_path,
                    "remoteDir": remote_dir,
                    "overwrite": overwrite,
                }),
            )?;
            print_transfer_started(&body);
            Ok(())
        }
        AgentCmd::Rm { server, path, recursive } => {
            let id = resolve_server(&ep, &server)?;
            guard_mangled_remote_path(&ep, &id, &path)?;
            http_post(
                &ep,
                "/delete",
                serde_json::json!({ "sessionId": id, "path": &path, "recursive": recursive }),
            )?;
            println!("deleted {path}");
            Ok(())
        }
        AgentCmd::Mv { server, from, to } => {
            let id = resolve_server(&ep, &server)?;
            guard_mangled_remote_path(&ep, &id, &from)?;
            guard_mangled_remote_path(&ep, &id, &to)?;
            http_post(
                &ep,
                "/rename",
                serde_json::json!({ "sessionId": id, "from": &from, "to": &to }),
            )?;
            println!("renamed {from} → {to}");
            Ok(())
        }
        AgentCmd::Mkdir { server, path } => {
            let id = resolve_server(&ep, &server)?;
            guard_mangled_remote_path(&ep, &id, &path)?;
            http_post(&ep, "/mkdir", serde_json::json!({ "sessionId": id, "path": &path }))?;
            println!("created {path}");
            Ok(())
        }
        AgentCmd::UploadDir { server, local_dir, remote_dir, overwrite } => {
            let id = resolve_server(&ep, &server)?;
            guard_mangled_remote_path(&ep, &id, &remote_dir)?;
            let body = http_post(
                &ep,
                "/upload_dir",
                serde_json::json!({
                    "sessionId": id,
                    "localDir": local_dir,
                    "remoteDir": remote_dir,
                    "overwrite": overwrite,
                }),
            )?;
            let ids = body
                .get("transferIds")
                .and_then(|v| v.as_array())
                .cloned()
                .unwrap_or_default();
            let total = body.get("totalBytes").and_then(|v| v.as_u64()).unwrap_or(0);
            let root = body.get("remoteRoot").and_then(|v| v.as_str()).unwrap_or("?");
            println!(
                "queued {} file uploads ({}) → {}",
                ids.len(),
                fmt_bytes(total),
                root
            );
            if let Some(first) = ids.first().and_then(|v| v.as_str()) {
                println!("poll with: ghostftp-cli agent transfer {first}");
            }
            Ok(())
        }
        AgentCmd::Sync { server, local_dir, remote_dir, direction, mirror, dry_run } => {
            let id = resolve_server(&ep, &server)?;
            let dir_word = match direction {
                Dir::Push => "push",
                Dir::Pull => "pull",
            };
            let strategy = if mirror { "mirror" } else { "additive" };
            let body = http_post(
                &ep,
                "/sync",
                serde_json::json!({
                    "sessionId": id,
                    "localDir": local_dir,
                    "remoteDir": remote_dir,
                    "direction": dir_word,
                    "strategy": strategy,
                    "dryRun": dry_run,
                }),
            )?;
            let copies = body.get("copyCount").and_then(|v| v.as_u64()).unwrap_or(0);
            let deletes = body.get("deleteCount").and_then(|v| v.as_u64()).unwrap_or(0);
            let bytes = body.get("totalBytes").and_then(|v| v.as_u64()).unwrap_or(0);
            eprintln!(
                "\n{}\n  copies   {}\n  deletes  {}\n  bytes    {}",
                dim(&format!(
                    "Plan: {local_dir} {} {remote_dir} ({strategy}, {dir_word})",
                    if dry_run { "↔" } else { "→" }
                )),
                copies,
                deletes,
                fmt_bytes(bytes),
            );
            if dry_run {
                if let Some(arr) = body.get("copies").and_then(|v| v.as_array()) {
                    for c in arr.iter().take(50) {
                        let rel = c.get("relative").and_then(|v| v.as_str()).unwrap_or("");
                        let size = c.get("size").and_then(|v| v.as_u64()).unwrap_or(0);
                        let reason = c.get("reason").and_then(|v| v.as_str()).unwrap_or("");
                        eprintln!("  {reason:<11}  {:>10}  {rel}", fmt_bytes(size));
                    }
                    if arr.len() > 50 {
                        eprintln!("  …and {} more copies", arr.len() - 50);
                    }
                }
                if let Some(arr) = body.get("deletes").and_then(|v| v.as_array()) {
                    for d in arr.iter().take(50) {
                        let rel = d.get("relative").and_then(|v| v.as_str()).unwrap_or("");
                        eprintln!("  {}       {rel}", warn("delete"));
                    }
                    if arr.len() > 50 {
                        eprintln!("  …and {} more deletes", arr.len() - 50);
                    }
                }
                return Ok(());
            }
            if body.get("status").and_then(|v| v.as_str()) == Some("in-sync") {
                eprintln!("Already in sync.");
                return Ok(());
            }
            let ids = body
                .get("transferIds")
                .and_then(|v| v.as_array())
                .cloned()
                .unwrap_or_default();
            println!("queued {} transfers; deletes applied", ids.len());
            if let Some(first) = ids.first().and_then(|v| v.as_str()) {
                println!("poll with: ghostftp-cli agent transfer {first}");
            }
            Ok(())
        }
        AgentCmd::Diff { a, b, hash, json } => {
            let (server_a, path_a) = split_agent_diff_side(&a);
            let (server_b, path_b) = split_agent_diff_side(&b);
            let mut req = serde_json::json!({ "pathA": path_a, "pathB": path_b, "hash": hash });
            if let Some(s) = server_a {
                req["sessionA"] = serde_json::Value::String(s);
            }
            if let Some(s) = server_b {
                req["sessionB"] = serde_json::Value::String(s);
            }
            let body = http_post(&ep, "/diff", req)?;
            if json {
                println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
            } else {
                print_agent_diff(&body);
            }
            Ok(())
        }
        AgentCmd::Transfer { transfer_id } => {
            let body = http_post(&ep, "/transfer", serde_json::json!({ "transferId": transfer_id }))?;
            println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
            Ok(())
        }
        AgentCmd::Tail { server, path, lines } => {
            let id = resolve_server(&ep, &server)?;
            let body = http_post(
                &ep,
                "/tail",
                serde_json::json!({ "sessionId": id, "path": path, "lines": lines }),
            )?;
            if let Some(out) = body.get("stdout").and_then(|v| v.as_str()) {
                let mut so = io::stdout();
                so.write_all(out.as_bytes()).ok();
                so.flush().ok();
            }
            if body.get("timedOut").and_then(|v| v.as_bool()).unwrap_or(false) {
                eprintln!("{}", warn("tail stream timed out after 30s"));
            }
            Ok(())
        }
        AgentCmd::Info { server } => {
            let id = resolve_server(&ep, &server)?;
            let body = http_post(&ep, "/info", serde_json::json!({ "sessionId": id }))?;
            println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
            Ok(())
        }
        AgentCmd::History { server, limit } => {
            let mut req = serde_json::json!({ "limit": limit });
            if let Some(s) = server {
                let id = resolve_server(&ep, &s)?;
                req["sessionId"] = serde_json::Value::String(id);
            }
            let body = http_post(&ep, "/history", req)?;
            if let Some(arr) = body.get("history").and_then(|v| v.as_array()) {
                for h in arr {
                    let kind = h.get("kind").and_then(|v| v.as_str()).unwrap_or("");
                    let ok = h.get("ok").and_then(|v| v.as_bool()).unwrap_or(false);
                    let detail = h.get("detail").and_then(|v| v.as_str()).unwrap_or("");
                    let mark = if ok { "ok " } else { "ERR" };
                    println!("[{mark}] {kind:<8} {detail}");
                }
            }
            Ok(())
        }
        AgentCmd::Run { server, dry_run, name } => {
            let id = resolve_server(&ep, &server)?;
            let body = http_post(&ep, "/run", serde_json::json!({ "sessionId": id, "name": name, "dryRun": dry_run }))?;
            if let Some(out) = body.get("stdout").and_then(|v| v.as_str()) {
                let mut so = io::stdout();
                so.write_all(out.as_bytes()).ok();
                so.flush().ok();
            }
            if let Some(e) = body.get("stderr").and_then(|v| v.as_str()) {
                if !e.is_empty() {
                    let mut se = io::stderr();
                    se.write_all(e.as_bytes()).ok();
                    se.flush().ok();
                }
            }
            if body.get("timedOut").and_then(|v| v.as_bool()).unwrap_or(false) {
                eprintln!("{}", warn("command timed out before finishing"));
            }
            let code = body.get("exitCode").and_then(|v| v.as_i64()).unwrap_or(0);
            std::process::exit(code as i32)
        }
        AgentCmd::Commands => {
            let body = http_get(&ep, "/commands")?;
            let cmds = body
                .get("commands")
                .and_then(|v| v.as_array())
                .cloned()
                .unwrap_or_default();
            if cmds.is_empty() {
                eprintln!("No saved commands yet. Add some in Ghost FTP's Agent Bridge panel.");
            }
            for c in &cmds {
                let name = c.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                let command = c.get("command").and_then(|v| v.as_str()).unwrap_or("");
                let desc = c.get("description").and_then(|v| v.as_str()).unwrap_or("");
                if desc.is_empty() {
                    println!("{name:<20} {command}");
                } else {
                    println!("{name:<20} {command}");
                    println!("{:<20} — {desc}", "");
                }
            }
            Ok(())
        }
    }
}

fn cmd_skill(action: SkillCmd) -> Result<()> {
    let ep = read_endpoint()?;
    warn_if_stale(&ep);
    match action {
        SkillCmd::List { json } => {
            let body = http_get(&ep, "/skills")?;
            if json {
                println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
                return Ok(());
            }
            let skills = body
                .get("skills")
                .and_then(|v| v.as_array())
                .cloned()
                .unwrap_or_default();
            if skills.is_empty() {
                eprintln!(
                    "No skills yet. Create one in Ghost FTP's Skills panel, or have the AI propose one."
                );
                return Ok(());
            }
            for s in &skills {
                let name = s.get("name").and_then(|v| v.as_str()).unwrap_or("?");
                let desc = s.get("description").and_then(|v| v.as_str()).unwrap_or("");
                let steps = s.get("steps").and_then(|v| v.as_array()).map(|a| a.len()).unwrap_or(0);
                let params = s
                    .get("params")
                    .and_then(|v| v.as_array())
                    .map(|a| {
                        a.iter()
                            .filter_map(|p| p.get("name").and_then(|n| n.as_str()))
                            .collect::<Vec<_>>()
                            .join(", ")
                    })
                    .unwrap_or_default();
                let params_note = if params.is_empty() {
                    String::new()
                } else {
                    format!("  params: {params}")
                };
                println!(
                    "{name:<20} {steps} step{}  → {}{params_note}",
                    if steps == 1 { "" } else { "s" },
                    describe_skill_targets(s),
                );
                if !desc.is_empty() {
                    println!("{:<20} {}", "", dim(desc));
                }
                if skill_status(s) == "proposed" {
                    println!("{:<20} {}", "", warn("proposal — approve in Ghost FTP before running"));
                }
            }
            Ok(())
        }
        SkillCmd::Run { name, target, param, dry_run, json } => {
            let mut params = serde_json::Map::new();
            for p in &param {
                let Some((k, v)) = p.split_once('=') else {
                    bail!("--param must be key=value (got '{p}')");
                };
                params.insert(k.trim().to_string(), serde_json::Value::String(v.to_string()));
            }
            let mut req = serde_json::json!({
                "name": name,
                "params": serde_json::Value::Object(params),
                "dryRun": dry_run,
            });
            if !target.is_empty() {
                req["targets"] = serde_json::json!(target);
            }
            let body = http_post(&ep, "/skill_run", req)?;
            if json {
                println!("{}", serde_json::to_string_pretty(&body).unwrap_or_default());
                return Ok(());
            }
            if dry_run {
                print_skill_dry_run(&body);
                return Ok(());
            }
            let failed = print_skill_run(&body);
            if failed > 0 {
                std::process::exit(1);
            }
            Ok(())
        }
    }
}

fn skill_status(s: &serde_json::Value) -> &str {
    s.get("status").and_then(|v| v.as_str()).unwrap_or("approved")
}

/// One-line summary of a skill's default target selector for `skill list`.
fn describe_skill_targets(s: &serde_json::Value) -> String {
    let t = s.get("targets");
    if t.and_then(|v| v.get("all")).and_then(|v| v.as_bool()).unwrap_or(false) {
        return "all servers".to_string();
    }
    let sessions = t
        .and_then(|v| v.get("sessions"))
        .and_then(|v| v.as_array())
        .map(|a| a.iter().filter_map(|x| x.as_str()).collect::<Vec<_>>().join(", "))
        .unwrap_or_default();
    if sessions.is_empty() {
        "no default targets".to_string()
    } else {
        sessions
    }
}

/// Print any targets the bridge couldn't run (never silently dropped).
fn print_skill_skipped(body: &serde_json::Value) {
    let skipped = body.get("skipped").and_then(|v| v.as_array()).cloned().unwrap_or_default();
    for s in &skipped {
        let t = s.get("target").and_then(|v| v.as_str()).unwrap_or("?");
        let reason = s.get("reason").and_then(|v| v.as_str()).unwrap_or("");
        eprintln!("{}", warn(&format!("skipped {t}: {reason}")));
    }
}

fn print_skill_dry_run(body: &serde_json::Value) {
    let name = body.get("skill").and_then(|v| v.as_str()).unwrap_or("");
    println!("Dry run — skill \x1b[1m{name}\x1b[0m:");
    let targets = body.get("targets").and_then(|v| v.as_array()).cloned().unwrap_or_default();
    if targets.is_empty() {
        println!("  {}", warn("no runnable targets"));
    }
    for t in &targets {
        let tn = t.get("sessionName").and_then(|v| v.as_str()).unwrap_or("?");
        println!("\n  ▸ \x1b[1m{tn}\x1b[0m");
        let cmds = t.get("commands").and_then(|v| v.as_array()).cloned().unwrap_or_default();
        for (i, c) in cmds.iter().enumerate() {
            println!("    {}. {}", i + 1, c.as_str().unwrap_or(""));
        }
    }
    print_skill_skipped(body);
    if body.get("needsApproval").and_then(|v| v.as_bool()).unwrap_or(false) {
        println!("\n{}", dim("A real run will ask for approval in Ghost FTP."));
    }
}

/// Render a completed skill run; returns the number of failed targets.
fn print_skill_run(body: &serde_json::Value) -> u64 {
    let succeeded = body.get("succeeded").and_then(|v| v.as_u64()).unwrap_or(0);
    let failed = body.get("failed").and_then(|v| v.as_u64()).unwrap_or(0);
    let results = body.get("results").and_then(|v| v.as_array()).cloned().unwrap_or_default();
    for r in &results {
        let tn = r.get("sessionName").and_then(|v| v.as_str()).unwrap_or("?");
        let ok = r.get("ok").and_then(|v| v.as_bool()).unwrap_or(false);
        let marker = if ok { "\x1b[32m✓\x1b[0m" } else { "\x1b[31m✗\x1b[0m" };
        println!("{marker} \x1b[1m{tn}\x1b[0m");
        let steps = r.get("steps").and_then(|v| v.as_array()).cloned().unwrap_or_default();
        for st in &steps {
            let sok = st.get("ok").and_then(|v| v.as_bool()).unwrap_or(false);
            let n = st.get("step").and_then(|v| v.as_u64()).unwrap_or(0);
            let label = st.get("name").and_then(|v| v.as_str()).unwrap_or("");
            if let Some(e) = st.get("error").and_then(|v| v.as_str()) {
                println!("  {} step {n}: {}", if sok { "·" } else { "✗" }, warn(e));
                continue;
            }
            let code = st
                .get("exitCode")
                .and_then(|v| v.as_i64())
                .map(|c| c.to_string())
                .unwrap_or_else(|| "?".to_string());
            let dot = if sok { "\x1b[32m·\x1b[0m" } else { "\x1b[31m✗\x1b[0m" };
            println!("  {dot} step {n} (exit {code}) {}", dim(label));
            if !sok {
                if let Some(se) = st.get("stderr").and_then(|v| v.as_str()) {
                    for line in se.lines().take(10) {
                        println!("      {line}");
                    }
                }
            }
        }
    }
    print_skill_skipped(body);
    println!("\n{succeeded} ok, {failed} failed");
    failed
}

fn print_agent_entries(body: &serde_json::Value) {
    let Some(entries) = body.get("entries").and_then(|v| v.as_array()) else {
        return;
    };
    for e in entries {
        let name = e.get("name").and_then(|v| v.as_str()).unwrap_or("");
        let kind = e
            .get("kind")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_lowercase();
        if kind.contains("dir") {
            println!("{name}/");
        } else {
            let size = e.get("size").and_then(|v| v.as_u64()).unwrap_or(0);
            println!("{name}\t{}", fmt_bytes(size));
        }
    }
}

/// Split a `agent diff` side into (server, path). `local:/x` and a plain path
/// (or a `C:\…` drive) mean the local filesystem (server = None); `name:/x`
/// targets a connected server. Mirrors the bridge, which treats an absent
/// sessionA/B as local.
fn split_agent_diff_side(raw: &str) -> (Option<String>, String) {
    if let Some((name, path)) = raw.split_once(':') {
        if name.eq_ignore_ascii_case("local") {
            return (None, path.to_string());
        }
        // A Windows drive letter (`C:\…`) is a local path, not a server.
        if name.len() == 1 && name.as_bytes()[0].is_ascii_alphabetic() {
            return (None, raw.to_string());
        }
        return (Some(name.to_string()), path.to_string());
    }
    (None, raw.to_string())
}

/// Render the Agent Bridge `/diff` response (only differing entries are sent;
/// the summary counts the rest).
fn print_agent_diff(body: &serde_json::Value) {
    let entries = body.get("entries").and_then(|v| v.as_array()).cloned().unwrap_or_default();
    for e in &entries {
        let rel = e.get("relative").and_then(|v| v.as_str()).unwrap_or("");
        let class = e.get("class").and_then(|v| v.as_str()).unwrap_or("");
        let a_size = e.get("aSize").and_then(|v| v.as_u64());
        let b_size = e.get("bSize").and_then(|v| v.as_u64());
        let (marker, detail) = match class {
            "onlyInA" => ("\x1b[36mA only\x1b[0m", a_size.map(fmt_bytes).unwrap_or_default()),
            "onlyInB" => ("\x1b[35mB only\x1b[0m", b_size.map(fmt_bytes).unwrap_or_default()),
            "different" => {
                let why = match e.get("reason").and_then(|v| v.as_str()) {
                    Some("size") => format!(
                        "size {} → {}",
                        a_size.map(fmt_bytes).unwrap_or_default(),
                        b_size.map(fmt_bytes).unwrap_or_default()
                    ),
                    Some("content") => "content".to_string(),
                    _ => String::new(),
                };
                ("\x1b[33mdiffer\x1b[0m", why)
            }
            _ => ("?", String::new()),
        };
        let hash_note = if e.get("hashError").is_some() {
            dim("  [hash unavailable]")
        } else {
            String::new()
        };
        if detail.is_empty() {
            println!("{marker}  {rel}{hash_note}");
        } else {
            println!("{marker}  {rel}  {}{hash_note}", dim(&detail));
        }
    }
    if let Some(s) = body.get("summary") {
        let g = |k: &str| s.get(k).and_then(|v| v.as_u64()).unwrap_or(0);
        let hashed = body.get("hashed").and_then(|v| v.as_bool()).unwrap_or(false);
        eprintln!(
            "\n{}",
            dim(&format!(
                "{} only in A · {} only in B · {} differ · {} same{}",
                g("onlyInA"),
                g("onlyInB"),
                g("different"),
                g("same"),
                if hashed { " (hashed)" } else { "" }
            ))
        );
    }
    if body.get("listTruncated").and_then(|v| v.as_bool()).unwrap_or(false) {
        eprintln!("{}", warn("entry list truncated — use --json for the full result"));
    }
}

/// Render a `/job` poll: stream out/err, then a status line.
fn print_job(body: &serde_json::Value) {
    if let Some(err) = body.get("error").and_then(|v| v.as_str()) {
        eprintln!("{}", warn(err));
        return;
    }
    if let Some(out) = body.get("stdout").and_then(|v| v.as_str()) {
        let mut so = io::stdout();
        so.write_all(out.as_bytes()).ok();
        so.flush().ok();
    }
    if let Some(e) = body.get("stderr").and_then(|v| v.as_str()) {
        if !e.is_empty() {
            let mut se = io::stderr();
            se.write_all(e.as_bytes()).ok();
            se.flush().ok();
        }
    }
    let running = body.get("running").and_then(|v| v.as_bool()).unwrap_or(false);
    if running {
        eprintln!("\n{}", dim("job still running — poll again"));
    } else {
        let code = body
            .get("exitCode")
            .and_then(|v| v.as_i64())
            .map(|c| c.to_string())
            .unwrap_or_else(|| "?".into());
        eprintln!("\n{}", dim(&format!("job finished (exit {code})")));
    }
}

fn print_transfer_started(body: &serde_json::Value) {
    let id = body.get("transferId").and_then(|v| v.as_str()).unwrap_or("?");
    println!("transfer started: {id}");
    if let Some(dir) = body.get("localDir").and_then(|v| v.as_str()) {
        println!("  → {dir}");
    }
    println!("poll with: ghostftp-cli agent transfer {id}");
}

// ---- Transfer helpers --------------------------------------------------

/// Upload `local_path` into `remote_parent`, naming the result `name`. Uses
/// each backend's native streaming path so we don't pull the whole file into
/// memory.
async fn upload_file(
    session: &Session,
    local_path: &str,
    remote_parent: &str,
    name: &str,
) -> Result<()> {
    let local_size = tokio::fs::metadata(local_path)
        .await
        .with_context(|| format!("stat {local_path}"))?
        .len();
    let remote_path = join_remote(remote_parent, name);
    let bar = make_bar(local_size, name);

    match session {
        Session::Ssh(ssh) => {
            use tokio::io::{AsyncReadExt, AsyncWriteExt};
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            let mut remote = sftp
                .create(&remote_path)
                .await
                .with_context(|| format!("create remote {remote_path}"))?;
            let mut local = tokio::fs::File::open(local_path).await?;
            let mut buf = vec![0u8; 64 * 1024];
            loop {
                let n = local.read(&mut buf).await?;
                if n == 0 {
                    break;
                }
                remote.write_all(&buf[..n]).await?;
                bar.inc(n as u64);
            }
            remote.flush().await?;
        }
        Session::Ftp(ftp) => upload_ftp(ftp.clone(), local_path.to_string(), remote_path.clone()).await?,
        Session::Object(obj) => {
            upload_object(obj.clone(), local_path, &remote_path, &bar).await?
        }
        Session::Webdav(_)
        | Session::Http(_)
        | Session::Dropbox(_)
        | Session::OneDrive(_)
        | Session::GDrive(_)
        | Session::Box(_)
        | Session::Shopify(_)
        | Session::HubSpot(_)
        | Session::Dynamics(_) => {
            anyhow::bail!("uploads to this connection type are not yet supported in the CLI")
        }
        Session::Agent(_) => anyhow::bail!("ghostftp-agent connections are not supported in the CLI"),
    }

    bar.finish_with_message(format!("uploaded {name}"));
    Ok(())
}

async fn upload_ftp(
    ftp: Arc<FtpSession>,
    local_path: String,
    remote_path: String,
) -> Result<()> {
    ftp.with_stream(move |stream| {
        let file = std::fs::File::open(&local_path)
            .with_context(|| format!("open {local_path}"))?;
        let mut reader = std::io::BufReader::new(file);
        stream.put_from_reader(&remote_path, &mut reader)?;
        Ok(())
    })
    .await
}

async fn upload_object(
    obj: Arc<ObjectSession>,
    local_path: &str,
    remote_path: &str,
    bar: &ProgressBar,
) -> Result<()> {
    use tokio::io::AsyncReadExt;
    let key = remote_path.trim_start_matches('/');
    let p = object_store::path::Path::from(key);
    let mut file = tokio::fs::File::open(local_path).await?;
    let meta = file.metadata().await?;
    if meta.len() <= 16 * 1024 * 1024 {
        let mut buf = Vec::with_capacity(meta.len() as usize);
        file.read_to_end(&mut buf).await?;
        obj.store
            .put(&p, bytes::Bytes::from(buf).into())
            .await
            .with_context(|| format!("s3 put {key}"))?;
        bar.inc(meta.len());
    } else {
        let mut upload = obj.store.put_multipart(&p).await?;
        let mut buf = vec![0u8; 8 * 1024 * 1024];
        loop {
            let mut filled = 0;
            while filled < buf.len() {
                let n = file.read(&mut buf[filled..]).await?;
                if n == 0 {
                    break;
                }
                filled += n;
            }
            if filled == 0 {
                break;
            }
            let chunk = bytes::Bytes::copy_from_slice(&buf[..filled]);
            upload.put_part(chunk.into()).await?;
            bar.inc(filled as u64);
            if filled < buf.len() {
                break;
            }
        }
        upload.complete().await?;
    }
    Ok(())
}

async fn download_file(session: &Session, remote_path: &str, local_dir: &str) -> Result<()> {
    let local_dir_path = PathBuf::from(local_dir);
    tokio::fs::create_dir_all(&local_dir_path).await.ok();
    let name = basename(remote_path);
    let final_path = local_dir_path.join(&name);

    let bar = make_bar(0, &name);

    match session {
        Session::Ssh(ssh) => {
            use tokio::io::{AsyncReadExt, AsyncWriteExt};
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            let mut remote = sftp
                .open(remote_path)
                .await
                .with_context(|| format!("open {remote_path}"))?;
            let mut local = tokio::fs::File::create(&final_path).await?;
            let mut buf = vec![0u8; 64 * 1024];
            loop {
                let n = remote.read(&mut buf).await?;
                if n == 0 {
                    break;
                }
                local.write_all(&buf[..n]).await?;
                bar.inc(n as u64);
            }
            local.flush().await?;
        }
        Session::Ftp(ftp) => {
            let ftp = ftp.clone();
            let remote = remote_path.to_string();
            let final_path_clone = final_path.clone();
            ftp.with_stream(move |stream| {
                let file = std::fs::File::create(&final_path_clone)
                    .with_context(|| format!("create {}", final_path_clone.display()))?;
                stream.retr_to_writer(&remote, std::io::BufWriter::new(file))?;
                Ok(())
            })
            .await?;
        }
        Session::Object(obj) => {
            use tokio::io::AsyncWriteExt;
            let key = remote_path.trim_start_matches('/');
            let p = object_store::path::Path::from(key);
            let get = obj.store.get(&p).await?;
            let mut file = tokio::fs::File::create(&final_path).await?;
            let mut stream = get.into_stream();
            while let Some(chunk) = stream.next().await {
                let chunk = chunk?;
                file.write_all(&chunk).await?;
                bar.inc(chunk.len() as u64);
            }
            file.flush().await?;
        }
        Session::Webdav(_)
        | Session::Http(_)
        | Session::Dropbox(_)
        | Session::OneDrive(_)
        | Session::GDrive(_)
        | Session::Box(_)
        | Session::Shopify(_)
        | Session::HubSpot(_)
        | Session::Dynamics(_) => {
            anyhow::bail!("downloads from this connection type are not yet supported in the CLI")
        }
        Session::Agent(_) => anyhow::bail!("ghostftp-agent connections are not supported in the CLI"),
    }

    bar.finish_with_message(format!("downloaded {name}"));
    Ok(())
}

// ---- Misc helpers ------------------------------------------------------

fn make_bar(total: u64, name: &str) -> ProgressBar {
    let bar = if total > 0 {
        let b = ProgressBar::new(total);
        b.set_style(
            ProgressStyle::with_template(
                "{bar:30.cyan/blue} {bytes:>10} / {total_bytes:>10}  {msg}",
            )
            .unwrap(),
        );
        b
    } else {
        let b = ProgressBar::new_spinner();
        b.set_style(ProgressStyle::with_template("{spinner} {bytes:>10}  {msg}").unwrap());
        b
    };
    bar.set_message(name.to_string());
    bar
}

fn basename(path: &str) -> String {
    path.rsplit(|c| c == '/' || c == '\\')
        .next()
        .unwrap_or(path)
        .to_string()
}

fn parent_of(path: &str) -> String {
    match path.rfind(|c: char| c == '/' || c == '\\') {
        Some(0) => "/".to_string(),
        Some(i) => path[..i].to_string(),
        None => ".".to_string(),
    }
}

fn join_remote(parent: &str, name: &str) -> String {
    if parent.ends_with('/') || parent.ends_with('\\') {
        format!("{parent}{name}")
    } else if parent.is_empty() {
        name.to_string()
    } else {
        format!("{parent}/{name}")
    }
}

fn reason_label(r: &ghostftp_lib::sync::SyncReason) -> &'static str {
    match r {
        ghostftp_lib::sync::SyncReason::Missing => "new   ",
        ghostftp_lib::sync::SyncReason::Newer => "newer ",
        ghostftp_lib::sync::SyncReason::SizeChanged => "size  ",
        ghostftp_lib::sync::SyncReason::Edited => "edited",
    }
}

fn fmt_bytes(n: u64) -> String {
    if n < 1024 {
        format!("{n} B")
    } else if n < 1024 * 1024 {
        format!("{:.1} KB", n as f64 / 1024.0)
    } else if n < 1024 * 1024 * 1024 {
        format!("{:.1} MB", n as f64 / (1024.0 * 1024.0))
    } else {
        format!("{:.2} GB", n as f64 / (1024.0 * 1024.0 * 1024.0))
    }
}

#[cfg(test)]
mod tests {
    use super::{
        asset_url, check_mangled_remote_path, is_windows_drive_path, parse_semver, swap_binary_at,
    };

    #[test]
    fn asset_url_points_at_the_release() {
        assert_eq!(
            asset_url(None, "ghostftp-cli-windows-x86_64.exe"),
            "https://ghostftp.com/downloads/ghostftp-cli-windows-x86_64.exe"
        );
        assert_eq!(
            asset_url(Some("v1.4.0"), "ghostftp-cli-linux-x86_64"),
            "https://ghostftp.com/downloads/v1.4.0/ghostftp-cli-linux-x86_64"
        );
    }

    // Plan 10 Phase 0b: the swap must replace the target's bytes even though (on
    // Windows) a running exe can't be overwritten in place — exercised here on a
    // temp file to prove the rename-aside/move-in mechanism.
    #[test]
    fn swap_binary_replaces_target_bytes() {
        let dir = tempdir_unique();
        let target = dir.join("ghostftp-cli.exe");
        std::fs::write(&target, b"OLD BINARY").unwrap();
        swap_binary_at(&target, b"NEW BINARY BYTES").unwrap();
        assert_eq!(std::fs::read(&target).unwrap(), b"NEW BINARY BYTES");
        // No stray temp file left behind.
        assert!(!dir.join("ghostftp-cli.exe.new").exists());
        let _ = std::fs::remove_dir_all(&dir);
    }

    /// A unique temp dir without pulling in the tempfile crate (Date/rand are
    /// fine in a test binary, unlike the workflow sandbox).
    fn tempdir_unique() -> std::path::PathBuf {
        let mut p = std::env::temp_dir();
        let uniq = format!(
            "ghostftp-cli-test-{}-{:?}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        );
        p.push(uniq);
        std::fs::create_dir_all(&p).unwrap();
        p
    }

    #[test]
    fn mangled_path_rejected_only_on_non_windows_target() {
        // Drive-prefixed path against a POSIX server → rejected with the hint.
        let err = check_mangled_remote_path("C:/Program Files/Git/var/www", false)
            .expect_err("should reject a mangled path on a POSIX target");
        let msg = format!("{err}");
        assert!(msg.contains("MSYS_NO_PATHCONV=1"));
        assert!(msg.contains("agent write"));
        // Same path against a genuine Windows machine → allowed (real drive path).
        assert!(check_mangled_remote_path("C:/Users/me/app", true).is_ok());
        // A normal POSIX remote path is always fine, target OS regardless.
        assert!(check_mangled_remote_path("/var/www/html", false).is_ok());
        assert!(check_mangled_remote_path("//var/www", false).is_ok());
    }

    #[test]
    fn collapsed_windows_path_is_rejected_on_any_target() {
        // `C:\Users\User` unquoted in bash arrives like this; Windows would
        // resolve it drive-relative and drop the file in the wrong directory.
        for target_is_windows in [true, false] {
            let err = check_mangled_remote_path("C:UsersUser", target_is_windows)
                .expect_err("collapsed drive path should be rejected");
            assert!(format!("{err}").contains("missing its path separators"));
        }
        // Properly separated drive paths are still fine on a Windows target.
        assert!(check_mangled_remote_path(r"C:\Users\User", true).is_ok());
        assert!(check_mangled_remote_path("C:/Users/User", true).is_ok());
        // A bare relative path has no drive letter to collapse.
        assert!(check_mangled_remote_path("public_html/wp-content", false).is_ok());
    }

    #[test]
    fn detects_windows_drive_paths() {
        // MSYS-mangled remote paths (Plan 10 Phase 3) — these trip the guard.
        assert!(is_windows_drive_path("C:/Program Files/Git/var/www"));
        assert!(is_windows_drive_path(r"C:\Users\me"));
        assert!(is_windows_drive_path("D:/data"));
        // Real remote POSIX paths — never flagged.
        assert!(!is_windows_drive_path("/var/www/html"));
        assert!(!is_windows_drive_path("//var/www")); // the leading-// escape
        assert!(!is_windows_drive_path("./rel"));
        assert!(!is_windows_drive_path("home/user"));
        // A bare drive letter with no separator isn't a path prefix.
        assert!(!is_windows_drive_path("C:"));
    }

    #[test]
    fn semver_parses_and_orders() {
        assert_eq!(parse_semver("1.3.19"), Some((1, 3, 19)));
        assert_eq!(parse_semver("1.4.0"), Some((1, 4, 0)));
        // pre-release / build metadata is ignored down to the core triple.
        assert_eq!(parse_semver("2.0.0-rc1"), Some((2, 0, 0)));
        assert_eq!(parse_semver("2.0.0+build.7"), Some((2, 0, 0)));
        // short forms default missing components to 0.
        assert_eq!(parse_semver("3"), Some((3, 0, 0)));
        assert_eq!(parse_semver("3.2"), Some((3, 2, 0)));
        // garbage suppresses the check rather than firing a bogus warning.
        assert_eq!(parse_semver("nightly"), None);
    }

    #[test]
    fn stale_when_cli_older_than_app() {
        // The staleness warning fires iff cli < app (Plan 10 Phase 0a).
        assert!(parse_semver("1.3.10") < parse_semver("1.3.19"));
        assert!(parse_semver("1.2.99") < parse_semver("1.3.0"));
        // Same or newer is NOT stale.
        assert!(!(parse_semver("1.3.19") < parse_semver("1.3.19")));
        assert!(!(parse_semver("1.4.0") < parse_semver("1.3.19")));
    }
}
