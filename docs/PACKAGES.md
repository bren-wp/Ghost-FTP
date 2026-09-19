# Packages and distribution

Ghost FTP distributes maintained application artifacts for **Windows, Linux and Android**, plus local browser helper packages.

## Windows

Public Windows artifacts:

- `Ghost-FTP-<version>-Setup.exe`
- `Ghost-FTP-<version>-Portable.exe`

The public executables are universal launchers carrying x64, x86 and ARM64 native payloads.

## Linux

For each supported distro family:

- Debian Installer + Portable
- Ubuntu Installer + Portable
- Fedora Installer + Portable

Each bundle carries amd64, arm64 and i386 payloads.

## Android

Public Android artifact:

- `Ghost-FTP-<version>-Android.apk`

The release metadata records the actual signing state and signer fingerprint policy.

## Browser helpers

Supporting helper packages are produced for:

- Chrome
- Edge
- Firefox
- Opera

The browser helper does not implement FTP transport itself. On Windows it can hand off a sanitized connection descriptor to the installed desktop application without passwords, private-key passphrases, query data or fragments.

## Active release metadata

```text
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```

The 13 platform artifacts are:

- Windows: 2
- Linux: 6
- Android: 1
- Browser helpers: 4

Release metadata adds:

- RELEASE-NOTES.txt
- BUILD-METADATA.txt
- SHA256.txt

for a total of 16 public files.

## Signing boundary

Protected publication fails closed when required Windows or Android publisher credentials are unavailable. No platform is presented as more strongly signed than the produced artifact actually is.

The former macOS distribution is retired and is not part of active package totals, workflows or source-platform metadata.

