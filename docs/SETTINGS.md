# Ghost FTP settings

Ghost FTP **0.0.8** treats settings as validated runtime policy rather than decorative UI state. Persisted values are accepted only within shared bounds, and visible controls map to real engine/runtime behavior.

Windows and Linux share the desktop settings model. Android 0.0.8 is a public native application with platform-appropriate settings and SAF/lifecycle constraints; unsupported desktop-only controls are not fabricated. macOS is a maintained native AppKit frontend and public 0.0.8 release target; it consumes the shared model where the native surface implements the option.

## Current shared desktop settings

- `language` — local UI language; invalid state normalizes to English.
- `appearance` — `dark` or `light`; fresh/invalid state resolves to Dark.
- `parallelism` — concurrent transfers, **1–8**, default **2**.
- `uploadLimitKiBPerSecond` — aggregate upload ceiling, **0–1,048,576 KiB/s**, default **0 = unlimited**.
- `downloadLimitKiBPerSecond` — aggregate download ceiling, same range/default.
- `connectionTimeoutSeconds` — **5–60 seconds**, default **15**.
- `autoRetryCount` — **0–3**, default **0**.
- `retryDelaySeconds` — **1–30 seconds**, default **3**.
- `conflictPolicy` — canonical destination conflict behavior.
- `confirmDelete` — confirmation for user-initiated destructive operations.

## Runtime ownership

A non-zero bandwidth value is the aggregate ceiling for that direction. The scheduler divides budgets conservatively across configured worker slots; running attempts snapshot their effective budget at start. FTP/FTPS use curl `limit-rate`; desktop SFTP uses OpenSSH `sftp -l` with conservative conversion.

Windows exposes **one application-owned native Settings dialog** for **language**, appearance, concurrency, independent upload/download bandwidth, timeout, retries, conflict policy and delete confirmation. Language is intentionally not shown in the main navigation rail. **Restore defaults** loads the canonical safe values into the dialog without saving them; Apply is still the persistence boundary. **Invalid input keeps the dialog open**; Cancel/X discards the pending candidate.

Linux exposes the same validated policy through the maintained native X11/XWayland-compatible overlay; language and product utility actions are kept inside Settings rather than the master rail. macOS uses native AppKit controls backed by the same typed settings model, including language, draft-only Restore Defaults and fail-closed delete confirmation when settings cannot be read.

## Update, download and Premium actions

Across Windows, Linux, Android and macOS, **Update** is an explicit local simulation. It may animate progress and report that the current build is updated, but it does not rewrite the installed binary, falsify the build version or contact a release API. Installing a genuinely newer signed build remains a separate download/install action.

**Download latest**, **Premium** and **Official website** open only fixed HTTPS destinations on `ghostftp.com`. There is no GitHub/user-facing release link, no passive startup update request and no transmission of FTP credentials, usernames, server paths, local file paths or transfer metadata.

## Android settings boundary

Android 0.0.8 keeps native settings appropriate to its current public client: Dark/Light appearance, opt-in Quick Connect metadata persistence, file-size display, delete confirmation, Restore app defaults, local Update simulation and official-site Download/Premium/Website actions. Restore defaults clears remembered Quick Connect metadata while keeping saved connections and the user-selected SAF folder authority. Connection/storage choices preserve strict FTPS verification, Storage Access Framework authority, transfer/lifecycle ownership and local-only non-secret saved-site state. **SFTP remains hidden** until strict maintained native host-key verification exists.

## Appearance and localization

Dark is the fresh/fallback desktop appearance and Light is a maintained secondary choice. Both are local source-defined palettes with no remote theme/font/style dependency. English is the default/fallback and the shared desktop registry contains **24 languages**.

## Fresh connection and retry policy

A fresh desktop quick connection starts on **explicit FTPS, port 21**. Plain FTP remains an explicit compatibility choice; failed FTPS is never silently retried as FTP. Desktop SFTP requires strict host-key verification/pinning.

Canonical conflict values are `skip`, `replace` and `replace_backup`. Automatic retry applies only to retryable failures; trust, validation, unsafe-path and explicit-cancel failures do not become blind retry loops.

## Credential persistence

Credential persistence is separate from ordinary non-secret settings persistence. Windows uses its maintained current-user protected-secret boundary; Linux retains explicit persistence consent plus trusted AskPass provenance; macOS development profiles use the maintained Keychain-backed path; Android saved-site state remains intentionally non-secret.

## 0.0.8 lifecycle note

Ghost FTP 0.0.8 retains the Windows profile/file/Remote Edit in-flight guards and connection-generation ownership that prevent stale asynchronous completion from mutating replacement state. Android binds connection and transfer completion to Activity/session generations. These are runtime correctness guarantees rather than user-configurable toggles.

See [Architecture](ARCHITECTURE.md), [Privacy](PRIVACY.md), [Security](SECURITY.md), [Testing](TESTING.md) and [Localization](LOCALIZATION.md).
