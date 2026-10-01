# Ghost FTP Roadmap

This roadmap separates the current **0.20.10 development/release** state from future work. Planned, recommended and long-term items are not claims of implemented functionality. Previous canonical release: **0.20.9**.

## Implemented

- Native React + TypeScript + Tauri + Rust desktop application for Windows and Linux.
- Native Kotlin Android application.
- FTP, explicit FTPS and SFTP connection paths.
- SFTP password/private-key authentication and host-key verification.
- Ephemeral Quick Connect by default with explicitly saved profiles.
- Sites management with favorites, recent metadata, bookmarks, tags, folders, import/export and connection testing.
- Dual-pane local/remote file management, queue/history, retry/cancel/pause/resume controls and bandwidth settings.
- SHA-256, file properties and supported permission/chmod workflows.
- OS credential/keychain-backed secret separation where supported.
- Custom frameless Ghost FTP titlebar and adaptive desktop layouts.
- Advertised interface locales with automated key-parity CI.
- Native Windows NSIS and Linux executable/AppImage/DEB/RPM build targets.
- Installable Android APK build path with package/signature verification in CI.
- Quality, protocol E2E, canonical native build, Android and Windows hardening gates.
- Release-image provenance checks for README/documentation images.
- Privacy-first default: no required analytics or telemetry.

## Current hardening focus

- Keep one authoritative Windows/Linux production build path with native-window QA evidence.
- Keep documentation imagery pinned to the latest published release.
- Keep Windows helper/background processes free of unintended console-window flashes.
- Keep dependency graphs locked and version/Cargo metadata synchronized atomically.
- Maintain real FTP/explicit-FTPS/SFTP E2E coverage for release-relevant source.
- Maintain Android lint/build/package verification on every pull request to `main`.

## Before stable 1.0

- Windows 10/11 clean install, upgrade, reinstall and uninstall lifecycle acceptance.
- Windows installed + portable visual acceptance including titlebar and snap behavior.
- Linux package install/update/remove acceptance on target distributions.
- Android install/upgrade/storage/protocol acceptance on target devices.
- Security failure-path review on target systems.
- Production signing policy and verification.
- Final accessibility/keyboard/high-contrast review.
- Broader protocol failure-injection/overwrite/resume/reconnect coverage.
- Reproducible-build/provenance documentation with independently repeatable verification.

## Recommended

- Per-profile reconnect and keep-alive policy.
- Transfer-history export and richer schedule management.
- Per-profile bandwidth limits.
- Verify-after-transfer checksums where both sides support them.
- Batch rename and remote-edit conflict detection.
- Encrypted selected-profile import/export.
- Duplicate-profile detection and richer Sites templates/search.
- Full screen-reader, high-contrast and touch-target audits.
- SBOM, dependency vulnerability scanning, secret scanning and provenance attestations.
- Optional managed Linux repositories when distribution policy requires them.

For the detailed engineering backlog, see [product/ROADMAP.md](product/ROADMAP.md). For authoritative current source/build state, see [product/STATUS_AND_NEXT.md](product/STATUS_AND_NEXT.md) and [build/STATUS.md](build/STATUS.md).
