//! `ghostftp-agent-proto` — the wire protocol and encrypted channel shared by Ghost FTP
//! (the controller) and `ghostftp-agentd` (the daemon running on a paired machine).
//!
//! Deliberately Tauri-free and small, so the daemon links it without dragging in
//! the GUI stack. See `docs/remote-agent.md` for the architecture.
//!
//! The four layers, bottom-up:
//!   * [`frame`] — length-prefixed framing over any async stream.
//!   * [`secure`] — a Noise handshake ([`SecureChannel`]) with pairing + pinning.
//!   * [`identity`] — persistent static keypairs + fingerprints.
//!   * [`msg`] — the [`Request`]/[`Response`] operation set.
//!
//! Plus [`delta`] — the FastCDC/BLAKE3 block-level delta-sync engine shared by
//! the app and the daemon so both run the identical chunking algorithm.

pub mod delta;
pub mod frame;
pub mod identity;
pub mod msg;
pub mod pairing;
pub mod secure;

pub use identity::{decode_public, encode_public, fingerprint_of, Identity};
pub use msg::{
    DirEntry, FileKind, Hello, Request, Response, SystemInfo, PROTOCOL_VERSION, SERVICE_TYPE,
};
pub use pairing::{generate_code, is_valid_code, psk_from_code, CODE_LEN};
pub use secure::{Auth, Role, SecureChannel};
