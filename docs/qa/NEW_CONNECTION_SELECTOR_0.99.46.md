# Ghost FTP v0.99.46 — Windows/Linux New Connection selector QA

**Author:** Brendigo — https://brendigo.com  
**Source release for baseline captures:** v0.99.43 Preview, commit `48295d60eb171f34b516bfbf650ad2975f19ae4f`  
**Genuine native build:** [GitHub Actions run 42903097538](https://github.com/bren-wp/Ghost-FTP/actions/runs/42903097538)

## Reproduced mismatch, not simulated

Two real application New Connection screenshots, Windows WebView2 and Linux GTK/WebKit, were compared at **1290 × 852**, **1,099,080 RGBA coordinate pixels each**, with the same idle SFTP dialog state.

| Platform | Actual screenshot | SHA-256 |
| --- | --- | --- |
| Windows | `GhostFTP-Windows-Preview-New-Connection.png` (native QA archive) | `fa2107f79e39c091b49b1071efb4e53c4cd70a1b6e01744c686c70f61e22f7a6` |
| Linux | `newConnection/Linux-New-Connection-1290x852.png` (native QA archive) | `e3afdc6d36713c5d74423311ea1cd7c87d761d8af3bef47c07a2523d0b5e8bcf` |

**Cross-platform difference:** 511,488 of 1,099,080 pixels (46.5378%). This includes legitimate platform chrome, font rasterization, different local file data and layout differences; it is NOT a comparison against the 59 design references.

A concrete defect is visible in the Protocol control at screenshot coordinate (520, 290): Linux = RGBA `(52, 52, 52, 255)` (GTK gray), Windows = RGBA `(11, 25, 43, 255)` (branded dark field).

## Implemented source remediation

- `ghostftp-desktop/src/components/QuickConnectionDialog.tsx`: keep the actual native `<select>` and real `onChange` logic. Wrap it with a shared presentational chevron (`aria-hidden`, pointer-events disabled); never create a synthetic menu.
- `ghostftp-desktop/src/styles.css`: normalize platform browser appearance, force dark native option popup matching the dialog, consistent padding/foreground/background and fixed ChevronDown position.
- Preserve real Windows high-contrast/forced-colors behavior by restoring OS-native select chrome and `Canvas`/`CanvasText`.
- `ghostftp-desktop/scripts/check-ui-contract.mjs`: fail CI if native selection, dark option styling, decorative/non-interactive icon or high-contrast fallback is removed.
- No new images, fake profiles, simulated transfer data or startup splash.

## Acceptance still required

The source fix requires actual installed Windows/Linux functional verification: SFTP/FTP/FTPS switching with keyboard and pointer, smaller viewports, Linux GTK variants, Windows high-contrast and accurate source-SHA capture provenance. Pixel matching to historical design files is not required.
