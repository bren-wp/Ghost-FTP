# Ghost FTP Security Audit — 0.30.20

Previous canonical release: **0.30.19**.

Security posture is carried forward from the previously reviewed 0.30.15 baseline; the 0.30.16 code delta changes color tokens and does not replace a full independent security audit.

0.30.17 review scope: in-app help destinations now use the documented project GitHub repository over HTTPS; Android retains explicit URL allowlisting. The application update-service URL is deliberately unchanged, as are the transfer protocols. A completed macOS TLS/SSH identity-verification implementation is **not** claimed.

0.30.18 incremental review: About workspace UI changes preserve approved HTTPS URL allowlisting in Tauri and Android, canonical project links, and the macOS restriction against FTPS/SFTP file operations before trusted identity verification. This does not replace an independent full security audit.

0.30.19 incremental review: native OS file selection is invoked after a user-facing action and only against active connections, using unchanged authenticated transfer and overwrite-conflict controls. Directory refresh re-reads current lists, not an unprompted remote mutation. This is not an independent full security audit.

## 0.30.20 privacy and security review

- **Android diagnostics:** secrets and userinfo/token patterns are masked before the 600-character display truncation. Oversized messages are bounded before regex processing, and exceedingly long supplied secrets fail closed. Unit regressions cover truncated-password prefixes and giant server-origin error strings.
- **macOS profile backup:** an import of up to 512 profiles no longer permits the **merged total** to silently exceed the 512-profile cap. Rejected imports leave current profiles and persisted data unchanged; Swift tests exercise the failure.
- **macOS transfer-history privacy:** persisted history now carries bounded cleaned basenames, not original absolute local/remote paths. Persistence and restore tests include control characters and private path prefixes. Stored history from older versions is migrated on load: full private paths and control characters are removed and the cleaned records re-persisted. The migration is covered by a Swift regression test.
- **Windows/Linux:** hidden windows stop the otherwise unconditional one-second clock timer, lowering background activity without affecting timers for actual transfer progress. Authenticated transport, certificate and SSH host-key checks and verified updater endpoints remain unchanged.
- **Documentation screenshots:** seven README-linked PNG files are directly hashed against the successfully published Windows 0.30.19 native-window QA artifact, pinned to the exact source commit, successful run, and SHA256. No fake concept files replace live screenshots; all unrelated documentation images stay at their previous release-proven checksums.

**Outstanding:** no independent penetration test or full hardware UI/accessibility acceptance has been performed in this audit. macOS FTPS/SFTP are not enabled; plain FTP has no confidentiality protection. Do not describe those preview transports as secure or production-ready.

## Current controls

- Saved-profile secrets remain outside ordinary profile JSON and use the OS credential/keychain layer where supported.
- Ephemeral Quick Connect does not need to persist a saved profile.
- The native Tauri application keeps a restrictive CSP and no required analytics/telemetry.
- SSH host-key and TLS verification failures are handled as security failures rather than success states.
- Diagnostic/user-facing error handling retains credential redaction across desktop and Android; 0.30.12 retains centralized diagnostic redaction, strict transport-host validation and authenticated health probes while adding bounded encrypted desktop restore, Ghost FTP-only credential restore targets, secure Android window/tapjacking defenses, and device-only unlocked macOS Keychain storage.
- Transfer-history CSV export excludes raw backend error text and neutralizes spreadsheet-formula prefixes from user/server-controlled path cells.
- The desktop updater keeps its public verification key embedded in application configuration. Private updater signing material is optional and, when used, belongs only in CI secrets.
- Android FTP/FTPS/SFTP staged uploads verify remote-target state before promotion and fail closed if an existing target cannot be preserved or restored.
- Desktop SFTP Rename conflict probing distinguishes protocol-confirmed absence from permission/connection/protocol failures; ambiguous probes fail closed rather than selecting a potentially colliding path.

## Updater publication hardening

A stable GitHub release does not require private updater signing keys. When signing material is absent, verified Windows/Linux packages, Android assets, checksums and QA evidence may still be published while the in-app updater manifest and Update-Service package are omitted.

If updater signing is enabled, release orchestration requires an all-or-nothing set: signed Windows NSIS updater artifact, signed Linux AppImage updater artifact, exact-version verified `latest.json`, and matching Update-Service package containing the same manifest. Both the release workflow and `.github/scripts/publish-release.sh` fail closed if that optional set is partial or inconsistent.

## Android signing semantics

The production-identity `com.ghostftp.android` CI artifact remains intentionally unsigned unless a persistent production signing key is deliberately introduced through secure CI secrets. The separately debug-key-signed, non-debuggable `com.ghostftp.android.preview` artifact is an installable preview, not a production-signed release. Keystores/private signing keys must never be committed.

## Remaining security acceptance

- Real FTPS certificate-failure matrix.
- Keep changed SFTP host-key mismatch/replacement coverage enforced in real protocol E2E.
- Broader reconnect/timeout/server-disconnect transfer recovery.
- Dependency vulnerability/SBOM/provenance expansion across npm, Cargo, Gradle/Maven and Go.
- Target-OS signed-update/install acceptance and any future Android production-key continuity validation.

Status: source controls are present; exact-SHA CI and target-system failure-path testing remain release evidence, not assumptions.
