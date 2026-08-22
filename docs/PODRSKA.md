# Support and troubleshooting

## Information to collect

Before reporting a problem, record the ByFTP version, operating system and architecture, selected protocol, server port, whether the problem occurs before or after authentication, and the exact sanitized error message. Never publish passwords, private keys, key passphrases, customer data or production credentials.

## Connection issues

- **Server not found:** verify DNS/host spelling.
- **Connection refused:** verify protocol, port and server service availability.
- **Timeout:** verify network/firewall routes and provider availability.
- **Authentication rejected:** verify the complete account username and password.
- **FTPS verification failure:** verify certificate and hostname configuration.
- **SFTP host key changed:** do not blindly accept it; confirm the new fingerprint with the administrator.

## Transfer issues

ByFTP distinguishes queued, running, completed, skipped, failed and cancelled transfers. The Windows queue supports pause/resume, cancellation, retry of failed/cancelled jobs and clearing terminal jobs. If only some batch operations succeed, the UI reports partial success and refreshes the remote view to show actual server state.

Symbolic links are intentionally treated conservatively. They are not automatically followed for recursive transfer or destructive operations where doing so could escape the selected tree.

## UI and localization

English is the default language. The Windows client can change language without restarting; Linux/macOS use `language <code>`. If a supported locale is selected, application workflow text must come from the localization catalog rather than a mixed-language fallback.

## Reporting defects

Use the GitHub issue template and provide minimal reproduction steps. Security-sensitive reports should avoid public disclosure of exploitable details or credentials. The project does not collect telemetry, so reproducible user-provided diagnostics are important.