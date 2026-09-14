# Ghost FTP security

Ghost FTP **0.0.6** uses explicit transport, identity, path, secret, process and release boundaries. Security-sensitive behavior is tested in the shared engine, native platform adapters, Web FTP contract tests and protected release workflow.

## Transport policy

- **FTP** is an intentional unencrypted compatibility mode.
- **Explicit FTPS** protects FTP with TLS and must validate certificate + hostname identity. Secure failure is not silently retried as FTP.
- **Desktop SFTP** requires strict SSH host-key trust/pinning.
- **Android SFTP** remains intentionally hidden until strict host-key identity verification has a maintained Android implementation.
- **Web SFTP** requires an expected SHA-256 server host-key fingerprint before password authentication.

## Input and path boundaries

Hosts, ports, control characters, remote paths and destructive filesystem operations are validated before use. Desktop local destructive operations preserve root/symlink/reparse protections. Android local authority remains scoped through the Storage Access Framework.

Web FTP normalizes remote paths under the configured root and rejects control characters. FTP command paths cannot contain CR/LF command injection characters.

## Transfer lifecycle

Transfers are lifecycle-owned operations rather than blind copies. Maintained tests cover staged commit/rollback, upload source snapshots, cancellation before final activation, connection-generation ownership, stale-completion rejection and cleanup after failure.

## FTPS trust

Desktop FTPS follows strict certificate/hostname validation. Web FTP uses PHP cURL explicit TLS with both peer verification and hostname verification enabled. It preserves the original hostname for TLS verification while pinning the validated DNS result for the connection.

## SFTP trust

A changed SSH host key is an identity event. Desktop users must verify fingerprints through a trusted channel before accepting an intentional rotation.

Web FTP accepts only an expected `SHA256:...` fingerprint. It obtains the host key with `ssh2_fingerprint`, compares using `hash_equals`, and performs password authentication only after the fingerprint matches.

## Web SSRF protection

A public Web FTP deployment can otherwise become an SSRF primitive. The supplied source therefore blocks private, loopback, link-local and reserved IP destinations by default. All DNS results are checked. FTP/FTPS requests pin the accepted address using cURL `CURLOPT_RESOLVE`; SFTP connects to the accepted address and verifies host-key identity.

Private-target access is opt-in through `GHOSTFTP_WEB_ALLOW_PRIVATE=1` and should only be enabled on a trusted private deployment.

## Web credential boundary

The supplied web client has no account database and no durable credential store. Connection values live in page memory and are submitted over HTTPS with the requested operation. The PHP application does not intentionally log or persist them.

The web server necessarily handles those credentials in memory while executing a request. Reverse proxies, PHP-FPM, hosting providers and operating systems must be configured not to log request bodies. HTTPS is mandatory for a real deployment.

## Saved credentials on native platforms

Credential persistence is opt-in and platform-local. Windows uses the current-user protection boundary; Linux uses maintained local protected-secret handling; macOS development uses native Keychain-backed protection; Android current saved-site state remains non-secret.

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

The second marker is a deliberate trust boundary: structural/cross-build verification is not relabeled as native ARM64 runtime evidence.

Official publication is signed-only. `GHOSTFTP_SIGNING_PFX_BASE64` and `GHOSTFTP_SIGNING_PASSWORD` must be available to the protected release job and `Get-AuthenticodeSignature` must report `Valid` for both public EXEs.

## Android signing security

The public APK is built from exact release source and must be signed with the protected production keystore. `apksigner verify --verbose --print-certs` must succeed and the signer certificate SHA-256 must exactly match `GHOSTFTP_ANDROID_CERT_SHA256`. CI-smoke/debug identities are never production substitutes.

## Browser helper security

Chrome, Edge, Firefox and Opera helpers have zero browser permissions and zero host permissions, no remote code, no telemetry backend and no supported browser-to-desktop handoff. They parse/copy explicitly entered targets locally and do not become a protocol engine.

## Release supply chain

The release workflow pins actions, disables Go telemetry/external module resolution, runs exact-head quality gates, requires Windows/Android production signing, assembles only the canonical **13 platform artifacts / 16 public files**, writes SHA-256 metadata, refuses tag/release rewrites and reads published assets back before treating publication as valid.

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

The Python suite includes `scripts/test_web_contract.py`; on maintained Linux runners with PHP CLI it also executes the Web FTP source audit and PHP syntax lint.

## Reporting a vulnerability

Never put real passwords, private keys, passphrases, server private data, signing credentials or notarization secrets into a public issue. Provide the Ghost FTP version, platform/protocol, a synthetic reproduction and privacy-safe logs.

See [Privacy](PRIVACY.md), [Web](WEB.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Support](SUPPORT.md).
