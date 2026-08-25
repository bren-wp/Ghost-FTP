# ByFTP documentation

English is the canonical documentation language. Runtime translations belong in `internal/i18n`; technical documentation stays English-first so code, CI and release instructions have one authoritative source.

## Core documentation

- [Installation](INSTALLATION.md)
- [Shared hosting](SHARED-HOSTING.md)
- [Architecture](ARCHITECTURE.md)
- [Security](SECURITY.md)
- [Privacy](PRIVACY.md)
- [Testing](TESTING.md)
- [Support](SUPPORT.md)
- [Roadmap](ROADMAP.md)
- [Contributing](CONTRIBUTING.md)
- [GitHub releases](GITHUB-RELEASES.md)
- [Release verification](RELEASE-VERIFICATION.md)
- [Signing](SIGNING.md)
- [Third-party notices](THIRD-PARTY-NOTICES.md)

## Platform guides

- [Android source and APK build guide](../android/README.md)
- [iOS source, Xcode and unsigned IPA build guide](../ios/README.md)
- [Build and verification tooling](../scripts/README.md)

The native mobile implementations intentionally live in separate top-level platform directories: Android under `android/` and iOS under `ios/`. Neither mobile app is a WebView wrapper around the desktop client.
