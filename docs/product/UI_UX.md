# Ghost FTP — UI / UX Contract

The Ghost FTP UI is a real component-based desktop interface. The approved reference images are acceptance targets only; they must never be used as a full-screen application background or hotspot map.

## Canonical desktop frame

Primary reference frame:

- Window: **1290×852**
- Custom titlebar: **51 px**
- Application menu: **42 px**
- Quick Connect row: **50 px**
- Toolbar: **62 px**
- Sites rail: **216 px**
- File workspace: **416 px**
- Transfer/log band: **191 px**
- Status bar: **40 px**

Reference modal envelopes:

- New Connection: **752×628**
- File Properties: **530×770**

## Visual language

Approved Ghost FTP core colors:

- Electric Blue: `#38ABFF`
- Deep Navy: `#0B1E36`
- Slate Blue: `#132D52`
- Ice White: `#EAF6FF`

The default experience should use dark navy surfaces, restrained electric-blue emphasis, thin blue borders, readable ice-white text and selective glow. Glow is an accent, not a substitute for hierarchy.

## Window chrome

Requirements:

- Only the Ghost FTP custom titlebar is visible in the desktop product.
- No browser/origin/address strip.
- No duplicate OS/Chromium title strip.
- Minimize, maximize/restore and close remain real native window actions.
- The draggable title region must not overlap interactive controls.
- Double-click titlebar maximize/restore must remain functional.

## Responsive behavior

The 1290×852 reference remains canonical. Smaller windows adapt; the application is not globally scaled like an image.

Preferred order of adaptation:

1. Reduce spacing and label width.
2. Compress sidebars.
3. Allow local horizontal scrolling for dense tool/menu rows.
4. Allow internal modal scrolling.
5. Preserve primary actions.
6. Never hide critical connection, transfer or window controls merely to make the layout fit.

Minimum supported native window target remains 480×600, with 480×800 used in the visual QA set.

## Interaction principles

- Every visible action must be a real control.
- Disabled controls must visibly look disabled.
- Destructive actions need deliberate affordances.
- Completed transfers must not expose active-only actions such as Cancel.
- Failure states should present recovery (Retry) instead of dead-end status.
- Keyboard focus must remain visible.
- Controls should have accessible names.
- `prefers-reduced-motion` should reduce non-essential motion.
- Dynamic transfer progress uses semantic progress controls.

## Screen-specific expectations

### Main File Manager

- Sites rail at left.
- Dual file panes.
- Clear local/remote context.
- Toolbar actions map to real operations.
- Bottom transfer/log band remains visually subordinate to file work.

### Site Manager

- Full application surface, not a generic web modal.
- Search/list/details composition.
- Favorites/tags/folders remain easy to scan.

### New Connection

- Protocol, host, port and credentials are grouped clearly.
- Primary Connect action is visually dominant.
- Save Profile is secondary.
- Private-key mode must not make password/key state ambiguous.

### Preferences

- Category navigation at left.
- Settings cards grouped by purpose.
- Cancel restores the original snapshot.
- Apply commits the current state.
- Locked privacy/security statements must look informational, not broken toggles.

### Transfer Center

- Clear direction/status/progress.
- Failed transfers have retry actions.
- Completed transfers have no active-only controls.
- Filters and queue actions remain discoverable.

### File Properties

- General and Checksums are separate tabs.
- Permission matrix and numeric chmod agree.
- Unsupported backend fields remain blank/explicit instead of fabricated.

## Acceptance rule

A release is not described as pixel-perfect/FINAL until the real native Windows/Linux render is compared against the approved references at the documented sizes and the measured differences are accepted.


## RC11 production-fidelity acceptance

- The supplied Ghost FTP reference screenshots are treated as visual acceptance specifications, never as runtime backgrounds.
- File Manager chrome is isolated from standalone Site Manager, Preferences, Transfer Center and About surfaces.
- Menu dropdowns are overlay layers and must never push Quick Connect or toolbars out of position.
- New Connection sizes to real content with a 752 px reference width and viewport-safe maximum height; dead vertical filler regions are rejected.
- The 1290×852 reference geometry is primary while 1024×768 through 480×800 use component-local scrolling and compaction rather than whole-window overflow.
- No production UI may seed example.com, Production Server, Staging Server or other demo records.
