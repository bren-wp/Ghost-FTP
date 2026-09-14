# Ghost FTP versioning

Ghost FTP uses semantic versioning with root `VERSION` as the authoritative production version source.

Current source candidate: **0.0.6**.

## Version format

```text
MAJOR.MINOR.PATCH
```

Production tags use `ghostftp-vMAJOR.MINOR.PATCH`.

Current identity:

```text
VERSION=0.0.6
TAG=ghostftp-v0.0.6
CHANNEL=Current
PRERELEASE=false
```

## Public numbering line

The public numbering starts at `0.0.1`; `0.0.0` is reserved and must never be published. For Ghost FTP, **major version `0` does not imply prerelease**. The current 0.0.x line uses `prerelease=false` unless an explicit reviewed policy changes that rule.

## Platform and version boundaries

Current public application platforms:

```text
WINDOWS,LINUX,ANDROID
```

Public companion packages:

```text
BROWSER_HELPER=Chrome,Edge,Firefox
```

Active native source platforms:

```text
WINDOWS,LINUX,ANDROID,MACOS
```

Android 0.0.6 is public only through the protected production-signing path. Android SFTP remains hidden until strict maintained host-key verification exists. Browser-helper publication does not make it an application platform and does not add desktop launch/handoff. macOS remains a development/source platform until real Developer ID signing + notarization succeeds.

## Latest-only public release retention

Ghost FTP intentionally keeps only the **latest public version** visible in release infrastructure. After a new release passes exact remote readback, `.github/workflows/release-retention.yml` removes superseded Ghost FTP Releases, `ghostftp-v*` tags, superseded canonical release branches and obsolete GHCR package versions while retaining current identities. It never rewrites `main` history.

## Release trigger

A `VERSION` edit or ordinary push to `main` does not publish a release. Canonical release branches use:

```text
release/ghostftp-v<version>
```

For 0.0.6:

```text
release/ghostftp-v0.0.6
```

The branch trigger accepts it only when branch version equals root `VERSION` and branch SHA equals exact current `main`. It dispatches `release.yml`, waits for the exact release run to succeed, then dispatches and waits for retention.

## Current GitHub Release rule

```text
CHANNEL=Current
PRERELEASE=false
```

The 0.0.6 release contains **18 platform artifacts / 21 public files**:

- 2 universal Windows executables;
- 12 canonical Linux Debian/Ubuntu/Fedora/Portable artifacts;
- 1 production-signed Android APK;
- 3 deterministic browser-helper ZIPs;
- 3 metadata/verification files.

Verified bundle:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

The GHCR object is a distribution bundle, not a runtime container.

## Windows packaging/signing identity

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Native x64, x86 and ARM64 staging payloads are internal and embedded in those same two public files. The current metadata contract is:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

Windows Authenticode is a **required public-release trust boundary**. **Absence of the production Authenticode identity is a release failure.** No architecture-specific public EXE aliases or unsigned official fallback are allowed.

## Linux packaging identity

`linux/BUILD-DISTROS.sh` publishes Debian DEBs for `amd64`, `arm64`, `i386`; Ubuntu DEBs for the same architectures; Fedora RPMs for `x86_64`, `aarch64`, `i686`; and Portable tarballs for `amd64`, `arm64`, `i386`.

## Android release and development identities

Public 0.0.6 Android artifact:

```text
Ghost-FTP-0.0.6-Android.apk
```

The release `versionName` equals root `VERSION`. Publication requires the protected production keystore/alias/password credentials and an exact signer-certificate fingerprint match against `GHOSTFTP_ANDROID_SIGNER_SHA256`.

Development artifact:

```text
Ghost-FTP-Android-dev.apk
```

Debug builds add `-dev` and use a separate development application ID. An ephemeral CI signing identity is only pipeline evidence; it is never the production publisher.

## Browser-helper identity

Public 0.0.6 packages:

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
```

Browser packages remain local parser/copy helpers with no supported browser-to-desktop URI/native-messaging handoff.

## macOS development identity

macOS source is bound to root `VERSION`. Its universal development app may be ad-hoc signed for CI/regression use. Public macOS distribution can be claimed only after the real Developer ID signing/notarization path succeeds with protected credentials; development success does not enlarge the public 21-file release.

## Version source integrity

Release/CI validation rejects malformed versions, `0.0.0`, release-branch/source mismatch, non-exact-main release branches, conflicting tags/releases, incomplete asset sets, public Windows architecture leakage, absent/failed Windows signatures, absent/failed Android production signing or signer mismatch, source/main drift, incorrect prerelease flags, missing GHCR exact-version bundle, failed remote readback or failed retention cleanup.

Active release-bound documentation must agree with root `VERSION`, `CHANNEL=Current`, `PRERELEASE=false`, the 18/21 packaging contract, public/source platform boundaries and package identity.

## Changelog and release notes

`CHANGELOG.md` contains a `## <VERSION>` section. `scripts/release_notes.py` extracts the current section and must describe the same release shape and security boundaries as canonical `release.yml`.

Historical versions remain valid only as historical records; they are not current publication instructions.

## 0.0.6 release checklist

The exact candidate must pass:

- `gofmt`, `go test -race ./...` and `go vet ./...`;
- repository/platform/desktop/dependency/version/localization/security/privacy/docs/release audits;
- the full Python regression suite;
- FTP/FTPS/SFTP trust/no-downgrade and filesystem/transfer safeguards;
- Windows profile/file-mutation/Remote Edit/transfer-generation lifecycle contracts;
- queue ordering, filtering/sorting/search/comparison/bookmark/start-directory contracts;
- Windows x64/x86/ARM64 staging, PE/resources, universal bootstrap and two-public-EXE leakage checks;
- `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` until maintained native ARM64 execution evidence exists;
- Linux package metadata/extraction/binary parity plus maintained native distro lifecycle matrix;
- Android source/security/lifecycle/JVM/lint/debug/release build checks and authentic emulator evidence;
- protected Android production signing and exact signer SHA-256 verification;
- deterministic Chrome/Edge/Firefox helper package checks;
- universal macOS development-app validation without claiming public notarization;
- 24-language desktop localization and exact-head cross-platform UI evidence;
- exact-head PR gates and exact post-merge `main` gates;
- exact-main `release/ghostftp-v0.0.6` validation;
- `ghostftp-v0.0.6`, `prerelease=false`, exact **21-file** GitHub Release readback;
- `ghcr.io/bren-wp/ghost-ftp:0.0.6` publication/readback;
- successful latest-only retention cleanup.

## Next release

Only after 0.0.6 publication and retention are completely green may root `VERSION` advance again through a separate reviewed release-prep change.
