# Ghost FTP desktop reference UI

Ghost FTP **0.0.1** uses a focused native two-pane desktop layout on Windows and Linux. The same typed Core owns protocol, profile, transfer and Remote Edit behavior; platform frontends should differ only where the operating system requires native presentation details.

## Visual direction

The maintained dark application palette uses restrained navy/charcoal surfaces with clear blue interaction accents. The UI should remain professional, low-noise and task-oriented rather than adding permanent panels for every capability.

Core dark palette references:

```text
Window/background: #0B0F17
Panel:             #121824
Raised surface:    #161D2A
Border:            #2C3648
Primary text:      #F2F5FA
Muted text:        #97A3B8
Accent:            #5B7CFA
Accent strong:     #7A98FF
```

Classic Light remains the fresh/fallback appearance where currently documented.

## Main Workspace

![Ghost FTP Main Workspace](images/ghost-ftp-main-workspace.png)

The primary workspace keeps connection controls, Local pane, Remote pane, queue/status and high-value actions visible without turning the application into a dense toolbar grid.

The Remote side exposes normal file management plus **Edit** for one supported regular remote text file. Edit should not create a permanent third pane.

## Site Manager

![Ghost FTP Site Manager](images/ghost-ftp-site-manager.png)

Site Manager provides profile selection/editing while preserving the same credential-consent and trust semantics as the main connection workflow.

## Settings

![Ghost FTP Settings](images/ghost-ftp-settings.png)

Settings keeps related options in one application-owned native surface and validates a complete settings candidate before persistence.

## About

![Ghost FTP About](images/ghost-ftp-about.png)

About displays the runtime product/version identity generated from the canonical build version. For the current line this resolves to **Ghost FTP 0.0.1**.

## Built-in Remote Editor

Remote Edit is deliberately simple:

- one Edit action from the Remote file surface;
- one native/modal editor surface;
- **Save**, **Reload** and **Close**;
- dirty-state indication;
- clear conflict/error status;
- keyboard support including save/reload shortcuts;
- no external editor process and no permanent extra application panel.

Windows uses the application-owned native text editor dialog. Linux uses the maintained native X11/XWayland-compatible overlay. Both use the same Remote Edit engine for bounded open/save, conflict detection, line-ending preservation, permission handling and read-back verification.

After a verified successful save, the remote list refreshes metadata so size/mtime state does not remain stale.

## Responsive behavior

Windows startup/minimum geometry must respect the active monitor work area. Mixed-DPI transitions use the destination monitor, and negative monitor origins remain valid. A compact/smaller screen must not force the app outside the usable work area.

Linux layout must preserve the same major workflow priorities at supported sizes without exposing fake controls whose backend behavior is incomplete.

## Action clarity

Buttons must be enabled only when the underlying action is valid. A file action should not appear available for an unsupported selection, disconnected session or incompatible object type.

High-value actions are preferred over duplicate controls. Context-specific actions should stay close to the pane/object they affect.

## Authentic screenshot contract

The four maintained UI images are generated from the real Windows production application by `.github/workflows/ui-screenshots.yml` and are treated as release evidence.

**Mockups, image-generation output and manually composed approximations are not accepted** as production UI evidence.

The screenshot workflow must continue to capture:

```text
images/ghost-ftp-main-workspace.png
images/ghost-ftp-site-manager.png
images/ghost-ftp-settings.png
images/ghost-ftp-about.png
```

For `VERSION` or maintained desktop UI changes, pull-request capture provides exact-head evidence. Release-prep capture remains evidence-only and must not silently persist stale screenshots back into the release-prep branch.

## Accessibility and usability

- visible keyboard focus where native controls support it;
- readable contrast in Light and Dark appearances;
- clear destructive-action confirmation when enabled;
- localized user-facing labels through the maintained 24-language catalog;
- no credential or server-secret text in screenshots/documentation evidence.

See [Settings](SETTINGS.md), [Platform parity](PLATFORM-PARITY.md), [Testing](TESTING.md) and [Privacy](PRIVACY.md).
