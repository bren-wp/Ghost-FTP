# Release verification

Ghost FTP release verification applies to the maintained Windows, Linux and Android applications plus browser helper packages.

## Required platform artifacts

- Windows Setup
- Windows Portable
- Debian Installer
- Debian Portable
- Ubuntu Installer
- Ubuntu Portable
- Fedora Installer
- Fedora Portable
- Android APK
- Chrome helper
- Edge helper
- Firefox helper
- Opera helper

That is **13 platform artifacts**.

The release directory also contains:

- RELEASE-NOTES.txt
- BUILD-METADATA.txt
- SHA256.txt

for a total of **16 public files**.

## Required metadata

```text
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```

## Verification gates

1. source VERSION matches the requested release version;
2. release is built from the exact expected main SHA;
3. Windows package structure passes;
4. Linux package matrix passes;
5. Android build and signing-state checks pass;
6. browser helper packages are deterministic and sanitized;
7. security/privacy audits pass;
8. CodeQL and Govulncheck pass;
9. authentic Windows/Linux/Android runtime screenshots are bound to the tested source SHA;
10. every final file is covered by SHA256.txt;
11. release readback confirms the uploaded assets match the assembled release directory.

The retired macOS application contributes no current artifact, signing gate or release count.

