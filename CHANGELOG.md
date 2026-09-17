# Changelog

## 0.0.7 - 2026-09-17

### Cumulative polish, cleanup and product hardening

- Includes the complete maintained codebase accumulated after 0.0.6, including repository cleanup, removal of retired website/Web FTP surfaces, documentation cleanup and release-contract corrections.
- Polished maintained Windows, Linux, Android and browser-helper product surfaces while preserving the no-telemetry, no-advertising, no-mandatory-account and direct-to-user-server privacy model.
- Hardened exact-head release audits, platform contracts, documentation contracts, UI evidence checks and regression coverage so release identity follows root `VERSION` consistently.

### Windows and browser-to-desktop handoff

- Added sanitized `ghostftp:` protocol registration for installed Windows builds with transactional registry snapshot/rollback behavior.
- Added browser helper **Open in Ghost FTP** handoff using sanitized FTP/FTPS/SFTP target data only; passwords, private-key passphrases, private keys, query data and fragments are excluded and the desktop app never auto-connects.
- Hardened startup target forwarding, single-instance handoff, Site Manager teardown guards, retry behavior and post-close lifecycle handling.
- Improved integrated uninstall and upgrade compatibility while retaining universal x64, x86 and ARM64 payloads in the two public Windows executables.

### Android and cross-platform maintenance

- Refined Android theme behavior, navigation, profile storage, remote-edit handling and lifecycle safety while keeping SFTP hidden until strict host-key verification is maintained.
- Preserved strict FTPS certificate/hostname verification, desktop SFTP host-key trust/pinning, staged transfer safety and fail-closed credential handling.
- Expanded browser helper tests and product-surface contracts and kept the four official Chrome, Edge, Firefox and Opera packages at zero browser permissions and zero host permissions.

### Release engineering

- Advances root `VERSION` to **0.0.7** and publishes on the Current channel with `ghostftp-v0.0.7`, `prerelease=false`.
- Canonical public release remains **13 platform artifacts / 16 public files**: 2 Windows, 6 Linux, 1 Android, 4 browser helper packages and 3 metadata/verification files.
- Aligned active documentation, version audits and release verification contracts with 0.0.7 while preserving historical 0.0.6 runtime evidence and release history.
- macOS remains maintained source and CI coverage only until real Developer ID Application signing and Apple notarization are verified.

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

- Added deterministic public release ZIPs for Chrome, Microsoft Edge and Firefox:
  - `Ghost-FTP-0.0.6-Chrome-Extension.zip`
  - `Ghost-FTP-0.0.6-Edge-Extension.zip`
  - `Ghost-FTP-0.0.6-Firefox-Extension.zip`
- Kept the browser helper privacy-minimal: no broad host, tab, history, storage, scripting or network permissions are added by the release contract.
- The browser helper remains a local parser/copy companion. There is still no supported browser-to-desktop URI or native-messaging launch/handoff.

### Windows and Linux

- Preserved the two official universal Windows executables with embedded native x64, x86 and ARM64 application payloads and fail-closed trusted Authenticode publication.
- Preserved `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci`; cross-build and structural verification are not represented as native ARM64 runtime execution.
- Preserved the canonical twelve Linux Debian/Ubuntu/Fedora/Portable packages and metadata/extraction/binary-parity gates across amd64/arm64/i386 or equivalent RPM architectures.
- Preserved native package-manager/runtime/GUI evidence for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 without inventing native execution claims for cross-built architectures.

### Release engineering and verification

- Advanced root `VERSION` to **0.0.6** and retained the Current channel with `ghostftp-v0.0.6`, `prerelease=false`.
- Expanded the canonical public release shape from 14 platform artifacts / 17 public files to **18 platform artifacts / 21 public files**.
- Added Android and browser stages to the canonical `Publish Ghost FTP` workflow and kept macOS outside public publication until real Developer ID signing and Apple notarization succeed.
- Expanded `BUILD-METADATA.txt`, `SHA256.txt`, exact remote asset readback, digest readback and latest-only retention to the complete 21-file allow-list.
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
