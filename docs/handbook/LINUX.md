# Ghost FTP for Linux — complete source feature and acceptance guide

**Candidate:** 0.93.0 · **Platform:** Linux x86_64; tested packaging targets AppImage, DEB and RPM · **Stack:** React/TypeScript, Tauri 2, Rust, system WebKitGTK. This shares much of the desktop feature source with Windows but has different OS credential storage, package lifecycle, file permissions and runtime dependencies. Passing a Windows test does not certify Linux.

## 1. Surface and capability inventory

| Area | Implemented in shared desktop source | Linux-specific behavior / scope |
| --- | --- | --- |
| Shell | Single main application window, branded sidebar, title bar and settings | GTK/WebKitGTK window decoration and desktop shell integration require compositor-by-compositor QA |
| Connections | FTP, explicit FTPS, SFTP, Quick Connect, saved Sites and Test Connection | Native network/TLS/SSH path; passwords and host-key verification are mandatory |
| Object storage | S3/R2/B2/MinIO/Wasabi-compatible storage, Azure Blob, GCS | Same Rust adapters as Windows; only HTTPS custom endpoint allowed in candidate |
| Cloud drives | Google Drive, Dropbox, OneDrive, Box and WebDAV/Nextcloud adapters | Existing protocol adapters; separate token provisioning and OS keyring constraints |
| Saved sites | Favorites, folders, tags, bookmarks, recent profiles, import/export and duplicate advisory | User-specific configuration and file permissions; imported backups must be treated as untrusted |
| Local/remote files | Dual pane, directory navigation, sorting, hidden files, context menu, details/grid | Permission errors and symlink behavior must be handled without surprising deletes |
| Transfers | Upload/download, folder transfers, queue, status, retry, cancel, speed and conflict handling | Resuming FTP needs verified remote prefix; queue must survive expected UI transitions |
| File metadata | Properties, SHA-256, chmod and remote permissions when supported | Symlinks, POSIX modes and owner/group display differ between local and remote FS |
| Sync/analysis | Additive/mirror, diff, duplicate detection, disk scan/treemap, search | External mounts and sparse files need explicit safety test; mirror is destructive |
| Terminal/agent | SSH terminal, agent, command approvals, snippets, shortcuts | Shell conventions and process signals differ from PowerShell/Windows |
| Settings | Themes, density, language, notifications, bandwidth, overwrite rules, update checker | Desktop notifications and keyring facilities differ across distributions |
| Packaging | AppImage, .deb, .rpm, standalone binary | Bundled runtime needs WebKitGTK compatibility; AppImage FUSE fallback is a QA concern |
| Help/privacy | Local-first provider connections and credential redaction | No required cloud proxy or analytics; temporary files must respect filesystem permissions |

## 2. Linux-specific operational design

### Installation and removal

AppImage is a single packaged executable; DEB uses the distribution's package database, and RPM provides another package format. Each must launch the GUI, locate icons and branding, open file dialogs, receive desktop notifications where supported, and uninstall without abandoning profile/secret material contrary to the published retention policy. **Do not delete user data during an upgrade**. Document how users explicitly purge their own configuration and credentials if they request total removal.

A production CI build must produce nonempty deb, rpm and AppImage packages, test metadata and payloads, run the bundled application under a real windowing session (Xvfb tests are only a baseline), verify installed binaries and startup, and preserve logs for errors. WebKitGTK, GLib, crypto/TLS and libayatana system packages may affect runtime compatibility.

### Keyring and filesystem privacy

The current desktop manifest enables Linux-native keyring integration, but actual availability can depend on desktop session, keyutils and sandbox. Handle locked/unavailable vaults explicitly and fail safely rather than falling back to unprotected plaintext. Configuration backups must not export real credentials without separate user encryption and warning. Sensitive temporary files should be created with owner-only permissions, written atomically and cleaned on partial failure.

### Network and provider models

