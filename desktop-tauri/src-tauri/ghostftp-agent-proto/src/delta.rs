//! Block-level delta sync: content-defined chunking (FastCDC v2020) + BLAKE3,
//! rsync-style, without a rolling hash. Identical content chunks identically on
//! both sides, so matching is a plain hash → offset lookup (the restic/Borg
//! model). This engine lives in the proto crate so the app (controller) and
//! `ghostftp-agentd` (daemon) run the *identical* algorithm and constants.
//!
//! Flow (upload): the controller fetches the remote (old) file's
//! [`FileSignature`], runs [`plan_delta`] against the local new file — matched
//! chunks become `Copy` recipe ops, unmatched bytes stream into a patch file as
//! `Literal` ops — then the daemon runs [`apply_delta`] to reassemble
//! disk-to-disk, fsync, and BLAKE3-verify before the caller renames the result
//! over the destination. Any error anywhere ⇒ the caller falls back to a
//! whole-file copy; delta never changes *whether* bytes move, only *how*.

use std::collections::HashMap;
use std::fs::File;
use std::io::{BufReader, BufWriter, Read, Seek, SeekFrom, Write};
use std::path::Path;

use anyhow::{anyhow, Context, Result};
use base64::Engine as _;
use fastcdc::v2020::StreamCDC;
use serde::{Deserialize, Serialize};

/// Minimum chunk size for the FastCDC chunker.
pub const CHUNK_MIN: u32 = 32 * 1024;
/// Average (target) chunk size for the FastCDC chunker.
pub const CHUNK_AVG: u32 = 256 * 1024;
/// Maximum chunk size for the FastCDC chunker.
pub const CHUNK_MAX: u32 = 1024 * 1024;
/// Files bigger than this never get a signature — the signature itself would
/// cost too much; the caller does a whole-file copy.
pub const SIGNATURE_MAX_FILE: u64 = 100 * 1024 * 1024 * 1024;
/// Files smaller than this are not worth the delta round trips; whole-file copy.
/// Overridable via the `GHOSTFTP_DELTA_MIN_SIZE` env var (bytes).
pub const DELTA_MIN_SIZE: u64 = 8 * 1024 * 1024;

/// Buffered-read/write size used everywhere in this module.
const IO_BUF: usize = 1024 * 1024;

fn b64() -> base64::engine::general_purpose::GeneralPurpose {
    base64::engine::general_purpose::STANDARD
}

fn encode_hash(hash: &blake3::Hash) -> String {
    b64().encode(hash.as_bytes())
}

fn decode_hash(s: &str) -> Result<[u8; 32]> {
    let bytes = b64().decode(s).context("decode base64 blake3 hash")?;
    bytes
        .try_into()
        .map_err(|_| anyhow!("blake3 hash must decode to 32 bytes"))
}

/// One chunk of a [`FileSignature`]: where it sits in the file and its hash.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ChunkEntry {
    pub offset: u64,
    pub len: u32,
    /// base64 of the chunk's BLAKE3 hash.
    pub hash: String,
}

/// The chunk map of one file. Small (~150 KiB for 800 MiB) and cheap to send
/// over the wire in either direction.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct FileSignature {
    pub size: u64,
    pub min: u32,
    pub avg: u32,
    pub max: u32,
    pub chunks: Vec<ChunkEntry>,
    /// base64 of the whole file's BLAKE3 hash.
    pub whole_hash: String,
}

/// One reassembly instruction for [`apply_delta`].
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "camelCase")]
pub enum RecipeOp {
    /// Copy `len` bytes from the basis (old) file at `basis_offset`.
    Copy { basis_offset: u64, len: u64 },
    /// Copy `len` bytes from the patch file at `patch_offset` — bytes that
    /// crossed the wire because no basis chunk matched.
    Literal { patch_offset: u64, len: u64 },
}

/// The output of [`plan_delta`]: the reassembly recipe plus the accounting the
/// caller needs for the worthwhile heuristic and progress reporting.
#[derive(Debug, Clone)]
pub struct DeltaPlan {
    pub recipe: Vec<RecipeOp>,
    pub literal_bytes: u64,
    pub reused_bytes: u64,
    /// base64 BLAKE3 of the NEW file — what [`apply_delta`] must verify against.
    pub whole_hash: String,
}

