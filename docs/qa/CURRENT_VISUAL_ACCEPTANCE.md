# Ghost FTP by Brendigo — Current Windows/Linux visual acceptance

**Status: not yet verified for every screen.** The user's target is 1:1 alignment with approved references. A release or pull request must not claim that target merely because CSS checks, builds, or individual snapshots pass.

## Acceptance procedure

1. Freeze the **approved** visual reference for each Windows/Linux screen, dialog and interactive state, including screen identity and source revision. Concepts/mockups must be labeled as concepts; they are not evidence of application functionality.
2. Record application commit, exact executable/build artifact, OS image, native WebView/WebKit version, theme, locale, window client dimensions, system scaling/DPI, font rendering settings and sanitized test data.
3. Launch an actual signed or unsigned QA build on each native OS. Exercise connection, disconnection, upload/download, cancel/retry, overwrite, failure, authentication, keyboard and narrow-window states—not only static first-paint imagery.
4. Capture the real application with user secrets removed **before** sharing. Do not synthesize, repaint, crop to hide defects or silently rename historical captures as current-version screenshots.
5. Run `python tools/visual/pixel_compare.py --reference approved.png --actual actual.png --require-exact` at identical image dimensions for each exact reference. For multi-screen concept sets, run `python tools/qa/compare-screen-references.py --references supplied.zip --captures actual-captures --output visual-report` and review `results.json`, `results.csv` and highlighted differences.
6. Record an explicit pass/fail/not-tested result per screen and interaction. A missing reference, unrecorded native capture, font/DPI mismatch or failed interaction is **not verified**, not a pass. Check Windows and Linux separately.

## Important boundaries

Pixel-exactness is meaningful only for a controlled rendering environment with a genuinely approved reference. Differences from dynamic timestamps, filesystem entries, font rasterizers and OS chrome must be identified rather than hidden behind invented percentages. Visual parity does not establish accessibility, transfer integrity, encryption or release readiness. All required security, protocol and installer gates remain independent.

The current tracked `docs/qa/PREMIUM_REFERENCE_EVIDENCE.json` contains unverified screen entries. Earlier README screenshots have documented native-build provenance but must not be described as new-version captures. Re-capture before a current-version screenshot claim.

See [visual tools](../../tools/visual/pixel_compare.py), [existing reference inventory](PREMIUM_REFERENCE_EVIDENCE.json), [QA overview](README.md) and [Stable release requirements](../release/STABLE_RELEASE_POLICY.md).
