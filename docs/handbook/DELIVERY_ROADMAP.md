# Ghost FTP — implementation status, gap register and production delivery plan

**Baseline:** publicly verified v0.92.2. **Development candidate:** v0.93.0 in PR #125, subject to exact-SHA CI and review. **Status method:** repository source inspection, CI logs and feature testing. A source change is not a completed release; a feature marked planned has no implied delivery date.

## 1. Summary of what is already built

- Windows/Linux: the full Tauri/Rust desktop UI with FTP/FTPS/SFTP, connection profiles, authenticated cloud provider adapters, remote browser, transfer queue, sync/analysis, terminal/agent and desktop packaging.
- macOS: dedicated SwiftUI app, saved connection editor, macOS Keychain, protected profile import/export, FTP control/data operations and recent transfer history. FTPS/SFTP file operations are not yet implemented on this macOS Preview.
- Android: Kotlin app with FTP/FTPS/SFTP connection and actions, Android document selection, local settings backup, ephemeral in-memory secrets and bounded activity UI.
- New 0.93.0 candidate: Android and macOS direct signed HTTPS one-object PUT/GET transfers; Windows/Linux custom S3/Azure HTTPS endpoint protection. The latter already had much broader cloud integration.
- Current QA foundation: six canonical gate families, platform smoke testing, real protocol tests, source format/lint, release-lineage and asset checks. All must rerun on the exact final commit after modifications.

## 2. Prioritization rules

**P0 = security or data-loss/blocking issue.** No release should ship knowingly with a P0 regression in a shipped feature. **P1 = essential parity/reliability.** May ship only with the absent feature correctly disclosed and disabled when unsafe. **P2 = enhancement.** Schedule after P0/P1. Every delivery item requires source, regression tests, negative security tests, UI behavior, documentation and exact-SHA CI. No deadline or "100%" claim can replace that evidence.

## 3. Iteration A — repair and stabilize v0.93.0 candidate (P0)

| Workstream | Technical implementation | Completion proof |
| --- | --- | --- |
| Rust CI failure | rustfmt existing object.rs and webdav.rs test cases | cargo fmt --all -- --check on Ubuntu and Windows |
| All-platform version consistency | version.json drives Tauri/Gradle/Cargo/app metadata | Central version sync --check and version policy green |
| macOS cloud signed links | Build and execute Swift tests; enforce TLS, no redirect, ephemeral URL and no secrets | macOS CI success, invalid link/HTTP status/size tests |
| Android cloud signed links | Compile Kotlin and instrument all cloud picker flows | Android unit+emulator+real-device smoke |
| Desktop object-store endpoints | HTTPS with no userinfo/query/fragment | Rust endpoint security unit tests and provider smoke |
| Test real services | S3/MinIO, R2, B2, GCS and Azure SAS test objects without customer data | PUT/GET expected bytes, expiry/403, wrong signature, 3xx refusal |
| Cloud error handling | Bounded status, size, timeout and redaction | Negative tests for 404/429/5xx, truncated response and invalid XML/JSON |
| Artifact build | NSIS/MSI/portable, DEB/RPM/AppImage, macOS app, APK/AAB | Verified artifacts matching source SHA with smoke/provenance |
| Release gates | PR six checks; merge; main six; official signed/hashed release | Public tag immutable; assets nonzero, SHA256 manifest matches |

## 4. Iteration B — cloud network and credential hardening (P0)

- Resolve untrusted destinations safely: reject loopback, link-local, RFC1918, cloud metadata, IPv4-mapped IPv6, alternative IP encodings and DNS rebinding; proxy behavior must obey the same protections.
- Provider-signed link trust: optionally enforce known provider suffixes and user-approved custom endpoints; avoid routing uploads to arbitrary HTTPS servers merely because a query field looks like a signature.
- Token scopes: least privilege and short expiry. Put every refresh/API token behind OS secure storage and explicit revocation/clear workflows.
- Cloud response integrity: object size and expected digest/version checks where provider supports it. Never treat a completed HTTP request alone as proof of correct contents.
- Secure staging: temporary owner-only paths, atomic local replace when supported, explicit partial SAF document handling on Android, cleanup on cancel/disk full/process death.
- No secret leakage in error paths: fuzz server responses, SDK exceptions, request logging and support export. Derive status messages only from non-secret categories and HTTP status.
- Pin updater and distribution provenance. Review CI Actions permissions and dependency security. Harden backup parser and archive extraction.

