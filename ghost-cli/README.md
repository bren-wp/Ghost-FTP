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

Builds require a Rust toolchain, C compiler and the native dependencies already documented for the desktop runtime. Every release builds independent Windows **Portable EXE and unsigned NSIS Setup EXE**, plus Linux **executable, tar.gz, DEB and RPM** from the same source SHA. Ghost CLI remains a standalone console product, even if installed alongside Ghost FTP Client. Windows Setup has current-user install and EULA acceptance; Linux packages do not automatically run the CLI or change shell profiles.

## Portable encrypted backup — explicit cross-device handoff

The **existing** Ghost CLI also implements `backup export <path>` and `backup import <path>`. These call the same encrypted backup implementation as the native app. Export includes profiles, encrypted credentials and application state; importing **replaces or stages** existing local state, so first preserve a separate recovery backup, close the app if possible and verify the destination and trusted file. Transfer the encrypted backup file through a secure channel to another supported device, and restore it intentionally there.

Commands (do not paste real credentials into public terminals or scripts):
```sh
ghostftp-cli backup export ./ghostftp-portable.ghostftpbak
ghostftp-cli backup import ./ghostftp-portable.ghostftpbak
```

The current CLI uses `GHOSTFTP_BACKUP_PASSWORD` or `--password` for the backup encryption secret. **Avoid `--password`** because process arguments can be inspected or captured in shell history. Prefer an appropriately protected secret provider to populate the environment only for the intended process, and clear it afterward; environment variables can also be observable to privileged local processes. Ensure the password is strong, unique and not stored alongside the archive. This is **manual, user-approved encrypted portability**, **not** invisible automatic synchronization, a cloud account, or a claim of Android/macOS CLI package availability. Runtime and database compatibility must be verified by target-platform restoration tests before production reliance.

## Safety and synchronization

Real network connections and transfer operations **only** run when explicitly invoked against an authorized profile. `--dry-run` must be used when previewing synchronization. Secrets should remain in the OS-backed credential store; do not put passwords or private keys on the command line. Profile handling shares the desktop's local store on the same OS user account; it does **not** imply automatic cloud synchronization of credentials between separate devices. Dangerous mutations remain operator-initiated and subject to backend authorization. The separately distributed Ghost FTP Server is a daemon; Ghost CLI is a client, never a replacement for server administration or the GUI.
