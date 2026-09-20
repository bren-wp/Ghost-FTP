# Ghost FTP Pixel-Parity QA — RC12

The approved images under `docs/assets/screenshots/` are the visual specification. They are documentation/QA references only and are never loaded as application backgrounds or used as click maps.

## Canonical main-window geometry

At the approved **1290×852** desktop frame the real Ghost FTP component shell targets:

- 52 px integrated custom titlebar + application menubar;
- 72 px Quick Connect row;
- 60 px toolbar;
- 216 px Sites rail;
- 416 px file workspace;
- 210 px transfer/log band;
- 40 px status bar;
- thin separators/borders consume the remaining pixels of the 852 px native inner frame.

Reference modal envelopes:

- New Connection: **640×496** at the canonical desktop frame
- File Properties: **490×716** at the canonical desktop frame

## Visual system

Core Ghost FTP colors:

- Electric Blue: `#38ABFF`
- Deep Navy: `#0B1E36`
- Slate Blue: `#132D52`
- Ice White: `#EAF6FF`

The UI should use dark navy hierarchy, restrained Electric Blue emphasis, readable Ice White text, thin blue borders and selective glow.

## Reference screens

Current branded reference files cover:

- Main File Manager;
- Site Manager;
- New Connection;
- Preferences;
- Transfer Center;
- File Properties;
- About;
- Windows/Linux platform presentation;
- brand identity.

## Acceptance rule

RC12 does **not** claim pixel-perfect FINAL acceptance yet.

Before FINAL, capture real native Windows renders at 100% display scale and compare:

- window bounds;
- typography;
- row heights;
- dividers;
- radii;
- icon placement;
- selected/hover/focus states;
- modal dimensions;
- custom titlebar;
- spacing and alignment.

Measured differences must be accepted or corrected before the release is described as 1:1 / pixel-perfect FINAL.


## Stacking acceptance

At every documented viewport size:

- opening New Connection, Site Manager, Preferences, Transfer Center or About must fully cover the underlying application chrome;
- no titlebar, Quick Connect row or toolbar control may paint above a modal/standalone surface;
- top application menus must float over the Quick Connect row without changing row height, pushing content or clipping;
- screenshots used for comparison are specifications only; production UI must remain real React/Tauri controls.


## RC12 compositor hardening

Menu and protocol popovers use explicit high stacking contexts and overflow-visible ancestors. Critical standalone/dialog surfaces do not use transform entrance animations, and press feedback no longer scales controls. These are source-level flicker mitigations; real Windows/Linux soak and screenshot comparison remain required before pixel-perfect acceptance.


## Measured RC13 reference envelopes

The current supplied 1672×941 artwork was measured directly rather than relying on the older nominal dialog dimensions. The New Connection border is approximately 639×496 px inside an approximately 1291 px wide application frame; File Properties is approximately 489×716 px. RC13 therefore targets 640×496 and 490×716 respectively at canonical desktop scale, with viewport-safe internal scrolling below that size.
