# Ghost FTP Security Audit — 0.20.9

Previous canonical release: **0.20.8**.

## Current controls

- Saved-profile secrets remain outside ordinary profile JSON and use the OS credential/keychain layer where supported.
- Ephemeral Quick Connect does not need to persist a saved profile.
- The native Tauri application keeps a restrictive CSP and no required analytics/telemetry.
- SSH host-key and TLS verification failures are handled as security failures rather than success states.
- Diagnostic/user-facing error handling retains credential redaction.
- Transfer-history CSV export excludes raw backend error text and neutralizes spreadsheet-formula prefixes from user/server-controlled path cells.
- The desktop updater keeps its public verification key embedded in application configuration; private updater signing material belongs only in CI secrets.

## Stable updater hardening

For `channel: stable`, release orchestration now requires:

- signed Windows NSIS updater artifact;
- signed Linux AppImage updater artifact;
- exact-version verified `latest.json`;
- matching Update-Service package containing the same manifest.

Both the release workflow and `.github/scripts/publish-release.sh` fail closed before stable tag/release mutation if updater proof is missing or inconsistent. Explicit preview/CI builds may omit updater signatures only when they are not stable publications.

## Android signing semantics

The production-identity `com.ghostftp.android` CI artifact remains intentionally unsigned unless a persistent production signing key is deliberately introduced through secure CI secrets. The separately debug-key-signed, non-debuggable `com.ghostftp.android.preview` artifact is an installable preview, not a production-signed release. Keystores/private signing keys must never be committed.

## Remaining security acceptance

- Real FTPS certificate-failure matrix.
- Unknown/changed SFTP host-key acceptance.
- Broader reconnect/timeout/server-disconnect transfer recovery.
- Dependency vulnerability/SBOM/provenance expansion across npm, Cargo, Gradle/Maven and Go.
- Target-OS signed-update/install acceptance and any future Android production-key continuity validation.

Status: source controls are present; exact-SHA CI and target-system failure-path testing remain release evidence, not assumptions.
