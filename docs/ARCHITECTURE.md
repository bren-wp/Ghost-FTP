# Ghost FTP architecture

Ghost FTP **0.0.5** uses one typed Go core with native Windows and Linux production frontends plus active Android and macOS development/source clients. The current public release allow-list remains Windows/Linux only; Android and macOS are validated independently and do not enter the public 17-file release contract without a separate production publication decision and evidence chain.

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

Application entry point for the maintained desktop product. It initializes product identity, local state and the platform frontend. It does not own protocol behavior.

### `internal/api`

Typed application engine used by maintained desktop frontends. It coordinates local/remote navigation, connection state, bookmarks, profile start directories, Remote Edit and tree transfers without exposing platform-specific UI details to protocol code.

### `internal/remote`

FTP/FTPS/SFTP connection and transfer implementation. This layer owns protocol command construction, process lifecycle for system transfer tools, host-key trust, remote staging/commit behavior and privacy-safe diagnostic classification.

### `internal/transfer`

Transfer queue and lifecycle state. Jobs have explicit generations, status transitions, progress snapshots, queue-priority ordering and cancellation/retry boundaries. UI consumers receive immutable/snapshot-style state instead of mutating in-flight protocol objects.

### `internal/config`

Settings and profile persistence. Writes use bounded local state, atomic/replace-oriented behavior and backup/recovery logic. Saved secrets are encrypted before durable profile storage. Fresh/fallback appearance resolves to Classic Light; explicitly persisted Dark remains supported by maintained desktop frontends.

### `internal/security`

Shared validators and local safety primitives: host/path validation, protected secret handling, remote file path checks, symlink/reparse-aware removal and SFTP fingerprint validation.

### `internal/desktop`

Native Windows and Linux frontends. Both use the same typed engine and product semantics. Platform-native rendering/input differences are permitted; protocol, privacy and transfer semantics are not duplicated. Current-folder filtering, bounded recursive search, directory comparison, bookmarks, queue priority, Remote Edit, appearance and file sorting are wired through maintained cross-platform contracts.

### Android client

The Android source surface is an active development client with Android 35 build/lint/APK verification, JVM regression tests and authentic emulator UI capture. It keeps Android-local boundaries such as Storage Access Framework document access and Activity lifecycle semantics while preserving the same security intent: explicit secure protocol selection, strict FTPS verification, account-bound saved paths, bounded/staged transfers and no hidden product backend.

Android SFTP remains hidden until strict native host-key identity verification has a maintained implementation. The development APK is not a public release artifact in 0.0.5.

### macOS client

The macOS source surface is an active native AppKit development frontend connected to the shared engine. It maintains site/profile workflows, file navigation and mutation, transfer queue behavior, filtering/search/comparison, Remote Edit and other parity work under native macOS lifecycle/UI constraints.

`.github/workflows/macos-app.yml` builds and verifies the universal native development app. That evidence proves maintained source/build parity; it does **not** claim a public Developer ID/notarized macOS distribution. macOS remains outside the current 17-file public release allow-list until a dedicated production signing/notarization/publication chain is completed and verified.

### `cmd/installer`

Windows per-user Setup/maintenance application. Installation is staged, validated and rollback-oriented. Integrated uninstall registration points back to the installed Ghost FTP maintenance path rather than a separate permanent uninstaller executable.

### `cmd/windowsbootstrap`

Public Windows architecture bootstrap. The tracked source contains no permanent architecture payload binaries; production packaging temporarily stages x64, x86 and ARM64 Setup or Portable payloads under the embedded payload tree, compiles the public bootstrap, then removes the temporary payload source directories.

At runtime the bootstrap asks Windows for the native processor through `GetNativeSystemInfo`, maps that identity to `x64`, `x86` or `arm64`, reads the corresponding embedded executable, writes it to a temporary Local AppData path, verifies the staged bytes against the embedded content identity and starts only the selected payload. The bootstrap does not download an architecture-specific executable.

## Connection architecture

A connection profile is normalized and validated before transport setup. Transport choice is explicit:

- **FTPS** is the fresh/default quick-connect transport on maintained desktop paths;
- FTP remains an explicit compatibility choice where unencrypted transport is deliberately selected;
- SFTP provides SSH-based transfer with host-key trust policy where maintained.

Failed secure transport is not silently converted to a weaker transport.

The shared engine drives connection lifecycle on desktop surfaces. A successful connection exposes remote list/operation state only after the transport session is established. Connection generation/identity invalidates stale asynchronous callbacks and transfer work when the user cancels, disconnects or reconnects.

Windows Add, Retry and Cancel-selected transfer completions are explicitly bound to the owning `connectionGeneration`; a callback from an obsolete session cannot update status or queue state belonging to a replacement session.

