# Ghost FTP release history

## 0.1.1 — 2026-09-06 Beta

Ghost FTP 0.1.1 advances the pre-1.0 Windows/Linux line with a real Linux graphical frontend, stronger runtime verification and a unified Ghost FTP visual system. Windows remains native Win32; Linux uses a dependency-free raw X11/XWayland-compatible frontend over the same typed Engine and retains the terminal for headless/explicit fallback.

The release also refreshes native Windows runtime screenshots, documents implicit FTPS, aligns desktop palette tokens with the maintained Web brand source, and preserves the existing no-telemetry, SFTP host-trust, credential-lifetime, filesystem and release-readback protections.


This document expands the concise `CHANGELOG.md` into a durable explanation of the current Ghost FTP release line. Published tags and their artifacts remain immutable; documentation may add context but never rewrites release provenance.

## 0.1.0 — 2026-09-05 — Beta

Theme: **new Beta maturity baseline, premium Windows/Linux FTP workflow, signing hardening and release-contract cleanup**.

Status:

`0.1.0` is the active Ghost FTP Beta baseline. All `0.x.y` releases are intentionally prereleases while the Windows/Linux application is being completed and stabilized. The first version eligible for the stable channel is `1.0.0`.

This version reset does **not** delete or undo previous engineering work. The existing Windows/Linux functionality, security hardening, localization, transfer behavior and historical release provenance are preserved. The reset only establishes a more realistic pre-1.0 maturity line for the current product.

What changed — version and release lifecycle:

- Reset the active machine-readable `VERSION` to `0.1.0` while preserving historical tags, commits, release notes and implementation work.
- Defined the active progression as `0.1.0 Beta → 0.x.y Beta → 1.0.0 stable`.
- Made every pre-1.0 GitHub Release a Beta/Prerelease and reserved stable publication for `1.0.0` or later.
- Kept Windows Setup and Portable on one canonical version instead of independent counters.
- Kept the namespaced immutable tag format `ghostftp-vX.Y.Z`.
- Preserved the current Windows/Linux release contract: nine platform artifacts plus `RELEASE-NOTES.txt`, `BUILD-METADATA.txt` and `SHA256.txt`, for twelve public release files total.

What changed — Windows experience:

- Preserved the native graphite/navy Ghost FTP design while reorganizing the workspace around the information hierarchy proven useful in professional dual-pane FTP clients.
- Added a native top menu for File, Sites, Transfers, View and Help without replacing the existing native Win32 frontend.
- Added a real native Site Manager with saved-site selection and connection details in one clear workspace.
- Added one-click Sites access from the main toolbar.
- Rebalanced the local/server panes to prioritize file browsing and transfers rather than oversized decorative controls.
- Promoted current connection/activity state into a readable session strip.
- Kept a persistent full-width transfer queue at the bottom with compact pause/resume/cancel/retry/clear controls.
- Fixed clipped file-operation labels at smaller supported window sizes.
- Kept the disconnected SERVER surface visually consistent with the dark theme.
- Made the refined layout persist across resize, DPI, language and protocol changes.
- Hid SFTP-only private-key/passphrase controls for FTP/FTPS instead of reserving dead visual space.
- Expanded Site Manager with protocol, server, port, username, password, local path, remote path, SFTP private-key path, optional key passphrase and security state.
- Fixed Quick connection so **Connect** actually starts a connection instead of merely copying values back to the main window.
- Kept stored password/passphrase material write-only from the user's perspective: saved secrets are not revealed back as plaintext in Site Manager.

What changed — Linux parity and storage:

