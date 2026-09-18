# Ghost FTP release verification

Ghost FTP **0.0.8** is the current release target. The user-requested publication mode is the exact-main **no-secret distribution**; the stricter protected production-signing transaction remains available separately.

The canonical 0.0.8 publication contains **14 platform artifacts / 17 public files**.

## 0.0.8 release identity

```text
VERSION=0.0.8
TAG=ghostftp-v0.0.8
TITLE=Ghost FTP 0.0.8
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
PROTECTED_RELEASE_TAG=ghostftp-v0.0.7
PROTECTED_RELEASE_POLICY=PRESERVE_TAG_RELEASE_AND_EXISTING_PACKAGE
```

The release source must be the exact current `main` commit that passed every required gate.

## Canonical public files

Windows:

```text
Ghost-FTP-0.0.8-Setup.exe
Ghost-FTP-0.0.8-Portable.exe
```

Linux:

```text
Ghost-FTP-0.0.8-Linux-Debian-Installer.run
Ghost-FTP-0.0.8-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.8-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.8-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.8-Linux-Fedora-Installer.run
Ghost-FTP-0.0.8-Linux-Fedora-Portable.tar.gz
```

Android and browser helpers:

```text
Ghost-FTP-0.0.8-Android.apk
Ghost-FTP-0.0.8-Chrome-Extension.zip
Ghost-FTP-0.0.8-Edge-Extension.zip
Ghost-FTP-0.0.8-Firefox-Extension.zip
Ghost-FTP-0.0.8-Opera-Extension.zip
```

Metadata/verification:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

macOS is included as `Ghost-FTP-0.0.8-macOS.app.zip`, an ad-hoc signed universal validation app. It is explicitly not Developer ID signed and not Apple notarized.

## No-secret release dispatch

`.github/workflows/release-no-key.yml` publishes only from the dedicated `release/ghostftp-v0.0.8-no-key` branch and refuses publication unless that branch SHA equals exact current `main`. It reads no repository signing secrets and refuses to overwrite an existing tag or release.

## Canonical protected release dispatch

The canonical branch namespace is `release/ghostftp-vX.Y.Z`; the 0.0.8 release branch is `release/ghostftp-v0.0.8`. It must point to exact fully verified current `main`, and its version must match root `VERSION`.

Canonical `.github/workflows/release.yml` is `workflow_dispatch`-only. The branch trigger validates source/version equality, dispatches canonical `release.yml`, waits for the exact new release run to finish successfully, then dispatches and verifies retention.

A push to `main`, including a `VERSION` change, must never publish a release directly.

## Source verification

Before publication:

1. root `VERSION` equals `0.0.8`;
2. `release/ghostftp-v0.0.8` equals exact current `main`;
3. every exact-head release-prep workflow for the final candidate is successful;
4. every required post-merge push workflow on the exact merge SHA is successful;
5. authentic Windows/Linux/Android runtime evidence is bound to that exact source revision;
6. release quality, Windows, Linux, Android and browser jobs succeed again from fresh source;
7. Windows Setup/Portable are verified as intentionally unsigned and that state is recorded;
8. the Android APK passes signature verification and its temporary compatibility signer fingerprint is recorded without claiming a protected publisher identity;
9. the macOS universal app passes ad-hoc codesign verification without claiming notarization;
10. the release contains exactly the canonical **17-file** set.

## SHA-256 verification

`SHA256.txt` contains a checksum for every public file except itself. A downloaded artifact is accepted for exact-byte integrity only when its local hash matches the manifest entry.

```bash
sha256sum -c SHA256.txt
```

The release workflow additionally performs remote digest readback against the exact locally assembled bundle.

