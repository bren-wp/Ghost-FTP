# Reference UI

Ghost FTP uses supplied Windows, Linux and Android master references to define the visual target for the maintained applications.

## Global design language

- dark premium surfaces;
- Ghost Gold accent;
- clear green connected/completed state;
- clear red failed state;
- rounded cards and buttons;
- consistent local icons;
- readable labels without clipping;
- no developer-only placeholder content.

## Desktop master hierarchy

Windows and Linux use:

1. Ghost FTP identity and four-item navigation rail;
2. connection / Quick Connect row;
3. Back, Forward, Refresh, New Folder, Upload, Download, Bookmarks, More;
4. Local Files + Remote Files;
5. Transfer Queue;
6. bottom status bar.

The normal Files workspace avoids duplicate permanent technical rows. Secondary engine-backed operations belong in More or their dedicated modal surface.

## Android hierarchy

Android uses:

1. app header + connection state;
2. Production Server card;
3. two-row action grid;
4. Local Files + Remote Files;
5. Transfer Queue;
6. bottom navigation.

Labels and touch targets must remain legible on supported phone widths.

## Runtime evidence

Checked-in 0.0.8 screenshots are historical exact-head evidence for that release. The 0.0.9 development line requires new Windows / Linux / Android authentic runtime screenshots before publication.

Reference images guide implementation, but they are not runtime evidence and must never replace real captures from the application.

## Retired platform

macOS is not part of the maintained reference UI contract.

