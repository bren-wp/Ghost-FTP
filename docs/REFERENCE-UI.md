# Reference UI

The current Ghost FTP reference contract covers **Windows, Linux and Android**.

## Canonical Files workspace

Primary navigation:

- Files
- Connections / Sites
- Transfer Queue / Transfers
- Settings

Primary Files actions:

- Back
- Forward
- Refresh
- New Folder
- Upload
- Download
- Bookmarks
- More

Desktop file panes:

- **Local Files** — Name, Size, Modified
- **Remote Files** — Name, Size, Modified, Permissions

Transfer Queue:

- File
- Direction
- Progress
- Status
- Speed
- ETA
- Clear Completed

The status surface must reflect real connection/queue state and must never fabricate server identity, transfer rows or progress.

## Windows

Use the supplied Windows reference for layout proportions, control order, text visibility, dark surface hierarchy and Ghost Gold accents while preserving native Windows behavior.

## Linux

Use the supplied Linux reference for the same application composition inside Linux window/system chrome. Linux-specific paths and lifecycle remain native.

## Android

Use the supplied Android reference for the mobile composition: branded app bar, current connection card, action rows, Local/Remote file cards, Transfer Queue and five-item bottom navigation.

## Dialogs and More surfaces

Quick Connect, connection editing, file mutation prompts, Settings, bookmarks, errors and More actions must use the same dark Ghost FTP design language. Dialogs must be backed by real actions and must not expose development/debug copy.

## Text visibility

No shipping label may be unintentionally clipped. Narrow surfaces may wrap labels where the design allows it; touch targets must remain usable.

## Evidence rule

Only authentic runtime screenshots are product evidence. Generated images and visual references are design targets, not proof of execution.


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. Current hosted CI does not claim native Windows-on-ARM runtime execution evidence.
