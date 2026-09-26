//! In-app SSH key generation — the "no more PuTTYgen" helper.
//!
//! The New Connection editor lets a user create an SFTP key right there instead
//! of shelling out to PuTTYgen / ssh-keygen: generate an Ed25519 (or RSA)
//! keypair, write the private key next to a `.pub`, point the connection at it,
//! and hand back the public-key line to paste into the server's
//! `~/.ssh/authorized_keys`.
//!
//! The private key is written in **native OpenSSH format**
//! (`-----BEGIN OPENSSH PRIVATE KEY-----`, encrypted with bcrypt-pbkdf +
//! aes256-ctr like `ssh-keygen` when a passphrase is given), via the `ssh-key`
//! crate. That format is read back both by OpenSSH's own `ssh`/`ssh-keygen`
//! *and* by russh's `load_secret_key` — the call the SSH connect path makes — so
//! a generated key works everywhere, not just inside Ghost FTP. (russh can only
//! *write* PKCS#8 PEM, which OpenSSH refuses to load for Ed25519.) The **public**
//! key is emitted in standard OpenSSH one-line form (`ssh-ed25519 AAAA… comment`)
//! — exactly what `sshd` wants in `authorized_keys`.

use anyhow::{anyhow, bail, Context, Result};
use russh_keys::key::PublicKey;
use russh_keys::PublicKeyBase64;
use serde::{Deserialize, Serialize};
use ssh_key::private::{KeypairData, RsaKeypair};
use ssh_key::rand_core::OsRng;
use ssh_key::{Algorithm, HashAlg, LineEnding, PrivateKey};
use std::path::{Path, PathBuf};

/// Default RSA modulus size when the caller doesn't specify one.
const DEFAULT_RSA_BITS: usize = 4096;

/// Expand a leading `~` / `~/` (or `~\` on Windows) to the user's home dir.
/// Anything else is returned unchanged. Used both to resolve a save path for a
/// generated key and — in the SSH connect path — so a stored `~/.ssh/…` key
/// path actually loads (`russh_keys::load_secret_key` does no tilde expansion).
pub fn expand_tilde(path: &str) -> PathBuf {
    if path == "~" {
        if let Some(home) = dirs::home_dir() {
            return home;
        }
    } else if let Some(rest) = path
        .strip_prefix("~/")
        .or_else(|| path.strip_prefix("~\\"))
    {
        if let Some(home) = dirs::home_dir() {
            return home.join(rest);
        }
    }
    PathBuf::from(path)
}

/// Which algorithm to generate. Kept intentionally small — Ed25519 is the modern
/// default; RSA is offered for older servers that still refuse Ed25519.
#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum KeyType {
    Ed25519,
    Rsa,
}

/// Request payload for `generate` (mirrors `GenerateKeyRequest` in the frontend).
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GenerateKeyRequest {
    pub key_type: KeyType,
    /// RSA modulus size in bits; ignored for Ed25519. Defaults to 4096.
    #[serde(default)]
    pub bits: Option<usize>,
    /// Encrypt the private key with this passphrase. Empty / absent → unencrypted.
    #[serde(default)]
    pub passphrase: Option<String>,
    /// Where to write the private key. May start with `~`. The `.pub` is written
    /// alongside it.
    pub path: String,
    /// Trailing comment on the public-key line (e.g. `user@host`). Optional.
    #[serde(default)]
    pub comment: Option<String>,
    /// Overwrite an existing key at `path` instead of erroring. Default false, so
    /// we never silently clobber a key the user already relies on.
    #[serde(default)]
    pub overwrite: bool,
}

/// The generated (or derived) key, as the editor needs it.
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct GeneratedKey {
    /// Absolute path the private key was written to (`.pub` sits beside it).
    pub path: String,
    /// The full public-key line, ready to paste into `~/.ssh/authorized_keys`.
    pub public_key: String,
    /// OpenSSH-style `SHA256:…` fingerprint of the public key.
    pub fingerprint: String,
    /// Wire key type: `ssh-ed25519` or `ssh-rsa`.
    pub key_type: String,
}

/// Suggested defaults for the generator UI.
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SshKeyDefaults {
    /// Absolute path of the user's `~/.ssh` directory (created on generate).
    pub dir: String,
    /// A ready-to-use, non-colliding private-key path under `dir`.
    pub suggested_path: String,
}

