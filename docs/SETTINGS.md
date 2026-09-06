# Ghost FTP settings

Ghost FTP **1.1.0 Stable** treats settings as validated runtime policy, not decorative UI state. A persisted option is accepted only within the bounds enforced by `internal/config/settings.go`.

## Current persisted settings

- `language` — canonical local UI language; invalid stored values normalize to English.
- `appearance` — Windows desktop appearance: `dark` or `light`; missing/invalid legacy state normalizes safely to `dark`.
- `parallelism` — concurrent transfers, range **1–8**, default **2**.
- `connectionTimeoutSeconds` — connection establishment timeout, range **5–60 seconds**, default **15**.
- `autoRetryCount` — automatic retries, range **0–3**, default **0**.
- `retryDelaySeconds` — retry delay, range **1–30 seconds**, default **3**.
- `conflictPolicy` — destination conflict behavior.
- `backupBeforeOverwrite` / `skipExisting` — compatibility mirrors derived from `conflictPolicy` for older state/readers; they are not separate user-facing choices.
- `confirmDelete` — explicit confirmation for user-initiated destructive operations.

Corrupt or unavailable state does not select a less-safe policy. Defaults remain bounded and conservative.

## Appearance

Ghost FTP deliberately exposes only one appearance decision rather than separate background, accent, icon, list and button color switches.

### Windows

- `dark` — the established Ghost FTP dark workspace.
- `light` — **Classic Light**, a bright neutral two-pane workspace inspired by the clarity of traditional professional FTP clients while using Ghost FTP's own branding, iconography and palette.

The Windows selection is persisted locally and is applied on the next application start. This is intentional: native title bar, menu, combo, edit, list/header and owner-drawn control styling are selected before the complete window tree is created, preventing mixed-theme fragments and avoiding runtime repaint races.

### Linux

The native Linux desktop uses the **Classic Light** palette as the canonical 1.1 workspace. No extra Linux appearance toggle is exposed until complete runtime switching can be provided without introducing redraw/race complexity. This keeps the settings surface honest and avoids a control whose backend behavior would differ from its label.

Appearance changes do not load remote styles, fonts, images or theme services and do not create network traffic.

## Conflict policy

The canonical conflict values are:

### `skip`

Leave an existing destination untouched.

### `replace`

Replace through the supported safe transfer/commit path without intentionally keeping a recovery backup after success.

### `replace_backup`

Replace through the safe path and retain recovery-oriented backup behavior where supported. This is the conservative default.

Legacy boolean combinations are migrated deterministically to one of these canonical states. Unknown stored values fail back to the safe default; unknown new values are rejected.

## Retry policy

Automatic retry is for errors classified as retryable by the shared remote/transfer layer. Validation failures, trust failures, unsafe paths and explicit cancellation must not be turned into blind retry loops.

Retry count/delay remain bounded to avoid accidental server hammering.

## Connection timeout

The configured timeout applies to connection establishment. It is validated identically for Windows and Linux. Transfer cancellation/progress uses its own lifecycle and is not fabricated from the connection-timeout value.

## Language

English is default/fallback and the canonical registry contains 24 languages. Unsupported locale values never create online translation traffic. See [Localization](LOCALIZATION.md).

## Delete confirmation

Delete confirmation is a safety feature and defaults to enabled. Frontends must route destructive actions through the shared validated setting rather than bypassing it with platform-specific shortcuts.

## Persistence/recovery

Settings storage uses bounded local files and safe replacement/recovery logic. Loaded data is normalized before becoming effective runtime policy. A failed save must not leave partially trusted in-memory state masquerading as persisted configuration.

## Avoiding duplicate options

One behavior must have one canonical setting. Compatibility fields may remain in serialized state for migration, but they must not become duplicate UI controls. In particular, destination conflict handling is represented by `conflictPolicy`; the old overwrite booleans are compatibility mirrors only.

A new option is release-ready only when it has:

1. one clear runtime owner;
2. a safe/bounded default;
3. migration behavior for old state;
4. honest platform exposure where the backend is complete;
5. localized label/help copy;
6. regression tests for accepted/rejected values;
7. no path that weakens certificate validation, SFTP host-key trust, local containment, secret protection or privacy policy.

Security-bypass and telemetry-enable switches are outside the maintained product policy.

See [Security](SECURITY.md), [Platform parity](PLATFORM-PARITY.md) and [Testing](TESTING.md).
