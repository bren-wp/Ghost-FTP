# Ghost FTP signing

Ghost FTP **0.0.5** uses Windows Authenticode as a required trust boundary for the official public Windows release workflow. Development builds and ordinary CI packaging may be unsigned, but the canonical `Publish Ghost FTP` workflow must not publish unsigned Setup or Portable executables.

Signing improves publisher identity and Windows trust UX. Ghost FTP never fabricates a trusted production identity when a real code-signing certificate is unavailable.

## Production publication policy

Official Windows publication is **signed-only**.

The canonical release workflow requires both protected production credentials:

```text
GHOSTFTP_SIGNING_PFX_BASE64
GHOSTFTP_SIGNING_PASSWORD
```

`GHOSTFTP_SIGNING_TIMESTAMP_URL` is optional and is used only when a production timestamp service is intentionally configured.

If the production PFX or its password is missing, the official release job fails before Windows artifacts can be published. There is no supported `state=unsigned` continuation path for `Publish Ghost FTP`.

A successful official release records:

```text
WINDOWS_AUTHENTICODE=signed
```

in `BUILD-METADATA.txt`.

## Development and ordinary CI builds

Local development builds and non-public CI packaging are allowed to remain unsigned. This keeps the build/test path usable without distributing production private-key material to ordinary jobs.

An unsigned development artifact is **not** an official public Windows release artifact. It must not be represented as publisher-signed or as proof that the production signing identity is configured.

## Protected secret handling

The production workflow reads signing material only from protected GitHub Actions secrets. The PFX is decoded into the runner temporary directory for the signing step and is removed in an `always()` cleanup step.

Private-key material must never be:

- committed to the repository;
- written into release metadata;
- uploaded as a workflow artifact;
- copied into the GHCR distribution bundle;
- printed to logs or error messages.

The repository stores only the signing integration code and public verification policy, never the production private key.

## Why Ghost FTP does not generate a production key automatically

A self-signed or locally generated certificate can exercise signing mechanics, but it does not establish a publicly trusted Windows publisher identity. Creating one automatically and treating it as production Authenticode would be misleading.

The production workflow therefore never creates its own long-lived publisher key. The separate CI signing smoke test may create a short-lived development certificate only to test the signing pipeline mechanically. That fixture is never a production identity and is never accepted by the public release workflow.

## Build ordering

`BUILD-WINDOWS.ps1` signs finalized Windows targets through the configured signing path. Public Setup and Portable artifacts are verified after their final byte mutations and before publication. Release hashes are generated from final artifact bytes.

The ordinary build path can still run without production credentials; the canonical public-release workflow adds the fail-closed signing requirement around that build.

## Verification before publication

For each public Windows artifact, the release job calls the operating-system Authenticode verification API and requires:

- a signer certificate to be present; and
- signature status `Valid`.

The two public Windows files for 0.0.5 are:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

The publish job also requires `WINDOWS_SIGNING_STATE=signed`. `scripts/verify_release.py` independently rejects unsigned public Setup/Portable artifacts when it runs inside `Publish Ghost FTP`.

This gives the release path two independent signing checks: Windows signature verification during artifact production and fail-closed release verification before publication.

## End-user verification

A downloaded official Windows artifact can be inspected with:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-0.0.5-Setup.exe | Format-List
Get-AuthenticodeSignature .\Ghost-FTP-0.0.5-Portable.exe | Format-List
```

For the current public-release contract, an official Windows file is expected to have a valid trusted Authenticode signature. A missing or invalid signature is a release-integrity failure, not an accepted official unsigned state.

Users should also verify the matching SHA-256 value from `SHA256.txt` in the same GitHub Release. Authenticode and SHA-256 serve different purposes: publisher trust and exact-byte integrity respectively.

## Timestamping

When `GHOSTFTP_SIGNING_TIMESTAMP_URL` is configured, the signing step uses that service according to the Windows build policy. A timestamp endpoint is release infrastructure used by the CI runner; it is not an application telemetry destination and the Ghost FTP client does not contact it at runtime.

## Linux and other development surfaces

Linux artifacts do not use Windows Authenticode. Their integrity is enforced through exact-source build provenance, package metadata checks, binary parity checks, release allow-list verification and `SHA256.txt`.

Android and macOS remain separately validated development/source surfaces under the 0.0.5 release contract and are not silently added to the current 17-file Windows/Linux public release allow-list.

## Failure policy

The official Windows release fails closed if:

- either required protected signing credential is absent;
- PFX decoding fails;
- signing fails;
- Setup or Portable lacks a signer certificate;
- Authenticode status is not `Valid`; or
- the publish job receives any signing state other than `signed`.

Do not weaken this policy by adding an unsigned-publication fallback, a self-signed production substitute or a metadata-only claim of signing.

## Packages

The GHCR distribution bundle contains already-built verified release files. Signing credentials are never part of its build context or payload. Release metadata records the verified public Windows signing state but never carries signing key material.

See [Release verification](RELEASE-VERIFICATION.md), [Security](SECURITY.md), [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
