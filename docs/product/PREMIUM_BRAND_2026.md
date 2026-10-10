# Ghost FTP premium UI — implementation and acceptance contract

Canonical mark: exact `assets/branding/ghostftp-app-icon.svg` geometry and corresponding React/Android vector implementations.

Approved horizontal lockup: `ghostftp-desktop/branding/ghostftp-logo.svg` now follows the ZIP's exact 1280 × 330 wordmark, background, ghost mark, palette, spacing and uppercase tagline; its SVG title/description provide accessible naming. The product README uses this asset directly.

Source: user-provided `GhostFTP-Premium-Brand-UI(2).zip` (identical 75-screen reference family, inspected 9 October 2026).

## Reference contract

- Screens are **visual concepts**, not real product screenshots or operational examples.
- 30 Windows and 29 Linux mockups share a 1290 × 852 desktop frame and layout.
- 16 Android mockups use a 480 × 960 reference frame with compact navigation rail.
- No macOS mockups were provided; macOS adopts the common palette, spacing hierarchy, brand icon and component semantics with platform-correct SwiftUI controls.
- Mockup data, sample server names, metrics and fake transfer rows must never be seeded in the product.
- Never display screenshot PNGs as clickable UI backgrounds or infer backend support from a pictured button.

## Canonical premium tokens

| Purpose | Hex |
| --- | --- |
| Canvas | `#070E1A` |
| Panel | `#0D192B` |
| Raised panel | `#111F34` |
| Navy | `#0B1E36` |
| Slate blue | `#132D52` |
| Action/accent | `#38ABFF` |
| Accent highlight | `#7EDEFF` |
| Primary text | `#EAF6FF` |
| Success | `#26DC8F` |
| Error | `#FF7284` |

Inter, Segoe UI and Noto Sans are preferred with platform system fallbacks. No font files are bundled.

## Real implementation

- Windows/Linux: existing React/Tauri components for navigation, file panes, transfers, dialogs and settings; shared dark semantic tokens and GhostMark. Linux preserves native window-manager differences.
  - Quick tools from the approved reference sidebar are connected to the existing, functional SFTP Terminal, command palette and local duplicate scanner. Terminal is disabled without an active SFTP session.
- Android: existing native Kotlin UI with shared semantic colors and vector launcher mark. Standard-width phones now dock the 64 dp rail rather than covering the file workspace with a menu; devices under 360 dp retain an accessible overlay. Touch-target and accessibility checks stay mandatory.
- macOS: existing SwiftUI workspace shell with shared palette/tint. Light appearance remains independently supported.
- Secrets, permission prompts and security warnings must retain the existing platform policies.

## 0.30.15 implementation traceability

- The canonical `docs/qa/premium-75-screen-contract.json` records every visual reference and its live application source. A GitHub quality job verifies the 30 Windows + 29 Linux + 16 Android set without claiming visual acceptance.
- Android now uses the identical existing launcher/vector mark in its real header and navigation, not a separate hand-painted approximation. Android 12+ launch screens reuse that approved mark.
- At 1290px the desktop dark primary sidebar uses the approved 214px reference width; below 1100px the earlier compact layout remains.
- macOS receives its own functional SwiftUI spacing and branded connection action rather than a static Android/Windows screenshot; its minimum window size now supports 840×560.

## 0.30.16 live color-surface correction

- Desktop Windows and Linux navigation now consumes the same theme-aware `--bg-panel` RGB token as the rest of the current theme, removing the obsolete hard-coded sidebar shade. The dark theme resolves this to `#0D192B`.
- Android row/input surfaces reference their single existing `Brand.panel` / `Brand.background` constants rather than independently duplicating RGB triples.
- macOS SwiftUI sidebar now uses the canonical `#0D192B` exact RGB components. This matches the dark-mode color reference without forcing platform-specific light mode to use a dark background.
- GitHub quality gates assert all three implementation contracts, while screenshots/interaction comparisons remain necessary before accepting individual reference screens.
- No credential handling, transfer implementation or transport security settings were changed by this color-only refinement.

## Release gating / not yet completed

