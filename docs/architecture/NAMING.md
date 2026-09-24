# Ghost FTP Repository Naming Policy

Ghost FTP uses a simple naming rule so the repository stays branded without destabilizing framework-required build internals.

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

Examples:

- `GhostFTP-Windows-x64-Portable-v2.1.1-RC21.exe`
- `GhostFTP-Windows-x64-Setup-v2.1.1-RC21.exe`
- `GhostFTP-Linux-x86_64-v2.1.1-RC21.AppImage`
- `GhostFTP-v2.1.1-RC21-Desktop-Source.zip`

## Branded top-level source paths

Current repository source roots are intentionally branded:

- `ghostftp-desktop/`
- `tools/ghostftp-runtime/`
- `tools/ghostftp-installer/`
- `website/`
- `updates/`

These names replace generic or framework-first top-level paths and make the repository easier to understand at a glance.

## Framework-required names that remain

A small number of internal names remain because they are consumed directly by the desktop framework or its ecosystem:

- `src-tauri/`
- `tauri.conf.json`
- `@tauri-apps/*`
- `tauri-apps/tauri-action`

They are implementation details, not Ghost FTP branding. Renaming them only for appearance would add build and maintenance risk.

## Naming rules for future files

Prefer:

- `ghostftp-<purpose>.<ext>`
- `GhostFTP-<Platform>-<Arch>-<Role>-v<Version>.<ext>`

Avoid:

- generic names such as `app-final-new2.zip`;
- framework-first public filenames;
- versionless release executables;
- filenames that do not communicate platform or architecture.

## Stability rule

Do not rename persisted identifiers, bundle identifiers, updater identifiers or protocol/deep-link schemes unless there is a migration plan.

Current stable identifiers include:

- bundle identifier: `com.ghostftp.desktop`
- deep-link scheme: `ghostftp://`
- public domain: `ghostftp.com`
