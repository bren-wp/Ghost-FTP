# Ghost FTP Android

Native Android application source for Ghost FTP.

This Android app lives in `/android` so the Windows and Linux RC21 desktop release line remains isolated. The product identity stays aligned with Ghost FTP and Brendigo:

- Product: Ghost FTP
- Brand: Brendigo
- Version: 2.1.1-rc.21
- Display: 2.1.1 RC21
- Build: 2026.09.24.21

## Current Android surface

The first Android surface is a native Kotlin single-activity app with a mobile layout that follows the desktop product structure:

- Header with Ghost FTP RC21 release identity.
- Session status card.
- New connection card for FTP, explicit FTPS and SFTP endpoint control.
- Remote workspace card for current server context.
- Transfer queue card for mobile-first queue state.
- Brendigo footer.
- Passwords kept in memory only for the current action and cleared on disconnect.
- No analytics, no telemetry and no required account sign-in.

## Build

From the repository root:

```bash
gradle -p android assembleDebug
```

The GitHub workflow installs the Android SDK and runs:

```bash
gradle -p android lintDebug assembleDebug
bash android/scripts/check-android-contract.sh
```

## Release gate

Android is not published from the desktop RC21 release workflow. An Android release must pass its own native build, UI review, protocol acceptance and mobile security checks before any APK/AAB is published.

The desktop release remains authoritative for Windows and Linux until the Android protocol engine, Android QA evidence and release workflow are approved.
