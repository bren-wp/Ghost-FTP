# Ghost FTP Native Titlebar QA — RC10

## Required product behavior

Ghost FTP uses its own custom application titlebar. The desktop window is configured frameless with native decorations disabled, and the React titlebar provides:

- Ghost FTP branding;
- minimize;
- maximize/restore;
- close;
- draggable title region;
- double-click maximize/restore.

The old browser-host compatibility build that exposed a visible `127.0.0.1` address/origin strip is not part of the production release path.

## Acceptance criteria

For both the portable and installed Windows builds:

- no browser/origin/address bar;
- no duplicate black/native caption above the Ghost FTP titlebar;
- minimize works;
- maximize/restore works;
- close works;
- drag works;
- double-click titlebar maximize/restore works;
- restored 1290×852 geometry matches the approved reference composition;
- maximized layout preserves clean custom chrome.

## Remaining FINAL evidence

Native Windows 10/11 screenshots of the actual RC10 executable are still required before a pixel-perfect/FINAL claim is made.

Status: **source/build architecture is correct; Windows screenshot acceptance remains open.**
