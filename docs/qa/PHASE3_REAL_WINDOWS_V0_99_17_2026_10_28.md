# Phase 3 — actual v0.99.17 Windows build vs canonical reference (28 October 2026)

## Provenance

- Build source: **main `c78756525ac9dedd4fc5cd49bc0dffb85f9a276d`**, published as [v0.99.17](https://github.com/bren-wp/Ghost-FTP/releases/tag/v0.99.17).
- Tested native Windows QA: [GitHub Actions run 40774845652](https://github.com/bren-wp/Ghost-FTP/actions/runs/40774845652), downloadable workflow artifact **12854018458** (`GhostFTP-Windows-x64-Preview-QA`, 42 items: 7 canonical + 14 smaller-size PNGs plus per-image capture metadata).
- Analyzed exact downloaded archive `GhostFTP-Windows-v0.99.17-Native-QA.zip`, SHA-256: `583236b1642e271f87cdfaee6ebbfde57a614458f3c11669971f75f491a4b891`.
- Compared with supplied `GhostFTP-Premium-Brand-UI(10).zip` SHA-256 `1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877` (byte-identical to accepted v8 concept source).
- Both sides are **1290 × 852**. Capture metadata records `WindowTitle=Ghost FTP`, `WindowBounds=1290x852` and the executable in the real Windows CI Rust/Tauri release build. The native QA uses `PrintWindow(PW_RENDERFULLCONTENT)` for many views and WebView2 compositor fallback for Files, not synthetic design PNGs.
- Method: decode both images to RGBA; every nonzero channel delta counts as one changed pixel; RGB MAE is the mean over all three absolute 8-bit RGB channel differences.

## Seven actual capture comparisons

| Design screen | Actual Windows QA PNG basename | Different pixels | Different fraction | RGB MAE / 255 | Actual capture SHA-256 prefix |
| --- | --- | ---: | ---: | ---: | --- |
| `windows/files` | `GhostFTP-Windows-Preview-Native-Window.png` | 1,063,027 | 96.72% | 15.24 | `88e3094ea19f4c16` |
| `windows/sites` | `GhostFTP-Windows-Preview-Site-Manager.png` | 1,032,856 | 93.97% | 17.67 | `4c8fc79f1cbad25a` |
| `windows/new-connection` | `GhostFTP-Windows-Preview-New-Connection.png` | 1,098,667 | 99.96% | 15.66 | `8d55515a9eb8efc6` |
| `windows/settings` | `GhostFTP-Windows-Preview-Preferences.png` | 1,069,768 | 97.33% | 13.87 | `7393269dbc8ca964` |
| `windows/transfers` | `GhostFTP-Windows-Preview-Transfer-Center.png` | 1,068,513 | 97.22% | 15.01 | `490286d942acb008` |
| `windows/properties` | `GhostFTP-Windows-Preview-File-Properties.png` | 1,098,702 | 99.97% | 15.75 | `18b095e96ebaf10c` |
| `windows/help` | `GhostFTP-Windows-Preview-About.png` | 1,064,914 | 96.89% | 14.05 | `4b8559e13346f0ba` |

**The fraction is the proportion of RGBA pixels whose *coordinates* differ, NOT an overall percentage of unusable UI or an objective UI quality rating.** Small shifts or different real file names alter most per-coordinate pixels. The reference is illustrative, with fabricated saved profiles and transfer data; the real Windows QA deliberately uses honest disconnected/empty state, so it must not be filled with fake reference data merely to improve this metric. Window decorations/compositor capture methods also affect alignment. Dark navy/cyan brand colors are visually shared but not sufficient for pixel-exact signoff.

## Confirmed UI discrepancies and bounded next steps

1. **Site Manager:** reference has a two-column site inventory and selected-detail panel; genuine fresh-install capture shows a centered **Add your first site** card. This is a legitimate *state difference*, but the empty-state layout can still be assessed against the reference's spacing/card styling. Capture an additional actual state with a deliberately created disposable **local test server** connection, NOT a fabricated production profile, to inspect the populated path.
2. **Files:** the build captures the live Windows runner's local C: directory and honest disconnected remote panel; do not replace it with the reference's fictional file tree or pretend a transfer is active.
3. **New Connection, Settings, Transfers, Properties, Help:** before comparing geometry line by line, use matching UI state, theme, locale, fonts/window decorations and screen scale. Preserve real errors and accessible control labels even when the illustrative design differs.
4. Compare seven screens at compact/minimum window sizes from the same QA artifact for clipping/responsiveness, *separately* from the canonical 1290×852 pixel measurement.
5. Capture additional installed UI states for all **remaining 23 Windows + 29 Linux + 16 Android** concepts. The current 7 are **compared, not human accepted**. No macOS reference exists in the supplied ZIP.
6. After genuine on-device UX review, record per-screen SHA, state, commit, OS, viewport, interaction/accessibility evidence in `docs/qa/PREMIUM_REFERENCE_EVIDENCE.json`. Never claim all 75 based on build success or the reference ZIP.

## Acceptance status

- Source reference integrity: **75/75** design PNGs verified (30 Windows, 29 Linux, 16 Android).
- Actual independent Windows installed-build comparison performed: **7/75** source concepts (all **non-exact**).
- Independently human-approved 75-screen parity: **0/75**. No existing evidence status was silently upgraded.
- This audit was performed against the preceding released build **0.99.17**. It is a regression baseline for PR #242 / **0.99.18**, **not** fresh evidence of that unmerged build's visual acceptance.
