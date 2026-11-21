# Ghost FTP by Brendigo — Third-party notices

Ghost FTP for Windows and Linux includes and links open-source and operating-system components, each subject to its own licences and copyright notices. The Ghost FTP commercial licence does not override those independent obligations.

## Dependency provenance
For each release, the authoritative dependency graph must be derived from the *exact published source tag* and resolved build:
- `ghostftp-desktop/package-lock.json`: TypeScript, React, Tauri JavaScript tooling and their transitive dependencies;
- `ghostftp-desktop/src-tauri/Cargo.lock`: Rust/Tauri, FTP/FTPS/SFTP, crypto and other transitive components;
- current Go module files where applicable for compatibility/build tooling;
- Windows WebView2, Linux WebKitGTK and other system packages as separately licensed by suppliers.

Do not reuse retired Android/macOS, standalone Server, CLI or Demo dependency lists as an inventory of the current desktop product. Package version changes require licence and vulnerability review, including bundled notices or source obligations where applicable.

## Trademark and distribution conditions
Interoperability references to third-party operating systems, protocols, servers and cloud brands are descriptive, not endorsements. Before publication, review the actual resolved dependencies and release contents, preserve required attribution/licence texts, and verify any reciprocal/source-offer obligations. A vulnerability audit cannot substitute for a licence-compliance review.

**Ghost FTP by Brendigo** · [Licence](../../LICENSE.txt) · [Privacy](PRIVACY.md) · [Security](../../SECURITY.md).
