# Packages

Ghost FTP release packaging targets the active native applications **Windows, Linux and Android** plus optional browser-helper ZIPs.

## Application artifacts

- Windows Setup
- Windows Portable
- Linux Debian installer + portable
- Linux Ubuntu installer + portable
- Linux Fedora installer + portable
- Android APK

## Companion packages

Optional local helpers are packaged for Chrome, Edge, Firefox and Opera. They are not independent FTP protocol engines.

## Release metadata

Current production metadata uses:

```text
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID
```

The public bundle contains **13 platform artifacts / 16 public files** when all current application/browser artifacts plus metadata files are included.

macOS artifacts are retired and must not appear in current release assembly or metadata.

## Signing boundary

- Windows official publication requires the configured trusted Authenticode boundary.
- Android official publication requires the protected publisher identity and exact signer fingerprint verification.
- Linux packages are verified by exact-source build/lifecycle checks and release checksums.

See [Signing](SIGNING.md) and [Release verification](RELEASE-VERIFICATION.md).

## Windows architecture evidence

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The public Windows Setup and Portable launchers carry x64, x86 and ARM64 native payloads. CI verifies the ARM64 payload structure and PE identity, but does not claim native ARM64 runtime execution; that limitation is recorded explicitly by `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci`.
