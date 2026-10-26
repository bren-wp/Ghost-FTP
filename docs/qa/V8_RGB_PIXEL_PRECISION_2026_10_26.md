# v8 reference pixel-count precision audit (26 October 2026)

The v8 installed-application screenshot comparison tool used `diff.convert("L").histogram()[0]` to count changed pixels. This wrongly reports a pixel with an RGB delta of only 1 in a single channel as unchanged after grayscale integer rounding. Example: reference (0,0,0), actual (1,0,0) yields **zero** changed pixels by the old logic, despite `exact=False`.

The corrected implementation forms a per-pixel maximum of each RGB difference channel and counts only pixels with all three difference channels equal to zero as unchanged. Mean RGB absolute difference, exact-equality classification, image-size mismatch rejection, checksum provenance and the v8 reference archive SHA gate are unchanged.

The new `tools/qa/test_compare_reference_v8.py` regression covers red/green/blue one-unit deltas, exact identical screenshots and size mismatches. Locally, the per-channel correction passed a synthetic PNG arithmetic smoke test. The full repository integration suite has **not** been run for this separate source branch.

No macOS concept images are supplied and no Windows/Linux/Android screen is automatically labeled 1:1 by this correction. The tool still requires real installed-app captures at precisely the reference dimensions.
