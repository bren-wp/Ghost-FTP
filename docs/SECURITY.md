# Security

Ghost FTP keeps transport, credential, remote-path, account-state, archive-processing and filesystem checks fail-closed.

**Current Ghost FTP release: 1.1.0**

## Desktop core

FTP over TLS validates certificates. SFTP pins and verifies the host key. Uploads use stable local snapshots, remote writes use temporary staging and destination revalidation, and overwrite paths use backup/rollback logic. Local recursive operations guard against symlinks, junctions and reparse-point traversal.

Private upload-source snapshots retain their owned cleanup path if fail-closed removal itself fails, allowing cleanup to be retried instead of losing the only reference to residual sensitive data. After a downloaded replacement is committed locally, failure to remove a rollback copy is reported instead of silently returning success.

Credentials must never be logged. Windows saved profile secrets are protected with DPAPI. Runtime secret material is kept ephemeral on supported platforms. External processes receive a minimized environment and bounded output. OpenSSH AskPass is parent/token constrained, clears inherited credential state before secret use and refuses unknown/MFA-style prompts instead of supplying a stored secret.

The maintained desktop toolchain is pinned by CI. Production builds disable Go telemetry and external module downloads before compiling.

## Windows installer and upgrade boundary

Ghost FTP Setup uses an application-only verified payload and does not publish a standalone `Uninstall.exe`.

The installer validates its embedded payload manifest and digest, stages verified bytes, protects installation paths against unsafe redirection/reparse behavior and uses rollback-aware file/registry transactions.

A small set of historical identifiers is intentionally retained only to preserve safe upgrades from existing installations:

- installed executable name `GhostFTP.exe`;
- old Windows App Paths entry for `GhostFTP.exe`;
- old GhostFTP uninstall registry key used for migration/cleanup;
- installer payload member name `GhostFTP.exe`.

These are compatibility identifiers, not public branding. Setup dialogs, build outputs, PE VERSIONINFO and manifests use **Ghost FTP**. Public Windows release files are `Ghost-FTP-X.Y.Z-Setup-x64.exe`, `Ghost-FTP-X.Y.Z-Setup-x86.exe` and the byte-identical x32 alias of x86.

## Android

Android uses a separate native protocol boundary:

- SFTP requires an OpenSSH-style `SHA256:` host-key fingerprint.
- Fingerprints are decoded and validated before use by the SSH layer.
- Permissive/promiscuous SFTP verifiers are forbidden.
- Explicit/implicit FTPS use platform trust, endpoint/hostname validation and protected data channels.
- Plain FTP remains available only as an explicitly unencrypted compatibility mode.
- Remote paths fail closed on traversal, dot components, duplicate separators, backslashes, NULs and noncanonical names.
- Host/port input is bounded and credential control characters are rejected.
- Passwords/passphrases are session-only and are not persisted by Ghost FTP.
- Remembered connection metadata excludes secrets.
- Password UI state is cleared after connection attempts and lifecycle teardown.
- Storage Access Framework is used instead of broad storage permissions.
- Lifecycle generation guards prevent stale callbacks from mutating a newer/disconnected session.

The Android package/application identifier may retain a legacy `GhostFTP` namespace for installed-app identity compatibility. The visible application name is **Ghost FTP**.

CI produces an installable APK. A production Play signing key must remain outside the repository; a debug-signed CI artifact must never be represented as store-signed production software.

## iOS

The native iOS application currently uses platform networking for its supported FTP/FTPS transport surface.

- TLS validation remains platform-controlled; there is no trust-all callback or global ATS bypass.
- Remote paths and server-reported roots reject traversal, backslashes, duplicate separators, NULs and dot components.
- EPSV is preferred and PASV fallback does not trust a server-supplied alternate host.
- FTP commands/arguments and credentials reject CR/LF/NUL injection.
- Reads/listings are bounded and downloads use temporary files instead of unbounded in-memory accumulation.
- Session generation prevents stale asynchronous work from mutating a newer session.
- Credential UI/runtime copies are cleared after use where practical.
- Persisted connection metadata excludes secrets and uses restrictive Keychain accessibility.
- Failed/stale temporary downloads are cleaned up.
- No analytics SDK, WebView wrapper or fixed Ghost FTP backend endpoint is part of the app.

The existing Xcode project/bundle identifiers may retain legacy `GhostFTP` naming for application identity compatibility. Public application naming is **Ghost FTP**.

The CI IPA is a real arm64 device build but is unsigned. Normal device/TestFlight/App Store distribution requires a legitimate Apple signing identity and provisioning profile managed outside the repository.

## Web/PWA

`GhostFTP WEB/` is the legacy-named source directory for the **Ghost FTP** shared-hosting application. The source path and some internal PHP symbols are retained for compatibility; the product/UI/package metadata are Ghost FTP.

Authentication, encrypted profiles, runtime state, archive processing and temporary-transfer budgets are security state:

