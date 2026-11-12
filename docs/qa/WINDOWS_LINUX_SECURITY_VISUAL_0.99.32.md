# Ghost FTP 0.99.32 — Windows/Linux security and QA evidence

**Source status:** Preview code, awaiting exact-head CI and native capture review.

| Windows/Linux surface | Change | Automated verification | Unverified physical acceptance |
| --- | --- | --- | --- |
| Files · Quick Connect | Clear transient password on failed and successful attempt; preserve ephemeral no-persist session behavior | Premium Files UI source regression; TypeScript production build; native protocol E2E | Real credential-manager/Secret Service locked cases; functional keyboard and screen-reader checks |
| Files · connection banner | Reflect real health state; warning for unhealthy, neutral for unknown/checking, green only for known healthy encrypted transport | Premium Files UI source regression; native status health probe remains operational | Long-lived socket disconnections and live UI observation |
| Native status bar | Suspend 1-second clock when hidden; refresh on focus and visibility | Source regression and React production typecheck | Windows 10/11 and GNOME/KDE power/wake measurements |
| Grant link import | No environment proxy, bounded response stream, reject malformed SSH targets and invalid login fields | Rust grant JSON/target validation tests; native build | Untrusted remote issuer network fault injection; OS proxy environment and DNS rebind testing |
| Windows packaging | Existing Setup EXE/MSI install and uninstall workflow | Native Windows build + installer smoke | Signed Windows in-place upgrades and rollback |
| Linux packaging | Existing DEB, RPM, AppImage matrix | Native Linux build + package/screenshot QA | Unverified distros, Wayland/x86_64 variants, ARM/musl |

Historical reference PNGs are concept artwork, not a running program. Only genuine native-app screenshots qualify as product images; reference comparison is not a release acceptance condition.

## Measured native v0.99.31 Linux pixel baseline (not functional defect rates)

Compared seven real Tauri/WebKit screenshots from successful [native QA run #41548117028](https://github.com/bren-wp/Ghost-FTP/actions/runs/41548117028) against the user-provided `screens/linux/*.png` references, all at exactly 1290×852 (1,099,080 RGB pixels each), without rescaling, color changes or simulated server content:

| Screen | Pixels differing in any RGB channel | Difference fraction | Mean absolute RGB channel error (0–255) |
| --- | ---: | ---: | ---: |
| Files | 1,070,468 | 97.40% | 14.89 |
| Sites | 980,486 | 89.21% | 13.40 |
| New Connection | 1,099,054 | 100.00% | 15.73 |
| Transfers | 1,058,773 | 96.33% | 14.22 |
| Settings | 1,058,718 | 96.33% | 12.94 |
| Sync | 1,059,621 | 96.41% | 14.23 |
| Help & About | 1,054,813 | 95.97% | 13.37 |

**Interpretation:** this is a strict *image* difference, not a percentage of failed functionality. The real capture uses English desktop locale and real disconnected files/transfers, while the Croatian concept has illustrative populated sessions and different OS font rendering. Never synthesize fake sessions/transfers to reduce this number.

**New Connection position:** native v0.99.31 approximately x=320–970/y=116–736; Linux concept approximately x=422–1082/y=163–793. This PR centers the modal within the right-hand workspace, moves optional Site Name to the still-accessible Advanced section and retains real Save and Test Connection actions. A fresh native capture is required to validate the resulting geometry; no 1:1 certification is claimed.

## Measured native v0.99.31 Windows pixel baseline

Also compared seven genuine Windows WebView2 1290×852 screenshots from the successful Windows native [PR run #41548117028](https://github.com/bren-wp/Ghost-FTP/actions/runs/41548117028) against the corresponding Windows concept references, with exact RGB per-pixel comparisons and no rescaling:

| Screen | Changed RGB pixels | Changed portion | Mean absolute RGB error (0–255) |
| --- | ---: | ---: | ---: |
| Files | 1,074,037 | 97.72% | 15.34 |
| Sites | 1,030,012 | 93.72% | 13.57 |
| New Connection | 1,098,930 | 99.99% | 15.79 |
| Transfers | 1,065,056 | 96.90% | 14.45 |
| Settings | 1,065,039 | 96.90% | 13.17 |
| Sync | 1,065,462 | 96.94% | 14.45 |
| Help & About | 1,061,201 | 96.55% | 13.31 |

These strict pixel proportions are dominated by locale, actual disconnected/empty device state, font rasterization and overall geometry. They are **not functional failure rates**. Do not replace actual captures with supplied concept images or fabricate cloud/server data. The newer 0.99.32 modal alignment change requires a fresh Windows and Linux capture before acceptance.
