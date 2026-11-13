# Ghost FTP 0.99.60 — desktop production-readiness assessment

> **Active source:** Ghost FTP 0.99.60 Preview. **Previous canonical release:** v0.99.59. Supported platforms: Windows and Linux.

**Status: PREVIEW, not certified Stable.** Source metadata is not evidence of a published release. Always check the [GitHub Releases index](https://github.com/bren-wp/Ghost-FTP/releases). The historical pixel-equality release quota is retired; real responsive/accessibility acceptance remains necessary.

| Boundary | Current automated evidence | Remaining independent acceptance |
| --- | --- | --- |
| Windows desktop | Native build, NSIS/MSI/portable, selected smoke and Win32 filesystem checks | Authenticode, Windows 10/11 install/upgrade/rollback, credentials and data preservation |
| Linux desktop | Native ELF/AppImage/DEB/RPM, selected package lifecycle and native QA | Package authenticity, supported distros, locked keyrings, WebKit/Wayland/X11, upgrades |
| Rust, React and Go | Build, source tests, Clippy, typecheck and static security/dead-code audits | Threat review, fuzzing, load/crash tests and external security assessment |
| FTP/FTPS/SFTP | Actual localhost protocol roundtrip with TLS/SSH identity regressions | Diverse real servers, large files, interference, cancellation and recovery |
| Cloud integrations | Source adapters and mocked HTTP/provider cases | Authorized account-based provider tests and token refresh/revoke |
| Local files | Staging and no-overwrite safety regression cases | Adversarial filesystem race, full disk, power loss and rollback on real volumes |
| UX | Genuine native captures and interaction contracts | Accessible controls, high DPI/zoom, responsive windows and manual usability |
| Release | SHA-256 inventory, exact commit CI and immutable tags | Real publisher identity, signed update path, production install and rollback |

## High-priority next work

Enforce GitHub branch protection requiring PR reviews and all mandatory status checks; confirm Windows/Linux signing and install/upgrade/rollback evidence; test authorized cloud tenants; verify SSH/FTPS identity and protected vaults under failure; benchmark large concurrent transfers; and complete keyboard/screen-reader/resize QA.

See [Stable policy](../release/STABLE_RELEASE_POLICY.md), [installer QA](../qa/INSTALLER.md) and [cloud live acceptance](../qa/CLOUD_E2E.md). This static review is not an independent penetration test.
