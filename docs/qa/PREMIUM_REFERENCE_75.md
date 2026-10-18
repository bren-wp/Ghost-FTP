# Premium Ghost FTP visual QA — 75-screen reference contract

## Provenance and limits

User-supplied archive `GhostFTP-Premium-Brand-UI(7).zip`, original README: **visual design concept**, not screenshots from a working Ghost FTP app. It contains 75 PNG screens, a static HTML concept, vectors and branding. Example filenames, fake transfer progress, file content and accounts inside concepts are illustrative and must never be injected into application behavior.

The archive records an earlier source snapshot (0.30.12 / 8 October 2026), not proof of today's rendered platform layout.

## Visual reference inventory

| Platform | Screens | Reference image size | Notes |
| --- | ---: | --- | --- |
| Windows | 30 | 1290×852 | Dark split-pane desktop app, navigation, toolbar and transfer workspace |
| Linux | 29 | 1290×852 | Same interaction model, platform-labelled gallery |
| Android | 16 | 480×960 | Touch-first layout, left navigation rail and context actions |
| **Total** | **75** | | Concepts are not CI captures |

Windows adds the `brand` concept; the other 29 named Windows/Linux screens match. Android concepts: files, sites, connect, transfers, sync, settings, about, new-folder, rename, delete, upload, download, host-key, activity, disconnected, transfer-detail.

## Approved design tokens

- Base canvas: `#070E1A`; surface: `#0D192B`; secondary surface `#111F34`; raised `#142640`.
- Brand blue `#38ABFF`, light cyan `#7EDEFF`, light text `#EAF6FF`.
- Success `#26DC8F` in archive README; the HTML concept uses `#41DFAB`. Before final pixel approval, resolve this source-level discrepancy explicitly. Do not invent an approved color.
- Error `#FF7284`. Fonts: Inter/Segoe UI/Noto Sans/system fallback; do not commit redistributed font binaries.

## Runtime capture evidence and false-PASS prevention

The traceable [runtime evidence registry](PREMIUM_REFERENCE_EVIDENCE.json) contains exactly one record for each of the 75 concepts. All currently have `status: "unverified"`. They are **not** software screenshots and no successful comparison is implied.

Quality CI runs `check-runtime-75-evidence.mjs` and its negative tests. Mark `pass` or `fail` **only** after capturing a genuine PNG under `docs/qa/runtime-captures/<platform>/` from an installed build, and recording:

- the original application source commit SHA (40 hexadecimal characters), capture SHA-256 and exact reference viewport;
- real PNG bytes at the stated dimensions (no symlinks, placeholders, concept art or path traversal);
- human reviewer/timestamp, keyboard, screen-reader/accessibility and interaction verification;
- an explicitly approved review for PASS or a documented discrepancy for FAIL.

A passing registry validation with `75 unverified` proves **only** traceability and prevention of unsupported assertions. It does not demonstrate visual parity, UI functionality, screen-reader acceptance or runtime screenshot collection. DPI 125/150/200%, smaller window sizes and installed OS/device testing remain separate mandatory acceptance evidence.

## Acceptance plan (not yet completed)

1. **Actual screenshots:** Capture real, freshly built, exact-SHA Windows, Linux and Android apps at matching viewports. Keep the capture output separate from design assets; never relabel a concept as a UI test result.
2. **Measured visual comparison:** Check nav rail width, header height, panel gutters, typography, semantic colors, button affordances, focus rings, disabled/loading states, scrolling, dialogs and dynamic content. Record each screen's *actual* PNG and a measurable mismatch report.
3. **Click-by-click behavior:** Verify every visible menu, toolbar button, dialog, file action and transfer control using real local/FTP/FTPS/SFTP fixtures. Block success when a visible element is non-functional or placeholder.
4. **Platform differences:** Windows Setup/MSI/Portable, Linux AppImage/DEB/RPM and Android APK/AAB must launch; macOS requires its own app-run, security and signing/notarization acceptance rather than claiming the desktop screenshot applies to SwiftUI.
5. **Security and privacy:** No mock credentials, signed URLs, tokens, personal data or server addresses in committed QA evidence. Redact network logs. Respect reduced motion and accessibility contrast.

**Current acceptance:** color tokens and several shell elements already overlap with the reference; there is **no verified 75/75 real installed-screen 1:1 parity** and no proof that all illustrated features are implemented. Existing QA outputs must stay identified as actual captures, and these reference concepts must stay identified as design inputs.

## Reference archive re-audit — v0.95.17

The supplied `GhostFTP-Premium-Brand-UI(7).zip` contains `manifest.json`, a static reference renderer, 30 Windows PNGs, 29 Linux PNGs and 16 Android PNGs. Windows/Linux PNGs are 1290×852 pixels and Android PNGs 480×960 pixels. Its own README explicitly warns that paths, accounts, statistics and transfers are illustrative mockups, **not** a functioning application. The working Windows/Android/Linux UI must be assessed against these files at the exact native build SHA; automated screenshot existence and capture checks do not establish all 75 pixel comparisons. macOS has no supplied screen reference in this archive.

Release 0.95.17 improves real backend file-operation QA and the Android SAF recovery click path, but this is not all-screen visual sign-off.
