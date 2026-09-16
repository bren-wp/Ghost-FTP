# Ghost FTP security

Ghost FTP **0.0.6** uses explicit transport, identity, path, secret, process and release boundaries. Security-sensitive behavior is tested in the shared engine, native platform adapters, browser-helper contracts and protected release workflow.

## Transport policy

- **FTP** is an intentional unencrypted compatibility mode.
- **Explicit FTPS** protects FTP with TLS and must validate certificate + hostname identity. Secure failure is not silently retried as FTP.
- **Desktop SFTP** requires strict SSH host-key trust/pinning.
- **Android SFTP** remains intentionally hidden until strict host-key identity verification has a maintained Android implementation.
- Browser helpers are local utilities only and do not implement network transport.

## Input and path boundaries

Hosts, ports, control characters, remote paths and destructive filesystem operations are validated before use. Desktop local destructive operations preserve root/symlink/reparse protections. Android local authority remains scoped through the Storage Access Framework.

## Transfer lifecycle

Transfers are lifecycle-owned operations rather than blind copies. Maintained tests cover staged commit/rollback, upload source snapshots, cancellation before final activation, connection-generation ownership, stale-completion rejection and cleanup after failure.

## FTPS trust

Desktop FTPS follows strict certificate and hostname validation. A secure FTPS failure is not silently downgraded to plain FTP.

## SFTP trust

A changed SSH host key is an identity event. Desktop users must verify fingerprints through a trusted channel before accepting an intentional rotation.

## Saved credentials on native platforms

Credential persistence is opt-in and platform-local. Windows uses the current-user protection boundary; Linux uses maintained local protected-secret handling; macOS uses native Keychain-backed protection; Android current saved-site state remains non-secret.

Changing endpoint/account/key identity must not silently carry an existing protected credential into the changed profile identity.

## Linux external-tool trust

Linux accepts `curl`, `ssh`, `sftp` and `ssh-keyscan` only through trusted root-controlled executable/directory provenance. Credential-bearing OpenSSH AskPass requires both trusted Ghost FTP helper identity and trusted immediate ssh/sftp parent identity; mutable Portable/per-user paths fail closed rather than weakening that boundary.

## Windows universal package security

Public Windows output is exactly `Ghost-FTP-0.0.6-Setup.exe` and `Ghost-FTP-0.0.6-Portable.exe`. Each carries internal **x64, x86 and ARM64** payloads selected locally. Staged payload bytes are verified before execution and no architecture payload is fetched from the network.

The release evidence metadata must retain these exact markers:

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

Official publication is signed-only. `GHOSTFTP_SIGNING_PFX_BASE64` and `GHOSTFTP_SIGNING_PASSWORD` must be available to the protected release job and `Get-AuthenticodeSignature` must report `Valid` for both public EXEs.

## Android signing security

The public APK is built from exact release source and must be signed with the protected production keystore. `apksigner verify --verbose --print-certs` must succeed and the signer certificate SHA-256 must exactly match `GHOSTFTP_ANDROID_CERT_SHA256`. Test-only signing identities are never production substitutes.

## Browser helper security

Chrome, Edge, Firefox and Opera helpers have zero browser permissions and zero host permissions, no remote code and no telemetry backend. They parse explicitly entered targets locally and never become a protocol engine.

The explicit Windows desktop handoff uses only `ghostftp://connect` with an allowlist of protocol, host, optional port, optional username and optional remote path. The desktop parser rejects unknown or duplicated fields, URL userinfo, fragments, invalid schemes/ports, unsafe host/path data and control characters. Passwords, passphrases, private keys and arbitrary command arguments are not represented in the handoff contract.

When Ghost FTP is already running, the secondary process forwards the same validated payload to the primary process through local Windows `WM_COPYDATA`. The receiver re-validates the payload before use, does not open a socket or write a handoff file, does not auto-connect, and leaves credentials empty for explicit entry in the native client.

## Retired web surface

The product website and Web FTP implementation have been removed from the repository. Their source, workflow, contract tests and documentation are treated as retired surfaces and automated audits block their reintroduction.

## Release supply chain

The release workflow pins actions, disables Go telemetry/external module resolution, runs exact-head quality gates, requires Windows/Android production signing, assembles only the canonical **13 platform artifacts / 16 public files**, writes SHA-256 metadata, refuses tag/release rewrites and reads published assets back before treating publication as valid.

A public macOS artifact remains outside the release contract until real Developer ID Application signing and Apple notarization succeed.

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

Never put real passwords, private keys, passphrases, server private data, signing credentials or notarization secrets into a public issue. Provide the Ghost FTP version, platform/protocol, a synthetic reproduction and privacy-safe logs.

See [Privacy](PRIVACY.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Support](SUPPORT.md).
