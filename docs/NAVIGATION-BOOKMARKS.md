# Navigation and bookmarks

Ghost FTP **0.0.8** maintains navigation and bookmark behavior across Windows, Linux and Android.

## Windows and Linux

The desktop Files workspace provides:

- Back;
- Forward;
- current local path;
- current remote path;
- bookmarks;
- saved connection start directories;
- More for secondary file/navigation operations.

Navigation history is bounded and must not duplicate entries indefinitely.

Bookmarks use the shared engine/profile state and do not create a second protocol stack.

## Android

Android exposes saved Sites and Bookmarks using platform-appropriate navigation and storage lifecycle behavior.

## Security

Bookmark and profile navigation must never embed plaintext passwords, private-key passphrases or secret query fragments into public handoff URLs or logs.

## Release boundary

The active native applications are Windows, Linux and Android. The release contract is **13 platform artifacts / 16 public files**.
