# Contributing to ByFTP

Thanks for helping improve ByFTP.

## Development rules

- Keep the desktop application native Win32; do not reintroduce a browser/localhost UI.
- Do not add telemetry, analytics, advertising SDKs, remote crash reporting or automatic external API calls.
- Do not add external Go modules without an explicit architecture/security review.
- Never place passwords, passphrases or private-key material in command-line arguments or persistent logs.
- Preserve SFTP host-key verification and local path traversal/reparse protections.
- Prefer typed Go interfaces over generic JSON/in-process dispatch layers.
- Keep user-facing errors concise and non-development-oriented.

## Before opening a pull request

Run:

```text
go test ./...
go vet ./...
python scripts/audit_privacy.py
```

On Windows, also run:

```powershell
.\BUILD-WINDOWS.ps1
```

If the change affects transfers, connection lifecycle, local/remote paths, profiles, SFTP trust, installer or uninstaller behavior, add regression coverage.

## Privacy-safe bug reports

Never include:

- real passwords or passphrases
- private SSH keys
- production hostnames unless intentionally public
- FTP/SFTP usernames
- customer directory structures or confidential filenames

Use sanitized examples instead.
