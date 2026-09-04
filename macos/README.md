# Ghost FTP for macOS

The macOS build produces a universal **Ghost FTP** package for Intel x86_64 and Apple Silicon arm64.

## Build

```bash
go telemetry off
bash macos/BUILD.sh
```

The public GitHub Release publishes:

```text
Ghost-FTP-X.Y.Z-macOS-Universal.pkg
```

## Signing

A successful package build is not the same as Apple publisher signing. Developer ID signing and notarization require valid private Apple credentials. When those credentials are not configured, release documentation must state that limitation rather than implying notarization.

## Compatibility identifiers

The visible application name is **Ghost FTP**. The bundle identifier and executable name may temporarily retain legacy `byftp`/`ByFTP` identifiers to preserve upgrade/application identity while the product is rebranded. New public package names and UI must use Ghost FTP.
