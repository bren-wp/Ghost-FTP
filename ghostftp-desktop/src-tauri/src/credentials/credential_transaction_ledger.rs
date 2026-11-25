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
use std::io::{Read, Write};
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

#[derive(Debug, Eq, PartialEq, Serialize, Deserialize)]
struct Entry {
    intent: Intent,
    last: Option<Observation>,
    archived: bool,
}

#[derive(Debug, Eq, PartialEq, Serialize, Deserialize)]
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
        event_aad(&self.scope, seq)
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
        let history = replay(key, &self.scope, trusted, &self.envelopes()?)?;
        ensure!(
            anchor.read()? == trusted,
            "anchor changed during verification"
        );
        Ok(history)
    }

    fn envelopes(&self) -> Result<Vec<(u64, Vec<u8>)>> {
        let mut stmt = self.conn.prepare(
            "SELECT seq, length(envelope), CASE WHEN length(envelope) <= 16384
             THEN envelope ELSE NULL END FROM credential_events ORDER BY seq LIMIT 4097",
        )?;
        let mut rows = stmt.query([])?;
        let mut envelopes = Vec::new();
        while let Some(row) = rows.next()? {
            let seq: u64 = row.get(0)?;
            let size: usize = row.get(1)?;
            ensure!(
                (28..=MAX_ENVELOPE).contains(&size),
                "invalid ledger envelope size"
            );
            envelopes.push((seq, row.get(2)?));
        }
        ensure!(envelopes.len() as u64 <= LIMIT, "ledger capacity exceeded");
        Ok(envelopes)
    }

    // Retain the complete source ledger. A compact authenticated projection
    // is exported alongside the original encrypted chain, never in its place.
    fn checkpoint(
        &self,
        key: &[u8; 32],
        anchor: &mut impl Anchor,
        destination: &Path,
    ) -> Result<()> {
        let history = self.verify(key, anchor)?;
        let rows = self.envelopes()?;
        ensure!(
            replay(key, &self.scope, history.head, &rows)? == history,
            "checkpoint source changed"
        );
        let archive = serde_json::to_vec(&rows)?;
        let projection = serde_json::to_vec(&history)?;
        let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid ledger key"))?;
        let mut nonce = [0u8; 12];
        OsRng.fill_bytes(&mut nonce);
        let encrypted = cipher
            .encrypt(
                Nonce::from_slice(&nonce),
                Payload {
                    msg: &projection,
                    aad: &checkpoint_aad(&self.scope, history.head),
                },
            )
            .map_err(|_| anyhow!("checkpoint encryption failed"))?;
        let checkpoint = [nonce.as_slice(), encrypted.as_slice()].concat();
        write_new_copy(&destination.join("events.json"), &archive)?;
        write_new_copy(&destination.join("checkpoint.bin"), &checkpoint)?;
        File::open(destination)?.sync_all()?;
        ensure!(
            verify_copy(key, &self.scope, anchor, destination)? == history,
            "checkpoint readback mismatch"
        );
        Ok(())
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

fn event_aad(scope: &str, seq: u64) -> Vec<u8> {
    let size = (scope.len() as u64).to_be_bytes();
    let seq = seq.to_be_bytes();
    [DOMAIN, &size, scope.as_bytes(), &seq].concat()
}

fn replay(
    key: &[u8; 32],
    scope: &str,
    trusted: Head,
    rows: &[(u64, Vec<u8>)],
) -> Result<History> {
    ensure!(
        trusted.seq <= LIMIT && rows.len() as u64 == trusted.seq,
        "ledger count differs from anchor"
    );
    let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid ledger key"))?;
    let mut history = History {
        head: Head::genesis(),
        entries: BTreeMap::new(),
    };
    for (seq, envelope) in rows {
        ensure!(
            *seq == history.head.seq + 1 && (28..=MAX_ENVELOPE).contains(&envelope.len()),
            "invalid ledger sequence or size"
        );
        let bytes = cipher
            .decrypt(
                Nonce::from_slice(&envelope[..12]),
                Payload {
                    msg: &envelope[12..],
                    aad: &event_aad(scope, *seq),
                },
            )
            .map_err(|_| anyhow!("ledger authentication failed"))?;
        let record: Record = serde_json::from_slice(&bytes)?;
        ensure!(record.previous == history.head, "ledger chain mismatch");
        history.apply(record.event)?;
        history.head = Head {
            seq: *seq,
            digest: Sha256::digest(envelope).into(),
        };
    }
    ensure!(history.head == trusted, "ledger root mismatch");
    Ok(history)
}

fn checkpoint_aad(scope: &str, head: Head) -> Vec<u8> {
    [
        b"ghostftp.credential-checkpoint.v1".as_slice(),
        &(scope.len() as u64).to_be_bytes(),
        scope.as_bytes(),
        &head.seq.to_be_bytes(),
        &head.digest,
    ]
    .concat()
}

// Bounded by the existing ledger event/size limit, including JSON expansion.
const MAX_COPY: u64 = LIMIT * MAX_ENVELOPE as u64 * 4 + 1024 * 1024;

fn write_new_copy(path: &Path, bytes: &[u8]) -> Result<()> {
    ensure!(bytes.len() as u64 <= MAX_COPY, "recovery copy too large");
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(path)?;
    file.write_all(bytes)?;
    file.sync_all()?;
    Ok(())
}

fn read_copy(path: &Path) -> Result<Vec<u8>> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
        .open(path)?;
    let meta = file.metadata()?;
    ensure!(
        meta.is_file() && meta.nlink() == 1 && meta.mode() & 0o077 == 0,
        "unsafe recovery copy"
    );
    ensure!(meta.len() <= MAX_COPY, "recovery copy too large");
    let mut bytes = Vec::new();
    file.take(MAX_COPY + 1).read_to_end(&mut bytes)?;
    ensure!(bytes.len() as u64 == meta.len(), "recovery copy changed");
    Ok(bytes)
}

