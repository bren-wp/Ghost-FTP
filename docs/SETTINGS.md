# Ghost FTP settings

Ghost FTP **0.0.3 development** treats settings as validated runtime policy rather than decorative UI state. Persisted values are accepted only within bounds enforced by the shared configuration layer, and visible controls must map to behavior in the shared engine rather than maintaining frontend-only shadow state.

## Current persisted settings

- `language` — canonical local UI language; invalid state normalizes to English.
- `appearance` — Windows appearance, `light` or `dark`; fresh/invalid state resolves to Classic Light.
- `parallelism` — concurrent transfers, range **1–8**, default **2**.
- `uploadLimitKiBPerSecond` — aggregate upload ceiling in **KiB/s**, range **0–1,048,576**, default **0 = unlimited**.
- `downloadLimitKiBPerSecond` — aggregate download ceiling in **KiB/s**, range **0–1,048,576**, default **0 = unlimited**.
- `connectionTimeoutSeconds` — range **5–60 seconds**, default **15**.
- `autoRetryCount` — range **0–3**, default **0**.
- `retryDelaySeconds` — range **1–30 seconds**, default **3**.
- `conflictPolicy` — canonical destination conflict behavior.
- `confirmDelete` — confirmation for user-initiated destructive operations.

Compatibility state such as older overwrite booleans may be normalized internally but must not become duplicate user-facing controls.

## Runtime ownership

Every exposed option has one explicit runtime owner:

| Option | Runtime effect |
| --- | --- |
| Parallel transfers | Bounds how many transfer workers may run concurrently. |
| Upload bandwidth | Applies the validated aggregate upload budget to actual FTP/FTPS/SFTP transport work. |
| Download bandwidth | Applies the validated aggregate download budget to actual FTP/FTPS/SFTP transport work. |
| Connection timeout | Bounds connection establishment and related connection work. |
| Automatic retries | Limits retries for failures classified as retryable. |
| Retry delay | Defines the bounded delay between eligible automatic retries. |
| Conflict policy | Selects skip, safe replace, or safe replace with retained recovery backup. |
| Delete confirmation | Controls user confirmation before destructive local/server deletion. |
| Appearance | Selects the maintained Windows Classic Light/Dark workspace. |
| Language | Selects one of the local 24-language catalogs with English fallback. |

The Windows and Linux settings surfaces consume the same shared model for options they expose. A frontend must not silently accept a value that the shared configuration layer rejects.

## Bandwidth policy

Bandwidth values are expressed in binary **KiB/s** (`1 KiB = 1024 bytes`) and are directional. Upload and download limits are independent. `0` is deliberately reserved for **unlimited**, which preserves the transfer behavior of settings files created by Ghost FTP 0.0.2 and older builds.

A non-zero value is an **aggregate ceiling for that direction**, not a per-transfer entitlement. The shared transfer scheduler divides the configured directional budget conservatively across all configured parallel worker slots. Idle slots do not temporarily lend their allowance to another transfer. This intentionally favors a stable aggregate ceiling over opportunistic bursting as jobs start and finish.

The resulting per-attempt cap is enforced by the transport itself:

- FTP and FTPS use curl's native `limit-rate` setting;
- SFTP uses OpenSSH `sftp -l`, converted to Kbit/s by flooring so conversion cannot round above the scheduler budget.

There is no application busy-wait loop and no UI-only timer pretending to throttle traffic.

A transfer attempt snapshots its effective bandwidth budget when that attempt starts. Saving a new limit never mutates or corrupts an already-running transport process. A new transfer, or a later retry attempt, samples the currently saved bandwidth settings and therefore observes the new limit.

## Compatibility and migration

Older or partial settings payloads are migrated only when a missing value can be distinguished safely from an explicit user value.

