# Ghost FTP dependencies

Ghost FTP **0.0.6** minimizes bundled third-party code, keeps the maintained root Go module free of external module requirements and makes operating-system, platform and release-tool prerequisites explicit.

The published 0.0.6 release includes Windows, Linux, Android and browser packages. macOS remains an active native source surface with a separate distribution-signing boundary.

## Go module contract

The root Go module is intentionally standard-library-only. CI rejects unexpected `require`, `replace`, `exclude` or `retract` directives, an unreviewed `go.sum` graph and vendored Go modules.

Production workflows use:

```text
GOPROXY=off
GOSUMDB=off
```

and explicitly disable Go telemetry before build/test.

## Desktop protocol prerequisites

The maintained desktop transport uses controlled system protocol tools where documented. Environment/configuration is sanitized so ambient proxy, jump-host or credential state cannot silently redirect the selected connection.

FTP/FTPS requires strict certificate behavior for explicit FTPS and never silently downgrades to plain FTP. Desktop SFTP uses maintained OpenSSH integration with strict host-key trust/pinning and bounded protected credential delivery.

## Windows

The Windows UI uses native Win32/DWM/common-control facilities and does not bundle a large remote or browser runtime solely for rendering. Official public packages require protected Authenticode signing infrastructure; that release tooling is not a runtime dependency.

## Linux

Canonical packages require platform CA trust, `curl` and OpenSSH client tooling as documented by distro. Portable packages do not bundle replacement CA stores/network tools. Security-sensitive helper provenance fails closed when trust cannot be established.

## Android

Java, Gradle and Android SDK components are build/platform dependencies. Runtime file access uses Android platform APIs and Storage Access Framework rather than a Ghost FTP synchronization backend. Android SFTP remains hidden until a maintained strict host-key implementation exists.

## macOS

The native AppKit source uses Apple platform facilities. Public distribution additionally requires `codesign`, Developer ID Application identity, `xcrun notarytool` and Gatekeeper validation. These are release infrastructure rather than FTP runtime services.

## Browser extensions and native companion

The published 0.0.6 browser packages contain local extension code. For the 0.0.7 architecture, direct FTP/SFTP parity may require a local Ghost FTP native-messaging companion because browser sandboxes do not expose arbitrary raw protocol sockets.

A native companion is an explicit product dependency, not a hidden localhost service. It must be versioned, installed locally, registered through browser-supported native messaging, use least privilege and never relay credentials or file data through Ghost FTP servers.

Do not add analytics SDKs, advertising libraries, fingerprinting, remote executable code or dependencies whose only purpose is telemetry/crash upload.

## GitHub Actions dependencies

CI/release workflows use pinned GitHub Actions revisions for checkout, language setup, analysis and artifact transfer. These are build-system dependencies, not installed-application dependencies.

## GHCR

`ghcr.io/bren-wp/ghost-ftp:0.0.6` is a verified distribution bundle built from release artifacts. It is not a runtime product backend and does not add a network service dependency to Ghost FTP.

## Adding a dependency

Any new runtime/library dependency must document:

1. exact component/version;
2. why existing standard-library/system/platform facilities are insufficient;
3. license/provenance;
4. security/update ownership;
5. network/telemetry behavior;
6. whether it is bundled, system-provided or build-only;
7. affected platforms;
8. rollback/removal plan;
9. CI/audit changes that prevent unreviewed drift.

See [Third-party notices](THIRD-PARTY-NOTICES.md), [Security](SECURITY.md), [Privacy](PRIVACY.md), [Signing](SIGNING.md) and [`../macos/README.md`](../macos/README.md).
