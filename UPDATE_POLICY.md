# Ghost FTP Update Policy

The native Ghost FTP updater is configured to query **https://ghostftp.com/updates/latest.json**. Release metadata should include the version/build, platform, architecture, package URL, SHA-256, Tauri-compatible signature, release notes and minimum supported version.

The intended flow is: check metadata, compare versions, download the platform-specific package, verify integrity/signature, stage, apply, restart, and retain a rollback/recovery path if the apply phase fails. Unsigned or invalidly signed native packages must not be installed by the updater.

The source uses `@tauri-apps/plugin-updater` for signed package verification and separates manual update checks from background/quiet checks. Update UI must not claim that the current build is up to date until an actual check has completed successfully.

The Go compatibility fallback does not install application updates; its About view routes the user to the official download area and states that update installation belongs to the native build.