/// Stream `path` once, chunking with our FastCDC params (normalization level 1,
/// the `StreamCDC::new` default) and hashing each chunk plus the whole file
/// in the same single pass.
pub fn signature_of_file(path: &Path) -> Result<FileSignature> {
    let size = std::fs::metadata(path)
        .with_context(|| format!("stat {}", path.display()))?
        .len();
    anyhow::ensure!(
        size <= SIGNATURE_MAX_FILE,
        "file too large for delta signature: {size} bytes > {SIGNATURE_MAX_FILE}"
    );
    let file = File::open(path).with_context(|| format!("open {}", path.display()))?;
    let reader = BufReader::with_capacity(IO_BUF, file);
    let chunker = StreamCDC::new(reader, CHUNK_MIN, CHUNK_AVG, CHUNK_MAX);
    let mut whole = blake3::Hasher::new();
    let mut chunks = Vec::new();
    let mut offset = 0u64;
    for chunk in chunker {
        let chunk = chunk.with_context(|| format!("chunk {}", path.display()))?;
        whole.update(&chunk.data);
        chunks.push(ChunkEntry {
            offset,
            len: chunk.data.len() as u32,
            hash: encode_hash(&blake3::hash(&chunk.data)),
        });
        offset += chunk.data.len() as u64;
    }
    Ok(FileSignature {
        size,
        min: CHUNK_MIN,
        avg: CHUNK_AVG,
        max: CHUNK_MAX,
        chunks,
        whole_hash: encode_hash(&whole.finalize()),
    })
}

/// Plan how to rebuild `new_file` from a basis described by `basis`: matched
/// chunks become `Copy` ops, unmatched chunk bytes stream into `patch_out` as
/// coalesced `Literal` ops. Consecutive `Copy` ops (contiguous in the basis)
/// and consecutive `Literal` runs are coalesced so the recipe stays small.
/// An empty `new_file` yields an empty recipe and the hash of empty input.
pub fn plan_delta(
    basis: &FileSignature,
    new_file: &Path,
    mut patch_out: impl Write,
) -> Result<DeltaPlan> {
    // hash → list of (offset, len); duplicates kept, first (lowest offset) wins
    // on lookup — identical content, so any entry is correct.
    let mut index: HashMap<[u8; 32], Vec<(u64, u32)>> = HashMap::with_capacity(basis.chunks.len());
    for c in &basis.chunks {
        let key = decode_hash(&c.hash)
            .with_context(|| format!("invalid basis chunk hash at offset {}", c.offset))?;
        index.entry(key).or_default().push((c.offset, c.len));
    }

    let file =
        File::open(new_file).with_context(|| format!("open {}", new_file.display()))?;
    let reader = BufReader::with_capacity(IO_BUF, file);
    let chunker = StreamCDC::new(reader, CHUNK_MIN, CHUNK_AVG, CHUNK_MAX);
    let mut whole = blake3::Hasher::new();
    let mut recipe: Vec<RecipeOp> = Vec::new();
    let mut literal_bytes = 0u64;
    let mut reused_bytes = 0u64;
    let mut patch_len = 0u64;
    for chunk in chunker {
        let chunk = chunk.with_context(|| format!("chunk {}", new_file.display()))?;
        whole.update(&chunk.data);
        let len = chunk.data.len() as u64;
        match index.get(blake3::hash(&chunk.data).as_bytes()).and_then(|v| v.first()) {
            Some(&(basis_offset, basis_len)) => {
                debug_assert_eq!(basis_len as u64, len); // same hash ⇒ same bytes ⇒ same len
                reused_bytes += len;
                match recipe.last_mut() {
                    Some(RecipeOp::Copy { basis_offset: prev_off, len: prev_len })
                        if *prev_off + *prev_len == basis_offset =>
                    {
                        *prev_len += len;
                    }
                    _ => recipe.push(RecipeOp::Copy { basis_offset, len }),
                }
            }
            None => {
                patch_out.write_all(&chunk.data).context("write patch chunk")?;
                literal_bytes += len;
                match recipe.last_mut() {
                    Some(RecipeOp::Literal { len: prev_len, .. }) => *prev_len += len,
                    _ => recipe.push(RecipeOp::Literal { patch_offset: patch_len, len }),
                }
                patch_len += len;
            }
        }
    }
    patch_out.flush().context("flush patch")?;
    Ok(DeltaPlan {
        recipe,
        literal_bytes,
        reused_bytes,
        whole_hash: encode_hash(&whole.finalize()),
    })
}

/// Rebuild the new file at `out` from `basis` + `patch` following `recipe`,
/// fsync it, then verify the whole-file BLAKE3 against `expected_hash`.
/// Every op is range-validated against the actual file lengths BEFORE any byte
/// is written — a hostile or corrupt recipe must error, never panic or read
/// out of bounds. On any error (including hash mismatch) the partial `out`
/// file is deleted.
pub fn apply_delta(
    basis: Option<&Path>,
    patch: &Path,
    recipe: &[RecipeOp],
    out: &Path,
    expected_hash: &str,
) -> Result<()> {
    let result = apply_delta_inner(basis, patch, recipe, out, expected_hash);
    if result.is_err() {
        let _ = std::fs::remove_file(out);
    }
    result
}

