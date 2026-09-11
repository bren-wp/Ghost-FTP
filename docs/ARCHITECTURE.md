# Ghost FTP architecture

Ghost FTP **0.0.4** uses one typed Go core with native Windows and Linux desktop frontends plus an active Android development client. Public release artifacts remain Windows/Linux only; Android is validated as a development APK and does not enter the public 17-file release allow-list until a separate production signing/publication contract exists.

The product is designed around explicit platform adapters, local-only persistent state, fail-closed transfer/security boundaries and shared behavior wherever platform constraints permit it.

## Release identity

The root `VERSION` file is the production version source. Release binaries receive the version through linker flags; development source keeps a `dev` fallback and does not hard-code a production number.

The official public tag namespace is:

```text
ghostftp-vX.Y.Z
```

A release candidate always uses a new semantic version rather than moving an existing current release tag. After a newer release is successfully published and verified, the latest-only retention workflow may remove superseded public Ghost FTP release/tag identities without rewriting `main` commit history.

## Main layers

### `cmd/ghostftp`

Application entry point for the desktop product. It initializes product identity, local state and the platform desktop frontend. It does not own protocol behavior.

### `internal/api`

Typed application engine used by both desktop frontends. It coordinates local/remote navigation, connection state, bookmarks, profile start directories, Remote Edit and tree transfers without exposing platform-specific UI details to protocol code.

### `internal/remote`

FTP/FTPS/SFTP connection and transfer implementation. This layer owns protocol command construction, process lifecycle for system transfer tools, host-key trust, remote staging/commit behavior and privacy-safe diagnostic classification.

### `internal/transfer`

Transfer queue and lifecycle state. Jobs have explicit generations, status transitions, progress snapshots, queue-priority ordering and cancellation/retry boundaries. UI consumers receive immutable/snapshot-style state instead of mutating in-flight protocol objects.

### `internal/config`

Settings and profile persistence. Writes use bounded local state, atomic/replace-oriented behavior and backup/recovery logic. Saved secrets are encrypted before durable profile storage. Fresh/fallback appearance resolves to Classic Light; an explicitly persisted Dark selection is supported by both native desktop frontends.

### `internal/security`

Shared validators and local safety primitives: host/path validation, protected secret handling, remote file path checks, symlink/reparse-aware removal and SFTP fingerprint validation.

### `internal/desktop`

Native Windows and Linux frontends. Both use the same typed engine and product semantics. Platform-native rendering/input differences are permitted; protocol, privacy and transfer semantics are not duplicated. Current-folder filtering, bounded recursive search, directory comparison, bookmarks, queue priority, Remote Edit, appearance and file sorting are wired through maintained cross-platform contracts.

### Android client

The Android source surface is an active development client with Android 35 build/lint/APK verification and authentic emulator UI capture. It keeps Android-local platform boundaries such as Storage Access Framework document access and activity lifecycle semantics while preserving the same security intent: explicit secure protocol selection, strict FTPS verification, account-bound saved paths, bounded/staged transfers and no hidden product backend. The development APK is not a public release artifact in 0.0.4.

### `cmd/installer`

Windows per-user Setup/maintenance application. Installation is staged, validated and rollback-oriented. Integrated uninstall registration points back to the installed Ghost FTP maintenance path rather than a separate permanent uninstaller executable.

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
- cancellation checks before final-name activation;
- cleanup that refuses unsafe traversal through symlink/reparse boundaries;
- bounded retry/cancel lifecycle;
- truthful progress, speed and ETA snapshots rather than fabricated completion state.

## State and privacy architecture

Settings, profiles and protected credentials remain local. Ghost FTP has no application telemetry backend and no product account service. Production workflows explicitly disable Go telemetry.

Saved secrets are opt-in:

- Windows uses the current-user operating-system protection boundary;
- Linux uses local authenticated encryption with user-private key material plus process-local protected-secret handles for runtime lifetime control.

The desktop profile-save flows require explicit credential-persistence consent. Entering a password/passphrase does not by itself authorize durable storage, and profile identity binding prevents stored credentials from silently migrating to another endpoint/account identity.

