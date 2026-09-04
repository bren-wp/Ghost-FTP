# ByFTP documentation

English is the canonical documentation language. Runtime translations belong in `internal/i18n`; technical documentation stays English-first so code, CI and release instructions have one authoritative source.

**Current release: 1.9.1**

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

## Platform and build guides

- [Windows build/release overview](../README.md#build-from-source)
- [Linux source and DEB build guide](../linux/README.md)
- [macOS source and Universal PKG build guide](../macos/README.md)
- [Android source and APK build guide](../android/README.md)
- [iOS source, Xcode and unsigned IPA build guide](../ios/README.md)
- [ByFTP WEB shared-hosting guide](../ByFTP%20WEB/README.md)
- [Build, audit and release tooling](../scripts/README.md)

The maintained release surfaces are intentionally separated by platform while sharing the canonical release number from root `VERSION`. Windows/Linux/macOS use the reviewed Go 1.27.1 desktop core; Android uses AGP 9.4.0/Gradle 9.7.1; iOS has its native SwiftUI/Xcode project; ByFTP WEB is an audited PHP/PWA shared-hosting application with a deterministic deployable release ZIP.

Release 1.9.1 preserves the app-only Windows Setup/no-standalone-uninstaller contract while adding retryable private upload snapshot cleanup, explicit local rollback cleanup failure reporting, bounded WEB JSON state, correct FTP raw-list filename handling and delayed public release asset/digest readback. See [Installation](INSTALLATION.md), [GitHub releases](GITHUB-RELEASES.md) and [Release verification](RELEASE-VERIFICATION.md).
