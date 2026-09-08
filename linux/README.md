# Ghost FTP for Linux

Ghost FTP **1.1.6 Stable** is the current published release. The maintained `main` source may contain post-1.1.6 hardening and packaging improvements before a later maintenance release is prepared. Linux uses the same connection, profile, local-filesystem, remote-operation, transfer, settings and localization engine as the Windows application.

## Build

```bash
go telemetry off
bash linux/BUILD.sh
```

The build always creates package-manager-neutral portable archives for `amd64`, `arm64` and `i386`:

```text
dist/Ghost-FTP-X.Y.Z-Linux-amd64.tar.gz
dist/Ghost-FTP-X.Y.Z-Linux-arm64.tar.gz
dist/Ghost-FTP-X.Y.Z-Linux-i386.tar.gz
```

When `dpkg-deb` is available, the same build also creates:

```text
dist/Ghost-FTP-X.Y.Z-Linux-amd64.deb
dist/Ghost-FTP-X.Y.Z-Linux-arm64.deb
dist/Ghost-FTP-X.Y.Z-Linux-i386.deb
```

Production CI sets `GHOSTFTP_REQUIRE_DEB=1`, so an official Linux production build fails closed if the DEB tooling is unavailable. On distributions that do not ship `dpkg-deb`, a local source build can still produce the portable tarballs without installing a Debian packaging tool solely for that purpose.

The portable archive and DEB for each architecture are built from the same compiled `ghostftp` binary. CI extracts both formats and compares those executables byte-for-byte before accepting the Linux job.

### Published 1.1.6 note

The already published Ghost FTP 1.1.6 release is immutable and keeps its original Linux asset set: three DEB files plus `Ghost-FTP-1.1.6-Linux-multiarch.zip`. The portable `.tar.gz` output is a post-1.1.6 source/CI packaging improvement and is **not** retroactively claimed as a 1.1.6 release asset.

## Distro-neutral portable use

The `.tar.gz` package is intended for Linux distributions where a Debian package is not the native installation format, including RPM-based and rolling-release environments. It does not pretend to be an RPM, Flatpak, AppImage, Snap or distribution repository package; those formats require their own verified packaging lifecycle before they can be called officially supported artifacts.

Extract the archive matching the machine architecture:

```bash
tar -xzf Ghost-FTP-X.Y.Z-Linux-amd64.tar.gz
cd Ghost-FTP-X.Y.Z-Linux-amd64
./ghostftp
```

Each archive contains:

```text
ghostftp
ghost-ftp.desktop
ghost-ftp.png
LICENSE
README.md
```

For optional per-user desktop integration without root privileges:

```bash
mkdir -p "$HOME/.local/bin" \
  "$HOME/.local/share/applications" \
  "$HOME/.local/share/icons/hicolor/512x512/apps"
install -m 0755 ghostftp "$HOME/.local/bin/ghostftp"
install -m 0644 ghost-ftp.desktop "$HOME/.local/share/applications/ghost-ftp.desktop"
install -m 0644 ghost-ftp.png "$HOME/.local/share/icons/hicolor/512x512/apps/ghost-ftp.png"
```

Ensure `$HOME/.local/bin` is on `PATH` before launching from the desktop entry. System protocol prerequisites still apply: a usable CA certificate store, `curl` for FTP/FTPS and OpenSSH client tools for SFTP.

## Installed identity

- Debian package: `ghost-ftp`
- executable installed by DEB: `/usr/bin/ghostftp`
- user-local portable executable: `$HOME/.local/bin/ghostftp` when installed as shown above
- desktop name: **Ghost FTP**
- desktop entry: `ghost-ftp.desktop`

The DEB declares runtime dependencies on `ca-certificates`, `curl` and `openssh-client`. The portable archive intentionally does not bundle distribution package-manager metadata or copies of those system tools.

## Graphical desktop

When a local `DISPLAY` is available, `ghostftp` starts the native Ghost FTP graphical frontend by default. The GUI is implemented directly against X11/XWayland-compatible display transport without GTK, Qt, Electron, a webview or an external Go GUI module.

The graphical workspace includes Quick Connect, FTP/FTPS/implicit-FTPS/SFTP selection, SFTP host-key trust, saved profiles, dual local/server file panes, single-file and tree transfers, queue controls, local/remote file operations, remote permissions and validated transfer settings.

**Classic Light is the canonical Linux appearance.** The Linux frontend does not expose a theme switch whose backend cannot provide complete native runtime switching.

The fresh Quick Connect protocol is **explicit FTPS on port 21**. Plain FTP remains available as an explicit compatibility choice for servers that intentionally require unencrypted FTP; failed FTPS is not silently retried as FTP.

For a headless session, or to explicitly use the hardened command interface, set:

```text
GHOSTFTP_UI=terminal ghostftp
```

A graphical session requires a local X11-compatible display (native X11 or XWayland). The file-transfer protocols continue to use the system transport prerequisites documented above.

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

Local operations use the shared guarded filesystem service, including safe child-name validation, no-replace rename behavior, symlink/reparse protections, rooted local creation/deletion and root-delete blocking.

## File and folder transfers

```text
get <remote-file> <local-file>
put <local-file> <remote-file>
gettree <remote-directory> <local-directory>
puttree <local-directory> <remote-directory>
```

Relative local paths resolve from the active local terminal directory. Relative remote paths resolve from the active remote directory.

Folder operations use the shared bounded tree-transfer planner. They retain the normal Ghost FTP item/depth limits, symlink handling, path validation, conflict policy and transfer queue behavior. Tree-download directory preparation remains anchored to opened filesystem roots so a later boundary or parent pathname swap cannot redirect directory creation.

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
