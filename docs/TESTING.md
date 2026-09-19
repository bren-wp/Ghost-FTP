# Testing

Ghost FTP testing covers the active Windows, Linux and Android applications plus shared engine, security and release behavior.

## Shared gates

- Go formatting, tests and vet
- security/privacy audits
- localization/documentation contracts
- CodeQL
- Govulncheck
- release/version integrity

## Windows

Validation includes universal packaging, native runtime behavior, keyboard/modal tests and authentic runtime screenshots.

## Linux

Validation includes Debian/Ubuntu/Fedora package generation, install lifecycle checks, native UI contracts and authentic runtime screenshots.

## Android

Validation includes native build/lint/signing contracts, navigation/UI contracts and authentic emulator screenshots.

## File filtering and bounded search

The current-folder filter and sorting regression contract verifies non-destructive filtering over already-loaded entries, including filtering and subsequent sorting. It is deliberately separate from bounded recursive search so the filter cannot silently trigger filesystem or network traversal.

The bounded recursive search regression contract validates the explicit recursive-search path, its bounds and its cancellation/error behavior independently of the current-folder filter.

## Authentic UI evidence

Runtime screenshot workflows must launch the real application and capture real UI state from the exact tested source SHA. Generated artwork, design references or mockups are not accepted as release evidence.

## macOS

macOS-specific workflows and tests are retired and removed.

## Merge rule

Required checks and evidence must apply to the exact PR head being merged.

## Windows architecture evidence

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The public Windows Setup and Portable launchers carry x64, x86 and ARM64 native payloads. CI verifies the ARM64 payload structure and PE identity, but does not claim native ARM64 runtime execution; that limitation is recorded explicitly by `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci`.
