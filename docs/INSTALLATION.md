# Installation

Ghost FTP maintains installation packages for Windows, Linux and Android.

Current published version: **0.0.8**.

## Windows

Choose one:

- `Ghost-FTP-0.0.8-Setup.exe`
- `Ghost-FTP-0.0.8-Portable.exe`

Setup installs the application and registers supported integration points. Portable runs without a traditional installation.

## Linux

Choose the bundle for your distro family:

### Debian

- `Ghost-FTP-0.0.8-Linux-Debian-Installer.run`
- `Ghost-FTP-0.0.8-Linux-Debian-Portable.tar.gz`

### Ubuntu

- `Ghost-FTP-0.0.8-Linux-Ubuntu-Installer.run`
- `Ghost-FTP-0.0.8-Linux-Ubuntu-Portable.tar.gz`

### Fedora

- `Ghost-FTP-0.0.8-Linux-Fedora-Installer.run`
- `Ghost-FTP-0.0.8-Linux-Fedora-Portable.tar.gz`

Each bundle carries amd64, arm64 and i386 payloads and selects the compatible payload locally.

## Android

Install:

- `Ghost-FTP-0.0.8-Android.apk`

Verify the documented signing state for the specific release before sideloading.

Android exposes FTP and strict explicit FTPS. Android SFTP remains hidden until strict host-key verification is maintained.

## Browser helpers

Optional local helper ZIPs are published for Chrome, Edge, Firefox and Opera.

## Verify downloads

Always verify SHA-256 checksums from `SHA256.txt` and use the official GitHub release location.

The active 0.0.8 packaging contract is **13 platform artifacts / 16 public files**. macOS is no longer an active application target.
