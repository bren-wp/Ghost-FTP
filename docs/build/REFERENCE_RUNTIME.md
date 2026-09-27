# Ghost FTP support/runtime tooling

The Go projects under `tools/ghostftp-runtime/` and `tools/ghostftp-installer/` are support/compatibility tooling.

They are **not** the authoritative Ghost FTP Windows/Linux production GUI and must not be documented or distributed as native Tauri desktop builds.

## Production authority

The production desktop implementation is `ghostftp-desktop/`. It contains the native React/TypeScript interface, Tauri host, Rust protocol/session/transfer engine, credential handling and production packaging configuration.

The production Android implementation is `android/`.

## Support-tool contract

The Go tools may be used for compatibility, local development or installer/runtime support scenarios. They must not:

- simulate successful FTP/FTPS/SFTP sessions as if they were production protocol results;
- replace the native desktop release path;
- be described as Tauri builds;
- introduce user-visible shell/console flashes for background Windows helper processes;
- become a second independent version source.

Any network reachability behavior in support tooling is distinct from the real native protocol engine.
