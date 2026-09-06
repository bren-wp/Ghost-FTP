# Changelog

## 0.1.1 - 2026-09-06 Beta

### Added

- Native dependency-free Linux X11/XWayland graphical desktop with Quick Connect, SFTP host trust, profiles, dual file panes, file/tree transfer actions and transfer queue controls.
- Linux graphical local/remote New folder, Rename and Delete actions plus remote Permissions/chmod.
- Linux graphical transfer Settings overlay for parallelism, conflict policy, retry count/delay, connection timeout and delete confirmation.
- Production X11 runtime smoke coverage that verifies protocol setup, a real mapped Ghost FTP window and stable process lifetime.

### Changed

- Desktop palette now mirrors the maintained Ghost FTP Web brand tokens (`#080A0F`, `#0F131C`, `#151A25`, `#5277F5`, `#7293FF`) while retaining native Win32/X11 rendering.
- Linux desktop packaging launches the graphical application by default when a display is present; the hardened terminal remains an explicit/headless fallback.
- Documentation, package examples and authentic Windows runtime screenshots are refreshed for the 0.1.1 Beta line.
- Implicit FTPS (`ftpsi`, conventional port 990) is documented as a maintained desktop protocol option.

### Fixed / hardened

- Corrected raw X11 CreateGC value-mask handling and GetKeyboardMapping wire offsets found by real runtime testing.
- Destructive Linux GUI actions honor the canonical confirm-delete setting and retain typed Engine validation.
- No Linux password/passphrase persistence was introduced; existing secret-lifetime boundaries remain unchanged.


## 0.1.0 - Unreleased Beta

### Active version baseline

- Reset the active product-development baseline to **0.1.0 Beta** while preserving all existing implementation work, commits, tags and historical release records.
- Defined the active maturity path as `0.1.0 Beta → 0.x.y Beta → 1.0.0 stable`.
- Reserved **1.0.0** for the first release that passes the complete stable product-quality gate rather than assigning a stable version before the Windows/Linux application is fully finished and verified.
- Kept the root `VERSION` file as the single machine-readable `X.Y.Z` source for Windows PE metadata, Linux package metadata, build scripts, release tags and CI.
- Added a user-facing Beta display policy for pre-1.0 versions without corrupting numeric package metadata.
- Made GitHub Releases for `0.x.y` versions publish and verify as prereleases with a Beta release title and explicit `RELEASE_CHANNEL=beta` metadata.
- Kept Windows Setup and Windows Portable on one shared canonical version; they never have independent version counters.

### Windows native FTP workflow

- Preserved and refined the native graphite/navy Windows interface rather than replacing prior work.
- Reworked the main workspace around a professional dual-pane FTP layout with balanced local/server panes, visible session state and a full-width transfer queue.
- Improved layout persistence across resize, DPI, protocol and language changes so the refined workspace does not fall back to the previous geometry.
- Added one-click **Sites** access to the main toolbar while retaining the same Site Manager implementation used by the native menu.
- Expanded Site Manager with protocol, server, port, username, password, local path, remote path, SFTP private key, SFTP key passphrase and security state.
- Fixed Quick connection so the Site Manager **Connect** action performs a real connection instead of only copying fields back to the main window.
- Kept saved credentials protected: stored passwords/passphrases are not displayed back as plaintext when a saved site is selected.
- Kept controls fail-closed while connected or during a connection transition.

### Windows/Linux reliability and packaging

- Preserved the shared Windows/Linux core, protocol, transfer, settings, localization and security work completed before the version-baseline reset.
- Kept Windows x64/x86 Setup and Portable production builds driven by the canonical `VERSION`.
- Kept Linux amd64/arm64/i386 package generation driven by the same canonical version.
- Preserved the nine-platform-artifact / twelve-public-file release contract with SHA-256 verification.
- Kept CI gates for formatting, race tests, vet, repository integrity, security, privacy, dependencies, localization, documentation, Web companion integrity and Windows/Linux production builds.

