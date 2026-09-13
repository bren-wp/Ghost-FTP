# Ghost FTP signing

Ghost FTP **0.0.5** separates public-release signing from ordinary development builds. The current public GitHub Release is Windows/Linux only: official Windows publication is Authenticode **signed-only**, while Linux release integrity is enforced through exact-source/package verification and SHA-256 metadata. macOS has a separate fail-closed Developer ID + Apple notarization path for production distribution, but macOS is not part of the current 17-file public release allow-list.

Signing policy is deliberately platform-specific. Ghost FTP never fabricates a trusted production identity when a real platform signing credential is unavailable.

## Windows production publication policy

Official Windows publication is **signed-only**.

The canonical `Publish Ghost FTP` workflow requires both protected production credentials:

```text
GHOSTFTP_SIGNING_PFX_BASE64
GHOSTFTP_SIGNING_PASSWORD
```

`GHOSTFTP_SIGNING_TIMESTAMP_URL` is optional and is used only when a production timestamp service is intentionally configured.

If the production PFX or its password is missing, the official release job fails before Windows artifacts can be published. There is no supported `state=unsigned` continuation path for `Publish Ghost FTP`.

A successful official release records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

in `BUILD-METADATA.txt`.

## Windows development and ordinary CI builds

Local development builds and non-public CI packaging are allowed to remain unsigned. This keeps the build/test path usable without distributing production private-key material to ordinary jobs.

An unsigned development artifact is **not** an official public Windows release artifact. It must not be represented as publisher-signed or as proof that the production signing identity is configured.

## Windows protected-secret handling

The production workflow reads signing material only from protected GitHub Actions secrets. The PFX is decoded into the runner temporary directory for the signing step and is removed in an `always()` cleanup step.

Private-key material must never be:

- committed to the repository;
- written into release metadata;
- uploaded as a workflow artifact;
- copied into the GHCR distribution bundle;
- printed to logs or error messages.

The repository stores only the signing integration code and public verification policy, never the production private key.

## Why Ghost FTP does not generate a Windows production key automatically

A self-signed or locally generated certificate can exercise signing mechanics, but it does not establish a publicly trusted Windows publisher identity. Creating one automatically and treating it as production Authenticode would be misleading.

The production workflow never creates its own long-lived publisher key. The separate CI signing smoke test may create a short-lived development certificate only to test the signing pipeline mechanically. That fixture is never a production identity and is never accepted by the public release workflow.

## Windows build and signing ordering

The canonical Windows build has two signing layers because the two public executables embed native application/installer payloads.

First, `BUILD-WINDOWS-ARCH-STAGE.ps1` builds native Setup and Portable staging pairs for:

```text
x64
x86
arm64
```

For each architecture, deterministic PE resources are applied before signing. The Portable executable is signed before `scripts/make_payload.py` creates the installer payload, so Setup embeds the same signed client bytes. The native Setup is then finalized, signed and verified. When production signing is configured, ARM64 follows the same protected signing path as x64 and x86.

Second, `BUILD-WINDOWS.ps1` embeds the already verified native x64/x86/ARM64 payloads into the public universal Setup and Portable bootstraps. PE resources are finalized on each public bootstrap and the **outer public executable is signed after its final byte mutation**.

Release hashes are generated only from final public artifact bytes.

Architecture-specific staging executables remain internal. Their signing is part of embedded-payload integrity; they do not become extra public release files.

## Windows verification before publication

For each public Windows artifact, the release job calls the operating-system Authenticode verification API and requires:

- a signer certificate to be present; and
- signature status `Valid`.

The two public Windows files for 0.0.5 are:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

No public `-x64.exe`, `-x86.exe`, `-x32.exe` or `-arm64.exe` alias is part of the current release.

The publish job also requires `WINDOWS_SIGNING_STATE=signed`. `scripts/verify_release.py` independently rejects unsigned public Setup/Portable artifacts when it runs inside `Publish Ghost FTP`.

This gives the release path two independent signing checks: Windows signature verification during artifact production and fail-closed release verification before publication.

## Windows ARM64 evidence boundary

Signing support is not the same as native runtime evidence. Current CI cross-builds and structurally verifies the ARM64 PE32+ payloads and runs them through the same signing mechanics, but the maintained Windows runner is not ARM64. Therefore the release metadata records:

```text
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

Do not reinterpret a successful ARM64 cross-build or valid signature as proof that the binary was executed natively on Windows ARM64 hardware. That claim requires separate maintained ARM64 runtime evidence.

## Windows end-user verification

A downloaded official Windows artifact can be inspected with:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-0.0.5-Setup.exe | Format-List
Get-AuthenticodeSignature .\Ghost-FTP-0.0.5-Portable.exe | Format-List
```

