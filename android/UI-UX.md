# Ghost FTP Android UI/UX

The Android workspace uses the same canonical dark visual system as the Windows and Linux Ghost FTP distributions rather than a separate mobile brand.

## Canonical dark palette

Android maps the desktop `internal/uipalette.Dark` colors 1:1:

- Window `#0B0F17`
- Panel `#121824`
- List `#161D2A`
- Border `#2C3648`
- Text `#F2F5FA`
- Muted `#97A3B8`
- Accent `#5B7CFA`
- Accent strong `#7A98FF`
- Success `#4AD79B`
- Warning `#F2BA55`
- Danger `#FF6878`
- Selection `#202F50`

`GhostTheme` owns runtime widget styling, list/spinner adapters, button states, badges, rounded panels and status colors. `colors.xml` and `Theme.GhostFTP` keep the Android system bars and window background aligned with that same palette.

## Responsive workspace

The app is deliberately one-column on phone-sized displays. At `screenWidthDp >= 700`, the Local and Server cards become equal-width side-by-side panes. Connection, saved-site and transfer/status cards remain full-width so controls retain usable touch targets.

Each file pane contains its current path, navigation controls, file-management controls, saved-start/bookmark controls and a bounded list view. A normal tap on a directory opens it. Long-press selects either a file or directory for Rename/Delete without changing the normal navigation gesture.

## Local file management

Local operations stay inside the active Android Storage Access Framework tree. Ghost FTP does not request all-files storage access.

- **New folder** uses `DocumentsContract.createDocument` with directory MIME type.
- **Rename** uses `DocumentsContract.renameDocument` and verifies the resulting `COLUMN_DISPLAY_NAME`.
- **Delete** uses `DocumentsContract.deleteDocument` only after explicit confirmation.
- Create/Rename perform a fresh directory listing and reject an exact-name conflict before mutation.
- After a successful mutation, Ghost FTP performs another fresh listing before the visible pane snapshot is replaced.

## Remote file management

Remote FTP/FTPS mutations use the same authenticated session and the same strict FTPS trust boundary as navigation and transfers.

- **New folder** uses `MKD`.
- **Rename** requires `RNFR` followed by confirmed `RNTO`.
- **Delete file** uses `DELE`.
- **Delete directory** uses non-recursive `RMD`; Ghost FTP does not recursively erase server directory trees.
- Item names must be one safe path segment and cannot contain `/`, `\\`, NUL, CR/LF, `.` or `..`.
- Transport I/O failures fail closed and invalidate an uncertain session. An ambiguous final rename also closes the connection rather than silently reusing it.
- After a successful mutation, Ghost FTP freshly lists the current server directory before committing the visible pane snapshot.

Destructive local, remote and saved-site deletion uses an explicit confirmation dialog.

## Transfers

Only selected files can be uploaded/downloaded. Directories are never silently treated as files. Existing staged upload, staged SAF download, fail-closed passive data setup, real-I/O progress reporting and active-transfer cancellation remain authoritative.

During a transfer, the connection badge changes to **TRANSFER ACTIVE** and the Disconnect action becomes **Cancel transfer**. Progress text is derived from bytes actually read/written; final success still belongs to the staged commit path, not the progress counter.

## Saved sites and bookmarks

Saved sites remain non-secret. Passwords are memory-only. Local and server bookmarks can now also be explicitly removed from the mobile workspace. Quick Connect still does not create a hidden profile or bookmark.

## Security boundary

Android exposes FTP and strict FTPS only. SFTP remains intentionally absent until the Android implementation can provide strict host-key identity verification/pinning equivalent to desktop Ghost FTP. No telemetry, analytics, ads, hidden backend or external crash-reporting service is introduced by the UI layer.
