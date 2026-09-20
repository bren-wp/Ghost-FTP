# Ghost FTP Recommended Roadmap

This file separates **recommended next work** from features already completed. Nothing below should be represented as shipped until implementation and QA evidence exist.

## Priority 1 — FINAL acceptance

- Complete Windows 10/11 frameless-titlebar screenshots.
- Run pixel-delta review against Main, Site Manager, New Connection, Preferences, Transfer Center, File Properties and About references.
- Complete required responsive viewport captures.
- Run clean install → launch → upgrade → uninstall on Windows.
- Execute real FTP, FTPS and SFTP end-to-end matrices, including invalid credentials, certificate/host-key mismatch, reconnect and interrupted-transfer cases.

## Priority 2 — transfer confidence

- Add deterministic integration test servers for FTP/FTPS/SFTP in CI where safe.
- Add transfer-resume integrity verification using known hashes.
- Add conflict-policy regression tests.
- Add retry/backoff and reconnect chaos tests.
- Add large-file and many-small-file performance baselines.

## Priority 3 — UX polish

- Keyboard-only audit for every modal and full-window surface.
- Focus-ring and tab-order audit.
- 125%, 150% and 200% Windows display-scaling review.
- Long-translation clipping audit across all 14 languages.
- High-contrast and reduced-motion review.
- Empty/loading/error state consistency pass.

## Priority 4 — operational features

- Saved synchronization jobs.
- Per-profile transfer policies.
- Deployment groups for repeated uploads to several servers.
- Exportable support bundle with automatic credential redaction.
- Optional encrypted portable profile vault.
- More detailed connection diagnostics.

## Priority 5 — release engineering

- Add signed Windows binaries when production signing material is available.
- Add Linux package-signing policy.
- Publish SBOM and dependency inventory.
- Add automated release-note generation from labelled changes.
- Add reproducibility notes and artifact provenance.