## 5. Iteration C — authentic cloud accounts on macOS and Android (P1)

| Provider | Primary API/auth design | App work needed | Never implement as |
| --- | --- | --- | --- |
| Amazon S3 / R2 / B2 / compatibles | SigV4 and scoped bucket/region credentials, short-lived sessions preferred | Provider picker, bucket/object pagination, upload/download, safe copy/delete, secure key vault | Reusable plaintext secret in profile JSON |
| Google Drive | Google OAuth 2 Authorization Code + PKCE, drive.file scope if suitable | System browser account selector, token storage, root/shared drive list, paging, upload/download, disconnect/revoke | GCS service account or pasted shared link |
| Microsoft OneDrive | Microsoft identity platform OAuth/PKCE, Graph scopes | Account selection, token refresh, Drive/Item listing, resumable upload and safe permission UX | Generic S3 key workflow |
| Dropbox | OAuth/PKCE and file/content API | Folder listing, upload/download, delta cursor, revoke and conflict handling | Treat Dropbox share URL as API login |
| Box | OAuth/PKCE and Box file API | Account folder tree, upload sessions, scope/revoke, safe APIs | Treat direct signed object link as persistent login |
| Google Cloud Storage | Service account or workload identity, bucket IAM | Project/bucket selection, least privilege, GCS XML/JSON API as appropriate | Google Drive identity |
| Azure Blob | Azure identity/SAS/account-key with restrictive container scope | Container/prefix list, block blob upload/download, expiry/revocation policy | Userinfo or SAS query in profile base endpoint |
| WebDAV/Nextcloud | HTTPS, app password where relevant, PROPFIND XML parser and safe paths | Directory browse, create/rename/delete, TLS, redirect and XML hardening | Plain HTTP with password |

A full cloud feature is not complete merely when a provider appears in the connection picker. Acceptance requires authenticated listing, upload, download, navigation, errors, token expiry, revoke, pagination, permissions and destructive-action confirmation. Separate accounts must not cross-contaminate credentials or filenames. No own database/backend is inherently required for these direct provider flows.

## 6. Iteration D — macOS transport parity (P0/P1)

1. Ship authenticated FTPS rather than a reachability probe: AUTH TLS, PASV/EPSV, verify certificate chain and hostname; reject insecure downgrade.
2. Build SSH/SFTP session with strict host-key prompts/pins and no silent trust, with private-key/passphrase handling in Keychain.
3. Add genuine remote file create/rename/delete/chmod where supported; show capability-specific controls, not disabled features presented as active.
4. Add queue, retry/cancel and verified resume with per-transfer progress, bounded memory and connection cleanup.
5. Add real folder sync/diff and transparent preview of mirror deletes. Keep "connection settings backup" distinct.
6. Real macOS installer/dmg/signing/notarization and launch smoke; preserve privacy under Sandbox and file-panel scoped permissions.
7. Validate accessibility, localization and UI matching under multiple window sizes; never call app 1:1 until real captures demonstrate it.

## 7. Iteration E — Android transfer and UX parity (P0/P1)

