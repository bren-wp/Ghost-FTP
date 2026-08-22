# GitHub release process

ByFTP releases are produced by the repository release workflow, not by manually uploading arbitrary binaries.

## Preconditions

The canonical `VERSION` must use `major.minor.patch`. README, CHANGELOG and platform metadata must agree. A published `v<version>` tag is immutable. If a tag already exists, production-sensitive changes require the next version.

## Quality gates

The publisher depends on successful quality, Windows, Linux and macOS jobs. These run localization/version/docs/security/privacy/release audits, Python regressions, Go unit/race/vet checks and platform packaging.

## Public artifacts

The publisher expects an exact set of Windows x64/x86 executables and ZIPs, Linux DEBs and the macOS Universal PKG. It rejects missing or extra platform artifacts before generating common SHA-256 metadata.

The GitHub release publisher validates existing asset identity/digests and fails closed on mismatches rather than silently replacing a different release file.

## GitHub Windows package

`ByFTP.Windows` is built only after release assets pass staging. The workflow requires exactly one NuGet package with the expected versioned filename before pushing it to GitHub Packages.

## Signing

Authenticode Verified Publisher and Apple Developer ID/notarization require real organization credentials. CI must not claim those properties when signing credentials are not configured.