Runtime diagnostic text is treated as a privacy boundary. Tests reject credential-like material in user-facing error paths.

## Dependency architecture

The maintained Go module has no external module requirements. CI/release jobs run with:

```text
GOPROXY=off
GOSUMDB=off
```

Platform transfer capabilities are explicit system-runtime dependencies and are checked rather than downloaded dynamically by the application.

## Windows architecture

Windows uses native Win32 surfaces and controls. Classic Light is the fresh/fallback appearance; Dark remains an explicit persisted choice.

The canonical public release contains exactly two Windows executables:

```text
Ghost-FTP-0.0.4-Setup.exe
Ghost-FTP-0.0.4-Portable.exe
```

Verified native x64/x86 Setup and Portable payloads remain internal staging/evidence inputs. The public x86-compatible bootstrap uses `GetNativeSystemInfo` to select the matching embedded native payload, verifies staged bytes and performs no runtime download.

Production Authenticode is an optional hardening layer. If a protected trusted certificate is configured, the release pipeline signs and verifies the Windows artifacts. If it is absent, the release remains explicitly unsigned and records that state in `BUILD-METADATA.txt`. The production workflow never creates a self-signed publisher identity as a substitute for a real trusted certificate.

## Linux architecture

Linux uses the maintained native X11/XWayland-compatible desktop path and packages the same core for amd64, arm64 and i386. Classic Light is the fresh/fallback palette and Dark is an explicit persisted native runtime choice using the same appearance setting as Windows.

Canonical Linux publication is built by `linux/BUILD-DISTROS.sh` as Debian, Ubuntu, Fedora and distro-neutral Portable families. One production executable per architecture is reused across matching package variants and verified byte-for-byte. Native package-manager/runtime lifecycle is continuously exercised on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; additional architectures keep exact-head build, metadata, extraction and binary-parity coverage.

Trusted SFTP password/private-key-passphrase delivery remains constrained to the root-controlled package/system AskPass provenance boundary. User-writable Portable/per-user execution fails closed for automatic credential delivery rather than weakening helper provenance.

## Authentic UI evidence architecture

`.github/workflows/ui-screenshots.yml` is a read-only exact-head evidence workflow. It does not commit or push screenshots back into the tested branch.

The workflow launches real runtime surfaces and produces 15 verified images:

- Windows: 5;
- Linux: 3;
- Android: 7.

`scripts/assemble_ui_evidence.py` verifies the exact source SHA, workflow-run identity, expected filename set, byte counts and SHA-256 values before producing `ghostftp-authentic-ui-verified-bundle`. Mockups, image-generation output and manually composed approximations are not accepted as production UI evidence.

## Release architecture

The release workflow runs a complete quality gate before artifact publication:

1. formatting, race tests and vet;
2. repository, platform, dependency, security, privacy, localization and documentation audits;
3. complete Python regression suites;
4. Windows and Linux production builds;
5. canonical Linux distro package and native lifecycle gates;
6. Android source/lint/installable-APK validation as an active development surface;
7. exact-head Windows/Linux/Android authentic UI evidence;
8. signing-state/metadata checks, plus Authenticode verification when configured;
9. explicit Windows/Linux public release allow-list assembly;
10. SHA-256 manifest generation;
11. GitHub Release publication and read-back;
12. current GitHub Packages/GHCR distribution-bundle publication and registry read-back;
13. latest-only public release/tag/package retention only after successful verification.

The public release remains **14 platform artifacts / 17 public files**: two Windows executables, twelve Linux package/archive artifacts and three metadata/verification files. Android does not enter that allow-list in 0.0.4.

The GitHub Package is built only from the verified release directory with Docker build networking disabled. It is a distribution artifact, not an application runtime container.

## Supported production boundary

Ghost FTP 0.0.4 publicly releases Windows and Linux. Android is an active source/development APK surface with its own build and authentic-runtime gates, but is not represented as a public production download.

Product behavior, tests, release assets and documentation must stay aligned with those explicit boundaries.

See also [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Packages](PACKAGES.md) and [Release verification](RELEASE-VERIFICATION.md).
