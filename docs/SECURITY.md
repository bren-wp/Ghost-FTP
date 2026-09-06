# Ghost FTP security

Ghost FTP **1.0.0 Stable** uses explicit transport, path, secret, process and release boundaries. Security-sensitive behavior is implemented in typed Go code and covered by platform-specific regression tests plus repository audits.

## Supported transport security

Ghost FTP supports FTP, FTPS and SFTP.

- Plain FTP is an unencrypted compatibility mode and must not be confused with a secure transport.
- FTPS uses TLS protection and does not silently downgrade a failed secure request to plain FTP.
- SFTP uses SSH semantics and enforces host-key trust/fingerprint validation before a server is treated as trusted.

Connection profiles are validated before use: host, port, protocol, remote path, credential fields and key/fingerprint inputs pass through bounded validation logic.

## SFTP host-key trust

SFTP host-key fingerprints are normalized and validated by the shared security layer. A changed/unexpected key is an identity problem, not a harmless connectivity warning. Users should verify the new fingerprint through an independent trusted channel before accepting an intentional server-key rotation.

Private-key authentication validates local key paths and keeps passphrases out of durable plaintext profile fields.

## FTPS certificate trust

FTPS relies on normal certificate/hostname verification for the selected server. The application does not ship a general “trust everything” mode for production use. A TLS failure is surfaced as a connection error rather than retried through a weaker transport.

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

Some FTP/SFTP functionality uses explicitly detected system transfer tools. Process construction, environment handling, tool capability probing and lifecycle are covered by regression tests. Credentials are not intentionally placed into user-visible command output.

Tool availability is diagnosed; the application does not silently download replacement networking tools.

## Saved credential protection

Saved credentials are opt-in.

### Windows

Protected profile secrets use the current-user Windows protection boundary. Sensitive runtime values are not intended to be serialized as plaintext profile fields.

### Linux

Saved secrets use local authenticated encryption with user-private key material. Key/profile handling includes local permission and binding checks.

If protected data cannot be safely decrypted, Ghost FTP should require the user to re-enter the secret rather than falling back to plaintext persistence.

## Runtime secret minimization

Runtime secrets are kept only as long as required for the selected operation. Diagnostic/error classification is deliberately separated from secret values. Tests cover privacy-safe error reporting and profile-secret guards.

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
- requires protected trusted Authenticode for stable Windows publication;
- removes temporary signing material from the runner;
- assembles only an explicit release file set;
- generates SHA-256 checksums;
- prevents an existing version tag from being rewritten to another commit;
- verifies the published GitHub Release asset set and stable `prerelease=false` state;
- publishes the stable GHCR release bundle only from the verified `release/` directory;
- verifies the registry artifact can be read back.

Private signing material must never be committed to source.

## GitHub Packages boundary

The stable package at `ghcr.io/bren-wp/ghost-ftp` is a release distribution bundle, not a runtime container. Its build uses `FROM scratch`, copies only the verified release directory and disables Docker networking during build. It must not contain source worktrees, user data or protected release secrets.

## Security testing

The exact 1.0.0 candidate is expected to pass:

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

Dedicated Go tests additionally cover host validation, SFTP fingerprints, private-key handling, FTP protocol behavior, transfer staging/cleanup, process lifecycle, filesystem hardening and configuration recovery.

## Reporting a vulnerability

Do not put real passwords, private keys, passphrases, server private data or signing secrets into public issues. Provide the Ghost FTP version, operating system, protocol, a synthetic/minimal reproduction and privacy-safe logs.

See [Support](SUPPORT.md) for reporting guidance and [Privacy](PRIVACY.md) for data-handling guarantees.