- Preserved the same typed `internal/api.Engine`, transfer manager, remote manager, settings, profiles and localization layer used by Windows.
- Linux supports FTP, explicit FTPS and SFTP using password authentication or private key plus optional passphrase as appropriate.
- Preserved explicit SFTP host-key fingerprint trust.
- Linux exposes local and remote working directories, guarded local file operations, remote file operations, file transfers and bounded folder/tree transfers.
- Linux exposes transfer queue controls, shared validated settings and saved-profile management through the terminal frontend.
- Added authenticated AES-GCM protection for Linux profile state with a per-user key file.
- The verified Ghost FTP leaf state directory is restricted to `0700`; the key file is a regular non-symlink file restricted to `0600`.
- Added regression coverage for symlink directory/key substitution and AES-GCM tampering.

What changed — Windows signing:

- Added `scripts/Sign-WindowsArtifacts.ps1` as the canonical Authenticode signing helper.
- Added `scripts/New-DevCodeSigningCertificate.ps1` for short-lived RSA-3072/SHA-256 self-signed development certificates used only to test the signing pipeline.
- Private signing keys, PFX/P12 files and signing passwords are explicitly blocked from tracked source.
- Windows Portable is signed after final PE-resource mutation and **before** it is embedded into Setup, ensuring Setup contains the same signed client bytes that are published separately.
- Setup is signed only after its own final PE resources are applied.
- SHA-256 release manifests are generated only after all signing mutations are complete.
- GitHub production publication supports a protected secret-backed PFX identity reconstructed only inside the ephemeral Windows runner and removed after use.
- Development/self-signed signatures are never represented as a Microsoft-trusted publisher identity.
- Stable Windows publication at `1.0.0` or later is blocked unless a trusted production Authenticode identity is configured.

What changed — CI, security and repository hygiene:

- Go formatting, race tests, vet, repository audits, platform-contract audits, security/privacy audits, localization, documentation, Web-companion audit and release audit remain mandatory gates.
- Windows x64/x86 production Setup/Portable builds remain mandatory.
- Linux amd64/arm64/i386 DEB builds and metadata checks remain mandatory.
- Added an Authenticode private-key pipeline smoke test using a temporary self-signed development identity on copies of real Windows artifacts.
- Added repository auditing that rejects tracked PFX/P12/private-key material and generated signing directories.
- Removed the redundant `BUILD-WINDOWS.cmd` wrapper so `BUILD-WINDOWS.ps1` is the single canonical Windows build entry point.
- Removed retired Android/iOS/Xcode ignore rules from the active `.gitignore` and retained only Windows/Linux/current tooling boundaries.
- Reworked stale regression tests so they protect the active 0.1.0 Beta lifecycle instead of forcing obsolete 2.x marketing text back into current documentation.
- Preserved delayed GitHub Release asset readback, repeated `main` verification, remote `SHA256.txt` comparison and immutable-tag protection.

What changed — documentation and authentic screenshots:

- Reframed README, installation, architecture, security, testing, signing, release, release-verification, localization, dependency, contributor and roadmap documentation around 0.1.0 Beta and Windows/Linux-only active application support.
- Preserved historical sections that truthfully describe earlier platform matrices instead of rewriting old release facts.
- Added an authentic Windows UI capture workflow that builds and launches the real production Portable executable and captures the native main workspace and Site Manager rather than generating mockups.
- The screenshot workflow can persist verified captures under `docs/images/` so README images remain tied to the actual built client.

Release readiness:

`0.1.0` must not be called stable. A Beta release is publishable only after the current source commit passes the complete Core, Windows and Linux gates and the assembled release metadata/assets are verified. Stable status is reserved for the later `1.0.0` gate.

## 2.0.0 — 2026-09-05 — Historical development snapshot

Theme: **Windows/Linux consolidation, premium desktop hardening and explicit platform scope**.

Status:

`2.0.0` was an internal pre-publication development/release-candidate concept used while the repository was being consolidated to Windows and Linux. It was later superseded by the active `0.1.0 Beta` maturity baseline so the product could continue through a realistic pre-1.0 stabilization cycle. This section is retained as engineering history and is not the current product version.

Why this snapshot existed:

