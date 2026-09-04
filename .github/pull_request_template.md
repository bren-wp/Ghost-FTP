## Authorization

- [ ] This source-code change was explicitly requested or authorized for Ghost FTP.
- [ ] I have read and will follow `LICENSE` and the contribution documentation.

> Ghost FTP is source-available software. Opening a pull request or fork does not itself grant rights beyond the repository license and applicable GitHub platform rights.

## Summary

Describe the change and why it is needed.

## Branding and compatibility

- [ ] New public UI, documentation and release assets use **Ghost FTP**.
- [ ] Any retained `ByFTP`/`byftp` identifier is required for backward compatibility, migration or installed-application identity and is not user-facing branding.

## Security and privacy

- [ ] No telemetry, analytics, advertising SDK or external crash-reporting service was added.
- [ ] No automatic external API or hidden network destination was added.
- [ ] Passwords, private-key passphrases and private keys do not end up in command-line arguments or persistent logs.
- [ ] SFTP host-key verification and pinning remain active.
- [ ] Local path-traversal, symlink, junction and reparse-point protections remain active.
- [ ] No external Go module was added without explicit architecture/security review.
- [ ] Tests, screenshots and fixtures contain no real credentials, production servers or customer data.

## Validation

- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] `python scripts/audit_security.py`
- [ ] `python scripts/audit_privacy.py`
- [ ] PHP syntax checks pass when web code changed.
- [ ] Native platform build checks pass for affected platforms.

## Regression coverage

Describe tests added or changed for connections, transfers, paths, profiles, installation/uninstallation, localization, release packaging or other affected behavior.
