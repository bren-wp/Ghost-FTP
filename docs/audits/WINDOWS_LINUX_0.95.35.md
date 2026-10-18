# Ghost FTP Windows + Linux — source, security, dead-code and UX audit (v0.95.35)

**Scope:** the existing Windows and Linux desktop product (Tauri 2, WebView2/WebKitGTK, React/TypeScript and Rust), its local filesystem adapter, transfer manager, three native protocols and installer workflows. The source baseline is published [v0.95.34](https://github.com/bren-wp/Ghost-FTP/releases/tag/v0.95.34). This document is an auditable review and targeted code remediation, **not** an assertion that every runtime option has been manually tested.

## Findings and direct remediation

| Priority | Area | Audited finding | v0.95.35 change | Verification |
| --- | --- | --- | --- | --- |
| Critical | Windows/Linux downloads | `Path::exists` and `metadata` follow symlinks and treat dangling links as absent during Skip/Rename decisions | `symlink_metadata` with fail-closed errors | Rust dangling-link and rename collision regressions |
| Critical | Backup integrity | Successful staging promotion could remove an existing same-ID backup without checking its contents | No-replace moves for backups, staging and restoration; no blind backup deletion | Windows/Linux preserved-backup and normal-replacement regressions |
| High | Download target integrity | Replacing a directory, symlink or unusual node is not a normal file overwrite | Regular-file-only promotion and staging verification | Rust occupied-directory and Unix symlink tests |
| High | Local upload | Native file picker followed selected symbolic-link targets; Linux backslash filename could be interpreted as a separator | `symlink_metadata` and native OS filename component with safe-name validation | Rust guards and source review |
| High | Directory upload | Unsupported nodes could be skipped silently; massive trees and lossy names could be incompletely uploaded | Inventory first; refuse special entries, non-UTF-8 and unsafe names; 100k entries / 64 levels / 16 MiB cumulative remote paths | Rust boundary regressions, Rustfmt/Clippy |
| High | Dual-pane UI | Old remote drag payload could outlive a connection switch and be queued against a new session | Validate JSON structure and source-session ID in both panes | TS compile, frontend contract guard |
| Existing controls | Protocol and credentials | FTP, FTPS and SFTP transfer errors, key authentication and server trust | Preserve existing FTPS/SFTP verification and secret redaction | Real loopback FTP/FTPS/SFTP E2E workflow |
| Existing controls | Installer | Windows NSIS/MSI and Linux AppImage/DEB/RPM install, update and UI startup | No installer behavior changed in this increment | Native CI bundle/install/uninstall smoke |
| Existing controls | Responsive UX | Seven real historical Windows screenshots; 75 design-concept screens | Preserve genuine images, no generated mockup substituted | Native screenshot QA; **manual image comparison still pending** |

## Dead-code audit — exact source boundaries

- `npm run check:dead` runs `ghostftp-desktop/scripts/check-source-reachability.mjs`: checks reachability from `src/main.tsx` and `packages/file-ui/src/index.ts`, Rust module graphs in **four production crates**, and referenced CI/Android/update utility scripts. Orphan source files fail CI.
- `npm run check` invokes TypeScript with `strict` and `noUnusedLocals`; `npm run check:ui` includes interaction and UI contracts. Cargo's workspace Rustfmt, tests and Clippy jobs verify Rust compilation and lint consistency.
- This combination is a strong **reachable-file, static type, unused-local and lint audit**, not mathematical proof that every branch, CSS selector, runtime flag or public exported API is used. Safe removal of public API code requires external consumer and compatibility analysis. No mass deletion is warranted merely because a file lacks obvious imports.
- Audit acceptance requires exact v0.95.35 head CI checks; previous-version green checks do not certify this candidate.

## Functional simulation and QA matrix

| Area | Automated harness | Remaining acceptance gap |
| --- | --- | --- |
| FTP/FTPS/SFTP login, listing, transfer and host fingerprint | GitHub Actions real protocol E2E with local FTP/FTPS and OpenSSH SFTP | IIS, ProFTPD, Pure-FTPd, proxy/NAT/IPv6 cross-vendor fixtures |
| Download overwrite, backups, symlinks and rename | Rust tempfile and filesystem regression scenarios | Crash/power-loss fault injection; adversarial concurrent process races |
| Directory upload recursion and malformed inventory | Rust boundary tests and full scanner traversal | 100k+ real-file stress on Windows shares/NAS |
| Transfer queue, pause, retry, cancel and bandwidth | Existing Rust queue and token-bucket tests | Sustained high-volume network loss / multiple distant servers |
| Files/Sites/Transfers/Settings/terminal/sync menus | Desktop TypeScript contracts, Windows hardening, native capture QA | Every click, dialog, focus and high-DPI combination must be manually signed off |
| Windows installer and Linux packages | Native CI build + install/uninstall/lifecycle smoke | Real heterogeneous production machines, signed-store distributions |
| Cross-pane drag/drop | Typed UI contract + session identity guard | Instrumented real-WebView2/WebKitGTK event-flow stress |
| Visual reference 30 Windows + 29 Linux screens | 1290x852 reference registry, real runtime QA screenshots | **59 individual native screenshot comparisons unverified** |

## FileZilla benchmark (free Client)

FileZilla's FTP/FTPS/SFTP, resume, tabbed sites, proxy, IPv6, folder comparison, remote editing, synchronized navigation, bookmarks and transfer queue form the functional baseline; see [parity matrix](../qa/FILEZILLA_PARITY.md). Ghost FTP's additional workflows (such as supported cloud backends, offline management tooling and deeper transfer controls) are potential differentiators, **not proof of superiority**.

Priority gaps to validate before parity claims: HTTP/SOCKS5/FTP proxy E2E, active FTP/IPv6, MLSD/MLST feature negotiation, large-file (>4 GiB) resume across vendors, synchronized browsing, full remote search/filter coverage and all native UI screens. Use reproducible scenario timings and evidence rather than marketing superlatives.

## Visual fidelity rule

The reference archive contains 30 Windows and 29 Linux concepts. Their registry is an inventory, not pixel-parity acceptance. Keep existing dark premium brand, dimensions, icons and authentic native screenshots. The CI capture workflow cannot automatically compare every screen to every reference or prove usable operation on all DPIs. Record viewport, OS, build SHA, captures, pixel differences and a manual pass/fail per screen before declaring 1:1.
