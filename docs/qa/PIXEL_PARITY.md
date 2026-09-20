# Ghost FTP Pixel-Parity QA — RC11

The approved images under `docs/assets/screenshots/` are the visual specification. They are documentation/QA references only and are never loaded as application backgrounds or used as click maps.

## Canonical main-window geometry

At the approved **1290×852** desktop frame the real Ghost FTP component shell targets:

- 51 px custom titlebar;
- 42 px application menu;
- 50 px Quick Connect row;
- 62 px toolbar;
- 216 px Sites rail;
- 416 px file workspace;
- 191 px transfer/log band;
- 40 px status bar.

Reference modal envelopes:

- New Connection: **752×628**
- File Properties: **530×770**

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

RC11 does **not** claim pixel-perfect FINAL acceptance yet.

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
