# Changelog

## 1.1.0 - Unreleased

- Expanded the canonical English-first desktop/setup language registry from 18 to 24 languages, adding Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.
- Added regional locale normalization including Norwegian `nb`/`nn` and Simplified Chinese aliases.
- Replaced the weak eight-string supplemental-locale threshold with measured translation coverage and a 30-string translated-core floor for supplemental catalogs.
- Added localized primary Windows Setup confirmation/completion/launch/warning copy for all 24 canonical languages while retaining English as the safe fallback.
- Added complete Android resource catalogs for the same 24-language set and a fail-closed CI audit for exact key parity, format placeholders, locale-directory drift and minimum real translation coverage.
- Consolidated destination-conflict behavior into one canonical `conflictPolicy` (`skip`, `replace`, `replace_backup`) while preserving and synchronizing legacy overwrite booleans for backward compatibility.
- Replaced the two potentially contradictory Windows overwrite Yes/No prompts with one native three-state conflict-policy selector.
- Added regression coverage for legacy conflict-policy migration, contradictory compatibility flags, invalid new saves and fail-closed recovery from unknown persisted policy values.
- Improved the dependency provenance contract: no external Go modules, zero third-party Web Composer packages, exactly pinned Android dependencies, and fail-closed rejection of tracking/ads/crash SDKs or dynamic versions.
- Added detailed localization, dependency, settings and immutable release-history documentation.
- 1.1.0 remains unreleased until iOS and Web/PWA localization plus remaining protocol/option work pass the full cross-platform release gate.

## 1.0.7 - 2026-09-05

- Took atomic-upload staging cleanup ownership before FTP/SFTP transfer begins, closing the gap where a partial remote staging file could survive when the transport failed before returning success.
- Added verified cleanup for residual staging files; if deletion/absence cannot be confirmed, Ghost FTP reports the generated staging recovery name without exposing nested transport error text.
- Reconciles ambiguous backup-creation renames where a server moves the original file but still reports an error, surfacing the confirmed recovery backup name and cleaning staging without activating the replacement.
- Stopped silently swallowing failure to remove an old-version backup after successful promotion. The new file remains active and the user receives an explicit partial-success warning with the retained backup name and a do-not-retry instruction.
- Added runtime regression coverage for partial-upload cleanup success, cleanup failure, target preservation, old-backup retention and nested-error non-disclosure.
- Extended the Web source audit so future changes cannot reintroduce post-success staging ownership or silent backup-deletion swallowing.

## 1.0.6 - 2026-09-05

- Hardened Web atomic overwrite recovery so a failed promotion cannot silently hide a failed restoration of the original file.
- Detects ambiguous remote rename outcomes where the server reports a promotion error after the destination path becomes available; both the new destination and original recovery backup are preserved for inspection.
- Recovery errors expose the generated backup name needed for manual restoration while never concatenating raw nested transport exception text into the public message.
- Added a focused runtime regression covering failed restore, ambiguous promotion, backup preservation and staging cleanup.
- Removed the leaked one-shot 1.0.5 audit workflow from production source and made the repository audit reject tracked `.github/workflows/one-shot-*` helpers permanently.

## 1.0.5 - 2026-09-05

- Extended the shared Web public-error boundary to account, login migration, registration, user administration, application settings and first-run setup HTML flows so unexpected Throwable details are no longer rendered in pages.
- Removed nested Throwable-message concatenation from move fallback and user-workspace deletion wrappers, preserving the original exception only as the internal cause.
- Removed the unused transport-level `write()` interface and FTP/SFTP implementations; all real inline writes remain on the bounded, exact-write, atomic `RemoteOperations::writeAtomic()` path.
- Aligned the remote read interface default with the canonical 4 MiB browser editor limit.
- Added regression/audit coverage for HTML error disclosure, nested exception wrapping and dead transport write-contract removal.
- Completed additional visible Web/PWA branding cleanup so user-facing labels, setup/login text, install prompts, image alt text and default installation name use **Ghost FTP**.

