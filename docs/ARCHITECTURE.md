# Architecture

Ghost FTP 2.x is a **Windows + Linux** desktop product with one shared transfer/security core and platform-specific presentation layers.

The repository also retains the Ghost FTP Web companion as a separate shared-hosting/PWA source surface. The Web companion is not part of the Windows/Linux desktop application artifact contract.

## Design goals

The architecture is optimized for:

- one protocol/transfer implementation shared across desktop frontends;
- typed in-process UI-to-engine calls rather than browser/localhost IPC;
- strict credential lifetime and trust boundaries;
- conservative remote overwrite/recovery semantics;
- platform-native behavior where it improves security or usability;
- no external Go module dependency graph in the desktop/core module;
- explicit, auditable OS transport prerequisites;
- no application telemetry, analytics or background tracking.

## Layer overview

### `cmd/ghostftp`

Application entrypoint and secure AskPass helper mode. It owns startup, single-instance/process setup and the narrowly scoped credential prompt bridge required by OpenSSH.

### `internal/desktop`

Platform presentation layer.

- Windows files use Win32 controls, custom owner-drawn visual elements, high-DPI layout, native dialogs and graphical local/remote/queue panels.
- Linux uses the terminal frontend in `other.go` with a `linux` build tag.

Both frontends call the same typed `internal/api.Engine` methods.

### `internal/api`

The stable in-process application boundary. It exposes typed operations for:

- profiles and settings;
- connect/disconnect/trust;
- local filesystem actions;
- remote filesystem actions;
- transfer queue control;
- single-file and tree transfer planning.

There is no generic JSON dispatcher, localhost HTTP server or browser IPC layer between UI and engine.

### `internal/remote`

Connection/session ownership and protocol adapters.

Responsibilities include:

- endpoint/account/private-key identity binding;
- protected secret handoff;
- SFTP host-key trust;
- connection probing/diagnostics;
- safe session operation acquisition/release;
- bounded disconnect cleanup;
- protocol-specific FTP/FTPS/SFTP operations.

Current protocol execution delegates to OS tools:

- `curl` for FTP/FTPS;
- OpenSSH `ssh`/`sftp` for SFTP.

Ghost FTP supplies constrained configuration/environment rather than inheriting unsafe ambient proxy/SSH behavior.

### `internal/transfer`

Shared transfer scheduler and queue. It owns:

- parallelism;
- pause/resume;
- cancellation;
- retry and retry delay;
- conflict policy;
- event/snapshot reporting;
- safe final status handling.

Windows buttons and Linux terminal queue commands invoke this same manager.

### `internal/config`

Persistent settings/profile storage with validation, migration and guarded atomic writes.

The canonical conflict policy is one of:

- `skip`;
- `replace`;
- `replace_backup`.

`replace_backup` is the conservative default and persisted unknown states migrate back to it.

### `internal/security`

Reusable validation and filesystem/secret primitives, including:

- connection validation;
- secret validation;
- SFTP fingerprint validation;
- runtime secret protection/forgetting;
- local path/root checks;
- recursive remove-without-follow protections;
- reparse/symlink checks.

### `internal/localfs`

Local filesystem browser/action layer. It applies no-follow/reparse-point and rename/delete safeguards rather than allowing each UI to manipulate files directly.

### `internal/i18n`

Canonical English-first runtime registry and catalogs. English is the default/fallback and the current registry contains 24 languages.

### `internal/platform`

OS-specific native behavior such as Windows hardening/dialogs/credential protection/file moves and cross-platform equivalents where appropriate.

## Connection flow

A desktop connection follows this sequence:

1. frontend collects raw protocol/endpoint/credential input;
2. strict raw connection validation runs before normalization can hide malformed input;
3. profile resolution may supply saved secrets only when endpoint/account/private-key identity still matches;
4. protected runtime secret objects are created near the transport boundary;
5. SFTP performs host-key discovery/trust validation when needed;
6. the remote manager creates the session;
7. the existing initial listing is used for lightweight connection diagnostics;
8. the transfer manager is enabled only after a confirmed connection.

Saved credentials must not silently migrate to a different endpoint/account/key identity.

## SFTP process boundary

Ghost FTP creates a constrained OpenSSH configuration that disables ambient features such as proxy commands, jump hosts, identity agents and forwarding.

Password/passphrase prompts use the Ghost FTP AskPass helper with an unpredictable runtime token and protected secret blob. The application rejects untrusted helper-parent context and does not create an on-disk password/passphrase file.

## FTP/FTPS process boundary

Ghost FTP invokes curl with configuration supplied on standard input, suppresses ambient curl config, clears proxy use and sanitizes proxy-related environment variables. Passwords are not placed in command-line arguments.

FTPS certificate validation remains enabled and the application does not use a blanket revocation-check bypass.

## Transfer safety

Upload/download operations use staging and validation before final promotion. High-risk invariants include:

- destination paths validated before queueing;
- local path constrained to the expected root;
- download part file validated against symlink/reparse substitution;
- overwrite recovery controlled by conflict policy;
- directory transfer planning bounded by depth/item limits;
- symlink handling is explicit;
- cancellation cannot rewrite a completed result after success;
- disconnect waits for active session operations within bounded cleanup rules.

## Windows presentation

Windows is the graphical reference frontend. The 2.0 visual system uses a premium graphite/navy palette, high-DPI scaling and native owner-drawn buttons while remaining free of a third-party GUI framework.

The Win32 layer owns only presentation/input orchestration; core connection/transfer/security behavior remains outside the UI files.

## Linux presentation

Linux 2.0 uses a hardened terminal interface over the same engine. It supports the same SFTP password/private-key/passphrase model, remote actions, transfer scheduler and validated settings store.

The frontend build tag is explicitly `linux`; retired macOS application handling no longer shares this source path.

See [Platform parity](PLATFORM-PARITY.md) for the current exposed-feature matrix.

## Web companion boundary

`GhostFTP WEB/` is a separate PHP/PWA implementation intended for shared hosting. It has its own web threat model, session/CSRF boundaries and PHP extension requirements.

It is kept in the same repository for product/source continuity but must not be described as a Windows/Linux desktop runtime component or counted as a 2.x desktop release platform artifact.

## Retired platforms

Android, iOS and macOS application source trees were removed from active 2.x. Their 1.x implementation history remains in Git and published release provenance.

New desktop work must not reintroduce those platform roots without an explicit future product decision and a new compatibility/release review.

## Architectural change rule

New connection/transfer options should be implemented in shared core first, then exposed by Windows and Linux frontends. A frontend-specific duplicate protocol or scheduler implementation is considered architectural drift and should fail review.
