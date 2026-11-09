# Ghost FTP desktop handbook

**Current development:** v0.99.36 Preview. **Published predecessor:** v0.99.35. Supported platforms are Windows, Linux and macOS desktop.

## Guides

- [Windows app](WINDOWS.md): native file manager, protocols, transfer queue, credentials and installer checks
- [Linux app](LINUX.md): packages, WebKitGTK, keyring, transfer security and desktop sessions
- [macOS app](MACOS.md): SwiftUI, Keychain, FTP and limited cloud operations
- [Desktop feature matrix](FEATURE_MATRIX.md): implemented, restricted, unsupported and unverified operations
- [Security and privacy](SECURITY_PRIVACY.md): trust boundaries, credentials, transfer integrity, threat models and release governance
- [Delivery roadmap](DELIVERY_ROADMAP.md): outstanding acceptance work
- [QA index](../qa/README.md): real transfer protocol and native UI tests
- [Version and release policy](../releases/VERSIONING.md): immutable tags and Preview requirements

## Evidence standard

A source implementation does not prove that an external server supports its behavior. All release claims require exact-head CI, native package verification, real credentials only in authorized private fixtures, genuine installed captures for Windows/Linux and fully documented outstanding gaps. macOS FTPS/SFTP remain unsupported until independently implemented and verified. Historic release documents may describe previous products but do not define the current platform lineup.