- FTP transmits credentials unencrypted: show warning and recommend FTPS/SFTP.
- FTPS uses certificate/hostname validation; no trust-all mode.
- SFTP uses SSH identity checking; host-key mismatches block sensitive operations.
- S3-compatible endpoint options include bucket, region, access key and optional custom endpoint. Candidate 0.93.0 only permits HTTPS custom S3/Azure endpoints with no embedded query/userinfo.
- Google Drive, OneDrive, Dropbox and Box need provider-specific authorization, scope, paging, token refresh and revocation. GCS credentials are not Google Drive credentials.
- WebDAV/Nextcloud is HTTPS-based and uses the configured DAV root path. Do not silently follow cross-host redirects carrying credentials.

## 3. Per-operation manual test scenarios

| Workflow | Required real-world check | Expected safe behavior |
| --- | --- | --- |
| Site creation | Missing hostname, bad port, invalid key path, wrong password | Informative redacted errors; no leaked token |
| Saved-site import | Malformed JSON, duplicate IDs, duplicate endpoint, corrupt file | Reject/repair atomically; do not reuse wrong secret |
| Upload/download | 0-byte, >2 GB where supported, Unicode file name, space and slash edge cases | Byte-accurate content and preserved original on error |
| Remote delete | File, empty folder, nonempty folder and root | Intent confirmation; refuse accidental root removal |
| Resume/retry | Terminate mid-transfer and reconnect | No blind append without verified committed prefix |
| Cloud listing | S3/GCS/Azure bucket with pagination and limited permissions | Partial errors surfaced; no silent data skip |
| OAuth drives | Expired/revoked credential, account switch, 429 response | Graceful reauth with least privilege; no fallback logging of tokens |
| Sync | Two-way changes, mirror deletion, Unicode, symlinks, race updates | Dry-run preview and explicit destructive consent |
| Local file safety | Mount disappearing mid-operation, symlinks and read-only volumes | Confined operations with no path traversal or accidental overwrite |
| Network | TLS expiry, DNS changes, redirect, proxy, offline, rate limiting | Fail closed, retry boundedly, never leak Authorization cross-origin |
| GUI | Wayland/X11 desktops, KDE/GNOME, 100–200% scaling, keyboard-only | Usable layout, screen-reader labels and correct window behavior |
| Packages | AppImage/deb/rpm launch+remove, missing library case | No blank screen or orphan process; actionable error |

## 4. Security/privacy improvements

**P0:** refuse TLS bypass; prevent SSRF to local/cloud metadata endpoints; enforce redirect stripping or disable redirects on authenticated requests; protect OAuth tokens in an OS vault, with refresh rotation and revocation; use bounded network timeouts and read limits; preserve old file on failed replacements. Prioritize live signed-provider testing rather than mere object-store constructor tests.

**P1:** secure temporary files (0700 directories, 0600 secrets), symlink/path traversal checks, zip-slip guards for backup restores, history and cache cleanup, deterministic error redaction, logging opt-in, secret-free crash traces and directory-confinement checks. Evaluate cloud provider permissions before enabling destructive actions.

**P2:** per-distro integration test matrix, live status/health diagnostics, low-memory transfer scheduling, sparse-file checks, proxy policy, improved cancellation and localization review.

## 5. Cloud and sync correctness constraints

Object storage offers logical prefixes, not Unix directories. Rename can be copy-then-delete and may expose a failure window; choose safe transactional UX and never claim atomicity where a provider cannot guarantee it. Sync must compare identifiers, modification times, checksums and source snapshots without assuming an ETag is always a content hash. Cloud API 429 and 5xx should use bounded backoff. Credentials must be redacted from request paths and access logs; signed URLs must never appear in progress messages.

## 6. Release gates and artifacts

Linux PR and merged-main must pass source format, Rust check/tests/clippy, shared TypeScript checks, protocol E2E and Linux packaging smoke. An AppImage extracted smoke does not substitute for launching on real popular distributions. Official releases need immutable version/tag, archive checksums and provenance. Full parity with Windows UI and every cloud provider is subject to real-device tests.

## 7. Relevant source

- ghostftp-desktop/src/components and ghostftp-desktop/src/lib for UI and protocol options.
- ghostftp-desktop/src-tauri/src/session and remotefs for FTP/SSH/cloud operations.
- ghostftp-desktop/src-tauri/src/profiles for saved site model and recovery.
- .github/workflows/ghostftp-build.yml for DEB/RPM/AppImage package and launch checks.
- docs/qa and docs/product/CLOUD_STORAGE.md for acceptance scope.
