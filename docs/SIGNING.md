# Ghost FTP signing

Ghost FTP **1.0.0 Stable** treats Windows code signing as a release security boundary, not a cosmetic badge.

## Stable requirement

A stable Windows release is blocked unless the protected GitHub Actions environment provides a trusted Authenticode identity and the resulting Setup/Portable binaries verify successfully.

The release workflow reports the signing state through `WINDOWS_SIGNING_STATE`. For `MAJOR >= 1`, the publish job requires:

```text
WINDOWS_SIGNING_STATE=signed
```

before the stable GitHub Release and GitHub Package publication can complete.

## Protected secrets

Production signing uses protected GitHub Actions secrets:

```text
GHOSTFTP_SIGNING_PFX_BASE64
GHOSTFTP_SIGNING_PASSWORD
GHOSTFTP_SIGNING_TIMESTAMP_URL
```

Private key material must never be committed to the repository, written into release metadata or copied into GitHub Packages.

## Runner lifetime

The Windows release job decodes the protected PFX only into the runner temporary directory, exposes the path through the job environment for the build/signing step and removes the temporary file in an `always()` cleanup step.

The repository stores only the signing integration code, not the production private key.

## Build ordering

`BUILD-WINDOWS.ps1` signs the Portable executable before it is embedded into Setup, then signs Setup after its payload/resources are finalized. Checksums are generated after artifact mutation/signing so the published hashes represent final bytes.

## Verification

During production build, each Windows Setup/Portable artifact is checked with the operating-system Authenticode verification API. A configured signing state whose produced file does not report a valid signature fails the job.

End users can inspect a downloaded artifact with:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-1.0.0-Setup-x64.exe | Format-List
```

Also verify the file against `SHA256.txt` from the same official release.

## Development signing smoke test

CI can create a short-lived development code-signing certificate to validate the signing pipeline mechanically. That certificate is a test fixture and must never be represented as the trusted production publisher identity.

The development smoke test exists to catch broken signing scripts and PE-signature plumbing before the protected production job.

## Timestamping

When the production timestamp URL is configured, signing uses it according to the build script policy. A timestamp service is not an application telemetry endpoint; it is used only during release signing by the CI runner.

## Linux integrity

Linux DEB packages do not inherit Windows Authenticode. Their direct GitHub Release integrity is established through exact source/release provenance, DEB metadata checks and `SHA256.txt`. A future repository-signing model would require its own protected key, rotation and revocation policy.

## Failure policy

Do not weaken the stable gate because signing secrets are unavailable. Fix the protected release configuration and rerun the exact release candidate. A stable release must not be made “green” by relabeling an unsigned binary as signed.

## Packages

The GHCR distribution bundle contains already-built verified release files. The PFX/password are not part of the bundle context. Package metadata may state whether Windows artifacts were signed, but it never carries the signing key.

See [Release verification](RELEASE-VERIFICATION.md), [Security](SECURITY.md) and [Packages](PACKAGES.md).
