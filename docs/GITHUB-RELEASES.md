# Ghost FTP GitHub Releases

Ghost FTP **0.0.8** is the current release target. The user-requested 0.0.8 publication uses the explicit **no-secret distribution** workflow. The separate protected production-signing workflow remains available and fail-closed, but is not required for this 0.0.8 download.

## Release identity

```text
Tag: ghostftp-v0.0.8
Title: Ghost FTP 0.0.8
Prerelease: false
```

Root `VERSION` is authoritative. Major version zero does not automatically make a Ghost FTP release a GitHub prerelease.

## Release trigger modes

The protected production workflow remains `.github/workflows/release.yml` and still requires trusted platform signing identities. The requested no-secret distribution is implemented separately in `.github/workflows/release-no-key.yml`; it never reads repository secrets and records the resulting trust state in `BUILD-METADATA.txt` and `RELEASE-NOTES.txt`.

## Canonical release trigger

A push to `main`, including a `VERSION` change, **does not publish a release directly**. Canonical `.github/workflows/release.yml` is `workflow_dispatch`-only. Canonical release branches use:

```text
release/ghostftp-vX.Y.Z
```

For 0.0.8 the branch is `release/ghostftp-v0.0.8`. The branch trigger accepts it only when its semantic version equals root `VERSION` and its SHA equals exact current `main`.

The trigger snapshots existing workflow runs, dispatches `Publish Ghost FTP`, waits for the exact new run to finish successfully, and only then dispatches and verifies release retention. A successful dispatch request is not publication evidence.

## 0.0.8 public files

Ghost FTP 0.0.8 publishes **14 platform artifacts / 17 public files**.

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

Android:

```text
Ghost-FTP-0.0.8-Android.apk
```

Browser helper packages:

```text
Ghost-FTP-0.0.8-Chrome-Extension.zip
Ghost-FTP-0.0.8-Edge-Extension.zip
Ghost-FTP-0.0.8-Firefox-Extension.zip
Ghost-FTP-0.0.8-Opera-Extension.zip
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

macOS is published in the no-secret 0.0.8 distribution as `Ghost-FTP-0.0.8-macOS.app.zip`, an ad-hoc signed universal validation app. It is not Developer ID signed and not Apple notarized.

## Exact-head transaction

Before publication, `release.yml` requires its source SHA to remain exact current `main`. A moved `main`, pre-existing tag or pre-existing release fails closed rather than rewriting release identity.

After successful publication, release readback verifies the **17-file** asset count and exact tag/main identity. It must preserve the immutable published `ghostftp-v0.0.7` release/tag plus the current release/tag, while removing only other superseded Ghost FTP releases/tags, obsolete canonical release branches and obsolete GHCR versions. An existing 0.0.7 GHCR package is preserved when present. `main` history is never rewritten.

## Windows trust state

The protected production workflow still requires trusted Authenticode. The requested no-secret 0.0.8 distribution intentionally publishes the same verified universal Setup/Portable bytes **without Authenticode** and records `WINDOWS_AUTHENTICODE=unsigned`. Windows may therefore show an Unknown Publisher/SmartScreen warning. This is disclosed rather than disguised as production signing.

## Android trust state

`Ghost-FTP-0.0.8-Android.apk` in the no-secret distribution is the installable **release APK signed with a temporary one-run compatibility certificate** produced without a protected publisher secret. The workflow verifies the APK signature and records its certificate SHA-256 fingerprint, but does **not** claim that fingerprint is a long-lived production publisher identity. Android SFTP remains hidden until strict maintained host-key verification exists.

## Browser helper gate

Chrome, Edge, Firefox and Opera packages are rebuilt deterministically from the 0.0.8 source tree under `extensions/`. Official manifests remain zero-permission/zero-host-permission and publication preserves the sanitized `ghostftp://connect` browser-to-desktop handoff on supported Windows installs without adding a remote service or automatic update mechanism.

## Linux universal-distribution gate

`linux/BUILD-DISTROS.sh` builds one Installer and one Portable bundle per Debian, Ubuntu and Fedora. Every bundle carries amd64, arm64 and i386 payloads and chooses the native payload locally.

Native installer/runtime/GUI coverage is maintained on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. ARM64/i386 remain build/package verified unless native execution evidence is separately available.

## Artifact allow-list

`BUILD-METADATA.txt` records at least:

```text
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
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox,Opera
BROWSER_DESKTOP_HANDOFF=sanitized-ghostftp-connect-no-autoconnect
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

The sorted remote GitHub Release asset set must match the exact 17-file allow-list on publication readback. `Prerelease: false` remains part of the current-channel contract.

## What counts as release evidence

For the no-secret 0.0.8 distribution, release evidence is the exact-main build, successful security/privacy/tests, explicit unsigned/debug/ad-hoc trust metadata, SHA-256 manifest and exact 17-file GitHub Release readback. The stricter protected production-signing workflow remains a separate higher-trust distribution path.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Versioning](VERSIONING.md).
