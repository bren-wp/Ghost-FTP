# Ghost FTP compatibility runtime

The `runtime/` and `installer/` Go/Chromium hosts are **compatibility/fallback builds only**. They exist so local-file controls, layout, installer flow and Windows frameless-window behavior can be exercised in an environment that cannot compile the Rust/Tauri application.

They are not the native Ghost FTP protocol-engine release and must never be described as Tauri builds. Fake FTP/FTPS/SFTP success and simulated transfer progress are intentionally disabled. Network testing in this runtime is TCP reachability only.

The production implementation is `desktop-tauri/`, which contains the real FTP/FTPS/SFTP/session/transfer engine and frameless Tauri host.