- JSON state reads/writes are bounded and fail closed on malformed/corrupt primary state.
- Login/registration rate-limit budgets are consumed before sensitive account mutation work.
- Saved connection secrets are bound to the exact endpoint/account/key identity.
- SFTP requires a pinned SHA-256 host fingerprint before a profile can be persisted and verifies the connected server key against that pin again at the client boundary.
- Inline editor/new-file content is centrally bounded and local staging must be complete before any remote promotion.
- SFTP key temp files are permission-restricted before key material is written, and uploads verify the resulting remote size when the server exposes it.
- Destructive/batch mutation inputs fail closed before partial application when their shape or source set is invalid.
- Multi-file upload validates the complete request shape, temporary upload identity and normalized remote destination set before the first remote mutation.
- Atomic overwrite recovery treats failed backup restoration and ambiguous promotion outcomes as explicit recoverable states: the original backup name is retained for manual recovery without exposing nested transport error text.
- Backup creation is also reconciled after rename errors: Ghost FTP re-reads the target and candidate backup paths so a move-then-error response cannot hide the confirmed recovery filename or accidentally continue promotion.
- Staging cleanup ownership begins before FTP/SFTP upload starts, so a partial transport failure triggers verified remote-temp cleanup; an unverifiable cleanup exposes only the generated staging recovery name.
- Remote temporary cleanup re-checks absence after a delete error before escalating, so servers that report an error after actually removing a temp object do not produce false residual-data warnings.
- After successful promotion, failure to remove the previous-version backup is reported as an explicit partial-success state with the backup name and a do-not-retry warning instead of silently retaining old remote data.
- Public Web error responses expose deliberate validation messages but replace unexpected PHP/extension Throwable details with a generic internal-error response.
- Known application validation failures use HTTP 400; unexpected internal `Throwable` failures use HTTP 500 without exposing their raw message to the client.
- The same public-error mapping is used by account, registration, settings, user-administration, login-migration and setup HTML flows; nested internal exceptions are preserved as causes without concatenating their raw text into a user-visible `RuntimeException`.
- Browser editor/new-file writes use only `RemoteOperations::writeAtomic()`; the unused direct transport `write()` contract has been removed to prevent a weaker duplicate write path from drifting back into use.
- Password changes/rehashes use generation-aware compare-and-swap behavior.
- User deletion is two-phase/retryable and does not traverse unsafe workspace-root symlinks.
- Encryption keys are not rotated over pre-existing encrypted data during recovery.
- Connection target validation resolves/validates exact targets before transport use.
- FTP/FTPS and SFTP downloads are bounded; partial/oversized temporary files are cleaned.
- ZIP extraction validates archive topology, existing-remote conflicts and decompressed size before remote writes.
- Diagnostics are authorization-protected because hosting/runtime capability information is operationally sensitive.

Ghost FTP Web uses SameSite=Strict/HttpOnly cookies, CSRF tokens, cross-site POST filtering, CSP, HSTS on HTTPS, no-store behavior for sensitive surfaces and explicit no-index protections. Saved connection secrets use authenticated encryption with an installation-specific key.

The PWA cache namespace is `ghostftp-static-vX.Y.Z`; activation removes superseded Ghost FTP caches and legacy `GhostFTP-static-*` caches. Navigation, API, account, setup, diagnostics, download and preview responses are never stored in the offline cache.

`scripts/package_web.py` builds `Ghost-FTP-X.Y.Z-Web.zip` from tracked production files only and rejects symlinks, unsafe paths and case-fold collisions. Runtime users/config/cache/backup data must not enter the public archive.

## Repository, privacy and release integrity

The repository-wide audit checks tracked files for case-insensitive path collisions, Windows-reserved components, symlinks, generated/cache artifacts, temporary one-shot workflows, malformed UTF-8 text, NUL/BOM issues, trailing whitespace, missing final newlines, merge-conflict markers and stale current-release references.

Temporary audit/patch workflows must run only on isolated branches and must be removed before production validation or merge; `scripts/audit_repository.py` rejects any tracked `.github/workflows/one-shot-*` file in a release tree.

Security/privacy audits additionally protect:

- no application telemetry/analytics vendor integrations;
- no fixed runtime HTTP(S) destination in the desktop core;
- no plaintext credential artifacts written for AskPass;
- minimized proxy/network-tool environment;
- SFTP fingerprint/trust invariants;
- local/remote path boundaries;
- transfer ownership/generation checks;
- safe state-file opening and cleanup behavior.

`.github/workflows/release.yml` is the single production publication path. It assembles exactly **10 platform artifacts** plus `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, for **13 public files** total.

Before publication, the workflow verifies that `main` still points to the release commit. Existing `ghostftp-vX.Y.Z` tags are never moved to another commit. The historical `v1.0.0` and other GhostFTP tags remain untouched.

## Package integrity and signing

Release consumers should verify `SHA256.txt` before installation.

Windows PE metadata is verified for architecture, GUI subsystem, resource presence and platform mitigations. Authenticode signing status is reported explicitly; Verified Publisher requires a legitimate external Ghost FTP code-signing certificate.

Android CI signing is installable/development signing unless an external production identity is configured. iOS release artifacts are unsigned. Publisher credentials and private signing keys must not be committed to this repository.

See [Signing](SIGNING.md), [Release verification](RELEASE-VERIFICATION.md) and [GitHub Releases](GITHUB-RELEASES.md).

## Reporting

Report vulnerabilities through the repository Security policy. Never publish working passwords, private keys, signing credentials, production endpoints or customer data in a public issue.