The application-platform contract changed from the 1.x multi-platform model to **Windows and Linux only**. The 2.0 snapshot initially represented that support reduction as a semantic-major change. The later 0.1.0 Beta reset kept the implementation and Windows/Linux scope but changed the maturity/version policy. Historical Android, iOS and macOS source/releases remain available through immutable Git history and older tags; those targets are not part of the maintained current application source tree or production build matrix.

What changed — platform architecture:

- Removed active Android, iOS and macOS application source trees and their package/audit tooling from the maintained tree.
- Added a fail-closed platform-contract audit that rejects reintroduction of retired application roots/tooling and requires Windows/Linux production jobs.
- Consolidated CI into three maintained gates: shared core/security/documentation, Windows x64/x86 production build and Linux amd64/arm64/i386 production build.
- Consolidated the desktop release contract to nine platform artifacts and twelve public release files including `RELEASE-NOTES.txt`, `BUILD-METADATA.txt` and `SHA256.txt`.
- Kept the existing Ghost FTP Web companion as a separately audited source surface; it is not counted as a Windows/Linux application-platform artifact in the desktop release contract.
- Restricted Linux desktop/terminal build tags explicitly to Linux rather than relying on broad `!windows` selectors that could accidentally reactivate an unsupported platform.

What changed — Linux parity and FTP workflow:

- Linux SFTP authentication now supports password authentication when no private key is supplied.
- Linux SFTP supports a private key with an optional passphrase through the same typed Engine/remote manager used by Windows.
- Removed the former Linux frontend restriction that effectively required a private key and rejected normal passphrase-based key authentication.
- Preserved explicit SFTP host-key fingerprint trust and endpoint-bound stored trust behavior.
- Added transfer-queue inspection and controls for pause, resume, cancel, retry and clear-finished operations.
- Added validated runtime transfer settings for parallelism, conflict policy, automatic retry count, retry delay, connection timeout and delete confirmation.
- Added saved-profile inspection while keeping password/passphrase values out of terminal output.
- Preserved remote list/navigation/create/rename/delete/chmod/upload/download operations on the shared typed engine instead of adding a separate Linux protocol implementation.
- Preserved the bounded terminal parser: no shell evaluation, bounded command length/argument count and rejection of embedded NUL/newline control characters.

What changed — Windows experience and setup:

- Refreshed the native Windows visual system with a deeper graphite/navy palette, stronger contrast, clearer primary/destructive/subtle action hierarchy and more consistent interaction states.
- Improved owner-drawn button rendering and focus treatment while retaining the native Win32 frontend and avoiding an additional GUI framework dependency.
- Kept Windows x64 and x86 Setup plus Portable artifacts as canonical production outputs; x32 remains an explicitly verified byte-identical release alias of x86 where the public release contract requires it.
- Preserved the localized Windows Setup flow and installer transaction boundaries: verified embedded payload, staged write, digest validation, rollback-aware replacement and guarded registry/application-path updates.
- Continued to report Authenticode state explicitly rather than presenting unsigned binaries as Verified Publisher software.

What changed — localization:

- English remains the canonical source language, first language in the registry and safe fallback for damaged/unknown locale state.
- The maintained registry remains at 24 languages: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.
- Localization CI was updated for the Windows/Linux contract: Windows live localization, localized Windows Setup, Linux runtime language switching, key/catalog compatibility, format verbs and meaningful translation coverage.
- Obsolete iOS/Android localization-resource requirements were removed from the active gate without rewriting their historical 1.x release record.

What changed — security and privacy:

