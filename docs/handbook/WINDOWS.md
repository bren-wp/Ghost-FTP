# Ghost FTP for Windows — complete source feature and acceptance guide

**Candidate:** 0.93.0 · **Platform:** Windows 10/11 x64 · **Stack:** React, TypeScript, Tauri 2, Rust, system WebView · **Release basis:** verified v0.92.2. This document describes available source implementations, not a blanket guarantee that every path works on every Windows installation. Consult the exact commit's six CI gates, real installer smoke artifacts and manual QA.

## 1. What the application is

Ghost FTP for Windows is the full desktop client. It has a persistent single-window shell with its own title bar, a sidebar, local and remote file browsing, saved sites, transfer management, configuration and productivity tools. Desktop functionality is not automatically present in the Android or SwiftUI macOS versions.

| Area | Source capability and user behavior | Evidence / constraints |
| --- | --- | --- |
| Navigation and shell | Single main Tauri window; Files, Sites, Transfers and Settings surfaces; contextual About/Help | App.tsx, ReferenceSiteSidebar.tsx, TitleBar.tsx; verify real DPI and window decorations |
| Connections | FTP, explicit FTPS, SFTP; temporary Quick Connect; reconnect and session management | SiteManagerDialog, QuickConnectionDialog, Rust session adapters |
| Cloud connections | S3 and compatible R2/B2/MinIO/Wasabi etc, Azure Blob, GCS, WebDAV/Nextcloud, Dropbox, OneDrive, Google Drive and Box | Rust session and RemoteFs adapters; cloud provider credentials/consent must be correctly provisioned |
| Other connectors | HTTP and source-defined Shopify, HubSpot, Dynamics and Ghost FTP Agent adapters | Availability and scope vary; do not assume all remote file operations apply to each SaaS backend |
| Site Manager | Saved connections, favorites, folders, tags, bookmarks, recents, import/export, duplicate discovery, connection test | Changes to unsaved forms require confirmation; security-sensitive identity collision handling |
| Authentication | Password, SSH key path, agent, key references; TLS verification, SSH host-key decision | Credentials have protocol-specific semantics; never silently trust unknown/mismatched host keys |
| Browser | Local/remote dual pane, folder browsing, list/details/grid, search, sorting, hidden files, navigation | File operations depend on RemoteFs capabilities and server permissions |
| Actions | Upload/download, folders, create, rename, delete, duplicate, file properties, appropriate chmod/owner details and SHA-256 | Backend support differs; dangerous actions need explicit confirmation |
| Previews | Remote image previews and edit-in-place workflow, configurable editor | Optional features, bounded previews; remote file content may enter temporary local storage |
| Transfer center | Queue, active/completed/failed lists, concurrency, speed/progress, pause/resume/cancel/retry and conflict policies | Pause/resume availability depends on protocol; a queued item is not a completed transfer |
| Synchronization | Local/remote sync planning, additive/mirror strategies, scan/diff/duplicate/search/disk usage | Mirror can delete destination objects; preview dry-run and confirmation required |
| Terminal | SSH-related terminal, command palette, snippets, variables, keyboard shortcuts | Shell access requires authenticated authorized SSH/Agent path |
| Ghost FTP Agent | Discovery, pairing, remote control and guarded command approvals | Must not bypass trust, pairing and explicit destructive-action authorization |
| Preferences | Theme/light/dark, accent, density, language, hidden files, thumbnails, transfer limits, retries, notifications, editor, updater | Settings reset/cancel should restore live engine state correctly |
| Language selector | English, Croatian, German, French, Spanish, Italian, Portuguese, Dutch, Polish, Slovenian, Serbian, Bosnian, Macedonian, Albanian | Key parity check exists; full human linguistic review not established |
| Updates | Version check, signed installer update path, restart flow | Signed updater needs correctly published private-key-signed release metadata |
| Packages | Setup NSIS, MSI, Windows portable executable | CI exercises silent install, GUI launch and uninstall; check actual artifacts and hashes |

## 2. Connection and security details

### FTP / FTPS / SFTP

The protocol selector determines TLS/SSH behavior, default port, supported commands and data path. Plain FTP is unencrypted and must display its privacy warning. FTPS must verify certificate chain and hostname; SFTP must verify a pinned host key before sensitive authentication. Never add a "skip verification" production fallback. Different server operations have different support: SSH/SFTP allows permissions and server-side commands where authorized, while object stores do not expose POSIX chmod or directory semantics.

### Object stores, Google Drive and SaaS

S3 uses SigV4-compatible credentials and region, bucket and optional provider endpoint; Azure uses account/container keys; GCS supports the configured service-account flows. Dropbox, OneDrive, Google Drive and Box have separate API-specific adapters. **Google Drive is not Google Cloud Storage.** Do not reuse S3 credentials for Google Drive or store OAuth refresh tokens in ordinary profile JSON. The new 0.93.0 guard rejects custom S3/Azure URLs if scheme is not HTTPS or URL contains userinfo, query or fragment. Verify whether each provider adapter supports listings, uploads, downloads, rename, delete, paging, token refresh and error recovery using live disposable test accounts before claiming production parity.

### Credentials and persisted state

