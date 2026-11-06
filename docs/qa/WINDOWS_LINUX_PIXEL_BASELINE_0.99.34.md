# Ghost FTP — measured Windows and Linux UI baseline for 0.99.34

**Design inputs:** user-provided `GhostFTP-Premium-Brand-UI(20261102-200213).zip`, `screens/windows/*.png` (30 concepts) and `screens/linux/*.png` (29 concepts).

**Runtime captures:** genuine v0.99.33 Windows WebView2 and Linux WebKitGTK native QA ZIP artifacts from [PR #262](https://github.com/bren-wp/Ghost-FTP/pull/262) and exact-source native CI. Comparison against each same-platform 1290×852 concept is strict 8-bit RGB (any channel mismatch counts as a pixel difference), no rescaling, fake data, or image generation.

| Screen | Linux changed RGB pixels (%) | Linux mean RGB absolute error (0–255) | Windows changed RGB pixels (%) | Windows mean RGB absolute error |
| --- | ---: | ---: | ---: | ---: |
| Files | 97.40 | 14.88 | 97.72 | 15.33 |
| Sites | 89.21 | 13.40 | 93.72 | 13.58 |
| New Connection | 99.99 | 14.39 | 99.99 | 14.78 |
| Transfers | 96.33 | 14.23 | 96.90 | 14.45 |
| Settings | 96.33 | 12.94 | 96.90 | 13.17 |
| Sync | 96.41 | 14.23 | 96.94 | 14.45 |
| Help & About | 95.97 | 13.37 | 96.55 | 13.31 |

**Interpretation:** this is **not** a feature failure percentage. Fonts, platform rasterization, locale (English app vs Croatian concept), gradients and **real disconnected / empty remote content** differ. Pure pixel equality with a mock illustration would require dishonestly fabricating connections/transfers, which is forbidden. The current exact-1:1 design request remains **unverified**.

**Acceptance sequence:** Splash/brand → Files/home → Sites → New Connection/Quick Connect → Transfers/Failed/Queue → Sync → Settings → dialogs/terminal/agent/cloud screens. Capture real native Windows and Linux views at 1290×852, 480×600 and minimum display sizes; measure typography, layout, colors, dialog placement, tab order, focus, icons, empty/error states and connected-server content based on real test endpoints. No 1:1 certification is claimed for 0.99.34 until these tests and comparisons are completed.
