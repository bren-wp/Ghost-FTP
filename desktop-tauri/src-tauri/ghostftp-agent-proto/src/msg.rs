//! The request/response message set exchanged over a [`crate::SecureChannel`].
//!
//! These mirror the operations Ghost FTP already performs against a remote server —
//! browse, read, write, transfer, exec — so the daemon can satisfy the app's
//! `RemoteFs`, transfer, and bridge-exec paths without a bespoke API per call
//! site. Field names are camelCase to match the rest of Ghost FTP's JSON.

use serde::{Deserialize, Serialize};

/// Protocol version, bumped on a breaking change to these types. Exchanged in
/// the [`Hello`] so a mismatched daemon/controller fails loudly instead of
/// misparsing a frame.
pub const PROTOCOL_VERSION: u32 = 1;

/// mDNS service type the daemon advertises and the controller browses.
pub const SERVICE_TYPE: &str = "_ghostftp-agent._tcp.local.";

/// Bounds a caller-supplied exec timeout (`Exec.timeoutMs`) is clamped to. Shared
/// so the daemon and the Agent Bridge agree on the ceiling — otherwise a
/// `--timeout-ms 900000` accepted by the bridge would be silently re-capped to a
/// lower value by the daemon (the drift Plan 10 Phase 0e closes). The floor keeps
/// a typo like `1` from insta-killing a command; the ceiling (15 min) keeps a
/// runaway command from parking forever.
pub const EXEC_TIMEOUT_MS_MIN: u64 = 1_000;
pub const EXEC_TIMEOUT_MS_MAX: u64 = 900_000;

/// Kind of a directory entry — mirrors Ghost FTP's `remotefs::FileKind`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FileKind {
    File,
    Directory,
    Symlink,
    Other,
}

/// One directory entry — mirrors Ghost FTP's `remotefs::DirEntry` so `AgentFs` can
/// hand these straight back to the UI.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DirEntry {
    pub name: String,
    pub path: String,
    pub kind: FileKind,
    pub size: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub modified: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mode: Option<u32>,
}

/// Facts about the controlled machine, shown in Ghost FTP's UI + handed to the AI so
/// it knows which OS's commands to run.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SystemInfo {
    pub os: String, // "windows" | "linux"
    pub hostname: String,
    pub arch: String,
    pub shell: String, // the shell exec uses: "powershell" | "sh" | ...
    pub username: String,
    pub home_dir: String,
    pub agentd_version: String,
}

/// Sent by the controller as the very first (post-handshake) message so the
/// daemon can reject an incompatible protocol version before doing any work.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Hello {
    pub protocol_version: u32,
    /// Human label for the controller (its hostname), for the daemon's audit log.
    pub client_name: String,
}

/// A request from the controller to the daemon.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "op", rename_all = "camelCase")]
pub enum Request {
    Ping,
    SystemInfo,
    ListDir {
        path: String,
    },
    Stat {
        path: String,
    },
    /// Read a whole (text) file, capped at `maxBytes`.
    ReadFile {
        path: String,
        max_bytes: u64,
    },
    /// Ranged read for streaming a download. `len == 0` reads to EOF from
    /// `offset` up to the daemon's per-chunk cap.
    ReadChunk {
        path: String,
        offset: u64,
        len: u64,
    },
    /// Ranged write for streaming an upload. `truncate` opens/creates the file
    /// fresh (first chunk); `done` closes it. `data` is base64.
    WriteChunk {
        path: String,
        offset: u64,
        data: String,
        truncate: bool,
        done: bool,
    },
    /// Compute the content-defined-chunking signature of a file (delta sync).
    /// Read-only. An older daemon doesn't know this op and fails the request;
    /// the controller then falls back to a whole-file copy.
    Signature {
        path: String,
    },
    /// Reassemble `dest` from `basis` (the old file, when one exists) + `patch`
    /// (the literal bytes already uploaded) following `recipe`, verify the
    /// result against `expected_hash`, and atomically rename it over `dest`.
    /// Write-gated like [`Request::WriteChunk`].
    DeltaAssemble {
        basis: Option<String>,
        patch: String,
        recipe: Vec<crate::delta::RecipeOp>,
        dest: String,
        expected_hash: String,
    },
    Delete {
        path: String,
        recursive: bool,
    },
    CreateDir {
        path: String,
    },
    Rename {
        from: String,
        to: String,
    },
    Chmod {
        path: String,
        mode: u32,
    },
    /// Run a native shell command, bounded by time and output size.
    Exec {
        command: String,
        timeout_ms: u64,
        max_bytes: u64,
    },
    /// Launch `command` as a **detached background job** and return at once with
    /// its `jobId` (poll with [`Request::ExecPoll`]). The daemon-target analogue
    /// of the SSH `~/.ghostftp/jobs` dir — retires the `nohup … & ; tail -f log` loop
    /// for multi-minute work that would blow the exec timeout (Plan 10 Phase 4).
    /// The caller (Agent Bridge) supplies the `job_id` so id generation lives in
    /// one place. These three ops are **additive** — a pre-Plan-10 daemon doesn't
    /// know them and will drop the request; the controller degrades to a clear
    /// "update the daemon" message. Gated like [`Request::Exec`].
    ExecStart {
        job_id: String,
        command: String,
        max_bytes: u64,
    },
    /// Poll a detached job's captured (capped) stdout/stderr and status.
    ExecPoll {
        job_id: String,
    },
    /// Kill a running detached job (best-effort). Replies [`Response::Ok`].
    ExecKill {
        job_id: String,
    },
}

