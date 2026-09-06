# Ghost FTP dependencies

Ghost FTP **1.0.0 Stable** minimizes bundled third-party code, keeps the maintained Go module free of external module requirements and makes operating-system protocol prerequisites explicit.

## Go module contract

The root Go module is intentionally standard-library-only. CI rejects unexpected `require`, `replace`, `exclude` or `retract` directives, an unreviewed `go.sum` graph and vendored Go modules.

Production workflows use:

```text
GOPROXY=off
GOSUMDB=off
```

and explicitly disable Go telemetry before build/test.

## Runtime protocol prerequisites

Ghost FTP delegates protocol execution to audited system tools rather than embedding a second third-party networking stack into the Go module.

### FTP / FTPS

The maintained transport uses system `curl`. Ghost FTP supplies a controlled configuration/environment so ambient proxy/configuration state cannot silently redirect a selected FTP/FTPS connection.

Security invariants include:

- no user curl config inheritance for the managed operation;
- proxy-environment sanitization;
- protected runtime credential handling;
- download staging validation;
- FTPS certificate validation;
- no blanket certificate-revocation disable switch.

### SFTP

The maintained SFTP transport uses system OpenSSH `ssh`/`sftp`. Ghost FTP creates a constrained SSH configuration that disables ambient proxy/jump/agent/forwarding behavior that would escape the selected connection boundary.

Passwords/passphrases use the bounded AskPass/runtime-secret path and are not intentionally written into a password file.

## Windows UI dependency boundary

The Windows desktop frontend uses native Win32/DWM/common-control facilities. Ghost FTP does not bundle a large cross-platform UI runtime solely to render the workstation.

Windows production packages are native application executables/Setup wrappers generated from the repository build.

## Linux UI dependency boundary

Linux uses the maintained native X11/XWayland-compatible frontend backed by the same Engine. The Linux package therefore requires the normal display/runtime environment appropriate to that frontend in addition to protocol tools.

The Linux renderer is not a second protocol implementation.

## Accurate dependency wording

Ghost FTP has **zero external Go modules** in the maintained root module, but it does have operating-system runtime prerequisites. Documentation must not misrepresent that distinction as “zero runtime dependencies.”

## GitHub Actions dependencies

CI/release workflows use pinned GitHub Actions revisions for checkout, language setup and artifact transfer. These are build-system dependencies, not installed-application dependencies.

The stable release additionally uses Docker available on the GitHub-hosted Ubuntu runner to construct the GHCR distribution bundle from `FROM scratch`. Docker is not bundled into Ghost FTP and is not required to run the desktop application.

## GitHub Packages

The OCI release package has no runtime base image. It copies the already verified `release/` directory into `/ghostftp-release/`. The package is a distribution bundle only and does not add a runtime dependency to Ghost FTP.

## Tracking and analytics prohibition

Do not add runtime dependencies for application telemetry, advertising, behavior analytics, session replay, marketing attribution or automatic remote crash collection.

Repository privacy/dependency audits scan for known tracking/vendor markers and unexpected network behavior.

## Adding a dependency

Any proposal for a new runtime/library dependency must document:

1. exact component/version;
2. why existing standard-library/system facilities are insufficient;
3. license/provenance;
4. security/update ownership;
5. network/telemetry behavior;
6. whether it is bundled or system-provided;
7. rollback/removal plan;
8. CI/audit changes that prevent unreviewed drift.

A dependency should not be introduced merely to simplify a small helper that the existing platform layer can implement safely.

See [Third-party notices](THIRD-PARTY-NOTICES.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
