# Phase 3 — real Linux Files screenshot versus v8/v10 design (2026-10-28)

## Evidence source
- Actual running Tauri/WebKit Ghost FTP on **Linux GitHub Actions Xvfb**, source main commit `c78756525ac9dedd4fc5cd49bc0dffb85f9a276d` (v0.99.17).
- [Workflow run 40774845652](https://github.com/bren-wp/Ghost-FTP/actions/runs/40774845652), artifact **12855455436**: `GhostFTP-Linux-x86_64-Preview-QA`. Its `README.txt` explicitly states the screenshot was taken at 1290×852 in an isolated Xvfb session with no fabricated server connections, files or transfers.
- Analyzed `GhostFTP-Linux-v0.99.17-Native-QA.zip` SHA-256 `8d74c7146019926f8af9ac4cc94a39cb1c669bc270805c8f4ce0c8ef12f4a3f3`.
- Actual PNG `Linux-Files-1290x852.png` SHA-256 `30e963ff5e37062551f80785c6a3c3af9a8b0df4febc76f691f309bc4cdb6767`.
- Reference image `screens/linux/files.png` from design ZIP SHA-256 `1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877`.

## Exact coordinate-by-coordinate result
- Both images: **1290×852**, full decoded RGBA comparison.
- Changed pixels: **1,055,516** of **1,099,080**, or **96.04%**.
- Mean absolute RGB channel difference: **15.41 / 255**.
- **Pixel-exact = false.** This is expected to be influenced by live user-independent Linux filesystem content, disconnected remote state, OS frame/font differences and any unaligned layout. It does not prove a 96% functional defect rate or authorize replacing actual app data with reference fiction.

## Coverage and required review
Together with [seven real Windows QA comparisons](PHASE3_REAL_WINDOWS_V0_99_17_2026_10_28.md), this is **8/75 reference screens with a comparable real app capture**: 7 Windows + 1 Linux + 0 Android. **0/75 screens are human approved** and the real-evidence registry remains `unverified` until individual interactive/visual reviews can be recorded. Android CI debug-only view renders are separate evidence to be reviewed and cannot silently count as 1:1 acceptance.

For the remaining Linux and Windows views, reproduce the actual UI state, viewport, DPI, theme, language, font and frame before pixel comparison; preserve legitimate real states, privacy, keyboard and screen-reader behavior.
