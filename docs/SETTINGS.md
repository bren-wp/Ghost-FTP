# Ghost FTP settings

Ghost FTP **1.1.1 Stable** treats settings as validated runtime policy, not decorative UI state. A persisted option is accepted only within the bounds enforced by `internal/config/settings.go`.

## Current persisted settings

- `language` — canonical local UI language; invalid stored values normalize to English.
- `appearance` — Windows desktop appearance: `dark` or `light`; **fresh, missing or invalid state resolves to `light` / Classic Light**. An explicitly stored `dark` choice remains preserved.
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

- `light` — **Classic Light**, the primary/fresh Ghost FTP appearance: a bright neutral two-pane workspace inspired by the clarity of traditional professional FTP clients while using Ghost FTP's own branding, iconography and palette.
- `dark` — the optional established Ghost FTP dark workspace. If the user explicitly selects and saves it, normalization preserves that preference.

The Windows selection is persisted locally and is applied on the next application start. This is intentional: native title bar, menu, combo, edit, list/header and owner-drawn control styling are selected before the complete window tree is created, preventing mixed-theme fragments and avoiding runtime repaint races.

Unknown or missing appearance state fails to Classic Light rather than Dark. This keeps fresh-install behavior aligned with the documented product default without overriding an intentional stored Dark preference.

### Linux

The native Linux desktop uses the **Classic Light** palette as the canonical 1.1 workspace. No extra Linux appearance toggle is exposed until complete runtime switching can be provided without introducing redraw/race complexity. This keeps the settings surface honest and avoids a control whose backend behavior would differ from its label.

Appearance changes do not load remote styles, fonts, images or theme services and do not create network traffic.

## Fresh connection protocol

Protocol selection itself is not persisted as a global preference. For a new/quick connection, both maintained desktop frontends start on **explicit FTPS, port 21**. This is a secure default, not a hidden downgrade policy:

- FTPS stays selected unless the user explicitly chooses another protocol or loads a profile;
- SFTP remains available with its SSH host-key policy;
- plain FTP remains available for legacy compatibility but must be selected explicitly;
- failed FTPS negotiation is never retried automatically as plain FTP;
- invalid/missing Windows protocol-selection state falls back to the first canonical FTPS entry.

Saved profiles continue to restore their explicitly saved protocol/port; the fresh default does not rewrite an existing profile.

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

Retry count/delay remain bounded to avoid accidental server hammering. Retries are also bound to the connection generation/identity so queued work cannot silently migrate to a later server session.

## Connection timeout

The configured timeout applies to connection establishment. It is validated identically for Windows and Linux. Transfer cancellation/progress uses its own lifecycle and is not fabricated from the connection-timeout value.

## Language

English is default/fallback and the canonical registry contains 24 languages. Unsupported locale values never create online translation traffic. Credential-persistence consent is also covered for all 24 maintained languages. See [Localization](LOCALIZATION.md).

## Credential persistence

Credential persistence is intentionally **not** a global on/off setting. It is a per-save privacy decision so there is no hidden background behavior or duplicate switch:

- entering a password or SFTP private-key passphrase and saving a Windows profile requires explicit consent before those new credentials are persisted;
- both the main Save Profile flow and Site Manager use the same localized consent model;
- declining consent still permits saving the non-secret connection profile and clears/removes stored credential fields for that profile;
- profile binding logic prevents old credentials from being silently carried across a changed server/account/private-key identity;
- Linux session-only protected-secret handles are not promoted into persistent profile data.

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
