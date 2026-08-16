## Authorization

- [ ] This source-code change was expressly requested or authorized by Brendigo.
- [ ] I have read and will comply with the repository `LICENSE` and `CONTRIBUTING.md`.

> ByFTP is proprietary/source-available software. Opening a pull request or fork does not grant permission to modify, redistribute, rebrand, sell, sublicense or create derivative distributions outside the rights expressly granted by Brendigo and the GitHub platform terms.

## Summary

Describe the change and why it is needed.

## Security & privacy checklist

- [ ] No telemetry, analytics, advertising SDK or external crash-reporting service was added.
- [ ] No automatic external API or hidden network destination was added.
- [ ] Passwords, passphrases and private-key material are not placed in command-line arguments or persistent logs.
- [ ] SFTP host-key verification and pinning remain enforced.
- [ ] Local path traversal, symlink, junction and reparse-point protections remain enforced.
- [ ] No external Go module was added without explicit architecture/security approval.
- [ ] No real credentials, production hostnames or customer data are included in tests, screenshots or fixtures.

## Validation

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `python scripts/audit_privacy.py`
- [ ] Windows production build was tested when the change affects Windows-specific code.

## Regression coverage

Describe tests added or updated for connection lifecycle, transfers, paths, profiles, installer/uninstaller or other affected behavior.