fn apply_delta_inner(
    basis: Option<&Path>,
    patch: &Path,
    recipe: &[RecipeOp],
    out: &Path,
    expected_hash: &str,
) -> Result<()> {
    let basis_len = match basis {
        Some(p) => Some(
            std::fs::metadata(p)
                .with_context(|| format!("stat basis {}", p.display()))?
                .len(),
        ),
        None => None,
    };
    let patch_len = std::fs::metadata(patch)
        .with_context(|| format!("stat patch {}", patch.display()))?
        .len();

    // Validate every op up front — checked arithmetic, no writes before this passes.
    for op in recipe {
        match *op {
            RecipeOp::Copy { basis_offset, len } => {
                let blen = basis_len
                    .ok_or_else(|| anyhow!("recipe has Copy ops but no basis file"))?;
                let end = basis_offset
                    .checked_add(len)
                    .ok_or_else(|| anyhow!("Copy range overflows u64"))?;
                anyhow::ensure!(
                    end <= blen,
                    "Copy range [{basis_offset}, {end}) exceeds basis length {blen}"
                );
            }
            RecipeOp::Literal { patch_offset, len } => {
                let end = patch_offset
                    .checked_add(len)
                    .ok_or_else(|| anyhow!("Literal range overflows u64"))?;
                anyhow::ensure!(
                    end <= patch_len,
                    "Literal range [{patch_offset}, {end}) exceeds patch length {patch_len}"
                );
            }
        }
    }

    let mut basis_file = match basis {
        Some(p) => Some(File::open(p).with_context(|| format!("open basis {}", p.display()))?),
        None => None,
    };
    let mut patch_file =
        File::open(patch).with_context(|| format!("open patch {}", patch.display()))?;
    let out_file = File::create(out).with_context(|| format!("create {}", out.display()))?;
    let mut out_writer = BufWriter::with_capacity(IO_BUF, out_file);
    let mut buf = vec![0u8; IO_BUF];
    for op in recipe {
        match *op {
            RecipeOp::Copy { basis_offset, len } => {
                // Validated above: basis exists and the range is in bounds.
                let f = basis_file.as_mut().expect("basis validated above");
                copy_range(f, basis_offset, len, &mut out_writer, &mut buf)?;
            }
            RecipeOp::Literal { patch_offset, len } => {
                copy_range(&mut patch_file, patch_offset, len, &mut out_writer, &mut buf)?;
            }
        }
    }
    out_writer.flush().context("flush output")?;
    out_writer.get_ref().sync_all().context("fsync output")?;
    drop(out_writer);

    let expected = decode_hash(expected_hash).context("decode expected hash")?;
    let mut hasher = blake3::Hasher::new();
    hasher
        .update_reader(File::open(out).with_context(|| format!("reopen {}", out.display()))?)
        .context("hash output")?;
    anyhow::ensure!(
        hasher.finalize().as_bytes() == &expected,
        "output hash mismatch — reassembled file does not match the plan"
    );
    Ok(())
}

fn copy_range<R: Read + Seek>(
    from: &mut R,
    offset: u64,
    mut len: u64,
    out: &mut impl Write,
    buf: &mut [u8],
) -> Result<()> {
    from.seek(SeekFrom::Start(offset)).context("seek")?;
    while len > 0 {
        let n = std::cmp::min(len, buf.len() as u64) as usize;
        from.read_exact(&mut buf[..n]).context("read source range")?;
        out.write_all(&buf[..n]).context("write output")?;
        len -= n as u64;
    }
    Ok(())
}

/// Largest single coalesced range the download side of a delta fetches from
/// the daemon (itself streamed in `ReadChunk`-sized steps) — a cap so one
/// giant miss run can't turn into a single unbounded request run.
pub const DOWNLOAD_FETCH_MAX: u64 = 16 * 1024 * 1024;

