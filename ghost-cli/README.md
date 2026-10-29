# Ghost CLI — native Ghost FTP command-line client

**Official entry point:** `/ghost-cli`. The CLI implementation is the **existing, shared Rust session engine** in `ghostftp-desktop/src-tauri/ghostftp-cli`. The packaging scripts in this directory build that *same* crate and avoid a parallel FTP/SFTP engine with diverging authentication and transfer behavior.

## Real commands

`ghostftp-cli --help` shows the supported commands and exact options. Currently: `ls`, `cp`, `mv`, `rm`, `mkdir`, `sync`, `diff`, `dedupe`, `search` and more. Target syntax is an explicitly configured saved profile such as `my-server:/some/path` or a local file path. FTP, explicit FTPS and SFTP use the same profiles and backends as the desktop client; cloud provider features depend on a valid configured account and upstream permissions.

Examples (no real credentials or hosts embedded):
```sh
ghostftp-cli --help
ghostftp-cli ls my-server:/
ghostftp-cli cp ./report.txt my-server:/incoming/report.txt
ghostftp-cli cp my-server:/incoming/report.txt ./report.txt
ghostftp-cli sync ./reports my-server:/reports --dry-run
```

## Build

Linux:
```sh
bash ghost-cli/build.sh
./ghostftp-desktop/src-tauri/target/release/ghostftp-cli --version
```

Windows PowerShell:
```powershell
./ghost-cli/build.ps1
./ghostftp-desktop/src-tauri/target/release/ghostftp-cli.exe --version
```

Builds require a Rust toolchain, C compiler and the native dependencies already documented for the desktop runtime. The release workflow produces and checks independent `GhostCLI-Windows-x64-v<version>.exe` and `GhostCLI-Linux-x86_64-v<version>` artifacts from the same source SHA.

## Safety and synchronization

Real network connections and transfer operations **only** run when explicitly invoked against an authorized profile. `--dry-run` must be used when previewing synchronization. Secrets should remain in the OS-backed credential store; do not put passwords or private keys on the command line. Profile handling shares the desktop's local store on the same OS user account; it does **not** imply automatic cloud synchronization of credentials between separate devices. Dangerous mutations remain operator-initiated and subject to backend authorization. The separately distributed Ghost FTP Server is a daemon; Ghost CLI is a client, never a replacement for server administration or the GUI.
