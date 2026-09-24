# Ghost FTP Android

Native Android application source for Ghost FTP.

This Android app lives in `/android` so the Windows and Linux desktop release line remains isolated. The product identity stays aligned with Ghost FTP and Brendigo:

- Product: Ghost FTP
- Brand: Brendigo
- Version: 2.1.1-rc.22
- Display: 2.1.1 RC22
- Build: 2026.09.24.22

## Android surface

The Android surface is a native Kotlin single-activity app with a mobile layout that follows the desktop product structure:

- Header with Ghost FTP RC22 release identity.
- Session status card.
- New connection card for FTP, explicit FTPS and SFTP endpoint control.
- SFTP host key fingerprint field for explicit server identity verification.
- Remote workspace card for current server context and remote folder listing.
- Transfer queue card for mobile-first queue state.
- Brendigo footer.
- Passwords kept in memory only for the current action and cleared on disconnect.
- No analytics, no telemetry and no required account sign-in.

## Protocol support

The Android app uses native protocol clients:

- FTP remote login and folder listing.
- Explicit FTPS remote login and protected data-channel listing.
- SFTP remote login and folder listing with SHA-256 host key fingerprint verification.

Credentials are passed only into the active connection action. The app does not add telemetry, accounts or ordinary password persistence.

## Build

From the repository root:

```bash
gradle -p android lintDebug lintRelease assembleDebug assembleRelease
```

The pull-request Android workflow runs the Android contract, both lint variants and both APK builds. It uploads installable RC22 APK artifacts and checksums for review.

## Release rule without external keys

Every next GitHub release must include an Android APK asset. The dedicated workflow `.github/workflows/ghostftp-android-release.yml` builds from the published tag, creates an installable APK without requiring GitHub signing secrets, verifies the APK signature and uploads both the APK and its SHA-256 checksum to that release.

No repository keystore, GitHub secret or manual signing key is required for this release path.

For long-term Android upgrade continuity, a stable signing key can be introduced later. Without a stable saved signing key, Android may treat APKs from different release runs as separately signed builds.
