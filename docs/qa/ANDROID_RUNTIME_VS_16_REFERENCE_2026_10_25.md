# Ghost FTP Android — authentic 16-screen reference comparison (v0.99.10 runtime → v0.99.11 improvements)

**Acceptance:** NOT pixel-perfect / NOT 16-of-16 certified. This is a first real-capture inventory and qualitative visual gap review, not a mathematically registered pixel diff or proof that every on-screen action is functional.

## Source and reproducible provenance

- Real Android app: [successful v0.99.10 Android CI run](https://github.com/bren-wp/Ghost-FTP/actions/runs/40136310707), exact main source SHA `af9eb96093f8655792163bbe19bc8012ede3781e`, workflow artifact `GhostFTP-Android-Premium-Visual-QA-480x960` (artifact ID `12578100059`). The 13 captured native app views are 480 × 960 PNGs from Android emulator at 160 dpi; production FLAG_SECURE was not disabled to take screenshots. They are credential-free, actual View-tree renders, not fabricated FTP sessions.
- Approved appearance references: the user's previously supplied **Ghost FTP Android UI maketa.png** (1536 × 1024, SHA256 `0d757905ab5905dd3ab722ef87c245dc167e9d670505c7ef873f91b0b4e989d1`) and **Ghost FTP Android UI Showcase.png** (1586 × 992, SHA256 `189445f52079342606d1fa0a4d5b6977ce4f4d1a1b113bfd753fdaf093cc245c`). These are 16-screen *collages*, not pixel-registered individual 480 × 960 screenshots. Their hashes are **different** from earlier `docs/qa/android-16-screen-reference-2026-10-16.json` variant file hashes and must not be mistaken for the exact binary variant recorded there.
- The mockups intentionally contain invented server names, accounts, file lists and transfer-progress values. Production MUST NOT insert those values merely to match pixels. Compare layout in authentic empty/disconnected states, and separately test genuine connected data with an isolated real test server.

## All 16 reference screens, observed runtime state and unverified gaps

| # | User's design screen | Real 0.99.10 emulator capture | Assessment and next real acceptance |
| --- | --- | --- | --- |
| 1 | Splash Screen | `splash.png` | Brand, vertical placement and progress baseline exist; v0.99.11 unifies launch, system and in-app violet symbol. Design's glow and type sizing remain to be measured in an installed v0.99.11 capture. |
| 2 | Sidebar / Navigation | `drawer.png` | Real slide-over navigation exists; icon scale, row spacing and interaction states need measured design comparison. No fake active server status permitted. |
| 3 | Sites List | `sites.png` | Real empty-state and search exist. Concept's populated mock list is not an appropriate runtime oracle; compare search, saved profile tiles using actual isolated profiles and empty-state usability. |
| 4 | Add / Edit Site | `site-editor.png` | Real form and tabs exist. Compare input height, label contrast, tab alignment, keyboard/IME behavior and secret masking. |
| 5 | Connect Screen | **Not captured** | Capture true selected saved-site Connect route without persisted passwords; test authenticated connection, host-key/certificate feedback and cancel. |
| 6 | File Manager (Dual Pane) | `files.png` | Actual disconnected local/remote panes and footer exist; compare pane geometry, icons, readable file rows and responsive split. No fictitious remote files. |
| 7 | Transfer Queue | `transfers.png` | Real idle queue captured; mock percentages are not evidence. Capture real queued/upload/download/cancel/pause states with disposable test-server files. |
| 8 | Transfer Details | `transfer-details.png` | Real idle detail route exists; compare typography/action layout. Validate true byte counts, error/retry/cancel transitions through protocol E2E. |
| 9 | Cloud Storage | `cloud.png` | Provider cards show honest disconnected state; compare provider vector artwork, spacing and explanatory navigation; never simulate an OAuth login. |
| 10 | Bookmarks | `bookmarks.png` | Actual empty-state captured; compare folder icon/list card with temporary real bookmarks; no stored credentials. |
| 11 | Remote File Search | `search.png` | Real form exists. Concept includes file size and modification-time selectors not present in the inspected real idle form; do not add decorative nonfunctional filters. Test remote search and unsupported recursive paths honestly. |
| 12 | Sync / Compare | `sync.png` | Real directory comparison and disabled unsupported destructive options; compare toggle sizes/labels. Must not enable mirror/delete without verified safe engine. |
| 13 | Remote File Editor | **Not captured** | Capture with a disposable real remote text file, ensure syntax display, save/overwrite integrity and server error handling, never a fake PHP screenshot. |
| 14 | Batch Operations | **Not captured** | Capture real selected entries and preview. Test no-overwrite, case-folding, NFC collision and cancel before invoking any server mutation. |
| 15 | Settings | `settings.png` | Real categories render in two columns; compare tap target, readable titles, spacing and font scaling at 320 dp and large-font 393 dp. |
| 16 | About | `about.png` | Real product facts/actions and icon exist; compare brand size and card geometry without claiming mock version number is the installed version. |

**Real screenshot inventory:** 13/16 captures. **Approved pixel-perfect screens:** 0/16. **Untested reference states:** Connect, Remote File Editor, Batch Operations (3). `PNG exists` is *not* equivalent to `pixel comparison passed`.

## Actual code improvements in v0.99.11

- One reviewed Ghost silhouette for Android adaptive launcher, Android 12+ system splash and brand in the app; correct violet Android palette rather than cyan platform icon drift.
- Full 48 dp touch targets on real top-bar Back/Files/Quick buttons, 48 dp settings rows, button helpers and action footer; preserve 52 dp bar.
- Six splash, action and spoken protocol labels localized across 24 Android languages, tested with the resource checker.
- Desktop PNG/ICO asset validation and cross-platform navigation/brand token checks; they do not prove exact screenshot parity on Windows, Linux or macOS.
- Production FTP, FTPS, SFTP credentials, paths, filenames, flags and database are unchanged by these visual alterations.

## How to reach honest 1:1 acceptance

1. Capture **all 16 real app states** on each target Android device profile at fixed 480×960/160dpi, plus narrow/large-font/tablet/landscape variants, recording commit, package, theme, OS, density and SHA256.
2. Obtain individually exported 480×960 reference screenshots. Collage crops must be perspective-corrected and aligned first; a numeric pixel delta from different aspect ratios, clipping and system bars would be misleading.
3. Compare aligned image pairs for measured bounds, typography, icons, color, focus/accessibility, scrolling and baseline geometry; log deviations and retest after each targeted fix.
4. Run every action in the rendered state against real isolated local and remote files; preserve fail-closed security and make legitimate empty/error states explicit.
5. Record reviewed per-screen evidence before changing visual-acceptance flags. Do not automatically approve from CI compilation or static color assertions.

## Other platforms

The repository inventories **30 Windows** + **29 Linux** design concepts, but the original 75-reference image ZIP is not available in this capture review. Windows and Linux share the Tauri UI; exact visual comparison must inspect authentic installed windows and package snapshots, not assume parity from shared source. The native Swift macOS UI has **no approved reference mockups**; match semantic branding and native behavior, but do not assert screenshot-exact 1:1 parity. Secure FTPS/SFTP implementation on native macOS remains issue [#223](https://github.com/bren-wp/Ghost-FTP/issues/223).
