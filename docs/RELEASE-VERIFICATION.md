# Release verification

`VERSION` is the only canonical ByFTP product-version source. Build scripts inject it into binaries and package metadata.

Verify release files with SHA-256 before redistribution. On Windows use `Get-FileHash`; on Linux use `sha256sum`; on macOS use `shasum -a 256`.

Release automation must validate expected filenames, architecture and version metadata. Publish artifacts produced by the gated workflow rather than rebuilding unverified files manually.
