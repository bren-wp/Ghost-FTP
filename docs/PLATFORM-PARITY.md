# Platform parity

Ghost FTP maintains one product language across **Windows, Linux and Android** while preserving platform-native lifecycle behavior.

## Shared product hierarchy

All maintained applications expose the same primary concepts:

- Files;
- Connections / Sites;
- Transfer Queue / Transfers;
- Settings;
- Bookmarks;
- Quick Connect / connection state.

The Files workspace is reference-driven around:

- Back / Forward / Refresh;
- New Folder;
- Upload / Download;
- Local Files;
- Remote Files;
- Transfer Queue;
- readable status, progress, speed and ETA;
- More for secondary engine-backed actions.

## Windows

Windows is a primary desktop target. It publishes universal Setup and Portable applications with x64, x86 and ARM64 payloads.

The maintained Files surface follows the desktop master layout and keeps secondary operations in native menus rather than permanently cluttering the workspace.

## Linux

Linux is a primary desktop target. It publishes Debian, Ubuntu and Fedora Installer + Portable bundles.

The Linux surface mirrors the Windows master hierarchy while retaining Linux window management and runtime behavior. The left rail is limited to Files, Connections, Transfer Queue and Settings. Bookmarks and secondary actions live in the master toolbar / More surface.

## Android

Android is a primary mobile target.

The Android layout adapts the same concepts to touch:

- Production Server / connection card;
- Back / Forward / Refresh / New Folder / Upload / Download / Bookmarks / More;
- Local Files and Remote Files cards;
- Transfer Queue;
- bottom navigation for Files, Sites, Bookmarks, Transfers and Settings.

All touch targets and labels must remain readable and unclipped.

## Retired platform boundary

The former macOS application is no longer an active platform. Its app source, build scripts, signing/notarization workflows and dedicated regression tests have been removed from the maintained source tree.

## Evidence rule

A platform is not declared visually complete merely because a reference mockup exists. Runtime parity claims require authentic screenshots tied to the exact tested source SHA.

