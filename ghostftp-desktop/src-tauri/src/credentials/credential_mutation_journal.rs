//! Isolated journal candidate. No production schema or automatic replay.
//! One immutable intent per journal; an external anchor is required for recovery.
use aes_gcm::{
    aead::{Aead, KeyInit, Payload},
    Aes256Gcm, Nonce,
};
use anyhow::{anyhow, ensure, Result};
use rand::{rngs::OsRng, RngCore};
use rusqlite::{params, Connection, OptionalExtension, TransactionBehavior};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::path::Path;

const DOMAIN: &[u8] = b"ghostftp.credential-mutation-intent.v1";
const LIMIT: usize = 16384;

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
enum Commitment {
    Absent,
    Present([u8; 32]),
}

impl Commitment {
    fn of(value: Option<&str>) -> Self {
        match value {
            None => Self::Absent,
            Some(value) => Self::Present(Sha256::digest(value.as_bytes()).into()),
        }
    }
}

#[derive(Serialize, Deserialize)]
struct Intent {
    scope: String,
    service: String,
    account: String,
    before: Commitment,
    desired: Commitment,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(super) enum Observation {
    DesiredObserved,
    OriginalObserved,
    Conflict,
    Unavailable,
}

// Not Clone; the only way to obtain this token is a new durable intent and
// verified external anchor. Reopening an existing journal never yields one.
pub(super) struct DispatchPermit {
    _private: (),
}

impl DispatchPermit {
    pub(super) fn dispatch(self, mutate: impl FnOnce() -> Result<()>) -> Result<()> {
        mutate()
    }
}

pub(super) struct Journal {
    conn: Connection,
    scope: String,
}

impl Journal {
    pub(super) fn open(path: &Path, scope: &str) -> Result<Self> {
        ensure!(!scope.is_empty(), "missing mutation scope");
        let conn = Connection::open(path)?;
        conn.pragma_update(None, "synchronous", "FULL")?;
        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS credential_intent (
                slot INTEGER PRIMARY KEY CHECK(slot = 1),
                envelope BLOB NOT NULL
            );",
        )?;
        Ok(Self {
            conn,
            scope: scope.into(),
        })
    }

    fn aad(&self) -> Vec<u8> {
        [DOMAIN, self.scope.as_bytes()].concat()
    }

    fn row(&self) -> Result<Option<Vec<u8>>> {
        self.conn
            .query_row(
                "SELECT envelope FROM credential_intent WHERE slot = 1 AND length(envelope) <= ?1",
                [LIMIT],
                |row| row.get(0),
            )
            .optional()
            .map_err(Into::into)
    }

    // Anchor persistence is intentionally supplied by the caller. It must be
    // independent of SQLite and must return a fresh, verified observation.
    // A crash between SQLite commit and anchor commit leaves a blocked intent.
    pub(super) fn prepare(
        &mut self,
        key: &[u8; 32],
        service: &str,
        account: &str,
        before: Option<&str>,
        desired: Option<&str>,
        persist_anchor: impl FnOnce([u8; 32]) -> Result<[u8; 32]>,
    ) -> Result<DispatchPermit> {
        ensure!(
            !service.is_empty() && !account.is_empty(),
            "missing credential identity"
        );
        let intent = Intent {
            scope: self.scope.clone(),
            service: service.into(),
            account: account.into(),
            before: Commitment::of(before),
            desired: Commitment::of(desired),
        };
        ensure!(intent.before != intent.desired, "mutation has no effect");
        let bytes = serde_json::to_vec(&intent)?;
        let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid journal key"))?;
        let mut nonce = [0u8; 12];
        OsRng.fill_bytes(&mut nonce);
        let encrypted = cipher
            .encrypt(
                Nonce::from_slice(&nonce),
                Payload {
                    msg: &bytes,
                    aad: &self.aad(),
                },
            )
            .map_err(|_| anyhow!("journal authentication failed"))?;
        let envelope = [nonce.as_slice(), encrypted.as_slice()].concat();
        ensure!(envelope.len() <= LIMIT, "journal envelope too large");
        let anchor: [u8; 32] = Sha256::digest(&envelope).into();
        let tx = self
            .conn
            .transaction_with_behavior(TransactionBehavior::Immediate)?;
        tx.execute(
            "INSERT INTO credential_intent(slot, envelope) VALUES(1, ?1)",
            params![&envelope],
        )?;
        tx.commit()?;
        ensure!(
            persist_anchor(anchor)? == anchor,
            "journal anchor readback mismatch"
        );
        ensure!(
            self.row()?.as_deref() == Some(envelope.as_slice()),
            "journal readback mismatch"
        );
        Ok(DispatchPermit { _private: () })
    }

