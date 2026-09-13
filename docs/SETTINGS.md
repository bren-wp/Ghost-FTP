# Ghost FTP settings

Ghost FTP **0.0.5** treats settings as validated runtime policy rather than decorative UI state. Persisted values are accepted only within bounds enforced by the shared configuration layer, and visible controls map to real engine/runtime behavior.

Windows and Linux are the current public desktop release platforms. The active macOS development frontend consumes the shared desktop settings model where the native surface exposes the corresponding option. Android uses its own native settings/preferences boundary and must not be given unsupported desktop-setting parity claims.

## Current shared desktop settings

- `language` — local UI language; invalid state normalizes to English.
- `appearance` — maintained desktop `light` or `dark`; fresh/invalid state resolves to Classic Light.
- `parallelism` — concurrent transfers, **1–8**, default **2**.
- `uploadLimitKiBPerSecond` — aggregate upload ceiling, **0–1,048,576 KiB/s**, default **0 = unlimited**.
- `downloadLimitKiBPerSecond` — aggregate download ceiling, same range/default.
- `connectionTimeoutSeconds` — **5–60 seconds**, default **15**.
- `autoRetryCount` — **0–3**, default **0**.
- `retryDelaySeconds` — **1–30 seconds**, default **3**.
- `conflictPolicy` — canonical destination conflict behavior.
- `confirmDelete` — confirmation for user-initiated destructive operations.

## Runtime ownership

| Option | Runtime effect |
| --- | --- |
| Parallel transfers | Bounds concurrently running transfer workers. |
| Upload/download bandwidth | Applies validated aggregate directional budgets to actual transport work. |
| Connection timeout | Bounds connection establishment work. |
| Automatic retries / retry delay | Bounds eligible retry attempts and spacing. |
| Conflict policy | Selects skip, safe replace, or replace with recovery backup. |
| Delete confirmation | Controls confirmation before destructive local/server deletion. |
| Appearance | Selects the maintained Classic Light or Dark desktop palette. |
| Language | Selects one of 24 local desktop catalogs with English fallback. |

Windows and Linux consume the same validated settings model. The macOS development frontend consumes the same typed model for maintained parity controls. Native frontends must not silently accept values rejected by shared configuration.

## Bandwidth policy

Bandwidth is expressed in binary KiB/s. Upload and download values are independent aggregate directional ceilings and `0 = unlimited`. Each non-zero configured value is the aggregate ceiling for that direction. The scheduler divides non-zero budgets conservatively across configured worker slots; idle slots do not create an undocumented burst entitlement.

FTP/FTPS enforce effective limits through curl `limit-rate`; SFTP uses OpenSSH `sftp -l` with conservative unit conversion on maintained shared-engine desktop paths. A running attempt snapshots its budget at start. Saving settings affects future/retried attempts without mutating an already-running transport process.

## Compatibility and migration

- Missing legacy `parallelism=0` migrates to default 2; explicit negative/above-range values remain invalid.
- Missing bandwidth fields remain unlimited (`0`).
- Corrupt negative/above-maximum bandwidth values normalize safely rather than becoming unintended throttles.
- Missing timeout/retry delay values use canonical defaults.
- Unknown conflict policy fails closed to conservative recovery behavior.
- Invalid/missing appearance state resolves to Classic Light.

Migration exists to preserve safe user intent, not to accept arbitrary invalid state.

## Windows settings surface

Windows exposes one application-owned native Settings dialog for appearance, concurrency, upload/download bandwidth, timeout, retry behavior, conflict policy and delete confirmation. Numeric values are validated before one complete candidate is persisted. Invalid input keeps the dialog open; Cancel/X discards pending values.

## Linux settings surface

Linux exposes the same runtime policy through the maintained native X11/XWayland-compatible Settings overlay. Bounded steppers/presets preserve the same shared ranges and `0 = unlimited` semantics.

## macOS development settings surface

The native AppKit development frontend exposes the maintained shared settings contract through native controls, including the shared 24-language registry, Classic Light/Dark appearance, transfer parallelism, independent bandwidth limits, timeout, retries, conflict policy and delete confirmation.

Those settings use the same typed model; macOS UI code must not create a parallel security/transfer interpretation. This is development/source parity and does not imply that a public notarized macOS release has been published.

## Android settings boundary

Android has platform-specific native settings appropriate to the current development client. Desktop configuration fields are not automatically promised on Android merely because they exist in the shared desktop model.

Android-specific connection/storage choices must preserve strict FTPS verification, Storage Access Framework authority and lifecycle ownership. SFTP remains hidden until strict native host-key verification is implemented.

## Appearance

Classic Light is the fresh/fallback desktop appearance. Dark is a maintained explicit choice. Both are local source-defined palettes with no remote theme/font/style dependency. Linux applies persisted appearance before first paint; Windows maintains coherent title-bar/control theming; macOS maps the maintained product appearance contract through native AppKit behavior.

## Fresh connection protocol

A fresh desktop quick connection starts on **explicit FTPS, port 21**. SFTP remains available on maintained desktop implementations with strict host-key verification/pinning. Plain FTP remains explicit compatibility. Failed FTPS is never silently retried as FTP.

## Conflict and retry policy

Canonical conflict values are `skip`, `replace` and `replace_backup`. Automatic retry applies only to retryable failures; trust, validation, unsafe-path and explicit-cancel failures do not become blind retry loops.

## Language

English is the default/fallback and the canonical shared desktop registry contains **24 languages**. Localization is local and creates no online translation traffic.

## Credential persistence

Credential persistence is a per-save privacy decision and remains distinct from ordinary non-secret settings persistence.

- Windows uses the maintained current-user protected-secret boundary.
- Linux requires bounded explicit confirmation before newly entered password/private-key-passphrase material is persisted and retains trusted AskPass provenance as a separate runtime boundary.
- macOS development profiles use the maintained native Keychain-backed protection path rather than serializing secrets as ordinary settings.
- Android current saved-site state remains intentionally non-secret.

## 0.0.5 lifecycle note

The 0.0.5 Windows profile-save/delete path adds an in-flight mutation guard. While encrypted profile persistence is active, duplicate profile mutations are rejected and application close does not terminate that operation mid-write. This is lifecycle protection, not a new user-facing setting.

Android 0.0.5 similarly binds pending connection state to the owning Activity lifecycle, while macOS maintains native frontend lifecycle ownership around shared-engine operations. These are runtime correctness guarantees, not configurable toggles.

See [Architecture](ARCHITECTURE.md), [Privacy](PRIVACY.md), [Security](SECURITY.md), [Testing](TESTING.md) and [Localization](LOCALIZATION.md).
