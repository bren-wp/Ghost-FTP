> **Current development (10 November 2026): Ghost FTP 0.99.41 Preview.** Windows/Linux open directly in Files without an artificial splash; native QA validates seven real workspaces. Scheduled transfer checks also respond to window wake and connectivity restoration. Previous published edition: v0.99.40 (release verification required). Signing, live providers and 1:1 reference acceptance remain separate.

> **Current development (10 November 2026): Ghost FTP 0.99.40 Preview.** Repair late GitHub prerelease metadata drift with verified checksums and post-publication regression gates. Public predecessor: v0.99.39. Live-provider, production signing and full pixel-parity acceptance remain separate.

> **Current development (10 November 2026): Ghost FTP 0.99.39 Preview.** One canonical release asset inventory, exact SHA-256 verification and reliable Preview post-release checks. Public predecessor: v0.99.38. Production signing and 1:1 visual acceptance remain separate.

> **Current development (10 November 2026): Ghost FTP 0.99.38 Preview.** Windows/Linux transfer-history reliability and release safety improvements. Public predecessor: v0.99.37. Production signing and 1:1 visual acceptance remain separate.

> **Current development (9 November 2026): Ghost FTP 0.99.37 Preview.** Windows/Linux DST-safe scheduled transfers, resumable SHA-256-verified release uploads and stricter release gates. Previous published release: v0.99.36. Signed-production status and 1:1 visual parity are not claimed.

# Ghost FTP desktop roadmap — 0.99.36 Preview

**Current development line:** 0.99.41. **Previous canonical release:** v0.99.40. The supported product is a graphical file-transfer client for Windows, Linux and macOS; historical editions remain immutable.

## Current work

- **Security and privacy:** verify FTP/FTPS/SFTP authentication, SSH known-host identity, TLS hostname/CA validation, temporary credentials, local keyring/Keychain and backup handling
- **Integrity:** per-file upload and download checks, interrupted transfer state, durable staging, overwrite/rename/skip semantics, same-size mutation, multi-gigabyte boundaries and failure recovery
- **Server compatibility:** test with actual FTP/explicit FTPS/SFTP servers; audit WebDAV, cloud, object storage and account connectors using real permitted endpoints
- **Windows:** native WebView2 startup, screen navigation, installer/portable use, upgrade/rollback, OS credential vault and file association checks
- **Linux:** GTK/WebKit startup, AppImage/DEB/RPM package behavior, Secret Service, GNOME/KDE/Wayland tests and native paths
- **macOS:** SwiftUI client FTP correctness, Keychain, signed HTTPS/provider flows and DMG/ZIP validation; FTPS/SFTP remain unavailable until implemented securely
- **Visual acceptance:** native 1290×852 capture for 30 Windows and 29 Linux references, plus actual interaction/keyboard/a11y tests at multiple display scales

## Production gates

A Preview source change is not a stable release or independent penetration test. Require exact commit SHA from quality, protocol E2E, native desktop build, Windows hardening and both macOS verification workflows. Then test installation and authenticated transfer operations on physical devices. Preserve the signed updater's fail-closed policy and immutable release lineage.

No mock accounts, invented server responses, fabricated screenshots or unsupported claims of pixel-perfect parity.
