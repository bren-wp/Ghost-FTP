# Changelog

## Unreleased - 0.0.7 hardening

### Reliability and user experience

- Hardened the browser Native Messaging bridge with bounded pending-request lifetimes, duplicate-request rejection and single-flight transfer polling so an unresponsive local bridge cannot leave the extension permanently busy.
- Android saved-site loading now isolates malformed stored entries instead of discarding every otherwise valid saved server when one profile is damaged.
- Simplified Android Remote Edit errors so normal users receive actionable file, connection and conflict guidance instead of implementation terminology.
- Kept strict UTF-8, binary-file, size, line-ending and remote-change protections for Remote Edit while reducing unnecessary SHA-256 hex-formatting allocations.
- Added regression coverage for the optimized digest path and browser bridge lifecycle boundaries.

### Maintenance

- Production cleanup continues to remove obsolete implementation wording and stale documentation without weakening protocol, privacy, signing or release safety checks.
- Release documentation is kept aligned with the immutable published 0.0.6 asset set rather than historical development targets.

## 0.0.6 - 2026-09-14

### Public Android release

- Promoted Android from development-only packaging to the canonical Ghost FTP 0.0.6 public release through a protected production-signing path.
- The release workflow requires `GHOSTFTP_ANDROID_KEYSTORE_BASE64`, `GHOSTFTP_ANDROID_KEYSTORE_PASSWORD`, `GHOSTFTP_ANDROID_KEY_ALIAS`, `GHOSTFTP_ANDROID_KEY_PASSWORD` and `GHOSTFTP_ANDROID_CERT_SHA256`; publication fails closed if any required value is absent or invalid.
- The signed `Ghost-FTP-0.0.6-Android.apk` is verified with Android `apksigner` and the signing certificate SHA-256 fingerprint is matched against the protected expected fingerprint before publication.
- Ordinary CI keeps the explicitly separate `Ghost-FTP-Android-dev.apk` identity and uses only an ephemeral signing identity to exercise the signing pipeline.
- Android SFTP remains intentionally hidden until strict, maintained host-key verification exists. Public Android signing does not weaken this security boundary.

### Android parity, privacy and lifecycle hardening

- Preserved the non-destructive current-folder filter over already-loaded entries with no hidden filesystem/network scan.
- Preserved bounded recursive local/server search as a separate cancellable, deadline-limited workflow rather than conflating it with current-folder filtering.
- Navigation bookmarks and profile start directories remain bound to fresh validation, account identity and stale-session protection across maintained desktop surfaces.
- Added local SAF create-directory, rename and delete controls plus remote FTP/FTPS create-directory, rename, delete and permission changes with fresh server-list readback and destructive confirmations.
- Hardened remote mutation paths against root aliases, dot-segment traversal, command-control injection and ambiguous mutation completion.
- Reject malformed MLSD child names containing path separators or ISO control characters before they can become actionable Android remote-entry paths, while preserving valid Unicode filenames.
- Bound pending connection and transfer lifecycle to the owning Android Activity/session generation and prevented stale callbacks from publishing into replacement state.
- Redacted authentication/server reply bodies from user-facing credential failure paths and hardened passive FTP response validation.
- Preserved authentic Android emulator UI evidence in the exact-head cross-platform evidence bundle.

### Browser helper public packages

- Added deterministic public release ZIPs for Chrome, Microsoft Edge, Firefox and Opera:
  - `Ghost-FTP-0.0.6-Chrome-Extension.zip`
  - `Ghost-FTP-0.0.6-Edge-Extension.zip`
  - `Ghost-FTP-0.0.6-Firefox-Extension.zip`
  - `Ghost-FTP-0.0.6-Opera-Extension.zip`
- Kept the browser helper privacy-minimal with zero browser permissions and zero host permissions in the published 0.0.6 packages.

### Windows and Linux

- Preserved the two official universal Windows executables with embedded native x64, x86 and ARM64 application payloads and fail-closed trusted Authenticode publication.
- Preserved `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci`; cross-build and structural verification are not represented as native ARM64 runtime execution.
- Preserved the canonical six Linux Debian/Ubuntu/Fedora Installer + Portable release files across the supported public distribution targets.
- Preserved native package-manager/runtime/GUI evidence for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 without inventing native execution claims for cross-built architectures.

### Release engineering and verification

- Advanced root `VERSION` to **0.0.6** and retained the Current channel with `ghostftp-v0.0.6`, `prerelease=false`.
- Published **13 platform artifacts / 16 public files**: two Windows packages, six Linux packages, one Android APK, four browser extension ZIPs, plus `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.
- Added Android and browser stages to the canonical `Publish Ghost FTP` workflow and kept macOS outside public publication until real Developer ID signing and Apple notarization succeed.
- Expanded `BUILD-METADATA.txt`, `SHA256.txt`, exact remote asset readback, digest readback and latest-only retention to the complete 16-file public allow-list.
- The verified GHCR object remains a distribution bundle rather than a supported runtime container.

### Security and privacy

- Production signing material remains outside the repository. The publication workflow never generates a replacement long-lived Windows or Android publisher identity.
- Preserved strict FTPS certificate/hostname verification, strict desktop SFTP host-key trust/pinning, protected saved-secret lifetime, rooted local path protections, staged transfer commit rules and no silent secure-to-plain downgrade.
- Preserved the no-telemetry, no-analytics, no-advertising, no-fingerprinting, no-automatic-crash-upload, no-hidden-backend and no-mandatory-account contract.

## 0.0.5 - 2026-09-12

- Hardened Windows profile/file-mutation/Remote Edit/transfer cancellation lifecycle ownership.
- Added native ARM64 payloads to the same two public universal Windows Setup/Portable files while retaining the explicit `not-native-ci` ARM64 runtime-evidence boundary.
- Hardened Android connection lifecycle and authentication/server-reply privacy.
- Added browser-helper source with no desktop handoff and kept it outside the then-current Windows/Linux release allow-list.
- Made official Windows publication signed-only and preserved exact-head release verification.

## 0.0.4 - 2026-09-11

- Completed Linux appearance, sorting, profile credential-save, queue-priority and navigation/bookmark parity work.
- Hardened Android parser bounds, FTPS/SAF transfer safety and authentic emulator evidence.
- Added four-way queued transfer ordering and conservative directory comparison/synchronized navigation.
- Kept the then-current Windows/Linux release shape at 14 platform artifacts / 17 public files.

## 0.0.3 - 2026-09-10

- Added directional bandwidth ceilings and aggregate scheduling enforcement.
- Reduced public Windows downloads to two universal Setup/Portable packages.
- Promoted distro-specific Linux Debian/Ubuntu/Fedora/Portable packages to the canonical release set.
- Established the 14 platform artifacts / 17 public files Windows/Linux release contract.

## 0.0.2 - 2026-09-10

- Hardened canonical release-branch dispatch and retention sequencing.
- Added local/server filtering, bounded recursive search and conservative directory comparison.
- Expanded authentic repository-local product media and documentation verification.
