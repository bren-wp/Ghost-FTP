# Architecture

Ghost FTP is a three-application file-transfer product built around shared typed Go engine logic and platform-native frontends for **Windows**, **Linux** and **Android**.

## Core engine

The shared engine owns protocol and transfer behavior. UI layers must not invent parallel protocol state.

Core responsibilities include:

- connection profiles;
- local and remote navigation;
- FTP / FTPS / SFTP desktop transport;
- transfer queue lifecycle;
- bookmarks and start directories;
- file mutations;
- remote permissions;
- Remote Edit;
- directory comparison;
- search and filtering;
- settings and localization state.

## Windows

Windows uses the native desktop frontend and universal packaging path. The maintained public artifacts are Setup and Portable executables carrying x64, x86 and ARM64 payloads.

## Linux

Linux uses the native desktop frontend and distro-specific universal bundles. Debian, Ubuntu and Fedora Installer + Portable packages carry amd64, arm64 and i386 payloads.

## Android

Android is a native application with Android lifecycle and storage-access constraints. It shares Ghost FTP product concepts, not desktop window ownership.

Android exposes FTP and strict explicit FTPS. SFTP stays hidden until the Android transport has the same maintained strict host-key trust boundary as desktop.

## Browser helpers

Browser packages are local helpers, not a second FTP engine. On supported Windows installs they may hand off a sanitized connection descriptor to the desktop application without secrets and without automatic connection.

## Retired platform

The former macOS application and its platform-specific source, build, signing and notarization pipeline were removed from the active repository surface.

## Privacy

No maintained frontend may introduce analytics, telemetry, advertising, a mandatory product account or a hidden transfer proxy.
