# Ghost FTP Feature Matrix

Legend: **Implemented** = present in source; **CI verified** = exercised by automated checks; **OS/E2E pending** = requires final target-system evidence.

| Area | Feature | Status | Notes |
| --- | --- | --- | --- |
| Connections | Quick Connect | Implemented | Ephemeral connection path without saving a profile |
| Connections | Saved profiles | Implemented | Site Manager persistence |
| Protocols | FTP | Implemented | Final real-server E2E still required |
| Protocols | FTPS | Implemented | Final explicit/implicit TLS E2E still required |
| Protocols | SFTP | Implemented | Password/key/host-key E2E still required |
| File UI | Dual-pane local/remote browser | Implemented | Core production workflow |
| File ops | Upload/download | Implemented | Real protocol acceptance still required |
| File ops | Rename/delete/new folder | Implemented | Backend capability-aware |
| File ops | Duplicate/open containing folder | Implemented | Available where supported |
| Integrity | SHA-256 | Implemented | Local/SFTP paths where supported |
| Permissions | Numeric chmod + matrix | Implemented | Protocol-aware |
| Transfers | Queue/progress/speed/ETA | Implemented | UI + engine path |
| Transfers | Pause/resume/cancel/retry | Implemented | Failure-path acceptance still required |
| Organization | Favorites/tags/folders/bookmarks | Implemented | Persisted Site Manager metadata |
| Preferences | Language/appearance/transfers | Implemented | 14-language UI coverage |
| Preferences | Connection/security/privacy | Implemented | Includes no-required-telemetry policy |
| Updates | Signed update configuration | Implemented | Production endpoint configured |
| Recovery | Profile store recovery | Implemented | Corrupt copy preserved |
| Recovery | SQLite recovery | Implemented | DB/WAL/SHM quarantine path |
| Distribution | Windows portable EXE | Build target | Native release pipeline |
| Distribution | Windows Setup EXE | Build target | NSIS prerelease packaging |
| Distribution | AppImage/DEB/RPM | Build targets | Linux x86-64 |
| Visual QA | 1290×852 reference geometry | Implemented in source | Final native pixel proof pending |
| Responsive | 1280×800 to 480×800 targets | Implemented in CSS | Final native render proof pending |

## Useful options worth adding later

These are recommendations, not claims about current functionality:

- Per-site bandwidth limits and schedules stored directly on profiles.
- Visual transfer conflict rules editor with reusable presets.
- Remote file diff before overwrite.
- Multi-server deployment groups with dry-run mode.
- Saved synchronization jobs with preview and rollback metadata.
- Optional portable encrypted profile vault for users who cannot use OS credential storage.
- Connection health dashboard and protocol diagnostics.
- Release-channel selector with stable/beta/RC policies.
- Built-in exportable diagnostic bundle with automatic secret redaction.
- Accessibility audit automation for keyboard order, contrast and screen-reader labels.