- Missing legacy `parallelism=0` migrates to the canonical default **2**, because valid user values begin at 1.
- Missing bandwidth fields deserialize as `0`, which is the canonical **unlimited** default and therefore needs no destructive migration.
- Missing connection timeout and retry delay continue to migrate to their canonical safe defaults.
- Explicit invalid parallelism such as a negative value or a value above 8 is still rejected rather than silently rewritten.
- Explicit negative bandwidth limits or values above **1,048,576 KiB/s** are rejected on save; corrupt persisted values normalize to unlimited rather than becoming an unintended throttle.
- Unknown persisted conflict-policy state fails closed to the conservative replace-with-recovery-backup behavior.
- Legacy overwrite booleans are synchronized from the one canonical `conflictPolicy` field when settings are saved.

Regression tests cover migration, independent upload/download values, aggregate allocation and continued rejection of explicit invalid values.

## Windows settings surface

Windows exposes one application-owned native Settings dialog for appearance, transfer concurrency, independent upload/download bandwidth ceilings, connection timeout, retry behavior, conflict policy and delete confirmation. Numeric values are validated before one complete settings candidate is persisted.

Bandwidth labels always state `KiB/s` and `0 = unlimited`. The native dialog accepts all six maintained numeric settings in one transaction rather than opening secondary prompts.

Invalid input keeps the dialog open, shows localized corrective text and restores keyboard focus to the invalid field instead of partially committing the remaining settings. A successful **OK** returns one complete settings candidate to the typed engine. Closing with **X** or **Cancel** closes only Settings and does not end the application message loop.

## Appearance

### Windows

- `light` — Classic Light and the fresh-install fallback.
- `dark` — the maintained Ghost FTP dark workspace.

An explicitly saved Dark preference is preserved. Appearance does not load remote fonts, styles, images or theme services.

### Linux

Linux uses the maintained Classic Light workspace until a complete runtime appearance switch can be provided without introducing a platform-only control whose behavior differs from Windows.

The Linux Settings overlay exposes the same upload/download bandwidth values as Windows. Because the X11 surface uses bounded steppers rather than free-form numeric text entry, bandwidth controls advance through maintained presets from unlimited up to the same validated maximum; persisted values that came from another supported surface remain valid and the next step moves to the adjacent bounded preset.

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

English is the default/fallback and the canonical registry contains **24 languages**. Localization is local and does not create online translation traffic. Bandwidth labels are maintained for all 24 languages and retain explicit `KiB/s` units in every locale.

## Credential persistence

Credential persistence is a per-save privacy decision rather than a hidden global toggle.

- Saving newly entered Windows profile credentials requires explicit consent.
- Declining consent can still save non-secret profile fields while removing stored credentials.
- Profile binding prevents old credentials from silently moving to a changed server/account/private-key identity.
- Linux session-only protected-secret handles remain session-only.

## Delete confirmation

Delete confirmation defaults to enabled. Destructive actions must respect the validated shared setting.

## Button and option quality rule

A control is not considered implemented merely because it is visible. Main desktop controls are covered by a regression contract that compares the Windows button IDs with their command handlers and the Linux rendered control rectangles with their click handlers. Settings changes additionally require a backend validation path and tests proving the setting changes runtime behavior or policy.

Directory comparison/synchronized browsing, recursive search/filter and queue priority are implemented maintained capabilities. Future power-user options remain roadmap items until their complete engine + Windows + Linux + localization + test path exists; they must not appear as decorative or non-functional switches.

## Persistence and recovery

Settings are stored in bounded local state with safe replacement/recovery behavior. Loaded data is normalized before it becomes effective runtime policy, and the state-directory identity is pinned so later pathname replacement cannot silently redirect settings/profile I/O.

## Option design rule

One behavior has one canonical setting. A new option is release-ready only when it has a clear runtime owner, safe bounded default, migration behavior, honest platform exposure and localized user-facing copy where required.

See [Architecture](ARCHITECTURE.md), [Privacy](PRIVACY.md), [Security](SECURITY.md), [Testing](TESTING.md) and [Localization](LOCALIZATION.md).