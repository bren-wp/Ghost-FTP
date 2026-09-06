# Ghost FTP architecture

Ghost FTP **1.0.0 Stable** is a native Windows/Linux desktop client for FTP, FTPS and SFTP. The product is designed around a small typed Go core, explicit platform adapters, local-only persistent state and fail-closed transfer/security boundaries.

## Release identity

The root `VERSION` file is the production version source. Release binaries receive the version through linker flags; development source keeps a `dev` fallback and does not hard-code a production number.

The official tag namespace is:

```text
ghostftp-vX.Y.Z
```

## Main layers

### `cmd/ghostftp`

Application entry point. It initializes product identity, local state and the platform desktop frontend. It does not own protocol behavior.

### `internal/api`

Typed application engine used by both desktop frontends. It coordinates local/remote navigation, connection state and tree transfers without exposing platform-specific UI details to protocol code.

### `internal/remote`

FTP/FTPS/SFTP connection and transfer implementation. This layer owns protocol command construction, process lifecycle for system transfer tools, host-key trust, remote staging/commit behavior and privacy-safe diagnostic classification.

### `internal/transfer`

Transfer queue and lifecycle state. Jobs have explicit generations, status transitions, progress snapshots and cancellation/retry boundaries. UI consumers receive immutable/snapshot-style state instead of mutating in-flight protocol objects.

### `internal/config`

Settings and profile persistence. Writes use bounded local state, atomic/replace-oriented behavior and backup/recovery logic. Saved secrets are encrypted before durable profile storage.

### `internal/security`

Shared validators and local safety primitives: host/path validation, protected secret handling, remote file path checks, symlink/reparse-aware removal and SFTP fingerprint validation.

### `internal/desktop`

Native Windows and Linux frontends. Both use the same typed engine and product semantics. Platform-native rendering/input differences are permitted; protocol, privacy and transfer semantics are not duplicated.

### `cmd/installer`

Windows per-user Setup/maintenance application. Installation is staged, validated and rollback-oriented. Integrated uninstall registration points back to the installed Ghost FTP maintenance path rather than a separate unrelated product.

## Connection architecture

A connection profile is normalized and validated before transport setup. Transport choice is explicit:

- FTP for compatibility where unencrypted transport is deliberately selected;
- FTPS where TLS protection is requested;
- SFTP for SSH-based transfer with host-key trust policy.

Failed secure transport is not silently converted to a weaker transport.

Connection errors pass through `internal/usererror` and shared-hosting diagnostic classification before presentation. The user receives actionable categories while passwords, passphrases and protected secret payloads remain excluded from error copy.

## Transfer integrity

Ghost FTP uses staged/rollback-oriented operations for remote writes and local destination changes where the protocol/tooling permits it. The transfer layer binds jobs to the connection generation that created them so stale work cannot silently continue against a later connection.

Important invariants include:

- local path containment before filesystem mutation;
- source snapshots for uploads so mid-transfer source changes can be detected/handled deterministically;
- remote destination validation before commit where available;
- cleanup that refuses unsafe traversal through symlink/reparse boundaries;
- bounded retry/cancel lifecycle;
- truthful progress, speed and ETA snapshots rather than fabricated completion state.

## State and privacy architecture

Settings, profiles and protected credentials remain local. Ghost FTP has no application telemetry backend and no product account service. Production workflows explicitly disable Go telemetry.

Saved secrets are opt-in:

- Windows uses the current-user operating-system protection boundary;
- Linux uses local authenticated encryption with user-private key material.

Runtime diagnostic text is treated as a privacy boundary. Tests reject credential-like material in user-facing error paths.

## Dependency architecture

The maintained Go module has no external module requirements. CI/release jobs run with:

```text
GOPROXY=off
GOSUMDB=off
```

Platform transfer capabilities are explicit system-runtime dependencies and are checked rather than downloaded dynamically by the application.

## Windows architecture

Windows uses native Win32 surfaces and controls. The release pipeline produces x64 and x86 Setup/Portable packages; the x32 Setup name is a byte-identical compatibility alias of x86.

Stable release publication requires the protected trusted Authenticode identity. Signing key material is never stored in the repository.

## Linux architecture

Linux uses the maintained native X11/XWayland-compatible desktop path and packages the same core for amd64, arm64 and i386. DEB metadata is generated from `VERSION` and verified before publication.

## Release architecture

The release workflow runs a complete quality gate before artifact publication:

1. formatting, race tests and vet;
2. repository, platform, dependency, security, privacy, localization and documentation audits;
3. Python regression suites;
4. Windows and Linux production builds;
5. signing/metadata checks;
6. explicit release allow-list assembly;
7. SHA-256 manifest generation;
8. GitHub Release publication and read-back;
9. stable GitHub Packages/GHCR distribution-bundle publication and registry read-back.

The GitHub Package is built only from the verified `release/` directory with Docker build networking disabled. It is a distribution artifact, not an application runtime container.

## Supported production boundary

Ghost FTP 1.0.0 maintains Windows and Linux as the active application platforms. Product behavior, tests, release assets and documentation must stay aligned with that boundary.

See also [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Packages](PACKAGES.md) and [Release verification](RELEASE-VERIFICATION.md).
