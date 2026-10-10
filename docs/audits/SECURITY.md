# Ghost FTP Security Audit — 0.92.1

Previous canonical release: **0.92.0**.

0.92.1 macOS restore boundary: imported backups cannot redirect a saved UUID to a different protocol, server, port or username while retaining the UUID-bound Keychain secret. Conflicting imports abort before any profile mutation; XCTest covers both redirects and atomic rollback. This is a targeted source fix, not a penetration assessment.

0.92.0 advisory duplicate matching on macOS reads non-secret protocol, host, port and username. Neither credentials nor Keychain values are inspected; profiles are never merged, deleted or sent over the network. Unit tests cover account casing, hostname normalization, invalid inputs and membership of duplicate groups. This is not a new full penetration test.

0.91.1 focused fixes: Android error diagnostics redact FTP/SFTP URL userinfo with or without an inline password; macOS prevents silently overwriting unreadable profile data by retaining an original-byte recovery copy and disallows duplicate-ID backup imports. Windows/Linux preserve in-progress site edits across selection changes. These checks are not an independent security assessment.

0.91.0 incremental scope: Windows/Linux Site Manager duplicate discovery compares non-secret endpoint/account metadata and leaves profiles, credentials, keys, stored bookmarks and ephemeral Quick Connect records unchanged. It neither attempts network connections nor auto-merges identities. This is not a new independent penetration test.

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

## 0.30.21 targeted source changes (pending exact-SHA CI)

- **Windows/Linux diagnostic privacy:** before redaction regexes run, discard wholly oversized untrusted error messages (64 KiB cap). Unterminated private-key blocks are treated as sensitive through the end of the message, and macOS-style home-directory paths are masked. Executable TypeScript regression cases are now part of `npm run check:ui`, including truncation-boundary and oversized-input tests.
- **macOS FTP parser:** reject malformed EPSV `229` responses that contain trailing fields or omit the required final delimiter. Swift regression tests cover hostile server replies.
- These are focused hardening changes, **not** a complete penetration test or all-platform 1:1 visual acceptance. All six CI gates, merged-main validation and release verification remain required before publication.

## 0.90.0 incremental protection review (requires exact-SHA CI)

- **Windows/Linux diagnostic privacy:** URL-style FTP/SFTP userinfo, including usernames without passwords, is removed from displayed errors; the protocol and host remain readable to assist troubleshooting. Automated regressions exercise the shipped TypeScript redactor and existing query-token/size-boundary controls.
- **macOS FTP Preview:** EPSV passive-port replies now reject populated protocol/address fields instead of accepting unexpected server-supplied fields. Alternate valid delimiters remain supported; Swift tests cover accepted and rejected replies. This change does **not** enable FTPS or SFTP on macOS.
- **Release integrity:** 0.90.0 development progression is guarded by executable version-train tests, with patch fixes (0.90.1, etc.) and consecutive feature minors (0.91.0 through 0.99.0); 1.0.0 requires its own production acceptance. Existing published tags are immutable.
- **Limitations:** This is an incremental source hardening review, not a completed independent penetration test, every-control hardware smoke test or 75-screen installed-build visual signoff.

## 0.90.1 incremental data-integrity and stability review

- **Settings migration / privacy:** only delete legacy settings after exact readback of **every serialized key/value pair**. Matching row counts are not sufficient proof of a durable migration. Extra or incorrect database rows cannot erase the known-good copy.
- **Startup resilience:** invalid or corrupted legacy settings JSON is retained for recovery and ignored without aborting application startup. Legacy sensitive terminal/notification history is still purged.
- **Password-generation performance:** invalid, non-finite or excessively large requested lengths are rejected before the synchronous cryptographic loop, preventing UI hangs.
- **Verification:** executable tests for exact readback, corrupted/partial writes, write exceptions, existing DB precedence, malformed legacy JSON, privacy cleanup and bounded random password generation. This does not represent an independent penetration test or 75-screen 1:1 signoff.
