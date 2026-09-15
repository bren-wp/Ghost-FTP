# Navigation bookmarks and profile start directories

Ghost FTP **0.0.6** maintains navigation bookmarks and explicit local/server profile start directories across the desktop engine. Android has its own saved-site/bookmark model. Browser extensions do not own desktop bookmark state.

## Security model

Bookmarks are non-secret metadata. Remote bookmark and profile-start identity is bound to protocol, canonical host, port and exact username. Passwords, passphrases, private-key data and host-key trust material are never bookmark fields.

Stored paths are requested destinations, not pre-verified authority. Local and remote activation performs fresh validation/listing and rejects account mismatch, disconnect/reconnect races and stale completion before visible state commits.

## Windows and Linux

Windows and Linux expose native bookmark management over the same shared Engine ownership. Create/open/delete actions must go through authoritative Engine paths and may not bypass session generation/account validation.

## macOS development behavior

The native AppKit source uses the **same shared bookmark Engine APIs** and preserves **source/development parity**. macOS remains a separately validated native development/source frontend; a successful source build is not Developer ID signing/notarization evidence.

## Android

Android bookmark/site state uses platform-native UI and lifecycle rules. It must not persist plaintext credentials or weaken protocol trust boundaries.

## Privacy and failure behavior

Bookmark/profile metadata remains local. Failures preserve previously verified state where possible and never manufacture a successful empty listing. Paths and usernames may be sensitive diagnostic metadata and should be minimized in logs.

## 0.0.6 release boundary

The published Ghost FTP 0.0.6 release is **13 platform artifacts / 16 public files**. Navigation/bookmark behavior remains part of the maintained product contract, while Android SFTP remains hidden until strict host-key verification exists and macOS remains outside the public release.
