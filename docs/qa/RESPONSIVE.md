# Ghost FTP Responsive QA — RC9

The canonical desktop target remains 1290×852. Responsive behavior is adaptive rather than screenshot scaling: controls compact, sidebars narrow and dense horizontal regions use contained local scrolling instead of hiding essential actions.

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

## Source contract

Native CSS includes adaptive breakpoints around 1180, 1040, 980, 900, 760, 640, 540 and 480 px. The Tauri minimum window remains 480×600. Quick Connect, application menus and toolbars use contained scrolling at constrained widths; dialogs stay within the viewport and become internally scrollable where required.

## RC9 acceptance status

Source geometry and containment rules are in place. FINAL still requires native Windows/Linux render evidence at the required viewport sizes, with particular attention to text clipping, menu reachability, dialog containment, dual-pane usability and titlebar controls.
