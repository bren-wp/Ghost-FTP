# Platform parity

Ghost FTP parity is defined around shared product behavior, not identical platform internals.

## Active platforms

| Capability | Windows | Linux | Android |
| --- | ---: | ---: | ---: |
| Files workspace | Yes | Yes | Yes |
| Connections / saved sites | Yes | Yes | Yes |
| Transfer queue | Yes | Yes | Yes |
| Settings | Yes | Yes | Yes |
| Bookmarks | Yes | Yes | Yes |
| FTP / FTPS | Yes | Yes | Yes |
| SFTP | Yes | Yes | Hidden until strict host-key verification |
| Remote permissions | Yes | Yes | Platform-appropriate only |
| Remote Edit | Yes | Yes | No desktop-equivalence claim |
| Recursive search / filters | Yes | Yes | Platform-appropriate only |
| Directory comparison | Yes | Yes | No desktop-equivalence claim |

## UI contract

The supplied Ghost FTP reference composition defines the hierarchy for the primary Files surface:

1. product branding and primary navigation;
2. connection address/status/Quick Connect;
3. Back, Forward, Refresh, New Folder, Upload, Download, Bookmarks and More;
4. Local Files / Remote Files;
5. Transfer Queue;
6. connection/transfer status.

Windows and Linux preserve this hierarchy with desktop-native interaction. Android preserves the same product nouns and state while adapting the layout for mobile.

## Retired platform

macOS is removed from current source/release support and is not part of parity accounting.

## Evidence

Parity claims require exact-head automated tests and authentic runtime screenshots. Reference images and generated mockups can guide layout work but cannot substitute for runtime evidence.


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. The metadata value above is intentionally explicit: current hosted CI does not claim native Windows-on-ARM runtime execution evidence.
