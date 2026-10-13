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
- intentionally unsigned production `com.ghostftp.android` release-check APK on every cycle;
- unsigned production APK package identity, version, SDK levels, launcher and non-debuggable verification;
- emulator instrumentation click-through smoke;
- clean install, same-build reinstall and launch for the non-debuggable preview candidate;
- real emulator UI screenshot artifacts for debug and the preview candidate.

## Emulator click-through smoke

The test opens the real `MainActivity` and checks:

0. The app starts in the Files workspace with a persistent left navigation rail; non-active workspaces are hidden rather than stacked in one long scroll.
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

The Android gate always creates two distinct artifacts: a non-debuggable installable preview used for clean-install/reinstall/launcher smoke, and an intentionally unsigned production `com.ghostftp.android` package whose identity/version/SDK/launcher metadata are verified. No private release keystore or signing secret is required.

The canonical release workflow downloads both artifacts for the exact release SHA. It publishes the production build as `GhostFTP-Android-v<version>.apk.unsigned` and keeps the tested installable preview separately as `GhostFTP-Android-v<version>-Installable-Preview.apk`. Android requires a signature for installation, so the unsigned production file is not described as directly installable.

## 0.20.0 transfer/lifecycle regression coverage

The Android transfer layer stages downloads locally and uploads remotely before promotion, preserving existing targets until a completed replacement is ready. Activity destruction and disconnect propagate cooperative cancellation through upload, download, delete and folder creation. Upload document streams are closed at the controller ownership boundary even when cancellation occurs before protocol setup.

The document picker still requests persistable URI access, while `takePersistableUriPermission` is called only with valid READ/WRITE grant modes. Android release validation no longer has a second optional production-signing code path, reducing CI/release branching while preserving package metadata and emulator installability checks.

## Stable / FINAL acceptance

Before a stable/FINAL claim, also test on representative physical Android devices:

- install/upgrade/uninstall lifecycle;
- Android document providers and Downloads storage;
- background/interruption behavior;
- real FTP/FTPS/SFTP servers over Wi-Fi/mobile/VPN as applicable;
- accessibility, font scaling and rotation;
- installation behavior for the chosen distribution/signing method, if a future production signing policy is introduced.

## 0.95.0 — protected 480 × 960 premium UI evidence

Normal Android `adb screencap` can be black because `MainActivity` deliberately keeps `FLAG_SECURE`. A black screenshot from CI is **not** valid visual evidence. `PremiumVisualReferenceTest.kt` now draws the authentic empty Activity view hierarchy only in debug instrumentation and rejects unexpected non-default editable values before writing any bytes.

The Android workflow configures its isolated test emulator to **480 × 960 pixels at 160dpi**, checks actual **64dp navigation rail / 58dp brand bar** bounds and captures seven real empty workspaces: Files, Sites, Transfers, Sync, Cloud, Settings and About. A separate artifact `GhostFTP-Android-Premium-Visual-QA-480x960` contains seven PNGs, the instrumentation log and SHA-256 hashes. It is explicitly **not** distributed as an app release asset.

Captured empty states are not automatically equivalent to the supplied concept screens; the concept images include illustrative connections and file rows which must never be faked. Of the 16 Android reference screens, six share names with these seven empty workspaces; Cloud is an additional product screen without an approved reference image. The remaining dialog, connected and transfer-detail scenarios still require real data from approved test systems, explicit per-screen visual comparisons and functional tap/accessibility acceptance.

## 0.95.10 reusable connection workflow

Sites now separates immediate Connect from Save new connection profile. The saved Sites chooser pre-fills only protocol, host, port, username, trusted host-key fingerprint, remote path and keep-alive; its explicit load action clears the previous password. Profile metadata is bounded in app-private preferences and never includes the session password. No saved-profile selection initiates a network connection; the user must enter credentials and tap Connect. On-device rotation, restore, lifecycle, keyboard and long-list acceptance remains required.