- Automated builds and functional protocol gates **do not** prove pixel-by-pixel parity; acceptance requires screenshot comparison at reference sizes on each platform.
- All 75 screens must be mapped to real working components before claiming 1:1 visual/functional completion. Current progress is design-token/logo alignment and the mobile navigation geometry, not a complete screenshot-by-screenshot acceptance.
- macOS FTPS/SFTP file operations remain blocked until trusted TLS certificate/hostname and SSH host-key verification are implemented. macOS Preview is not notarized.
- Future screen-level refinements must be scoped to real components and ship only after exact-head CI and package validation.

## 0.30.18 — live Help/About structure

- **Windows and Linux Help screen:** the live React workspace now has a reference-shaped official identity/fact card and right-side support panel; real version/build values, approved Ghost mark and tagline, working update action, and source-controlled help links.
- **Android About screen:** the Kotlin workspace now has a centered official icon, actual installed version, localized security/privacy facts, and accessible resource cards within the existing ScrollView.
- **macOS:** adopts the same semantic product/resources layout through SwiftUI; the ZIP has no macOS reference screenshot.
- **Visual acceptance:** All 75 mockup references remain **pending pixel-by-pixel acceptance**. No installed build has yet established 1:1 matching for every reference screen; no synthetic servers, files or progress indicators are seeded.

## 0.30.19 — live dual-pane file manager interaction parity

- The default desktop Files UI remains two real local/remote panes, closer to the reference than any screenshot-like mockup.
- A connected user can now click Upload without selecting a local entry. The OS file picker supplies actual files for the current real server path. When files are selected, one-click transfer remains available.
- The single-pane browsing mode also supports the same picker when local view has no selection, without changing user layout preferences.
- When queued/transferring/paused work drains to zero, both live FilePane listings refresh, preventing stale newly transferred files.
- No prepopulated demonstration paths, servers, progress or fake transfer rows have been introduced. All 75 reference screenshots still require installed-build visual acceptance.

## 0.30.20 — verified Windows screenshot refresh and release-safe audit

- Seven existing README-linked PNG images were replaced with 1290×852 screenshots **captured from a running Windows build** at v0.30.19 in GitHub Actions QA run 37879036243. Each is pinned by the exact published-run evidence and SHA256 in .github/scripts/check-release-images.mjs.
- Main README presentation (image positions, HTML table/gallery and other page structure) was not redesigned. Only routine active-version metadata needs updating to 0.30.20.
- React/Tauri Windows/Linux, Android Kotlin, macOS SwiftUI and shared protocol engines retained production QA gates. New privacy/security/performance fixes are documented in CODE.md and SECURITY.md.
- **1:1 acceptance is still pending** for all 75 supplied reference mockups; replacing old READMEs with real UI photographs does not automatically make the running UI pixel-identical to a concept image.

## 0.94.0 — approved ZIP revalidation and live UI alignment

The user supplied `GhostFTP-Premium-Brand-UI(4).zip` as the canonical visual reference. Its `manifest.json` confirms **30 Windows + 29 Linux + 16 Android** separate PNG designs. This is a design-only reference; illustrative file names, transfer states, connection passwords and progress indicators are not application fixtures.

The following three vector files already present in `assets/branding` were compared byte-for-byte to the user's ZIP and their approved Git-blob SHA-1 values match the enforcement script: `ghostftp-adaptive-foreground.svg`, `ghostftp-logo-light.svg` and `ghostftp-symbol-monochrome.svg`. Existing official branding assets remain the source of launcher and app header identity; no new fabricated screenshot is introduced.

- **Single token registry:** `docs/branding/premium-ui-tokens.json` defines the canonical palette and reference geometry. `check-premium-ui-tokens.mjs` verifies real desktop CSS, Android Kotlin colors/vector resources and macOS SwiftUI palettes in CI.
- **Windows + Linux React/Tauri:** default dark semantic colors now resolve to reference `#070E1A` canvas, `#0D192B` panels, `#111F34` raised surfaces, `#38ABFF` action state, `#26DC8F` success and `#FF7284` danger. Real sidebar interaction and focus states use reference gradient/tint rather than painting the mockup PNG.
- **Android:** on standard-width phones (360–599 dp) the docked navigation now uses **64dp**, line-art vector icons, localized visible labels, accessible full labels and a 20dp icon size. The existing under-360dp overlay remains so short screens do not lose file controls. An instrumented emulator test asserts all seven branded navigation destinations are interactive, localized and have icons.
- **macOS:** the existing SwiftUI workspace shell uses aligned background, panel, raised, accent, cyan, success, danger and selected-navigation colors. Cloud cards use reference-derived accent borders and working cancellation controls. There are **no supplied macOS mockups**; exact pixel parity for that platform cannot be asserted.
- **CI contract:** `ghostftp-quality.yml` now runs the token regression and existing 75-screen traceability checks. Real application captures and functional click-by-click acceptance are still required.