/// Download-direction planning (pure addition for Phase 2; the mirror of
/// [`plan_delta`]): match the TARGET (new, remote) file's signature against
/// the local BASIS signature. Matched target chunks become `Copy` ops from the
/// basis; unmatched chunks become `Literal` ops whose patch offsets are
/// assigned sequentially — the caller must append the fetched bytes to the
/// patch file in exactly recipe order (which is also ascending file order).
/// Returns the plan plus the ranges of the target file to fetch, adjacent
/// misses coalesced and each range capped at [`DOWNLOAD_FETCH_MAX`].
pub fn plan_download(
    basis: &FileSignature,
    target: &FileSignature,
) -> Result<(DeltaPlan, Vec<(u64, u64)>)> {
    // Same hash → offset index as plan_delta.
    let mut index: HashMap<[u8; 32], Vec<(u64, u32)>> = HashMap::with_capacity(basis.chunks.len());
    for c in &basis.chunks {
        let key = decode_hash(&c.hash)
            .with_context(|| format!("invalid basis chunk hash at offset {}", c.offset))?;
        index.entry(key).or_default().push((c.offset, c.len));
    }

    let mut recipe: Vec<RecipeOp> = Vec::new();
    let mut needed: Vec<(u64, u64)> = Vec::new();
    let mut literal_bytes = 0u64;
    let mut reused_bytes = 0u64;
    let mut patch_len = 0u64;
    for chunk in &target.chunks {
        let len = chunk.len as u64;
        let key = decode_hash(&chunk.hash)
            .with_context(|| format!("invalid target chunk hash at offset {}", chunk.offset))?;
        match index.get(&key).and_then(|v| v.first()) {
            Some(&(basis_offset, basis_len)) => {
                debug_assert_eq!(basis_len as u64, len); // same hash ⇒ same bytes ⇒ same len
                reused_bytes += len;
                match recipe.last_mut() {
                    Some(RecipeOp::Copy { basis_offset: prev_off, len: prev_len })
                        if *prev_off + *prev_len == basis_offset =>
                    {
                        *prev_len += len;
                    }
                    _ => recipe.push(RecipeOp::Copy { basis_offset, len }),
                }
            }
            None => {
                literal_bytes += len;
                match recipe.last_mut() {
                    Some(RecipeOp::Literal { len: prev_len, .. }) => *prev_len += len,
                    _ => recipe.push(RecipeOp::Literal { patch_offset: patch_len, len }),
                }
                patch_len += len;
                // Coalesce the fetch range with the previous miss when adjacent
                // and under the cap.
                match needed.last_mut() {
                    Some((off, rlen)) if *off + *rlen == chunk.offset && *rlen < DOWNLOAD_FETCH_MAX => {
                        *rlen = (*rlen + len).min(DOWNLOAD_FETCH_MAX);
                        if *off + *rlen < chunk.offset + len {
                            // The cap split this chunk's tail into a new range.
                            let used = *off + *rlen - chunk.offset;
                            needed.push((chunk.offset + used, len - used));
                        }
                    }
                    _ => needed.push((chunk.offset, len)),
                }
            }
        }
    }
    let plan = DeltaPlan {
        recipe,
        literal_bytes,
        reused_bytes,
        whole_hash: target.whole_hash.clone(),
    };
    Ok((plan, needed))
}

/// Gate for attempting delta at all: feature on, a basis to diff against, and
/// the file big enough to be worth the round trips. `GHOSTFTP_DELTA_MIN_SIZE`
/// (bytes) overrides [`DELTA_MIN_SIZE`] when set to a parseable u64.
pub fn should_attempt_delta(size: u64, basis_exists: bool, enabled: bool) -> bool {
    let min = std::env::var("GHOSTFTP_DELTA_MIN_SIZE")
        .ok()
        .and_then(|v| v.parse::<u64>().ok())
        .unwrap_or(DELTA_MIN_SIZE);
    enabled && basis_exists && size >= min
}

/// Abort the delta if the plan barely saves anything: false when literal bytes
/// are ≥ 60% of the file size (integer math, no floats).
pub fn delta_worthwhile(plan: &DeltaPlan, size: u64) -> bool {
    (plan.literal_bytes as u128) * 5 < (size as u128) * 3
}