### Documentation

- Added a dedicated versioning policy describing the 0.x Beta progression, the 1.0.0 stable gate and the Setup/Portable shared-version invariant.
- Reframed the README, documentation index, roadmap and release documentation around the active 0.1.0 Beta baseline.
- Preserved previous changelog/release-history entries as historical provenance instead of deleting or rewriting them.

## 2.0.0 - Unreleased historical development snapshot

> This was an internal pre-publication development line. It is preserved as historical engineering context, but it was superseded by the active 0.1.0 Beta maturity baseline and is not the current product version.

### Platform consolidation

- Started a new semantic-major line because the supported application-platform contract changed from the former multi-platform 1.x matrix to **Windows + Linux**.
- Removed Android, iOS and macOS application source trees from active 2.x development together with their packaging and platform-specific audit scripts.
- Preserved the complete 1.x Git/tag/release history instead of rewriting or deleting historical provenance.
- Added a fail-closed platform-contract audit that rejects reintroduction of retired application roots/tooling and requires both Windows and Linux CI/release jobs.
- Rebuilt CI around one shared quality/security/documentation gate plus Windows and Linux production build gates.
- Rebuilt publishing around 9 application artifacts: 5 Windows and 4 Linux artifacts, plus release notes, build metadata and SHA-256 manifest for 12 public files total.
- Linux DEB packages are now published individually for amd64, arm64 and i386 in addition to the multiarch bundle.

### Linux parity and connection reliability

- Restricted the non-Windows desktop frontend build target explicitly to Linux for the 2.x application line.
- Fixed a major SFTP parity defect where Linux previously required a private key and rejected a non-empty key passphrase.
- Linux SFTP now uses the same shared `ConnectionConfig`/`Engine` authentication model as Windows: password authentication when no key is supplied, or private-key authentication with an optional passphrase.
- Preserved explicit SFTP host-key fingerprint trust and the shared fail-closed remote/session layer for both Windows and Linux.
- Added Linux transfer-queue controls for listing jobs, pause, resume, cancel, retry and clear-finished operations through the same transfer manager used by Windows.
- Added Linux inspection/editing of shared validated transfer settings including parallelism, conflict policy, retries, retry delay, connection timeout and delete confirmation.
- Added Linux saved-profile inspection through the shared profile store.
- Retained localized remote navigation, listing, create, rename, delete, chmod, upload and download behavior.
- Added a compact premium terminal header while retaining a dependency-light terminal presentation instead of introducing a cross-platform GUI framework.

### Windows design and interaction quality

- Refined the native Windows dark theme to a deeper graphite/navy surface system with softer contrast for long transfer sessions.
- Improved action hierarchy for primary, destructive, default and subtle owner-drawn buttons.
- Increased button corner radius, spacing and focus inset while preserving keyboard focus visibility and existing high-DPI behavior.
- Kept the Windows UI native and dependency-free at the Go module level; no external GUI framework was introduced.

### Localization

- Retained the canonical English-first registry of **24 languages**: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.
- English remains the first/default locale and safe fallback for invalid or missing locale state.
- Removed obsolete iOS localization release gates from the active audit while continuing to verify exact desktop catalog integrity, format verbs and translation coverage.
- CI now explicitly verifies Windows live localization, all-language Windows Setup primary copy and Linux runtime language switching.

### Security, privacy and dependency policy

- Updated security auditing for the Windows/Linux product boundary, including Linux SFTP password/key/passphrase parity, transfer controls and shared settings paths.
- Retained protected runtime secret handling, AskPass no-secret-file policy, SFTP private-key reparse/symlink checks, endpoint-bound host trust, session-close race protection and root-delete protections.
- Retained FTP/FTPS proxy-environment isolation and prohibition on fully disabling FTPS revocation checking.
- Removed obsolete mobile dependency allowlists from the active dependency audit.
- Desktop/core remains standard-library-only at the Go module level: no external Go modules, `go.sum` dependency graph or vendored module tree.
- Made the remaining OS transport prerequisites explicit and audited instead of making a misleading zero-runtime-dependency claim: FTP/FTPS uses OS `curl`; SFTP uses OS OpenSSH `ssh`/`sftp`.
- Retained fail-closed telemetry/analytics/ads/crash-SDK rejection and runtime fixed-URL privacy checks.

