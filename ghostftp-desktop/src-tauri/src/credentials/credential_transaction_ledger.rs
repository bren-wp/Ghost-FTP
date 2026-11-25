//! Test-only append-only credential ledger; external anchor is authoritative.
use aes_gcm::{
    aead::{Aead, KeyInit, Payload},
    Aes256Gcm, Nonce,
};
use anyhow::{anyhow, ensure, Result};
use rand::{rngs::OsRng, RngCore};
use rusqlite::{params, Connection, TransactionBehavior};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::fs::{File, OpenOptions};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::Path;

const LIMIT: u64 = 4096;
const MAX_ENVELOPE: usize = 16384;
const DOMAIN: &[u8] = b"ghostftp.credential-transaction-ledger.v1";

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
struct Head {
    seq: u64,
    digest: [u8; 32],
}

impl Head {
    fn genesis() -> Self {
        Self {
            seq: 0,
            digest: [0; 32],
        }
    }
}

// Implementations must use an independent trusted store; missing anchors
// are errors, never an invitation to initialize an empty ledger.
trait Anchor {
    fn read(&mut self) -> Result<Head>;
    fn advance(&mut self, previous: Head, next: Head) -> Result<()>;
}

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

#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
struct Intent {
    id: String,
    service: String,
    account: String,
    before: Commitment,
    desired: Commitment,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
enum Observation {
    Desired,
    Original,
    Conflict,
    Unavailable,
}

impl Intent {
    fn observe(&self, read: impl FnOnce(&str, &str) -> Result<Option<String>>) -> Observation {
        match read(&self.service, &self.account) {
            Ok(value) => {
                let current = Commitment::of(value.as_deref());
                if current == self.desired {
                    Observation::Desired
                } else if current == self.before {
                    Observation::Original
                } else {
                    Observation::Conflict
                }
            }
            Err(_) => Observation::Unavailable,
        }
    }
}

#[derive(Serialize, Deserialize)]
enum Event {
    Prepared(Intent),
    Observed { id: String, state: Observation },
    Archived { id: String },
}

#[derive(Serialize, Deserialize)]
struct Record {
    previous: Head,
    event: Event,
}

struct Entry {
    intent: Intent,
    last: Option<Observation>,
    archived: bool,
}

struct History {
    head: Head,
    entries: BTreeMap<String, Entry>,
}

impl History {
    fn apply(&mut self, event: Event) -> Result<()> {
        match event {
            Event::Prepared(intent) => {
                ensure!(
                    intent.before != intent.desired
                        && !intent.id.is_empty()
                        && !intent.service.is_empty()
                        && !intent.account.is_empty(),
                    "invalid transaction intent"
                );
                ensure!(
                    !self.entries.contains_key(&intent.id),
                    "duplicate transaction id"
                );
                ensure!(
                    !self.entries.values().any(|entry| {
                        !entry.archived
                            && entry.intent.service == intent.service
                            && entry.intent.account == intent.account
                    }),
                    "unresolved credential mutation"
                );
                self.entries.insert(
                    intent.id.clone(),
                    Entry {
                        intent,
                        last: None,
                        archived: false,
                    },
                );
            }
            Event::Observed { id, state } => {
                let entry = self
                    .entries
                    .get_mut(&id)
                    .ok_or_else(|| anyhow!("unknown transaction"))?;
                ensure!(!entry.archived, "transaction already archived");
                entry.last = Some(state);
            }
            Event::Archived { id } => {
                let entry = self
                    .entries
                    .get_mut(&id)
                    .ok_or_else(|| anyhow!("unknown transaction"))?;
                ensure!(
                    !entry.archived && entry.last == Some(Observation::Desired),
                    "transaction not eligible for archive"
                );
                entry.archived = true;
            }
        }
        Ok(())
    }
}

struct Ledger {
    conn: Connection,
    scope: String,
    _lock: File,
}

impl Ledger {
    fn open(root: &Path, scope: &str) -> Result<Self> {
        ensure!(!scope.is_empty(), "missing ledger scope");
        let path = root.join("credential-ledger.lock");
        let lock = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .truncate(false)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
            .open(&path)?;
        let meta = lock.metadata()?;
        ensure!(
            meta.is_file() && meta.nlink() == 1 && meta.mode() & 0o077 == 0,
            "unsafe ledger lock"
        );
        lock.try_lock()
            .map_err(|_| anyhow!("credential ledger already locked"))?;
        let current = std::fs::symlink_metadata(&path)?;
        ensure!(
            current.is_file()
                && current.nlink() == 1
                && current.dev() == meta.dev()
                && current.ino() == meta.ino(),
            "ledger lock entry changed"
        );
        let conn = Connection::open(root.join("credential-ledger.db"))?;
        conn.pragma_update(None, "synchronous", "FULL")?;
        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS credential_events (
                seq INTEGER PRIMARY KEY CHECK(seq > 0),
                envelope BLOB NOT NULL
            );",
        )?;
        Ok(Self {
            conn,
            scope: scope.into(),
            _lock: lock,
        })
    }

