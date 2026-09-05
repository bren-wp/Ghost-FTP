# Settings and option semantics

Ghost FTP settings are part of transfer safety. An option is not added to the UI unless the runtime engine can enforce it consistently and tests cover its effect.

## Current persisted settings

The desktop settings model currently stores:

- `language` — canonical UI language. Unknown/invalid locale codes normalize safely to English when loading old state; new saves reject unsupported languages.
- `parallelism` — number of concurrent transfers, range 1–8, safe default 2.
- `connectionTimeoutSeconds` — connection timeout, range 5–60 seconds, safe default 15.
- `autoRetryCount` — automatic retry count, range 0–3, safe default 0.
- `retryDelaySeconds` — delay between automatic retries, range 1–30 seconds, safe default 3.
- `backupBeforeOverwrite` — retain a recovery backup of an existing destination during safe overwrite behavior.
- `skipExisting` — do not overwrite a destination item that already exists.
- `confirmDelete` — require an explicit confirmation before user-initiated delete operations in the desktop UI.

Corrupt or unavailable settings storage does not grant a more permissive transfer policy. `DefaultSettings()` is conservative: overwrite recovery and delete confirmation remain enabled.

## Validation and migration

`internal/config/settings.go` owns the canonical bounds and defaults. UI code must not invent different limits. Old settings files are normalized when loaded, and values outside the supported range return to safe defaults instead of becoming unbounded runtime parameters.

New saves are validated before the file is written. A rejected save does not mutate the in-memory effective settings.

## Existing-file behavior

There are currently two persisted booleans involved in destination conflicts:

1. `skipExisting=true` means the destination is not overwritten.
2. When existing files are not skipped, `backupBeforeOverwrite` controls whether the safe temporary recovery backup is kept according to the transfer implementation.

This model is backward compatible but is not the final 1.1.x settings UX. A future settings UI should present one explicit conflict policy instead of making users reason about combinations of two booleans. Migration must preserve the meaning of existing settings:

- `skipExisting=true` → **Skip existing**.
- `skipExisting=false` + `backupBeforeOverwrite=true` → **Replace safely and keep recovery backup**.
- `skipExisting=false` + `backupBeforeOverwrite=false` → **Replace safely and remove temporary recovery backup after commit**.

The storage migration must be introduced only together with engine/UI tests; the current booleans remain the canonical persisted contract until that migration lands.

## Retry behavior

Retries are intended for transient connection/transport failures, not validation or trust failures. Host-key mismatch, invalid path, invalid credentials, unsafe local/remote target state and explicit user cancellation must not be converted into blind automatic retries.

Retry delay is bounded so the application cannot accidentally create an unbounded hammering loop against a hosting server.

## Connection timeout

The configured timeout is applied to connection establishment. Long-running file transfers use their own cancellation/progress boundaries and must not be truncated merely because the initial connection timeout is small.

## Language

English is the primary/default language. The canonical language registry is documented in [Localization](LOCALIZATION.md). Regional codes normalize to supported canonical codes where appropriate.

## Planned option-quality rules for 1.1.x

New advanced settings must meet all of these conditions before release:

- One clearly defined runtime owner for the option.
- Safe default that does not weaken TLS, SFTP host-key verification, path validation or overwrite recovery.
- Bounded numeric ranges where applicable.
- Backward-compatible settings migration.
- User-visible label/help text in the localization system.
- Unit/regression tests proving both enabled and disabled behavior.
- No option may disable certificate validation, accept any SFTP host key, enable telemetry, bypass remote/local path guards or persist plaintext secrets.

Candidates that satisfy those rules include a consolidated conflict policy, configurable transfer verification policy where the protocol can provide reliable metadata, and explicit UI visibility preferences. Security-bypass toggles are intentionally out of scope.
