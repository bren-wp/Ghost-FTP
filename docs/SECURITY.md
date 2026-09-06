# Ghost FTP security

Ghost FTP **1.1.1 Stable** uses explicit transport, path, secret, process and release boundaries. Security-sensitive behavior is implemented in typed Go code and covered by platform-specific regression tests plus repository audits.

## Supported transport security

Ghost FTP supports FTP, FTPS and SFTP.

- Plain FTP is an unencrypted compatibility mode and must not be confused with a secure transport.
- Fresh/quick-connect defaults use explicit **FTPS on port 21** on Windows and Linux.
- FTPS uses TLS protection and does not silently downgrade a failed secure request to plain FTP.
- SFTP uses SSH semantics and enforces host-key trust/fingerprint validation before a server is treated as trusted.

Connection profiles are validated before use: host, port, protocol, remote path, credential fields and key/fingerprint inputs pass through bounded validation logic.

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

## Transfer staging and commit safety

Transfers are treated as lifecycle operations rather than blind file copies. The maintained release includes tests for:

- upload-source snapshots;
- staged remote operations and cleanup;
- remote destination/commit revalidation;
- local rollback cleanup;
- transfer generation binding across reconnects;
- cancellation/failure terminal-state correctness;
- symlink-safe filesystem handling.

The goal is fail-closed behavior when the source/destination identity changes while an operation is in flight.

## Process execution boundary

Some FTP/SFTP functionality uses explicitly detected system transfer tools. Process construction, environment handling, tool capability probing and lifecycle are covered by regression tests. Credentials are not intentionally placed into user-visible command output or persisted runtime credential files.

Tool availability is diagnosed; the application does not silently download replacement networking tools.

## Saved credential protection

Saved credentials are opt-in. The main Save Profile flow and Windows Site Manager use the same explicit consent policy before a newly entered password or private-key passphrase is persisted.

### Windows

Protected profile secrets use the current-user Windows protection boundary. Sensitive runtime values are not intended to be serialized as plaintext profile fields.

### Linux

Saved secrets use local authenticated encryption with user-private key material. Runtime session secret handles are process-local and ownership-aware; session-only credentials are not promoted into persistent state merely because a connection was attempted.

If protected data cannot be safely decrypted, Ghost FTP should require the user to re-enter the secret rather than falling back to plaintext persistence.

## Runtime secret minimization

Runtime secrets are kept only as long as required for the selected operation. Diagnostic/error classification is deliberately separated from secret values. Tests cover privacy-safe error reporting, profile-secret binding and owned/borrowed secret lifetime.

## Settings/profile durability

Settings and profiles use local persistence with replacement/recovery behavior rather than unbounded append logs. Validation runs again when data is loaded. Malformed or invalid state must not become trusted merely because it came from a local file.

## Network privacy boundary

Ghost FTP has no application telemetry service, ad SDK or account backend. Production CI and release jobs explicitly disable Go telemetry. Network activity is user-directed transport traffic plus the selected server diagnostics required to operate the chosen protocol.

## Release supply-chain security

The production release workflow:

- pins GitHub Actions to exact revisions;
- disables Go telemetry and external Go module resolution;
- runs race tests, vet and security/privacy/dependency audits;
- builds Windows/Linux artifacts from exact source;
- optionally signs Windows artifacts with a protected trusted Authenticode identity when configured;
- verifies every configured production signature and never labels unsigned artifacts as signed;
- never generates a self-signed production publisher identity;
- removes temporary signing material from the runner when signing is used;
- assembles only an explicit release file set;
- records the Windows signing state in `BUILD-METADATA.txt`;
- generates SHA-256 checksums;
- prevents an existing version tag from being rewritten to another commit;
- verifies the published GitHub Release asset set and stable `prerelease=false` state;
- publishes the stable GHCR release bundle only from the verified `release/` directory;
- verifies the registry artifact can be read back.

Private signing material must never be committed to source. Absence of a production code-signing certificate is represented truthfully as an unsigned Windows release rather than “fixed” with an untrusted generated key.

## GitHub Packages boundary

The stable package at `ghcr.io/bren-wp/ghost-ftp` is a release distribution bundle, not a runtime container. Its build uses `FROM scratch`, copies only the verified release directory and disables Docker networking during build. It must not contain source worktrees, user data or protected release secrets.

## Security testing

The exact 1.1.1 candidate is expected to pass:

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

Dedicated Go tests additionally cover host validation, SFTP fingerprints, private-key handling, FTP/FTPS protocol behavior, `remote.Manager.Connect()` lifecycle, transfer staging/cleanup, process lifecycle, filesystem hardening, configuration recovery and protected-secret ownership.

## Reporting a vulnerability

Do not put real passwords, private keys, passphrases, server private data or signing secrets into public issues. Provide the Ghost FTP version, operating system, protocol, a synthetic/minimal reproduction and privacy-safe logs.

See [Support](SUPPORT.md) for reporting guidance and [Privacy](PRIVACY.md) for data-handling guarantees.