## Windows verification

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=unsigned
```

Architecture-specific staging executables are internal verified inputs and must never appear among public assets. `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` means current CI cross-builds and structurally verifies ARM64 but does not claim native Windows ARM64 execution.

The protected production workflow still has no unsigned fallback. The separate no-secret 0.0.8 workflow intentionally verifies that its Windows downloads do **not** contain an Authenticode signer and records that state.

## Linux verification

Each distro publishes one installer and one portable archive. Every bundle carries amd64, arm64 and i386 payloads and selects the native payload locally.

The verifier checks:

- exactly six Linux public artifacts;
- deterministic archive construction;
- valid amd64/arm64/i386 executable payloads;
- installer and portable payload parity for the host architecture;
- runtime dependency preflight and CA-trust availability;
- actual install, GUI startup and uninstall on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64;
- no stale `.deb`, `.rpm` or architecture-suffixed public Portable artifacts.

ARM64 and i386 are build/package verified unless maintained native execution evidence is separately available.

## Android no-secret signing verification

The public APK is:

```text
Ghost-FTP-0.0.8-Android.apk
```

The no-secret workflow builds the unsigned release APK, generates a temporary one-run compatibility certificate, signs the release APK with Android `apksigner`, verifies it with `apksigner verify --verbose --print-certs`, and records the observed certificate SHA-256 digest in `BUILD-METADATA.txt`. No protected Android publisher secret is read or claimed. A future APK signed with a different key may require uninstall/reinstall.

Android SFTP remains hidden until strict maintained host-key verification/pinning exists and fails closed for unknown or mismatched hosts.

## Browser-helper verification

The browser job validates zero-permission manifests and builds deterministic ZIPs for **Chrome, Edge, Firefox and Opera**. Each archive must be non-empty and pass ZIP integrity verification. Browser publication preserves the sanitized local `ghostftp://connect` handoff on supported Windows installs and does not add a network relay, telemetry backend or automatic update service.

## Build metadata

`BUILD-METADATA.txt` binds release identity to source and records at least:

```text
BRAND=Ghost FTP
VERSION=0.0.8
RELEASE_TAG=ghostftp-v0.0.8
RELEASE_CHANNEL=current
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS,BROWSER_HELPER
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=unsigned
LINUX_DEBIAN_INSTALLER=universal-amd64-arm64-i386
LINUX_DEBIAN_PORTABLE=universal-amd64-arm64-i386
LINUX_UBUNTU_INSTALLER=universal-amd64-arm64-i386
LINUX_UBUNTU_PORTABLE=universal-amd64-arm64-i386
LINUX_FEDORA_INSTALLER=universal-amd64-arm64-i386
LINUX_FEDORA_PORTABLE=universal-amd64-arm64-i386
ANDROID_APK=temporary-compatibility-certificate
ANDROID_SIGNER_SHA256=<verified temporary compatibility signer SHA-256>
ANDROID_SFTP=hidden-until-strict-host-key-verification
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox,Opera
BROWSER_DESKTOP_HANDOFF=sanitized-ghostftp-connect-no-autoconnect
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.8
```

## Authentic runtime evidence

Exact-head UI evidence is source-bound. Maintained workflows capture real Windows, Linux and Android runtime surfaces and assemble a verified evidence bundle containing source SHA, filenames, byte counts and SHA-256 hashes. Mockups, image-generation output and manually composed approximations are not release evidence.

The Windows evidence does not claim native ARM64 execution. For this no-secret release, macOS evidence is limited to the ad-hoc validation build and does not claim Developer ID or notarization.

## Remote release readback

The publish workflow requires the remote GitHub Release asset set to match the exact **17-file** allow-list immediately and after a delay. It requires `prerelease=false` and refuses to rewrite an existing tag/release.

The digest-readback verifier compares GitHub's per-asset SHA-256 digests with the exact source-workflow bundle and checks that `BUILD-METADATA.txt` binds `COMMIT` to the expected source SHA.

## Protected retention verification

The no-secret 0.0.8 workflow does not perform destructive release/package retention. It verifies `ghostftp-v0.0.8` is non-draft/non-prerelease, has **17 assets** and points to exact current `main`. The published `ghostftp-v0.0.7` release/tag is a protected immutable baseline and must remain present and unchanged; an existing 0.0.7 GHCR package is preserved when present. `main` history is never rewritten.

See [GitHub Releases](GITHUB-RELEASES.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).


Verified no-secret macOS asset: `Ghost-FTP-0.0.8-macOS.app.zip`.
