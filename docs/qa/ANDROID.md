# Ghost FTP Android QA

## Scope

Android is a native Kotlin application and uses the same Ghost FTP product identity, protocol terminology and core Files/Sites/Transfers model as the Windows/Linux desktop application, adapted for touch/mobile layout.

## Automated exact-SHA gate

The **Ghost FTP Android** workflow runs for every pull request to `main` and every `main` push.

It verifies:

- product/version/build metadata against root `version.json`;
- FTP, explicit FTPS and SFTP production contract;
- strict SFTP host-key verification and FTPS protected data channel configuration;
- timeout/session/channel cleanup rules;
- guarded destructive/upload actions and remote-path validation;
- debug, release and installable preview lint/build;
- maintained protocol dependency contract;
- installable preview APK signature/package identity, versionCode/versionName, minSdk/targetSdk and non-debuggable state;
- intentionally unsigned release-check APK when persistent signing is unavailable on a preview cycle;
- a fail-closed stable-channel signing gate: any non-preview channel requires all four persistent `ANDROID_RELEASE_*` secrets;
- signed stable APK signature, `com.ghostftp.android` identity, version and non-debuggable verification when stable signing is enabled;
- emulator instrumentation click-through smoke;
- clean install, same-build reinstall and launch for the preview candidate, plus the exact persistently signed stable APK when present;
- real emulator UI screenshot artifacts for debug, preview candidate and signed stable release where applicable.

## Emulator click-through smoke

The test opens the real `MainActivity` and checks:

1. Ghost FTP application shell renders.
2. desktop-aligned Files workspace navigation renders.
3. primary toolbar actions render.
4. Connect with missing host produces the expected validation state.
5. Disconnect returns the application to Ready/idle state.
6. Refresh in idle state is safe.
7. Download without a connection is blocked with “Connect first”.
8. New Folder without a connection is blocked with “Connect first”.
9. Delete without a connection is blocked with “Connect first”.

The source contract additionally requires Files, Sites, Transfers, Settings and Help & About navigation and verifies upload/document-picker wiring.

After instrumentation, CI removes any existing preview package, installs the exact `app-preview.apk`, immediately exercises `adb install -r` with that same APK, launches its declared activity, requires the activity to reach RESUMED state and captures a second screenshot. This closes the previous gap where only `app-debug.apk` was actually installed and launched.

## Release artifact

The Android gate always creates the non-debuggable isolated preview APK as CI evidence. On a preview channel without persistent release signing it also retains a clearly named unsigned release-check artifact.

For a stable/non-preview channel, the Android gate now fails immediately unless all four persistent `ANDROID_RELEASE_*` secrets are available. With signing enabled it additionally creates and verifies the persistently signed `com.ghostftp.android` APK, exercises clean install, same-signature reinstall and launcher/RESUMED smoke on the emulator, records its certificate SHA-256 digest and makes that signed package the only acceptable canonical stable Android input.

The canonical release workflow downloads the verified Android artifact bundle for the exact release SHA and, on the stable channel, normalizes only the signed `com.ghostftp.android` APK into `GhostFTP-Android-v<version>.apk`. Never commit a keystore or signing password to the repository.

## 0.20.0 transfer/lifecycle regression coverage

The Android transfer layer stages downloads locally and uploads remotely before promotion, preserving existing targets until a completed replacement is ready. Activity destruction and disconnect propagate cooperative cancellation through upload, download, delete and folder creation. Upload document streams are closed at the controller ownership boundary even when cancellation occurs before protocol setup.

The document picker still requests persistable URI access, while `takePersistableUriPermission` is called only with valid READ/WRITE grant modes. Release signing continuity is fail-closed once a previous stable `com.ghostftp.android` APK exists: inability to download, verify or inspect that APK blocks publication instead of silently skipping the comparison.

## Stable / FINAL acceptance

Before a stable/FINAL claim, also test on representative physical Android devices:

- install/upgrade/uninstall lifecycle;
- Android document providers and Downloads storage;
- background/interruption behavior;
- real FTP/FTPS/SFTP servers over Wi-Fi/mobile/VPN as applicable;
- accessibility, font scaling and rotation;
- production signing-key continuity.
