# Contributing to Ghost FTP

Ghost FTP **0.0.5** is the current maintained public release line. Ghost FTP is source-available proprietary software; a public repository does not automatically grant permission to redistribute modified builds. Contributions must respect the repository [`LICENSE`](../LICENSE).

A published current release identity is never rewritten in place. A successor uses a new semantic version; only after that successor is successfully published and verified may the latest-only retention workflow remove superseded public Ghost FTP release/tag identities. Source commit history remains intact.

## Contribution priorities

Preferred changes improve:

- crash/race/deadlock resistance;
- FTP/FTPS/SFTP correctness and interoperability;
- connection lifecycle and secure-default correctness;
- transfer staging/rollback/retry/cancel behavior;
- privacy-safe diagnostics and secret lifetime;
- local filesystem/path safety;
- Windows/Linux production parity and Android/macOS development-source quality;
- native UI accessibility, localization, consistency and efficiency;
- Setup/Portable/DEB/RPM/tar.gz reliability;
- release verification and documentation accuracy.

## Before changing code

Identify the correct layer first. Protocol/transfer behavior belongs in the shared core where possible; platform frontends should not fork security or transfer semantics just to expose a UI control.

Do not add a new dependency, remote service, telemetry path or signing mechanism without explicit review of security, privacy, provenance and release impact. Do not implement a UI-only switch: a visible option must have one typed runtime owner, safe migration/default behavior and regression coverage.

## Required local checks

```text
go telemetry off
gofmt
go test -race ./...
go vet ./...
python scripts/audit_repository.py
python scripts/audit_platform_contract.py
python scripts/audit_desktop_surface.py
python scripts/audit_dependencies.py
python scripts/audit_version.py
python scripts/audit_localization.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_docs.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

Run affected native platform gates as well. A change that touches Android or macOS source/build behavior must pass the corresponding maintained native workflow; development build success must not be represented as public signing/notarization evidence.

## Connection/protocol changes

Connection changes must preserve explicit protocol identity. The fresh quick-connect policy is explicit FTPS/21 on maintained desktop paths; plain FTP remains an explicit legacy compatibility choice and must never become an automatic fallback from failed FTPS.

Changes to connection establishment should exercise the shared connection lifecycle and include failure-state coverage so invalid login, cancellation or failed secure handshake cannot expose an operational connected state. Android FTP/FTPS changes must also preserve Activity ownership and authentication-error redaction.

## Security/privacy requirements

Never commit real passwords, private keys, key passphrases, protected profile payloads, signing private material, production server private data or CI secret values. Tests use synthetic credentials and isolated fixtures.

Credential persistence is opt-in. Main-profile and Site Manager flows must retain equivalent consent semantics. Secret-lifetime work must distinguish session-owned from borrowed profile-owned protected material.

## Dependency policy

The maintained Go module has no external module requirements. Production workflows intentionally use `GOPROXY=off` and `GOSUMDB=off`.

## UI changes

Windows changes must preserve DPI/resize behavior, keyboard focus, native control semantics, shared Light/Dark palette and the approved dual-pane hierarchy. Linux changes must preserve the same typed Engine behavior and avoid unnecessary continuous redraw. macOS changes must preserve native AppKit lifecycle/accessibility behavior while using the shared engine/security contracts rather than creating a parallel protocol stack.

Classic Light is the fresh/fallback appearance. An explicitly stored Dark selection remains supported. Auxiliary dialogs and security/privacy prompts must honor the maintained localization contract rather than silently falling back to hardcoded English where a catalog-backed string exists.

Only the **About** surface may expose author/publisher identity. Main workspace, Settings, package metadata, support copy and release documentation use only the **Ghost FTP** product identity.

## Authentic UI evidence

When a maintained UI change affects documented appearance or behavior, use real runtime evidence from the applicable native workflow. Mockups, generated approximations and screenshots from a different source revision are not release evidence.

The current immutable cross-platform UI evidence bundle covers maintained Windows/Linux/Android runtime surfaces. macOS has a separate native development build/validation path; do not reinterpret that as Developer ID/notarized public-distribution evidence.

## Documentation changes

Update active documentation when user-visible behavior, package names, platform status, security/privacy boundaries or release behavior changes. Commit history remains engineering provenance, while active public release documentation follows the current latest-only release policy. All local links and release-policy contracts must pass `scripts/audit_docs.py`.

## Release changes

Changes to release workflow, packaging or signing must preserve fail-closed behavior:

- official public Windows Setup and Portable require trusted Authenticode and `WINDOWS_AUTHENTICODE=signed`;
- local/development or ordinary CI Windows outputs may be unsigned but must not be represented as official public release artifacts;
- no generated/self-signed identity represented as a trusted production publisher;
- exact `main` commit binding;
- no in-place rewrite of the current version tag or release assets;
- canonical **14 platform artifacts / 17 public files** Windows/Linux allow-list for 0.0.5;
- SHA-256 generation;
- GitHub Release read-back with `prerelease=false`;
- current GitHub Package/GHCR distribution-bundle read-back;
- latest-only cleanup only after the successor release has been fully verified;
- no release secret material inside artifacts.

For the canonical `Publish Ghost FTP` workflow, a missing production Authenticode identity **is a release failure**. A malformed/partial identity, signing failure or invalid signature is also a release failure. Ordinary CI remains intentionally usable without those protected production credentials.

Android and macOS remain active source/development surfaces outside the current 17-file public release allow-list. Adding either to public distribution requires an explicit production publication contract and evidence rather than a documentation-only change.

## Pull request expectations

A pull request should explain the defect/requirement, implementation approach, security/privacy implications, affected platforms, tests/audits run and documentation/release changes when applicable. A green compile alone is not sufficient for security-, transfer-, UI- or release-sensitive changes.
