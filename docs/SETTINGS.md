# Settings and option semantics

Ghost FTP settings are part of transfer safety. An option is not added to the UI unless the runtime engine can enforce it consistently and tests cover its effect.

## Current persisted settings

The desktop settings model stores:

- `language` — canonical UI language. Unknown/invalid locale codes normalize safely to English when loading old state; new saves reject unsupported languages.
- `parallelism` — number of concurrent transfers, range 1–8, safe default 2.
- `connectionTimeoutSeconds` — connection timeout, range 5–60 seconds, safe default 15.
- `autoRetryCount` — automatic retry count, range 0–3, safe default 0.
- `retryDelaySeconds` — delay between automatic retries, range 1–30 seconds, safe default 3.
- `conflictPolicy` — canonical behavior when the transfer destination already exists.
- `backupBeforeOverwrite` and `skipExisting` — compatibility mirrors retained for older state/readers; they are synchronized from `conflictPolicy` and are no longer independent user choices.
- `confirmDelete` — require an explicit confirmation before user-initiated delete operations in the desktop UI.

Corrupt or unavailable settings storage does not grant a more permissive transfer policy. `DefaultSettings()` is conservative: replace-with-recovery and delete confirmation remain enabled.

## Validation and migration

`internal/config/settings.go` owns the canonical bounds and defaults. UI code must not invent different limits. Old settings files are normalized when loaded, and values outside the supported range return to safe defaults instead of becoming unbounded runtime parameters.

New saves are validated before the file is written. A rejected save does not mutate the in-memory effective settings.

### Conflict policy migration

Ghost FTP 1.1.0 development consolidates the former pair of overwrite booleans into one policy:

- `skip` — **Skip existing**. Existing destinations are left untouched. Compatibility values become `skipExisting=true` and `backupBeforeOverwrite=false`.
- `replace` — **Replace safely**. Existing destinations may be replaced through the transfer implementation's safe commit path without retaining the recovery backup after success. Compatibility values become `skipExisting=false` and `backupBeforeOverwrite=false`.
- `replace_backup` — **Replace safely + keep recovery backup**. This is the conservative default. Compatibility values become `skipExisting=false` and `backupBeforeOverwrite=true`.

Legacy files are migrated deterministically:

- `skipExisting=true` maps to `skip`, even if an old file also had the irrelevant backup flag enabled.
- `skipExisting=false` + `backupBeforeOverwrite=true` maps to `replace_backup`.
- both false maps to `replace`.

An unknown/corrupt policy read from persisted state fails closed to `replace_backup`. An unknown policy supplied by a new settings save is rejected instead of silently accepted.

The Windows settings flow presents these three states as one native selector. Users no longer have to reason about contradictory combinations of two Yes/No dialogs.

## Retry behavior

Retries are intended for transient connection/transport failures, not validation or trust failures. Host-key mismatch, invalid path, invalid credentials, unsafe local/remote target state and explicit user cancellation must not be converted into blind automatic retries.

Retry delay is bounded so the application cannot accidentally create an unbounded hammering loop against a hosting server.

## Connection timeout

The configured timeout is applied to connection establishment. Long-running file transfers use their own cancellation/progress boundaries and must not be truncated merely because the initial connection timeout is small.

## Language

English is the primary/default language. The canonical language registry is documented in [Localization](LOCALIZATION.md). Regional codes normalize to supported canonical codes where appropriate.

## Option-quality rules for 1.1.x

New advanced settings must meet all of these conditions before release:

- One clearly defined runtime owner for the option.
- Safe default that does not weaken TLS, SFTP host-key verification, path validation or overwrite recovery.
- Bounded numeric ranges where applicable.
- Backward-compatible settings migration.
- User-visible label/help text in the localization system.
- Unit/regression tests proving both enabled and disabled behavior.
- No option may disable certificate validation, accept any SFTP host key, enable telemetry, bypass remote/local path guards or persist plaintext secrets.

Candidates that satisfy those rules include configurable transfer verification policy where the protocol can provide reliable metadata and explicit UI visibility preferences. Security-bypass toggles are intentionally out of scope.
