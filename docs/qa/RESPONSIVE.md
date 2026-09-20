# Ghost FTP Responsive QA — RC9

The canonical desktop target remains **1290×852**. Ghost FTP adapts controls rather than globally scaling the reference composition like an image.

## Required viewport targets

- 1290×852
- 1280×800
- 1024×768
- 960×720
- 800×600
- 768×720
- 640×720
- 540×720
- 480×800

## Responsive contract

The desktop CSS includes adaptive breakpoints around 1180, 1040, 980, 900, 760, 640, 540 and 480 px.

At constrained sizes Ghost FTP should:

1. reduce non-essential spacing;
2. narrow supporting rails;
3. keep key connection/transfer actions visible;
4. use contained horizontal scrolling for dense tool rows;
5. constrain dialogs to the viewport;
6. use internal dialog scrolling when required;
7. avoid whole-window horizontal overflow.

The native minimum-window target remains 480×600.

## Screen priorities

### Main File Manager

The file panes remain the primary workspace. Supporting navigation and transfer/log panels compress before critical file actions are hidden.

### New Connection

Protocol, host, port, authentication and Connect remain reachable. The 752×628 reference envelope becomes viewport-constrained on smaller windows.

### File Properties

The 530×770 reference envelope becomes internally scrollable when vertical space is insufficient.

### Preferences / Transfer Center / Site Manager

These are full application surfaces and must remain navigable without clipped primary actions.

## RC9 acceptance status

Source geometry and containment rules are implemented. FINAL still requires native Windows/Linux render evidence for every target size, including text clipping, toolbar reachability, dialog containment, dual-pane usability and custom titlebar controls.
