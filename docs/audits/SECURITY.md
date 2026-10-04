# Ghost FTP Security Audit — 0.30.6

Previous canonical release: **0.30.5**.

## Current controls

- Saved-profile secrets remain outside ordinary profile JSON and use the OS credential/keychain layer where supported.
- Ephemeral Quick Connect does not need to persist a saved profile.
- The native Tauri application keeps a restrictive CSP and no required analytics/telemetry.
- SSH host-key and TLS verification failures are handled as security failures rather than success states.
- Diagnostic/user-facing error handling retains credential redaction.
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