    fn aad(&self, seq: u64) -> Vec<u8> {
        let size = (self.scope.len() as u64).to_be_bytes();
        let seq = seq.to_be_bytes();
        [DOMAIN, &size, self.scope.as_bytes(), &seq].concat()
    }

    fn verify(&self, key: &[u8; 32], anchor: &mut impl Anchor) -> Result<History> {
        let trusted = anchor.read()?;
        ensure!(trusted.seq <= LIMIT, "ledger capacity exceeded");
        let count: u64 = self.conn.query_row(
            "SELECT count(*) FROM credential_events",
            [],
            |row| row.get(0),
        )?;
        ensure!(count == trusted.seq, "ledger count differs from anchor");
        let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid ledger key"))?;
        let mut history = History {
            head: Head::genesis(),
            entries: BTreeMap::new(),
        };
        let mut stmt = self.conn.prepare(
            "SELECT seq, length(envelope), CASE WHEN length(envelope) <= 16384
             THEN envelope ELSE NULL END FROM credential_events ORDER BY seq",
        )?;
        let mut rows = stmt.query([])?;
        while let Some(row) = rows.next()? {
            let seq: u64 = row.get(0)?;
            let size: usize = row.get(1)?;
            ensure!(
                seq == history.head.seq + 1 && (28..=MAX_ENVELOPE).contains(&size),
                "invalid ledger sequence or size"
            );
            let envelope: Vec<u8> = row.get(2)?;
            let bytes = cipher
                .decrypt(
                    Nonce::from_slice(&envelope[..12]),
                    Payload {
                        msg: &envelope[12..],
                        aad: &self.aad(seq),
                    },
                )
                .map_err(|_| anyhow!("ledger authentication failed"))?;
            let record: Record = serde_json::from_slice(&bytes)?;
            ensure!(record.previous == history.head, "ledger chain mismatch");
            history.apply(record.event)?;
            history.head = Head {
                seq,
                digest: Sha256::digest(&envelope).into(),
            };
        }
        ensure!(history.head == trusted, "ledger root mismatch");
        ensure!(anchor.read()? == trusted, "anchor changed during verification");
        Ok(history)
    }

    fn append(&mut self, key: &[u8; 32], anchor: &mut impl Anchor, event: Event) -> Result<()> {
        let mut history = self.verify(key, anchor)?;
        let previous = history.head;
        ensure!(previous.seq < LIMIT, "ledger full; preserve history");
        let seq = previous.seq + 1;
        let bytes = serde_json::to_vec(&Record { previous, event })?;
        let record: Record = serde_json::from_slice(&bytes)?;
        history.apply(record.event)?;
        let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid ledger key"))?;
        let mut nonce = [0u8; 12];
        OsRng.fill_bytes(&mut nonce);
        let encrypted = cipher
            .encrypt(
                Nonce::from_slice(&nonce),
                Payload {
                    msg: &bytes,
                    aad: &self.aad(seq),
                },
            )
            .map_err(|_| anyhow!("ledger encryption failed"))?;
        let envelope = [nonce.as_slice(), encrypted.as_slice()].concat();
        ensure!(envelope.len() <= MAX_ENVELOPE, "ledger envelope too large");
        let next = Head {
            seq,
            digest: Sha256::digest(&envelope).into(),
        };
        let tx = self
            .conn
            .transaction_with_behavior(TransactionBehavior::Immediate)?;
        tx.execute(
            "INSERT INTO credential_events(seq, envelope) VALUES(?1, ?2)",
            params![seq, envelope],
        )?;
        tx.commit()?;
        // Never roll back or delete a committed record if this step fails.
        anchor.advance(previous, next)?;
        ensure!(self.verify(key, anchor)?.head == next, "anchor readback failed");
        Ok(())
    }

    fn prepare<'a>(
        &'a mut self,
        key: &[u8; 32],
        anchor: &mut impl Anchor,
        intent: Intent,
    ) -> Result<Permit<'a>> {
        let id = intent.id.clone();
        self.append(key, anchor, Event::Prepared(intent))?;
        Ok(Permit { ledger: self, id })
    }

    fn reconcile(
        &mut self,
        key: &[u8; 32],
        anchor: &mut impl Anchor,
        id: &str,
        read: impl FnOnce(&str, &str) -> Result<Option<String>>,
    ) -> Result<Observation> {
        let history = self.verify(key, anchor)?;
        let entry = history
            .entries
            .get(id)
            .ok_or_else(|| anyhow!("unknown transaction"))?;
        ensure!(!entry.archived, "transaction already archived");
        let state = entry.intent.observe(read);
        self.append(
            key,
            anchor,
            Event::Observed {
                id: id.into(),
                state,
            },
        )?;
        Ok(state)
    }
}

