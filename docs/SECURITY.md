# Security

Ghost FTP keeps transport, credential, remote-path, local-filesystem, account-state and transfer/recovery boundaries fail-closed.

**Current Ghost FTP release: 0.2.1**

The active desktop application platforms are **Windows and Linux**. The current `0.x` line is Beta until the complete stability/release criteria are met; the first stable release is `1.0.0`. Historical releases may document additional platforms that existed at the time, but those historical facts are not the active security/support contract.

## Desktop security boundary

Windows and Linux use the same typed `internal/api.Engine`, remote manager, transfer manager, settings/profile stores and security primitives. Frontends do not implement their own FTP/SFTP stack or queue scheduler.

Important invariants include:

- no generic JSON dispatcher or localhost/browser IPC between desktop UI and engine;
- strict raw connection validation before normalization can hide malformed input;
- secrets never written to application logs;
- endpoint/account/private-key binding for saved profile credentials;
- SFTP host-key fingerprint trust;
- bounded connection/session cleanup;
- staging and destination validation before final transfer promotion;
- recursive local operations protected against symlink/junction/reparse traversal;
- filesystem-root recursive deletion blocked;
- conflict/retry behavior centralized in the transfer engine;
- UI toolbar/menu actions mirror the canonical action-state checks rather than bypassing them.

## Credential handling

### Windows

Saved profile secrets use Windows DPAPI-backed protection. Runtime secret material is kept near the transport boundary and cleared/forgotten where practical.

### Linux

Linux uses the shared profile/runtime secret infrastructure. The current frontend supports SFTP password authentication and private-key authentication with an optional key passphrase.

Persisted Linux profile envelopes use authenticated AES-GCM storage with a per-user key file. The Ghost FTP state directory is required to be owned by the current user and private; the implementation may tighten only the verified leaf state directory to `0700`. The key file is a regular non-symlink file restricted to `0600`, and authenticated-encryption tampering is rejected.

Linux does not silently downgrade password/passphrase persistence to plaintext merely to imitate Windows DPAPI. Current Linux profile commands can persist non-secret connection metadata and verified public endpoint pins without reconstructing an already-cleared credential.

### Process handoff

External protocol processes receive a minimized environment. Password/passphrase values are not exposed as ordinary command-line arguments.

OpenSSH AskPass is constrained by an unpredictable runtime token and trusted parent-process checks. Ghost FTP does **not** create an on-disk AskPass password/passphrase file. Unknown/MFA-style prompts are refused instead of receiving a stored secret.

## SFTP

Ghost FTP uses OS OpenSSH `ssh`/`sftp` with an application-generated constrained configuration.

The configuration disables ambient behaviors that could change the trust/network boundary, including:

- ProxyCommand;
- ProxyJump;
- identity-agent inheritance;
- agent forwarding;
- ordinary forwarding;
- global known-host inheritance;
- DNS/update-host-key behavior used outside the application's explicit trust flow.

Private-key paths are checked using `Lstat`/reparse-point protections before use.

A new server key requires explicit fingerprint confirmation. Saved trust is bound to the expected endpoint rather than silently reused across a different server.

## FTP and FTPS

FTP/FTPS uses OS `curl` with configuration supplied through standard input.

Ghost FTP:

- starts curl in a mode that suppresses ambient user curl config;
- disables proxy use for the session;
- strips proxy-related environment variables;
- protects password lifetime around invocation;
- validates download staging files before promotion.

Explicit FTPS keeps certificate validation enabled. The application does not add a blanket `ssl-no-revoke` bypass.

Plain FTP remains supported only as an explicitly unencrypted compatibility option.

## Transfer and filesystem hardening

Transfers use staging/commit semantics and conflict-policy handling rather than overwriting destinations in an uncontrolled sequence. Temporary artifacts and recovery backups are named and validated within the intended destination namespace.

Remote cleanup and commit operations revalidate state where necessary so a server-side topology change cannot silently redirect finalization to an unsafe path.

Local recursive operations reject traversal through symlink/junction/reparse structures and block destructive recursion at filesystem roots.

## Privacy and network boundaries

Ghost FTP has no telemetry, analytics, advertising SDK, background account service or external crash-reporting service.

The application initiates network traffic only as required for the user's configured FTP/FTPS/SFTP destination and for explicit operating-system transport operations. There is no Ghost FTP cloud account or hidden synchronization endpoint.

See [Privacy](PRIVACY.md) and [Dependencies](DEPENDENCIES.md) for the maintained data and dependency contracts.

## Installer and release security

Windows Setup uses the same application binary as Portable for product functionality. Setup owns installation registration, shortcuts and the Windows Installed Apps uninstall entry; uninstall is integrated through the installed `GhostFTP.exe --uninstall` path rather than a separate uninstaller executable.

Installation mutations are transactional where supported. Registry values changed by Setup are snapshotted so a failed install/upgrade can restore the previous App Paths and Installed Apps registration state.

Production release workflows:

- derive version metadata from the root `VERSION` file;
- disable Go telemetry;
- build Windows and Linux in independent jobs;
- refuse mutable reuse of an existing release tag;
- generate SHA-256 checksums for all public release files;
- perform release asset read-back verification;
- keep signing credentials outside the repository;
- require a trusted Authenticode identity for stable Windows releases.

Pre-1.0 Beta Windows artifacts may be unsigned when no production signing identity is configured; that state is recorded explicitly in build metadata instead of being misrepresented as signed.

## Security issue reporting

Do not include production credentials, private keys, customer server addresses or secret material in public issues, screenshots, logs or test fixtures. Reports should contain the minimum reproduction data needed to demonstrate the issue safely.
