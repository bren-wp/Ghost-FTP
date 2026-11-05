# Ghost FTP 0.99.32 — Windows/Linux security and QA evidence

**Source status:** Preview code, awaiting exact-head CI and native capture review.

| Windows/Linux surface | Change | Automated verification | Unverified physical acceptance |
| --- | --- | --- | --- |
| Files · Quick Connect | Clear transient password on failed and successful attempt; preserve ephemeral no-persist session behavior | Premium Files UI source regression; TypeScript production build; native protocol E2E | Real credential-manager/Secret Service locked cases; visual 1:1 |
| Files · connection banner | Reflect real health state; warning for unhealthy, neutral for unknown/checking, green only for known healthy encrypted transport | Premium Files UI source regression; native status health probe remains operational | Long-lived socket disconnections and live UI observation |
| Native status bar | Suspend 1-second clock when hidden; refresh on focus and visibility | Source regression and React production typecheck | Windows 10/11 and GNOME/KDE power/wake measurements |
| Grant link import | No environment proxy, bounded response stream, reject malformed SSH targets and invalid login fields | Rust grant JSON/target validation tests; native build | Untrusted remote issuer network fault injection; OS proxy environment and DNS rebind testing |
| Windows packaging | Existing Setup EXE/MSI install and uninstall workflow | Native Windows build + installer smoke | Signed Windows in-place upgrades and rollback |
| Linux packaging | Existing DEB, RPM, AppImage matrix | Native Linux build + package/screenshot QA | Unverified distros, Wayland/x86_64 variants, ARM/musl |

Reference PNGs are concept artwork, not a real running program; only CI-captured Tauri/WebKit images qualify as application screenshots. The screen-by-screen pixel comparison must normalize resolution, DPI, locale and honest filesystem/connection state and cannot claim exact 1:1 with invented server transfers.
