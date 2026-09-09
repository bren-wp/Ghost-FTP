# Ghost FTP roadmap

Ghost FTP **0.0.1** starts the current public release line. The roadmap prioritizes correctness, security, privacy, reliability, Windows/Linux parity and measured performance before broad new surface area.

## Current 0.0.1 foundation

The current release gate includes:

- native Windows and Linux desktop clients backed by one typed Engine;
- FTP, FTPS and SFTP workflows;
- SFTP password/key/passphrase authentication and host-key trust;
- local-only profiles with platform-local protected secret storage;
- validated connection timeout, retry, conflict and parallel-transfer settings;
- transfer generation binding, source snapshots and staged/rollback-oriented operations;
- privacy-safe connection diagnostics;
- truthful transfer progress, speed and ETA;
- built-in Remote Edit with bounded text handling, revision/conflict protection and verified save/read-back;
- native Windows Setup/Portable packaging and Linux DEB/portable packaging;
- 24-language local catalog with English default/fallback;
- production race/vet/security/privacy/dependency/documentation audits;
- current GitHub Release publication with `prerelease=false`;
- truthful Windows signing-state metadata with fail-closed verification when trusted production signing is configured;
- current GitHub Packages/GHCR distribution-bundle publication and read-back;
- latest-only release/tag/package retention only after a successor is successfully published and verified.

## Next 0.0.x priorities

After 0.0.1 is fully published and verified, the next public release is **0.0.2**. Work should be driven by demonstrated defects or high-value functionality, including:

1. connection lifecycle, stale-session state and reconnect correctness;
2. cancel/retry/partial-transfer/interrupted-transfer behavior;
3. large-directory/list memory and UI responsiveness;
4. further Remote Edit shutdown/disconnect/conflict edge cases;
5. transfer atomicity, overwrite decisions, temp-file cleanup and application-shutdown behavior;
6. Windows/Linux functional parity for file operations, queue state, shortcuts, settings and error handling;
7. measured performance improvements such as avoiding duplicate stat/list work, stale callbacks and unnecessary redraws;
8. accessibility and keyboard refinements;
9. documentation and authentic real-application screenshots synchronized with exact release source;
10. carefully selected FileZilla/WinSCP/Cyberduck-class capabilities only where they add real user value without clutter.

## macOS direction

macOS remains a separate future scope. Documentation must not claim macOS support until the common engine has been audited for Darwin, platform contracts and CI/build gates exist, and a real native macOS frontend is built and tested. Any future macOS implementation must reuse the same FTP/FTPS/SFTP engine, security/privacy rules and Remote Edit logic rather than introducing a third protocol stack.

## Non-negotiable constraints

Future work must preserve:

- no application telemetry/advertising/fingerprinting;
- no mandatory Ghost FTP account;
- no silent secure-transport downgrade;
- SFTP host-key verification/pinning;
- FTPS certificate/hostname verification;
- protected saved-secret handling;
- local path containment and symlink/reparse safety;
- fail-closed transfer cleanup/commit behavior;
- trusted Linux transport and AskPass provenance;
- exact-object/ownership-aware Windows installer/uninstaller cleanup;
- exact source/version binding for public releases;
- truthful Windows signing states: configured trusted signatures verify fail-closed, otherwise current publication remains explicitly `WINDOWS_AUTHENTICODE=unsigned`;
- no generated/self-signed production identity represented as a trusted publisher;
- verified GitHub Release and GitHub Package publication;
- no unreviewed external Go dependencies.

## Performance direction

Optimization work should target measured hotspots:

- avoid full-workspace redraw when state is unchanged;
- keep UI work out of transfer/network critical paths;
- avoid repeated filesystem scans when a validated snapshot is sufficient;
- keep transfer progress publication bounded and truthful;
- reduce unnecessary allocations/copies in listing and transfer planning;
- prevent goroutine leaks, stale callbacks and incomplete timeout cleanup;
- keep CI deterministic and offline for the Go dependency graph.

## Security/privacy direction

Security hardening should favor deterministic rejection and actionable errors over permissive fallback. Privacy improvements should reduce secret lifetime and diagnostic exposure rather than adding remote reporting.

Release security should improve publisher trust when a real certificate is available without weakening integrity verification or inventing trust when it is not.

## Release direction

The public sequence is `0.0.1`, `0.0.2`, `0.0.3`, and so on. `0.0.0` is reserved. A current release identity is never rewritten in place. A newer version must pass exact-head and post-merge verification, publication and remote read-back before superseded public release/tag/package identities can be removed by retention.

## Definition of roadmap completion

A roadmap item is complete only after code, regression tests, security/privacy implications, active documentation and production CI/release gates agree on the behavior. Git commit history remains engineering provenance; the active public release catalog follows the verified latest-only lifecycle.