## 1.0.4 - 2026-09-05

- Added a shared fail-closed Web public-error boundary: deliberate validation failures remain actionable, while unexpected PHP/extension Throwable details return a generic 500 response instead of leaking internals.
- Applied the same safe error mapping to API, file download, ZIP download and image preview endpoints, and stopped appending text errors after binary response headers have already been sent.
- Made multi-file upload fully preflighted: request shape, upload status, temporary files, remote paths and duplicate targets are validated before the first remote upload mutation.
- Made ZIP path-list parsing reject malformed non-string rows instead of silently skipping them.
- Added regression tests and source audits for public error disclosure, upload preflight ordering and strict path-list parsing.
- Removed remaining visible `GhostFTP` diagnostics text in favor of the canonical **Ghost FTP** brand.

## 1.0.3 - 2026-09-05

- Required a canonical pinned SHA-256 SFTP host fingerprint before Web profiles can be persisted, while retaining the connection-boundary pin requirement as defense in depth.
- Centralized the Web editor/new-file 4 MiB content limit and required complete local staging writes before atomic remote promotion.
- Hardened SFTP temporary key handling with fail-closed Unix `0600` permissions before key material is written and exact write-length checks.
- Added SFTP upload size verification against remote metadata when available so staging fails closed on incomplete transfers.
- Made destructive Web batch deletion use a two-pass preflight so every item and type is validated before the first remote delete.
- Made batch rename reject malformed rows and duplicate source paths before any remote mutation.
- Removed duplicate archive-processing code commentary and expanded regression coverage for the new fail-closed boundaries.
- Bound documentation current-release markers and the 10-artifact/13-file release contract to the canonical VERSION audit.

## 1.0.2 - 2026-09-04

- Bounded the non-Windows in-memory runtime-secret store and made capacity exhaustion fail closed instead of allowing unbounded growth.
- Added runtime-secret regression coverage for copy isolation, capacity limits, cleanup and capacity reuse.
- Hardened Web credential envelope parsing with strict driver validation, maximum encoded size, explicit truncation checks and authenticated-tamper rejection tests.
- Removed a stale hard-coded release number from Composer metadata and added a runtime metadata regression test so human-readable package descriptions cannot drift from canonical versioning.
- Advanced the web PWA cache namespace and all canonical version surfaces to 1.0.2.
- Refreshed root documentation to describe the current Ghost FTP security, release and package contract without version-specific metadata drift.

## 1.0.1 - 2026-09-04

- Completed the hard-cut Ghost FTP identity across tracked paths, namespaces and runtime identifiers.
- Standardized Android, iOS, Go, Windows installer and web technical identities on GhostFTP/Ghost FTP.
- Added a fail-closed repository brand audit.
- Added Windows portable x64/x86 artifacts and GitHub Packages publication under `GhostFTP`.
- Verified the complete multi-platform release and registry readback pipeline.

## 1.0.0 - 2026-09-04

- Established **Ghost FTP** as the canonical public product identity across Windows, Linux, macOS, Android, iOS and Web/PWA surfaces.
- Started the Ghost FTP semantic-version line at **1.0.0** with sequential patch releases.
- Introduced namespaced release tags (`ghostftp-vX.Y.Z`) so current releases never collide with historical generic tags.
- Standardized Linux packaging as `ghost-ftp` with the `ghostftp` executable and Ghost FTP desktop entry.
- Established the multi-platform Release contract and SHA-256/build-metadata verification model.
- Preserved strict path validation, transfer staging/rollback, SFTP host-key verification, encrypted profile secrets, rate limiting, session hardening and release provenance controls.
- Kept mobile and desktop signing status explicit rather than representing unsigned/debug-signed artifacts as store-signed packages.

## Historical provenance

Git tags and commits created before the current Ghost FTP release sequence remain immutable for repository provenance and reproducibility. Current product releases exclusively use the `ghostftp-vX.Y.Z` namespace.
