# Ghost FTP native reference UI

Ghost FTP **0.0.5** uses focused native two-pane desktop layouts on Windows and Linux, a **macOS native development frontend** built with AppKit and a purpose-built native mobile development workspace on Android.

Windows and Linux are the current public desktop release surfaces. macOS and Android are active native development/source surfaces. This document distinguishes runtime/source parity from public distribution evidence so screenshots or development builds are never used to imply a release state that has not been proven.

This is a source/runtime contract, not a mockup specification. Visible controls must map to real capability and state.

## Canonical desktop workspace

The maintained desktop hierarchy consists of application/profile actions, Quick Connect, Local and Remote file panes, transfer actions, transfer queue and restrained status/version surfaces. The Remote side exposes normal file management plus **Remote Edit** for one supported regular text file.

Windows remains the canonical visual/behavior reference. Linux and macOS use native platform presentation while retaining shared engine/security/transfer semantics where the capability is maintained.

## Appearance contract

Classic Light is the fresh/missing/invalid-state primary appearance. Its maintained source palette uses workspace `#EEF1F5`, panel `#F6F8FB` and list `#FAFBFD`. Classic Light deliberately avoids pure white as the dominant application surface so file panes, application chrome and cards retain visible hierarchy without high-contrast glare.

Dark is the maintained explicit choice and uses workspace `#0B0F17`, panel `#121824` and list `#161D2A`. Both palettes are local source data; no remote stylesheet, font, theme or analytics service is loaded.

Windows applies appearance coherently across title bar, menus and controls. Linux applies the validated persisted palette before first frame and applies a newly saved palette immediately. macOS maps the maintained product hierarchy/palette intent through native AppKit chrome and accessibility behavior rather than simulating Win32 controls.

## Native dialog and lifecycle contract

Ghost FTP-owned Windows dialogs visually belong to the active appearance and must not terminate the application message loop when they close. Only the main desktop window owns process-level `WM_QUIT`/`PostQuitMessage` lifecycle.

Closing bounded dialogs such as **Nova mapa**, **Preimenuj**, **Postavke**, **Dijagnostika** or **O programu** closes only that dialog/session surface. Nested modal loops preserve a received process shutdown request instead of swallowing it.

Windows Settings is one application-owned modal surface for appearance, concurrency, independent upload/download bandwidth ceilings, connection timeout, retry policy, destination conflict policy and delete confirmation. Bandwidth fields state `KiB/s` and `0 = unlimited`.

Additional maintained rules include:

- Prompt, Option, Settings, information cards and Remote Edit close only their bounded modal/session loop;
- application-owned dialogs use the active Ghost FTP owner where available;
- controls, fonts and geometry scale from current DPI;
- 0.0.5 mutation/session guards prevent stale command re-entry while asynchronous profile, file or Remote Edit work is active.

Linux overlays own only their bounded overlay lifecycle and cannot fabricate engine state. macOS uses native AppKit window/sheet lifecycle and must not create parallel engine state merely because presentation is platform-native.

## Site Manager and saved profiles

![Ghost FTP Site Manager](images/ghost-ftp-site-manager.png)

Site Manager/profile workflows preserve explicit credential-consent and trust semantics. Windows and Linux can save non-secret profile state independently from newly entered credentials. The macOS development frontend exposes the maintained native profile workflow with Keychain-backed durable-secret protection. 0.0.5 additionally prevents parallel Windows encrypted profile-persistence mutations.

## Main Workspace

![Ghost FTP Main Workspace](images/ghost-ftp-main-workspace.png)

Actions are enabled from real state. A disabled operation remains disabled regardless of whether the user attempts to reach it by button, menu, list gesture or keyboard. Stale async callbacks are invalidated by connection/session identity where the runtime is asynchronous.

## File panes and transfer queue

Maintained desktop panes use shared engine/filesystem/remote validation layers. Sorting is shared for Name, Type, Size and Modified plus remote Permissions where maintained; directories remain first and filter+sort operates over copies of authoritative loaded snapshots.

Pause/resume/cancel/retry/clear operate through the transfer manager. Four-way queued **Top / Up / Down / Bottom** reorders queued scheduler slots only. Progress/speed/ETA may be shown only when backed by real state.

0.0.5 guards Windows local/remote create-directory, rename, delete and permission mutations against overlapping duplicate/stale commands without globally locking unrelated opposite-side workflow. Windows Add, Retry and Cancel-selected transfer completions are bound to the active `connectionGeneration`, so a callback from an obsolete connection cannot update the visible status/queue surface of a replacement session.

