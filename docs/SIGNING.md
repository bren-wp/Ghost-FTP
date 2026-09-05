# Signing and publisher identity

Ghost FTP distinguishes **successful reproducible builds** from **publisher signing**. A build pipeline can create valid Windows/Linux artifacts, but it cannot legitimately manufacture a trusted publisher identity.

## Windows

Authenticode signing requires a real code-signing certificate/private key controlled outside the public repository.

If no signing identity is configured:

- Ghost FTP Setup/portable executables remain unsigned;
- release metadata must say so explicitly;
- documentation must not describe the artifact as trusted/signed merely because CI built it successfully.

Never commit:

- PFX/P12 certificate files;
- private signing keys;
- certificate passwords;
- hardware-token PINs;
- cloud signing credentials.

### SmartScreen and Unknown Publisher

Microsoft SmartScreen/publisher reputation is not solved by a custom license system or by adding metadata to the executable. The legitimate path is Authenticode signing with an appropriate code-signing identity and maintaining publisher/reputation hygiene.

Ghost FTP should not weaken Windows security prompts or attempt to bypass SmartScreen. Until signing is configured, release notes should clearly identify the signing state.

## Linux

The canonical 2.x Linux deliverables are Debian packages plus a multiarch ZIP. Release integrity is provided by:

- GitHub release provenance;
- `BUILD-METADATA.txt` commit/version information;
- `SHA256.txt` hashes;
- package metadata verification in CI.

Distribution-specific repository signing can be added if Ghost FTP is later published through an APT repository. That requires a protected repository signing key and documented rotation/revocation procedures.

A `.deb` produced by CI and distributed directly from GitHub Releases is not equivalent to a package signed by a distribution repository.

## Retired platform signing history

Historical 1.x releases may contain unsigned/debug-signed Android/iOS/macOS artifacts because those platforms were supported by the release matrix at that time. They remain historical provenance only and are not part of the active 2.x signing policy.

## Release metadata

Every Ghost FTP 2.x desktop release contains:

- `RELEASE-NOTES.txt`;
- `BUILD-METADATA.txt`;
- `SHA256.txt`.

Hashes provide integrity verification; they do **not** prove publisher identity. If Windows Authenticode signing is enabled in the future, verification documentation must check both the cryptographic hash and the signature chain/timestamp.

## Signing change controls

Before enabling a signing system:

1. document the certificate/key owner;
2. keep secrets outside source control;
3. restrict CI secret access to the production release path;
4. verify signatures after signing and before publication;
5. record timestamping behavior;
6. document key rotation/revocation;
7. ensure unsigned fallback artifacts cannot be mislabeled as signed.