### Mandatory 1:1 acceptance matrix

| Platform | Reference screens | Implementation evidence | Pixel/interaction acceptance |
| --- | ---: | --- | --- |
| Windows | 30 | Live React/Tauri source + shared palette + reference vectors | **Pending:** running Windows screenshots and full click QA |
| Linux | 29 | Same React/Tauri source with Linux packaging | **Pending:** running Linux screenshots and full click QA |
| Android | 16 | Real Kotlin UI, 64dp icon rail, emulator smoke/unit tests | **Pending:** 480×960 screenshots, modal and device QA |
| macOS | No supplied reference screens | SwiftUI native controls and shared palette | **Pending:** approved Mac reference or separate platform styling acceptance |

A complete visual pass requires actual screenshots from launched builds captured at the defined resolution with production components (not illustration PNGs), full-size and narrow-window overflow/accessibility assessment, and verification that each enabled action executes the corresponding real operation. CI palette checks and a mapping registry alone are not a 1:1 pixel certification. No live data or successful network transfers may be invented to match concept art.

## 0.95.0 — actual protected Android render evidence

The initial single Android ADB screenshot was effectively black due to the existing `FLAG_SECURE` policy. **Do not disable this privacy control to obtain marketing images.** The new `PremiumVisualReferenceTest.kt` renders only a credential-free debug Activity view tree at the 480px reference width, after failing closed on unexpected form values. An emulator run at 480×960/160dpi captures seven authentic idle workspaces with exact per-file SHA-256 hashes to a dedicated internal QA artifact. No concept PNG is used as an interactive background, and no fake connection/file/transfer state is inserted.

This is a capture *foundation*, not 16 completed visual comparisons; all 75 concept entries remain `visual_acceptance: pending`. The Android 64dp rail and 58dp header receive real laid-out geometry assertions. Desktop real screenshot comparison and separate macOS platform acceptance remain necessary.

## 0.95.0 — genuine Windows/Linux Files information architecture (Issue #131)

The Files workspace now includes a visible page heading, working **New connection** button, reference-style inline **Host / Username / Port / Protocol** form and **Review & connect** action. This form does **not** request or save a password: it hands non-secret input into the pre-existing validated connection dialog, which owns credential, host-key and TLS decisions. The real current connection state is displayed rather than a fabricated demo host.

Beneath the actual local/remote file panes, a constrained responsive strip shows up to three **real active/queued/paused transfers** from the backend store, genuine byte-count progress, a truthful empty state and an actionable link to Transfer Center. No mock transfer rows are seeded. The desktop CI has a source-level action/data-binding contract at `ghostftp-desktop/scripts/check-premium-files-reference.mjs` plus the existing Windows native QA captures; actual 1290×852 pixel comparison and keyboard/click QA remain pending. Issue [#131](https://github.com/bren-wp/Ghost-FTP/issues/131) is not accepted until Windows/Linux running-build screenshots are reviewed with responsive states.


## 0.95.0 desktop audit provenance and Linux capture

The 0.95.0 Windows native QA screenshots for Files, Sites, Transfers, New Connection, Settings and Help/About have been compared against the six corresponding premium concept images. The [visual comparison audit](../qa/0.95.0-DESKTOP-VISUAL-COMPARISON.md) records the specific differences. A real Linux Tauri executable was verified through the build and GUI smoke; a separate 1290×852 Xvfb screenshot step is now under CI test and **cannot** count as Linux pixel acceptance until the actual PNG is examined. All 75 reference screen statuses remain pending.