## Built-in Remote Editor

Remote Edit remains deliberately compact: one Edit action, one native/modal editor, **Save / Reload / Close**, dirty-state indication and conflict/error status. Windows uses an application-owned native editor; Linux uses the maintained X11/XWayland-compatible overlay; macOS development uses the native AppKit frontend while preserving shared conflict/read-back semantics.

0.0.5 keeps the Windows editor session busy from initial open through save/reload continuations until terminal close/error, preventing parallel re-entry during async cycles.

## Windows universal package and UI evidence boundary

The two public Windows downloads remain:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

They embed verified native x64, x86 and ARM64 payloads. The public x86 bootstrap selects the native processor through `GetNativeSystemInfo`, verifies staged payload bytes and performs no runtime architecture download.

Release metadata states:

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The Windows screenshots below and in CI are real runtime evidence for the architecture of the maintained Windows runner. They demonstrate the actual Windows UI at the exact tested source revision, but they are **not native ARM64 runtime evidence**. A future ARM64 runtime claim requires a maintained Windows ARM64 runner/device and separate source-bound execution evidence.

## macOS native development workspace

The active macOS AppKit frontend uses the shared `internal/api.Engine` and maintains real native actions for connection/disconnection, Site Manager, bookmarks, local/remote navigation, file mutations, filtering, bounded recursive search, directory comparison, permissions, Remote Edit, upload/download, transfer queue and shared settings.

The development app is built as a universal Intel + Apple Silicon artifact. Native macOS development validation is separate from the current Windows/Linux/Android screenshot evidence bundle.

A successful development build is **not** proof of a public macOS release. Public distribution requires the dedicated Developer ID signing/notarization path to succeed with real protected Apple credentials, including signature verification, accepted notarization, stapling/validation and Gatekeeper assessment.

## Android native workspace

Android 0.0.5 source exposes real **Files**, **Sites**, **Bookmarks**, **Transfers**, **Settings** and **About** surfaces plus semantic navigation. Small screens use an application drawer; wider layouts use a visible sidebar. System-bar insets keep controls outside reserved system UI.

Android supports FTP and strict explicit FTPS. Local access uses Android Storage Access Framework capabilities; passwords remain memory-only in the current source line; saved sites contain non-secret connection/navigation metadata; transfers use staged final-name commit/cancellation gating.

In 0.0.5 an in-flight connection belongs to one Activity instance. Destruction/recreation aborts its pending session and stale completion callbacks cannot reactivate an obsolete Activity UI. Authentication failures also discard server-controlled reply bodies before user-facing error text is produced.

SFTP remains absent from the Android protocol selector until strict native host-key identity verification exists.

## Settings and About evidence

![Ghost FTP Settings](images/ghost-ftp-settings.png)

![Ghost FTP About](images/ghost-ftp-about.png)

About displays runtime product/version identity generated from canonical build `VERSION`; current public desktop production identity is **Ghost FTP 0.0.5**, while development artifacts may carry platform-specific development identity according to their build contract.

## Authentic screenshot evidence

`.github/workflows/ui-screenshots.yml` captures real exact-head runtime UI. **Mockups, image-generation output and manually composed approximations are not accepted** as production UI evidence.

The current immutable cross-platform evidence contract is:

- Windows — 5 images: Main Workspace, Site Manager, Bookmarks, Settings, About;
- Linux — 3 images: Main Workspace, Bookmarks, Settings;
- Android — 7 images: Files, Navigation, Sites, Bookmarks, Transfers, Settings, About.

The final verifier requires exactly **15 runtime images**, verifies source/workflow identity, filenames, byte counts and SHA-256, and emits the read-only `ghostftp-authentic-ui-verified-bundle`. The workflow does **not** commit or push screenshots back to the tested branch.

Windows evidence is architecture-specific to the runner that produced it and must not be generalized into an unexecuted ARM64 runtime claim. macOS native development build/validation is maintained separately. It must not be silently counted as one of those 15 images or described as notarized public-release evidence unless the dedicated production distribution path actually succeeds.

Repository-local documentation assets remain:

- `images/ghost-ftp-main-workspace.png`;
- `images/ghost-ftp-site-manager.png`;
- `images/ghost-ftp-settings.png`;
- `images/ghost-ftp-about.png`.

See [Settings](SETTINGS.md), [Platform parity](PLATFORM-PARITY.md), [Testing](TESTING.md), [Privacy](PRIVACY.md) and [`../macos/README.md`](../macos/README.md).