// Holds the exclusive ledger borrow (and therefore the OS lock) until consumed.
// Reconciliation never creates a new permit, even after desired is observed.
struct Permit<'a> {
    ledger: &'a mut Ledger,
    id: String,
}

impl Permit<'_> {
    fn dispatch(
        self,
        key: &[u8; 32],
        anchor: &mut impl Anchor,
        read: impl Fn(&str, &str) -> Result<Option<String>>,
        mutate: impl FnOnce() -> Result<()>,
    ) -> Result<()> {
        let history = self.ledger.verify(key, anchor)?;
        let entry = history
            .entries
            .get(&self.id)
            .ok_or_else(|| anyhow!("missing prepared transaction"))?;
        ensure!(
            !entry.archived && entry.last.is_none(),
            "transaction is not newly prepared"
        );
        ensure!(
            entry.intent.observe(&read) == Observation::Original,
            "credential changed before dispatch"
        );
        mutate()?;
        let state = self.ledger.reconcile(key, anchor, &self.id, read)?;
        ensure!(state == Observation::Desired, "mutation outcome uncertain");
        // Only a successful live dispatch can reach this boundary.
        self.ledger
            .append(key, anchor, Event::Archived { id: self.id })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    struct FixtureAnchor(Head);

    impl Anchor for FixtureAnchor {
        fn read(&mut self) -> Result<Head> {
            Ok(self.0)
        }

        fn advance(&mut self, previous: Head, next: Head) -> Result<()> {
            ensure!(self.0 == previous, "anchor compare failed");
            self.0 = next;
            Ok(())
        }
    }

    const KEY: [u8; 32] = [91; 32];
    const SCOPE: &str = "disposable-ledger-fixture";

    fn root() -> std::path::PathBuf {
        let path = std::env::temp_dir().join(format!("ghostftp-ledger-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir(&path).unwrap();
        path
    }

    fn intent(id: &str, account: &str) -> Intent {
        Intent {
            id: id.into(),
            service: "fixture".into(),
            account: account.into(),
            before: Commitment::of(Some("old")),
            desired: Commitment::of(Some("new")),
        }
    }

    #[test]
    fn ledger_multiple_pending_and_immutable_archive() {
        let root = root();
        let mut anchor = FixtureAnchor(Head::genesis());
        let mut ledger = Ledger::open(&root, SCOPE).unwrap();
        assert!(Ledger::open(&root, SCOPE).is_err());
        // Lost acknowledgement leaves the first intent open.
        assert!(ledger
            .prepare(&KEY, &mut anchor, intent("a", "one"))
            .unwrap()
            .dispatch(
                &KEY,
                &mut anchor,
                |_, _| Ok(Some("old".into())),
                || Err(anyhow!("lost acknowledgement")),
            )
            .is_err());
        let current = std::cell::Cell::new("old");
        ledger
            .prepare(&KEY, &mut anchor, intent("b", "two"))
            .unwrap()
            .dispatch(
                &KEY,
                &mut anchor,
                |_, _| Ok(Some(current.get().into())),
                || {
                    current.set("new");
                    Ok(())
                },
            )
            .unwrap();
        assert_eq!(
            ledger
                .reconcile(&KEY, &mut anchor, "a", |_, _| Ok(Some("new".into())))
                .unwrap(),
            Observation::Desired
        );
        assert!(ledger
            .prepare(&KEY, &mut anchor, intent("c", "one"))
            .is_err());
        let history = ledger.verify(&KEY, &mut anchor).unwrap();
        assert!(!history.entries["a"].archived);
        assert!(history.entries["b"].archived);
        assert_eq!(history.head.seq, 5);
        drop(ledger);
        let ledger = Ledger::open(&root, SCOPE).unwrap();
        assert_eq!(ledger.verify(&KEY, &mut anchor).unwrap().head.seq, 5);
        drop(ledger);
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn ledger_rejects_rollback_and_tampered_history() {
        let root = root();
        let mut anchor = FixtureAnchor(Head::genesis());
        let mut ledger = Ledger::open(&root, SCOPE).unwrap();
        drop(
            ledger
                .prepare(&KEY, &mut anchor, intent("a", "one"))
                .unwrap(),
        );
        let first = anchor.0;
        drop(
            ledger
                .prepare(&KEY, &mut anchor, intent("b", "two"))
                .unwrap(),
        );
        let latest = anchor.0;
        anchor.0 = first;
        assert!(ledger.verify(&KEY, &mut anchor).is_err());
        anchor.0 = latest;
        assert!(ledger.verify(&[0; 32], &mut anchor).is_err());
        ledger
            .conn
            .execute("DELETE FROM credential_events WHERE seq = 2", [])
            .unwrap();
        assert!(ledger.verify(&KEY, &mut anchor).is_err());
        assert!(ledger
            .prepare(&KEY, &mut anchor, intent("c", "three"))
            .is_err());
        drop(ledger);
        std::fs::remove_dir_all(root).unwrap();
    }
}