    // Only observes. Even a matching old state is not evidence that an
    // outstanding RPC cannot commit later. No mutation or cleanup here.
    pub(super) fn reconcile(
        &self,
        key: &[u8; 32],
        anchor: &[u8; 32],
        observe: impl FnOnce(&str, &str) -> Result<Option<String>>,
    ) -> Result<Observation> {
        let envelope = self
            .row()?
            .ok_or_else(|| anyhow!("journal missing or oversized"))?;
        ensure!(envelope.len() >= 28, "invalid journal envelope");
        let actual: [u8; 32] = Sha256::digest(&envelope).into();
        ensure!(&actual == anchor, "journal anchor mismatch");
        let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid journal key"))?;
        let bytes = cipher
            .decrypt(
                Nonce::from_slice(&envelope[..12]),
                Payload {
                    msg: &envelope[12..],
                    aad: &self.aad(),
                },
            )
            .map_err(|_| anyhow!("journal authentication failed"))?;
        let intent: Intent = serde_json::from_slice(&bytes)?;
        ensure!(intent.scope == self.scope, "journal scope mismatch");
        ensure!(intent.before != intent.desired, "invalid journal intent");
        let Ok(observed) = observe(&intent.service, &intent.account) else {
            return Ok(Observation::Unavailable);
        };
        let observed = Commitment::of(observed.as_deref());
        Ok(if observed == intent.desired {
            Observation::DesiredObserved
        } else if observed == intent.before {
            Observation::OriginalObserved
        } else {
            Observation::Conflict
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const KEY: [u8; 32] = [41; 32];
    const SCOPE: &str = "disposable-journal-fixture";

    fn fixture() -> std::path::PathBuf {
        let root = std::env::temp_dir().join(format!("ghostftp-intent-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir(&root).unwrap();
        root
    }

    #[test]
    fn journal_reopen_classifies_without_replay() {
        let root = fixture();
        let path = root.join("journal.db");
        let mut journal = Journal::open(&path, SCOPE).unwrap();
        let mut anchor = [0; 32];
        let permit = journal
            .prepare(&KEY, "service", "account", Some("old"), Some("new"), |a| {
                anchor = a;
                Ok(a)
            })
            .unwrap();
        assert!(permit.dispatch(|| Err(anyhow!("uncertain write"))).is_err());
        drop(journal);
        let mut journal = Journal::open(&path, SCOPE).unwrap();
        for (value, expected) in [
            (Some("new"), Observation::DesiredObserved),
            (Some("old"), Observation::OriginalObserved),
            (None, Observation::Conflict),
            (Some("other"), Observation::Conflict),
        ] {
            assert_eq!(
                journal
                    .reconcile(&KEY, &anchor, |service, account| {
                        assert_eq!((service, account), ("service", "account"));
                        Ok(value.map(str::to_owned))
                    })
                    .unwrap(),
                expected
            );
        }
        assert_eq!(
            journal
                .reconcile(&KEY, &anchor, |_, _| Err(anyhow!("locked")))
                .unwrap(),
            Observation::Unavailable
        );
        assert!(journal
            .prepare(&KEY, "service", "account", Some("old"), None, |_| {
                panic!("must not replace an unresolved intent")
            })
            .is_err());
        drop(journal);
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn journal_rejects_tampering_and_unanchored_intent() {
        let root = fixture();
        let path = root.join("journal.db");
        let mut journal = Journal::open(&path, SCOPE).unwrap();
        let mut anchor = [0; 32];
        assert!(journal
            .prepare(&KEY, "service", "account", Some("old"), None, |a| {
                anchor = a;
                Err(anyhow!("anchor unavailable"))
            })
            .is_err());
        assert!(journal
            .reconcile(&KEY, &[0; 32], |_, _| panic!("must not observe"))
            .is_err());
        assert!(journal
            .reconcile(&[9; 32], &anchor, |_, _| panic!("must not observe"))
            .is_err());
        let mut envelope = journal.row().unwrap().unwrap();
        envelope[15] ^= 1;
        journal
            .conn
            .execute(
                "UPDATE credential_intent SET envelope = ?1",
                params![envelope],
            )
            .unwrap();
        assert!(journal
            .reconcile(&KEY, &anchor, |_, _| panic!("must not observe"))
            .is_err());
        drop(journal);
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn journal_survives_application_process_kill() {
        const ROOT: &str = "GHOSTFTP_INTENT_CRASH_FIXTURE";
        if let Some(root) = std::env::var_os(ROOT) {
            let root = std::path::PathBuf::from(root);
            let mut journal = Journal::open(&root.join("journal.db"), SCOPE).unwrap();
            let permit = journal
                .prepare(&KEY, "service", "account", Some("old"), None, |a| {
                    use std::io::Write;
                    let mut file = std::fs::File::create(root.join("anchor"))?;
                    file.write_all(&a)?;
                    file.sync_all()?;
                    Ok(a)
                })
                .unwrap();
            permit
                .dispatch(|| {
                    // Synthetic provider accepted the deletion; the application
                    // is killed before it can observe or report that result.
                    std::fs::remove_file(root.join("credential"))?;
                    std::fs::File::open(&root)?.sync_all()?;
                    std::fs::write(root.join("ready"), b"1")?;
                    loop {
                        std::thread::sleep(std::time::Duration::from_secs(1));
                    }
                })
                .unwrap();
            return;
        }
        let root = fixture();
        std::fs::write(root.join("credential"), b"old").unwrap();
        let mut child = std::process::Command::new(std::env::current_exe().unwrap())
            .arg("journal_survives_application_process_kill")
            .arg("--nocapture")
            .env(ROOT, &root)
            .spawn()
            .unwrap();
        let deadline = std::time::Instant::now() + std::time::Duration::from_secs(15);
        while !root.join("ready").exists() && std::time::Instant::now() < deadline {
            std::thread::sleep(std::time::Duration::from_millis(20));
        }
        let ready = root.join("ready").exists();
        let _ = child.kill();
        child.wait().unwrap();
        assert!(ready);
        let anchor: [u8; 32] = std::fs::read(root.join("anchor"))
            .unwrap()
            .try_into()
            .unwrap();
        let journal = Journal::open(&root.join("journal.db"), SCOPE).unwrap();
        assert_eq!(
            journal
                .reconcile(&KEY, &anchor, |_, _| {
                    assert!(!root.join("credential").exists());
                    Ok(None)
                })
                .unwrap(),
            Observation::DesiredObserved
        );
        drop(journal);
        std::fs::remove_dir_all(root).unwrap();
    }
}