/// Suggest a `~/.ssh` dir and a free filename in it for a new key.
pub fn defaults() -> SshKeyDefaults {
    let ssh_dir = dirs::home_dir()
        .map(|h| h.join(".ssh"))
        .unwrap_or_else(|| PathBuf::from(".ssh"));
    let suggested = free_path(&ssh_dir, "ghostftp_ed25519");
    SshKeyDefaults {
        dir: ssh_dir.to_string_lossy().into_owned(),
        suggested_path: suggested.to_string_lossy().into_owned(),
    }
}

/// Generate a keypair, write the private key (+ `.pub`), and return the public
/// key. CPU-bound (RSA-4096 in particular), so callers run this off the async
/// runtime.
pub fn generate(req: &GenerateKeyRequest) -> Result<GeneratedKey> {
    let priv_path = expand_tilde(&req.path);
    let pub_path = pub_path_for(&priv_path);

    // Never clobber an existing key unless explicitly told to — losing a private
    // key you can't regenerate is exactly the destructive surprise to avoid.
    if !req.overwrite && (priv_path.exists() || pub_path.exists()) {
        bail!(
            "a key already exists at {} — pick a different name or enable overwrite",
            priv_path.display()
        );
    }
    if let Some(parent) = priv_path.parent() {
        std::fs::create_dir_all(parent)
            .with_context(|| format!("creating {}", parent.display()))?;
    }

    // Public-key comment, ssh-keygen's `user@host` convention.
    let comment = req
        .comment
        .as_deref()
        .map(str::trim)
        .filter(|c| !c.is_empty())
        .unwrap_or("");

    let mut rng = OsRng;
    let key = match req.key_type {
        KeyType::Ed25519 => {
            let mut k = PrivateKey::random(&mut rng, Algorithm::Ed25519)
                .map_err(|e| anyhow!("generating Ed25519 key: {e}"))?;
            k.set_comment(comment);
            k
        }
        KeyType::Rsa => {
            let bits = req.bits.unwrap_or(DEFAULT_RSA_BITS);
            if bits < 2048 {
                bail!("RSA key size must be at least 2048 bits");
            }
            // PrivateKey::random pins RSA to 3072, so build the keypair
            // explicitly to honour the requested size.
            let rsa = RsaKeypair::random(&mut rng, bits)
                .map_err(|e| anyhow!("generating RSA key: {e}"))?;
            PrivateKey::new(KeypairData::from(rsa), comment)
                .map_err(|e| anyhow!("building RSA key: {e}"))?
        }
    };

    // Public key + fingerprint come from the unencrypted key (encrypting doesn't
    // change them). OpenSSH one-line form, comment included.
    let pub_line = key
        .public_key()
        .to_openssh()
        .map_err(|e| anyhow!("encoding public key: {e}"))?;
    let fingerprint = key.fingerprint(HashAlg::Sha256).to_string();
    let key_type = key.algorithm().as_str().to_string();

    // Serialize the private key to native OpenSSH PEM, encrypted (bcrypt-pbkdf +
    // aes256-ctr) iff a passphrase is set.
    let pass = req.passphrase.as_deref().filter(|p| !p.is_empty());
    let pem = match pass {
        Some(p) => key
            .encrypt(&mut rng, p.as_bytes())
            .map_err(|e| anyhow!("encrypting private key: {e}"))?
            .to_openssh(LineEnding::LF)
            .map_err(|e| anyhow!("encoding private key: {e}"))?,
        None => key
            .to_openssh(LineEnding::LF)
            .map_err(|e| anyhow!("encoding private key: {e}"))?,
    };

    write_private_key(&priv_path, pem.as_bytes())?;
    std::fs::write(&pub_path, format!("{pub_line}\n"))
        .with_context(|| format!("writing {}", pub_path.display()))?;

    Ok(GeneratedKey {
        path: priv_path.to_string_lossy().into_owned(),
        public_key: pub_line,
        fingerprint,
        key_type,
    })
}

/// Generate an ed25519 keypair entirely in memory, returning the OpenSSH
/// public-key line and the unencrypted OpenSSH-format private-key PEM. Nothing
/// touches disk — used by the grant-accept flow, which uploads the public half
/// and stores the private half straight into the OS keychain. CPU-bound, so
/// callers run this on the blocking pool.
pub fn generate_ed25519_in_memory(comment: &str) -> Result<(String, String)> {
    let mut rng = OsRng;
    let mut key = PrivateKey::random(&mut rng, Algorithm::Ed25519)
        .map_err(|e| anyhow!("generating Ed25519 key: {e}"))?;
    key.set_comment(comment);
    let pub_line = key
        .public_key()
        .to_openssh()
        .map_err(|e| anyhow!("encoding public key: {e}"))?;
    let pem = key
        .to_openssh(LineEnding::LF)
        .map_err(|e| anyhow!("encoding private key: {e}"))?;
    Ok((pub_line, pem.to_string()))
}

