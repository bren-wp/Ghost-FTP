# Ghost FTP architecture

Ghost FTP **1.1.1 Stable** is a native Windows/Linux desktop client for FTP, FTPS and SFTP. The product is designed around a small typed Go core, explicit platform adapters, local-only persistent state and fail-closed transfer/security boundaries.

## Release identity

The root `VERSION` file is the production version source. Release binaries receive the version through linker flags; development source keeps a `dev` fallback and does not hard-code a production number.

The official tag namespace is:

```text
ghostftp-vX.Y.Z
```

Previously published release tags are immutable history; a new stable candidate uses a new semantic version rather than moving an older tag.

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

Settings and profile persistence. Writes use bounded local state, atomic/replace-oriented behavior and backup/recovery logic. Saved secrets are encrypted before durable profile storage. Fresh/fallback appearance state resolves to Classic Light; invalid/missing state does not create a third implicit theme.

### `internal/security`

Shared validators and local safety primitives: host/path validation, protected secret handling, remote file path checks, symlink/reparse-aware removal and SFTP fingerprint validation.

### `internal/desktop`

Native Windows and Linux frontends. Both use the same typed engine and product semantics. Platform-native rendering/input differences are permitted; protocol, privacy and transfer semantics are not duplicated.

### `cmd/installer`

Windows per-user Setup/maintenance application. Installation is staged, validated and rollback-oriented. Integrated uninstall registration points back to the installed Ghost FTP maintenance path rather than a separate unrelated product.

## Connection architecture

A connection profile is normalized and validated before transport setup. Transport choice is explicit:

- **FTPS** is the fresh/default quick-connect transport on Windows and Linux;
- FTP remains an explicit compatibility choice where unencrypted transport is deliberately selected;
- SFTP provides SSH-based transfer with host-key trust policy.

Failed secure transport is not silently converted to a weaker transport.

The desktop frontends drive the shared `remote.Manager` connection lifecycle. A successful connection exposes remote list/operation state only after the transport session is established. Connection generation/identity invalidates stale asynchronous callbacks and transfer work when the user cancels, disconnects or reconnects.

Connection errors pass through `internal/usererror` and shared-hosting diagnostic classification before presentation. The user receives actionable categories while passwords, passphrases and protected secret payloads remain excluded from error copy.

## SFTP trust and protected-secret architecture

SFTP host-key verification can require a two-step trust flow. Pending trust state distinguishes protected secrets it owns from protected profile blobs it borrows.

Owned temporary credentials are forgotten on cancel, expiry, fingerprint mismatch, replacement or failed/abandoned setup. On successful confirmation, ownership transfers only when the exact protected blob is accepted by the SFTP session. Session Close forgets session-owned secrets but does not invalidate borrowed profile credentials.

This ownership model prevents both unnecessary secret retention and reconnect regressions caused by deleting profile-owned credential handles.

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
- Linux uses local authenticated encryption with user-private key material plus process-local protected-secret handles for runtime lifetime control.

The main profile flow and Windows Site Manager share one credential-persistence consent policy. Entering a password/passphrase does not by itself authorize durable storage.

Runtime diagnostic text is treated as a privacy boundary. Tests reject credential-like material in user-facing error paths.

## Dependency architecture

The maintained Go module has no external module requirements. CI/release jobs run with:

```text
GOPROXY=off
GOSUMDB=off
```

Platform transfer capabilities are explicit system-runtime dependencies and are checked rather than downloaded dynamically by the application.

## Windows architecture

Windows uses native Win32 surfaces and controls. Classic Light is the fresh/fallback appearance; Dark remains an explicit persisted choice. The release pipeline produces x64 and x86 Setup/Portable packages; the x32 Setup name is a byte-identical compatibility alias of x86.

Production Authenticode is an optional hardening layer. If a protected trusted certificate is configured, the release pipeline signs and verifies the Windows artifacts. If it is absent, the release remains explicitly unsigned and records that state in `BUILD-METADATA.txt`. The production workflow never creates a self-signed publisher identity as a substitute for a real trusted certificate.

## Linux architecture

Linux uses the maintained native X11/XWayland-compatible desktop path and packages the same core for amd64, arm64 and i386. Classic Light is the canonical Linux UI palette. DEB metadata is generated from `VERSION` and verified before publication.

## Authentic UI evidence architecture

The dedicated screenshot workflow builds and launches the real Windows x64 Portable executable, captures maintained product windows and validates/persists PNG evidence only when the capture still corresponds to the branch head. Documentation must not substitute generated or manually composed mockups for production UI evidence.

## Release architecture

The release workflow runs a complete quality gate before artifact publication:

1. formatting, race tests and vet;
2. repository, platform, dependency, security, privacy, localization and documentation audits;
3. Python regression suites;
4. Windows and Linux production builds;
5. signing-state/metadata checks, plus Authenticode verification when configured;
6. explicit release allow-list assembly;
7. SHA-256 manifest generation;
8. GitHub Release publication and read-back;
9. stable GitHub Packages/GHCR distribution-bundle publication and registry read-back.

The GitHub Package is built only from the verified `release/` directory with Docker build networking disabled. It is a distribution artifact, not an application runtime container.

## Supported production boundary

Ghost FTP 1.1.1 maintains Windows and Linux as the active application platforms. Product behavior, tests, release assets and documentation must stay aligned with that boundary.

See also [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Packages](PACKAGES.md) and [Release verification](RELEASE-VERIFICATION.md).