Treat connection profiles as metadata and secret references, not archives for plaintext credentials. Use the OS credential vault/keyring where supported and validate failure paths: locked vault, missing entry, permission change, profile import/export/restore and deletion. Passwords, session cookies, signed URLs, cloud tokens and private keys must not appear in diagnostics, telemetry, screenshots or CLI output. Use user-authorized Windows filesystem locations only, with restrictive access and no broad-permission cache.

## 3. File operations: click-by-click acceptance matrix

1. New connection: open Site Manager, choose protocol, fill valid endpoint/auth, test without saving, save, close and reopen; ensure secrets remain protected.
2. Temporary connection: Quick Connect, navigate and disconnect; verify no unexpected permanent Site entry or credential copy.
3. Navigation: switch local/remote roots, enter a directory, go up, refresh, sort, toggle hidden items, and handle empty/permission-denied locations.
4. Upload: choose an ordinary file and nested folder, verify contents/checksum, handle filename Unicode, collisions and interrupted network, and confirm no premature success UI.
5. Download: choose destination, verify bytes/partial cleanup and avoid overwriting a previous good file after a failed operation.
6. Rename/move/copy/delete: test protected roots, nonempty folders, cross-provider limitations, preflight warnings and undo/recovery where actually implemented.
7. Resume/pause/cancel: test FTP/FTPS/SFTP interruption against live services, reconnect, verify exact remote prefix, do not append unverified bytes.
8. Cloud: test S3 and compatible provider endpoint/region, listing, objects with UTF-8 names, pagination, expiry, error redaction, 3xx redirects and 403/404/429/5xx.
9. Google Drive/OneDrive/Dropbox/Box: test access permission scope, expired token, revocation, pagination, upload conflict, deletion and account switching with separate accounts.
10. Sync: preview additive and mirror plans, inject concurrent modifications and conflicts, confirm no unexpected deletion; verify resume and rollback semantics.
11. Window: launch installed and portable builds, resize to minimum supported size, DPI 100–200%, taskbar, Alt+Tab, keyboard focus and high contrast; compare approved visual references.
12. Installer: clean NSIS/MSI install, relaunch, portable smoke, uninstall, check leftover running processes/files; never mark a release green solely because compilation succeeds.

## 4. Privacy and hardening backlog

| Priority | Work item | Acceptance criterion |
| --- | --- | --- |
| P0 | Cloud credential lifecycle | All secrets in OS-protected storage; import/export excludes tokens; revoke/clear handles errors; no secret in logs |
| P0 | S3/HTTP SSRF and DNS rebinding | Revalidate resolved destinations at connection time, reject metadata/private addresses for untrusted links, prevent cross-host redirects |
| P0 | OAuth desktop lifecycle | PKCE/system browser with explicit consent; scoped permissions; encrypted vault refresh storage; token rotation and revoke/account removal |
| P0 | Upload/download integrity | Stream to temporary target; verify sizes/checksums/ETag where meaningful; atomic promote or preserve old data on failure |
| P0 | Uninstall/update safety | Signed manifests, SHA-256 assets, reliable install/smoke/uninstall on Windows 10 and 11 |
| P1 | Credential exposure audit | Static and runtime inspection of diagnostics, browser DevTools, crash reports, clipboard and notification bodies |
| P1 | Transfer pressure limits | Bounded memory, queued concurrency, bandwidth limits, exponential jitter, 429 Retry-After and explicit cancel |
| P1 | Cloud least privilege | Optional per-bucket/drive file scopes and account-level disconnect/revoke |
| P1 | Data retention | Clear temp files, recent history policy, per-user vault isolation and documented deletion |
| P1 | Accessibility/i18n | Keyboard-only 100% primary flows, UI automation, screen reader, RTL assessment and complete translations |
| P2 | Multi-account cloud UX | Clear provider/account identity, per-account tabs, conflict controls, capability-driven menu hiding |
| P2 | Offline behavior | Non-secret cache opt-in, integrity-aware cache invalidation, secure purge, no stale misleading listings |

## 5. Production release checklist

Must pass Rust fmt/check/clippy/test; TypeScript check/build/lint/i18n/interaction/dead-code audits; actual FTP/FTPS/SFTP E2E; installer NSIS/MSI and portable GUI launch smoke; endpoint security tests; real disposable cloud-service E2E; public release asset SHA-256 digest verification; and approved device screenshots. These requirements remain **pending** until the exact head commit's jobs and artifacts prove completion.

## 6. Relevant source map

- ghostftp-desktop/src/components — interactive shell, Site Manager, transfer queue, sync, search, terminal, settings, errors.
- ghostftp-desktop/src-tauri/src/session — protocol implementations (including cloud provider adapters).
- ghostftp-desktop/src-tauri/src/remotefs — backend-independent file operation interfaces and capability controls.
- ghostftp-desktop/src-tauri/src/bridge.rs — native command routing, authorization and agent commands.
- ghostftp-desktop/src/lib/types.ts — connection protocols, cloud provider presets and transfer domain models.
- .github/workflows/ghostftp-build.yml — Windows install, launch, uninstall and bundle QA.
- docs/product/CLOUD_STORAGE.md — cloud security boundaries and cross-platform parity.
