# Ghost FTP — installed-build visual acceptance

Visual reference authority: the user-supplied GhostFTP-Premium-Brand-UI(2).zip and the 75-entry source map in docs/qa/premium-75-screen-contract.json. The references are design **concepts**, never real file data or executable screens. Do not embed them as application backgrounds.

## What the 0.30.18 Help/About increment can verify

| Target | Reference file in ZIP | Genuine surface | Acceptance |
| --- | --- | --- | --- |
| Windows | screens/windows/help.png | ghostftp-desktop/src/components/AboutDialog.tsx + styles.css | **Pending** installed Windows capture |
| Linux | screens/linux/help.png | Same React About surface, native Linux packaging | **Pending** installed Linux capture |
| Android | screens/android/about.png | android/app/src/main/java/com/ghostftp/android/MainActivity.kt | **Pending** emulator/device capture |
| macOS | No supplied mockup | macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift | Shared-token/platform usability review only |

The source now implements the reference's main product identity, compact real version metadata, contextual facts, and adjacent support actions, but code presence **cannot** certify pixel parity. Do not set visual_acceptance to accepted until captured application screenshots have been reviewed.

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