1. Ensure Android FTP/FTPS/SFTP file operations and cloud PUT/GET survive rotation/background/low-memory without lost callbacks, inconsistent activity state or secret leaks.
2. Build a durable transfer queue that does not persist plaintext passwords; foreground service only for user-started ongoing transfers with permission-aware notifications and cancellation.
3. Atomic or safe-finalization downloads through SAF; handle partially created documents explicitly and verify expected hashes.
4. Secure saved Sites (if added): Android Keystore-backed credential wrapping, profile migration, duplicate identity checks, clear/revoke flow.
5. Add cloud provider account browsing from Iteration C, then folder sync/diff, history and richer file metadata based on backend capabilities.
6. Unify settings and error copy with supported desktop features while preserving Android touch, TalkBack, font scaling, tablet/foldable form and storage permission constraints.
7. Validate signed release APK/AAB installs on real devices, startup, file pickers, background execution and bad-network resilience.

## 8. Iteration F — Windows/Linux hardening and feature polish (P1/P2)

- Run provider-specific regression matrix for S3/R2/B2/GCS/Azure/Drive/Dropbox/OneDrive/Box/WebDAV; verify page cursors, SDK errors, cost/resource bounds, 429 backoff and token expiry.
- Improve copy/delete/move safety on object stores with two-phase UI, object generation checks and recovery rather than assuming atomic rename.
- Keep file operations consistent with RemoteFs capability flags; do not expose chmod or terminal controls on cloud backends without those capabilities.
- Improve memory/CPU transfer scheduling, throughput limits, IPv6 handling, resilience to broken proxies, offline UX and clear cancel semantics.
- Complete interactive Windows NSIS/MSI upgrade/uninstall and Linux packaging smoke with stable cache handling; never ignore failed packaging gates.
- Review keyboard navigation, 75-reference-design screenshot coverage, localization completeness and responsive windows.

## 9. Detailed quality assurance matrix

| Scope | Test mechanism | Blocking result |
| --- | --- | --- |
| Style and static analysis | rustfmt, Clippy, TypeScript type-check, Kotlin compilation, Swift compile, shell/node validation | Any compile/fmt failure blocks |
| Unit domain behavior | Protocol profiles, cloud URL validators, transfer byte offsets, credentials and error redaction | Failing or missing mandatory regression blocks |
| Network E2E | Disposable FTP/FTPS/SFTP servers plus cloud test provider accounts | Wrong TLS/SSH trust, corrupted bytes, unsafe retry or leaked secret blocks |
| App UI | Every visible button, dialogs, error states, keyboard/TalkBack/VoiceOver | Broken essential action blocks, noncritical issue documented |
| Artifact lifecycle | Windows setup/portable, Linux AppImage/DEB/RPM, macOS app, Android APK/AAB | Installation, launch or uninstall crash blocks |
| Security fuzz | URL, DNS, redirects, malicious XML/JSON, profile imports, paths | Secret disclosure, SSRF or unintended delete blocks |
| Privacy | No telemetry by default, clear/reset, backups, staged temp, logs | Plaintext secret persistence blocks |
| Distribution | Version sync, exact SHA, release notes, digest/SBOM/provenance | Bad lineage, mismatched tag/digest or missing package blocks |

## 10. Completion governance and definition of done

For every planned feature record: owner, platform, version candidate, protocol/provider, source commits, unit/E2E tests, negative tests, UI interaction, security/privacy review, documentation, real device/provider acceptance, CI run links, release artifact validation and residual known limitations. Merge only after exact-head checks pass. Do not modify an immutable tag; publish new versions only with meaningful improvements. Public release status is determined from the actual GitHub release API, not marketing copy.

## 11. Explicit non-goals for v0.93.0

No invented 100% macOS/Android feature parity, no automatic Drive OAuth without secure authorization, no fake screenshot evidence, no untested multi-file cloud sync, no background transfer promises, no plaintext-token convenience fallback, no Ghost FTP-hosted proxy or mandatory cloud account, no secretly broad cloud permissions, and no assertion that an unverified app is release-ready.

## 12. Current source documentation

Read docs/handbook/README.md and the four app-specific manuals, docs/handbook/FEATURE_MATRIX.md, docs/handbook/SECURITY_PRIVACY.md and docs/product/CLOUD_STORAGE.md. Keep all statuses accurate after every CI failure or functional change.