- Desktop/core remains free of external Go modules and bundled third-party Go libraries.
- The dependency audit explicitly distinguishes library dependencies from operating-system transport prerequisites instead of making a misleading zero-runtime-dependency claim.
- FTP/FTPS use the operating-system `curl` executable; SFTP uses OpenSSH `ssh`/`sftp`. Linux packages declare those runtime prerequisites and Windows relies on suitable system-provided components.
- Application telemetry, analytics, advertising and external crash-reporting SDKs remain forbidden by policy and automated audits.
- Fixed runtime analytics/update HTTP destinations remain blocked in the desktop core.
- Saved credentials remain endpoint/account/private-key bound and cannot silently cross to a changed connection identity.
- SFTP AskPass credential material remains protected without writing a reusable plaintext password/passphrase file.
- SFTP host-key changes remain fail-closed.
- Download staging, local rename/delete, recursive delete, remote session shutdown and profile/state-file operations retain no-follow/no-replace/bounded-lifecycle protections.
- Filesystem regressions explicitly test Linux no-replace rename behavior and Windows `MoveFileExW` no-replace/write-through behavior while ensuring retired application targets remain absent.

What changed — release integrity and regression quality:

- Reworked stale 1.x Python regression tests so they validate the maintained Windows/Linux platform contract instead of attempting to load removed Android/iOS/macOS files.
- Restored stronger production-release readback after the platform-workflow rewrite.
- After release publication, CI verifies the complete remote asset set by file name and size, downloads the published `SHA256.txt` and compares it byte-for-byte with the local release manifest.
- The release is verified immediately and again after a delay; `main` is checked again between those validations so publication cannot silently succeed after the branch moves.
- Existing `ghostftp-vX.Y.Z` tags are never moved to another commit.
- GitHub Packages/NuGet publication includes remote version readback before the workflow reports success.
- The build and audit suite verifies repository hygiene, version binding, localization, privacy, security, dependencies, Web-companion integrity and documentation before production artifacts are accepted.

Validation evidence recorded for the 2.0 development snapshot:

- Go race tests and vet passed across the maintained core packages.
- Repository, platform-contract, dependency, version, localization, security, privacy, documentation, Web companion and release audits passed.
- The Python regression suite passed after the Windows/Linux contract migration.
- Linux production packaging passed for amd64, arm64 and i386 including Debian package metadata verification.
- Windows production build passed for x64 and x86 Setup/Portable artifacts including release verification and artifact upload.

Historical release-integrity note:

The 2.0 development snapshot planned the same nine-platform-artifact/twelve-public-file Windows/Linux contract later retained by 0.1.0 Beta. The snapshot itself is historical engineering provenance, not the active release line.

## 1.1.0 — 2026-09-05

Theme: **24-language localization, settings clarity and cross-platform quality**.

What changed:

- English remains the canonical primary/fallback language while the supported registry expands to 24 languages: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.
- Regional aliases normalize safely, including Simplified Chinese variants and Norwegian `nb`/`nn` forms.
- Translation quality is measured rather than inferred from key presence; weak mostly-English supplemental catalogs are blocked by minimum real-translation coverage.
- Windows Setup localizes its primary confirmation/completion/launch/warning flow for the canonical language registry while keeping English as the safe technical fallback.
- Android ships native localized resources for the same 24-language set with fail-closed key, placeholder, locale-directory and translation-coverage audits.
- iOS ships native core localization resources for all 24 canonical languages. The Xcode project packages them through a real localized resource variant group, with `zh-Hans` and `nb` platform mappings.
- Web/PWA adds a per-user 24-language core registry and selector through the existing preference API. High-frequency dynamic file-browser copy is localized with English fallback, and locale-sensitive filtering is no longer hard-coded to Croatian.
- Destination-conflict behavior is consolidated behind one canonical three-state `conflictPolicy` (`skip`, `replace`, `replace_backup`) while legacy booleans remain synchronized for backward compatibility.
- The Windows settings UI replaces potentially contradictory overwrite prompts with one native three-state selector.
- Dependency provenance remains fail-closed: no external Go modules, zero third-party Web Composer packages, pinned Android dependencies, and rejection of telemetry/ads/crash SDKs or dynamic dependency versions.
- Cross-platform release documentation and audits are bound to the canonical `VERSION` so stale release metadata blocks publication.