fn verify_copy(
    key: &[u8; 32],
    scope: &str,
    anchor: &mut impl Anchor,
    destination: &Path,
) -> Result<History> {
    let head = anchor.read()?;
    let rows: Vec<(u64, Vec<u8>)> =
        serde_json::from_slice(&read_copy(&destination.join("events.json"))?)?;
    let history = replay(key, scope, head, &rows)?;
    let checkpoint = read_copy(&destination.join("checkpoint.bin"))?;
    ensure!(checkpoint.len() >= 28, "invalid checkpoint");
    let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid ledger key"))?;
    let bytes = cipher
        .decrypt(
            Nonce::from_slice(&checkpoint[..12]),
            Payload {
                msg: &checkpoint[12..],
                aad: &checkpoint_aad(scope, head),
            },
        )
        .map_err(|_| anyhow!("checkpoint authentication failed"))?;
    let projection: History = serde_json::from_slice(&bytes)?;
    ensure!(projection == history, "checkpoint projection mismatch");
    ensure!(anchor.read()? == head, "anchor changed during recovery");
    Ok(history)
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
        let _ = ledger
            .prepare(&KEY, &mut anchor, intent("a", "one"))
            .unwrap();
        let first = anchor.0;
        let _ = ledger
            .prepare(&KEY, &mut anchor, intent("b", "two"))
            .unwrap();
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

    #[test]
    fn ledger_checkpoint_preserves_source_and_rejects_tampering() {
        let source = root();
        let backup = root();
        let mut anchor = FixtureAnchor(Head::genesis());
        let mut ledger = Ledger::open(&source, SCOPE).unwrap();
        let _ = ledger
            .prepare(&KEY, &mut anchor, intent("a", "one"))
            .unwrap();
        let _ = ledger
            .prepare(&KEY, &mut anchor, intent("b", "two"))
            .unwrap();
        ledger.checkpoint(&KEY, &mut anchor, &backup).unwrap();
        assert!(ledger.checkpoint(&KEY, &mut anchor, &backup).is_err());
        let history = ledger.verify(&KEY, &mut anchor).unwrap();
        assert_eq!(history.head.seq, 2);
        drop(ledger);
        assert_eq!(
            verify_copy(&KEY, SCOPE, &mut anchor, &backup).unwrap(),
            history
        );
        let old = anchor.0;
        anchor.0 = Head::genesis();
        assert!(verify_copy(&KEY, SCOPE, &mut anchor, &backup).is_err());
        anchor.0 = old;
        let path = backup.join("checkpoint.bin");
        let mut bytes = std::fs::read(&path).unwrap();
        bytes[15] ^= 1;
        std::fs::write(path, bytes).unwrap();
        assert!(verify_copy(&KEY, SCOPE, &mut anchor, &backup).is_err());
        let ledger = Ledger::open(&source, SCOPE).unwrap();
        assert_eq!(ledger.verify(&KEY, &mut anchor).unwrap(), history);
        drop(ledger);
        std::fs::remove_dir_all(backup).unwrap();
        std::fs::remove_dir_all(source).unwrap();
    }

    #[test]
    fn ledger_lock_released_after_two_process_kills() {
        const ROOT: &str = "GHOSTFTP_LEDGER_LOCK_FIXTURE";
        if let Some(path) = std::env::var_os(ROOT) {
            let path = std::path::PathBuf::from(path);
            let _ledger = Ledger::open(&path, SCOPE).unwrap();
            std::fs::write(path.join("ready"), b"1").unwrap();
            loop {
                std::thread::sleep(std::time::Duration::from_secs(1));
            }
        }
        let source = root();
        for _ in 0..2 {
            let mut child = std::process::Command::new(std::env::current_exe().unwrap())
                .arg("ledger_lock_released_after_two_process_kills")
                .env(ROOT, &source)
                .spawn()
                .unwrap();
            let ready = source.join("ready");
            let deadline = std::time::Instant::now() + std::time::Duration::from_secs(15);
            while !ready.exists() && std::time::Instant::now() < deadline {
                std::thread::sleep(std::time::Duration::from_millis(20));
            }
            let held = ready.exists() && Ledger::open(&source, SCOPE).is_err();
            let _ = child.kill();
            child.wait().unwrap();
            assert!(held);
            std::fs::remove_file(ready).unwrap();
            assert!(Ledger::open(&source, SCOPE).is_ok());
        }
        std::fs::remove_dir_all(source).unwrap();
    }

    // A real independent Secret Service entry, only in a disposable session.
    struct NativeAnchor {
        account: String,
    }

    impl NativeAnchor {
        const SERVICE: &'static str = "com.ghostftp.native-vault.ledger.fixture";

        fn enroll(account: String) -> Result<Self> {
            use super::super::native_vault_candidate::{lookup, store};
            ensure!(
                lookup(Self::SERVICE, &account)?.is_none(),
                "anchor already enrolled"
            );
            store(
                Self::SERVICE,
                &account,
                &serde_json::to_string(&Head::genesis())?,
            )?;
            let mut anchor = Self { account };
            ensure!(anchor.read()? == Head::genesis(), "anchor enrollment failed");
            Ok(anchor)
        }
    }

    impl Anchor for NativeAnchor {
        fn read(&mut self) -> Result<Head> {
            use super::super::native_vault_candidate::lookup;
            let encoded = lookup(Self::SERVICE, &self.account)?
                .ok_or_else(|| anyhow!("trusted native anchor missing"))?;
            Ok(serde_json::from_str(&encoded)?)
        }

        fn advance(&mut self, previous: Head, next: Head) -> Result<()> {
            use super::super::native_vault_candidate::store;
            ensure!(
                next.seq == previous.seq + 1 && self.read()? == previous,
                "native anchor compare failed"
            );
            store(
                Self::SERVICE,
                &self.account,
                &serde_json::to_string(&next)?,
            )?;
            ensure!(self.read()? == next, "native anchor readback failed");
            Ok(())
        }
    }

    #[test]
    #[ignore = "requires isolated unlocked native Secret Service"]
    fn ledger_native_anchor_checkpoint() {
        use super::super::native_vault_candidate::{
            clear_direct_candidate, configure_direct_candidate, remove,
        };
        assert_eq!(
            std::env::var("GHOSTFTP_DIRECT_VAULT_TEST").as_deref(),
            Ok("disposable-fixtures")
        );
        configure_direct_candidate();
        let source = root();
        let backup = root();
        let account = uuid::Uuid::new_v4().to_string();
        let mut anchor = NativeAnchor::enroll(account.clone()).unwrap();
        let mut key = [0u8; 32];
        OsRng.fill_bytes(&mut key);
        let mut ledger = Ledger::open(&source, &account).unwrap();
        let _ = ledger
            .prepare(&key, &mut anchor, intent("a", "one"))
            .unwrap();
        let _ = ledger
            .prepare(&key, &mut anchor, intent("b", "two"))
            .unwrap();
        ledger.checkpoint(&key, &mut anchor, &backup).unwrap();
        drop(ledger);
        clear_direct_candidate();
        configure_direct_candidate();
        let mut anchor = NativeAnchor { account };
        assert_eq!(
            verify_copy(&key, &anchor.account.clone(), &mut anchor, &backup)
                .unwrap()
                .head
                .seq,
            2
        );
        remove(NativeAnchor::SERVICE, &anchor.account).unwrap();
        assert!(anchor.read().is_err());
        clear_direct_candidate();
        std::fs::remove_dir_all(backup).unwrap();
        std::fs::remove_dir_all(source).unwrap();
    }
}
