# Signing and publisher identity

Ghost FTP distinguishes **successful reproducible builds** from **publisher signing**. A build pipeline can create valid Windows/Linux artifacts, but it cannot legitimately manufacture a trusted publisher identity.

## Active lifecycle

The current desktop baseline is **0.1.0 Beta**. All `0.x.y` releases are Beta/Prerelease. The first stable release is **1.0.0**.

Setup and Portable executables always use the same canonical version from `VERSION`.

## Windows Authenticode

Authenticode signing requires a code-signing certificate/private key controlled outside the public repository.

Ghost FTP now has a dedicated signing path:

- `scripts/Sign-WindowsArtifacts.ps1` imports a supplied PFX into the current-user certificate store, selects exactly one currently valid Code Signing certificate with a private key, signs with SHA-256, verifies the signer thumbprint/state, and removes the imported certificate from the working store afterwards;
- `BUILD-WINDOWS.ps1` signs the Portable executable **before** generating the installer payload, ensuring Setup embeds the same signed client bytes that are published separately;
- Setup is signed only after all PE resources have been finalized;
- SHA-256 manifests are generated only after signing;
- if signing is configured but verification reports an unsigned executable, the build fails.

### Signing environment

The Windows build reads these environment variables:

- `GHOSTFTP_SIGNING_PFX_PATH` — path to the protected PFX file;
- `GHOSTFTP_SIGNING_PASSWORD` — PFX password;
- `GHOSTFTP_SIGNING_TIMESTAMP_URL` — optional timestamp service URL;
- `GHOSTFTP_ALLOW_UNTRUSTED_SIGNER` — development-only switch for a self-signed test certificate.

Never commit the PFX or its password. For GitHub production publication, the PFX should be reconstructed only inside the ephemeral Windows runner from a protected GitHub Actions secret and removed after signing.

## Development self-signing

`scripts/New-DevCodeSigningCertificate.ps1` creates a short-lived RSA-3072/SHA-256 self-signed Code Signing certificate for development/testing.

The script exports:

- a `.pfx` containing the temporary private key;
- a `.cer` containing only the public certificate.

The generated files are development secrets/artifacts and must not be committed to source control.

A self-signed signature is useful for validating the signing pipeline and detecting post-sign mutation, but it does **not** establish a Microsoft-trusted publisher. Windows systems that do not explicitly trust that development certificate can still report an unknown/untrusted publisher.

## Production signing

A production release should use a legitimate externally issued code-signing identity controlled by the Ghost FTP publisher.

The production private key must remain outside the repository and outside ordinary build logs. Recommended storage boundaries include:

- protected GitHub Actions secret containing an encrypted/base64 PFX plus a separate password secret;
- a managed signing service/HSM where available;
- an offline or hardware-backed certificate workflow with explicit artifact handoff.

Do not create a deterministic or repository-derived private key. Anyone who can reproduce such a key would be able to impersonate the publisher.

## SmartScreen and Unknown Publisher

Microsoft SmartScreen/publisher reputation is not solved by a custom license system, executable metadata, or a self-signed certificate. The legitimate path is Authenticode signing with an appropriate publisher identity and maintaining signing/reputation hygiene.

Ghost FTP must not weaken Windows security prompts or claim a trusted publisher when the signing chain is not trusted by Windows.

## Linux

The canonical Linux deliverables are Debian packages plus a multiarch ZIP. Release integrity is provided by:

- GitHub release provenance;
- `BUILD-METADATA.txt` commit/version information;
- `SHA256.txt` hashes;
- package metadata verification in CI.

Distribution-specific repository signing can be added if Ghost FTP is later published through an APT repository. That requires a protected repository signing key and documented rotation/revocation procedures.

A `.deb` produced by CI and distributed directly from GitHub Releases is not equivalent to a package signed by a distribution repository.

## Release metadata

Every active desktop release contains:

- `RELEASE-NOTES.txt`;
- `BUILD-METADATA.txt`;
- `SHA256.txt`.

Hashes provide integrity verification; they do **not** prove publisher identity. When Authenticode signing is enabled, release verification must check both the file hash and Authenticode signer state.

## Private-key rules

The repository must never contain:

- PFX/P12 certificate files;
- private signing keys;
- certificate passwords;
- hardware-token PINs;
- cloud signing credentials;
- deterministic test secrets that can sign public releases.

Development signing artifacts must remain in ignored/local temporary paths and should be deleted after use.

## Signing change controls

Before enabling or replacing a production signing identity:

1. document the certificate/key owner;
2. keep secrets outside source control;
3. restrict CI secret access to the production release path;
4. verify signatures after signing and before publication;
5. record timestamping behavior;
6. document key rotation/revocation;
7. ensure unsigned fallback artifacts cannot be mislabeled as signed;
8. verify the Portable payload is signed before it is embedded in Setup;
9. ensure SHA-256 manifests are generated after all signing mutations are complete.
