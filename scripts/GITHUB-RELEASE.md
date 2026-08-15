# GitHub release pipeline

The repository publishes official Windows distribution artifacts through `.github/workflows/release.yml`.

For version 2.12.0 the pipeline:

1. checks out the exact `main` source revision,
2. uses the production Go toolchain declared by the repository,
3. runs the ByFTP Windows build and verification pipeline,
4. produces Portable, Setup and Uninstaller x64 binaries,
5. publishes SHA-256 and verification files with the GitHub Release,
6. packages the same verified Windows distribution as `ByFTP.Windows` for GitHub Packages.

GitHub Releases are the recommended download channel for normal Windows users. GitHub Packages is an additional distribution/archive channel for package-oriented workflows.

Public production binaries should be Authenticode-signed with Brendigo's real code-signing identity before broad public distribution. The repository never substitutes a fake publisher certificate.
