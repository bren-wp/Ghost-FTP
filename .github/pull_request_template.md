## Authorization

- [ ] This source-code change was explicitly requested or authorized by Brendigo.
- [ ] I have read and will follow `LICENSE` and the contribution documentation.

> ByFTP is proprietary/source-available software. Opening a pull request or fork does not by itself grant rights to modify, redistribute, rebrand, sell, sublicense, or create derivative distributions beyond rights explicitly granted by Brendigo and limited GitHub platform rights.

## Summary

Describe the change and why it is needed.

## Security and privacy

- [ ] No telemetry, analytics, advertising SDK, or external crash-reporting service was added.
- [ ] No automatic external API or hidden network destination was added.
- [ ] Passwords, private-key passphrases, and private keys do not end up in command-line arguments or persistent logs.
- [ ] SFTP host-key verification and pinning remain active.
- [ ] Local path-traversal, symlink, junction, and reparse-point protections remain active.
- [ ] No external Go module was added without explicit architecture/security review.
- [ ] Tests, screenshots, and fixtures contain no real credentials, production servers, or customer data.
- [ ] New canonical user-facing text and documentation are English-first and runtime UI text is added through the localization system.

## Validation

- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] `python scripts/generate_brand_assets.py --check`
- [ ] `python scripts/audit_localization.py`
- [ ] `python scripts/audit_privacy.py`
- [ ] Windows production builds were verified when Windows-specific code changed.

## Regression coverage

Describe tests added or changed for connections, transfers, paths, profiles, installation/uninstallation, localization, or other affected behavior.
