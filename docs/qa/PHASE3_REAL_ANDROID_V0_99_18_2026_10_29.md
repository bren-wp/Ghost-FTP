# Phase 3: actual Android v0.99.18 screenshots vs 16-screen v8/v10 concept

## Reproducible source evidence

- Real running debug Activity View-tree captures: GitHub Actions [v0.99.18 main Android run 40786837838](https://github.com/bren-wp/Ghost-FTP/actions/runs/40786837838), artifact **12859837223** (`GhostFTP-Android-Premium-Visual-QA-480x960`). Instrumentation runs on a 480×960 at 160dpi isolated emulator, retains production `FLAG_SECURE`; capturing is restricted to test-only, credential-free workspace content.
- Downloaded ZIP SHA-256: `18a6619cfe2f89d4eac678d5934e640433eada7f1097f8c78905bca676007fc8`.
- Reference ZIP: user-supplied `GhostFTP-Premium-Brand-UI(10).zip`, SHA-256 `1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877`, byte-identical to canonical (8) reference.
- All listed pairs have matching decoded dimensions **480×960**. Method: unpremultiplied `RGBA` comparison; a pixel differs when any of its 4 channels differs from the reference coordinate. `mean_rgb_abs` averages the three RGB absolute differences on the 0–255 scale.

## Comparable reference pairs

| Reference concept | Real debug View screenshot | Different RGBA pixels / 460,800 | Fraction | mean_rgb_abs | Real PNG SHA-256 prefix |
| --- | --- | ---: | ---: | ---: | --- |
| `android/files` | `files.png` | 460,800 | 100.00% | 15.57 | `2194392afc703bf69312` |
| `android/sites` | `sites.png` | 460,793 | 100.00% | 15.92 | `376ba98fdeead6cb170a` |
| `android/transfers` | `transfers.png` | 460,800 | 100.00% | 18.72 | `2e5ceb64fdbf45dab04b` |
| `android/sync` | `sync.png` | 460,799 | 100.00% | 22.44 | `02456c2393decc9d57a7` |
| `android/settings` | `settings.png` | 460,800 | 100.00% | 14.25 | `f291881ec865cba37da1` |
| `android/about` | `about.png` | 460,800 | 100.00% | 15.07 | `9ced8e2ee33373614b04` |
| `android/transfer-detail` | `transfer-details.png` | 460,800 | 100.00% | 16.42 | `c5a2b1b5be4b8a905f81` |

**Interpretation:** These are different coordinate-by-coordinate pixels, not a claim of 100% defects. The Android debug View renderer deliberately excludes untrusted system chrome, captures empty/disconnected state and does not fabricate the reference's three demo server profiles. The design reference depicts device status/system bars and illustrative nonempty application state. The icon rail/top bar, labels, accent states and use of space also differ visibly and require work on the genuine UI.

## First identified and corrected UI gap in 0.99.19 source

On the v0.99.18 real `Sites` capture, the only creation control was a small top-right `+`; the concept has a prominent full-width new-connection action. The 0.99.19 source now adds a real full-width Create Connection CTA, separately distinguishes *zero saved profiles* from *no search matches*, and tests that tapping the CTA enters the genuine site editor. This is an incremental real-state improvement; there is **no post-change installed screenshot acceptance** yet.

## Remaining coverage and hard gates

- The Android QA ZIP contains 13 canonical-size real debug views and responsive captures, but several views (`cloud`, `drawer`, `splash`, `site-editor`, `search`, `transfer-details`, `bookmarks`) do not map one-to-one to the 16 exact reference names/states; they cannot be arbitrarily relabeled to claim completion.
- `android/connect`, `android/new-folder`, `android/rename`, `android/delete`, `android/upload`, `android/download`, `android/host-key`, `android/activity`, and `android/disconnected` still require real appropriately authenticated or genuinely disconnected interactive capture. Never create a fake connected session.
- With earlier v0.99.17 QA evidence: **7 Windows + 1 Linux + 7 Android = 15/75** *reference-compatible actual captures*, from **different released builds** and at least one state mismatch. This is not a single coherent 75-screen acceptance run.
- **0/75 are independently human accepted.** The evidence manifest `docs/qa/PREMIUM_REFERENCE_EVIDENCE.json` remains unverified; do not mark green without build SHA, real screenshot provenance, keyboard/accessibility/interactions, and a signed-off review.
