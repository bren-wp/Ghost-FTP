# Titlebar QA

Native Tauri source constructs the main window with `decorations(false)` and a 1290×852 initial size. The React titlebar contains Ghost FTP branding plus minimize, maximize/restore and close controls, uses a Tauri drag region, and now explicitly handles titlebar double-click through the same maximize/restore path.

The Windows Go compatibility host also strips `WS_CAPTION`, retains the resize frame and implements minimize/maximize/restore/close plus Win32 `WM_NCLBUTTONDOWN/HTCAPTION` dragging. This is fallback hardening only; it is not evidence that Edge/Chromium will never show an extra bar on every Windows configuration.

The required acceptance screenshot — showing **only** the Ghost FTP custom titlebar and no Chromium/Edge/native black caption — cannot be captured in this Linux sandbox. Windows Setup, Windows Portable and native Linux titlebar checks remain **BLOCKED**, so this release is not FINAL.
