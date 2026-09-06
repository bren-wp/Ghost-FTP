# Ghost FTP signing

Ghost FTP **1.1.1 Stable** supports Windows Authenticode signing as an optional production hardening layer. Signing improves publisher identity and Windows trust UX, but Ghost FTP does not fabricate a trusted identity when no real code-signing certificate is configured.

## Production policy

A stable Windows release may be published either:

- **signed** — when the protected GitHub Actions environment provides a real trusted Authenticode certificate and the produced Setup/Portable binaries verify successfully; or
- **unsigned** — when no production signing certificate is configured.

The release workflow reports the exact state through `WINDOWS_SIGNING_STATE`, and `BUILD-METADATA.txt` records it as:

```text
WINDOWS_AUTHENTICODE=signed
```

or:

```text
WINDOWS_AUTHENTICODE=unsigned
```

Unsigned publication is never relabeled as signed.

## Why Ghost FTP does not generate a production key automatically

A locally generated/self-signed certificate can prove that the signing code path works, but it does **not** create a publicly trusted Windows publisher identity. Automatically generating such a key and presenting it as production Authenticode would be misleading and would not provide the normal trust benefit expected from a CA-issued code-signing certificate.

Therefore the production workflow never creates its own long-lived publisher key. The separate CI smoke test may create a short-lived development certificate only to test signing mechanics.

## Optional protected secrets

If a real production certificate is available, signing uses protected GitHub Actions secrets:

```text
GHOSTFTP_SIGNING_PFX_BASE64
GHOSTFTP_SIGNING_PASSWORD
GHOSTFTP_SIGNING_TIMESTAMP_URL
```

If the PFX and password are absent, Windows artifacts remain unsigned and publication continues with explicit unsigned metadata. If one secret is supplied without its required counterpart, the workflow fails rather than silently guessing a signing state.

Private key material must never be committed to the repository, written into release metadata or copied into GitHub Packages.

## Runner lifetime

When signing is configured, the Windows release job decodes the protected PFX only into the runner temporary directory, exposes the path through the job environment for the build/signing step and removes the temporary file in an `always()` cleanup step.

The repository stores only the signing integration code, not the production private key.

## Build ordering

`BUILD-WINDOWS.ps1` signs the Portable executable before it is embedded into Setup, then signs Setup after its payload/resources are finalized. Checksums are generated after artifact mutation/signing so the published hashes represent final bytes.

When no signing identity is configured, the same deterministic build/package path is used without the signing mutation.

## Verification

For a release whose metadata says `WINDOWS_AUTHENTICODE=signed`, each Windows Setup/Portable artifact is checked with the operating-system Authenticode verification API during the production build. A configured signing state whose produced file does not report a valid signature fails the job.

End users can inspect a downloaded 1.1.1 artifact with:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-1.1.1-Setup-x64.exe | Format-List
```

If `BUILD-METADATA.txt` says `WINDOWS_AUTHENTICODE=unsigned`, an absent trusted signature is expected. In both cases, verify the file against `SHA256.txt` from the same official GitHub Release.

The candidate filename above does not prove that 1.1.1 has been published; release/tag/package read-back remains required before treating it as an official download.

## Windows warnings for unsigned builds

Unsigned Windows executables may produce SmartScreen or publisher warnings depending on Windows reputation and local policy. Ghost FTP does not recommend bypassing enterprise or operating-system security policy. A future CA-issued certificate can be added through protected secrets without changing the artifact naming/versioning contract.

## Development signing smoke test

CI can create a short-lived development code-signing certificate to validate the signing pipeline mechanically. That certificate is a test fixture and must never be represented as the trusted production publisher identity.

The development smoke test exists to catch broken signing scripts and PE-signature plumbing independently from production certificate availability.

## Timestamping

When a production timestamp URL is configured, signing uses it according to the build script policy. A timestamp service is not an application telemetry endpoint; it is used only during release signing by the CI runner.

## Linux integrity

Linux DEB packages do not use Windows Authenticode. Their direct GitHub Release integrity is established through exact source/release provenance, DEB metadata checks and `SHA256.txt`. A future repository-signing model would require its own protected key, rotation and revocation policy.

## Failure policy

Do not create or commit a self-signed production identity just to make a release appear signed. If a real Authenticode certificate is configured, verification is fail-closed. If it is not configured, publication remains truthfully unsigned and all release metadata/checksums continue to identify that state.

## Packages

The GHCR distribution bundle contains already-built verified release files. PFX/password material is never part of the bundle context. Package metadata records whether Windows artifacts were signed, but it never carries the signing key.

See [Release verification](RELEASE-VERIFICATION.md), [Security](SECURITY.md) and [Packages](PACKAGES.md).
