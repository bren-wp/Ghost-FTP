# Ghost FTP dependencies

Ghost FTP **0.0.5** minimizes bundled third-party code, keeps the maintained root Go module free of external module requirements and makes operating-system, native-platform and release-tool prerequisites explicit.

The current public release platforms are Windows and Linux. Android and macOS are active native development/source surfaces with their own build toolchains; those build-time dependencies do not silently become runtime dependencies of the public Windows/Linux product.

## Go module contract

The root Go module is intentionally standard-library-only. CI rejects unexpected `require`, `replace`, `exclude` or `retract` directives, an unreviewed `go.sum` graph and vendored Go modules.

Production workflows use:

```text
GOPROXY=off
GOSUMDB=off
```

and explicitly disable Go telemetry before build/test.

## Runtime protocol prerequisites

Ghost FTP delegates maintained desktop protocol execution to audited system tools rather than embedding a second third-party networking stack into the Go module.

### FTP / FTPS

The maintained desktop transport uses system `curl`. Ghost FTP supplies controlled configuration/environment so ambient proxy/configuration state cannot silently redirect the selected FTP/FTPS connection.

Security invariants include:

- no user curl config inheritance for the managed operation;
- proxy-environment sanitization;
- protected runtime credential handling;
- download staging validation;
- FTPS certificate validation;
- no blanket certificate-revocation disable switch;
- no silent FTPS-to-plain-FTP downgrade.

Explicit FTPS/21 is the fresh desktop quick-connect default; plain FTP remains an intentional compatibility selection rather than a fallback dependency mode.

### SFTP

The maintained desktop SFTP transport uses system OpenSSH `ssh`/`sftp`. Ghost FTP creates a constrained SSH configuration that disables ambient proxy/jump/agent/forwarding behavior that would escape the selected connection boundary.

Passwords/passphrases use the bounded AskPass/runtime-secret path and are not intentionally written into a password file. Linux supports password authentication and private-key authentication with an optional key passphrase; documentation/runtime strings must not claim a narrower obsolete capability set.

## Windows UI and packaging boundary

The Windows desktop frontend uses native Win32/DWM/common-control facilities. Ghost FTP does not bundle a large cross-platform UI runtime solely to render the workstation.

Windows packages are native application executables/Setup wrappers generated from repository source. Classic Light/Dark rendering uses local native drawing state and does not load a remote theme service.

Official public Windows publication additionally requires the protected production Authenticode identity and a working trusted signing/verification toolchain. That infrastructure dependency belongs to the canonical release environment, not to ordinary runtime use. Local development and normal CI builds may remain unsigned, but official `Publish Ghost FTP` must fail if the protected signing identity is unavailable or the final signatures are invalid.

## Linux UI and packaging boundary

Linux uses the maintained native X11/XWayland-compatible frontend backed by the same Engine. The Linux renderer is not a second protocol implementation.

Canonical Debian/Ubuntu packages declare `ca-certificates`, `curl` and `openssh-client`; Fedora packages declare corresponding `ca-certificates`, `curl` and `openssh-clients` runtime requirements. Distro-neutral Portable archives intentionally do **not** bundle copies of those networking tools or CA stores.

Canonical 0.0.5 Linux publication includes Debian, Ubuntu, Fedora and Portable families for the maintained amd64/arm64/i386 architecture mapping. One production executable per architecture is reused across matching package variants and verified byte-for-byte.

Creating a Portable tarball does not change Ghost FTP's runtime dependency model. A user-writable Portable/per-user executable also does not inherit the trusted root-controlled AskPass provenance of a package-manager installation, so automatic SFTP password/private-key-passphrase delivery fails closed when that boundary is unavailable.

## Android dependency boundary

Android is an active native development source/APK surface in 0.0.5, not a public release artifact. Java, Gradle and Android SDK components are build/platform dependencies and do not alter the public Windows/Linux 17-file release contract.

The Android client uses platform TLS/networking and Storage Access Framework APIs rather than introducing a hidden Ghost FTP sync backend. Exact-head CI pins the maintained Java/Gradle/Android SDK build contract, runs JVM regressions and lint, builds an installable APK and verifies the APK separately from public release publication.

Android SFTP remains hidden until strict native host-key identity verification has a maintained implementation; adding an SSH dependency only to expose a checkbox would not satisfy that security contract.

## macOS dependency boundary

macOS is an active native development/source surface in 0.0.5. The frontend uses AppKit and native Apple security/file-access facilities while protocol behavior remains tied to the shared engine/platform bridge.

The development build requires the maintained Apple development toolchain available on macOS CI. Production distribution additionally requires platform tools such as `codesign`, `security`, `xcrun notarytool` and `spctl`, plus a real Developer ID Application identity and Apple notarization credentials.

Those tools/credentials are build and release infrastructure, not Ghost FTP runtime services. The application does not contact Apple notarization infrastructure while performing FTP/FTPS/SFTP work. Development build success is not evidence that production notarization credentials were configured or that a public macOS release exists.

## Browser helper dependency boundary

The optional browser connection helper is intentionally small and local. It must not introduce remote executable-code loading, analytics SDKs, broad host permissions, credential persistence or a hidden localhost/backend dependency.

The current helper parses/copies supported FTP-family targets locally. No supported browser-to-desktop launch/handoff integration exists today, so documentation and dependencies must not imply a native-messaging/IPC stack that is not implemented.

## Accurate dependency wording

Ghost FTP has **zero external Go modules** in the maintained root module, but it does have operating-system/runtime and build/release prerequisites. Documentation must not misrepresent that distinction as “zero dependencies.”

A platform SDK/tool required only to compile, sign or notarize an artifact is not automatically an installed-application runtime dependency.

## GitHub Actions dependencies

CI/release workflows use pinned GitHub Actions revisions for checkout, language setup, analysis and artifact transfer. These are build-system dependencies, not installed-application dependencies.

The current release additionally uses Docker available on the GitHub-hosted Ubuntu runner to construct the GHCR distribution bundle from `FROM scratch`. Docker is not bundled into Ghost FTP and is not required to run the desktop application.

## GitHub Packages

The OCI release package has no runtime base image. It copies the already verified public release directory into `/ghostftp-release/`. The package is a distribution bundle only and does not add a runtime dependency to Ghost FTP.

## Tracking and analytics prohibition

Do not add runtime dependencies for application telemetry, advertising, behavior analytics, session replay, marketing attribution or automatic remote crash collection.

Repository privacy/dependency audits scan for tracking/vendor markers and unexpected dependency/network-policy drift.

## Adding a dependency

Any proposal for a new runtime/library dependency must document:

1. exact component/version;
2. why existing standard-library/system/platform facilities are insufficient;
3. license/provenance;
4. security/update ownership;
5. network/telemetry behavior;
6. whether it is bundled, system-provided or build-only;
7. affected public/development platforms;
8. rollback/removal plan;
9. CI/audit changes that prevent unreviewed drift.

A dependency should not be introduced merely to simplify a small helper that the existing platform layer can implement safely.

See [Third-party notices](THIRD-PARTY-NOTICES.md), [Security](SECURITY.md), [Privacy](PRIVACY.md), [Signing](SIGNING.md) and [`../macos/README.md`](../macos/README.md).
