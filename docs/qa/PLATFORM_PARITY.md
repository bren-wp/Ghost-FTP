# Ghost FTP desktop platform parity

**Source train:** v0.99.47 Preview. Capability matrix only; every shipped feature requires native runtime evidence.

| Capability | Windows | Linux |
| --- | --- | --- |
| Desktop UI | React/Tauri | React/Tauri |
| Production packaging | Portable EXE, NSIS, MSI | Binary, AppImage, DEB, RPM |
| Plain FTP | Implemented; transmission is unencrypted | Same shared Rust engine |
| Explicit FTPS | TLS certificate and hostname verification required | Same shared Rust engine |
| SFTP | SSH host-key verification required | Same shared Rust engine |
| Transfer persistence | Bounded transfer ledger, pause/retry safeguards | Same |
| Cloud/provider accounts | Adapter-specific; live credentialed E2E pending | Same |
| Visual references | 30 concepts at 1290×852 | 29 concepts at 1290×852 |
| Publisher signing | Windows Authenticode acceptance pending | Distribution/package-signing acceptance pending |

All source audits and installed runtime tests must operate on the exact release source SHA. Real Windows/Linux interactions, accessibility, and responsive layouts remain supported QA, without a legacy fixed-pixel requirement.
