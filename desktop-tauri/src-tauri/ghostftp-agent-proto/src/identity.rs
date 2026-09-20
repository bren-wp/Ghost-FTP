//! Persistent static-key identity for a Noise peer.
//!
//! Each machine (a Ghost FTP controller or a `ghostftp-agentd`) has one long-lived
//! X25519 static keypair. Its public half is what the other side pins at
//! pairing time; the fingerprint is what a human reads off to recognise it. The
//! private half never leaves the machine and is stored 0600 on Unix.

use anyhow::{Context, Result};
use base64::Engine as _;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::path::Path;

const NOISE_PATTERN: &str = "Noise_XX_25519_ChaChaPoly_SHA256";

fn b64() -> base64::engine::general_purpose::GeneralPurpose {
    base64::engine::general_purpose::STANDARD
}

/// A long-lived Noise static keypair, serialised as base64 in a small JSON file.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Identity {
    /// base64 of the 32-byte X25519 private key.
    private_key: String,
    /// base64 of the 32-byte X25519 public key.
    public_key: String,
}

impl Identity {
    /// Generate a fresh identity using snow's keypair generator (OS RNG).
    pub fn generate() -> Result<Self> {
        let params = NOISE_PATTERN.parse().context("parse noise params")?;
        let kp = snow::Builder::new(params)
            .generate_keypair()
            .context("generate X25519 keypair")?;
        Ok(Self {
            private_key: b64().encode(kp.private),
            public_key: b64().encode(kp.public),
        })
    }

    /// Load an identity from `path`, generating and persisting a new one if the
    /// file doesn't exist yet. This is the normal startup path for both sides.
    pub fn load_or_create(path: &Path) -> Result<Self> {
        if path.exists() {
            let bytes = std::fs::read(path).with_context(|| format!("read identity {path:?}"))?;
            return serde_json::from_slice(&bytes)
                .with_context(|| format!("parse identity {path:?}"));
        }
        let id = Self::generate()?;
        id.save(path)?;
        Ok(id)
    }

    /// Persist to `path` (locked to the owner on Unix).
    pub fn save(&self, path: &Path) -> Result<()> {
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent).ok();
        }
        let bytes = serde_json::to_vec_pretty(self)?;
        std::fs::write(path, &bytes).with_context(|| format!("write identity {path:?}"))?;
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let _ = std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600));
        }
        Ok(())
    }

    /// Raw 32-byte private key, for feeding into a Noise builder.
    pub fn private_bytes(&self) -> Result<Vec<u8>> {
        b64()
            .decode(&self.private_key)
            .context("decode private key")
    }

    /// Raw 32-byte public key.
    pub fn public_bytes(&self) -> Result<Vec<u8>> {
        b64().decode(&self.public_key).context("decode public key")
    }

    /// The public key as base64 — the canonical string form used to pin a peer.
    pub fn public_b64(&self) -> &str {
        &self.public_key
    }

    /// Short human-readable fingerprint of the public key (`ab:cd:…`, first 8
    /// bytes of its SHA-256). What the pairing UI shows so a person can confirm
    /// they're pinning the machine they think they are.
    pub fn fingerprint(&self) -> String {
        fingerprint_of(&self.public_bytes().unwrap_or_default())
    }
}

/// Fingerprint of an arbitrary public key (same format as [`Identity::fingerprint`]).
pub fn fingerprint_of(public_key: &[u8]) -> String {
    let digest = Sha256::digest(public_key);
    digest[..8]
        .iter()
        .map(|b| format!("{b:02x}"))
        .collect::<Vec<_>>()
        .join(":")
}

/// Encode a raw public key to its canonical base64 pin string.
pub fn encode_public(public_key: &[u8]) -> String {
    b64().encode(public_key)
}

/// Decode a base64 pin string back to raw public-key bytes.
pub fn decode_public(pin: &str) -> Result<Vec<u8>> {
    b64().decode(pin).context("decode peer public key")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn generate_is_32_bytes_and_stable_fingerprint() {
        let id = Identity::generate().unwrap();
        assert_eq!(id.private_bytes().unwrap().len(), 32);
        assert_eq!(id.public_bytes().unwrap().len(), 32);
        // Fingerprint is deterministic for a given key and formatted xx:xx:...
        assert_eq!(
            id.fingerprint(),
            fingerprint_of(&id.public_bytes().unwrap())
        );
        assert_eq!(id.fingerprint().split(':').count(), 8);
    }

    #[test]
    fn load_or_create_persists_and_reloads() {
        let dir = std::env::temp_dir().join(format!("ghostftp-id-test-{}", std::process::id()));
        let path = dir.join("identity.json");
        let _ = std::fs::remove_file(&path);
        let a = Identity::load_or_create(&path).unwrap();
        let b = Identity::load_or_create(&path).unwrap();
        assert_eq!(a.public_b64(), b.public_b64());
        let _ = std::fs::remove_dir_all(&dir);
    }
}
