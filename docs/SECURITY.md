# Ghost FTP security

Ghost FTP **1.0.0 Stable** uses explicit transport, path, secret, process and release boundaries. Security-sensitive behavior is implemented in typed Go code and covered by platform-specific regression tests plus fail-closed repository audits.

## Supported transport security

Ghost FTP supports FTP, FTPS and SFTP.

- Plain FTP is an unencrypted compatibility mode and must not be confused with a secure transport.
- FTPS uses TLS protection and does not silently downgrade a failed secure request to plain FTP.
- SFTP uses SSH semantics and enforces host-key trust/fingerprint validation before a server is treated as trusted.

Connection profiles are validated before use: host, port, protocol, remote path, credential fields and key/fingerprint inputs pass through bounded validation logic.

## SFTP host-key trust

SFTP host-key fingerprints are normalized and validated by the shared security layer. A changed/unexpected key is an identity problem, not a harmless connectivity warning. Verify intentional server-key rotation through an independent trusted channel before trusting the new key.

Private-key authentication validates local key paths and keeps passphrases out of durable plaintext profile fields.

## FTPS certificate trust

FTPS relies on certificate/hostname verification for the selected server. Ghost FTP does not ship a general production “trust everything” mode. A TLS failure is surfaced as a secure connection failure rather than retried through a weaker transport.

## Input and path validation

Untrusted values are bounded before use. Maintained validators cover host names/IP addresses, ports, control characters, remote file paths, local containment and destructive filesystem operations.

Remote/local tree operations are designed to avoid traversal through unsafe paths. Local recursive deletion includes symlink/reparse-aware protections so a selected tree cannot silently escape its intended root.

## Transfer staging and commit safety

Transfers are lifecycle operations rather than blind file copies. Maintained regression coverage includes:

- upload-source snapshots;
- staged remote operations and cleanup;
- remote destination/commit revalidation;
- local rollback cleanup;
- transfer generation binding across reconnects;
- cancellation/failure terminal-state correctness;
- symlink-safe filesystem handling;
- truthful byte/progress/speed/ETA reporting.

The goal is fail-closed behavior when source/destination identity changes while an operation is in flight.

## Process execution boundary

FTP/SFTP functionality uses explicitly detected system transfer tools. Process construction, environment sanitization, capability probing and lifecycle are covered by regression tests. Credentials are not intentionally placed into user-visible command output.

Connection/tool errors are mapped to stable privacy-safe categories rather than exposing raw `curl`/OpenSSH diagnostics, password data or private-key paths in normal UI messages.

Ghost FTP does not silently download replacement networking tools at runtime.

## Saved credential protection

Saved credentials are opt-in.

### Windows

Protected profile secrets use the current-user Windows protection boundary. Sensitive runtime values are not intended to be serialized as plaintext profile fields.

### Linux

Saved secrets use local authenticated encryption with user-private key material. Key/profile handling includes local permission and binding checks.

If protected data cannot be safely decrypted, Ghost FTP requires the user to re-enter the secret rather than falling back to plaintext persistence.

## Runtime secret minimization

Runtime secrets are kept only as long as required for the selected operation. Diagnostic/error classification is deliberately separated from secret values. Tests cover privacy-safe error reporting and profile-secret guards.

## Settings/profile durability

Settings and profiles use local persistence with replacement/recovery behavior rather than unbounded append logs. Validation runs again when data is loaded. Malformed state does not become trusted merely because it came from a local file.

## Network privacy boundary

Ghost FTP has no application telemetry service, ad SDK or account backend. Production CI/release jobs explicitly disable Go telemetry. Network activity is user-directed transport traffic plus protocol operations required against the selected server.

## Release supply-chain security

The production release workflow:

- pins GitHub Actions to exact revisions;
- disables Go telemetry and external Go module resolution;
- runs race tests, vet and security/privacy/dependency audits;
- builds Windows/Linux artifacts from exact source;
- uses protected Authenticode credentials only when genuinely configured;
- verifies every configured production signature and otherwise records Windows artifacts explicitly as unsigned;
- never substitutes a self-signed/development certificate as a production publisher identity;
- removes temporary production signing material from the runner;
- assembles only the explicit 12-file Release allow-list;
- generates SHA-256 checksums after final binary mutation;
- prevents an existing version tag from being rewritten;
- requires stable GitHub Release `draft=false` and `prerelease=false`;
- performs immediate and delayed Release read-back including `SHA256.txt`;
- publishes GHCR only after the GitHub Release has been verified;
- builds GHCR from `FROM scratch` with networking disabled and only `release/` in the package payload;
- uses temporary Docker credentials and removes them after package publication;
- removes the local OCI image, pulls the semantic-version package back and verifies OCI labels plus embedded SHA/build metadata.

Private signing material must never be committed to source or included in public artifacts.

## Windows signed versus unsigned provenance

A signed release can combine Authenticode, SHA-256 and official GitHub provenance. An explicitly unsigned release uses `SHA256.txt`, immutable release tag/source commit and official GitHub provenance and must not be represented as publisher-signed.

The release metadata makes that distinction machine-readable through:

```text
WINDOWS_AUTHENTICODE=signed|unsigned
WINDOWS_TRUST_MODE=...
```

This allows enterprise policy to require Authenticode without forcing the project to fabricate a cryptographic identity when a production certificate is not configured.

## GitHub Packages boundary

`ghcr.io/bren-wp/ghost-ftp` is a release distribution bundle, not a runtime container. Its filesystem contains `/ghostftp-release/` only. It must not contain source worktrees, user data, runtime credentials, CI tokens or protected signing secrets.

## Security testing

The exact stable candidate is expected to pass:

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

Do not put real passwords, private keys, passphrases, confidential server data or signing secrets into public issues. Provide Ghost FTP version, operating system, protocol and a synthetic/minimal reproduction.

See [Support](SUPPORT.md), [Privacy](PRIVACY.md), [Signing](SIGNING.md) and [Release verification](RELEASE-VERIFICATION.md).
