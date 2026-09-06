# Contributing to Ghost FTP

Ghost FTP **1.1.1 Stable** is the current maintenance candidate in the source tree. Ghost FTP is source-available proprietary software maintained by **BRENDIGO LTD**. A public repository does not automatically grant permission to redistribute modified builds; contributions must also respect the repository [`LICENSE`](../LICENSE).

Published 1.1.0 and 1.0.0 releases remain historical release identities and must not be rewritten by maintenance work.

## Contribution priorities

Preferred changes improve:

- crash/race/deadlock resistance;
- FTP/FTPS/SFTP correctness and interoperability;
- connection lifecycle and secure-default correctness;
- transfer staging/rollback/retry/cancel behavior;
- privacy-safe diagnostics and secret lifetime;
- local filesystem/path safety;
- Windows/Linux parity;
- native UI accessibility, localization and efficiency;
- Setup/Portable/DEB reliability;
- release verification and documentation accuracy.

## Before changing code

Identify the correct layer first. Protocol/transfer behavior belongs in the shared core where possible; platform frontends should not fork security or transfer semantics just to expose a UI control.

Do not add a new dependency, remote service, telemetry path or signing mechanism without explicit review of its security, privacy, provenance and release impact.

Do not implement a UI-only switch. A visible option must have one typed runtime owner, safe migration/default behavior and regression coverage.

## Required local checks

For Go changes:

```text
go telemetry off
gofmt
go test -race ./...
go vet ./...
```

Run relevant repository audits after changes:

```text
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

Changes to connection establishment should exercise the shared `remote.Manager.Connect()` lifecycle and include failure-state coverage so an invalid login, cancelled attempt or failed secure handshake cannot expose an operational connected state.

## Security/privacy requirements

Do not commit or paste:

- real FTP/SFTP passwords;
- private keys or key passphrases;
- protected saved-profile payloads;
- code-signing private keys/certificates with private material;
- production server private data;
- CI secret values.

Tests use synthetic credentials and isolated fixtures.

Credential persistence is opt-in. Main-profile and Site Manager flows must retain equivalent consent semantics. Secret-lifetime work must distinguish session-owned from borrowed profile-owned protected material so cleanup does not create reconnect regressions.

## Dependency policy

The maintained Go module has no external module requirements. A proposal to add one must demonstrate why the standard library/system-runtime approach is insufficient and include security, license, update and reproducibility analysis.

Production workflows intentionally use `GOPROXY=off` and `GOSUMDB=off`.

## UI changes

Windows changes should preserve DPI/resize behavior, keyboard focus, native control semantics and the approved dual-pane hierarchy. Linux changes should preserve the same typed Engine behavior and avoid unnecessary continuous redraw.

Classic Light is the fresh/fallback appearance. An explicitly stored Dark selection remains supported on Windows. Changes must not accidentally make the visual default depend on combobox/order side effects.

Do not add decorative controls with no backend behavior. A visible option must map to a real validated setting/operation. Auxiliary dialogs and security/privacy prompts must honor the maintained 24-language contract rather than silently falling back to hardcoded English.

## Authentic UI evidence

When a Windows UI change affects documented appearance or Site Manager behavior, the dedicated screenshot workflow must build and launch the real x64 Portable executable and regenerate verified screenshots. Mockups or generated approximations are not acceptable production documentation evidence.

## Documentation changes

Update active documentation when user-visible behavior, package names, security/privacy boundaries or release behavior changes. Historical release records should remain historical rather than being rewritten to the current version.

All local links must pass `scripts/audit_docs.py`.

## Release changes

Changes to `.github/workflows/release.yml`, packaging or signing must preserve fail-closed behavior:

- truthful Windows signing state: verify a configured trusted Authenticode identity fail-closed, otherwise preserve explicit `WINDOWS_AUTHENTICODE=unsigned` metadata;
- never generate or present a self-signed development certificate as the trusted production publisher identity;
- exact `main` commit binding;
- immutable version tags;
- explicit 9-platform-artifact / 12-public-file allow-list;
- SHA-256 generation;
- GitHub Release read-back;
- stable GitHub Package/GHCR distribution-bundle read-back;
- no release secret material inside artifacts.

A missing production code-signing certificate alone is not a release failure. A partially configured certificate or an invalid signature when signing is configured **is** a release failure.

## Pull request expectations

A pull request should explain:

1. the defect/requirement;
2. the implementation approach;
3. security/privacy implications;
4. Windows/Linux impact;
5. tests/audits run;
6. documentation/release changes if applicable.

A green compile alone is not sufficient for security-, transfer-, UI- or release-sensitive changes.
