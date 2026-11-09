# Ghost FTP desktop platform parity

**Source train:** v0.99.36 Preview. This is a capability/acceptance matrix, not proof that every listed operation succeeds on every public endpoint.

| Capability | Windows | Linux | macOS (native SwiftUI) |
| --- | --- | --- | --- |
| Desktop UI | React/Tauri | React/Tauri | SwiftUI |
| Production packaging | Portable EXE, NSIS, MSI | Binary, AppImage, DEB, RPM | Architecture-specific ZIP/DMG Preview |
| Plain FTP | Implemented; encrypted transport **not** provided | Same shared Rust engine | Implemented; separate session engine |
| Explicit FTPS | Implemented, TLS host/certificate verification required | Same | Not implemented; UI must not claim support |
| SFTP | Implemented, SSH host-key verification required | Same | Not implemented; UI must not claim support |
| Transfer persistence | Bounded transfer ledger, pause/retry safeguards | Same | Credential-free bounded history; not full desktop recovery |
| Cloud/provider accounts | Adapter-specific; live credentialed E2E pending | Same | Limited provider functionality; live E2E pending |
| Visual reference | 30 concepts at 1290×852 | 29 concepts at 1290×852 | No approved 1:1 reference supplied |
| Production signing | Authenticode deployment gate pending | Distribution/package-signing acceptance pending | Developer ID notarization pending |

All code audits and installed runtime tests must run on the exact release source. Windows/Linux screenshots must be native captures at matching DPI, language and state. Do not report pixel-perfect 1:1 parity from a source-token check or mockup. macOS FTPS/SFTP must stay disabled until verified native transports exist.
