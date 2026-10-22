# macOS shared desktop parity — acceptance and migration boundaries

## Architecture

The two native architecture-specific Ghost FTP Desktop.app macOS candidates under `ghostftp-desktop/` uses the **same React/TypeScript UI and Rust/Tauri protocol engines** as Windows and Linux. FTP, explicit FTPS, SFTP, host-key verification, sites, remote browsing, transfers, sync, search, editor and terminal are shared code, not separate mock screens.

The independent legacy `macos/` SwiftUI application remains a distinct FTP-only Preview until migration is proven safe. **Do not** rename its bundle or replace its release bytes with the new candidate. The shared app uses bundle ID `com.ghostftp.desktop`; the legacy app uses `com.brendigo.ghostftp.macos`. Their keychain/service identifiers and data locations must not be silently conflated.

## Automated build gates

1. `ghostftp-macos-shared-desktop.yml`: macOS-15 ARM64 plus macOS-15-intel x86_64 runners, central version and `Cargo.lock` exact graph, production frontend, native icon generation, Rust security tests, native `.app` compilation, ad-hoc signature and bundle identity, version validation, ZIP SHA-256 and extracted executable byte identity.
2. `ghostftp-macos.yml`: legacy SwiftUI XCTest and live FTP server integration, package extraction and signature.
3. `ghostftp-protocol-e2e.yml`: real FTP/explicit FTPS/SFTP server roundtrips and Windows filesystem regressions on the shared Rust engine.
4. Existing Windows, Linux, Android, quality, security and dead-code CI gates on the **same exact source commit**.

The build must fail closed when an Apple Keychain backend is missing, `Cargo.lock` drifts, a secure-protocol handshake is disabled or downgraded, a binary is absent, signatures fail, archives are corrupt, or the shared candidate is absent from verified release assets.

## Interactive acceptance still required on physical macOS

- Fresh install and launch on macOS 13+; Finder/Dock icon, keyboard shortcuts, native dialogs, local file permissions, dragging, resizing and minimum viewport.
- Saved and ephemeral FTP, explicit FTPS (valid trusted certificate; revoked, expired, mismatched and untrusted rejection), and SFTP (unknown/changed/revoked SSH host key, key/password auth) with **no plaintext fallback**.
- Remote listing, rename (including the atomic macOS `RENAME_EXCL` no-overwrite guarantee), move, delete, upload, download, retry, cancel, resume, remote-edit conflict and synchronization; compare SHA-256 on binary, Unicode, sparse, >4 GiB and interrupted transfers against original files.
- Apple Keychain create, edit, delete, locked-vault refusal, no plaintext persistence or sensitive logging. Test restarts, app updates and multiple accounts.
- Real installer upgrade of the **same shared bundle ID** must preserve profiles, encrypted credentials, SQLite WAL and transfer history; legacy SwiftUI profile import requires explicit opt-in and independently tested migration, not silent merging.
- Performance, battery/CPU idle activity, VoiceOver, reduced motion, high-DPI, dark/light themes, macOS 13/14/15/16, Intel and Apple Silicon hardware.
- Developer ID signing, hardened runtime, notarization, Gatekeeper and offline installation require separate production evidence.

## Claim boundaries

A successful CI run proves only the tested build/protocol and package properties. It does **not** prove target-device installation, App Store acceptance, notarization, no bugs, native UI pixel parity, complete Windows feature parity or preservation of unrelated legacy SwiftUI user data.

Never delete existing published releases, immutable tags, passwords, profiles, migrations or test evidence to make CI pass.

## Native macOS window and signed distribution path (0.99.5)

The shared Rust/Tauri macOS window uses an AppKit-native decorated overlay titlebar, preserving the standard red/yellow/green window controls; React reserves the traffic-light area and avoids duplicate custom buttons. Windows/Linux retain existing custom window controls. Automated macOS build and code review do not prove target-device window interaction.

The separate manually triggered `.github/workflows/ghostftp-macos-developer-id.yml` signs with a real Developer ID Application identity, requests Apple notarization with `notarytool`, staples the accepted ticket, validates Gatekeeper and publishes **candidate artifacts only** to GitHub Actions. The owner must configure secrets and execute the physical-device matrix in `docs/qa/MACOS_PRODUCTION_RELEASE.md` before any stable release may be claimed. Existing ad-hoc Preview packages do not become certified retroactively.
