# Ghost FTP Pixel-Parity QA — RC10

The approved Ghost FTP images in `docs/assets/screenshots/` are the visual specification. They are not runtime backgrounds or click maps.

## Canonical desktop target

The main desktop frame is **1290 × 852**.

Measured target segmentation:

- 51 px custom titlebar
- 42 px application menu
- 50 px Quick Connect row
- 62 px toolbar
- 216 px Sites rail
- 416 px file workspace
- 191 px transfer/log band
- 40 px status bar

Reference dialogs:

- New Connection: **752 × 628**
- File Properties: **530 × 770**

RC10 makes both dialog heights explicit at the canonical viewport and removes later duplicate width overrides so one canonical rule controls their primary geometry.

## Secondary application surfaces

Site Manager, Preferences, Transfer Center and About are full Ghost FTP application surfaces rather than generic centered browser dialogs. Their chrome, dividers, panel treatment, spacing and color system are maintained as real components.

## Color contract

- Electric Blue: `#38ABFF`
- Deep Navy: `#0B1E36`
- Slate Blue: `#132D52`
- Ice White: `#EAF6FF`

## FINAL acceptance still required

A source-level geometry match is not enough to claim exact pixel parity. Before FINAL, capture the real native Windows build at 100% scaling and compare:

- window bounds;
- typography/font metrics;
- icon size and offsets;
- row heights and divider positions;
- radii and shadows;
- selected/hover/focus states;
- menu/dropdown positioning;
- all seven primary reference screens.

Any measured deltas should be corrected before the build is labelled FINAL.
