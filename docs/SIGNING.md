# Ghost FTP signing and Windows trust

Ghost FTP **1.0.0 Stable** treats Windows signing state as explicit release metadata. The project never invents a publisher identity merely to make a release look signed.

## Two truthful production states

The Windows production job reports one of exactly two states:

```text
WINDOWS_AUTHENTICODE=signed
WINDOWS_AUTHENTICODE=unsigned
```

When a protected production Authenticode identity is configured, `BUILD-WINDOWS.ps1` signs the final Portable/Setup artifacts and the release job requires Windows to validate every produced signature before publication.

When no production identity is configured, the same verified artifacts may be published **explicitly unsigned**. `BUILD-METADATA.txt` then records:

```text
WINDOWS_AUTHENTICODE=unsigned
WINDOWS_TRUST_MODE=sha256+github-release-provenance
```

An unsigned artifact is never described as publisher-signed. Users should verify `SHA256.txt`, the official GitHub Release/tag and source revision before installation.

## Why the workflow does not self-sign production releases

A development/self-signed certificate proves that signing plumbing works; it does not prove BRENDIGO LTD publisher identity to another machine. Ghost FTP therefore never fabricates, embeds or substitutes a self-signed development identity for a missing protected production certificate.

CI may still create a short-lived development certificate solely for the **Authenticode pipeline smoke test**. That certificate and private key are ephemeral test fixtures and are never release publisher credentials.

## Protected production secrets

When production signing is enabled, the release job consumes only protected GitHub Actions secrets:

```text
GHOSTFTP_SIGNING_PFX_BASE64
GHOSTFTP_SIGNING_PASSWORD
GHOSTFTP_SIGNING_TIMESTAMP_URL
```

If a PFX is configured without its password, or the PFX payload is malformed, the job fails closed. Private signing material must never be committed to the repository or copied into Release/GHCR artifacts.

## Runner lifetime

The Windows job decodes protected PFX material into the runner temporary directory only. The path is supplied to the build/signing step and removed by an `always()` cleanup operation. The repository stores signing integration code, never the production private key.

## Build ordering

`BUILD-WINDOWS.ps1` finalizes PE resources/payload before final signing. Portable artifacts are finalized/signed before Setup embeds the payload; Setup is signed only after its own payload/resources are complete. Release checksums are generated after final artifact mutation so `SHA256.txt` describes the exact published bytes.

## Signed-state verification

If `WINDOWS_AUTHENTICODE=signed`, the production job checks each Windows Setup/Portable file with the operating-system Authenticode verification API. A configured signing identity whose output does not verify as valid fails publication.

End users can inspect a signed artifact with:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-1.0.0-Setup-x64.exe | Format-List Status,StatusMessage,SignerCertificate,TimeStamperCertificate
```

A signed-state release should report a valid signature under the local Windows trust policy and must also match `SHA256.txt`.

## Unsigned-state verification

If `WINDOWS_AUTHENTICODE=unsigned`, do not expect `Get-AuthenticodeSignature` to report a trusted signer. Instead verify:

1. the file came from the official `ghostftp-v1.0.0` GitHub Release;
2. the file SHA-256 equals the matching entry in `SHA256.txt`;
3. `BUILD-METADATA.txt` names the expected version/source commit and explicitly says `WINDOWS_AUTHENTICODE=unsigned`;
4. the Release is `prerelease=false` and the tag resolves to that same source commit.

This proves release integrity/provenance within the documented GitHub distribution boundary; it does **not** turn an unsigned binary into a publisher-signed binary.

## Timestamping

When a production timestamp URL is configured, signing uses it according to the build policy. Timestamp traffic exists only in the release CI runner and is not application telemetry.

## Linux integrity

Linux DEB packages do not inherit Windows Authenticode. Their release integrity is established through exact source/version provenance, DEB metadata checks, the Release `SHA256.txt` manifest and the verified GHCR distribution-bundle digest.

## GitHub Packages

The GHCR distribution bundle contains only already-built verified release files. Signing secrets are not part of the package context. `BUILD-METADATA.txt` carries the truthful Windows signing/trust state so mirrors and automation can make policy decisions without guessing.

## Policy invariant

Ghost FTP never fabricates or mislabels cryptographic state. If future protected production signing is configured, the same workflow automatically moves from explicit unsigned provenance to verified Authenticode + SHA-256 provenance without changing the application binary contract.

See [Release verification](RELEASE-VERIFICATION.md), [GitHub Releases](GITHUB-RELEASES.md), [Security](SECURITY.md) and [Packages](PACKAGES.md).
