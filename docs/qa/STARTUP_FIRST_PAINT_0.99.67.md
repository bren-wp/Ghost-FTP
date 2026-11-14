# Ghost FTP 0.99.67 — first-paint window startup acceptance

**Status: source fix committed, installed Windows/Linux visual acceptance NOT YET VERIFIED.**

## Root cause and implementation

- Tauri created the main WebView window visible before React and CSS were mounted, allowing the host WebView's temporary white surface and an unstyled dark/black frame to appear on launch.
- The single main window now starts `visible(false)` from its Rust window builder, using the same native settings/theme initialization as before.
- The React entry point reveals the window via `getCurrentWindow().show()` **after** the application shell mounts, with an explicit `core:window:allow-show` permission scoped to `main`. Non-Tauri browser previews do not call the native window API.
- The startup HTML includes a tiny synchronous dark/light fallback so the first document frame is never the browser's white default. This is not a substitute for the full stylesheet, nor a new splash screen.
- A hidden WebView can suppress `requestAnimationFrame`, so startup visibility does not depend on animation frames. React StrictMode cleanup avoids duplicate startup reveal timers.
- `npm run check:ui` now includes regression tests that fail when the window is made visible before first mount, the reveal command or permission disappears, or startup backgrounds regress.

## Acceptance checklist

Test actual **Windows 10/11 WebView2** and **Linux WebKitGTK** builds on both fast and slow hardware, with GPU acceleration enabled and disabled where supported.

1. Cold launch, warm launch and reopen after OS sleep, 20 repetitions each; capture the first 1–2 seconds at 60+ FPS. No blank white/black frame should be visible before the real app UI.
2. Repeat with saved Dark, Light, Ghost Ice and three other named dark themes. The first frame must use the chosen theme with no light-to-dark or dark-to-light flash.
3. Test a fresh profile, an upgraded profile, an empty settings DB, intentionally malformed saved appTheme (safe default), and first-launch offline conditions.
4. Reload after deliberately inducing a React render error and verify the error boundary appears rather than an indefinitely hidden window.
5. Test single-instance relaunch and `ghostftp://` activation when the first launch is still hidden, including window focus, close, minimize, maximize and dragging.
6. Run with screen-reader and reduced-motion settings, high-DPI/multiple monitors, and capture any OS compositor-specific flashes.
7. Confirm the single main-window restriction, no additional broad Tauri capabilities, no needless privacy-sensitive storage, and no exception details in startup logging.

**Release gate:** Exact-head Windows/Linux native builds plus manual first-frame video evidence are required to certify that the visible flash is eliminated on tested systems. Do not claim 100% reproducibility or flawless startup from source-only tests.
