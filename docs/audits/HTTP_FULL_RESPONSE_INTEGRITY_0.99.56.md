# v0.99.56 HTTP/cloud full-response integrity audit

## Failure before this change

The six streaming HTTP-backed download implementations treated any successful 2xx status as a valid full-file body (HTTP and WebDAV) or relied on adapter-returned success (Dropbox, OneDrive, Google Drive, Box). A remote endpoint could respond to a **non-ranged** GET with HTTP 206 and a perfectly matching *partial* Content-Length. The download could be promoted as complete when the original remote-size estimate was unknown or stale. Other adapters did not independently compare declared Content-Length with streamed and persisted byte counts.

## Hardened behavior

1. Before creating a local temporary file, require an un-ranged HTTP 200 response with no Content-Range header, plus identity encoding when explicitly declared.
2. Parse declared Content-Length; reject nonnumeric values, counter overflow and excess response bytes *before writing* each chunk.
3. After EOF, sync the file and require on-disk byte length to match actual received bytes and the declared Content-Length, when present.
4. Leave the existing sibling-staging promotion and destination recovery-backup rules unchanged. Any failure results in staging cleanup before promotion.
5. Add focused unit tests and an actual localhost HTTP server returning 206; no credentials or external cloud accounts are used in tests.


## Whole-file staging creation hardening

Non-resumable downloads formerly opened a deterministic sibling `.part` path with `File::create`, which truncates existing entries and follows symbolic links **before** the later promotion-time file-type check. The same path was reused after a transient retry. Now each whole-file attempt uses a fresh UUID-suffixed staging path, created atomically with `create_new` / `O_EXCL`-equivalent filesystem semantics. Existing destination files are never opened during the staging write, and failed attempts remove only their own unique staging path. FTP/FTPS blocking download creates the staging file exclusively as well. SFTP retains its existing verified-prefix resume and deterministic staging path; it is not covered by the new per-attempt exclusive-create assertion.

New Rust regressions prove that existing staging bytes are not truncated, two retries use different staging paths, the real destination survives a partial write and a pre-planted symlink cannot be followed on Unix. This does not certify the absence of all local-directory replacement races or replace installed-machine security tests.

## Download overwrite-policy race audit

An initial `Skip` or `Rename` existence check cannot reserve the destination for the duration of a queued/paused transfer. The old completed-download promotion replaced an existing regular file even if another process created it after policy resolution. This violated the user's non-overwrite selection, and replay from the transfer ledger could lose the original policy.

Current behavior: an explicit `allow_replace` flag propagates from initial policy resolution through in-memory retries and the credential-free persisted retry descriptor. Overwrite retains the prior backup/rollback promotion. Skip/Rename use `move_local_noreplace` at final commit, which fails atomically if a competing destination has appeared. Agent non-overwrite downloads bypass delta's destination-modifying path and use full-file staging followed by the same no-replace operation; failed Agent no-replace promotion cleans its own staging. Legacy descriptors missing the permission default to `false`, prioritizing preservation of existing local files after recovery.

Regression coverage now includes an intervening destination creation and a legacy JSON descriptor without overwrite permission. These checks protect against final-name collisions; they do not substitute for isolation against malicious replacement of intermediate parent directories or independently tested production filesystem ACLs.

## Residual risks

A dishonest server returning HTTP 200 with absent/false Content-Length can still provide a same-length modified payload; no SHA-256 source-of-truth proof is implied. Authorized live tenant tests, crash/rollback checks and platform signing remain separate Stable gates. This change makes no pixel-accuracy claim.
