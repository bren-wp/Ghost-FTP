# Roadmap

Ghost FTP 2.x focuses product development on **Windows and Linux**. The roadmap prioritizes reliability, security, protocol correctness, parity and premium usability over expanding the number of application platforms.

## 2.0 consolidation goals

The 2.0 line establishes the new baseline:

- Windows and Linux are the only active desktop application targets;
- Android, iOS and macOS application source/packaging are removed from active development;
- Windows remains the native graphical reference frontend;
- Linux uses the same shared core and closes authentication/queue/settings parity gaps;
- English remains the canonical/default language with 24 supported runtime languages;
- release CI is reduced to shared quality + Windows + Linux production gates;
- release artifacts are reproducible and checksummed;
- dependency provenance and no-tracking rules are fail-closed;
- documentation describes each release and preserves immutable 1.x history.

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
- improve profile management and connection diagnostics;
- improve keyboard accessibility/focus behavior;
- keep high-DPI behavior stable across common scale factors;
- keep Setup/portable packaging consistent and localized;
- pursue code signing when an appropriate signing identity/certificate is available, without pretending unsigned artifacts are signed.

### Linux parity

- keep every new core connection/transfer option accessible from Linux;
- improve terminal discoverability/help and structured status output;
- expose additional local/profile workflows where they can reuse shared engine methods;
- consider a native/lightweight Linux graphical frontend only if it meets dependency, security, reproducibility and maintenance requirements without forking protocol logic.

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

The desktop/core Go module should remain standard-library-only unless a reviewed change demonstrates clear security/reliability benefit.

Current OS-provided transport tools (`curl`, `ssh`, `sftp`) are explicit prerequisites. Replacing them with embedded protocol stacks is not a cosmetic dependency change; it would require protocol-level security review, compatibility testing, license/provenance review and a migration plan.

## Web companion

The existing shared-hosting Web companion remains a separate source surface. Work may continue on its PHP/PWA security, localization and hosting compatibility, but it is not part of the Windows/Linux desktop artifact matrix.

## Out of scope for the current 2.x line

The following are intentionally not active application targets:

- Android;
- iOS;
- macOS.

Historical sources/releases remain in Git provenance. Reintroducing any retired application platform would require an explicit future product decision, a new support/CI/release contract and compatibility analysis rather than quietly adding a directory back to the tree.

## Release quality principle

A roadmap item is not considered complete because UI code exists. It must have the relevant combination of:

- shared-core implementation;
- Windows/Linux exposure where applicable;
- regression/security tests;
- localization coverage;
- documentation;
- release notes;
- CI/package verification.
