# Ghost FTP Repository Naming Policy

Ghost FTP keeps product-facing naming consistent without destabilizing framework-required internals.

## Product-facing naming

Use **Ghost FTP** for:

- product copy;
- UI labels;
- marketing text;
- documentation prose;
- release titles.

Use **GhostFTP** for:

- filenames;
- archive names;
- package names;
- technical product identifiers;
- CI artifact names;
- executable names.

Canonical examples:

- `GhostFTP-Windows-x64-Portable-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.exe`
- `GhostFTP-Linux-x86_64-v<version>.AppImage`
- `GhostFTP-Android-v<version>.apk`
- `GhostFTP-v<version>-Desktop-Source.zip`

For future releases substitute the canonical `0.x`/later semantic version.

## Branded top-level source paths

- `ghostftp-desktop/`
- `android/`
- `tools/ghostftp-runtime/`
- `tools/ghostftp-installer/`
- `website/`
- `updates/`

## Framework-required names that remain

Internal names consumed by Tauri/ecosystem remain unchanged:

- `src-tauri/`
- `tauri.conf.json`
- `@tauri-apps/*`
- `tauri-apps/tauri-action`

## Future naming rules

Prefer:

- `ghostftp-<purpose>.<ext>`
- `GhostFTP-<Platform>-<Arch>-<Role>-v<Version>.<ext>`

Avoid:

- generic temporary/final names;
- framework-first public filenames;
- versionless release executables;
- legacy release-candidate identifiers in new product assets.

## Stability rule

Do not rename persisted identifiers, bundle identifiers, updater identifiers or protocol/deep-link schemes without a migration plan.

Current migration-sensitive identifiers include:

- desktop bundle identifier: `com.ghostftp.desktop`
- deep-link scheme: `ghostftp://`
- public domain: `ghostftp.com`
