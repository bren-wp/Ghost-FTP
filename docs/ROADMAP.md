# Ghost FTP Roadmap

This roadmap separates the shipped/current RC20 release-candidate work from future work. Items under Planned, Recommended and Long-term are not claims of implemented functionality.

## Implemented

- Native React + TypeScript + Tauri + Rust desktop application for Windows and Linux.
- FTP, explicit FTPS and SFTP connection paths.
- SFTP password and private-key authentication, including optional key passphrase input.
- Ephemeral Quick Connect by default; saved profiles remain an explicit choice.
- Site Manager with favorites, recent metadata, bookmarks, tags, folders, import/export and connection testing.
- Dual-pane local/remote file management, transfers, queue/history, retry/cancel/pause/resume controls and bandwidth settings.
- SHA-256, file properties and supported permission/chmod workflows.
- OS credential/keychain-backed secret separation where supported.
- Custom frameless Ghost FTP titlebar and adaptive desktop layouts.
- 14 advertised interface locales with automated key-parity CI.
- Native Windows NSIS and Linux executable/AppImage/DEB/RPM build targets.
- Quality, native-build and release GitHub Actions gates.
- Privacy-first default: no required analytics or telemetry.

## Planned

These are release-candidate acceptance items and should not be presented as completed until evidence exists.

- Extend protocol E2E with explicit overwrite/resume/reconnect failure-injection scenarios.
- Maintain explicit FTPS certificate identity and failure-path coverage in every release candidate.
- Maintain SFTP password/private-key/host-key acceptance in every release candidate.
- Windows 10/11 custom-titlebar screenshot acceptance for portable and installed builds.
- Windows clean install, upgrade, reinstall and Apps & Features uninstall acceptance.
- Responsive/pixel acceptance across all documented target sizes.
- Linguistic and clipping review across all 14 locales.
- Production signing decision and target-OS update verification.

## Recommended

- Per-profile reconnect and keep-alive policy.
- Transfer-history export and richer schedule management.
- Per-profile bandwidth limits.
- Verify-after-transfer checksum where both sides support it.
- Batch rename and remote-edit conflict detection.
- Encrypted selected-profile import/export.
- Duplicate-profile detection and richer Site Manager templates/search.
- Full keyboard, screen-reader, high-contrast and Windows snap-layout audits.
- SBOM, dependency vulnerability scanning, secret scanning and reproducible-build documentation.
- Containerized FTP/FTPS/SFTP integration servers for repeatable CI.

## Long-term

- Signed Windows binaries and signed Linux repository metadata where applicable.
- Optional managed APT/RPM repositories.
- Stronger visual-regression automation against approved Ghost FTP references.
- Advanced proxy/bastion workflows driven by user demand.
- Expanded sync/automation policies with clear conflict recovery.
- Release provenance/attestation and reproducible artifact verification.

For the detailed engineering backlog, see [product/ROADMAP.md](product/ROADMAP.md). For the authoritative current release state, see [product/STATUS_AND_NEXT.md](product/STATUS_AND_NEXT.md).
