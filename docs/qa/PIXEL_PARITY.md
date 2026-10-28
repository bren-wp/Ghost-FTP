# Ghost FTP — installed-build visual acceptance

Visual reference authority: the SHA-256-pinned `GhostFTP-Premium-Brand-UI(8).zip` reference, byte-identical to the user-supplied `(10).zip` (SHA-256 `1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877`), and the 75-entry source map in `docs/qa/premium-75-screen-contract.json`. Earlier v7 designs are historical only. The references are design **concepts**, never real file data or executable screens. Do not embed them as application backgrounds.

## What the 0.30.18 Help/About increment can verify

| Target | Reference file in ZIP | Genuine surface | Acceptance |
| --- | --- | --- | --- |
| Windows | screens/windows/help.png | ghostftp-desktop/src/components/AboutDialog.tsx + styles.css | **Pending** installed Windows capture |
| Linux | screens/linux/help.png | Same React About surface, native Linux packaging | **Pending** installed Linux capture |
| Android | screens/android/about.png | android/app/src/main/java/com/ghostftp/android/MainActivity.kt | **Pending** emulator/device capture |
| macOS | No supplied mockup | macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift | Shared-token/platform usability review only |

The source now implements the reference's main product identity, compact real version metadata, contextual facts, and adjacent support actions, but code presence **cannot** certify pixel parity. Do not set visual_acceptance to accepted until captured application screenshots have been reviewed.

The most recently supplied visual reference archive enumerates **30 Windows**, **29 Linux** and **16 Android** concepts (75 total). No supplied macOS mockups are present. The reference images are static design renderings, not valid production screenshots; they must never replace installed-app release evidence.

## Reference geometry to compare

Desktop mockups use **1290×852** reference images. The left primary rail targets **214 px** and main navigation rows **43 px** at that size. In the Help reference, the page presents a heading and two large adjacent cards (product facts on the left, support/resources on the right). Verify the real desktop About state at 1290×852, then at 1100×720, 840×560 and the minimum supported window size without cropped primary actions. Preserve the real top-level menu/navigation and avoid inserting a nested duplicate rail.

Android mockups are **480×960**. The About reference has the real Ghost brand centered inside a dark raised card, then a separate facts card; bottom navigation, system bars and scroll behavior must be tested on small devices as well. Unlike mockup-only elements, real external resource links must remain reachable by ordinary vertical scrolling and have spoken accessibility names.

The concept's *fake* paths, host names, credentials, status indicators, transfer percentages and file sizes must **never** be seeded in the installed product to imitate the screenshot.

## How to accept a screen

1. Install/run a CI-built package on the actual target OS, with no seeded test profiles or fake transfers; record exact commit SHA, OS version, app version, viewport and display scale.
2. Capture the named real app state; inspect typography, icon/wordmark proportions, sidebar geometry, card layout, spacing, radii, borders, focus order and keyboard access.
3. Compare against the correct ZIP reference at matching dimensions; record screenshot paths, measured differences, human acceptance, and legitimate deviations (especially accessibility or smaller window layouts).
4. Click every visible control, including Update, GitHub support/docs/privacy/EULA links, scroll/focus/keyboard navigation and window resize. Confirm honest errors if external browser is absent.
5. Only then update that single screen's visual_acceptance in the 75-screen source map, with traceable evidence. No blanket bulk acceptance.

For macOS, supplied reference screenshots do not exist; require separate functional, accessibility, security and premium-token acceptance without calling that platform a pixel-perfect 1:1 screenshot reproduction.

## Shared premium tokens

Canvas #070E1A; panel #0D192B; raised #111F34; action blue #38ABFF; primary text #EAF6FF. Respect actual application theme controls and reduced-motion preferences.

## Known gates

The six exact-HEAD CI suites verify builds, source regression contracts, selected security behavior, Android smoke tests, Swift unit tests and protocol roundtrips. **None automatically measures all 75 installed UI screenshots.** A green release build therefore does not grant 1:1 final sign-off.

Do not turn hidden or unfinished macOS FTPS/SFTP operations on until reliable certificate/hostname and SSH host-key validation is implemented and tested; the About panel must continue to describe this accurately.

## Verified Windows README captures (unreleased 0.30.20 documentation maintenance)

The existing seven `README.md` image **paths, anchors, ordering and layout are unchanged**. Their corresponding PNG file bytes are now sourced from the real Windows 0.30.19 native QA run `37879036243` at main SHA `9412fbac440d4231a3ebdb7869a1f171240e9681`, artifact `11593868391`; the workflow verified a 1290×852 window, Ghost FTP title and genuine desktop capture method for each screen. `.github/scripts/check-release-images.mjs` pins their SHA256 values and keeps every unrelated documentation image tied to the latest published tag.

