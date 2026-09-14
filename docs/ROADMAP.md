# Ghost FTP roadmap

Ghost FTP **0.0.6** is the current source/release candidate. The roadmap prioritizes correctness, security, privacy, reliability and verifiable cross-platform behavior before new surface area.

## Current 0.0.6 foundation

The current release gate includes:

- native Windows and Linux desktop applications backed by one typed Engine;
- a production-signed public Android APK with FTP and strict explicit FTPS, while Android SFTP remains hidden until maintained strict host-key verification exists;
- deterministic public Chrome, Edge and Firefox helper ZIPs built from one canonical shared runtime;
- an active native macOS development/source frontend, without claiming Developer ID signing or notarization;
- FTP, explicit FTPS and strict desktop SFTP workflows;
- protected local profile/credential behavior and explicit credential-save consent;
- transfer lifecycle, retries, pause/resume/cancel, Top/Up/Down/Bottom priority and truthful progress;
- current-folder filtering, bounded recursive search, conservative directory comparison, synchronized navigation and bookmarks/start directories;
- built-in Remote Edit with bounded UTF-8 handling, conflict checks and staged/read-back verification;
- two public universal Windows executables containing native **x64, x86 and ARM64** payloads selected locally, with `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` until maintained native ARM64 execution evidence exists;
- twelve canonical Debian/Ubuntu/Fedora/Portable Linux artifacts;
- 24-language local desktop catalog with English default/fallback;
- exact-head Android APK, browser package, macOS development-app, Windows/Linux build and authentic Windows/Linux/Android UI evidence;
- signed-only official Windows publication and protected production-signed Android publication;
- exact GitHub Release/GHCR digest readback and latest-only retention.

## 0.0.6 navigation work

Ghost FTP 0.0.6 retains the **non-destructive current-folder filter** for maintained desktop panes. It filters only already-loaded entries, performs **no hidden filesystem/network scan**, preserves an authoritative unfiltered snapshot and composes with shared sorting.

Ghost FTP 0.0.6 retains the **bounded recursive local/server search**, deliberately separate from current-folder filtering because it performs additional listing I/O. Search is cancellable, bounded by depth/item/result/time limits, avoids intentional symlink/reparse traversal and treats results as navigation hints that require fresh-list validation.

Ghost FTP 0.0.6 retains **conservative directory comparison and synchronized navigation**. Comparison is read-only and fail-closed for duplicate names, uncertain metadata, type mismatches and symlink-like entries. Synchronized navigation commits only after both target directories are freshly validated.

Ghost FTP 0.0.6 retains reusable **navigation bookmarks and profile start directories**. Remote navigation remains bound to protocol/host/port/username identity and is revalidated against the active session.

## High-value power-user lane

### P0 — directory comparison and synchronized navigation

**Status: implemented in Ghost FTP 0.0.6.** Windows and Linux expose shared conservative comparison states and Android exposes the maintained mobile comparison/navigation workflow without treating comparison as mutation authority.

### P0 — bounded recursive local/server search

**Status: implemented in Ghost FTP 0.0.6.** Current-folder filtering remains I/O-free; recursive search is a separate explicit bounded action with cancellation and fresh-list navigation.

### P0 — bandwidth-aware transfer controls

**Status: implemented in Ghost FTP 0.0.6.** Desktop upload/download limits are shared validated runtime policy using binary KiB/s with `0 = unlimited`, conservative aggregate directional scheduling and real transport enforcement.

### P0 — queue priority and reorder

**Status: implemented in Ghost FTP 0.0.6.** A selected queued desktop transfer can move Top, Up, Down or Bottom without rewriting transfer identity, running/terminal history slots or connection ownership.

See [Queue priority and reordering](QUEUE-PRIORITY.md).

### P1 — navigation bookmarks and profile start directories

**Status: implemented in Ghost FTP 0.0.6.** Bookmarks contain non-secret navigation metadata, local navigation is freshly validated and remote activation is account/session bound.

See [Navigation bookmarks and profile start directories](NAVIGATION-BOOKMARKS.md).

### P1 — stronger interrupted-transfer resume

Verified resume remains future work where protocol/server semantics can prove that a partial object is the intended object. Unsupported cases must safely restart rather than blindly append.

### P1 — multiple live sessions / tabs

Future multi-session work must bind every queue item to one explicit session identity, isolate close/cancel behavior and prevent implicit secret copying or unbounded session creation.

### P1 — proxy / jump-host boundary

Any future proxy/jump-host support must keep destination and intermediary trust explicit and must not weaken TLS or SSH host verification.

### P2 — profile import/export

Non-secret profile exchange may be added. Secret export must remain separately opt-in and strongly protected; plaintext credential export is not an acceptable default.

## UI/product rules for new features

A visible control is not a feature definition. Every new control requires a real runtime owner, enabled-state policy, success/failure/cancellation behavior, user-safe diagnostics, localization, regression coverage, active documentation and authentic UI evidence when the maintained surface changes.

## Performance direction

Optimization targets measured hotspots: avoid unnecessary full redraws/scans, keep UI work out of network critical paths, bound progress publication and search results, minimize allocations/copies, batch invalidation and reject goroutine/stale-callback leaks.

## Security/privacy direction

Security hardening favors deterministic rejection over permissive fallback. New features may not introduce hidden cloud state, telemetry, credential replication, trust-all TLS/SSH behavior or silent secure-to-plain downgrade.

Official Windows publication requires trusted Authenticode. Official Android publication requires the protected publisher keystore plus exact signer SHA-256 verification. Browser helpers remain local parsers/copy helpers with **no supported browser-to-desktop** handoff. macOS remains development/source only until real Developer ID signing and Apple notarization succeed.

## Non-negotiable constraints

Future work must preserve local path containment and symlink/reparse safety, fail-closed transfer cleanup/commit behavior, trusted Linux transport/AskPass provenance, exact source/version binding, exactly two public Windows executables, trusted Windows signatures, honest `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci`, no generated/self-signed production publisher identity, verified release/package publication and no unreviewed external Go dependency.

## Release direction

The public sequence is `0.0.1`, `0.0.2`, `0.0.3`, `0.0.4`, `0.0.5`, `0.0.6`, and so on. `0.0.0` is reserved. A current release identity is never rewritten in place.

The Ghost FTP 0.0.6 public contract is **18 platform artifacts / 21 public files**: Windows, Linux and production-signed Android application artifacts, three official browser helper packages and release metadata. macOS remains a separately validated development/source surface.

## Definition of roadmap completion

A roadmap item is complete only when code, regression tests, security/privacy implications, active documentation and production CI/release gates agree on the behavior. Git history remains engineering provenance; the public catalog follows the verified latest-only lifecycle.
