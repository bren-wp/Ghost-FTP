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

## Authentic UI evidence

Runtime screenshot workflows must launch the real application and capture real UI state from the exact tested source SHA. Generated artwork, design references or mockups are not accepted as release evidence.

## macOS

macOS-specific workflows and tests are retired and removed.

## Merge rule

Required checks and evidence must apply to the exact PR head being merged.
