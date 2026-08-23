# Contributing

Keep changes small, testable and consistent with the existing security model. English is the canonical language for source-facing user text, documentation and repository metadata; runtime translations belong in `internal/i18n`.

Do not add telemetry, advertising SDKs, hidden network destinations, insecure credential transport or certificate/host-key bypasses. Add regression tests for transfer, installer and security-sensitive behavior whenever practical.

Before opening a pull request, run formatting, unit tests, `go vet` and the relevant repository audit scripts.
