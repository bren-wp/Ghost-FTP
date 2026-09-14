# Ghost FTP signing

Ghost FTP **0.0.6** separates production publisher trust from development signing. Official Windows and Android publication both fail closed when their protected production identities are unavailable. Linux relies on exact-source/package/digest verification, and macOS remains outside the public release until its real Developer ID + Apple notarization path succeeds.

The production workflow never creates its own long-lived publisher key.

## Windows production publication policy

Official Windows publication is **signed-only**.

The canonical `Publish Ghost FTP` workflow requires:

```text
GHOSTFTP_SIGNING_PFX_BASE64
GHOSTFTP_SIGNING_PASSWORD
```

`GHOSTFTP_SIGNING_TIMESTAMP_URL` is used when a production timestamp endpoint is intentionally configured. There is **no supported `state=unsigned` continuation path** for official publication.

A successful release records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

The PFX is decoded only into runner temporary storage and removed in an `always()` cleanup path. Private-key material must never be committed, logged, uploaded as an artifact or included in release/GHCR payloads.

Local/ordinary CI Windows builds may remain unsigned, but they are not official release evidence. The CI signing smoke can create an ephemeral development certificate solely to test signing mechanics; that identity is never accepted by the production workflow.

## Windows build/signing ordering

`BUILD-WINDOWS-ARCH-STAGE.ps1` creates verified native x64, x86 and ARM64 Setup/Portable staging pairs. `BUILD-WINDOWS.ps1` embeds those payloads into the two public universal files:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

The outer executables are signed only after final byte mutation. `Get-AuthenticodeSignature` must report a signer certificate and `Valid` status before publication. `scripts/verify_release.py` independently rejects unsigned official public artifacts.

`WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` remains an explicit boundary: successful cross-build/signature verification is not native Windows ARM64 execution proof.

## Android production signing

Ghost FTP 0.0.6 adds a **production-signed public Android release**:

```text
Ghost-FTP-0.0.6-Android.apk
```

The canonical release workflow requires protected secrets:

```text
GHOSTFTP_ANDROID_KEYSTORE_BASE64
GHOSTFTP_ANDROID_KEYSTORE_PASSWORD
GHOSTFTP_ANDROID_KEY_ALIAS
GHOSTFTP_ANDROID_KEY_PASSWORD
GHOSTFTP_ANDROID_CERT_SHA256
```

The keystore is decoded only into runner temporary storage with restricted file permissions. The workflow builds the unsigned release APK, signs it with Android `apksigner`, verifies it with `apksigner verify --verbose --print-certs`, reads the signer certificate SHA-256 digest and requires exact equality with `GHOSTFTP_ANDROID_CERT_SHA256`. The temporary keystore is removed in an `always()` cleanup path.

Production Android publication fails closed if:

- any required protected signing value is missing;
- the decoded keystore is invalid/unexpectedly small;
- APK signing fails;
- `apksigner` verification fails;
- the signer certificate SHA-256 fingerprint cannot be read; or
- the actual fingerprint does not equal the protected expected fingerprint.

The release workflow must not run `keytool -genkeypair` or otherwise generate a replacement Android publisher identity.

### Android development signing

Ordinary exact-head Android CI still builds:

```text
Ghost-FTP-Android-dev.apk
```

It may use an ephemeral CI-only key to prove the signing pipeline mechanically. That artifact and identity are not the public production APK/publisher.

Production signing does not change protocol support. Android SFTP remains intentionally hidden until strict maintained host-key verification exists.

## Linux release integrity

Linux artifacts do not use Authenticode, Android signing or Apple Developer ID. Integrity is enforced through exact-source build provenance, package metadata, extracted binary parity, the canonical release allow-list and `SHA256.txt`.

## Browser helper distribution boundary

The public Chrome, Edge and Firefox ZIPs are deterministic source packages. Their presence in the 0.0.6 GitHub Release does not claim Chrome Web Store/Edge Add-ons/Firefox AMO signing or approval. They remain privacy-minimal local parser/copy helpers with no supported browser-to-desktop handoff.

## macOS development signing

`macos/BUILD.sh` may produce an ad-hoc signed universal native development artifact. An ad-hoc signature is not a Developer ID signature and is not Apple notarization evidence.

## macOS production signing and notarization

`macos/SIGN_AND_NOTARIZE.sh` is the separate fail-closed production-distribution path. It requires a real **Developer ID Application** identity, Hardened Runtime, secure timestamping, Apple notarization acceptance, ticket stapling and Gatekeeper verification. `.github/workflows/macos-production.yml` is the environment-gated CI path.

A successful development build does not prove production distribution readiness. Ghost FTP does not claim macOS publication until that credentialed path actually succeeds. macOS therefore remains outside the 0.0.6 21-file public release.

## Release shape and metadata

The 0.0.6 release contains **18 platform artifacts / 21 public files**. `BUILD-METADATA.txt` records public signing/evidence states but never secret key material, including:

```text
WINDOWS_AUTHENTICODE=signed
ANDROID_APK=production-signed
ANDROID_SIGNER_SHA256=<verified public certificate fingerprint>
ANDROID_SFTP=hidden-until-strict-host-key-verification
PUBLIC_PLATFORM_ARTIFACTS=18
PUBLIC_RELEASE_FILES=21
```

The GHCR object contains already-built verified release files only. It never contains signing credentials and is a distribution bundle, not a runtime container.

See [Release verification](RELEASE-VERIFICATION.md), [Security](SECURITY.md), [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Versioning](VERSIONING.md) and the [macOS development/distribution contract](../macos/README.md).