For the current public-release contract, an official Windows file is expected to have a valid trusted Authenticode signature. A missing or invalid signature is a release-integrity failure, not an accepted official unsigned state.

Users should also verify the matching SHA-256 value from `SHA256.txt` in the same GitHub Release. Authenticode and SHA-256 serve different purposes: publisher trust and exact-byte integrity respectively.

## Windows timestamping

When `GHOSTFTP_SIGNING_TIMESTAMP_URL` is configured, the signing step uses that service according to the Windows build policy. A timestamp endpoint is release infrastructure used by the CI runner; it is not an application telemetry destination and the Ghost FTP client does not contact it at runtime.

## macOS development signing

The maintained macOS development build is a universal Intel + Apple Silicon AppKit application using the shared `internal/api.Engine`. `macos/BUILD.sh` deliberately produces an **ad-hoc signed development artifact** for native build/regression validation.

An ad-hoc development signature is not a Developer ID signature and is not evidence of Apple notarization or public Gatekeeper distribution readiness. The macOS development artifact remains outside the current Windows/Linux **14 platform artifacts / 17 public files** release allow-list.

## macOS production signing and notarization

`macos/SIGN_AND_NOTARIZE.sh` is the fail-closed production-distribution path. It requires a real **Developer ID Application** identity, replaces ad-hoc signatures from nested code outward, enables Hardened Runtime and secure timestamping, validates the resulting signatures, submits the app through Apple notarization, requires acceptance, staples the ticket and performs Gatekeeper assessment before emitting the notarized ZIP.

The reusable local production path accepts Keychain references rather than raw signing material:

```bash
export MACOS_DEVELOPER_IDENTITY='Developer ID Application: Example Company (TEAMID)'
export MACOS_NOTARY_KEYCHAIN_PROFILE='ghostftp-notary'
bash macos/SIGN_AND_NOTARIZE.sh
```

`.github/workflows/macos-production.yml` is the separate manual environment-gated CI path. Its protected `macos-production` environment supplies the production certificate/notary credentials to an ephemeral Keychain and cleans temporary signing material in an `always()` path.

A missing Developer ID identity or Apple notarization credential is a production-macOS distribution failure. Ghost FTP does **not** generate a self-signed replacement, downgrade the requirement or relabel an ad-hoc development app as a notarized production artifact.

A successful macOS development build alone must never be documented as proof that Developer ID signing or Apple notarization succeeded. Production distribution can be claimed only after the credentialed production path actually completes successfully.

The production macOS workflow deliberately does not mutate the current public GitHub Release. Adding macOS to the public release requires a separate explicit release-contract change; it is not implied by source parity or by availability of the production signing workflow.

## Linux release integrity

Linux artifacts do not use Windows Authenticode or Apple Developer ID signing. Their current release integrity is enforced through exact-source build provenance, package metadata checks, binary parity checks, explicit release allow-list verification and `SHA256.txt`.

Canonical Linux package families remain Debian, Ubuntu, Fedora and Portable. Linux is part of the current 17-file Windows/Linux public release contract.

## Android development signing

The Android APK produced by current CI is a development/debug-signed artifact for installation and validation. It is not a production-signed public Android release and is outside the 17-file public release allow-list.

A future public Android release would require a protected production signing key, signature verification and an explicit release-contract expansion. Private Android signing material must never be committed to the repository.

## Browser helper distribution boundary

The browser connection helper is source/development material. Store signing/publication is not part of the current Ghost FTP release contract. Browser-helper source must not be described as a signed store release merely because the source packages can be loaded unpacked for validation.

## Failure policy

The official Windows release fails closed if:

- either required protected signing credential is absent;
- PFX decoding fails;
- signing fails for any required native staging payload when production signing is configured;
- Setup or Portable lacks a signer certificate;
- Authenticode status is not `Valid`; or
- the publish job receives any signing state other than `signed`.

The production macOS distribution path separately fails closed if Developer ID signing, Hardened Runtime verification, notarization, stapling or Gatekeeper verification cannot be completed.

Do not weaken either production policy by adding an unsigned-publication fallback, a self-signed production substitute or a metadata-only claim of signing.

## Packages

The GHCR distribution bundle contains already-built verified public release files. Signing credentials are never part of its build context or payload. Release metadata records the verified public Windows signing state and native payload set but never carries signing key material.

See [Release verification](RELEASE-VERIFICATION.md), [Security](SECURITY.md), [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Versioning](VERSIONING.md) and the [macOS development/distribution contract](../macos/README.md).
