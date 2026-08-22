# Contribution policy

ByFTP is proprietary/source-available software governed by the repository `LICENSE`. Access to source code or the ability to fork the GitHub repository does not grant rights beyond the license and applicable GitHub platform rights.

## Before changing code

Keep English as the canonical source/documentation language. User-facing runtime text must be added through `internal/i18n`; do not introduce new hard-coded language-specific UI strings into production code.

Preserve the privacy model: no telemetry, analytics, advertising SDKs, hidden network destinations or external crash reporting. Do not place credentials in command-line arguments, logs, fixtures or screenshots.

## Engineering expectations

Keep modules narrow. Protocol, transfer, config/security and UI responsibilities should not be merged for convenience. Avoid duplicate helpers and remove dead code only when compilation/tests demonstrate that it is unused.

Any connection, transfer, path, credential, installer, localization or process-lifecycle change needs regression coverage appropriate to the affected invariant.

Before review, run the project audits and Go tests. Windows-specific code must pass native Windows x64/x86 production build verification.

## Pull requests

Use the repository PR template. Describe user-visible behavior, security/privacy impact and tests. Do not request merge while any required CI job is failing. Release tags are immutable; fixes after a published tag require a new version.