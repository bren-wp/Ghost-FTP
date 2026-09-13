# Ghost FTP security

Ghost FTP **0.0.5** uses explicit transport, path, secret, process and release boundaries. Security-sensitive behavior is implemented in typed code and covered by platform-specific regression tests, exact-head native builds and repository audits.

The current public release platforms are Windows and Linux. Android and macOS are active native development/source surfaces and retain explicit platform-specific security boundaries without being silently promoted into the public 17-file release allow-list.

## Supported transport security

Ghost FTP supports FTP, FTPS and SFTP on the maintained desktop engine.

- Plain FTP is an unencrypted compatibility mode and must not be confused with a secure transport.
- Fresh/quick-connect defaults use explicit **FTPS on port 21** on maintained desktop paths.
- FTPS uses TLS protection and does not silently downgrade a failed secure request to plain FTP.
- SFTP uses SSH semantics and enforces host-key trust/fingerprint validation before a server is treated as trusted.

Connection profiles are validated before use: host, port, protocol, remote path, credential fields and key/fingerprint inputs pass through bounded validation logic.

The Android development client preserves the same no-silent-downgrade and strict FTPS certificate/hostname-verification intent while using platform-native storage/activity boundaries. Android SFTP remains intentionally hidden until strict native host-key identity verification has a maintained implementation. Its APK is development evidence in 0.0.5, not a public release artifact.

The macOS development frontend uses the shared engine for FTP/FTPS/SFTP and preserves fail-closed host-key/certificate behavior. Native AppKit/Keychain integration is a platform adapter, not a separate weaker protocol stack.

## SFTP host-key trust

SFTP host-key fingerprints are normalized and validated by the shared security layer. A changed/unexpected key is an identity problem, not a harmless connectivity warning. Users should verify the new fingerprint through an independent trusted channel before accepting an intentional server-key rotation.

Private-key authentication validates local key paths and keeps passphrases out of durable plaintext profile fields.

### Protected-secret ownership

Linux runtime protected secrets use explicit ownership semantics. Session-owned password/passphrase handles are forgotten when the session closes, while borrowed profile-owned handles remain available to the profile store. Constructor/setup failure paths clean newly owned secrets.

Pending host-key trust state follows the same rule: owned temporary credentials are cleaned on cancel, expiry, mismatch, replacement or abandoned setup, and successful confirmation transfers ownership only when the exact protected blob is accepted by the SFTP session. A credential captured for the actual trust attempt is not silently replaced by stale profile state unless the user explicitly supplies a new value.

On macOS, durable profile protection is handled through the maintained native security bridge/Keychain boundary. Raw durable protected-profile blobs are not treated as ordinary Swift settings state, and the native frontend does not introduce a second host-key trust model.

## FTPS certificate trust

FTPS relies on normal certificate/hostname verification for the selected server. The application does not ship a general “trust everything” mode for production use. A TLS failure is surfaced as a connection error rather than retried through a weaker transport.

Regression coverage explicitly verifies that an FTPS request aimed at a plaintext-only FTP endpoint fails instead of producing a plain FTP session.

## Android authentication privacy

Android FTP authentication uses a dedicated privacy boundary for `USER`/`PASS` negotiation. Server-controlled authentication reply bodies are not propagated verbatim into user-facing exceptions, including malformed replies received after a password is submitted.

This prevents a malicious or misconfigured FTP server from reflecting a credential into an error message and turning the UI/diagnostic path into a secret-exposure channel.

## Input and path validation

Untrusted values are bounded before use. Maintained validators cover host names/IP addresses, ports, control characters, remote file paths, local containment and destructive filesystem operations.

Remote/local tree operations are designed to avoid traversal through unsafe paths. Local recursive deletion includes symlink/reparse-aware protections so a selected tree cannot silently escape its intended root.

Android local-file access remains Storage Access Framework based rather than introducing arbitrary filesystem-path authority. Saved Android navigation state remains account/endpoint bound where remote identity matters.

macOS local file selection uses native file-panel/security primitives around shared engine operations. Platform adaptation must not bypass core path/remote identity validation.

## Transfer staging and commit safety

Transfers are treated as lifecycle operations rather than blind file copies. The maintained release includes tests for:

