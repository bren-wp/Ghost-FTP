# Ghost FTP — desktop feature matrix (v0.99.36 Preview)

**Legend:** SRC = implemented in reachable source, LIMITED = restricted or incomplete, TODO = unavailable, VERIFY = requires native/live acceptance. No statement here implies successful tests against every remote provider.

| Capability | Windows | Linux | macOS SwiftUI | Notes |
| --- | --- | --- | --- | --- |
| Files, Sites, Transfers, Settings | SRC | SRC | SRC (simplified) | Separate native macOS UI |
| FTP login, listing and file transfers | SRC | SRC | SRC | Plain FTP is unencrypted |
| Explicit FTPS | SRC | SRC | TODO | TLS certificate and channel identity mandatory |
| SFTP | SRC | SRC | TODO | SSH host-key pinning and prompt policy mandatory |
| Quick Connect and saved sites | SRC | SRC | LIMITED | Never persist temporary password silently |
| Folder operations, rename, delete, permissions | SRC backend-dependent | SRC backend-dependent | LIMITED | Confirm exact path and server permissions |
| Transfer queue/pause/retry | SRC | SRC | LIMITED | macOS history is not desktop-equivalent durable resume |
| Sync, directory diff, duplicates, disk usage | SRC | SRC | TODO | Requires measured runtime UX and failure tests |
| SSH terminal and Agent | SRC | SRC | TODO | Remote command execution must be permission-gated |
| Cloud object storage and WebDAV | SRC adapter-specific | SRC adapter-specific | LIMITED signed-object operations | Real tenant/credentialed E2E pending |
| Drive, Dropbox, OneDrive and Box accounts | SRC adapter-specific | SRC adapter-specific | TODO full account browsing | Token refresh/revoke/quota tests pending |
| Local credential protection | SRC OS vault | SRC Secret Service | SRC Keychain | Must test actual availability and migration |
| Automatic signed update | SRC when signed artifact exists | SRC when signed artifact exists | TODO full production update | Reject unsigned/downgrade |
| Native packages | EXE portable/NSIS/MSI | Binary/AppImage/DEB/RPM | ZIP/DMG Preview | Signatures and install behavior vary |
| 1:1 screenshot acceptance | VERIFY 30 views | VERIFY 29 views | No approved visual reference | Concept renders are not real screenshots |

## Source ownership and acceptance

Windows/Linux: `ghostftp-desktop/src/` (UI), `ghostftp-desktop/src-tauri/` (Rust native transfer and secrets), `tools/` (Go compatibility tooling). macOS: `macos/Sources/GhostFTPMacApp/`. CI and release governance: `.github/`, `updates/`, `version.json`.

To move a capability to accepted status: prove working code and UI path, malformed input and credential-safety tests, localhost or real provider protocol E2E, native install/launch, saved-data migration and independent reviewer evidence. Cross-platform sharing of branding never proves identical behavior.
