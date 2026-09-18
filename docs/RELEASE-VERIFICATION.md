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
