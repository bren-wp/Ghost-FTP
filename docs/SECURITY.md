# Ghost FTP security

Ghost FTP **0.0.4** uses explicit transport, path, secret, process and release boundaries. Security-sensitive behavior is implemented in typed code and covered by platform-specific regression tests, exact-head native builds and repository audits.

## Supported transport security

Ghost FTP supports FTP, FTPS and SFTP on the maintained desktop engine.

- Plain FTP is an unencrypted compatibility mode and must not be confused with a secure transport.
- Fresh/quick-connect defaults use explicit **FTPS on port 21** on Windows and Linux.
- FTPS uses TLS protection and does not silently downgrade a failed secure request to plain FTP.
- SFTP uses SSH semantics and enforces host-key trust/fingerprint validation before a server is treated as trusted.

Connection profiles are validated before use: host, port, protocol, remote path, credential fields and key/fingerprint inputs pass through bounded validation logic.

The Android development client preserves the same no-silent-downgrade and strict FTPS certificate/hostname-verification intent while using platform-native storage/activity boundaries. Its APK is development evidence in 0.0.4, not a public release artifact.

## SFTP host-key trust

SFTP host-key fingerprints are normalized and validated by the shared security layer. A changed/unexpected key is an identity problem, not a harmless connectivity warning. Users should verify the new fingerprint through an independent trusted channel before accepting an intentional server-key rotation.

Private-key authentication validates local key paths and keeps passphrases out of durable plaintext profile fields.

### Protected-secret ownership

Linux runtime protected secrets use explicit ownership semantics. Session-owned password/passphrase handles are forgotten when the session closes, while borrowed profile-owned handles remain available to the profile store. Constructor/setup failure paths clean newly owned secrets.

Pending host-key trust state follows the same rule: owned temporary credentials are cleaned on cancel, expiry, mismatch, replacement or abandoned setup, and successful confirmation transfers ownership only when the exact protected blob is accepted by the SFTP session. A credential captured for the actual trust attempt is not silently replaced by stale profile state unless the user explicitly supplies a new value.

## FTPS certificate trust

FTPS relies on normal certificate/hostname verification for the selected server. The application does not ship a general “trust everything” mode for production use. A TLS failure is surfaced as a connection error rather than retried through a weaker transport.

Regression coverage explicitly verifies that an FTPS request aimed at a plaintext-only FTP endpoint fails instead of producing a plain FTP session.

## Input and path validation

Untrusted values are bounded before use. The maintained validators cover host names/IP addresses, ports, control characters, remote file paths, local containment and destructive filesystem operations.

Remote/local tree operations are designed to avoid traversal through unsafe paths. Local recursive deletion includes symlink/reparse-aware protections so a selected tree cannot silently escape its intended root.

Android local-file access remains Storage Access Framework based rather than introducing arbitrary filesystem-path authority. Saved Android navigation state remains account/endpoint bound where remote identity matters.

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

The goal is fail-closed behavior when the source/destination identity changes while an operation is in flight. Android staged downloads preserve the same principle: incomplete or cancelled work must not be promoted to the final user-visible object.

## Process execution boundary

Some FTP/SFTP functionality uses explicitly detected system transfer tools. Process construction, environment handling, tool capability probing and lifecycle are covered by regression tests. Credentials are not intentionally placed into user-visible command output or persisted runtime credential files.

Tool availability is diagnosed; the application does not silently download replacement networking tools.

On Linux, `curl`, `ssh`, `sftp` and `ssh-keyscan` are accepted only through a root-owned executable and directory/symlink provenance chain that is not writable by group or other users. `PATH` is a discovery hint, not a trust boundary.

Linux OpenSSH AskPass adds a second executable-identity boundary. The Ghost FTP helper path used for password/private-key-passphrase delivery must have the same trusted root-controlled provenance and must identify the same inode as the already-running Ghost FTP image. `/proc/self/exe` is used only inside Ghost FTP as an identity oracle; it is never handed to OpenSSH as `SSH_ASKPASS`, because process-memory hardening may make another same-UID process unable to dereference that procfs path.

The immediate AskPass parent must also resolve to a trusted root-controlled `ssh` or `sftp` executable. If the running Ghost FTP path is user-writable, such as a directly extracted Portable/per-user copy, Ghost FTP continues to run but does not construct a credential-bearing AskPass environment. The failure occurs before the AskPass token is generated and before the OpenSSH child is started, so password/passphrase capability material is not exposed merely to preserve convenience on a mutable helper path.