Security/stability effect:

The release reduces configuration ambiguity, prevents locale/catalog drift from silently reaching production, preserves English recovery paths when translated copy is unavailable, and keeps localization changes inside the existing credential, CSRF, session, transport and signing trust boundaries.

Release integrity:

`ghostftp-v1.1.0` may be published only from the verified `main` merge commit after the full Core, Windows, Linux, macOS, Android and iOS release workflow succeeds and the GitHub Release/package readback checks pass. Signing limitations remain explicit: Windows/macOS publisher signing requires external identities, Android public CI is development/debug-signed unless production credentials are supplied, and iOS remains arm64 unsigned for external signing/provisioning.

## 1.0.7 — 2026-09-05

Theme: **remote atomic-upload residual-data integrity**.

What changed:

- Atomic Web/PWA upload staging became owned before FTP/SFTP transport begins. A transport that creates remote bytes and then fails can no longer bypass staging cleanup just because `upload()` did not return success.
- Residual staging cleanup is verified. If deletion reports an error, Ghost FTP checks whether the temporary object actually disappeared before escalating.
- If cleanup cannot be confirmed, the user receives the generated recovery staging name without nested transport exception details.
- Backup creation now reconciles ambiguous server behavior where the original file is moved to a backup but the server still reports a rename failure. The confirmed backup name is surfaced and the replacement is not activated in an uncertain state.
- Failure to remove the old backup after a successful promotion is no longer silently treated as success. The newly promoted target stays active, while Ghost FTP reports a partial-success state and tells the user not to repeat the upload.
- New runtime regressions cover partial-upload cleanup, retained staging, ambiguous backup creation, target preservation, old-backup retention and non-disclosure of internal error text.

Security/stability effect:

The release closes edge cases where user data was preserved but its recovery location could become unclear, or where obsolete remote content could remain silently after a successful replacement.

Release integrity:

`ghostftp-v1.0.7` was published from the verified `main` merge commit after the full CI and production publish workflow completed successfully.

## 1.0.6 — 2026-09-05

Theme: **atomic overwrite recovery**.

What changed:

- A failed promotion can no longer hide a failed restoration of the original remote file.
- Ambiguous promotion outcomes are detected when a server reports a rename failure even though the destination appears.
- The original recovery backup name is retained and exposed safely when manual restoration may be needed.
- Nested transport errors stay internal instead of being concatenated into public Web messages.
- Added focused runtime regression coverage for failed restore, ambiguous promotion, backup preservation and staging cleanup.
- Removed a leaked one-shot audit workflow from production source and made repository hygiene permanently reject tracked `.github/workflows/one-shot-*` files.

Security/stability effect:

Recovery state became explicit and observable instead of relying on a single server return code during rename/promotion.

## 1.0.5 — 2026-09-05

Theme: **public-error containment and write-path simplification**.

What changed:

- Extended the safe public-error boundary to account, registration, user administration, settings, setup and login-migration HTML flows.
- Unexpected PHP/extension exceptions no longer render raw internal details in public pages.
- Removed nested raw exception-message concatenation from move fallback and workspace deletion wrappers.
- Removed the unused direct remote `write()` transport contract. Browser editor/new-file writes now use the single bounded atomic write path.
- Aligned remote read defaults with the canonical 4 MiB editor limit.
- Added regressions for HTML disclosure, nested exception wrapping and dead write-contract removal.
- Completed another pass of visible Ghost FTP branding cleanup in Web/PWA surfaces.

Security/stability effect:

There are fewer competing write implementations and fewer places where infrastructure/extension error text can cross the public trust boundary.

## 1.0.4 — 2026-09-05

Theme: **Web fail-closed validation before mutation**.

What changed:

