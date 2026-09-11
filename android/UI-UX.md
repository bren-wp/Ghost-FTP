# Ghost FTP Android UI/UX contract

Ghost FTP Android is a mobile client, not a compressed copy of the desktop window. The Android shell deliberately exposes one active work surface at a time while preserving the same security and privacy invariants as the desktop source line where the underlying protocol feature exists.

## Navigation model

### Phone

Phones use a real left navigation drawer opened by the local hamburger vector icon. The drawer contains only runtime-owned surfaces:

- **Files**
- **Sites**
- **Bookmarks**
- **Transfers**
- **Settings**
- **About**

Selecting a destination hides all other surfaces and commits exactly one active workspace. The Back action closes an open drawer before leaving the Activity.

### Tablet

When Android reports at least 700 dp of screen width, the same navigation model is rendered as a persistent left sidebar. The content remains a single active surface; the wider layout does not re-create the desktop application as one long page.

The Files surface may place local and remote panes side-by-side only when there is enough width. On narrower layouts they stack inside the Files surface rather than mixing unrelated Sites, Bookmarks, Settings or About controls into the same scroll flow.

## Surface ownership

### Files

Files owns the active local and server directory views.

Local storage is restricted to Android Storage Access Framework capabilities granted by the user. The surface provides the local current path, folder picker, Up, Refresh, a real directory listing and local file selection.

The server pane displays the active remote path, Up, Refresh and the freshly listed remote entries for the active FTP/FTPS session. Upload and download actions are enabled only for actual file selections. Directory transfer is not implied by file buttons.

Staged upload/download, exact-name commit verification, FTPS verification and transfer-generation protections are protocol/runtime contracts and are not weakened by the UI split.

### Sites

Sites owns Quick Connect and explicitly saved connection profiles.

Quick Connect exposes only Android protocols that have complete runtime security ownership: FTP and explicit FTPS. FTPS keeps strict certificate and hostname verification. FTP remains visibly unencrypted by its runtime status messaging.

Passwords remain memory-only. Saving a site never persists a password. Quick Connect never creates a hidden profile.

Saved site identity continues to bind remote navigation state to protocol, canonical host, port and exact username. Changing that identity clears remote start-directory/bookmark state rather than silently carrying it to another endpoint.

### Bookmarks

Bookmarks owns local SAF navigation bookmarks, local site start folders, remote bookmarks and remote site start directories.

Local bookmarks are persisted SAF capability URIs. They must still have a persisted permission and produce a fresh directory query before the visible local state is changed.

Remote bookmarks remain account-bound. Opening one performs a fresh server listing before the visible remote path is committed. Quick Connect does not create hidden bookmarks.

The Android UI exposes explicit Add, Remove and Open actions. Bookmark records remain non-secret.

### Transfers

Transfers owns only real transfer state.

It shows the current transfer status/progress produced by actual bytes read or written and exposes **Cancel active transfer** only while the transfer is in a cancellable phase. Once the irreversible final-name commit gate has started, the control changes to **Finalizing…** and cancellation is disabled.

There is no decorative retry, resume, completed-history or queue control in this surface. Those controls must not appear until a real Android runtime implementation owns them.

### Settings

Settings exposes only controls with an actual Android runtime owner.

Current interactive settings are:

- whether non-secret Quick Connect endpoint metadata is remembered;
- whether file sizes are shown in Files lists.

Disabling endpoint persistence removes stored host, username, protocol and port metadata. It never affects the password rule because passwords are never persisted.

Security rows are informational and cannot weaken TLS verification, storage confinement, transfer staging or privacy behavior.

### About

About presents the Ghost FTP product identity, Android package/build identity, currently implemented Android protocol status and privacy information.

The repository root version remains `0.0.3` until a dedicated future release-prep change. The current Android APK is a post-0.0.3 development/debug-signed artifact and must not be represented as part of the already published Windows/Linux 0.0.3 release.

## Remote Desktop / RDP

**Remote Desktop is not shown in Android navigation.** There is currently no reviewed Android RDP runtime owner with the required credential and launcher/engine security contract. Ghost FTP does not expose a decorative RDP button, a fake Coming Soon destination or a control that has no working implementation behind it.

If Android RDP is introduced later, it requires a separate architecture/security review and must prove the actual runtime client/engine, availability detection and password handling before the navigation item can exist.

## SFTP status

Android SFTP is also fail-closed. The protocol picker exposes only `FTPS` and `FTP`; it does not offer an SFTP connection route until Android has strict host-key identity verification/pinning equivalent to the desktop security contract.

Documentation may state that SFTP is intentionally hidden. That informational text must never be confused with a runtime SFTP option.

## Visual system

Ghost FTP Android uses the project dark palette and local Android resources. Navigation uses local vector drawables for hamburger, Files, Sites, Bookmarks, Transfers, Settings and About.

The runtime UI must not depend on emoji icons, externally hosted fonts, tracking resources or decorative controls that look actionable but have no owner.

## Screenshot evidence

Documentation screenshots must come from a real build of the Android Activity running in an emulator or physical Android runtime. Generated mockups, Figma compositions and marketing renders must not be labeled as application screenshots.

The screenshot workflow must record the source/build SHA, verify PNG files and capture the real surfaces that exist: Files, Sites, Bookmarks, Transfers and Settings/About. RDP must not be captured or named unless it becomes a real Android runtime surface.
