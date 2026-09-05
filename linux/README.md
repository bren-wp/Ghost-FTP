# Ghost FTP for Linux

Ghost FTP Linux packages are built by `linux/BUILD.sh` for `amd64`, `arm64` and `i386`. Linux uses the same connection, profile, local-filesystem, remote-operation, transfer, settings and localization engine as the Windows application.

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

## Authentication

Linux supports the maintained desktop protocol contract:

- FTP with password authentication;
- explicit FTPS with certificate validation;
- SFTP with password authentication;
- SFTP with a private key and optional passphrase;
- explicit SFTP host-key fingerprint confirmation.

Passwords and key passphrases are cleared from the connection config after authentication. The accepted public SFTP fingerprint can remain as non-secret session metadata so a saved profile can retain the verified endpoint identity.

## Remote file commands

```text
pwd
ls [path]
cd <path>
mkdir <name>
rename <old> <new>
delete <name>
chmod <mode> <name>
```

## Local file commands

The Linux terminal now exposes a real local working directory rather than resolving relative transfer paths from the process working directory:

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

These commands operate on the same transfer manager used by Windows.

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

English is the canonical/default language. The maintained registry contains 24 languages, and Linux uses the same catalogs/fallback normalization as the Windows application and Setup.

## Safety model

The terminal parser does not invoke a shell for Ghost FTP commands. It bounds command length/argument count, supports quoted paths and rejects embedded NUL/newline control characters before dispatching only typed Engine calls.

Production build scripts require Go telemetry to be disabled and use controlled Go dependency settings from CI. See `docs/SECURITY.md`, `docs/PLATFORM-PARITY.md`, `docs/DEPENDENCIES.md` and `docs/TESTING.md` for the maintained release/security contract.
