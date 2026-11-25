# Native credential vault migration — design and acceptance

The current Linux path uses libsecret's secret-tool with the existing attributes application=ghostftp, service, account. Legacy profile compatibility remains in credentials.rs; strict recovery calls must never fall back to volatile storage.

## Candidate architecture

Evaluate a maintained direct Secret Service binding over D-Bus or libsecret, with encrypted sessions and bounded cancellation, rather than a persistent shell helper. The lockfile already contains zbus transitively; that alone does not establish the required API, feature set or compatibility. Do not switch the production backend or add a dependency until the following acceptance is demonstrated.

- Read credentials created by the existing secret-tool path, and verify the inverse on a disposable persistent keyring.
- Preserve exact attribute identity, UTF-8 bytes, collection semantics and existing duplicate/error behavior; locked or unavailable collections must not be treated as missing.
- Distinguish missing item, locked collection, transport error, timeout, prompt cancellation and service restart. Never downgrade to legacy or session storage on strict reads.
- Preserve write and delete read-back, HMAC/digest checks, interprocess locking and the existing on-disk format. Do not cache credentials or keys across operations.
- Test GNOME Keyring and supported KWallet configurations, persistent survival across service/OS restart, existing profile and OAuth compatibility, revocation and no credential disclosure in logs.
- Compare exact-source native ABBA runs and occupancy bands, publish raw observations, and keep the subprocess implementation until equivalence and failure semantics are verified.

Status: evaluation only; no direct backend or production migration is implemented in this change.
