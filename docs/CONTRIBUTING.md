# Contributing

Keep changes small, testable and consistent with the existing security model. English is the canonical language for source-facing user text, documentation and repository metadata; desktop runtime translations belong in `internal/i18n` and must preserve the canonical English key/format contract.

The active desktop application targets are **Windows and Linux only**. Android, iOS and macOS are historical repository provenance and must not be reintroduced into the maintained application tree, build matrix or release workflow.

## Code boundaries

Platform code must stay within its intended boundary:

- shared desktop/core behavior belongs in `cmd/` and `internal/`;
- Windows-native UI/platform behavior belongs in Windows build-tagged files under the existing desktop/platform packages;
- Linux graphical/terminal platform behavior belongs in Linux build-tagged files and `linux/` packaging.

Do not broaden a `//go:build linux` or `//go:build windows` implementation into a generic fallback merely to make another operating system compile. Adding another maintained application platform would require an explicit product decision, complete security/build/release design and a corresponding platform-contract change.

## Security and privacy

Do not add:

- telemetry, analytics, advertising or session-replay SDKs;
- hidden product-controlled network destinations;
- insecure credential transport or plaintext persisted secrets;
- certificate, TLS or SFTP host-key bypasses;
- proxy/jump/agent behavior that escapes the application connection boundary;
- unsafe recursive filesystem operations;
- private signing keys or signing passwords.

Add regression tests for transfer, installer, profile-storage, filesystem and security-sensitive behavior whenever practical.

## Dependencies

The desktop/core Go module intentionally has no external Go module dependency graph. A new dependency requires explicit provenance, license, security/update ownership and documented justification. Do not add a GUI or protocol library merely to simplify a small helper that the maintained platform layer can safely implement.

Current protocol execution uses operating-system `curl` for FTP/FTPS and OpenSSH `ssh`/`sftp` for SFTP. Changes to that transport boundary require security regression coverage.

## Localization

English (`en`) is the canonical/default/fallback language. Changes to user-facing catalog-backed text must:

1. preserve the complete English key set;
2. preserve compatible format verbs/placeholders;
3. keep all 24 canonical locale registrations valid;
4. update Windows Setup copy when the same concept appears in installation;
5. keep Windows live switching and Linux runtime switching functional.

## Versioning

The current lifecycle is:

```text
0.1.0 Beta → 0.x.y Beta → 1.0.0 stable
```

`VERSION` is the single production version source. Windows Setup and Portable always carry the same version. Do not increment the version for an untested cosmetic edit; advance it only for a meaningful tested milestone.

Every `0.x.y` GitHub Release is a Beta/Prerelease. Stable publication begins at `1.0.0` and requires all stable release gates, including a trusted Windows Authenticode signing identity.

## Windows signing

Never commit a PFX/P12/private key, certificate password, hardware-token PIN or cloud signing credential.

Development signing may use the short-lived self-signed helper documented in `SIGNING.md`, but it is only a pipeline test. Production signing credentials must remain outside source control and must be supplied through the protected release boundary.

## Before opening or merging a pull request

Run the relevant repository audits and platform build gates. At minimum, changes affecting the maintained desktop/release surface must remain compatible with:

```text
gofmt
go test -race ./...
go vet ./...
python scripts/audit_repository.py
python scripts/audit_platform_contract.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_docs.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

Windows changes must pass the x64/x86 Setup/Portable production build and Authenticode signing smoke test. Linux changes must pass amd64/arm64/i386 DEB builds and metadata verification.

Changes that affect release packaging must keep the exact artifact allowlist, Beta/stable channel rule, immutable tag behavior, SHA-256 metadata and documentation synchronized.
