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

## Isolated direct API prototype

`tools/native-vault/direct-secret-service.c` exercises a reusable libsecret service session, exact existing attribute matching, persistent default-collection selection, strict unique-item lookup without unlocking prompts, verified writes/deletes and compatibility with secret-tool. The workflow uses disposable GNOME Keyring sessions, tests a locked collection and attempts a fresh service session with the same isolated persistent data. It compares repeated lookup latency on the same runner, using ABBA ordering and raw samples. The prototype is not linked into the production Rust application; the outer test-process deadline is not a production cancellation mechanism. KWallet and in-process reconnect/cancellation remain explicit acceptance gates.

## Rust integration candidate

The `native-vault-candidate` Cargo feature builds a native bridge for Linux test binaries only. Explicit disposable native recovery fixtures may select it; normal release builds retain secret-tool. A thread-owned service session uses a joined GCancellable deadline worker and checks the current D-Bus service owner before and after each operation. Uncertain operations invalidate the session and are never automatically replayed. Existing attributes, read-back, and recovery locks are preserved. Cancellation is cooperative, not a hard OS process kill; live blocked-provider, KWallet and restart-during-operation acceptance remain required.

## Failure and acceptance gate

The candidate bridge is built unchanged into a standalone failure harness. In a disposable GNOME Keyring D-Bus session, a verified fixture daemon is stopped, a 200 ms lookup must return failure within 2 seconds, then the daemon is resumed and the credential reread. A second blocked lookup is interrupted by terminating that same daemon, followed by a new provider process and verified readback. A passing read-only failure test does not establish safe cancellation of in-flight mutation. `tools/native-vault/kwallet-acceptance.sh` is an explicit manual provider-identity-checked CRUD acceptance harness for an unlocked dedicated KWallet test account (KF 5.97+ or compatible KF6 Secret Service); it rejects GNOME fallback. KWallet lock/restart and mutation uncertainty remain release gates. The full recovery benchmark now requests 100 observations per backend and file size using ABBA/BAAB on the same runner. Default historical benchmark remains three samples.

## Transactional mutation uncertainty

The bridge distinguishes a dispatched mutation with uncertain outcome (-2) from a pre-dispatch error. The Rust adapter invalidates its session on uncertain outcomes or failed readback; it never retries automatically. An isolated callback gate after a successful native write/delete but before final owner validation allows the test harness to terminate the provider at the acknowledgement boundary and verify durable state after restart without replay. This is not an atomic transaction across the credential store and SQLite. Mid-RPC mutation fault coverage and durable reconciliation across application restart remain mandatory before production activation.