The captured file manager shows the real Windows CI runner's local files and the honest disconnected remote state. They are **not** the mocked reference's fake production servers. These seven captured states provide actual evidence, not acceptance of the entire 75-concept screenshot set. Pixel-parity and click-by-click reviews on additional installed OS/device configurations remain pending.

## 0.95.0 — Android screenshot evidence with FLAG_SECURE retained

Normal ADB screen capture is blocked or black while `FLAG_SECURE` protects Ghost FTP windows. The previously archived `post-test-screen.png` from Android CI was black; it cannot establish visual correctness. In a **debug-only, instrumented and idle** process, `PremiumVisualReferenceTest.kt` renders actual application Views to private PNGs without lowering the runtime screen-protection policy. A strict editable-field allowlist prevents saving credential-bearing state. The isolated emulator runs at 480×960/160dpi; seven named workspace renderings and SHA256 checksums are published only in the separate visual QA workflow artifact.

Do not accept a reference solely because a PNG exists. Every PNG must be compared to its appropriate concept after reviewing whether the genuine idle state can meaningfully match that design; verify text, icon positioning, panel geometry and interactive controls. No approved Android Cloud reference exists in the 16-image archive, so Cloud gets an additional evidence image without a 1:1 acceptance claim. All 75 registered reference acceptance fields remain pending until manually verified and recorded per screen.

## 0.99.11 cross-platform icon and accessibility baseline

The source now enforces the same reviewed Ghost silhouette and violet palette across Android adaptive launcher, Android 12+ system splash and Android workspace symbol. Native Back/Add and standard action targets require at least 48 dp; labels for these controls and startup status are provided in all 24 native language resources. Windows/Linux branded PNG and ICO bundles reject malformed header/directory entries, and shared source checks prevent dimension/palette drift. The Android emulator exercises both the actual icon-bearing screens and touch geometry.

This is an improved reproducible **implementation baseline**, not a blanket 1:1 approval. The 75 reference concepts are not themselves installed screenshots; their complete reference ZIP is not part of the repository. Real on-target captures and per-screen differences are still needed. No screen is moved from `pending` to accepted based on these source checks alone.

## 0.99.13 — final v8 reference supersedes historical violet Android designs

The new user-supplied `GhostFTP-Premium-Brand-UI(8).zip` is pinned by SHA-256 and exact 75-screen manifest. This includes 30 Windows, 29 Linux and 16 Android *concept images*, no macOS reference images. Cyan-blue `#38ABFF` with a persistent 62dp phone rail at 480dp is now the Android design target, superseding earlier violet references. Use `tools/qa/compare_reference_v8.py` and `docs/qa/REFERENCE_V8_PIXEL_AUDIT.md` to compare genuine installed captures. **Not one of the 75 screens is certified pixel-exact solely from these changes.**

## 0.99.18 Phase 3 — exact RGBA comparisons and proof boundaries

The canonical 75-entry visual audit is **30 Windows** and **29 Linux** references at **1290×852** plus **16 Android** references at **480×960**. It does not contain a native macOS screenshot set.

The Phase 3 comparator `tools/qa/compare_reference_v8.py` now compares **RGBA**, not merely RGB. A one-unit change in any color or alpha channel is counted as a changed pixel; images with invisible icons due to different opacity can no longer receive a false `exact: true` result. Each wrong-size, symlinked, invalid PNG or unreadable real capture is recorded individually, allowing the remaining images to be audited. `--require-all-captures` fails on missing or invalid images and `--require-exact` additionally fails when any RGBA pixel differs. These are *measurement gates*, **not proofs of real runtime origin**.

Example after collecting genuine target-OS screenshots:

```bash
python3 tools/qa/compare_reference_v8.py \
  --reference-zip "GhostFTP-Premium-Brand-UI(10).zip" \
  --captures-dir "build/qa/installed-captures" \
  --output "build/qa/phase3-rgba.json" \
  --require-all-captures
```

To check literal 1:1 RGBA equality as an additional diagnostic, append `--require-exact`. The comparison does **not** require that illustrative fake server/file values be reproduced in the real app. Legitimate data, accessibility, and platform-safe deviations may produce nonzero differences and require human review.

Capture directory structure is `<platform>/<screen-id>.png`, with an externally traceable record of the exact Git commit, installed package SHA-256, OS/device, viewport size, display scaling, locale/theme, app state, and relevant click-through interactions. Treat debug-only Android capture as privacy-sensitive; preserve `FLAG_SECURE` in production. Never copy the supplied concept PNG into an installed capture folder or promote a design image to evidence.

**Acceptance as of this phase:** 75 reference sources verified, 0/75 independently human-accepted installed screenshots. Any published README images must identify actual source captures and must not be described as newly verified 75/75 parity.

