# Ghost FTP platform parity

Last reviewed: 2026-09-30
Source version: 0.20.0
Primary platforms: Windows, Linux, Android

This file is a QA tracker, not a marketing claim. A platform or capability is only marked **CI verified** when the repository contains an automated gate for the exact build or behavior. **Implemented** means code exists but the full requested compatibility matrix is not yet proven. **Unverified** means Ghost FTP must not claim support until evidence exists.

## Current matrix

| Area | Windows | Linux | Android | Evidence / limitation |
| --- | --- | --- | --- | --- |
| Production build | CI verified | CI verified | CI verified | Native build matrix covers Windows and Ubuntu; Android workflow builds debug, release-check and installable preview variants. |
| Install / launch | CI verified | CI verified | CI verified for current release candidate | Windows installer lifecycle and Linux package lifecycle live in native-build CI. Android installs and launches the exact `app-preview.apk` that release consumes. |
| Same-build reinstall | Implemented | Implemented | CI verified | Android explicitly runs `adb install -r` against the exact preview artifact. |
| Cross-release upgrade signing continuity | Depends on existing release signing configuration | Depends on existing release signing configuration | **Unverified / release blocker for upgrade claims** | Android preview currently uses the CI debug signing identity. A persistent signing identity is required to prove upgrades across separately generated releases. Never commit a keystore or password. |
| FTP | Implemented + protocol CI | Implemented + protocol CI | Implemented | Broader server/OS interoperability matrix remains ongoing. |
| Explicit FTPS | Implemented + protocol CI | Implemented + protocol CI | Implemented | TLS validation must remain enabled; broader server matrix remains ongoing. |
| Implicit FTPS | Implemented on desktop paths where supported | Implemented on desktop paths where supported | Unverified | Do not claim Android parity until a real test proves it. |
| SFTP | Implemented + protocol CI | Implemented + protocol CI | Implemented | Android contract requires strict host-key checking and fingerprint verification. |
| Transfer queue | Implemented + durable recovery | Implemented + durable recovery | Partial | Desktop persists credential-free transfer rows/retry descriptors across restart/update and safely reconnects saved profiles on explicit retry. Android lifecycle recovery remains a separate mobile workstream. |
| Byte-range resume | Implemented + protocol CI for FTP/explicit FTPS/SFTP paths covered by the suite | Implemented + protocol CI for FTP/explicit FTPS/SFTP paths covered by the suite | Partial | Real E2E covers desktop FTP, explicit FTPS and SFTP pause/resume/cancel behavior. Implicit FTPS and broader server matrices remain unverified for full parity claims. |
| Windows console-free GUI helpers | CI verified | N/A | N/A | Advanced PATH lookup is shell-free; audited GUI helper launches use `CREATE_NO_WINDOW` / `hide_console`. |
| Files > More actions | CI contract | CI contract | N/A | Existing Rename / Properties / Delete intent retained. Portal positioning, focus return and keyboard navigation are guarded without redesigning the UI. |

## OS coverage policy

### Windows

- Windows 10 and Windows 11 are the quality baseline.
- Windows 8 / 8.1 remain **unverified**. Do not claim support merely because a binary can be produced.
- Legacy support must not weaken TLS, SSH, updater, WebView or runtime security.
- The native-build gate must keep installer lifecycle and native-window evidence enabled.

### Linux

Current native CI uses Ubuntu 22.04 and builds the configured DEB, RPM and AppImage bundles. This proves the build pipeline, not every distribution.

The following remain distribution-validation targets rather than blanket support claims until tested on representative systems:

- Ubuntu LTS
- Debian
- Linux Mint
- Fedora
- Pop!_OS
- Arch / Manjaro
- openSUSE
- GNOME / KDE
- Wayland / X11

### Android

Current project contract:

- `minSdk = 26`
- `targetSdk = 35`
- release-candidate package: `com.ghostftp.android.preview`
- canonical source of version metadata: root `version.json`

The Android CI must validate the exact release candidate with:

1. `apksigner verify`;
2. `aapt dump badging`;
3. package ID, versionCode, versionName, minSdk and targetSdk checks;
4. clean `adb install`;
5. `adb install -r`;
6. explicit launcher start;
7. RESUMED activity state;
8. live PID;
9. screenshot evidence.

A green Gradle build by itself is not sufficient.

## Promotion rules

A row may move from **Partial** or **Unverified** only when the relevant code and a repeatable test exist. Do not weaken a failing test to promote a status. Do not infer broad OS support from one CI image, and do not infer Android upgrade compatibility from a clean install.
