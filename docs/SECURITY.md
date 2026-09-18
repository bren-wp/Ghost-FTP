# Security

Ghost FTP security policy applies to the active Windows, Linux and Android applications.

## Protocol trust

- FTPS must verify TLS certificates and hostnames.
- Desktop SFTP must preserve strict host-key trust/pinning.
- Android SFTP remains hidden until an equivalent maintained strict host-key verification boundary exists.
- UI work must never disable or bypass protocol verification.

## Credentials

Credential persistence is opt-in and platform-local where implemented. Windows uses the maintained current-user protection boundary, Linux uses maintained protected-secret handling and provenance checks, and Android saved-site state remains intentionally non-secret.

Secrets must not be embedded in screenshots, browser helpers, logs, release metadata or source control.

## File and transfer safety

Paths must be validated before mutation, local/remote ownership stays explicit, destructive operations fail closed, and transfer identity must remain stable across queue actions.

## Release security

Official releases use exact-source build artifacts, checksums and platform-appropriate signing gates. Missing signing credentials must fail the protected publication path rather than create a falsely trusted artifact.

macOS signing/notarization is no longer part of the active product because macOS support is retired.


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. Current hosted CI does not claim native Windows-on-ARM runtime execution evidence.
