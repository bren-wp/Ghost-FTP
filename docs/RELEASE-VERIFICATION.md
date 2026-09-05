# Verifying Ghost FTP releases

Every current Ghost FTP desktop release publishes `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt` together with the Windows/Linux platform packages. Verification should be performed before installation or redistribution.

The active baseline is **0.1.0 Beta**. Every `0.x.y` GitHub Release must be marked **Prerelease**. The first version eligible for the stable channel is **1.0.0**.

## 1. Verify the release identity

The expected tag format is:

```text
ghostftp-vX.Y.Z
```

The version in the tag, package filename, root `VERSION`, release title and `BUILD-METADATA.txt` must agree.

For a pre-1.0 release, the release title must include `Beta` and GitHub must report the release as a prerelease.

Historical tags/releases remain available for provenance. They do not define the active Windows/Linux package matrix and must not be reused or moved to another commit.

## 2. Expected current release files

The desktop release contains **9 platform artifacts**:

1. `Ghost-FTP-X.Y.Z-Setup-x64.exe`
2. `Ghost-FTP-X.Y.Z-Setup-x86.exe`
3. `Ghost-FTP-X.Y.Z-Setup-x32.exe` — byte-identical compatibility alias of x86
4. `Ghost-FTP-X.Y.Z-Portable-x64.exe`
5. `Ghost-FTP-X.Y.Z-Portable-x86.exe`
6. `Ghost-FTP-X.Y.Z-Linux-amd64.deb`
7. `Ghost-FTP-X.Y.Z-Linux-arm64.deb`
8. `Ghost-FTP-X.Y.Z-Linux-i386.deb`
9. `Ghost-FTP-X.Y.Z-Linux-multiarch.zip`

The release also contains:

- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`
- `SHA256.txt`

That is **12 public release files** total.

## 3. Verify SHA-256

### Windows PowerShell

For the current Beta baseline:

```powershell
Get-FileHash .\Ghost-FTP-0.1.0-Setup-x64.exe -Algorithm SHA256
Get-FileHash .\Ghost-FTP-0.1.0-Portable-x64.exe -Algorithm SHA256
```

Compare each returned digest with the exact filename entry in `SHA256.txt`.

### Linux

```bash
sha256sum Ghost-FTP-0.1.0-Linux-amd64.deb
sha256sum Ghost-FTP-0.1.0-Linux-multiarch.zip
```

A mismatch means the package must not be installed or redistributed.

`SHA256.txt` is generated **after** all final Windows PE-resource and Authenticode mutations so the manifest describes the actual published bytes.

## 4. Verify Windows Authenticode

A SHA-256 match proves byte integrity relative to the GitHub Release. It does not by itself establish publisher identity.

Check a Windows executable with:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-0.1.0-Setup-x64.exe | Format-List Status,StatusMessage,SignerCertificate,TimeStamperCertificate
```

Or for Portable:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-0.1.0-Portable-x64.exe | Format-List Status,StatusMessage,SignerCertificate,TimeStamperCertificate
```

Interpretation:

- `Valid` means Windows validates the Authenticode signature under the local trust policy.
- `NotSigned` means the artifact is unsigned.
- `UnknownError`, `NotTrusted` or equivalent trust failure must not be presented as a verified publisher identity.
- `HashMismatch` means the executable changed after signing and must be rejected.

`BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=signed` or `WINDOWS_AUTHENTICODE=unsigned` for the publication workflow.

Beta releases may be explicitly unsigned when no protected production signing identity is configured. Stable releases at `1.0.0` or later are blocked by the release workflow unless a trusted production Authenticode identity is configured.

A development/self-signed certificate used by CI is only a signing-pipeline test. It is never a substitute for a trusted production publisher certificate and its private key is never published.

See [Signing](SIGNING.md).

## 5. Verify the x32 compatibility alias

`Setup-x32.exe` is intentionally a compatibility alias of `Setup-x86.exe`. The files must be byte-identical.

Windows PowerShell:

```powershell
(Get-FileHash .\Ghost-FTP-0.1.0-Setup-x86.exe -Algorithm SHA256).Hash
(Get-FileHash .\Ghost-FTP-0.1.0-Setup-x32.exe -Algorithm SHA256).Hash
```

The two hashes must match exactly. If they differ, treat the release as invalid.

## 6. Verify Linux package metadata

For each Debian package:

```bash
dpkg-deb -f Ghost-FTP-0.1.0-Linux-amd64.deb Package Version Architecture
```

Expected values for the amd64 package are:

```text
Package: ghost-ftp
Version: 0.1.0
Architecture: amd64
```

Repeat for `arm64` and `i386` and verify the architecture matches the filename.

The multiarch ZIP must contain exactly the three verified Debian packages for amd64, arm64 and i386.

## 7. Verify build metadata

`BUILD-METADATA.txt` records the publication boundary, including:

- public brand and technical identity;
- version and release tag;
- release channel (`beta` or `stable`);
- Git commit SHA;
- active application platforms (`WINDOWS,LINUX`);
- Windows Setup/Portable architecture coverage;
- Windows Authenticode state;
- Linux package architecture coverage;
- language count/default language;
- telemetry build policy;
- expected platform-artifact and public-file counts.

For `0.x.y`, `RELEASE_CHANNEL` must be `beta`.

## 8. Verify source provenance

The production release workflow verifies that `main` still points at the workflow commit immediately before publication. It refuses to move an existing `ghostftp-vX.Y.Z` tag to another commit.

After upload, the workflow performs two remote checks:

1. immediate release-asset readback;
2. a delayed readback after another `main` commit check.

Both passes compare the remote asset name/size set and download the published `SHA256.txt` for byte-for-byte comparison with the local manifest.

For high-assurance deployments, record together:

- release tag;
- Git commit SHA;
- artifact filename;
- SHA-256 digest;
- Authenticode signer/thumbprint and timestamp state for Windows artifacts when signed.

## 9. Active versus historical platforms

Current application release verification covers **Windows and Linux only**. Historical Android, iOS and macOS artifacts may remain attached to older historical releases, but they are not part of the active release matrix and must not be expected in `0.1.0 Beta` or later current Windows/Linux releases.
