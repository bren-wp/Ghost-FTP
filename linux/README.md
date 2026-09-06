# Ghost FTP for Linux

Ghost FTP **1.1.1 Stable** is the current source candidate. Linux packages are built by `linux/BUILD.sh` for `amd64`, `arm64` and `i386`. Linux uses the same connection, profile, local-filesystem, remote-operation, transfer, settings and localization engine as the Windows application.

## Build

```bash
go telemetry off
bash linux/BUILD.sh
```

The script creates:

```text
dist/Ghost-FTP-X.Y.Z-Linux-amd64.deb
dist/Ghost-FTP-X.Y.Z-Linux-arm64.deb
dist/Ghost-FTP-X.Y.Z-Linux-i386.deb
```

The GitHub Release combines these verified packages into one additional public download:

```text
Ghost-FTP-X.Y.Z-Linux-multiarch.zip
```

## Installed identity

- Debian package: `ghost-ftp`
- executable: `/usr/bin/ghostftp`
- desktop name: **Ghost FTP**
- desktop entry: `ghost-ftp.desktop`

Runtime dependencies declared by the package are `ca-certificates`, `curl` and `openssh-client`. Ghost FTP does not bundle those projects or add external Go modules to the desktop/core module.

## Graphical desktop

When a local `DISPLAY` is available, `ghostftp` starts the native Ghost FTP graphical frontend by default. The GUI is implemented directly against X11/XWayland-compatible display transport without GTK, Qt, Electron, a webview or an external Go GUI module.

The graphical workspace includes Quick Connect, FTP/FTPS/implicit-FTPS/SFTP selection, SFTP host-key trust, saved profiles, dual local/server file panes, single-file and tree transfers, queue controls, local/remote file operations, remote permissions and validated transfer settings.

**Classic Light is the canonical Linux appearance.** The Linux frontend does not expose a theme switch whose backend cannot provide complete native runtime switching.

The fresh Quick Connect protocol is **explicit FTPS on port 21**. Plain FTP remains available as an explicit compatibility choice for servers that intentionally require unencrypted FTP; failed FTPS is not silently retried as FTP.

For a headless session, or to explicitly use the hardened command interface, set:

```text
GHOSTFTP_UI=terminal ghostftp
```

A graphical session requires a local X11-compatible display (native X11 or XWayland). The file-transfer protocols continue to use the system transport prerequisites documented below.

## Authentication

Linux supports the maintained desktop protocol contract:

- FTP with password authentication;
- explicit FTPS with certificate validation;
- SFTP with password authentication;
- SFTP with a private key and optional passphrase;
- explicit SFTP host-key fingerprint confirmation.

Passwords and key passphrases are cleared from the public connection config after authentication. Runtime protected-secret handles distinguish session-owned and borrowed profile-owned material so session close/failed setup can forget owned secrets without invalidating stored-profile credentials needed for a later reconnect.

The accepted public SFTP fingerprint can remain as non-secret session metadata so a saved profile can retain the verified endpoint identity.

## Connection lifecycle

Linux uses the same shared remote manager as Windows. Regression coverage exercises successful manager connection, remote listing/operation access and disconnect, plus invalid FTP credentials and failure of FTPS against a plaintext-only FTP endpoint.

Connection/transfer generation binding prevents stale work from silently continuing against a later session after reconnect.

## Terminal fallback: remote file commands

```text
pwd
ls [path]
cd <path>
mkdir <name>
rename <old> <new>
delete <name>
chmod <mode> <name>
```

## Terminal fallback: local file commands

The terminal fallback exposes a real local working directory rather than resolving relative transfer paths from the process working directory:

```text
lpwd
lls [path]
lcd <path>
lmkdir <name>
lrename <old> <new>
ldelete <name>
```

Local operations use the shared guarded filesystem service, including safe child-name validation, no-replace rename behavior, symlink/reparse protections and root-delete blocking.

## File and folder transfers

```text
get <remote-file> <local-file>
put <local-file> <remote-file>
gettree <remote-directory> <local-directory>
puttree <local-directory> <remote-directory>
```

Relative local paths resolve from the active local terminal directory. Relative remote paths resolve from the active remote directory.

Folder operations use the shared bounded tree-transfer planner. They retain the normal Ghost FTP item/depth limits, symlink handling, path validation, conflict policy and transfer queue behavior.

## Transfer queue

```text
jobs
pause
resume
cancel <id>
retry <id>
clear
```

These commands operate on the same transfer manager used by Windows. Transfer progress/state is derived from real queue snapshots; the graphical renderer suppresses unnecessary idle full-workspace redraw when relevant state is unchanged.

## Profiles

```text
profiles
profile-show <id>
profile-save <name>
profile-remove <id>
```

`profile-save` stores the active endpoint and current local/remote working paths plus the verified SFTP fingerprint/private-key path when applicable. It deliberately does not reconstruct or silently persist a password/passphrase that has already been cleared after authentication.

Profile output reports only public metadata and boolean credential-presence flags; it never prints stored password/passphrase material.

## Settings and languages

```text
settings
set parallelism <1-8>
set conflict <skip|replace|replace_backup>
set retries <0-3>
set retry-delay <1-30>
set timeout <5-60>
set confirm-delete <true|false>
language <code>
```

Delete confirmation applies to both remote `delete` and local `ldelete` when enabled.

English is the canonical/default language. The maintained registry contains **24 languages**, and Linux uses the same catalogs/fallback normalization as the Windows application and Setup.

## Safety model

The terminal parser does not invoke a shell for Ghost FTP commands. It bounds command length/argument count, supports quoted paths and rejects embedded NUL/newline control characters before dispatching only typed Engine calls.

Production build scripts require Go telemetry to be disabled and use controlled Go dependency settings from CI. See `docs/SECURITY.md`, `docs/PLATFORM-PARITY.md`, `docs/DEPENDENCIES.md` and `docs/TESTING.md` for the maintained release/security contract.
