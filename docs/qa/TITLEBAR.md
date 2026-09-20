# Ghost FTP Native Titlebar QA — RC9

The production Tauri window is created frameless with `decorations(false)`. The React titlebar is therefore the intended and only application titlebar. It contains Ghost FTP branding plus minimize, maximize/restore and close controls, a Tauri drag region and double-click maximize/restore behavior.

The visible `127.0.0.1` Chromium/Edge bar reproduced in older compatibility executables was an architecture error in the fallback host, not the intended production UI. Those browser-host executables are excluded from the end-user release path.

## FINAL acceptance gate

Capture native Windows 10/11 screenshots of both portable and installed RC9 builds and verify:

- no browser/origin/address bar;
- no extra black native caption above the Ghost FTP titlebar;
- minimize, maximize/restore, close and drag behavior work;
- maximized and restored states preserve the custom chrome cleanly;
- 1290×852 restored geometry matches the approved reference composition.

This gate remains open until those target-OS screenshots are attached as evidence.