/// A received signature computed with different chunk params can never match
/// our chunks — treat as no-delta (whole-file fallback).
pub fn params_match(sig: &FileSignature) -> bool {
    sig.min == CHUNK_MIN && sig.avg == CHUNK_AVG && sig.max == CHUNK_MAX
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicUsize, Ordering};

    const KIB: usize = 1024;
    const MIB: usize = 1024 * 1024;

    /// Deterministic seeded pseudo-random bytes (xorshift64*) — no dev-deps.
    struct XorShift(u64);

    impl XorShift {
        fn next(&mut self) -> u64 {
            let mut x = self.0;
            x ^= x >> 12;
            x ^= x << 25;
            x ^= x >> 27;
            self.0 = x;
            x.wrapping_mul(0x2545_F491_4F6C_DD1D)
        }

        fn bytes(&mut self, n: usize) -> Vec<u8> {
            let mut out = Vec::with_capacity(n);
            while out.len() < n {
                out.extend_from_slice(&self.next().to_le_bytes());
            }
            out.truncate(n);
            out
        }
    }

    fn temp_dir() -> std::path::PathBuf {
        static COUNTER: AtomicUsize = AtomicUsize::new(0);
        let dir = std::env::temp_dir().join(format!(
            "ghostftp-delta-test-{}-{}",
            std::process::id(),
            COUNTER.fetch_add(1, Ordering::SeqCst)
        ));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    /// signature(base) + plan_delta(new) + apply_delta ⇒ byte-identical output,
    /// and the plan's whole_hash equals the new file's whole hash.
    fn round_trip(base: &[u8], new: &[u8]) -> DeltaPlan {
        let dir = temp_dir();
        let base_p = dir.join("base.bin");
        let new_p = dir.join("new.bin");
        let patch_p = dir.join("patch.bin");
        let out_p = dir.join("out.bin");
        std::fs::write(&base_p, base).unwrap();
        std::fs::write(&new_p, new).unwrap();

        let sig = signature_of_file(&base_p).unwrap();
        assert!(params_match(&sig));
        assert_eq!(sig.size, base.len() as u64);

        let plan = plan_delta(&sig, &new_p, File::create(&patch_p).unwrap()).unwrap();
        assert_eq!(plan.whole_hash, encode_hash(&blake3::hash(new)));
        assert_eq!(plan.literal_bytes + plan.reused_bytes, new.len() as u64);
        assert_eq!(std::fs::metadata(&patch_p).unwrap().len(), plan.literal_bytes);

        apply_delta(Some(&base_p), &patch_p, &plan.recipe, &out_p, &plan.whole_hash).unwrap();
        assert_eq!(std::fs::read(&out_p).unwrap(), new);

        let _ = std::fs::remove_dir_all(&dir);
        plan
    }

    #[derive(Clone, Copy)]
    enum Mutation {
        InsertHead,
        InsertMid,
        InsertTail,
        DeleteRange,
        Overwrite,
        Append,
        Truncate,
        PrependByte,
    }

    const MUTATIONS: [Mutation; 8] = [
        Mutation::InsertHead,
        Mutation::InsertMid,
        Mutation::InsertTail,
        Mutation::DeleteRange,
        Mutation::Overwrite,
        Mutation::Append,
        Mutation::Truncate,
        Mutation::PrependByte,
    ];

    fn mutate(base: &[u8], m: Mutation, rng: &mut XorShift) -> Vec<u8> {
        match m {
            Mutation::InsertHead => {
                let mut v = rng.bytes(KIB);
                v.extend_from_slice(base);
                v
            }
            Mutation::InsertMid => {
                let i = base.len() / 2;
                let mut v = base[..i].to_vec();
                v.extend_from_slice(&rng.bytes(KIB));
                v.extend_from_slice(&base[i..]);
                v
            }
            Mutation::InsertTail => {
                let mut v = base.to_vec();
                v.extend_from_slice(&rng.bytes(KIB));
                v
            }
            Mutation::DeleteRange => {
                let i = base.len() / 4;
                let end = (i + 4 * KIB).min(base.len());
                let mut v = base[..i].to_vec();
                v.extend_from_slice(&base[end..]);
                v
            }
            Mutation::Overwrite => {
                let mut v = base.to_vec();
                if !v.is_empty() {
                    let i = (v.len() / 3).min(v.len() - 1);
                    let n = KIB.min(v.len() - i);
                    let patch = rng.bytes(n);
                    v[i..i + n].copy_from_slice(&patch);
                }
                v
            }
            Mutation::Append => {
                let mut v = base.to_vec();
                v.extend_from_slice(&rng.bytes(MIB));
                v
            }
            Mutation::Truncate => base[..base.len() / 2].to_vec(),
            Mutation::PrependByte => {
                let mut v = vec![rng.next() as u8];
                v.extend_from_slice(base);
                v
            }
        }
    }

    #[test]
    fn round_trip_all_sizes_and_mutations() {
        let mut rng = XorShift(0x9E37_79B9_7F4A_7C15);
        for size in [0usize, 1, 100 * KIB, MIB, 8 * MIB, 20 * MIB] {
            let base = rng.bytes(size);
            for m in MUTATIONS {
                let new = mutate(&base, m, &mut rng);
                round_trip(&base, &new);
            }
        }
    }

    #[test]
    fn small_edit_is_mostly_reused() {
        let mut rng = XorShift(0xDEAD_BEEF_CAFE_F00D);
        let base = rng.bytes(16 * MIB);
        let mut new = base.clone();
        let i = 8 * MIB;
        new[i..i + KIB].copy_from_slice(&rng.bytes(KIB)); // overwrite 1 KiB mid-file

        let plan = round_trip(&base, &new);
        let size = new.len() as u64;
        assert!(
            plan.literal_bytes * 20 < size,
            "1 KiB edit: literal {} >= 5% of {size}",
            plan.literal_bytes
        );
        assert!(
            plan.reused_bytes * 10 > size * 9,
            "1 KiB edit: reused {} <= 90% of {size}",
            plan.reused_bytes
        );
    }

    #[test]
    fn duplicate_chunks_reconstruct() {
        let mut rng = XorShift(0x1234_5678_9ABC_DEF0);
        let pattern = rng.bytes(4 * KIB);
        let mut base = Vec::new();
        for _ in 0..2048 {
            base.extend_from_slice(&pattern); // 8 MiB of one repeated 4 KiB pattern
        }

        // Identical file: everything matches (all to the first occurrence),
        // zero literal bytes.
        let plan = round_trip(&base, &base.clone());
        assert_eq!(plan.literal_bytes, 0);
        assert_eq!(plan.reused_bytes, base.len() as u64);

        // Mutated duplicate file still reconstructs.
        let new = mutate(&base, Mutation::InsertMid, &mut rng);
        round_trip(&base, &new);
    }

    #[test]
    fn empty_file_edges() {
        let dir = temp_dir();
        let empty_p = dir.join("empty.bin");
        std::fs::write(&empty_p, b"").unwrap();

        let sig = signature_of_file(&empty_p).unwrap();
        assert_eq!(sig.size, 0);
        assert!(sig.chunks.is_empty());
        assert_eq!(sig.whole_hash, encode_hash(&blake3::hash(b"")));

        // Empty basis + empty new file ⇒ empty recipe, empty output.
        let patch_p = dir.join("patch.bin");
        let plan = plan_delta(&sig, &empty_p, File::create(&patch_p).unwrap()).unwrap();
        assert!(plan.recipe.is_empty());
        assert_eq!(plan.literal_bytes, 0);
        assert_eq!(plan.reused_bytes, 0);
        assert_eq!(plan.whole_hash, sig.whole_hash);
        let out_p = dir.join("out.bin");
        apply_delta(Some(&empty_p), &patch_p, &plan.recipe, &out_p, &plan.whole_hash).unwrap();
        assert_eq!(std::fs::read(&out_p).unwrap(), b"");
        // No basis at all with an empty recipe also works.
        let out2_p = dir.join("out2.bin");
        apply_delta(None, &patch_p, &[], &out2_p, &plan.whole_hash).unwrap();
        assert_eq!(std::fs::read(&out2_p).unwrap(), b"");

        // Empty basis + non-empty new file ⇒ everything is one literal run,
        // and it assembles with no basis.
        let mut rng = XorShift(0xABCD_EF01);
        let data = rng.bytes(100 * KIB);
        let new_p = dir.join("new.bin");
        std::fs::write(&new_p, &data).unwrap();
        let patch2_p = dir.join("patch2.bin");
        let plan2 = plan_delta(&sig, &new_p, File::create(&patch2_p).unwrap()).unwrap();
        assert_eq!(plan2.literal_bytes, data.len() as u64);
        assert_eq!(plan2.reused_bytes, 0);
        assert!(plan2
            .recipe
            .iter()
            .all(|op| matches!(op, RecipeOp::Literal { .. })));
        let out3_p = dir.join("out3.bin");
        apply_delta(None, &patch2_p, &plan2.recipe, &out3_p, &plan2.whole_hash).unwrap();
        assert_eq!(std::fs::read(&out3_p).unwrap(), data);

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn apply_delta_rejects_hostile_recipes() {
        let dir = temp_dir();
        let basis_p = dir.join("basis.bin");
        let patch_p = dir.join("patch.bin");
        let out_p = dir.join("out.bin");
        std::fs::write(&basis_p, b"0123456789").unwrap();
        std::fs::write(&patch_p, b"abcdefghij").unwrap();
        let dummy_hash = encode_hash(&blake3::hash(b"0123456789"));

        // Out-of-range Copy (offset + len > basis len).
        let r = apply_delta(
            Some(&basis_p),
            &patch_p,
            &[RecipeOp::Copy { basis_offset: 5, len: 6 }],
            &out_p,
            &dummy_hash,
        );
        assert!(r.is_err());
        assert!(!out_p.exists(), "partial out must be deleted");

        // Copy with no basis at all.
        let r = apply_delta(
            None,
            &patch_p,
            &[RecipeOp::Copy { basis_offset: 0, len: 1 }],
            &out_p,
            &dummy_hash,
        );
        assert!(r.is_err());
        assert!(!out_p.exists());

        // Out-of-range Literal (offset + len > patch len).
        let r = apply_delta(
            Some(&basis_p),
            &patch_p,
            &[RecipeOp::Literal { patch_offset: 9, len: 2 }],
            &out_p,
            &dummy_hash,
        );
        assert!(r.is_err());
        assert!(!out_p.exists());

        // Arithmetic overflow must error, never wrap or panic.
        let r = apply_delta(
            Some(&basis_p),
            &patch_p,
            &[RecipeOp::Copy { basis_offset: u64::MAX, len: 2 }],
            &out_p,
            &dummy_hash,
        );
        assert!(r.is_err());
        assert!(!out_p.exists());

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn apply_delta_rejects_wrong_hash_and_cleans_up() {
        let dir = temp_dir();
        let basis_p = dir.join("basis.bin");
        let patch_p = dir.join("patch.bin");
        let out_p = dir.join("out.bin");
        std::fs::write(&basis_p, b"hello delta world").unwrap();
        std::fs::write(&patch_p, b"").unwrap();
        let recipe = [RecipeOp::Copy { basis_offset: 0, len: 17 }];
        let wrong_hash = encode_hash(&blake3::hash(b"not the right content"));

        let r = apply_delta(Some(&basis_p), &patch_p, &recipe, &out_p, &wrong_hash);
        assert!(r.is_err());
        assert!(!out_p.exists(), "hash-mismatched out must be deleted");

        // Same recipe with the correct hash succeeds.
        let right_hash = encode_hash(&blake3::hash(b"hello delta world"));
        apply_delta(Some(&basis_p), &patch_p, &recipe, &out_p, &right_hash).unwrap();
        assert_eq!(std::fs::read(&out_p).unwrap(), b"hello delta world");

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn worthwhile_boundary() {
        let plan = |literal_bytes| DeltaPlan {
            recipe: Vec::new(),
            literal_bytes,
            reused_bytes: 0,
            whole_hash: String::new(),
        };
        assert!(delta_worthwhile(&plan(0), 100));
        assert!(delta_worthwhile(&plan(59), 100));
        assert!(!delta_worthwhile(&plan(60), 100)); // exactly 60% ⇒ not worth it
        assert!(!delta_worthwhile(&plan(100), 100));
        assert!(!delta_worthwhile(&plan(0), 0)); // empty file: 0 < 0 is false
    }

    #[test]
    fn should_attempt_delta_gate_and_env_override() {
        // Only test that touches GHOSTFTP_DELTA_MIN_SIZE; sets + removes it inline
        // so parallel test threads never see it set (no other test calls
        // should_attempt_delta).
        std::env::remove_var("GHOSTFTP_DELTA_MIN_SIZE");
        assert!(should_attempt_delta(DELTA_MIN_SIZE, true, true));
        assert!(!should_attempt_delta(DELTA_MIN_SIZE - 1, true, true));
        assert!(!should_attempt_delta(DELTA_MIN_SIZE, false, true)); // no basis
        assert!(!should_attempt_delta(DELTA_MIN_SIZE, true, false)); // disabled
        assert!(!should_attempt_delta(0, true, true));

        std::env::set_var("GHOSTFTP_DELTA_MIN_SIZE", "1024");
        assert!(should_attempt_delta(1024, true, true));
        assert!(!should_attempt_delta(1023, true, true));

        // Unparseable value falls back to the constant.
        std::env::set_var("GHOSTFTP_DELTA_MIN_SIZE", "banana");
        assert!(should_attempt_delta(DELTA_MIN_SIZE, true, true));
        assert!(!should_attempt_delta(1024, true, true));

        std::env::remove_var("GHOSTFTP_DELTA_MIN_SIZE");
        assert!(!should_attempt_delta(1024, true, true));
    }

    #[test]
    fn params_match_rejects_foreign_params() {
        let mut sig = FileSignature {
            size: 0,
            min: CHUNK_MIN,
            avg: CHUNK_AVG,
            max: CHUNK_MAX,
            chunks: Vec::new(),
            whole_hash: String::new(),
        };
        assert!(params_match(&sig));
        sig.avg = CHUNK_AVG * 2;
        assert!(!params_match(&sig));
    }

    #[test]
    fn wire_types_serde_round_trip() {
        let op = RecipeOp::Copy { basis_offset: 7, len: 99 };
        let json = serde_json::to_string(&op).unwrap();
        assert!(json.contains("\"type\":\"copy\""));
        let back: RecipeOp = serde_json::from_str(&json).unwrap();
        assert_eq!(back, op);

        let op = RecipeOp::Literal { patch_offset: 3, len: 4 };
        let json = serde_json::to_string(&op).unwrap();
        assert!(json.contains("\"type\":\"literal\""));
        assert_eq!(serde_json::from_str::<RecipeOp>(&json).unwrap(), op);

        let sig = FileSignature {
            size: 3,
            min: CHUNK_MIN,
            avg: CHUNK_AVG,
            max: CHUNK_MAX,
            chunks: vec![ChunkEntry { offset: 0, len: 3, hash: encode_hash(&blake3::hash(b"abc")) }],
            whole_hash: encode_hash(&blake3::hash(b"abc")),
        };
        let json = serde_json::to_string(&sig).unwrap();
        assert!(json.contains("\"wholeHash\""));
        let back: FileSignature = serde_json::from_str(&json).unwrap();
        assert_eq!(back.size, 3);
        assert_eq!(back.chunks.len(), 1);
        assert_eq!(back.whole_hash, sig.whole_hash);
    }

    /// Download direction: plan_download's recipe + fetched ranges must
    /// reconstruct the new file exactly, and a small edit must need only a
    /// small fraction of the file over the wire.
    #[test]
    fn plan_download_round_trip() {
        let mut rng = XorShift(0x5EED_5EED_5EED_5EED);
        for m in MUTATIONS {
            let base = rng.bytes(8 * MIB);
            let new = mutate(&base, m, &mut rng);

            let dir = temp_dir();
            let base_p = dir.join("base.bin");
            std::fs::write(&base_p, &base).unwrap();
            let basis_sig = signature_of_file(&base_p).unwrap();

            // The "remote" side: only its signature crosses the wire.
            let new_p = dir.join("new.bin");
            std::fs::write(&new_p, &new).unwrap();
            let target_sig = signature_of_file(&new_p).unwrap();

            let (plan, needed) = plan_download(&basis_sig, &target_sig).unwrap();
            assert_eq!(plan.whole_hash, target_sig.whole_hash);
            assert_eq!(plan.literal_bytes + plan.reused_bytes, new.len() as u64);
            // Needed ranges are in ascending order and inside the file.
            let mut prev_end = 0u64;
            for &(off, len) in &needed {
                assert!(off >= prev_end, "ranges must ascend");
                assert!(off + len <= new.len() as u64);
                assert!(len <= DOWNLOAD_FETCH_MAX, "range over the coalesce cap");
                prev_end = off + len;
            }

            // Fetch the needed ranges (here: straight out of the "remote" file)
            // into the patch, then reassemble.
            let patch_p = dir.join("patch.bin");
            {
                let mut patch = File::create(&patch_p).unwrap();
                for &(off, len) in &needed {
                    patch.write_all(&new[off as usize..(off + len) as usize]).unwrap();
                }
            }
            assert_eq!(
                std::fs::metadata(&patch_p).unwrap().len(),
                plan.literal_bytes,
                "patch must hold exactly the literal bytes"
            );

            let out_p = dir.join("out.bin");
            apply_delta(Some(&base_p), &patch_p, &plan.recipe, &out_p, &plan.whole_hash).unwrap();
            assert_eq!(std::fs::read(&out_p).unwrap(), new);
            let _ = std::fs::remove_dir_all(&dir);
        }
    }

    /// Adjacent misses coalesce into one fetch range; misses separated by a
    /// matched chunk stay separate.
    #[test]
    fn plan_download_coalesces_adjacent_misses() {
        let chunk = |offset: u64, byte: u8| ChunkEntry {
            offset,
            len: CHUNK_MIN,
            hash: encode_hash(&blake3::hash(&vec![byte; CHUNK_MIN as usize])),
        };
        let empty_basis = FileSignature {
            size: 0,
            min: CHUNK_MIN,
            avg: CHUNK_AVG,
            max: CHUNK_MAX,
            chunks: Vec::new(),
            whole_hash: encode_hash(&blake3::hash(b"")),
        };
        // Three consecutive unmatched chunks → ONE coalesced range.
        let target = FileSignature {
            size: 3 * CHUNK_MIN as u64,
            min: CHUNK_MIN,
            avg: CHUNK_AVG,
            max: CHUNK_MAX,
            chunks: vec![
                chunk(0, 1),
                chunk(CHUNK_MIN as u64, 2),
                chunk(2 * CHUNK_MIN as u64, 3),
            ],
            whole_hash: String::new(),
        };
        let (plan, needed) = plan_download(&empty_basis, &target).unwrap();
        assert_eq!(needed, vec![(0, 3 * CHUNK_MIN as u64)]);
        assert_eq!(plan.literal_bytes, 3 * CHUNK_MIN as u64);
        assert!(plan.recipe.iter().all(|op| matches!(op, RecipeOp::Literal { .. })));

        // Miss, hit, miss → two ranges. (The hit references chunk "1" present
        // in the basis.)
        let basis = FileSignature {
            size: CHUNK_MIN as u64,
            chunks: vec![chunk(0, 1)],
            ..empty_basis.clone()
        };
        let target2 = FileSignature {
            chunks: vec![
                chunk(0, 2),
                chunk(CHUNK_MIN as u64, 1),
                chunk(2 * CHUNK_MIN as u64, 3),
            ],
            ..target.clone()
        };
        let (plan2, needed2) = plan_download(&basis, &target2).unwrap();
        assert_eq!(needed2.len(), 2);
        assert_eq!(needed2[0], (0, CHUNK_MIN as u64));
        assert_eq!(needed2[1], (2 * CHUNK_MIN as u64, CHUNK_MIN as u64));
        assert_eq!(plan2.reused_bytes, CHUNK_MIN as u64);
    }
}
