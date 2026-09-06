# Security

Ghost FTP keeps transport, credential, remote-path, local-filesystem, account-state and transfer/recovery boundaries fail-closed.

**Current Ghost FTP release: 0.1.1**

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

## Remote listing and permission metadata

The Windows reference UI exposes a remote **Permissions** column only when the remote listing supplies a usable mode. The value is treated as display metadata, not as authorization proof.

Accepted display forms are intentionally narrow:

- UNIX-style symbolic modes such as `-rw-r--r--` and `drwxr-xr-x` from FTP `LIST` or SFTP `ls -la`;
- three/four-digit octal `unix.mode` values supplied by MLSD.

MLSD `perm=` values such as `adfrw` are capability strings and are not presented as POSIX file modes. Malformed or unexpected mode strings are discarded instead of being rendered as trusted permissions.

The permission display never relaxes the real server-side permission checks. Mutations still succeed or fail according to the remote transport/server response and normal Ghost FTP validation.

## Remote search boundary

The Windows remote-search field filters the already loaded directory model in memory. It does not transmit the search term to a Ghost FTP service or third party and does not issue a new remote-server request for every keystroke.

A separate full directory model is retained so filtered list indexes cannot accidentally redirect rename/delete/download operations to a different unfiltered item.

## Transfer and overwrite safety

The transfer engine uses staging/recovery logic rather than directly replacing destinations whenever safe recovery is required.

Security/reliability properties include:

- remote destination validation before queueing;
- local path constrained to the expected local root;
- download part files checked against symlink/reparse substitution;
- conservative default `replace_backup` conflict policy;
- bounded automatic retries and retry delay;
- safe final-status handling so late cancellation cannot overwrite a completed result;
- directory/tree planning bounded by maximum depth/item count;
- explicit symlink handling;
- bounded disconnect while active operations are released.

## Profile identity binding

Saved passwords are reused only when the account identity still matches. Saved private-key passphrases require matching private-key identity. SFTP fingerprints are associated with the expected endpoint.

Editing a saved profile to another host/account/key must not silently carry secrets/trust into the new identity.

## Local filesystem protections

Local file operations are routed through the local filesystem/security layers rather than direct UI filesystem mutation.

Protections include:

- no-follow/symlink/reparse checks;
- guarded recursive removal;
- filesystem-root deletion refusal;
- bounded recursion/item counts;
- no-replace rename semantics where supported;
- guarded atomic state-file writes;
- checks that opened/replaced state files are still the expected filesystem object.

## Windows process, UI and installer hardening

Windows startup uses process error/DLL-loading hardening and safe native picker flags.

The reference shell is a presentation layer over existing command/action-state paths. Toolbar/menu actions do not create alternate unchecked implementations of upload, download, rename, delete, connection or queue operations.

The local connection log is bounded and sanitizes embedded line-breaking control characters before display. The Diagnostics action renders local state only; it does not contain an upload endpoint.

Ghost FTP Setup uses an application-only verified payload and rollback-aware file/registry behavior. Setup and Portable package the same Windows application source, so security behavior after application startup does not fork by package type.

The repository does not contain publisher secrets. Authenticode signing requires an external legitimate signing identity; unsigned artifacts must be described as unsigned.

## Linux packaging and presentation boundary

Linux DEB packages are built from the same source version and verified for package name/version/architecture in CI. Current transport prerequisites are system-provided `curl` and OpenSSH tools; they are documented rather than hidden or downloaded by the application.

The maintained Linux frontend is currently terminal-based. A future Linux GUI must not fork the transfer/security engine or add hidden tracking. Any GUI runtime/toolkit prerequisite must be reviewed and documented rather than mislabeled as “zero dependency.”

## Privacy and tracking

Ghost FTP desktop runtime contains no application analytics/advertising/crash-reporting SDK and no fixed application telemetry backend.

Privacy auditing rejects:

- known telemetry/vendor markers;
- fixed HTTP(S) URLs in desktop runtime source;
- general-purpose runtime network imports outside the constrained protocol architecture;
- credential/proxy environment leakage;
- ineffective production telemetry-disable configuration.

Production build workflows explicitly execute `go telemetry off` and verify the state.

## Web companion security boundary

`GhostFTP WEB/` remains a separate shared-hosting/PWA implementation with its own PHP/session/CSRF threat model. It is not a Windows/Linux desktop runtime component.

Its maintained security properties include strict session/CSRF controls, authenticated encryption for saved secrets, bounded operations, host/path validation, SFTP fingerprint checks, staged remote writes and safe public-error handling.

The Web companion is audited separately and is not published as a desktop platform artifact.

## Repository and release integrity

Repository audits reject unsafe/generated source-tree drift, including tracked one-shot workflows and retired platform application roots.

The production release workflow publishes **9 platform artifacts** plus `RELEASE-NOTES.txt`, `BUILD-METADATA.txt` and `SHA256.txt`, for **12 public files** total.

Before publication:

- shared quality/security/docs must pass;
- Windows production build must pass;
- Linux production build must pass;
- `main` must still point to the build commit;
- an existing `ghostftp-vX.Y.Z` tag must already point to that same commit or publication fails;
- final release asset count is read back and verified;
- every `0.x.y` GitHub release is marked Beta/Prerelease;
- stable publication starts at `1.0.0`.

Published historical tags/releases are not moved to another commit.

## Dependency integrity

The desktop/core Go module has no external Go modules and CI rejects a new module/vendor graph.

This is distinct from OS runtime prerequisites. Current protocol execution uses:

- `curl` for FTP/FTPS;
- `ssh`/`sftp` for SFTP.

Any future embedded protocol library or Linux graphical runtime would require explicit license/provenance and security/dependency review.

## Package integrity and publisher identity

Release consumers should verify `SHA256.txt` and `BUILD-METADATA.txt` before installation in managed environments.

Checksums prove file integrity relative to the published manifest; they do not replace publisher signing. See [Signing](SIGNING.md) and [Release verification](RELEASE-VERIFICATION.md).

## Authentic UI evidence

The Windows UI screenshots used by maintained documentation are captured from the real production x64 Portable executable by the dedicated Actions workflow. The workflow validates PNG structure, dimensions, file size and SHA-256 and refuses to persist a stale capture if the source branch moved.

Mockup or generated-image output is not accepted as evidence that the production executable renders the claimed UI.

See [Desktop reference UI](REFERENCE-UI.md).

## Security regression gates

`scripts/audit_security.py`, `scripts/audit_privacy.py`, `scripts/audit_dependencies.py`, `scripts/audit_platform_contract.py` and the Go/Python regression suites are release gates, not informational reports.

The UI stability regression suite additionally checks that the remote Permissions column remains backed by validated LIST/SFTP/MLSD metadata and that Windows action surfaces do not revert to hardcoded Croatian strings when another runtime language is selected.

A high-risk invariant removed from code must either be replaced with an equivalent/stronger reviewed mechanism and tests or the change must fail review.

## Reporting

Report vulnerabilities through the repository Security policy. Never place working passwords, private keys, signing credentials, production endpoints or customer data in a public issue.
