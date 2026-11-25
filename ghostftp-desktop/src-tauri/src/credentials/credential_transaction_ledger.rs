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
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt};
use std::path::{Path, PathBuf};

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
    Rotated {
        next_segment: u64,
        base: Head,
        generation: String,
        manifest_digest: [u8; 32],
    },
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
    #[serde(default)]
    segments: BTreeMap<u64, u64>,
}

impl History {
    fn active_segment(&self) -> u64 {
        self.segments.last_key_value().map_or(0, |(_, value)| *value)
    }

    fn segment_for(&self, seq: u64) -> u64 {
        self.segments.range(..=seq).next_back().map_or(0, |(_, value)| *value)
    }

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
            Event::Rotated {
                next_segment,
                base,
                generation,
                manifest_digest,
            } => {
                ensure!(
                    base == self.head
                        && next_segment == self.active_segment() + 1
                        && generation.starts_with("generation-")
                        && manifest_digest != [0; 32],
                    "invalid segment rotation"
                );
                // The rotation record belongs to the old segment; the
                // first subsequent record belongs to the new segment.
                self.segments.insert(self.head.seq + 2, next_segment);
            }
        }
        Ok(())
    }
}

struct Ledger {
    conn: Connection,
    scope: String,
    root: PathBuf,
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
            );
            CREATE TABLE IF NOT EXISTS credential_segment_events (
                seq INTEGER PRIMARY KEY CHECK(seq > 0),
                segment INTEGER NOT NULL CHECK(segment > 0),
                envelope BLOB NOT NULL
            );",
        )?;
        Ok(Self {
            conn,
            scope: scope.into(),
            root: root.canonicalize()?,
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
            "SELECT (SELECT count(*) FROM credential_events) +
                    (SELECT count(*) FROM credential_segment_events)",
            [],
            |row| row.get(0),
        )?;
        ensure!(count == trusted.seq, "ledger count differs from anchor");
        let history = replay(key, &self.scope, trusted, &self.envelopes()?)?;
        let mut statement = self.conn.prepare(
            "SELECT seq, 0 FROM credential_events
             UNION ALL SELECT seq, segment FROM credential_segment_events ORDER BY seq",
        )?;
        let mut rows = statement.query([])?;
        while let Some(row) = rows.next()? {
            let seq: u64 = row.get(0)?;
            let segment: u64 = row.get(1)?;
            ensure!(
                history.segment_for(seq) == segment,
                "physical row assigned to wrong logical segment"
            );
        }
        ensure!(
            anchor.read()? == trusted,
            "anchor changed during verification"
        );
        Ok(history)
    }

    fn envelopes(&self) -> Result<Vec<(u64, Vec<u8>)>> {
        let mut stmt = self.conn.prepare(
            "SELECT seq, length(envelope), CASE WHEN length(envelope) <= 16384
             THEN envelope ELSE NULL END FROM (
               SELECT seq, envelope FROM credential_events
               UNION ALL SELECT seq, envelope FROM credential_segment_events
             ) ORDER BY seq LIMIT 4097",
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
        self.checkpoint_with_hook(key, anchor, destination, |_| Ok(()))
    }

    fn checkpoint_with_hook(
        &self,
        key: &[u8; 32],
        anchor: &mut impl Anchor,
        destination: &Path,
        hook: impl Fn(&str) -> Result<()>,
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
        hook("events_synced")?;
        write_new_copy(&destination.join("checkpoint.bin"), &checkpoint)?;
        hook("checkpoint_synced")?;
        File::open(destination)?.sync_all()?;
        ensure!(
            verify_copy(key, &self.scope, anchor, destination)? == history,
            "checkpoint readback mismatch"
        );
        Ok(())
    }

    // Publication is deliberately separate from the active ledger. The caller
    // chooses a private backup root; physical independence is not inferred.
    fn publish_generation(
        &self,
        key: &[u8; 32],
        anchor: &mut impl Anchor,
        backup: &Path,
        hook: impl Fn(&str) -> Result<()>,
    ) -> Result<PathBuf> {
        check_private_directory(backup)?;
        let head = self.verify(key, anchor)?.head;
        let id = format!("generation-{:016}-{}", head.seq, uuid::Uuid::new_v4());
        let destination = backup.join(&id);
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&destination)?;
        File::open(backup)?.sync_all()?;
        hook("directory_synced")?;
        self.checkpoint_with_hook(key, anchor, &destination, &hook)?;
        hook("copy_verified")?;

        let manifest = GenerationManifest {
            id: id.clone(),
            head,
            events: file_digest(&destination.join("events.json"))?,
            checkpoint: file_digest(&destination.join("checkpoint.bin"))?,
        };
        let sealed = seal_manifest(key, &self.scope, &manifest)?;
        write_new_copy(&destination.join("manifest.pending"), &sealed)?;
        File::open(&destination)?.sync_all()?;
        hook("manifest_synced")?;
        ensure!(
            anchor.read()? == head,
            "anchor advanced before publication"
        );
        // Atomic no-replace publication in the same directory. A torn or
        // unverified generation never authorizes mutation or source deletion.
        std::fs::hard_link(
            destination.join("manifest.pending"),
            destination.join("manifest.committed"),
        )?;
        hook("marker_linked")?;
        File::open(&destination)?.sync_all()?;
        // The pending name is intentionally retained. The committed marker
        // is allowed exactly these two links, checked by the verifier.
        hook("marker_synced")?;
        File::open(backup)?.sync_all()?;
        hook("root_synced")?;
        verify_generation(key, &self.scope, anchor, &destination)?;
        Ok(destination)
    }

    fn rotate_on_independent_device(
        &mut self,
        key: &[u8; 32],
        anchor: &mut impl Anchor,
        backup: &Path,
    ) -> Result<()> {
        verify_distinct_block_devices(&self.root, backup)?;
        let generation = self.publish_generation(key, anchor, backup, |_| Ok(()))?;
        // Detect mount changes around publication. This is a conservative
        // topology check, not proof of independent hardware or power domains.
        verify_distinct_block_devices(&self.root, backup)?;
        self.rotate_verified(key, anchor, &generation)
    }

    fn rotate_verified(
        &mut self,
        key: &[u8; 32],
        anchor: &mut impl Anchor,
        generation: &Path,
    ) -> Result<()> {
        let history = self.verify(key, anchor)?;
        ensure!(
            verify_generation(key, &self.scope, anchor, generation)? == history,
            "rotation checkpoint differs from ledger"
        );
        let id = generation
            .file_name()
            .and_then(|name| name.to_str())
            .ok_or_else(|| anyhow!("invalid generation identity"))?;
        let manifest_digest = publication_digest(generation)?;
        self.append(
            key,
            anchor,
            Event::Rotated {
                next_segment: history.active_segment() + 1,
                base: history.head,
                generation: id.into(),
                manifest_digest,
            },
        )
    }

    fn append(&mut self, key: &[u8; 32], anchor: &mut impl Anchor, event: Event) -> Result<()> {
        let mut history = self.verify(key, anchor)?;
        let previous = history.head;
        let segment = history.active_segment();
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
        insert_envelope(&tx, segment, seq, &envelope)?;
        tx.commit()?;
        // Never roll back or delete a committed record if this step fails.
        anchor.advance(previous, next)?;
        ensure!(
            self.verify(key, anchor)?.head == next,
            "anchor readback failed"
        );
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

fn insert_envelope(conn: &Connection, segment: u64, seq: u64, bytes: &[u8]) -> Result<()> {
    if segment == 0 {
        conn.execute(
            "INSERT INTO credential_events(seq, envelope) VALUES(?1, ?2)",
            params![seq, bytes],
        )?;
    } else {
        conn.execute(
            "INSERT INTO credential_segment_events(seq, segment, envelope) VALUES(?1, ?2, ?3)",
            params![seq, segment, bytes],
        )?;
    }
    Ok(())
}

fn event_aad(scope: &str, seq: u64) -> Vec<u8> {
    let size = (scope.len() as u64).to_be_bytes();
    let seq = seq.to_be_bytes();
    [DOMAIN, &size, scope.as_bytes(), &seq].concat()
}

fn replay(key: &[u8; 32], scope: &str, trusted: Head, rows: &[(u64, Vec<u8>)]) -> Result<History> {
    ensure!(
        trusted.seq <= LIMIT && rows.len() as u64 == trusted.seq,
        "ledger count differs from anchor"
    );
    let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid ledger key"))?;
    let mut history = History {
        head: Head::genesis(),
        entries: BTreeMap::new(),
        segments: BTreeMap::new(),
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

#[derive(Serialize, Deserialize)]
struct GenerationManifest {
    id: String,
    head: Head,
    events: [u8; 32],
    checkpoint: [u8; 32],
}

fn check_private_directory(path: &Path) -> Result<()> {
    let meta = std::fs::symlink_metadata(path)?;
    ensure!(
        meta.is_dir() && meta.mode() & 0o077 == 0,
        "unsafe recovery directory"
    );
    Ok(())
}

fn file_digest(path: &Path) -> Result<[u8; 32]> {
    Ok(Sha256::digest(read_copy(path)?).into())
}

fn manifest_aad(scope: &str) -> Vec<u8> {
    [b"ghostftp.credential-generation.v1".as_slice(), scope.as_bytes()].concat()
}

fn seal_manifest(key: &[u8; 32], scope: &str, manifest: &GenerationManifest) -> Result<Vec<u8>> {
    let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid ledger key"))?;
    let mut nonce = [0u8; 12];
    OsRng.fill_bytes(&mut nonce);
    let bytes = serde_json::to_vec(manifest)?;
    let encrypted = cipher
        .encrypt(
            Nonce::from_slice(&nonce),
            Payload {
                msg: &bytes,
                aad: &manifest_aad(scope),
            },
        )
        .map_err(|_| anyhow!("manifest authentication failed"))?;
    Ok([nonce.as_slice(), encrypted.as_slice()].concat())
}

fn verify_generation(
    key: &[u8; 32],
    scope: &str,
    anchor: &mut impl Anchor,
    path: &Path,
) -> Result<History> {
    check_private_directory(path)?;
    let pending = std::fs::symlink_metadata(path.join("manifest.pending"))?;
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
        .open(path.join("manifest.committed"))?;
    let meta = file.metadata()?;
    ensure!(
        meta.is_file()
            && pending.is_file()
            && meta.nlink() == 2
            && pending.dev() == meta.dev()
            && pending.ino() == meta.ino()
            && meta.mode() & 0o077 == 0
            && (28..=4096).contains(&meta.len()),
        "invalid publication marker"
    );
    let mut sealed = Vec::new();
    file.take(4097).read_to_end(&mut sealed)?;
    ensure!(sealed.len() as u64 == meta.len(), "manifest changed");
    let cipher = Aes256Gcm::new_from_slice(key).map_err(|_| anyhow!("invalid ledger key"))?;
    let bytes = cipher
        .decrypt(
            Nonce::from_slice(&sealed[..12]),
            Payload {
                msg: &sealed[12..],
                aad: &manifest_aad(scope),
            },
        )
        .map_err(|_| anyhow!("manifest authentication failed"))?;
    let manifest: GenerationManifest = serde_json::from_slice(&bytes)?;
    ensure!(
        path.file_name().and_then(|name| name.to_str()) == Some(manifest.id.as_str()),
        "generation identity mismatch"
    );
    ensure!(manifest.head == anchor.read()?, "generation is not current");
    ensure!(
        manifest.events == file_digest(&path.join("events.json"))?
            && manifest.checkpoint == file_digest(&path.join("checkpoint.bin"))?,
        "generation file digest mismatch"
    );
    let history = verify_copy(key, scope, anchor, path)?;
    ensure!(history.head == manifest.head, "generation head mismatch");
    Ok(history)
}

// Linux block topology: accept only a directly identifiable non-virtual
// whole block device or its partition. Reject overlay, network, loop, RAID,
// device-mapper and unknown topology instead of claiming independence.
fn block_root(path: &Path) -> Result<PathBuf> {
    let dev = std::fs::metadata(path)?.dev();
    let node = PathBuf::from(format!(
        "/sys/dev/block/{}:{}",
        libc::major(dev),
        libc::minor(dev)
    ))
    .canonicalize()?;
    let disk = if node.join("partition").exists() {
        node.parent()
            .ok_or_else(|| anyhow!("unknown partition parent"))?
            .to_path_buf()
    } else {
        node
    };
    ensure!(
        disk.starts_with("/sys/devices") && !disk.starts_with("/sys/devices/virtual"),
        "virtual or unknown block topology"
    );
    ensure!(
        disk.join("device").exists() && std::fs::read_dir(disk.join("slaves"))?.next().is_none(),
        "layered or unknown block topology"
    );
    Ok(disk)
}

fn verify_distinct_block_devices(source: &Path, backup: &Path) -> Result<()> {
    check_private_directory(backup)?;
    ensure!(
        std::fs::metadata(source)?.dev() != std::fs::metadata(backup)?.dev(),
        "source and backup share filesystem device"
    );
    ensure!(
        block_root(source)? != block_root(backup)?,
        "source and backup share underlying block device"
    );
    Ok(())
}

// Read the verified publication marker without using the generic single-link
// reader. Its identity and content are checked again by verify_generation.
fn publication_digest(path: &Path) -> Result<[u8; 32]> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
        .open(path.join("manifest.committed"))?;
    let meta = file.metadata()?;
    ensure!(
        meta.is_file() && meta.nlink() == 2 && (28..=4096).contains(&meta.len()),
        "invalid publication marker"
    );
    let mut bytes = Vec::new();
    file.take(4097).read_to_end(&mut bytes)?;
    ensure!(bytes.len() as u64 == meta.len(), "publication marker changed");
    Ok(Sha256::digest(bytes).into())
}

// Copy without changing the authenticated generation identity. A partially
// copied directory never has a committed marker. Never overwrite a copy.
fn migrate_generation(
    key: &[u8; 32],
    scope: &str,
    anchor: &mut impl Anchor,
    source: &Path,
    destination_root: &Path,
) -> Result<PathBuf> {
    let history = verify_generation(key, scope, anchor, source)?;
    check_private_directory(destination_root)?;
    let id = source
        .file_name()
        .ok_or_else(|| anyhow!("missing generation identity"))?;
    let destination = destination_root.join(id);
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&destination)?;
    File::open(destination_root)?.sync_all()?;
    for name in ["events.json", "checkpoint.bin"] {
        let bytes = read_copy(&source.join(name))?;
        write_new_copy(&destination.join(name), &bytes)?;
    }
    let marker = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
        .open(source.join("manifest.committed"))?;
    let mut sealed = Vec::new();
    marker.take(4097).read_to_end(&mut sealed)?;
    ensure!(
        sealed.len() <= 4096
            && <[u8; 32]>::from(Sha256::digest(&sealed)) == publication_digest(source)?,
        "source marker changed"
    );
    write_new_copy(&destination.join("manifest.pending"), &sealed)?;
    File::open(&destination)?.sync_all()?;
    ensure!(
        verify_copy(key, scope, anchor, &destination)? == history,
        "migrated checkpoint differs from source"
    );
    std::fs::hard_link(
        destination.join("manifest.pending"),
        destination.join("manifest.committed"),
    )?;
    File::open(&destination)?.sync_all()?;
    File::open(destination_root)?.sync_all()?;
    ensure!(
        verify_generation(key, scope, anchor, &destination)? == history,
        "migrated generation readback failed"
    );
    Ok(destination)
}

// Restore into a new empty ledger only. No secret writes, no automatic
// enrollment of anchor or key, no dispatch permit returned by restoration.
fn restore_generation(
    key: &[u8; 32],
    scope: &str,
    anchor: &mut impl Anchor,
    generation: &Path,
    new_root: &Path,
) -> Result<()> {
    let history = verify_generation(key, scope, anchor, generation)?;
    check_private_directory(new_root)?;
    ensure!(
        std::fs::read_dir(new_root)?.next().is_none(),
        "restore destination not empty"
    );
    let rows: Vec<(u64, Vec<u8>)> =
        serde_json::from_slice(&read_copy(&generation.join("events.json"))?)?;
    ensure!(
        replay(key, scope, history.head, &rows)? == history,
        "restore source changed"
    );
    let mut ledger = Ledger::open(new_root, scope)?;
    let tx = ledger
        .conn
        .transaction_with_behavior(TransactionBehavior::Immediate)?;
    for (seq, envelope) in rows {
        insert_envelope(&tx, history.segment_for(seq), seq, &envelope)?;
    }
    tx.commit()?;
    File::open(new_root)?.sync_all()?;
    ensure!(
        ledger.verify(key, anchor)? == history,
        "restored ledger readback failed"
    );
    Ok(())
}

// Select only a verified generation matching the current trusted head.
// A stale generation can be retained for manual forensic recovery, but must
// never be used to roll back live credential state. Errors preserve all files.
fn select_generation(
    key: &[u8; 32],
    scope: &str,
    anchor: &mut impl Anchor,
    backup: &Path,
) -> Result<PathBuf> {
    check_private_directory(backup)?;
    let head = anchor.read()?;
    let prefix = format!("generation-{:016}-", head.seq);
    let mut candidates = Vec::new();
    for entry in std::fs::read_dir(backup)? {
        let entry = entry?;
        if entry.file_name().to_string_lossy().starts_with(&prefix) {
            candidates.push(entry.path());
            ensure!(candidates.len() <= 128, "too many recovery candidates");
        }
    }
    candidates.sort();
    for path in candidates {
        if verify_generation(key, scope, anchor, &path).is_ok() {
            ensure!(anchor.read()? == head, "anchor changed during selection");
            return Ok(path);
        }
    }
    Err(anyhow!("no current verified recovery generation"))
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
            ensure!(
                anchor.read()? == Head::genesis(),
                "anchor enrollment failed"
            );
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
            store(Self::SERVICE, &self.account, &serde_json::to_string(&next)?)?;
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

    #[test]
    fn ledger_publication_crash_matrix() {
        const ROOT: &str = "GHOSTFTP_GENERATION_CRASH_FIXTURE";
        const PHASE: &str = "GHOSTFTP_GENERATION_CRASH_PHASE";
        if let Some(path) = std::env::var_os(ROOT) {
            let path = PathBuf::from(path);
            let phase = std::env::var(PHASE).unwrap();
            let head: Head =
                serde_json::from_slice(&std::fs::read(path.join("head")).unwrap()).unwrap();
            let mut anchor = FixtureAnchor(head);
            let ledger = Ledger::open(&path.join("source"), SCOPE).unwrap();
            ledger
                .publish_generation(&KEY, &mut anchor, &path.join("backup"), |stage| {
                    if stage == phase {
                        std::fs::write(path.join("ready"), b"1")?;
                        loop {
                            std::thread::sleep(std::time::Duration::from_secs(1));
                        }
                    }
                    Ok(())
                })
                .unwrap();
            panic!("fixture did not reach selected phase");
        }
        let path = root();
        let source = path.join("source");
        let backup = path.join("backup");
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&source)
            .unwrap();
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&backup)
            .unwrap();
        let mut anchor = FixtureAnchor(Head::genesis());
        let mut ledger = Ledger::open(&source, SCOPE).unwrap();
        let _ = ledger
            .prepare(&KEY, &mut anchor, intent("a", "one"))
            .unwrap();
        let previous = ledger
            .publish_generation(&KEY, &mut anchor, &backup, |_| Ok(()))
            .unwrap();
        let _ = ledger
            .prepare(&KEY, &mut anchor, intent("b", "two"))
            .unwrap();
        std::fs::write(path.join("head"), serde_json::to_vec(&anchor.0).unwrap()).unwrap();
        drop(ledger);
        assert!(select_generation(&KEY, SCOPE, &mut anchor, &backup).is_err());
        for phase in [
            "directory_synced",
            "events_synced",
            "checkpoint_synced",
            "copy_verified",
            "manifest_synced",
            "marker_linked",
            "marker_synced",
            "root_synced",
        ] {
            let mut child = std::process::Command::new(std::env::current_exe().unwrap())
                .arg("ledger_publication_crash_matrix")
                .env(ROOT, &path)
                .env(PHASE, phase)
                .spawn()
                .unwrap();
            let ready = path.join("ready");
            let deadline = std::time::Instant::now() + std::time::Duration::from_secs(15);
            while !ready.exists() && std::time::Instant::now() < deadline {
                std::thread::sleep(std::time::Duration::from_millis(20));
            }
            let held = ready.exists() && Ledger::open(&source, SCOPE).is_err();
            let _ = child.kill();
            child.wait().unwrap();
            assert!(held, "failed to reach publication phase {phase}");
            std::fs::remove_file(ready).unwrap();
            assert!(previous.join("manifest.committed").exists());
            let ledger = Ledger::open(&source, SCOPE).unwrap();
            assert_eq!(ledger.verify(&KEY, &mut anchor).unwrap().head.seq, 2);
            drop(ledger);
            let selected = select_generation(&KEY, SCOPE, &mut anchor, &backup);
            if matches!(
                phase,
                "marker_linked" | "marker_synced" | "root_synced"
            ) {
                assert_ne!(selected.unwrap(), previous);
            } else {
                assert!(selected.is_err());
            }
        }
        let selected = select_generation(&KEY, SCOPE, &mut anchor, &backup).unwrap();
        assert_eq!(
            verify_generation(&KEY, SCOPE, &mut anchor, &selected)
                .unwrap()
                .head
                .seq,
            2
        );
        std::fs::remove_dir_all(path).unwrap();
    }

    #[test]
    fn ledger_rotated_segment_and_replacement_restore() {
        let source = root();
        let backup = root();
        let migrated = root();
        let restored = root();
        for path in [&backup, &migrated, &restored] {
            use std::os::unix::fs::PermissionsExt;
            std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o700)).unwrap();
        }
        let mut anchor = FixtureAnchor(Head::genesis());
        let mut ledger = Ledger::open(&source, SCOPE).unwrap();
        let _ = ledger
            .prepare(&KEY, &mut anchor, intent("a", "one"))
            .unwrap();
        assert!(
            ledger
                .rotate_on_independent_device(&KEY, &mut anchor, &backup)
                .is_err()
        );
        let previous = ledger
            .publish_generation(&KEY, &mut anchor, &backup, |_| Ok(()))
            .unwrap();
        // Internal rotation boundary exercised without pretending the
        // temporary directories represent independent hardware.
        ledger.rotate_verified(&KEY, &mut anchor, &previous).unwrap();
        let _ = ledger
            .prepare(&KEY, &mut anchor, intent("b", "two"))
            .unwrap();
        let history = ledger.verify(&KEY, &mut anchor).unwrap();
        assert_eq!(history.active_segment(), 1);
        assert_eq!(history.head.seq, 3);
        assert_eq!(history.segment_for(2), 0);
        assert_eq!(history.segment_for(3), 1);
        let generation = ledger
            .publish_generation(&KEY, &mut anchor, &backup, |_| Ok(()))
            .unwrap();
        let moved = migrate_generation(&KEY, SCOPE, &mut anchor, &generation, &migrated).unwrap();
        assert!(migrate_generation(&KEY, SCOPE, &mut anchor, &generation, &migrated).is_err());
        restore_generation(&KEY, SCOPE, &mut anchor, &moved, &restored).unwrap();
        assert!(restore_generation(&KEY, SCOPE, &mut anchor, &moved, &restored).is_err());
        let replacement = Ledger::open(&restored, SCOPE).unwrap();
        assert_eq!(replacement.verify(&KEY, &mut anchor).unwrap(), history);
        drop(replacement);
        assert!(previous.join("manifest.committed").exists());
        assert!(generation.join("manifest.committed").exists());
        ledger
            .conn
            .execute(
                "UPDATE credential_segment_events SET segment = 2 WHERE seq = 3",
                [],
            )
            .unwrap();
        assert!(ledger.verify(&KEY, &mut anchor).is_err());
        drop(ledger);
        for path in [source, backup, migrated, restored] {
            std::fs::remove_dir_all(path).unwrap();
        }
    }
}
