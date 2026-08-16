# Contributing to ByFTP

ByFTP is proprietary/source-available software owned by Brendigo. Publication of the source code does **not** grant a general right to modify, redistribute, rebrand or create derivative works.

## Authorization required

Issues, bug reports and feature suggestions are welcome. Source-code modifications and pull-request contributions should be made only when Brendigo has expressly requested or authorized that contribution.

Opening a fork or pull request does not by itself grant permission to modify or redistribute ByFTP outside the limited GitHub platform rights described in GitHub's Terms of Service and the repository [LICENSE](LICENSE).

## Development rules for authorized contributors

- Keep the desktop application native Win32; do not reintroduce a browser/localhost UI.
- Do not add telemetry, analytics, advertising SDKs, remote crash reporting or automatic external API calls.
- Do not add external Go modules without an explicit architecture/security review.
- Never place passwords, passphrases or private-key material in command-line arguments or persistent logs.
- Preserve SFTP host-key verification and local path traversal/reparse protections.
- Prefer typed Go interfaces over generic JSON/in-process dispatch layers.
- Keep user-facing errors concise and non-development-oriented.
- Do not remove or replace ByFTP/Brendigo branding or copyright notices without written authorization.
- Do not incorporate third-party code unless Brendigo has confirmed that its license is compatible with the proprietary ByFTP distribution model.

## Before an authorized pull request

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

## Licensing of authorized contributions

By submitting an authorized contribution, you represent that you have the right to submit it and agree that Brendigo may use, modify, distribute and license the accepted contribution as part of ByFTP under the repository's proprietary licensing model, unless a separate written agreement with Brendigo states otherwise.
