# Ghost FTP — capability and acceptance roadmap

Baseline: published v0.99.87, main 2fee225756b9f85b1715f5dfed660b875dda744d (26 November 2026). Next source: 0.99.88 Preview. Windows 10/11 x64 and Linux x86_64 only. Original Ghost FTP code, brand and UI. Feature presence in source is not proof of installed native acceptance.

## Evidence and gaps

| Area | Source evidence at baseline | Remaining work |
| --- | --- | --- |
| Transfer reliability | transfer.rs, transfer/download_staging.rs, transfer/remote_upload_staging.rs, transfer/integrity.rs; real protocol CI | 4+ GiB concurrent transfers, full disk, cancellation, provider mutation, hard reset; independent digest verification; per-profile limits and safe segmented download |
| Sync | sync.rs, foldersync.rs, SyncDialog.tsx; one-way additive/mirror, indexed source snapshot, exclusions | Explicit three-way bidirectional planner, delete/modify conflict resolution, preview and persistent both-side baseline, content verification, no automatic destructive first sync |
| Search | search.rs, FleetSearch.tsx; filename/content, include/exclude globs, regex and bounded results | Shared filter presets and date/size rules across browse, transfer and sync; benchmark large inventories |
| Connection | profiles/mod.rs, session/mod.rs, session/ftp.rs; SSH agent/key/password, single jump host, explicit FTPS | Test network loss/reconnect, multi-hop SSH, independent bastion identity and auth, SOCKS5/HTTP CONNECT and implicit FTPS with explicit certificate validation |
| Sites and navigation | DualPaneBrowser.tsx, profiles/mod.rs; saved paths, live tabs, grouping, favorites, tags | Linked browsing rooted per profile, paired directory bookmarks, templates, richer imports with no plaintext secret persistence |
| Scheduler | transferScheduleStore.ts; one persisted target, once/daily/weekly | Durable multi-job native scheduler with independent execution IDs, missed-run policy, timezone/DST, restart/retry dedupe, dry-run and per-job logs; no separate public CLI |
| Editor | editor.rs, BuiltinCodeEditor.tsx; native external edit, embedded code editor | Verified remote revision before upload, conflict comparison, recovery copy, large-file bounds, multi-file edit |
| Cloud | remotefs and session adapters for object, WebDAV, HTTP, Drive, OneDrive, Dropbox, Box and business providers | Authorized real-provider tests, multipart checkpointing, conditional writes, server-side copy, version history and metadata controls only where supported |
| SSH and mount | terminal.rs, virtualfs/windows.rs; terminal, feature-gated Windows Cloud Filter | Local/dynamic forwarding with explicit bind scope, per-hop auth, lifecycle limits; installed mount acceptance and Linux feasibility; no standalone server |
| Credential protection | credentials.rs, linux_secret_service.rs, native candidate | Prepared security commits, real KWallet lock/restart, Windows installed acceptance, audit temporary memory, no production candidate switch |
| Encryption | backup.rs; encrypted local backup | Independently designed/maintained interoperable client-side encrypted vault, authenticated filenames and migration; no ad-hoc cryptography |
| Release and UX | real native captures, package workflows, static audits | Signed publisher identity, installed upgrades, accessibility, DPI, large directories, three audits and 15 assets per Preview; no claim of blanket superiority |

## Delivery order

- P0: recover the four prepared security commits, test exact SHA including native candidate, preserve all saved profiles, credentials and recovery data. Fix absolute sync destination root handling; audit failure paths. Reconcile documentation to real published version. No production backend migration.
- P1: transfer and navigation capability foundation: per-profile limits, safe conflict decisions, linked navigation, saved paired paths and transfer-rule presets. Define shared capability checks and wire all controls to real IPC. Test Windows/Linux and real FTP/FTPS/SFTP; no hidden automatic replay.
- P2: two-sided reconciliation and editor conflict handling. Start with read-only three-way planning, explicit conflicts and no delete execution before plan review; durable baseline update only after verified completion. Same-size changes and unavailable metadata must not be mistaken for equality.
- P3: native durable multi-job scheduling, resumable manifests and large-file adaptive segmentation. Use immutable identity and verified byte ranges; keep the existing safe fallback. Multi-connection benchmarks on the same runner, report raw samples and no claimed speedup without measurements.
- P4: additional protocol transport (implicit FTPS, SOCKS5/HTTP CONNECT, multi-hop SSH) and controlled forwarding; independently verify peer identities and fail closed. For server-to-server transfer prefer bounded verified relay; direct FXP only if security/endpoint policy permits, never arbitrary third-party FTP bounce.
- P5: cloud conditional operations and metadata, interoperable client-side encryption, advanced archive preview and safe batch renaming; archive extraction must reject traversal, symlinks, bombs and overwrite by default.
- P6: installed OS acceptance, screen-reader/keyboard support, responsive/high-DPI native UI, signed Windows/Linux packages, trusted updater, SBOM, independent audits. Stable only after actual evidence.

Every phase requires original implementation, real UI affordances only when operational, English-first plus all existing translations, no developer text, migration/rollback tests, no blind cleanup, one active PR, exact-head CI, immutable version/tag, and all 15 verified distribution assets. Preserve existing settings, transfers, files, remote data and secrets. Retain the internal Agent but do not revive retired public Server/CLI/Demo or platforms.

## External reference documentation (functional research, not source dependencies)

- WinSCP: https://winscp.net/eng/docs/feature_index and https://winscp.net/eng/docs/ui_pref_background
- FileZilla feature inventory: https://sources.debian.org/src/filezilla/3.68.1-1/README
- Cyberduck: https://docs.cyberduck.io/cryptomator/ and https://docs.cyberduck.io/cyberduck/sync/
- SmartFTP: https://www.smartftp.com/en-us/client/features
- CrossFTP: https://www.crossftp.com/sftp-client.htm
- Transmit: https://help.panic.com/releasenotes/transmit6/
- Bitvise: https://bitvise.com/ssh-client
- Rclone: https://rclone.org/bisync/

These are capability references only; no external implementation, assets or branding have been imported. Features classified as not confirmed require implementation/reachability review before duplication. No benchmark or native acceptance is inferred from documentation alone.
