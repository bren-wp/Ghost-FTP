# Ghost FTP settings

Ghost FTP **0.0.1** treats settings as validated runtime policy rather than decorative UI state. Persisted values are accepted only within bounds enforced by the shared configuration layer.

## Current persisted settings

- `language` — canonical local UI language; invalid state normalizes to English.
- `appearance` — Windows appearance, `light` or `dark`; fresh/invalid state resolves to Classic Light.
- `parallelism` — concurrent transfers, range **1–8**, default **2**.
- `connectionTimeoutSeconds` — range **5–60 seconds**, default **15**.
- `autoRetryCount` — range **0–3**, default **0**.
- `retryDelaySeconds` — range **1–30 seconds**, default **3**.
- `conflictPolicy` — canonical destination conflict behavior.
- `confirmDelete` — confirmation for user-initiated destructive operations.

Compatibility state such as older overwrite booleans may be normalized internally but must not become duplicate user-facing controls.

## Windows settings surface

Windows exposes one application-owned native Settings dialog for appearance, transfer concurrency, connection timeout, retry behavior, conflict policy and delete confirmation. Numeric values are validated before one complete settings candidate is persisted.

Invalid input keeps the dialog open, shows localized corrective text and restores keyboard focus to the invalid field instead of partially committing the remaining settings. A successful **OK** returns one complete settings candidate to the typed engine. Closing with **X** or **Cancel** closes only Settings and does not end the application message loop.

## Appearance

### Windows

- `light` — Classic Light and the fresh-install fallback.
- `dark` — the maintained Ghost FTP dark workspace.

An explicitly saved Dark preference is preserved. Appearance does not load remote fonts, styles, images or theme services.

### Linux

Linux uses the maintained Classic Light workspace until a complete runtime appearance switch can be provided without introducing a platform-only control whose behavior differs from Windows.

## Fresh connection protocol

A fresh/quick connection starts on **explicit FTPS, port 21**.

- FTPS stays selected unless the user explicitly changes it or loads a profile.
- SFTP remains available with strict host-key verification/pinning.
- Plain FTP remains explicit legacy compatibility.
- Failed FTPS negotiation is never silently retried as plain FTP.
- Saved profiles restore their explicitly stored protocol/port.

## Conflict policy

Canonical values are:

- `skip` — leave an existing destination untouched;
- `replace` — replace through the supported safe transfer/commit path;
- `replace_backup` — replace through the safe path while retaining supported recovery backup behavior.

Unknown values fail to a safe/default policy rather than silently enabling destructive behavior.

## Retry policy

Automatic retry applies only to errors classified as retryable by the shared engine. Validation failures, trust failures, unsafe paths and explicit cancellation must not become blind retry loops. Retry count/delay remain bounded and tied to connection identity/generation.

## Language

English is the default/fallback and the canonical registry contains **24 languages**. Localization is local and does not create online translation traffic.

## Credential persistence

Credential persistence is a per-save privacy decision rather than a hidden global toggle.

- Saving newly entered Windows profile credentials requires explicit consent.
- Declining consent can still save non-secret profile fields while removing stored credentials.
- Profile binding prevents old credentials from silently moving to a changed server/account/private-key identity.
- Linux session-only protected-secret handles remain session-only.

## Delete confirmation

Delete confirmation defaults to enabled. Destructive actions must respect the validated shared setting.

## Persistence and recovery

Settings are stored in bounded local state with safe replacement/recovery behavior. Loaded data is normalized before it becomes effective runtime policy, and the state-directory identity is pinned so later pathname replacement cannot silently redirect settings/profile I/O.

## Option design rule

One behavior has one canonical setting. A new option is release-ready only when it has a clear runtime owner, safe bounded default, migration behavior, honest platform exposure and localized user-facing copy where required.

See [Architecture](ARCHITECTURE.md), [Privacy](PRIVACY.md), [Security](SECURITY.md) and [Localization](LOCALIZATION.md).
