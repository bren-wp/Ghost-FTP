# Ghost FTP Android — 16-screen visual reference acceptance

Reference: user-provided **Ghost FTP Android UI maketa.png** (2026-10-16), a 16-screen premium dark navy/purple grid. This document records **real source implementation** and open acceptance gaps. No synthetic host names, saved accounts, transfer percentages, unverified cloud authorizations, or remote paths may be injected to make screenshots resemble the artwork.

## Functional and visual frame checklist

| Frame | Reference | Native Android source | Status (code-only, not 1:1 pixel acceptance) |
| --- | --- | --- | --- |
| 1 | Splash Screen | `SplashActivity.kt`, branded vector | Implemented; pixel comparison pending |
| 2 | Sidebar / Navigation | `MainActivity.buildNavigationRail`, real active-session card | Improved; real session required for connected-state capture |
| 3 | Sites List | `buildSitesOverview/refreshSiteRows`, FTP/FTPS/SFTP color-coded server tile | Improved; real saved sites required for populated-state screenshot |
| 4 | Add / Edit Site | `buildConnectionCard`, General/Advanced/Sync route, credential-safe edit | Partial; form geometry and focus need live pixel QA |
| 5 | Connect Screen | `buildConnectionCard` and `openConnection` | Partial; no standalone reference-style server details page |
| 6 | File Manager (Dual Pane) | `buildFilesCard`, real remote and app-owned local files, selectable dual/remote/local mode, vector actions | Improved; actual public Downloads are unavailable without SAF grants |
| 7 | Transfer Queue | `buildTransfersCard`, session operation queue/history and real busy indicator | Partial; native backend does not own a persisted parallel queue |
| 8 | Transfer Details | `openTransferDetails`, verified current operation/cancellation | Partial; no fabricated progress/ETA/throughput |
| 9 | Cloud Storage | `buildCloudCard`, provider cards and signed HTTPS object operations | Partial; Google Drive/Dropbox/OneDrive/iCloud account OAuth not implemented |
| 10 | Bookmarks | `buildBookmarksCard`, actual user saved remote bookmarks | Implemented; populated-data comparison pending |
| 11 | Remote File Search | `buildSearchCard`, bounded live folder or app-owned local search | Partial; no imaginary remote subtree indexing |
| 12 | Sync / Compare | `buildSyncBackupCard` plus `compareCurrentDirectories`, local/remote folders and fail-closed read-only comparison | Improved; automatic synchronization and destructive mirror intentionally disabled |
| 13 | Remote File Editor | No equivalent writable text-editor screen | Missing |
| 14 | Batch Operations | Existing single-object rename; no safe batch-rename preview/apply engine | Missing |
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