## Saved credential protection

Saved credentials are opt-in. Windows and Linux profile-save flows use explicit credential-persistence consent before a newly entered password or private-key passphrase is persisted.

### Windows

Protected profile secrets use the current-user Windows protection boundary. Sensitive runtime values are not intended to be serialized as plaintext profile fields.

### Linux

Saved secrets use local authenticated encryption with user-private key material. Runtime session secret handles are process-local and ownership-aware; session-only credentials are not promoted into persistent state merely because a connection was attempted.

If protected data cannot be safely decrypted, Ghost FTP should require the user to re-enter the secret rather than falling back to plaintext persistence.

On both desktop platforms, changing endpoint/account/private-key identity must not silently carry an existing protected credential into the changed profile identity.

## Runtime secret minimization

Runtime secrets are kept only as long as required for the selected operation. Diagnostic/error classification is deliberately separated from secret values. Tests cover privacy-safe error reporting, profile-secret binding and owned/borrowed secret lifetime.

## Settings/profile durability

Settings and profiles use local persistence with replacement/recovery behavior rather than unbounded append logs. Validation runs again when data is loaded. Malformed or invalid state must not become trusted merely because it came from a local file.

Bookmarks persist navigation metadata only; they are not a second credential store. Remote bookmark activation is revalidated against the active account/session before navigation becomes authoritative.

## Network privacy boundary

Ghost FTP has no application telemetry service, ad SDK or account backend. Production CI and release jobs explicitly disable Go telemetry. Network activity is user-directed transport traffic plus the selected server diagnostics required to operate the chosen protocol.

## Release supply-chain security

The production/release-validation workflows:

- pin GitHub Actions to exact revisions;
- disable Go telemetry and external Go module resolution;
- run race tests, vet and security/privacy/dependency audits;
- build Windows/Linux artifacts from exact source;
- lint/build/verify the Android development APK from exact source;
- run exact-head authentic Windows/Linux/Android UI evidence without allowing the evidence workflow to commit or push into the tested branch;
- optionally sign Windows artifacts with a protected trusted Authenticode identity when configured;
- verify every configured production signature and never label unsigned artifacts as signed;
- never generate a self-signed production publisher identity;
- remove temporary signing material from the runner when signing is used;
- assemble only an explicit Windows/Linux public release file set;
- record the Windows signing state in `BUILD-METADATA.txt`;
- generate SHA-256 checksums;
- prevent an existing version tag from being rewritten to another commit;
- verify the published GitHub Release asset set and current `prerelease=false` state;
- publish the current GHCR release bundle only from the verified release directory;
- verify the registry artifact can be read back;
- permit latest-only cleanup only after the newly published release has been fully verified.

Private signing material must never be committed to source. Absence of a production code-signing certificate is represented truthfully as an unsigned Windows release rather than “fixed” with an untrusted generated key.

Android remains outside the 0.0.4 public release allow-list. A development APK succeeding in CI is not authorization to publish it as a production mobile release.

## GitHub Packages boundary

The current package at `ghcr.io/bren-wp/ghost-ftp` is a release distribution bundle, not a runtime container. Its build uses `FROM scratch`, copies only the verified release directory and disables Docker networking during build. It must not contain source worktrees, user data or protected release secrets.

## Security testing

The exact 0.0.4 candidate is expected to pass:

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

Dedicated Go tests additionally cover host validation, SFTP fingerprints, private-key handling, FTP/FTPS protocol behavior, `remote.Manager.Connect()` lifecycle, transfer staging/cleanup, process lifecycle, filesystem hardening, configuration recovery and protected-secret ownership. Exact-head CI separately proves Windows/Linux production builds, Linux distro packaging/install lifecycle, Android APK validation and authentic cross-platform runtime evidence.

## Reporting a vulnerability

Do not put real passwords, private keys, passphrases, server private data or signing secrets into public issues. Provide the Ghost FTP version, operating system, protocol, a synthetic/minimal reproduction and privacy-safe logs.

See [Support](SUPPORT.md) for reporting guidance and [Privacy](PRIVACY.md) for data-handling guarantees.