### Documentation and release quality

- Rewrote the root README around the then-planned 2.x Windows/Linux product contract, supported protocols, transfer settings, 24 languages, privacy/security boundaries, dependency provenance and release artifacts.
- Added/updated detailed documentation for platform parity, installation, dependencies, releases, verification, security, testing, localization and roadmap direction.
- Clarified that the existing Ghost FTP Web companion source is maintained separately from the Windows/Linux desktop application release contract.
- Preserved detailed notes for all previously published 1.x releases below so every version continues to document what changed and why.

## 1.1.0 - 2026-09-05

- Expanded the canonical English-first desktop/setup language registry from 18 to 24 languages, adding Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.
- Added regional locale normalization including Norwegian `nb`/`nn` and Simplified Chinese aliases.
- Replaced the weak eight-string supplemental-locale threshold with measured translation coverage and a 30-string translated-core floor for supplemental catalogs.
- Added localized primary Windows Setup confirmation/completion/launch/warning copy for all 24 canonical languages while retaining English as the safe fallback.
- Added complete Android resource catalogs for the same 24-language set and a fail-closed CI audit for exact key parity, format placeholders, locale-directory drift and minimum real translation coverage.
- Added native iOS core localization resources for all 24 canonical languages, wired through the Xcode resource build phase with `zh-Hans` and `nb` platform mappings and fail-closed parity/packaging audits.
- Added an English-first Web/PWA 24-language core registry with per-user language persistence, regional alias normalization, a visible language selector and English fallback for untranslated shell strings.
- Replaced Croatian-only Web file-browser/PWA fallback text and hard-coded Croatian locale filtering with the persisted canonical locale while preserving existing CSRF, session, no-cache and privacy boundaries.
- Consolidated destination-conflict behavior into one canonical `conflictPolicy` (`skip`, `replace`, `replace_backup`) while preserving and synchronizing legacy overwrite booleans for backward compatibility.
- Replaced the two potentially contradictory Windows overwrite Yes/No prompts with one native three-state conflict-policy selector.
- Added regression coverage for legacy conflict-policy migration, contradictory compatibility flags, invalid new saves and fail-closed recovery from unknown persisted policy values.
- Improved the dependency provenance contract: no external Go modules, zero third-party Web Composer packages, exactly pinned Android dependencies, and fail-closed rejection of tracking/ads/crash SDKs or dynamic versions.
- Added detailed localization, dependency, settings and immutable release-history documentation.

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
- Started the historical Ghost FTP semantic-version line at **1.0.0** with sequential patch releases.
- Introduced namespaced release tags (`ghostftp-vX.Y.Z`) so current releases never collide with historical generic tags.
- Standardized Linux packaging as `ghost-ftp` with the `ghostftp` executable and Ghost FTP desktop entry.
- Established the multi-platform Release contract and SHA-256/build-metadata verification model.
- Preserved strict path validation, transfer staging/rollback, SFTP host-key verification, encrypted profile secrets, rate limiting, session hardening and release provenance controls.
- Kept mobile and desktop signing status explicit rather than representing unsigned/debug-signed artifacts as store-signed packages.

## Historical provenance

All previously created Git tags, releases and commits remain immutable for provenance, troubleshooting and reproducibility. The historical 2.0 development snapshot and 1.x notes above describe the repository state and platform matrix that existed at those times. They are retained intentionally and are not the active maturity baseline. Going forward, the active product line starts at **0.1.0 Beta**, advances through `0.x.y` Beta milestones and reaches the first stable line only at **1.0.0** after the complete stability gate is satisfied.
