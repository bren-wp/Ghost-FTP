# GitHub releases

ByFTP releases are produced by the repository release workflow. The workflow reads `VERSION`; it must not maintain a second current-version constant.

Windows setup/portable artifacts, Linux packages, the macOS package, checksums and release metadata must resolve to the same canonical version. Published releases are immutable; corrections require a new semantic release rather than silently replacing existing artifacts.