- Introduced a shared public-error mapper: deliberate validation failures remain actionable, unexpected internal failures become generic 500 responses.
- Applied safe error handling to API, direct download, archive download and preview endpoints.
- Stopped writing text errors after binary-response headers had already started.
- Multi-file uploads now validate request shape, upload state, temporary file identity, normalized remote paths and duplicate targets before the first remote mutation.
- ZIP path-list parsing rejects malformed rows instead of silently ignoring them.
- Added source and runtime regressions for the new preflight ordering and public-error behavior.

Security/stability effect:

Invalid batch requests fail before partially mutating the remote server, and internal implementation details are less likely to leak to unauthenticated/public surfaces.

## 1.0.3 — 2026-09-05

Theme: **SFTP trust and bounded Web operations**.

What changed:

- Web profiles require a canonical pinned SHA-256 SFTP host fingerprint before they can be persisted.
- The connected server key is verified again at the transport boundary.
- Browser editor/new-file content is centrally bounded and local staging must complete before remote promotion.
- SFTP temporary private-key files receive restrictive permissions before key material is written.
- SFTP uploads verify remote size when the server exposes reliable metadata.
- Batch deletion uses a two-pass preflight; batch rename rejects malformed rows and duplicate source paths before mutation.
- Documentation/version/release audits became more tightly bound to the canonical `VERSION` file and release artifact contract.

Security/stability effect:

SFTP identity, temporary key handling and destructive batch operations gained stronger fail-closed guarantees.

## 1.0.2 — 2026-09-04

Theme: **bounded secrets and credential envelopes**.

What changed:

- Non-Windows in-memory runtime-secret storage became capacity-bounded and fails closed when exhausted.
- Added tests for secret copy isolation, capacity limits, cleanup and capacity reuse.
- Hardened Web credential-envelope parsing with strict driver validation, encoded-size limits, truncation checks and authenticated-tamper rejection.
- Removed stale hard-coded release metadata from Composer surfaces and added a regression to keep package metadata synchronized with canonical versioning.
- Advanced PWA cache/version surfaces consistently.

Security/stability effect:

Secret state and encrypted credential inputs gained explicit resource and parsing limits instead of relying on unbounded or permissive behavior.

## 1.0.1 — 2026-09-04

Theme: **technical identity and release completeness**.

What changed:

- Completed the Ghost FTP technical-identity cutover across tracked runtime paths and identifiers while retaining only compatibility identifiers that are needed for safe upgrades.
- Standardized Android, iOS, Go, Windows installer and Web technical identities.
- Added a fail-closed repository brand audit.
- Added Windows portable x64/x86 artifacts.
- Added GitHub Packages publication under the `GhostFTP` package ID.
- Verified release and package-registry readback as part of publication.

Release effect:

The product gained one canonical current identity and a more complete Windows distribution surface without rewriting historical generic tags.

## 1.0.0 — 2026-09-04

Theme: **first canonical Ghost FTP release line**.

What changed:

- Established **Ghost FTP** as the public product identity across Windows, Linux, macOS, Android, iOS and Web/PWA.
- Started the dedicated semantic-version line at 1.0.0.
- Introduced `ghostftp-vX.Y.Z` tags so current releases do not collide with historical generic tags.
- Standardized Linux packaging as `ghost-ftp` with the `ghostftp` executable and desktop entry.
- Established the multi-platform release contract, SHA-256 manifest and build metadata.
- Preserved strict path validation, staged transfers/rollback, SFTP host-key verification, protected profile secrets, Web rate limiting/session hardening and release provenance checks.
- Kept signing status explicit: debug-signed Android and unsigned iOS artifacts are never represented as store-signed production packages.

Release contract established:

The public release pipeline builds platform-specific deliverables, records checksums/build metadata and verifies the published release remotely before a workflow is allowed to report success.

## Historical repository provenance

Older generic tags and pre-Ghost-FTP repository history remain immutable. They may contain earlier technical names, package identities or architecture decisions. Current releases use the dedicated Ghost FTP namespace and current documentation; historical tags are retained for reproducibility, not rewritten to look like modern releases.
