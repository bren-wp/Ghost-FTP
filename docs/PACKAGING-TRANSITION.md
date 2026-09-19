# Packaging transition

Ghost FTP packaging is now centered on three maintained native applications plus browser helpers.

## Windows

One public Setup and one public Portable executable carry universal x64 / x86 / ARM64 payloads.

Architecture-specific executables remain internal staging details and must not leak into the public release allow-list.

## Linux

Debian, Ubuntu and Fedora each receive:

- one architecture-selecting Installer;
- one architecture-selecting Portable bundle.

Each carries amd64, arm64 and i386 payloads.

## Android

One public APK is produced and its signing state is declared explicitly.

## Browser helpers

Chrome, Edge, Firefox and Opera each receive one helper ZIP.

## Release totals

13 platform artifacts + RELEASE-NOTES + BUILD-METADATA + SHA256 = 16 public files.

The retired macOS package is no longer part of the active packaging contract.
