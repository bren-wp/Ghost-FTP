# Ghost FTP macOS development app

The `macos/` tree is the dedicated native macOS development surface for Ghost FTP. It is deliberately separate from the Windows and Linux packaging trees, but it is not a separate product or a reduced feature edition.

**Windows desktop is the canonical visual and behavior reference.** The target is 1:1 feature parity in the sense that every supported Windows desktop action has an equivalent, real macOS action backed by the same product state and security rules. Native platform primitives may differ where macOS requires them, but capability, labels, ordering, validation, enabled/disabled state and workflow semantics must remain aligned.

The completed macOS client will use the same typed `internal/api.Engine` as the maintained Windows/Linux desktop clients. UI work must therefore follow one non-negotiable rule: **no decorative or dead controls**. A control is added to the visible Mac UI only when the corresponding engine-backed operation is functional and testable.

## Visual contract

The macOS frontend follows the maintained Windows layout hierarchy: branded header, profile/application actions, Quick Connect, Local and Remote panes, transfer actions, transfer queue, status and version surfaces. The native AppKit window may use macOS window chrome, focus behavior and accessibility APIs, while the Ghost FTP workspace remains visually aligned with Windows.

The local source palettes are identical to the desktop reference:

- Classic Light: workspace `#EEF1F5`, panel `#F6F8FB`, list `#FAFBFD`.
- Dark: workspace `#0B0F17`, panel `#121824`, list `#161D2A`.

The finished Mac client must expose the same **24 languages**, with English as canonical default/fallback, and the same FTP, explicit FTPS and SFTP protocol semantics as the maintained desktop product.

## Privacy and dependency boundary

The Mac app has no telemetry, analytics, advertising, tracking pixels, remote fonts, remote styles or mandatory Ghost FTP account. The build is source-local and must not download runtime UI or application code. macOS credential/keychain integration will be added only behind the same explicit credential-persistence consent and account-identity binding used by the shared desktop model.

## Current stage

This is an active **development surface**. The first maintained stage establishes a native AppKit application bundle, universal Intel/Apple-Silicon build verification, the Windows parity inventory and fail-closed platform audits. Feature controls are intentionally introduced only as their `internal/api.Engine` bridge is implemented; placeholders are not accepted as parity.

The Mac development artifact is **not part of the current 0.0.5 public release allow-list**. The existing verified Windows/Linux 14-platform-artifact / 17-public-file release contract remains unchanged until macOS reaches full functionality, runtime evidence and distribution/signing/notarization gates.

Build on macOS with:

```bash
bash macos/BUILD.sh
```

The development output is `macos/dist/Ghost-FTP-<VERSION>-macOS.app.zip` and contains `Ghost FTP.app`.

See `PARITY.md` for the exact Windows action inventory that must be completed before macOS can be promoted from development source to a public desktop release platform.
