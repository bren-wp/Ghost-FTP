//! Structured errors across the IPC boundary (Plan 12 Phase 3).
//!
//! Commands used to return `Result<T, String>`, so the frontend had only the
//! `Display` text to work with — it couldn't tell an auth failure from a
//! missing file without regexing English. [`GhostFTPError`] serializes as
//! `{ "kind": "...", "message": "..." }` so the frontend pattern-matches on a
//! stable discriminant and picks an actionable toast, while still showing the
//! human message.
//!
//! Classification happens once, here in Rust (the *backend* may inspect its own
//! error text; the point of the plan is that the *frontend* never has to).
//! Migration is command-group by command-group: a command that still returns
//! `String` simply arrives as an unstructured error the frontend treats as
//! generic — nothing breaks.

use serde::Serialize;

/// A stable, machine-readable error category. Keep these broad — the frontend
/// maps each to one actionable toast; over-splitting just fragments the UX.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "kebab-case")]
pub enum ErrorKind {
    /// Authentication / authorization failed — reconnect or re-enter creds.
    Auth,
    /// The caller lacks permission for the path/operation.
    Permission,
    /// The target file/dir/resource doesn't exist.
    NotFound,
    /// A connection/DNS/socket problem reaching the remote.
    Network,
    /// The remote took too long.
    Timeout,
    /// The backend doesn't support this operation (capability gap).
    Unsupported,
    /// The target already exists / a write conflicts with existing state.
    Conflict,
    /// The caller supplied an invalid path/value for the requested operation.
    InvalidInput,
    /// Anything not usefully categorised.
    Other,
}

/// The wire shape every migrated command emits on failure.
#[derive(Debug, Clone, Serialize)]
pub struct GhostFTPError {
    pub kind: ErrorKind,
    pub message: String,
}

impl GhostFTPError {
    pub fn new(kind: ErrorKind, message: impl Into<String>) -> Self {
        Self {
            kind,
            message: message.into(),
        }
    }

    /// An uncategorised error with an explicit message.
    pub fn other(message: impl Into<String>) -> Self {
        Self::new(ErrorKind::Other, message)
    }
}

impl std::fmt::Display for GhostFTPError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for GhostFTPError {}

impl From<anyhow::Error> for GhostFTPError {
    fn from(e: anyhow::Error) -> Self {
        let kind = classify_chain(&e, &e.to_string());
        GhostFTPError {
            kind,
            message: e.to_string(),
        }
    }
}

impl From<String> for GhostFTPError {
    fn from(s: String) -> Self {
        let kind = classify_message(&s);
        GhostFTPError { kind, message: s }
    }
}

impl From<&str> for GhostFTPError {
    fn from(s: &str) -> Self {
        GhostFTPError::from(s.to_string())
    }
}

/// Classify an `anyhow` error: prefer a concrete `io::Error` kind anywhere in the
/// cause chain, then fall back to message keywords.
fn classify_chain(e: &anyhow::Error, message: &str) -> ErrorKind {
    for cause in e.chain() {
        if let Some(io) = cause.downcast_ref::<std::io::Error>() {
            if let Some(k) = classify_io(io.kind()) {
                return k;
            }
        }
    }
    classify_message(message)
}

fn classify_io(kind: std::io::ErrorKind) -> Option<ErrorKind> {
    use std::io::ErrorKind as K;
    Some(match kind {
        K::NotFound => ErrorKind::NotFound,
        K::PermissionDenied => ErrorKind::Permission,
        K::TimedOut => ErrorKind::Timeout,
        K::AlreadyExists => ErrorKind::Conflict,
        K::ConnectionRefused
        | K::ConnectionReset
        | K::ConnectionAborted
        | K::NotConnected
        | K::AddrNotAvailable
        | K::BrokenPipe => ErrorKind::Network,
        K::Unsupported => ErrorKind::Unsupported,
        _ => return None,
    })
}

/// Keyword heuristics over an error message. Backend-side only — the frontend
/// gets the resulting `kind` and never repeats this matching.
pub(crate) fn classify_message(message: &str) -> ErrorKind {
    let s = message.to_ascii_lowercase();
    let has = |needles: &[&str]| needles.iter().any(|n| s.contains(n));

    // Order matters: check the more specific categories first.
    if has(&[
        "permission denied",
        "access is denied",
        "eacces",
        "forbidden",
        "403",
    ]) {
        return ErrorKind::Permission;
    }
    if has(&[
        "authentication",
        "auth failed",
        "unauthorized",
        "401",
        "invalid credential",
        "bad password",
        "incorrect password",
        "login failed",
        "api key",
    ]) {
        return ErrorKind::Auth;
    }
    if has(&[
        "no such file",
        "not found",
        "does not exist",
        "enoent",
        "404",
    ]) {
        return ErrorKind::NotFound;
    }
    if has(&["timed out", "timeout"]) {
        return ErrorKind::Timeout;
    }
    if has(&[
        "connection refused",
        "connection reset",
        "connection closed",
        "broken pipe",
        "unreachable",
        "dns",
        "failed to resolve",
        "network",
        "no route to host",
    ]) {
        return ErrorKind::Network;
    }
    if has(&["already exists", "conflict", "eexist"]) {
        return ErrorKind::Conflict;
    }
    if has(&[
        "unsupported",
        "not supported",
        "capability",
        "read-only",
        "not implemented",
    ]) {
        return ErrorKind::Unsupported;
    }
    ErrorKind::Other
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn serializes_kebab_case() {
        let e = GhostFTPError::new(ErrorKind::NotFound, "gone");
        let json = serde_json::to_string(&e).unwrap();
        assert_eq!(json, r#"{"kind":"not-found","message":"gone"}"#);
    }

    #[test]
    fn classifies_message_keywords() {
        assert_eq!(
            classify_message("Permission denied (os error 13)"),
            ErrorKind::Permission
        );
        assert_eq!(
            classify_message("No such file or directory"),
            ErrorKind::NotFound
        );
        assert_eq!(
            classify_message("Authentication failed for user"),
            ErrorKind::Auth
        );
        assert_eq!(classify_message("operation timed out"), ErrorKind::Timeout);
        assert_eq!(classify_message("Connection refused"), ErrorKind::Network);
        assert_eq!(classify_message("file already exists"), ErrorKind::Conflict);
        assert_eq!(
            classify_message("chmod is unsupported on this backend"),
            ErrorKind::Unsupported
        );
        assert_eq!(
            classify_message("something weird happened"),
            ErrorKind::Other
        );
    }

    #[test]
    fn classifies_io_error_kind_from_anyhow() {
        let io = std::io::Error::new(std::io::ErrorKind::PermissionDenied, "nope");
        let e: GhostFTPError = anyhow::Error::new(io).into();
        assert_eq!(e.kind, ErrorKind::Permission);

        let io = std::io::Error::new(std::io::ErrorKind::NotFound, "missing");
        let e: GhostFTPError = anyhow::Error::new(io).context("while listing").into();
        assert_eq!(e.kind, ErrorKind::NotFound);
    }

    #[test]
    fn from_string_classifies() {
        let e: GhostFTPError = "session abc not found".to_string().into();
        assert_eq!(e.kind, ErrorKind::NotFound);
    }
}
