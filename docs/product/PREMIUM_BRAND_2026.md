# Ghost FTP premium UI — implementation and acceptance contract

Canonical mark: exact `assets/branding/ghostftp-app-icon.svg` geometry and corresponding React/Android vector implementations.

Source: user-provided `GhostFTP-Premium-Brand-UI(1).zip` (8 October 2026).

## Reference contract

- Screens are **visual concepts**, not real product screenshots or operational examples.
- 30 Windows and 29 Linux mockups share a 1290 × 852 desktop frame and layout.
- 16 Android mockups use a 480 × 960 reference frame with compact navigation rail.
- No macOS mockups were provided; macOS adopts the common palette, spacing hierarchy, brand icon and component semantics with platform-correct SwiftUI controls.
- Mockup data, sample server names, metrics and fake transfer rows must never be seeded in the product.
- Never display screenshot PNGs as clickable UI backgrounds or infer backend support from a pictured button.

## Canonical premium tokens

| Purpose | Hex |
| --- | --- |
| Canvas | `#070E1A` |
| Panel | `#0D192B` |
| Raised panel | `#111F34` |
| Navy | `#0B1E36` |
| Slate blue | `#132D52` |
| Action/accent | `#38ABFF` |
| Accent highlight | `#7EDEFF` |
| Primary text | `#EAF6FF` |
| Success | `#26DC8F` |
| Error | `#FF7284` |

Inter, Segoe UI and Noto Sans are preferred with platform system fallbacks. No font files are bundled.

## Real implementation

- Windows/Linux: existing React/Tauri components for navigation, file panes, transfers, dialogs and settings; shared dark semantic tokens and GhostMark. Linux preserves native window-manager differences.
  - Quick tools from the approved reference sidebar are connected to the existing, functional SFTP Terminal, command palette and local duplicate scanner. Terminal is disabled without an active SFTP session.
- Android: existing native Kotlin UI with shared semantic colors and vector launcher mark. Standard-width phones now dock the 88 dp rail rather than covering the file workspace with a menu; devices under 360 dp retain an accessible overlay. Touch-target and accessibility checks stay mandatory.
- macOS: existing SwiftUI workspace shell with shared palette/tint. Light appearance remains independently supported.
- Secrets, permission prompts and security warnings must retain the existing platform policies.

## Release gating / not yet completed

- Automated builds and functional protocol gates **do not** prove pixel-by-pixel parity; acceptance requires screenshot comparison at reference sizes on each platform.
- All 75 screens must be mapped to real working components before claiming 1:1 visual/functional completion. Current progress is design-token/logo alignment and the mobile navigation geometry, not a complete screenshot-by-screenshot acceptance.
- macOS FTPS/SFTP file operations remain blocked until trusted TLS certificate/hostname and SSH host-key verification are implemented. macOS Preview is not notarized.
- Future screen-level refinements must be scoped to real components and ship only after exact-head CI and package validation.
