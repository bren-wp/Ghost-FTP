# Contributing to Ghost FTP

Ghost FTP **1.1.7 Stable** is the current maintained release line. Ghost FTP is source-available proprietary software; a public repository does not automatically grant permission to redistribute modified builds. Contributions must respect the repository [`LICENSE`](../LICENSE).

Published Stable releases remain immutable historical identities and must not be rewritten by maintenance work.

## Contribution priorities

Preferred changes improve:

- crash/race/deadlock resistance;
- FTP/FTPS/SFTP correctness and interoperability;
- connection lifecycle and secure-default correctness;
- transfer staging/rollback/retry/cancel behavior;
- privacy-safe diagnostics and secret lifetime;
- local filesystem/path safety;
- Windows/Linux parity;
- native UI accessibility, localization, consistency and efficiency;
- Setup/Portable/DEB/tar.gz reliability;
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

## Connection/protocol changes

Connection changes must preserve explicit protocol identity. The fresh quick-connect policy is explicit FTPS/21 on Windows and Linux; plain FTP remains an explicit legacy compatibility choice and must never become an automatic fallback from failed FTPS.

Changes to connection establishment should exercise the shared `remote.Manager.Connect()` lifecycle and include failure-state coverage so invalid login, cancellation or failed secure handshake cannot expose an operational connected state.

## Security/privacy requirements

Never commit real passwords, private keys, key passphrases, protected profile payloads, signing private material, production server private data or CI secret values. Tests use synthetic credentials and isolated fixtures.

Credential persistence is opt-in. Main-profile and Site Manager flows must retain equivalent consent semantics. Secret-lifetime work must distinguish session-owned from borrowed profile-owned protected material.

## Dependency policy

The maintained Go module has no external module requirements. Production workflows intentionally use `GOPROXY=off` and `GOSUMDB=off`.

## UI changes

Windows changes must preserve DPI/resize behavior, keyboard focus, native control semantics, shared Light/Dark palette and the approved dual-pane hierarchy. Linux changes must preserve the same typed Engine behavior and avoid unnecessary continuous redraw.

Classic Light is the fresh/fallback appearance. An explicitly stored Dark selection remains supported. Auxiliary dialogs and security/privacy prompts must honor the maintained 24-language contract rather than silently falling back to hardcoded English.

Only the **About** surface may expose author/publisher identity. Main workspace, Settings, package metadata, support copy and release documentation use only the **Ghost FTP** product identity.

## Authentic UI evidence

When a Windows UI change affects documented appearance, version text, dialogs, Site Manager, Settings or About, the dedicated screenshot workflow must build and launch the real x64 Portable executable and regenerate verified screenshots. Mockups are not acceptable release evidence.

## Documentation changes

Update active documentation when user-visible behavior, package names, security/privacy boundaries or release behavior changes. Historical release records remain historical rather than being rewritten as current behavior. All local links must pass `scripts/audit_docs.py`.

## Release changes

Changes to release workflow, packaging or signing must preserve fail-closed behavior:

- truthful Windows signing state with `WINDOWS_AUTHENTICODE=signed|unsigned`;
- no generated/self-signed identity represented as a trusted production publisher;
- exact `main` commit binding;
- immutable version tags;
- canonical **12 platform artifacts / 15 public files** allow-list for 1.1.7;
- SHA-256 generation;
- GitHub Release read-back;
- Stable GitHub Package/GHCR distribution-bundle read-back;
- no release secret material inside artifacts.

A missing production code-signing certificate alone is not a release failure. A partially configured certificate or invalid signature when signing is configured is a release failure.

## Pull request expectations

A pull request should explain the defect/requirement, implementation approach, security/privacy implications, Windows/Linux impact, tests/audits run and documentation/release changes when applicable. A green compile alone is not sufficient for security-, transfer-, UI- or release-sensitive changes.
