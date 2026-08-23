# GitHub releases

ByFTP releases are produced by the repository release workflow. The workflow reads root `VERSION`; no platform maintains a second current-version constant.

Windows Setup/Portable artifacts, Linux packages, the macOS package, checksums and release metadata resolve to the same canonical version. Starting with 1.1.0 the workflow also requires the Android source module to pass its static audit, JUnit tests, Android lint and APK compilation before publication may proceed.

The Android debug APK produced by CI/release validation is **not** a public production asset. Android distribution requires a stable private signing identity managed outside this repository. The workflow must not fabricate a signing identity or publish a debug-signed APK as production software.

Published releases are immutable; corrections require a new semantic release rather than silently replacing existing artifacts.
