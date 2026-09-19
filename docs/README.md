# Ghost FTP documentation

This documentation describes the maintained Ghost FTP product surface: **Windows, Linux and Android**, plus local browser helper packages.

Current published version: **0.0.8**  
Current development line: **0.0.9**

## Product boundary

The active native applications are:

- Windows — Setup + Portable;
- Linux — Debian / Ubuntu / Fedora Installer + Portable bundles;
- Android — installable APK.

The previous macOS application and its source, build scripts, signing/notarization workflows and platform-specific regression tests were retired after the 0.0.8 line and are no longer part of the active source tree.

## Reference-driven UI

The maintained applications follow the same Ghost FTP product hierarchy:

- Files;
- Connections / Sites;
- Transfer Queue / Transfers;
- Settings;
- Bookmarks and More as supporting actions.

Windows, Linux and Android are validated against authentic exact-head runtime screenshots. Reference images define the visual target; generated mockups are never accepted as execution evidence.

## Start here

- [Installation](INSTALLATION.md)
- [Reference UI](REFERENCE-UI.md)
- [Platform parity](PLATFORM-PARITY.md)
- [Security](SECURITY.md)
- [Privacy](PRIVACY.md)
- [Settings](SETTINGS.md)
- [Navigation and bookmarks](NAVIGATION-BOOKMARKS.md)
- [Queue priority](QUEUE-PRIORITY.md)
- [Testing](TESTING.md)
- [Signing](SIGNING.md)
- [Packages](PACKAGES.md)
- [Release verification](RELEASE-VERIFICATION.md)
- [Versioning](VERSIONING.md)

## Release evidence

Authentic UI evidence is maintained for Windows, Linux and Android. The current checked-in 0.0.8 evidence remains historical proof for that release; 0.0.9 development changes require fresh exact-head runtime captures before release.

## Privacy and security

Ghost FTP has no application telemetry, behavioral analytics, advertising, mandatory Ghost FTP account or hidden storage cloud.

Desktop SFTP remains strict about host-key trust. Explicit FTPS validates certificate and hostname identity. Android SFTP stays hidden until the same strict trust boundary is maintained there.

