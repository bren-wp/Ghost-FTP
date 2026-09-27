# Windows Store / MSIX readiness

Ghost FTP keeps the existing Windows Setup and Portable EXE release flow unchanged. The Store path is an additional, manually initiated distribution channel.

## Partner Center identity

The manual workflow requires the exact Package/Identity/Name and Package/Identity/Publisher values from the Ghost FTP Microsoft Partner Center product. It also requires the approved Windows.Desktop minimum version and maximum tested version. These values are intentionally not guessed or hardcoded.

## Manual workflow

The .github/workflows/ghostftp-msix-store.yml workflow is workflow_dispatch only. It validates the canonical Ghost FTP version, builds the native x64 application, packages it as a full-trust MSIX with the runFullTrust capability, unpacks the result to verify the expected manifest/executable, calculates SHA-256, and uploads a short-retention CI artifact.

The helper tools/msix/build-store-msix.ps1 generates package visual assets from the canonical Ghost FTP icon and creates the AppxManifest.xml from the explicit Partner Center inputs.

## Signing boundary

The helper intentionally does not fabricate a certificate, disable Windows verification, or attempt to bypass SmartScreen. For a Microsoft Store submission, certification/signing remains part of the Store process. For direct sideload distribution outside the Store, use an appropriate trusted code-signing service such as Microsoft's Azure Artifact Signing or another suitable trusted certificate/signing path.

## Existing Windows release flow

Windows Setup and Windows Portable remain canonical downloads and are not replaced by the Store MSIX. Published EXE bytes and hashes remain immutable.
