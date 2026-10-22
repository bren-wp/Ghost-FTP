# Ghost FTP — macOS production release acceptance

**Production is not verified by a passing CI build.** This checklist separates the native shared Tauri macOS app from the legacy SwiftUI macOS Preview. They have distinct bundle IDs and must not silently overwrite each other's profiles, SSH keys, credentials, SQLite data, settings or transfers.

## Required Apple secrets

The manually triggered `.github/workflows/ghostftp-macos-developer-id.yml` workflow needs a paid Apple Developer membership, plus GitHub Secrets `GHOSTFTP_MACOS_DEVELOPER_ID_IDENTITY` (Developer ID Application identity), `GHOSTFTP_MACOS_TEAM_ID`, `GHOSTFTP_MACOS_DEVELOPER_ID_P12_BASE64`, `GHOSTFTP_MACOS_DEVELOPER_ID_P12_PASSWORD`, `GHOSTFTP_MACOS_NOTARY_P8_BASE64`, `GHOSTFTP_MACOS_NOTARY_KEY_ID` and `GHOSTFTP_MACOS_NOTARY_ISSUER_ID`. Configure required human approval for the `macos-production-signing` GitHub environment. Do not commit these secrets.

The workflow checks the immutable `vX.Y.Z` source tag, compiles and runs Rust/frontend tests on **both Apple Silicon and Intel**, imports credentials into an ephemeral Keychain, requires the **Developer ID Application** certificate and hardened runtime, submits to Apple's notary service, staples the accepted ticket, verifies Gatekeeper and produces separate signed candidate ZIPs and SHA-256/evidence files. Missing credentials or Apple rejection must fail the job; ad-hoc signing is never a substitute. The workflow does **not** create or modify GitHub Releases.

## Physical-device acceptance — compulsory before production publication

For **one real Apple Silicon Mac and one real Intel Mac**, record tester, OS, source SHA, archive SHA-256, Team ID, notarization ID, date and results. Confirm browser-downloaded/quarantined app opens under Gatekeeper, Finder/Dock, native window traffic lights, Command shortcuts and Control shell keystrokes, macOS dialogs/permissions, HiDPI, resizing, VoiceOver and dark/light.

Confirm real FTP, trusted explicit FTPS, and verified SFTP hosts. Prove invalid TLS certificates, changed SSH host keys and key failures are rejected without insecure downgrade. Validate Keychain create/read/delete/lock/restart/upgrade; no plaintext secret fallback. Check file listing, Unicode, symlinks, concurrent transfers, rename collisions, uploads/downloads, cancel/retry/resume, directory sync, remote editing and notifications. Compare original/content SHA-256 on large binary files **over 4 GiB**, including interrupted transfers, and verify recovery without data loss. Test 10,000+ file listings, sleep/wake, battery/CPU idle, and realistic installation upgrades.

Back up and preserve existing `com.ghostftp.desktop` user state on upgrade. The older SwiftUI `com.brendigo.ghostftp.macos` is **not** automatically migrated or deleted. Do not label a notarized candidate "production" until these real-device tests, rollback and signed-updater acceptance are documented. Preserve every public release and tag.

Official documentation: https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution and https://v2.tauri.app/distribute/sign/macos/.
