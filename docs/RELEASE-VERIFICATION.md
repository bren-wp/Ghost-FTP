# Release verification

Ghost FTP release verification covers the active Windows, Linux and Android applications plus optional browser-helper packages.

## Required boundaries

1. release source SHA is exact and immutable;
2. root `VERSION` matches the intended tag;
3. Windows signing state matches the publication mode;
4. Android signer identity/fingerprint matches the publication mode;
5. Linux packages pass exact-source build and install lifecycle checks;
6. browser helper packages match the deterministic package contract;
7. SHA-256 metadata verifies every public file;
8. the public file set exactly matches the current allow-list.

## Current release metadata

```text
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```

## Authentic UI evidence

UI evidence must be generated from the real application at the exact tested source SHA. Reference artwork and generated mockups do not prove runtime behavior.

## macOS retirement

Current release verification contains no macOS application, Developer ID, notarization or macOS asset gate. Historical macOS artifacts are outside the active product contract.


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. The metadata value above is intentionally explicit: current hosted CI does not claim native Windows-on-ARM runtime execution evidence.


## Current public release shape

The active Windows/Linux/Android plus browser-helper publication contract contains **13 platform artifacts / 16 public files** after macOS retirement. Historical published releases remain immutable.

```text
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```
