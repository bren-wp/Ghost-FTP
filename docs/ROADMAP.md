# Ghost FTP roadmap

Ghost FTP **1.1.1 Stable** is the current maintenance candidate on top of the published 1.1.0 feature line. The roadmap prioritizes reliability, security, privacy, performance and Windows/Linux parity before broad new surface area.

## Completed stable foundation

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

## 1.1.1 maintenance priorities

The 1.1.1 patch line closes compatibility and quality gaps without inventing unrelated features:

1. make Classic Light the actual fresh/missing/invalid-state primary appearance while retaining an explicit persisted Dark choice;
2. use explicit FTPS/21 as the fresh quick-connect default on Windows and Linux while keeping plain FTP as an intentional legacy option;
3. deepen deterministic `remote.Manager.Connect()` and real loopback FTP lifecycle coverage;
4. preserve secure-to-plain downgrade blocking and SFTP host-key trust;
5. align credential-persistence consent between the main profile workflow and Windows Site Manager;
6. complete 24-language coverage for privacy-sensitive and native auxiliary Windows prompts;
7. keep large-directory/list and transfer UI redraw work event/state driven;
8. keep documentation and authentic real-application screenshots synchronized with the executable;
9. preserve Setup/Portable/DEB and release-package integrity.

## Next compatible priorities

Future patch/minor work should focus on measurable needs:

- crash/deadlock/race fixes found by reproducible production usage;
- protocol interoperability edge cases that can be reproduced deterministically;
- transfer-state and rollback correctness;
- large-directory/list performance and redraw efficiency;
- privacy-safe diagnostic quality;
- Setup/update/uninstall rollback reliability;
- accessibility and keyboard refinements;
- improved Linux visual parity without adding a heavy runtime framework;
- optional export/import of non-secret profile metadata with explicit security boundaries.

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

A roadmap item is complete only after code, regression tests, security/privacy implications, active documentation and production CI/release gates agree on the behavior. Published release tags and historical documentation remain immutable evidence rather than being rewritten to match a later roadmap state.