/// Derive the public-key line + fingerprint for an *existing* private key path,
/// so the user can copy it without regenerating. Needs the passphrase if the key
/// is encrypted.
pub fn public_key_for(path: &str, passphrase: Option<&str>) -> Result<GeneratedKey> {
    let priv_path = expand_tilde(path);
    let pass = passphrase.filter(|p| !p.is_empty());
    let keypair = russh_keys::load_secret_key(&priv_path, pass)
        .with_context(|| format!("loading key {}", priv_path.display()))?;
    let pk = keypair
        .clone_public_key()
        .map_err(|e| anyhow!("deriving public key: {e}"))?;
    Ok(GeneratedKey {
        path: priv_path.to_string_lossy().into_owned(),
        public_key: public_key_line(&pk, None),
        fingerprint: fingerprint(&pk),
        key_type: type_token(&pk).to_string(),
    })
}

/// Write the private key bytes, then tighten permissions to 0600 on Unix. On
/// Windows perms are ACL-based and `load_secret_key` doesn't enforce them, so we
/// leave inherited ACLs in place.
fn write_private_key(path: &Path, pem: &[u8]) -> Result<()> {
    std::fs::write(path, pem).with_context(|| format!("writing {}", path.display()))?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let _ = std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600));
    }
    Ok(())
}

/// `<name>` → `<name>.pub`, preserving any extension the private key path has.
fn pub_path_for(priv_path: &Path) -> PathBuf {
    let mut s = priv_path.as_os_str().to_os_string();
    s.push(".pub");
    PathBuf::from(s)
}

/// The wire key-type token for an `authorized_keys` line. Derived from the key
/// variant rather than `PublicKey::name()`, which returns the *signature* algo
/// name for RSA (`rsa-sha2-256`) — wrong as an `authorized_keys` type token.
fn type_token(pk: &PublicKey) -> &'static str {
    match pk {
        PublicKey::Ed25519(_) => "ssh-ed25519",
        PublicKey::RSA { .. } => "ssh-rsa",
        // We never generate EC keys; fall back to russh's own name.
        other => other.name(),
    }
}

/// Build the one-line OpenSSH public-key representation: `type base64 [comment]`.
fn public_key_line(pk: &PublicKey, comment: Option<&str>) -> String {
    let typ = type_token(pk);
    let body = pk.public_key_base64();
    match comment.map(str::trim).filter(|c| !c.is_empty()) {
        Some(c) => format!("{typ} {body} {c}"),
        None => format!("{typ} {body}"),
    }
}

/// OpenSSH-style `SHA256:<base64-nopad>` fingerprint. `PublicKey::fingerprint`
/// already returns the base64-nopad SHA256 body.
fn fingerprint(pk: &PublicKey) -> String {
    format!("SHA256:{}", pk.fingerprint())
}

