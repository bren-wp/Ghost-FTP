# Ghost FTP Android — 16-screen visual reference acceptance

Reference: user-provided **Ghost FTP Android UI maketa.png** (2026-10-16), a 16-screen premium dark navy/purple grid. This document records **real source implementation** and open acceptance gaps. No synthetic host names, saved accounts, transfer percentages, unverified cloud authorizations, or remote paths may be injected to make screenshots resemble the artwork.

## Functional and visual frame checklist

| Frame | Reference | Native Android source | Status (code-only, not 1:1 pixel acceptance) |
| --- | --- | --- | --- |
| 1 | Splash Screen | `SplashActivity.kt`, branded vector | Implemented; pixel comparison pending |
| 2 | Sidebar / Navigation | `MainActivity.buildNavigationRail`, real active-session card | Improved; real session required for connected-state capture |
| 3 | Sites List | `buildSitesOverview/refreshSiteRows`, FTP/FTPS/SFTP color-coded server tile | Improved; real saved sites required for populated-state screenshot |
| 4 | Add / Edit Site | `buildConnectionCard`, General Cancel/Save, Advanced Quick Connect, Sync navigation, credential-safe edit | Partial; responsive field widths and focus need live pixel QA |
| 5 | Connect Screen | `buildConnectPreview/showConnectPreview` show real saved-site host, port and username; session-only password; existing `openConnection` | Implemented; visual and real-network acceptance pending |
| 6 | File Manager (Dual Pane) | `buildFilesCard`, real remote/app-owned local files, wide dual-pane and adaptive narrow single-pane, vector actions | Responsive candidate; arbitrary public folders still require SAF grants |
| 7 | Transfer Queue | `buildTransfersCard`, session operation queue/history and real busy indicator | Partial; native backend does not own a persisted parallel queue |
| 8 | Transfer Details | `openTransferDetails`, verified current operation/cancellation | Partial; no fabricated progress/ETA/throughput |
| 9 | Cloud Storage | `buildCloudCard`, native provider vector icons and signed HTTPS object operations | Partial; Google Drive/Dropbox/OneDrive/iCloud account OAuth not implemented |
| 10 | Bookmarks | `buildBookmarksCard`, actual user saved remote bookmarks | Implemented; populated-data comparison pending |
| 11 | Remote File Search | `buildSearchCard`, bounded live folder or app-owned local search | Partial; no imaginary remote subtree indexing |
| 12 | Sync / Compare | `buildSyncBackupCard` plus `compareCurrentDirectories`, local/remote folders and fail-closed read-only comparison | Improved; automatic synchronization and destructive mirror intentionally disabled |
| 13 | Remote File Editor | `buildRemoteTextEditorCard`/fixed footer and `ConnectionController` strict UTF-8 streaming <=512 KiB, guarded staged upload | Functional candidate; syntax highlighting/multiple encodings not implemented, device QA pending |
| 14 | Batch Operations | `buildBatchRenameCard`, `RemoteBatchRenamePlanner` with real preview and bounded sequential collision-safe rename | Functional rename candidate; attributes/time and transactional rollback are not implemented |
| 15 | Settings | `buildSettingsCard`, premium category grid and real links/actions | Implemented; pixel comparison pending |
| 16 | About | `buildAboutCard`, real version/product data and link cards | Implemented; pixel comparison pending |

**Important:** “Implemented” means there is a real code path, **not** 1:1 pixel similarity or real-device acceptance. In particular, the current app cannot be called complete for all sixteen references. User-provided artwork contains illustrative sample FTP account/server names and transfer rates; copying them into installed code would misrepresent connections and device state.

## Visual/instrumentation process

1. Run Android 16 emulator at **480 × 960 pixels** with **480 dp** width and run `PremiumVisualReferenceTest` and `MainActivitySmokeTest`. Retain original APK/AAB and screenshots from the same source SHA.
2. Capture splash, closed/open drawer, empty/real Sites, general editor, connected/disconnected Files dual pane, Queue/History, real transfer Details, cloud, bookmarks, search, read-only sync, settings and about. Validate smaller 360dp phones, font scaling 200%, TalkBack, touch target widths and back navigation.
3. Compare side by side to each reference **for visual hierarchy, spacing, font size, purple gradients, vector icons, rounded card radius, button states, responsive layout and safe-area insets**. Pixel-perfect validation cannot be inferred from testing accessibility tree existence alone.
4. Real account integration is required for FTP/explicit FTPS/SFTP, invalid TLS/SSH identity rejections, large transfer interruption, cancel, hash verification, saved-site upgrades and document access. No images of user credentials, bearer links or private paths should be exported as QA evidence.
5. Address missing editor/batch functionality only with explicit safeguards: bounded downloads, optimistic conflict check, staging and rollback; previews, collision checks, atomic or fail-closed batch rename, no recursive delete without confirmation. Until validated, do not display enabled actions that imply these operations work.

The user-requested UI remains the target. Completion and publication depend on **per-screen reviewed live screenshots**, functional safety acceptance, and all exact-commit GitHub Actions jobs succeeding.

## Expanded Ghost FTP 0.99.6 viewport QA

Instrumented real-view captures cover **320×640**, **360×800**, **393×852 at 1.4× font scale**, **480×960** (reference), **720×1280** (tablet) and **960×540** (landscape). All eight primary workspaces must remain navigable with no 40dp-or-smaller visible Files actions and no clipped navigation affordances. The authoritative rendered PNG hashes are stored as `RESPONSIVE-SHA256SUMS.txt` beside the CI artifact.

All sixteen concept screens remain **not formally 1:1 certified** until every comparable real-state screenshot is visually inspected against the user artwork and any mismatch is fixed. Real login, current file listings, transfer progress and cloud accounts cannot be substituted with fictional mock data.

## 0.99.8 screenshot-driven Files footer correction

A live v0.99.7 Android Files screenshot at 480x960 shows the More footer caption clipped below the visible bottom edge. The 0.99.8 code uses the same padding/geometry as the other four controls and adds instrumentation assertions against visible text baselines at responsive viewports. Remaining frames continue to need genuine source-to-installed-app visual comparison; no invented remote files or credentials may be inserted to imitate the design.

## 0.99.9 Android batch rename safety regression

The remote batch-rename preflight now rejects destination paths already occupied by **any** object in the current remote listing, including directories, ignoring case and Unicode NFC spelling differences for conservative interoperability. It also rejects duplicate/equivalent and noncanonical source paths before any FTP/SFTP mutation. JVM regression cases cover the reported aliases. CI and real-server acceptance are still required; this change does **not** establish 1:1 visual acceptance or absence of runtime crashes.

## 0.99.8 About reference improvement

The v0.99.7 live Android About capture showed a small white Ghost mark inside a raised tile, while reference frame #16 uses a larger purple Ghost brand icon. The 0.99.8 About hierarchy now renders the existing scalable premium splash vector at 72dp and asserts its presence across responsive emulator sizes. This improves genuine design alignment; remaining exact screenshot acceptance is pending.