Connection errors pass through privacy-safe diagnostic classification before presentation. User-facing copy must not expose passwords, passphrases or protected secret payloads. Android additionally owns its in-flight FTP/FTPS session at Activity lifetime and sanitizes authentication failures so server-controlled replies cannot echo credentials into UI errors.

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

Saved secrets are opt-in. Platform-specific protected storage can differ, but persistent credential use must stay bound to explicit user consent and the intended connection identity.

Runtime diagnostic text is a privacy boundary. Tests reject credential-like material in user-facing error paths, including Android FTP authentication failures and desktop external-tool diagnostics.

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
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Internally, production packaging builds native Setup and Portable staging pairs for **x64, x86 and ARM64**. Each pair receives deterministic Ghost FTP VERSIONINFO, icon and manifest resources, is verified as the expected PE architecture, and passes through the same signing path when production signing is configured. The ARM64 PE contract uses machine `0xAA64`, PE32+ and `processorArchitecture="arm64"`.

The public bootstrap remains PE x86 for startup compatibility and uses `GetNativeSystemInfo` to select the matching embedded native payload. Architecture-specific staging binaries never enter the public release directory, so adding ARM64 does not add public `*-arm64.exe` files or change the public release count.

The canonical Windows metadata contract is:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

The ARM64 runtime-evidence marker is deliberately conservative. The maintained Windows CI runner cross-builds and structurally verifies ARM64 payloads, resources, bootstrap routing and signing mechanics, but it is not a native Windows ARM64 runner. Native ARM64 runtime execution must not be claimed until that separate evidence exists.

Official public Windows publication requires trusted Authenticode. The canonical release workflow requires the protected production signing identity, verifies both public executables and accepts only `WINDOWS_AUTHENTICODE=signed`. Local development/ordinary CI builds may be unsigned, but the official public workflow has no unsigned fallback and never creates a self-signed production identity as a substitute for a trusted certificate.

## Linux architecture

Linux uses the maintained native X11/XWayland-compatible desktop path and packages the same core for amd64, arm64 and i386. Classic Light is the fresh/fallback palette and Dark is an explicit persisted native runtime choice using the same appearance setting as Windows.

Canonical Linux publication is built by `linux/BUILD-DISTROS.sh` as Debian, Ubuntu, Fedora and distro-neutral Portable families. One production executable per architecture is reused across matching package variants and verified byte-for-byte. Native package-manager/runtime lifecycle is continuously exercised on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; additional architectures keep exact-head build, metadata, extraction and binary-parity coverage.

Trusted SFTP password/private-key-passphrase delivery remains constrained to the root-controlled package/system AskPass provenance boundary. User-writable Portable/per-user execution fails closed for automatic credential delivery rather than weakening helper provenance.

## Authentic UI evidence architecture

`.github/workflows/ui-screenshots.yml` is a read-only exact-head evidence workflow. It does not commit or push screenshots back into the tested branch.

The current Windows/Linux/Android evidence bundle verifies real runtime surfaces, exact source SHA, workflow-run identity, expected filenames, byte counts and SHA-256 values. Mockups, image-generation output and manually composed approximations are not accepted as production UI evidence.

The Windows UI capture is runtime evidence for the architecture used by the maintained Windows runner. It is not native ARM64 runtime evidence merely because the tested package also embeds an ARM64 payload.

macOS has its own native development build/validation workflow; do not reinterpret development build success as notarized public distribution evidence.

## Release architecture

The canonical public release workflow runs a complete quality gate before artifact publication:

1. formatting, race tests and vet;
2. repository, platform, dependency, security, privacy, localization and documentation audits;
3. complete Python regression suites;
4. Windows x64/x86/ARM64 native staging plus universal public packaging and Linux production builds;
5. canonical Linux distro package and native lifecycle gates;
6. independent Android/macOS development validation outside the public allow-list;
7. exact-head maintained runtime evidence where defined;
8. fail-closed trusted Authenticode verification for public Windows artifacts;
9. explicit Windows/Linux public release allow-list assembly;
10. SHA-256 manifest generation;
11. GitHub Release publication and read-back;
12. current GitHub Packages/GHCR distribution-bundle publication and registry read-back;
13. latest-only public release/tag/package retention only after successful verification.

The public release remains **14 platform artifacts / 17 public files**: two Windows executables, twelve Linux package/archive artifacts and three metadata/verification files. Android and macOS do not enter that allow-list in 0.0.5.

The GitHub Package is built only from the verified release directory with Docker build networking disabled. It is a distribution artifact, not an application runtime container.

## Supported production boundary

Ghost FTP 0.0.5 publicly releases Windows and Linux. Android and macOS are active development/source surfaces with maintained build gates, but neither is represented as a current public production download.

Product behavior, tests, release assets and documentation must stay aligned with those explicit boundaries.

See also [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Release verification](RELEASE-VERIFICATION.md).
