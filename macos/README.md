# Ghost FTP for macOS

The `macos/` tree is the native macOS desktop surface for Ghost FTP. It is a platform frontend for the same product and shared `internal/api.Engine` used by the maintained desktop code; it is not a separate FTP/SFTP implementation or a reduced edition.

**Windows desktop remains the canonical visual and behavior reference.** macOS uses native AppKit windowing, Keychain, file panels, accessibility and platform signing primitives where appropriate, while capability, validation, security, state ownership and transfer semantics stay aligned with the shared desktop contracts.

## Functional status

The macOS action inventory in `PARITY.md` is complete. The native AppKit application includes real engine-backed implementations for:

- FTP, explicit FTPS and SFTP connection/disconnection;
- SFTP private-key authentication and strict first-contact host-key trust;
- Site Manager with Save Profile, Remove Profile, Duplicate and Connect;
- explicit saved-credential consent and native Keychain-backed durable profile protection;
- Bookmarks with shared account-bound remote navigation;
- Local and Remote panes, refresh/up/navigation and native local folder selection;
- Local/Remote New Folder, Rename and Delete with stale-view rejection;
- Local/Remote Unicode-aware filtering and bounded recursive search;
- Remote Permissions/CHMOD;
- Remote Edit with shared conflict detection and verified read-back;
- Directory Compare with atomic paired navigation;
- Upload and Download through the shared transfer engine;
- native Transfer Queue with Pause, Resume, Cancel, Retry, Clear Finished and four-way queued reordering;
- Settings backed by shared `model.Settings`, including the shared 24-language registry, Classic Light/Dark appearance, parallelism, independent bandwidth limits, timeout, retries, conflict policy and delete confirmation;
- privacy-safe About and Diagnostics surfaces.

Every visible parity action is backed by real shared-engine/platform behavior and regression coverage. There are no intentional decorative parity controls.

## Security and privacy boundary

Ghost FTP for macOS keeps the same security/privacy contract as the rest of the desktop product:

- no telemetry, analytics, advertising, tracking or hidden Ghost FTP backend;
- no browser IPC, localhost app server or generic JSON credential dispatcher;
- Swift never receives protected saved-profile credential blobs;
- saved profile secrets use a native `Security.framework` Keychain-held AES-256 wrapping key with `WhenUnlockedThisDeviceOnly`;
- runtime secrets remain short-lived and same-user scoped;
- SFTP host-key verification/pinning remains fail-closed;
- file mutations and transfers stay bound to authoritative engine snapshots and navigation generations;
- symbolic links are not silently traversed by transfer actions;
- no new external runtime dependency is required by the native macOS app.

## Visual contract

The Mac workspace follows the maintained Windows hierarchy while using native macOS window chrome, focus behavior and accessibility APIs.

- Classic Light: workspace `#EEF1F5`, panel `#F6F8FB`, list `#FAFBFD`.
- Dark: workspace `#0B0F17`, panel `#121824`, list `#161D2A`.
- English is the canonical default/fallback and the shared registry exposes the same 24 languages.
- FTP, explicit FTPS and SFTP remain the desktop protocol set.

## Development build

Build the universal Intel + Apple Silicon development app on macOS with:

```bash
bash macos/BUILD.sh
```

The output is:

```text
macos/dist/Ghost-FTP-<VERSION>-macOS.app.zip
```

The development artifact is deliberately ad-hoc signed. It is suitable for CI/native regression validation, not for public Gatekeeper distribution.

## Developer ID signing and notarization

`macos/SIGN_AND_NOTARIZE.sh` is the fail-closed production distribution path. It:

1. builds the same universal app with `macos/BUILD.sh`;
2. requires an installed **Developer ID Application** identity;
3. replaces every ad-hoc signature explicitly from the innermost nested code outward;
4. enables Hardened Runtime and a secure timestamp;
5. verifies signatures and rejects `com.apple.security.get-task-allow`;
6. submits the signed app archive using `xcrun notarytool`;
7. continues only after Apple reports an accepted notarization;
8. staples and validates the ticket;
9. performs a Gatekeeper `spctl` assessment; and
10. only then emits `Ghost-FTP-<VERSION>-macOS-notarized.app.zip`.

The reusable signing script accepts only Keychain references:

```bash
export MACOS_DEVELOPER_IDENTITY='Developer ID Application: Example Company (TEAMID)'
export MACOS_NOTARY_KEYCHAIN_PROFILE='ghostftp-notary'
bash macos/SIGN_AND_NOTARIZE.sh
```

Raw certificate/private-key/notary credentials must never be committed to the repository or passed to this reusable signer.

## GitHub Actions production artifact

`.github/workflows/macos-production.yml` is a manual, environment-gated workflow that prepares the ephemeral Keychain required on a clean GitHub-hosted macOS runner, runs the same signer/notarizer and uploads the verified notarized ZIP as a workflow artifact. Configure these secrets in the protected `macos-production` GitHub Environment:

- `MACOS_DEVELOPER_ID_P12_BASE64`
- `MACOS_DEVELOPER_ID_P12_PASSWORD`
- `MACOS_DEVELOPER_IDENTITY`
- `APPLE_NOTARY_API_KEY_P8`
- `APPLE_NOTARY_API_KEY_ID`
- `APPLE_NOTARY_ISSUER_ID`

The workflow destroys the temporary signing Keychain/files in an `always()` cleanup step. It deliberately does **not** modify or upload to an existing public GitHub Release. Promotion of a notarized artifact into a public release remains an explicit release decision rather than an automatic side effect of configuring credentials.

## Release truthfulness

Source/native functionality is complete and continuously validated on universal macOS builds. A public macOS binary can only be called production-distributable after a real Developer ID certificate and Apple notarization credentials have been supplied and `macos-production.yml` (or the equivalent local `SIGN_AND_NOTARIZE.sh` path) has succeeded. Those private Apple credentials are external security material and are intentionally not stored in this repository.

See `PARITY.md` for the completed Windows ↔ macOS action/security contract.