- upload-source snapshots;
- staged remote operations and cleanup;
- remote destination/commit revalidation;
- local rollback cleanup;
- transfer generation binding across reconnects;
- cancellation/failure terminal-state correctness;
- cancellation checks before final-name activation;
- symlink-safe filesystem handling.

The goal is fail-closed behavior when source/destination identity changes while an operation is in flight. Android staged downloads preserve the same principle: incomplete or cancelled work must not be promoted to the final user-visible object.

Windows Add, Retry and Cancel-selected completion callbacks are all bound to the connection generation that initiated them. A stale cancellation callback from an obsolete connection is rejected rather than being allowed to update the queue/status state of a replacement connection.

## Process execution boundary

Some FTP/SFTP functionality uses explicitly detected system transfer tools. Process construction, environment handling, tool capability probing and lifecycle are covered by regression tests. Credentials are not intentionally placed into user-visible command output or persisted runtime credential files.

Tool availability is diagnosed; the application does not silently download replacement networking tools.

On Linux, `curl`, `ssh`, `sftp` and `ssh-keyscan` are accepted only through a root-owned executable and directory/symlink provenance chain that is not writable by group or other users. `PATH` is a discovery hint, not a trust boundary.

Linux OpenSSH AskPass adds a second executable-identity boundary. The Ghost FTP helper path used for password/private-key-passphrase delivery must have the same trusted root-controlled provenance and must identify the same inode as the already-running Ghost FTP image. `/proc/self/exe` is used only inside Ghost FTP as an identity oracle; it is never handed to OpenSSH as `SSH_ASKPASS`.

The immediate AskPass parent must also resolve to a trusted root-controlled `ssh` or `sftp` executable. If the running Ghost FTP path is user-writable, such as a directly extracted Portable/per-user copy, Ghost FTP continues to run but does not construct a credential-bearing AskPass environment. Failure occurs before the AskPass token is generated and before the OpenSSH child is started.

## Windows universal bootstrap boundary

The public Windows release contains only:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Each public package embeds internal native **x64, x86 and ARM64** payloads. The PE x86 bootstrap determines the native processor through `GetNativeSystemInfo`; environment variables are not the architecture trust source. It reads only the matching embedded payload and does not fetch architecture-specific executable code from the network.

The selected embedded executable is written to a temporary Local AppData path and verified against the embedded byte identity before execution. An empty, oversized, truncated or hash-mismatched staged payload fails closed.

Architecture-specific staging executables are build evidence only and are forbidden from the public release directory. Release metadata records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 evidence marker is also a security/trust statement: cross-build, PE/resource, embedded-payload and signing verification must not be inflated into a claim that native ARM64 runtime execution occurred when the maintained Windows runner is not ARM64.

## Saved credential protection

Saved credentials are opt-in. Maintained profile-save flows require explicit credential-persistence consent before a newly entered password or private-key passphrase is persisted.

### Windows

Protected profile secrets use the current-user Windows protection boundary. Sensitive runtime values are not intended to be serialized as plaintext profile fields.

### Linux

Saved secrets use maintained local authenticated-encryption/protected-secret handling with user-private material. Runtime session secret handles are process-local and ownership-aware; session-only credentials are not promoted into persistent state merely because a connection was attempted.

### macOS development frontend

Durable saved-profile secrets use the native Keychain-backed protection path documented under `macos/`. The frontend keeps saved-secret protection separate from ordinary profile/UI state and uses device/user-scoped platform protection semantics rather than a repository-stored encryption key.

### Android development client

The current saved-site contract intentionally persists non-secret connection/navigation metadata only. Android must not acquire silent durable password storage as a side effect of connection UI work.

If protected data cannot be safely decrypted, Ghost FTP should require the user to re-enter the secret rather than falling back to plaintext persistence.

Changing endpoint/account/private-key identity must not silently carry an existing protected credential into the changed profile identity.

## Runtime secret minimization

Runtime secrets are kept only as long as required for the selected operation. Diagnostic/error classification is deliberately separated from secret values. Tests cover privacy-safe error reporting, profile-secret binding, Android authentication-error redaction and owned/borrowed secret lifetime.

## Settings/profile durability

Settings and profiles use local persistence with replacement/recovery behavior rather than unbounded append logs. Validation runs again when data is loaded. Malformed or invalid state must not become trusted merely because it came from a local file.

Bookmarks persist navigation metadata only; they are not a second credential store. Remote bookmark activation is revalidated against the active account/session before navigation becomes authoritative.

