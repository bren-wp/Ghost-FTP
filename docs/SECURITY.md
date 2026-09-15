# Ghost FTP security

Ghost FTP **0.0.6** uses explicit transport, identity, path, secret, process and release boundaries. Security-sensitive behavior is verified in the shared engine, native platform adapters and protected release workflows.

## Transport policy

- **FTP** is an intentional unencrypted compatibility mode.
- **Explicit FTPS** protects FTP with TLS and validates certificate and hostname identity. Secure failure is not silently retried as plain FTP.
- **Desktop SFTP** requires strict SSH host-key trust and pinning.
- **Android SFTP** remains intentionally hidden until strict host-key identity verification has a maintained Android implementation.

## Input and path boundaries

Hosts, ports, control characters, remote paths and destructive filesystem operations are validated before use. Desktop local destructive operations preserve root, symlink and reparse-point protections. Android local authority remains scoped through the Storage Access Framework.

## Transfer lifecycle

Transfers are lifecycle-owned operations rather than blind copies. Maintained tests cover staged commit/rollback, upload source snapshots, cancellation before final activation, connection-generation ownership, stale-completion rejection and cleanup after failure.

Downloads remain rooted in an already-open local filesystem capability where applicable, and staged files are revalidated before activation. Recursive deletion is designed to avoid following swapped symlinks, junctions or reparse points outside the selected root.

## FTPS trust

FTPS uses certificate and hostname validation and must fail closed on trust errors. The application does not silently downgrade a failed secure connection to plain FTP.

## SFTP trust

A changed SSH host key is an identity event. Desktop users must verify fingerprints through a trusted channel before accepting an intentional rotation. Proxy/forwarding behavior is constrained and authentication helpers are isolated from unsafe environment inheritance.

## Saved credentials

Credential persistence is opt-in and platform-local. Windows uses the current-user protection boundary; Linux uses maintained protected-secret handling; macOS development uses native Keychain-backed protection; Android saved-site state remains non-secret.

Changing endpoint, account or private-key identity must not silently carry a protected credential into a different profile identity.

## Linux external-tool trust

Linux accepts required external transfer/SSH tools only through maintained executable provenance rules. Credential-bearing OpenSSH AskPass requires trusted helper identity and trusted immediate ssh/sftp parent identity. User-writable helper substitution fails closed.

## Windows universal package security

Public Windows output is exactly `Ghost-FTP-0.0.6-Setup.exe` and `Ghost-FTP-0.0.6-Portable.exe`. Each carries internal **x64, x86 and ARM64** payloads selected locally. Staged payload bytes are verified before execution and no architecture payload is fetched from the network.

Official publication is signed-only. Production signing material is supplied through protected release secrets and must not be committed to source or silently replaced with development credentials.

## Android signing security

The public APK is built from exact release source and must be signed with the protected production keystore. `apksigner verify --verbose --print-certs` must succeed and the signer certificate SHA-256 must exactly match `GHOSTFTP_ANDROID_CERT_SHA256`. CI-smoke/debug identities are never production substitutes.

## Browser helper security

Chrome, Edge, Firefox and Opera helpers use a deliberately narrow capability model. They contain no remote executable code, telemetry backend or credential store and have **no supported browser-to-desktop** launch/handoff behavior.

## Release supply chain

The release workflow pins actions, disables Go telemetry/external module resolution, runs exact-head quality gates, requires Windows/Android production signing, assembles only the canonical **13 platform artifacts / 16 public files**, writes SHA-256 metadata and verifies published assets before publication is treated as valid.

macOS remains outside the public release until real Developer ID Application signing and Apple notarization succeed.

## Security testing

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

## Reporting a vulnerability

Never put real passwords, private keys, passphrases, server private data, signing credentials or notarization secrets into a public issue. Provide the Ghost FTP version, platform/protocol, a synthetic reproduction and privacy-safe diagnostics.

See [Privacy](PRIVACY.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Support](SUPPORT.md).
