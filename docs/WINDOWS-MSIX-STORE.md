# Windows Store / MSIX readiness

Ghost FTP keeps the existing Windows Setup and Portable EXE release flow unchanged. The Store path is an additional, manually initiated distribution channel.

## Why this path is separate

Microsoft Store submission and direct EXE distribution have different signing behavior. A Store MSIX/AppX package is processed through Partner Center and is re-signed by Microsoft after certification. An EXE/MSI submission is not re-signed in the same way and remains subject to its own Authenticode/signing requirements.

The repository therefore does not replace or mutate the canonical Setup or Portable executables when preparing a Store package.

## Manual workflow

The workflow .github/workflows/ghostftp-msix-store.yml is workflow_dispatch only. It requires the exact Partner Center Package/Identity/Name, Package/Identity/Publisher, publisher display name, Windows.Desktop minimum version and maximum tested Windows version.

These values must come from the Ghost FTP Partner Center product identity. Do not invent or normalize them.

The workflow checks canonical version alignment, runs the normal frontend/UI validation, builds the native x64 release executable, packages it as a full-trust MSIX with runFullTrust, unpacks the produced MSIX to verify its expected manifest and executable, and uploads the Store submission package as a short-retention CI artifact.

## Signing boundary

tools/msix/build-store-msix.ps1 intentionally does not attempt to bypass Windows reputation or fabricate a certificate. The Store submission path relies on Microsoft Store certification/signing.

For direct sideload distribution of an MSIX outside the Store, use an appropriate trusted code-signing path such as Microsoft's current Azure Artifact Signing offering or another trusted certificate/signing service. That is a separate release decision.

## Existing Windows flow

Windows Setup and Windows Portable remain independent canonical downloads. Do not replace those files with the Store package and do not change their release hashes after publication.
