# Navigation and bookmarks

Ghost FTP maintains bounded navigation and bookmarks across its active product surfaces.

## Desktop navigation

Windows and Linux expose Back, Forward, current local/remote paths, Refresh and Bookmarks. Navigation history must not grow without bound or cross connection/profile ownership incorrectly.

## Bookmarks

Bookmarks are explicit user-created navigation targets. They must not contain hidden credentials or trigger an automatic connection without the normal connection decision path.

## Android

Android uses its native saved-site/bookmark model and mobile navigation. It does not replace the desktop ownership/history contract.

## UI contract

The main Files toolbar keeps **Back / Forward / Refresh / New Folder / Upload / Download / Bookmarks / More** in the canonical order where space permits.

macOS navigation/bookmark behavior is no longer maintained.

## Release boundary

Navigation/bookmark support belongs to the active Windows, Linux and Android product line. The current release shape is **13 platform artifacts / 16 public files** including browser helpers and release metadata.
