# Ghost FTP release history

This document expands the concise `CHANGELOG.md` into a durable explanation of the current Ghost FTP release line. Published tags and their artifacts remain immutable; documentation may add context but never rewrites release provenance.

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