/// The daemon's reply. `Error` carries a human-readable message for any request.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "result", rename_all = "camelCase")]
pub enum Response {
    Pong,
    SystemInfo(SystemInfo),
    Dir {
        entries: Vec<DirEntry>,
    },
    Stat {
        entry: DirEntry,
    },
    File {
        /// base64 of the (possibly truncated) file bytes.
        data: String,
        bytes: u64,
        truncated: bool,
    },
    Chunk {
        /// base64 of the bytes read; empty at EOF.
        data: String,
        /// True when this read reached end of file.
        eof: bool,
    },
    Written {
        bytes: u64,
    },
    /// A file's chunk signature (reply to [`Request::Signature`]).
    Signature {
        size: u64,
        min: u32,
        avg: u32,
        max: u32,
        chunks: Vec<crate::delta::ChunkEntry>,
        whole_hash: String,
    },
    /// A delta reassembly finished (reply to [`Request::DeltaAssemble`]).
    DeltaDone {
        bytes_reused: u64,
        bytes_written: u64,
    },
    Ok,
    Exec {
        stdout: String,
        stderr: String,
        exit_code: Option<i32>,
        truncated: bool,
        timed_out: bool,
    },
    /// A detached job was launched (reply to [`Request::ExecStart`]).
    ExecStarted {
        job_id: String,
    },
    /// A detached job's current state (reply to [`Request::ExecPoll`]): `running`
    /// until it exits, then `exit_code`; `stdout`/`stderr` are the capped capture
    /// so far. `not_found` means the id is unknown (never started, or pruned).
    ExecStatus {
        running: bool,
        exit_code: Option<i32>,
        stdout: String,
        stderr: String,
        truncated: bool,
        #[serde(default)]
        not_found: bool,
    },
    /// The daemon refused or failed the request. `denied` distinguishes a policy
    /// refusal (the machine's owner disallowed this class of op) from an
    /// operational failure, so the controller can phrase it correctly.
    Error {
        message: String,
        #[serde(default)]
        denied: bool,
    },
}

impl Response {
    /// Convenience for daemon handlers: an operational failure.
    pub fn error(message: impl Into<String>) -> Self {
        Response::Error {
            message: message.into(),
            denied: false,
        }
    }