/// First of `<base>`, `<base>_2`, `<base>_3`, … whose private *and* `.pub` paths
/// are both free.
fn free_path(dir: &Path, base: &str) -> PathBuf {
    let mut candidate = dir.join(base);
    let mut n = 2;
    while candidate.exists() || pub_path_for(&candidate).exists() {
        candidate = dir.join(format!("{base}_{n}"));
        n += 1;
    }
    candidate
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn generated_ed25519_roundtrips_and_line_is_wellformed() {
        let dir = std::env::temp_dir().join(format!("ghostftp-keytest-{}", uuid::Uuid::new_v4()));
        let path = dir.join("id_ed25519");
        let req = GenerateKeyRequest {
            key_type: KeyType::Ed25519,
            bits: None,
            passphrase: Some("hunter2".into()),
            path: path.to_string_lossy().into_owned(),
            comment: Some("alice@box".into()),
            overwrite: false,
        };
        let out = generate(&req).expect("generate");

        // Public line: correct type token, three space-separated fields.
        assert!(out.public_key.starts_with("ssh-ed25519 "));
        assert!(out.public_key.ends_with(" alice@box"));
        assert_eq!(out.key_type, "ssh-ed25519");
        assert!(out.fingerprint.starts_with("SHA256:"));

        // The encrypted private key loads back with the passphrase (the exact
        // call the connect path makes). The derived public key carries no
        // comment (that lives only in the .pub), so compare the type + key body
        // — the two fields that must round-trip.
        let derived = public_key_for(&out.path, Some("hunter2")).expect("reload");
        let body = |line: &str| line.split(' ').take(2).collect::<Vec<_>>().join(" ");
        assert_eq!(body(&derived.public_key), body(&out.public_key));
        assert_eq!(derived.fingerprint, out.fingerprint);

        // A second generate at the same path must refuse without overwrite.
        assert!(generate(&req).is_err());

        std::fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn generated_rsa_is_wellformed_and_reloads() {
        let dir = std::env::temp_dir().join(format!("ghostftp-rsatest-{}", uuid::Uuid::new_v4()));
        let path = dir.join("id_rsa");
        // 2048 bits keeps the test fast; the code path is identical to 4096.
        let out = generate(&GenerateKeyRequest {
            key_type: KeyType::Rsa,
            bits: Some(2048),
            passphrase: None,
            path: path.to_string_lossy().into_owned(),
            comment: None,
            overwrite: false,
        })
        .expect("generate rsa");

        assert!(out.public_key.starts_with("ssh-rsa "));
        assert_eq!(out.key_type, "ssh-rsa");
        // Loads back through the exact call the connect path uses.
        let derived = public_key_for(&out.path, None).expect("reload rsa");
        assert!(derived.public_key.starts_with("ssh-rsa "));

        std::fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn expand_tilde_leaves_plain_paths_alone() {
        assert_eq!(expand_tilde("/etc/ssh/key"), PathBuf::from("/etc/ssh/key"));
    }

    #[test]
    fn in_memory_ed25519_roundtrips_through_russh() {
        let (pub_line, pem) = generate_ed25519_in_memory("ghostftp-grant").expect("keygen");
        assert!(pub_line.starts_with("ssh-ed25519 "));
        assert!(pub_line.ends_with(" ghostftp-grant"));
        assert!(pem.starts_with("-----BEGIN OPENSSH PRIVATE KEY-----"));
        // The exact decode call the grant connect path makes on the stored PEM.
        let keypair = russh_keys::decode_secret_key(&pem, None).expect("decode");
        let derived = keypair.clone_public_key().expect("public key");
        let body = |line: &str| line.split(' ').take(2).collect::<Vec<_>>().join(" ");
        assert_eq!(body(&pub_line), format!("ssh-ed25519 {}", derived.public_key_base64()));
    }

    /// Real-world interop: OpenSSH's own `ssh-keygen` must accept the private key
    /// Ghost FTP writes, and the public key it derives from that private key must match
    /// the `.pub` line we hand the user to install. Skipped where `ssh-keygen`
    /// isn't on PATH (so CI without OpenSSH still passes).
    #[test]
    fn ssh_keygen_accepts_generated_key() {
        use std::process::Command;
        if Command::new("ssh-keygen").arg("-h").output().is_err() {
            eprintln!("skipping: ssh-keygen not found on PATH");
            return;
        }

        let dir = std::env::temp_dir().join(format!("ghostftp-interop-{}", uuid::Uuid::new_v4()));
        let path = dir.join("id_ed25519");
        // Unencrypted so `ssh-keygen -y` needs no interactive passphrase prompt.
        let out = generate(&GenerateKeyRequest {
            key_type: KeyType::Ed25519,
            bits: None,
            passphrase: None,
            path: path.to_string_lossy().into_owned(),
            comment: Some("ghostftp@interop".into()),
            overwrite: false,
        })
        .expect("generate");

        // `ssh-keygen -y` reads the PRIVATE key and prints its public half. That
        // it succeeds proves OpenSSH can load what we wrote; that the printed key
        // body matches ours proves the pair we told the user to install is right.
        let y = Command::new("ssh-keygen")
            .args(["-y", "-f"])
            .arg(&path)
            .output()
            .expect("run ssh-keygen -y");
        assert!(
            y.status.success(),
            "ssh-keygen -y failed: {}",
            String::from_utf8_lossy(&y.stderr)
        );
        let printed = String::from_utf8_lossy(&y.stdout);
        let body = |line: &str| line.split(' ').take(2).collect::<Vec<_>>().join(" ");
        assert_eq!(body(printed.trim()), body(&out.public_key));

        std::fs::remove_dir_all(&dir).ok();
    }
}
