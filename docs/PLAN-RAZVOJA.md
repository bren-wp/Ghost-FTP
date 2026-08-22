# Development roadmap

The roadmap prioritizes correctness and compatibility over feature count.

## Near-term priorities

1. Maintain complete multilingual Windows/terminal workflows while keeping English canonical.
2. Continue real protocol integration coverage, including opt-in provider-specific checks when private CI credentials are available.
3. Improve high-volume directory handling and transfer-queue ergonomics without weakening path or commit safety.
4. Continue responsive Windows UX work for DPI scaling, long translations and keyboard accessibility.
5. Expand protocol diagnostics so users can distinguish authentication, data-channel, TLS, permission and provider-limit failures.

## Security priorities

Preserve SFTP fingerprint pinning, identity-bound saved credentials, no-follow filesystem rules, staged transfer revalidation and fail-closed cleanup. Continue process-lifecycle and installer/uninstaller transaction audits.

## Packaging priorities

Keep exact release asset contracts, reproducible build metadata and SHA-256 verification. Organization signing/notarization may be enabled only with real credentials and must not be simulated.

## Non-goals

ByFTP will not bypass hosting-provider permissions, disable certificate verification to make FTPS appear to work, silently trust changed SFTP keys, or add telemetry to simplify debugging.