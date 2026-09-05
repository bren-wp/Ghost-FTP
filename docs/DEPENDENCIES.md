# Dependencies and external-component policy

Ghost FTP is designed to minimize third-party runtime dependencies, forbid application telemetry, and make every remaining dependency explicit and auditable.

## Policy goals

- No application analytics, advertising, behavioral tracking or external crash-reporting SDKs.
- No hidden Ghost FTP backend, remote configuration service or mandatory account service.
- No dynamic dependency versions such as `+`, `latest` or `SNAPSHOT` in production build inputs.
- No unreviewed dependency may be added merely for convenience.
- Security-sensitive protocol behavior must remain covered by regression tests even when a platform library is used.
- Dependency and signing claims in documentation must match the actual build.

`scripts/audit_dependencies.py` is a fail-closed CI gate for the currently approved dependency surface.

## Go desktop/core

The Go module currently has no external Go module requirements. `go.mod` contains the Ghost FTP module declaration and Go/toolchain metadata only. Production CI also sets `GOPROXY=off` and `GOSUMDB=off` so a build cannot silently download a new Go module.

The desktop FTP/FTPS implementation currently uses the operating system's `curl` executable as a hardened transport helper. Ghost FTP disables proxy inheritance for that process, supplies credentials through an ephemeral standard-input configuration, bounds child-process output, minimizes inherited environment state and requires normal TLS certificate validation for FTPS.

This system-tool dependency is not the same as an embedded analytics or cloud dependency, but it is still an external runtime component. The 1.1.x roadmap is to make the standard-library native FTP/FTPS transport the primary implementation after protocol-parity integration tests pass.

Desktop SFTP currently uses the operating system OpenSSH client/tooling rather than a third-party Go SSH module. Go's standard library does not provide an SSH/SFTP client, so documentation must continue to state this requirement until a reviewed in-repository implementation exists.

## Android

Android currently has two pinned runtime libraries:

- `commons-net:commons-net:3.13.0` for FTP/FTPS protocol support.
- `com.hierynomus:sshj:0.40.0` for SSH/SFTP support.

Unit tests use pinned `junit:junit:4.13.2`.

These coordinates are explicitly allowlisted by `scripts/audit_dependencies.py`. A new runtime library, version change or test framework change makes CI fail until the dependency is reviewed and the allowlist/documentation are intentionally updated.

The dependency audit also rejects known analytics, advertising and crash-reporting SDK markers.

## iOS

The native iOS client uses Apple platform networking for its supported FTP/FTPS transport surface and does not embed an analytics SDK, WebView wrapper or Ghost FTP cloud service.

## Web/PWA

The shared-hosting Web/PWA package has no Composer runtime dependencies. The public package is assembled from tracked project files and uses server-provided PHP extensions for protocol functionality.

No CDN, remote JavaScript bundle or analytics script is required for normal operation.

## GitHub Actions and build tooling

CI actions are pinned to immutable commit SHAs. Android builds require the Android/Gradle toolchain, Windows builds require the Go/Windows toolchain, and Apple builds require Xcode/macOS runners. These are build-environment requirements, not Ghost FTP application telemetry services.

## Adding a dependency

A proposed new dependency must document:

1. The capability that cannot reasonably be implemented with existing project/platform code.
2. Exact version and upstream provenance.
3. License and redistribution implications.
4. Network behavior and whether any telemetry exists or can be disabled.
5. Credential, filesystem and process-boundary impact.
6. Security update strategy.
7. Regression tests proving that the dependency cannot weaken Ghost FTP trust boundaries.
8. Removal/fallback strategy.

The change must update `scripts/audit_dependencies.py`, this document, `THIRD-PARTY-NOTICES.md` when applicable, and pass the full release CI matrix.
