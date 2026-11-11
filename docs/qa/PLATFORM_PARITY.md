# Ghost FTP desktop platform parity

**Source train:** v0.99.47 Preview. Capability/acceptance matrix; functionality requires native runtime evidence.

| Capability | Windows | Linux |
| --- | --- | --- |
| Desktop UI | React/Tauri | React/Tauri |
| Production packaging | Portable EXE, NSIS, MSI | ELF, AppImage, DEB, RPM |
| Plain FTP | Implemented; transport is unencrypted | Same shared Rust engine |
| Explicit FTPS | TLS hostname/certificate verification required | Same shared Rust engine |
| SFTP | SSH host-key verification required | Same shared Rust engine |
| Transfer persistence | Bounded transfer ledger, pause/retry safeguards | Same |
| Cloud/provider accounts | Adapter-specific; live credentialed E2E pending | Same |
| Visual reference | 30 concepts at 1290×852 | 29 concepts at 1290×852 |
| Production signing | Authenticode deployment gate pending | Package-signing acceptance pending |

All source audits and installed runtime tests must operate on the exact release source SHA. Native Windows/Linux screenshot comparisons must match DPI, language and state. Source-token checks do **not** prove 1:1 pixel fidelity.
