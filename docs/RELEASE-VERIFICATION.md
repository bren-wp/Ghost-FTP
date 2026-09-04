# Verifying Ghost FTP releases

Every Ghost FTP Release publishes `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt` together with the platform packages. Verification should be performed before installation or redistribution.

## 1. Verify the release identity

The expected Ghost FTP tag format is:

```text
ghostftp-vX.Y.Z
```

The version in the tag, package filename, `VERSION`, release title and release metadata must agree.

Historical tags from the former GhostFTP product line are preserved for provenance and are not reused for Ghost FTP releases.

## 2. Verify SHA-256

### Windows PowerShell

```powershell
Get-FileHash .\Ghost-FTP-1.0.0-Setup-x64.exe -Algorithm SHA256
```

Compare the returned digest with the exact filename entry in `SHA256.txt`.

### Linux

```bash
sha256sum Ghost-FTP-1.0.0-Linux-multiarch.zip
```

### macOS

```bash
shasum -a 256 Ghost-FTP-1.0.0-macOS-Universal.pkg
```

### Android / iOS / Web

Use any trusted SHA-256 implementation and compare the exact package filename to `SHA256.txt`.

A mismatch means the package must not be installed.

## 3. Check release metadata

`BUILD-METADATA.txt` records:

- product brand and version;
- Ghost FTP release tag;
- Git commit SHA;
- platform/architecture coverage;
- signing/provenance status;
- expected release-file counts.

For 1.0.x, the public platform contract is Windows x64/x86/x32 alias, Linux multiarch, universal macOS, Android, iOS and Web.

## 4. Understand signing state

A correct SHA-256 checksum proves that the downloaded bytes match the bytes published in that GitHub Release. It does **not** prove Authenticode, Apple notarization, Play signing or App Store signing.

Read the signing status in the release notes and metadata. The public Android CI package is debug-signed and the public iOS package is unsigned unless private deployment credentials are added outside the repository.

## 5. Windows x32 alias check

For Ghost FTP, `Setup-x32.exe` is a compatibility alias of `Setup-x86.exe`. Their SHA-256 digests must be identical. If they differ, treat the Release as invalid.

## 6. Source provenance

The release workflow verifies that `main` still points at the workflow commit immediately before publication. It also refuses to rewrite a Ghost FTP tag that already points to another commit.

For high-assurance deployments, record the release tag, commit SHA and package checksum together in deployment records.
