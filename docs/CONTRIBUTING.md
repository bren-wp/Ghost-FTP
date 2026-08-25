# Contributing

Keep changes small, testable and consistent with the existing security model. English is the canonical language for source-facing user text, documentation and repository metadata; desktop runtime translations belong in `internal/i18n` and mobile translations should be added only as complete reviewed platform resource sets.

Do not add telemetry, advertising SDKs, hidden network destinations, insecure credential transport or certificate/host-key bypasses. Add regression tests for transfer, installer and security-sensitive behavior whenever practical.

Platform code must stay within its intended boundary:

- desktop/Go behavior belongs in `cmd/` and `internal/`,
- Android-native code belongs in `android/`,
- iOS-native Swift/Xcode code belongs in `ios/`.

Do not replace the native Android or iOS applications with WebView wrappers or hidden project-controlled backend services. New mobile transports must preserve platform certificate/host verification, credential lifetime, path safety and lifecycle cleanup before they are advertised as supported.

Before opening or merging a pull request, run the relevant repository audits and platform build gates. Changes that affect release packaging must keep `VERSION` as the single production version source and update the exact release allowlist, SHA-256 metadata contract and documentation together.
