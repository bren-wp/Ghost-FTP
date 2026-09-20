# Ghost FTP Titlebar QA — RC10

The production desktop source creates the application as a frameless native WebView window and renders Ghost FTP's custom titlebar as the visible application chrome.

## Implemented

- custom Ghost FTP branding;
- minimize;
- maximize/restore;
- close;
- drag region;
- titlebar double-click maximize/restore path;
- canonical 51 px titlebar at the 1290×852 reference size.

Browser-host compatibility executables are not accepted as production GUI releases because visible browser/origin chrome breaks the approved design.

## Required FINAL evidence

Capture Windows 10 and Windows 11 production builds and confirm:

1. no `127.0.0.1` origin bar;
2. no Chromium/Edge application bar;
3. no second black native caption strip;
4. Ghost FTP controls remain clickable after maximize/restore;
5. dragging works across the intended titlebar area;
6. DPI scaling does not introduce an extra caption or clipping.

Until those screenshots are recorded, titlebar acceptance remains pending even when the native build compiles successfully.
