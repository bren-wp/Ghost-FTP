# Ghost FTP Responsive QA — RC3

The reference desktop surface remains 1290×852. Native CSS contains compact/adaptive rules for smaller windows including 900, 760, 640, 540 and 480 px breakpoints. Menus and the Quick Connect band use contained scrolling where necessary; toolbar actions remain available; sidebars compress; dialogs are viewport constrained and scrollable rather than overflowing the whole app.

RC3 also adds responsive rules for the split Quick Connect protocol control while preserving the 162 px reference width at the canonical desktop size.

Required final render targets remain 1290×852, 1280×800, 1024×768, 960×720, 800×600, 768×720, 640×720, 540×720 and 480×800. Chromium policy in this sandbox blocks reliable local UI screenshot automation, so those renders are not falsely marked passed.

## RC4 source review — 20 September 2026

The responsive source contract remains 1290×852 as the canonical desktop reference with adaptive behavior down to a 480×600 native minimum window. Small-window rules use local scrolling/compaction rather than hiding critical controls. Automated screenshot proof remains blocked in this sandbox because the installed Chromium policy blocks `127.0.0.1`; therefore visual acceptance at every requested viewport is not falsely marked as passed.

## RC9 executable compatibility-render evidence — 20 September 2026

The component runtime was rendered with Chromium/Playwright from the real HTML/CSS/JS controls, with API calls intercepted only for deterministic QA data. The reference image was **not** used as a background.

Required viewports exercised: 1290×852, 1280×800, 1024×768, 960×720, 800×600, 768×720, 640×720, 540×720 and 480×800.

For all nine sizes:
- document `scrollWidth` stayed equal to the physical viewport width (no whole-window horizontal overflow);
- the New Connection dialog stayed fully inside the viewport;
- no JavaScript page errors were observed in the render harness;
- at 1290×852 the New Connection dialog measured exactly 752×628 px;
- at constrained heights the dialog reduces vertically and becomes internally scrollable instead of escaping the viewport.

Machine-readable measurements are retained at `qa/rc6/responsive/metrics.json`. These are compatibility-render results; native Tauri Windows/Linux acceptance still requires a native build host.
