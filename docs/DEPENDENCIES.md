# Dependencies

Ghost FTP keeps runtime dependencies narrow and platform-specific.

## Shared Go code

The desktop engine and supporting packages use the Go toolchain declared by repository workflows. Dependencies must be reviewed for security, licensing and runtime necessity.

## Windows

Windows runtime behavior relies on native operating-system facilities and the maintained Windows OpenSSH Client boundary for desktop SFTP where applicable.

## Linux

Linux uses distribution-provided runtime facilities and OpenSSH tooling for desktop SFTP. Package workflows validate supported distro bundles and required system boundaries.

## Android

Android uses the maintained Android toolchain and platform APIs. Desktop dependencies are not pulled into the Android application.

## Browser helpers

Browser companion packages remain local helper surfaces and are not a parallel file-transfer engine.

## Retired macOS dependency boundary

Apple/AppKit/Developer-ID/notarization tooling is no longer required by the active product because macOS application support has been removed.