    /// Convenience for daemon handlers: a policy refusal.
    pub fn denied(message: impl Into<String>) -> Self {
        Response::Error {
            message: message.into(),
            denied: true,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::delta::{ChunkEntry, RecipeOp};

    #[test]
    fn delta_variants_serde_round_trip() {
        let req = Request::Signature {
            path: "/tmp/f.bin".to_string(),
        };
        let json = serde_json::to_string(&req).unwrap();
        assert!(json.contains("\"op\":\"signature\""), "{json}");
        match serde_json::from_str::<Request>(&json).unwrap() {
            Request::Signature { path } => assert_eq!(path, "/tmp/f.bin"),
            other => panic!("wrong variant: {other:?}"),
        }

        let req = Request::DeltaAssemble {
            basis: Some("/tmp/f.bin".to_string()),
            patch: "/tmp/f.bin.ghostftp-patch-1".to_string(),
            recipe: vec![
                RecipeOp::Copy {
                    basis_offset: 0,
                    len: 65_536,
                },
                RecipeOp::Literal {
                    patch_offset: 0,
                    len: 1_024,
                },
            ],
            dest: "/tmp/f.bin".to_string(),
            expected_hash: "aGFzaA==".to_string(),
        };
        let json = serde_json::to_string(&req).unwrap();
        assert!(json.contains("\"op\":\"deltaAssemble\""), "{json}");
        // Enum-level rename_all camelCases the variant tag only; struct-variant
        // fields stay snake_case — same as the existing Exec.timeout_ms etc.
        assert!(json.contains("\"expected_hash\""), "{json}");
        match serde_json::from_str::<Request>(&json).unwrap() {
            Request::DeltaAssemble {
                basis,
                patch,
                recipe,
                dest,
                expected_hash,
            } => {
                assert_eq!(basis.as_deref(), Some("/tmp/f.bin"));
                assert_eq!(patch, "/tmp/f.bin.ghostftp-patch-1");
                assert_eq!(recipe.len(), 2);
                assert_eq!(dest, "/tmp/f.bin");
                assert_eq!(expected_hash, "aGFzaA==");
            }
            other => panic!("wrong variant: {other:?}"),
        }

        let resp = Response::Signature {
            size: 3,
            min: crate::delta::CHUNK_MIN,
            avg: crate::delta::CHUNK_AVG,
            max: crate::delta::CHUNK_MAX,
            chunks: vec![ChunkEntry {
                offset: 0,
                len: 3,
                hash: "aABiAGM=".to_string(),
            }],
            whole_hash: "d2hvbGU=".to_string(),
        };
        let json = serde_json::to_string(&resp).unwrap();
        assert!(json.contains("\"result\":\"signature\""), "{json}");
        assert!(json.contains("\"whole_hash\""), "{json}");
        match serde_json::from_str::<Response>(&json).unwrap() {
            Response::Signature {
                size,
                chunks,
                whole_hash,
                ..
            } => {
                assert_eq!(size, 3);
                assert_eq!(chunks.len(), 1);
                assert_eq!(whole_hash, "d2hvbGU=");
            }
            other => panic!("wrong variant: {other:?}"),
        }

        let resp = Response::DeltaDone {
            bytes_reused: 65_536,
            bytes_written: 66_560,
        };
        let json = serde_json::to_string(&resp).unwrap();
        assert!(json.contains("\"result\":\"deltaDone\""), "{json}");
        assert!(json.contains("\"bytes_reused\":65536"), "{json}");
        match serde_json::from_str::<Response>(&json).unwrap() {
            Response::DeltaDone {
                bytes_reused,
                bytes_written,
            } => {
                assert_eq!(bytes_reused, 65_536);
                assert_eq!(bytes_written, 66_560);
            }
            other => panic!("wrong variant: {other:?}"),
        }
    }

    /// Compat contract (delta-sync Plan risk #2): a pre-delta daemon's `Request`
    /// enum has no `signature`/`deltaAssemble` variants, so its tagged-enum
    /// deserialize of these ops FAILS — its request loop then drops the
    /// connection, the controller's `AgentSession::request` re-dials once and
    /// still errors, and the transfer layer falls back to a whole-file copy.
    /// This pins the "unknown op ⇒ clean request failure" half of that chain.
    #[test]
    fn old_daemon_fails_to_parse_delta_ops() {
        /// The Request shape a pre-delta-sync daemon was built with (trimmed to
        /// a representative subset — the point is: no delta variants).
        #[allow(dead_code)]
        #[derive(Debug, Deserialize)]
        #[serde(tag = "op", rename_all = "camelCase")]
        enum OldRequest {
            Ping,
            ReadChunk {
                path: String,
                offset: u64,
                len: u64,
            },
            WriteChunk {
                path: String,
                offset: u64,
                data: String,
                truncate: bool,
                done: bool,
            },
        }

        // Old ops still parse under the old shape (the shared variants are
        // unchanged, so old controller ↔ new daemon keeps working too).
        let legacy = serde_json::to_string(&Request::ReadChunk {
            path: "/x".to_string(),
            offset: 0,
            len: 0,
        })
        .unwrap();
        assert!(serde_json::from_str::<OldRequest>(&legacy).is_ok());

        // The new ops do NOT parse under the old shape — one request errors,
        // which is what triggers the controller's whole-file fallback.
        for req in [
            Request::Signature {
                path: "/x".to_string(),
            },
            Request::DeltaAssemble {
                basis: None,
                patch: "/p".to_string(),
                recipe: vec![],
                dest: "/d".to_string(),
                expected_hash: "aGFzaA==".to_string(),
            },
        ] {
            let json = serde_json::to_string(&req).unwrap();
            assert!(
                serde_json::from_str::<OldRequest>(&json).is_err(),
                "old daemon shape must reject {json}"
            );
        }
    }
}
