# Ghost FTP Pixel-Parity QA — RC9

The approved images under `docs/assets/screenshots/` are the visual specification. They are documentation/QA references only and are not loaded as application backgrounds or used as click maps.

## Canonical main-window geometry

At the approved 1290×852 desktop frame the real Ghost FTP component shell uses:

- 51 px custom titlebar;
- 42 px application menu;
- 50 px Quick Connect row;
- 62 px toolbar;
- 216 px Sites rail;
- 416 px file workspace;
- 191 px transfer/log band;
- 40 px status bar.

These dimensions sum to the canonical 852 px application height. New Connection targets 752×628 px and File Properties targets 530×770 px while remaining viewport-constrained on smaller windows.

## Visual system

The default dark surface uses the Ghost FTP navy hierarchy with the approved Electric Blue accent and Ice White primary text. RC9 removes a redundant older dark-token block so only the canonical Ghost FTP visual system controls the default theme.

## Acceptance rule

RC9 does not claim complete pixel-perfect acceptance yet. Before FINAL, capture native Windows renders for Main, Site Manager, New Connection, Preferences, Transfer Center, File Properties and About at 100% display scale and compare bounds, typography, icon placement, dividers, radii, selected/hover/focus states and custom-window chrome against the matching reference images.
