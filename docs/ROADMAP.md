# Roadmap

Ghost FTP starts a new product line at **1.0.0**. The roadmap is capability- and quality-based; historical ByFTP release numbers do not define future Ghost FTP sequencing.

## 1.0.x priorities

The initial 1.0.x series focuses on stability, compatibility and distribution quality rather than rapid feature expansion:

- remove remaining user-visible legacy branding while preserving only migration-sensitive internal identifiers;
- keep Windows x64/x86 installation and upgrade behavior rollback-safe;
- broaden Linux package compatibility across amd64, arm64 and i386 without forking the shared desktop core;
- improve macOS packaging/signing readiness while preserving the Universal build;
- harden Android lifecycle, file-picker and secure credential behavior;
- expand iOS transport capability only when certificate/host verification and credential handling meet the same fail-closed security model;
- continue web/PWA shared-hosting compatibility, storage durability and strict no-cache boundaries for sensitive responses;
- reduce duplicate build/audit code and keep one canonical production release pipeline.

Patch versions advance sequentially: `1.0.0`, `1.0.1`, `1.0.2`, and so on.

## Protocol and transfer work

Future protocol work must not weaken existing trust boundaries. New or expanded transport features require:

- platform-appropriate TLS or host-key verification;
- strict path and connection-input validation;
- deterministic temporary-file cleanup;
- bounded transfer behavior;
- explicit failure semantics;
- regression coverage on every affected platform.

True mid-file cancellation remains a future capability only if the active protocol can abort safely and Ghost FTP can deterministically handle partial local/remote files. A UI button must not claim that a remote write was safely cancelled when the protocol state cannot prove it.

## Mobile distribution

Android CI currently produces an installable APK. Production Play distribution requires an externally managed production signing key and an explicit signing verification gate; repository automation must never substitute a debug or fabricated identity and label it production-signed.

iOS CI produces a real unsigned arm64 IPA. Device/TestFlight/App Store distribution requires a legitimate Apple signing identity and provisioning profile managed outside the repository. New iOS protocol support is advertised only after its trust, path, credential and lifecycle behavior is covered by native tests and CI.

## Desktop distribution

Windows remains a primary desktop target. The installer and upgrade transaction will continue to receive additional regression coverage for payload integrity, path redirection, rollback and legacy-install cleanup.

Linux packaging will remain based on the shared Go core and canonical `linux/BUILD.sh`. macOS packaging will remain based on the shared core and canonical `macos/BUILD.sh`; code signing/notarization can be added when valid publisher credentials are available.

## Web/PWA

The shared-hosting application will continue prioritizing:

- safe deployment on ordinary PHP hosting;
- no indexing of authenticated/private application surfaces;
- strict session/CSRF/rate-limit behavior;
- encrypted profile storage;
- fail-closed host and remote-path validation;
- bounded archive/upload/download operations;
- deterministic migration of PWA caches and application state.

Ghost FTP will not add background port scanning, hidden diagnostic destinations, certificate/host-key bypasses or secret-bearing analytics to obtain a richer status screen.

## Release engineering

Every production release must keep root `VERSION` as the canonical version source and must pass all applicable platform gates. The public release contract remains intentionally small: eight platform packages plus checksum, release notes and build metadata.

A feature is complete only when its failure modes are understood, regression coverage exists where practical and every affected build/security gate remains green.
