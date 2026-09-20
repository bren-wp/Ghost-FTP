# Ghost FTP Responsive QA — RC10

The 1290×852 desktop reference remains the exact large-window target. Smaller windows adapt rather than scaling a screenshot or removing essential actions.

## Required viewport set

- 1290×852
- 1280×800
- 1024×768
- 960×720
- 800×600
- 768×720
- 640×720
- 540×720
- 480×800

## RC10 source behavior

- menu and Quick Connect rows can use local horizontal scrolling where necessary;
- toolbar actions remain available;
- Sites rail compresses at smaller widths;
- dialogs stay viewport-constrained and internally scrollable;
- New Connection retains a 752×628 target envelope at canonical size;
- File Properties retains a 530×770 target envelope at canonical size;
- full-window secondary surfaces compress navigation widths before dropping content;
- reduced-motion preferences are honored.

## Still required before FINAL

The real native Windows/Linux build must be captured at every required viewport and reviewed for:

- whole-window horizontal overflow;
- clipped text;
- inaccessible buttons;
- off-screen dialogs;
- focus visibility;
- menu/dropdown placement;
- long translations;
- 125%, 150% and 200% Windows display scaling.

Source rules are implemented, but final target-OS render evidence remains a release gate.
