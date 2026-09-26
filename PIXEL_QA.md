# Pixel-Perfect QA

The ten Ghost FTP reference images under `docs/screenshots/` remain the visual specification and are not used as runtime backgrounds. Canonical desktop geometry remains centered on a 1290×852 application frame with the custom titlebar, menu, Quick Connect band, toolbar, sites rail, dual file panes, transfer/log band and status bar represented by real controls.

This pass preserved the canonical large-window rules while adding small-window overrides instead of scaling the whole screenshot. No claim of pixel-perfect completion is made without rendering the actual native app and comparing each target screen to its corresponding reference.

Before FINAL, capture native renders for Main, Site Manager, New Connection, Preferences, Transfer Center, File Properties and About at 100% scale; compare window bounds, row heights, dividers, radii, typography, icon offsets, focus/hover/selected states and modal geometry; iterate on measured deltas.

## RC7 measured parity pass

RC7 added a deterministic component-render harness and compared the real runtime surfaces at the canonical 1290×852 viewport. The main shell now follows the reference segmentation directly: 51 px custom titlebar, 42 px menu, 50 px Quick Connect row, 62 px toolbar, 216 px Sites rail, 416 px file workspace, 191 px transfer/log band and 40 px status bar. The New Connection surface is now measured at 752×628 px, matching the approved reference envelope. Site Manager, Preferences, Transfer Center and About render as full-window surfaces rather than generic web modals.

The render harness confirms geometry and responsive containment for the compatibility UI. Final pixel acceptance of the native Tauri host, including OS font metrics and native window composition, remains a separate release gate.
