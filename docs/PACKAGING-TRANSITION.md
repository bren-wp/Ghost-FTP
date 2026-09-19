# Packaging transition

Ghost FTP packaging is intentionally centered on **Windows, Linux and Android**.

## Windows

The public surface is one universal Setup executable and one universal Portable executable. Architecture-specific payloads remain internal to those launchers.

## Linux

Debian, Ubuntu and Fedora each receive one Installer and one Portable bundle. Public packages are distro-oriented, while architecture selection happens inside the package.

## Android

Android publishes one verified APK. SFTP remains hidden until strict maintained host-key verification is implemented on Android.

## Retired packaging

The macOS application and all browser-extension packages, source trees, workflows and release artifacts are retired. They must not re-enter active packaging or release metadata.

## Version discipline

A published package set is immutable. Every later product or release-facing change advances the semantic version and rebuilds exact-source packages, checksums and evidence.
