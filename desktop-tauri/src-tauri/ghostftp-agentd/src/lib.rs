//! `ghostftp-agentd` — the headless daemon that makes a machine controllable from
//! Ghost FTP over an encrypted, paired connection (see `docs/remote-agent.md`).
//!
//! The binary ([`main`](../main.rs)) is a thin CLI over this library; the library
//! form exists so the whole stack — TCP, handshake, pin check, ops — can be
//! integration-tested in-process (see `tests/`).

pub mod config;
pub mod discovery;
pub mod jobs;
pub mod ops;
pub mod server;
pub mod service;

pub use config::{config_dir, config_path, identity_path, Config, Policy};
pub use server::{handle_paired, pair_connection, pair_handshake, serve, Daemon, PairingWindow};

/// The name this machine reports in `SystemInfo` and advertises over mDNS.
/// `GHOSTFTP_AGENT_NAME` overrides it for hosts where the OS hostname is useless —
/// Android phones all report "localhost" — and embedders (the Android agent's
/// JNI layer) set it to the user-chosen device name before starting the daemon.
pub fn agent_name() -> String {
    std::env::var("GHOSTFTP_AGENT_NAME")
        .ok()
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| gethostname::gethostname().to_string_lossy().to_string())
}
