# Roadmap

Ghost FTP is currently on **0.2.1 Beta**, within the `0.x` development line that began at 0.1.0. Product development remains focused on **Windows and Linux**, with reliability, security, protocol correctness, parity and professional usability taking priority over expanding the number of application platforms.

The first stable milestone is **1.0.0**. Reaching it requires the maintained product and release pipeline to be complete and stable as a whole; it is not triggered merely by finishing one feature.

## 0.1.0 baseline

The 0.1.0 Beta baseline preserves the substantial implementation and hardening work already present in the repository while resetting the active maturity/version line to a pre-stable sequence.

The baseline includes:

- Windows and Linux as the maintained desktop application targets;
- native Windows Win32 GUI with a professional dual-pane FTP workflow;
- one-click Sites access and a dedicated Site Manager;
- saved profiles plus quick connection flow;
- FTP, FTPS and SFTP support through the shared core;
- SFTP password, private-key and passphrase authentication;
- explicit SFTP host-key trust;
- local and remote file operations;
- transfer queue lifecycle controls;
- validated transfer/retry/conflict settings;
- English-first localization with 24 maintained languages;
- Windows Setup and Portable x64/x86 outputs using one canonical version;
- Linux amd64/arm64/i386 packages using the same canonical version;
- reproducible/checksummed release artifacts;
- fail-closed dependency, privacy, security, documentation and release audits;
- immutable historical repository provenance.

Android, iOS and macOS are not active application targets. Their historical source/releases remain available in Git history where applicable and are not deleted to create the new Beta baseline.

## Version milestones

### 0.1.x — baseline correctness

Priorities:

- keep the current Windows UI stable across resize, DPI and localization changes;
- verify Site Manager quick connections and saved-site connections across FTP/FTPS/SFTP;
- close remaining connection-state and control-state inconsistencies;
- improve transfer queue state correctness and failure recovery;
- keep Windows Setup and Portable build/version metadata identical;
- expand automated regressions for current behavior before larger feature additions.

### 0.2.x — workflow refinement

Priorities:

- continue improving file-pane ergonomics, keyboard workflow and multi-selection operations;
- improve connection diagnostics and actionable server/authentication errors;
- improve queue progress, speed and ETA presentation without telemetry or persistent tracking;
- improve profile-management discoverability and validation;
- continue localization cleanup of remaining hard-coded frontend text.

### 0.3.x and later Beta milestones

Priorities are driven by verified gaps discovered through real FTP/FTPS/SFTP use, CI, authentic Windows UI captures and release testing. Minor Beta bumps should represent meaningful tested capability or quality milestones rather than arbitrary numbering.

### 0.9.x — release-candidate stabilization

The final Beta phase should emphasize:

- regression closure rather than feature expansion;
- install/upgrade/uninstall and Portable smoke testing;
- protocol interoperability with representative shared-hosting and SSH servers;
- localization and accessibility review;
- documentation completeness;
- release artifact and checksum verification;
- signing readiness and explicit unsigned-state communication where required;
- zero known release-blocking defects.

### 1.0.0 — first stable release

The project advances to 1.0.0 only after the complete stable gate in [VERSIONING.md](VERSIONING.md) is satisfied.

At minimum:

- Windows Setup and Portable are production-ready and carry version 1.0.0 together;
- Windows native UI and Site Manager are stable for normal desktop use;
- Linux maintained functionality satisfies the shared-core contract;
- FTP/FTPS/SFTP authentication and transfers are reliable under expected conditions;
- transfer conflict/retry/cancel/recovery behavior is covered by regression tests;
- SFTP host-key and credential protections remain fail-closed;
- all quality/security/privacy/dependency/documentation gates pass;
- release publication and remote readback pass;
- current documentation accurately describes the shipped product.

## Near-term priorities

### Transfer correctness

- continue expanding regression tests for interrupted upload/download, overwrite rollback and ambiguous server responses;
- strengthen retry classification so transient failures retry while authentication/validation failures fail immediately;
- improve progress/speed/ETA presentation without introducing persistent tracking/logging;
- keep directory/tree transfer planning bounded and symlink-safe;
- improve queue ergonomics and multi-selection operations on Windows and equivalent command operations on Linux.

### FTP/FTPS interoperability

- expand passive/data-channel error diagnostics for common shared-hosting servers;
- improve server capability detection without weakening validation;
- preserve strict FTPS certificate validation and avoid insecure compatibility switches;
- continue testing Unicode/path/listing edge cases.

### SFTP interoperability

- improve actionable errors for key format, passphrase, host-key and authentication failures;
- expand OpenSSH process smoke tests across supported Linux environments;
- preserve explicit host-key trust and endpoint binding;
- avoid inheriting ambient proxy/jump/agent forwarding state.

### Windows experience

- continue refining spacing, typography, status hierarchy and transfer queue readability;
- continue improving Site Manager without adding decorative options that lack backend behavior;
- improve keyboard accessibility/focus behavior;
- keep high-DPI behavior stable across common scale factors;
- keep Setup/Portable packaging consistent and localized;
- pursue code signing when an appropriate signing identity/certificate is available, without representing unsigned artifacts as signed.

### Linux parity

- keep every new shared-core connection/transfer option accessible from Linux where appropriate;
- improve terminal discoverability/help and structured status output;
- expose additional local/profile workflows where they can reuse shared engine methods;
- continue refining the existing dependency-free native Linux graphical frontend without forking protocol logic or weakening the terminal fallback.

### Localization

- continue improving real translation coverage across all 24 locales;
- eliminate remaining hard-coded frontend text by moving it into the canonical catalog;
- preserve English fallback for incomplete future translations;
- validate placeholders/format verbs and live language switching in CI.

### Security and privacy

- preserve zero application telemetry/analytics/advertising;
- preserve fixed-runtime-URL and tracking-vendor audit gates;
- continue credential-lifetime reduction and secret-zeroing work;
- expand tests around symlink/reparse-point races and malicious local state;
- keep SFTP AskPass free of disk secret artifacts;
- keep release/tag provenance immutable.

### Dependency strategy

The desktop/core Go module should remain standard-library-only unless a reviewed change demonstrates a clear security or reliability benefit.

Current OS-provided transport tools (`curl`, `ssh`, `sftp`) are explicit prerequisites. Replacing them with embedded protocol stacks is not a cosmetic dependency change; it would require protocol-level security review, compatibility testing, license/provenance review and a migration plan.

## Out of scope for the active Beta line

The following are intentionally not active application targets:

- Android;
- iOS;
- macOS.

Reintroducing a retired application platform would require an explicit product decision, a support/CI/release contract and compatibility analysis rather than quietly adding a directory back to the tree.

## Release quality principle

A roadmap item is not considered complete because UI code exists. It must have the relevant combination of:

- shared-core implementation;
- Windows/Linux exposure where applicable;
- regression/security tests;
- localization coverage;
- documentation;
- release notes;
- CI/package verification.

Beta version numbers should advance only after that evidence exists for the milestone being claimed.
