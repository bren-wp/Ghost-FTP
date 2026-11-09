# Ghost FTP 0.99.36 desktop release readiness

**Decision: NOT CERTIFIED FOR STABLE PRODUCTION.** Current source targets Windows, Linux and macOS desktop only. Historic published editions are immutable.

| Surface | Verified from source | Remaining evidence |
| --- | --- | --- |
| Windows/Linux Rust/Tauri | Transfer engine, staged downloads, SFTP/Agent verified-prefix resume and SSH/TLS guards implemented | Exact-head build/test/Clippy, native installer upgrade, credential vault, real servers, remote hashes and crash-recovery tests |
| Windows native | WebView2 UI/installer and Win32 tests integrated in CI | Signed Authenticode upgrade, target Windows variants, local vault and interaction capture |
| Linux native | Shared Rust engine and package formats integrated | Target distro/GTK/WebKit/Wayland compatibility, Secret Service, install/rollback |
| macOS SwiftUI | FTP session, preview package and Keychain source present | Real app runtime, signed/notarized deployment; FTPS/SFTP unavailable |
| Cloud providers | Adapter-specific API implementations and static security checks | Consent-based live tenant/OAuth/account/partial upload E2E, quota, refresh and revocation |
| Visual QA | 30 Windows/29 Linux source reference mappings | 59 real installed screenshots and per-control keyboard/a11y tests |

**Release criteria:** pass all exact-source quality/dead-code/security tests; Rust workspace, frontend, Go, Swift and Windows/Linux native CI; localhost FTP/FTPS/SFTP E2E; repeatable packages with SHA-256; genuine installed launch and transfer QA. Treat code scans as regression guards, not comprehensive independent penetration testing.

Do not label any desktop build stable or pixel-perfect until the above evidence is gathered.
