# Testing

Ghost FTP uses exact-head validation for the maintained Windows, Linux and Android product surfaces.

## Core CI

The main CI gate runs:

- Go formatting;
- Go tests with race detection where supported;
- Go vet;
- repository audits;
- platform-contract audits;
- localization audits;
- security and privacy audits;
- documentation and release audits;
- Python regression tests.

## Windows

Windows validation includes:

- universal Setup + Portable builds;
- x64 / x86 / ARM64 payload checks;
- packaging integrity;
- Authenticode pipeline smoke tests;
- modal keyboard/runtime tests;
- authentic runtime screenshots.

## Linux

Linux validation includes:

- Debian / Ubuntu / Fedora package generation;
- amd64 / arm64 / i386 payload checks;
- installer lifecycle tests;
- portable bundle tests;
- authentic X11 runtime screenshots.

## Android

Android validation includes:

- lint;
- debug / validation APK build;
- release-signing contracts;
- installability checks;
- emulator runtime screenshots;
- mobile navigation/readability contracts.

## Security

CodeQL and Govulncheck run independently of the normal feature tests.

## Runtime evidence

The authentic screenshot workflow captures:

- Windows;
- Linux;
- Android.

The workflow is read-only and tied to the exact tested source SHA. Reference/mockup images are never substituted for runtime proof.

## Retired tests

macOS-specific application, packaging and signing tests were removed together with the retired macOS application. The active regression suite must not reference missing macOS source or workflows.