## Browser connection helper boundary

The optional `ekstenzije/` helper is deliberately not a protocol engine or security bypass. Its maintained source contract permits local parsing of explicitly entered/pasted FTP-family targets and explicit copying of a safe target.

The helper must not:

- persist FTP passwords or private-key passphrases;
- request broad browsing/host access merely to discover links;
- scrape ordinary tab contents;
- load remote executable code;
- create a hidden localhost credential server;
- claim a browser-to-desktop launch/handoff mechanism that has not been implemented and reviewed.

It does not replace FTPS/SFTP trust enforcement, because actual transport security remains owned by the native client/platform implementation.

## Network privacy boundary

Ghost FTP has no application telemetry service, ad SDK or account backend. Production CI and release jobs explicitly disable Go telemetry. Network activity is user-directed transport traffic plus the selected server diagnostics required to operate the chosen protocol.

## Release supply-chain security

The production/release-validation workflows:

- pin GitHub Actions to exact revisions;
- disable Go telemetry and external Go module resolution;
- run race tests, vet and security/privacy/dependency/documentation/release audits;
- build Windows/Linux public artifacts from exact source;
- cross-build and verify native Windows x64/x86/ARM64 Setup/Portable staging payloads before universal packaging;
- reject architecture-specific Windows executables from the public artifact directory;
- lint/test/build/verify the Android development APK from exact source;
- build/validate the universal native macOS development app from exact source;
- run exact-head authentic Windows/Linux/Android UI evidence without allowing the evidence workflow to commit or push into the tested branch;
- require public Windows release artifacts to be signed with a protected trusted Authenticode identity;
- fail closed before publication when the production Windows signing identity is absent or either public executable is unsigned/invalid;
- verify every configured production signature and require valid Authenticode status before publication;
- never generate a self-signed production publisher identity;
- remove temporary signing material from the runner after signing;
- assemble only the explicit Windows/Linux public release file set;
- record `WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64`, `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` and `WINDOWS_AUTHENTICODE=signed` in verified public release metadata;
- generate SHA-256 checksums;
- prevent an existing version tag from being rewritten to another commit;
- verify the published GitHub Release asset set and current `prerelease=false` state;
- publish the current GHCR release bundle only from the verified release directory;
- verify the registry artifact can be read back;
- permit latest-only cleanup only after the newly published release has been fully verified.

Private signing material must never be committed to source. Public Windows publication requires the protected production signing identity; absence of that identity causes the release build to fail instead of publishing unsigned Windows binaries. Ordinary CI, development and local builds may remain unsigned because they are not public release artifacts.

Android and macOS remain outside the 0.0.5 public release allow-list. A successful development APK/app build is not authorization to publish it as a production mobile/macOS release.

For macOS, the separate production path requires real Developer ID signing and Apple notarization credentials and must complete signing, notarization, stapling/validation and Gatekeeper assessment before a production-distributable artifact can be claimed. Repository documentation must not infer that success from ordinary development CI.

## GitHub Packages boundary

The current package at `ghcr.io/bren-wp/ghost-ftp:0.0.5` is a release distribution bundle, not a runtime container. Its build copies only the verified release directory and must not contain source worktrees, user data or protected release secrets.

## Security testing

The exact 0.0.5 candidate is expected to pass:

```text
go test -race ./...
go vet ./...
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_dependencies.py
python scripts/audit_repository.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

Dedicated tests additionally cover host validation, SFTP fingerprints, private-key handling, FTP/FTPS protocol behavior, connection lifecycle, transfer staging/cleanup, process lifecycle, filesystem hardening, configuration recovery, protected-secret ownership and the Windows ARM64 universal-package contract. Exact-head CI separately proves Windows/Linux production builds, Linux distro packaging/install lifecycle, Android APK validation, macOS development-app validation and authentic maintained runtime evidence where defined.

## Reporting a vulnerability

Do not put real passwords, private keys, passphrases, server private data, signing credentials or notarization secrets into public issues. Provide the Ghost FTP version, operating system/platform, protocol, a synthetic/minimal reproduction and privacy-safe logs.

See [Support](SUPPORT.md), [Privacy](PRIVACY.md), [Signing](SIGNING.md) and [`../macos/README.md`](../macos/README.md) for the corresponding reporting and trust boundaries.
