# Ghost FTP — 75-screen pixel comparison (real captures versus design ZIP)

The original uploaded `GhostFTP-Premium-Brand-UI(8).zip` contains **75 individual PNG concepts**: Windows 30, Linux 29, Android 16. These are **mockups, not actual UI screenshots**. The supplied README cautions that server paths, login identities, progress values and transfer statuses are invented illustrative content. Never fabricate these in the product merely to achieve a numeric image match.

## Provenance and exact dimensions

- ZIP SHA-256: `1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877`.
- Windows: 30 screens, **1290 × 852 px**; Linux: 29 screens, **1290 × 852 px**.
- Android: 16 screens, **480 × 960 px**.
- Original assets: `branding/ghostftp-app-icon.svg`, `ghostftp-adaptive-foreground.svg`, `ghostftp-android-splash-1080x1920.png`, PNG/icon/logo variants, source HTML and manifest.
- The ZIP does **not** contain approved separate native macOS UI references. Never infer macOS pixel certification from the shared Windows/Linux concepts.

## Reproducible, non-destructive comparison

1. Install Python 3 and Pillow in a local isolated environment.
2. Capture the actual installed Ghost FTP screens at the **exact reference resolutions**. Arrange captures as `captures/windows/files.png`, `captures/linux/files.png`, `captures/android/files.png`, etc. Capture each named state without exposing genuine passwords, bearer links, tokens, server names or private filenames.
3. Run:

   ```bash
   python tools/qa/compare-screen-references.py \
     --references "GhostFTP-Premium-Brand-UI(8).zip" \
     --captures captures --output build/qa/pixel-diff --platform all
   ```

4. Review `results.csv` and `results.json` plus `<platform>/<screen>/diff.png`. The script refuses to rescale screenshots, reports missing/wrong-sized captures and exits nonzero until all selected references have comparable captures.
5. Use same OS theme, 100% display scale/DPI, system bars, density, language, font availability, connection state and fixed viewport before interpreting any pixel values. Dynamic information requires a separate safe functional assertion, not invented server data or screenshots.

**Important:** even when all 75 files are compared, low MAE alone is not evidence that actions, transport security or accessibility work. Review clickable bounds, state changes, responsive viewports, contrast, keyboard/TalkBack, and FTP/FTPS/SFTP end-to-end tests separately. No auto-approval of screens from a CI-green build.

## Compatibility with existing work

`docs/qa/ANDROID_RUNTIME_VS_16_REFERENCE_2026_10_25.md` evaluates a **different set of Android collage concepts** with 13 real v0.99.10 emulator captures. Those reference names and appearances are not interchangeable with this 75-screen ZIP. Treat them as two distinct design baselines until the user explicitly chooses one. Where concepts conflict, preserve functionality and document the choice rather than silently merging pixels.

Visual screen status: **0/75 pixel-approved** on this newly supplied artifact; the source ZIP is verified but paired, exact-viewport installed screenshots have not yet been reviewed.
