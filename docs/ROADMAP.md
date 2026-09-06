# Ghost FTP roadmap

Ghost FTP **1.0.0 Stable** is the maintained production baseline. The roadmap after 1.0 prioritizes reliability, security, privacy, performance and Windows/Linux parity before adding broad new surface area.

## Completed for 1.0

The stable gate includes:

- native Windows and Linux desktop clients backed by one typed Engine;
- FTP, FTPS and SFTP workflows;
- SFTP password/key/passphrase authentication and host-key trust;
- local-only profiles with platform-local protected secret storage;
- validated connection timeout, retry, conflict and parallel-transfer settings;
- transfer generation binding, source snapshots and staged/rollback-oriented operations;
- privacy-safe connection diagnostics;
- truthful transfer progress, speed and ETA;
- native Windows Setup/Portable packaging and Linux DEB packaging;
- 24-language local catalog with English default/fallback;
- production race/vet/security/privacy/dependency/documentation audits;
- normal stable GitHub Release publication with `prerelease=false`;
- truthful Windows signing-state metadata with fail-closed verification when trusted production signing is configured;
- stable GitHub Packages/GHCR distribution-bundle publication and read-back.

## 1.0.x priorities

Patch releases should focus on compatible fixes:

1. crash/deadlock/race fixes found by production usage;
2. protocol interoperability edge cases that can be reproduced deterministically;
3. transfer-state and rollback correctness;
4. large-directory/list performance and redraw efficiency;
5. privacy-safe diagnostic quality;
6. Setup/update/uninstall rollback reliability;
7. localization corrections without changing protocol semantics;
8. release/package integrity and signing pipeline maintenance;
9. documentation accuracy.

## 1.1 priorities

Potential backward-compatible improvements include:

- richer transfer filtering/history while preserving bounded state;
- better per-site connection option ergonomics;
- additional keyboard and accessibility refinements;
- improved Linux visual parity without adding a heavy runtime framework;
- deeper deterministic FTP/FTPS/SFTP interoperability fixtures;
- optional export/import of non-secret profile metadata with explicit security boundaries;
- more operational diagnostics that remain local and credential-safe.

Features are not considered accepted merely because they are visually attractive. They must have a clear security/privacy model, tests, documentation and Windows/Linux behavior.

## Non-negotiable constraints

Future work must preserve:

- no application telemetry/advertising/fingerprinting;
- no mandatory Ghost FTP account;
- no silent secure-transport downgrade;
- SFTP host-key verification;
- protected saved-secret handling;
- local path containment and symlink/reparse safety;
- fail-closed transfer cleanup/commit behavior;
- exact source/version binding for public releases;
- truthful Windows signing states: configured trusted signatures verify fail-closed, otherwise Stable publication remains explicitly `WINDOWS_AUTHENTICODE=unsigned`;
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
- keep CI deterministic and offline for the Go dependency graph.

## Security/privacy direction

Security hardening should favor deterministic rejection and actionable errors over permissive fallback. Privacy improvements should reduce secret lifetime and diagnostic exposure rather than adding remote reporting.

Release security should improve publisher trust when a real certificate is available without weakening integrity verification or inventing trust when it is not.

## Definition of roadmap completion

A roadmap item is complete only after code, regression tests, security/privacy implications, active documentation and production CI/release gates agree on the behavior.
