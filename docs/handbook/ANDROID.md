# Ghost FTP for Android — complete feature manual and release/security roadmap

**Candidate:** 0.93.0 · **Platform:** Android 8 (API 26) and newer, target SDK 35 · **Source:** Kotlin Activity-based UI, FTP/FTPS through Apache Commons Net, SFTP through JSch, Android document picker for files. **Stage:** real Android application with bounded feature set, not parity with full Windows/Linux desktop.

## 1. Navigation and workspace inventory

The current MainActivity constructs real pages and click handlers for the following workspaces:

| Workspace | Visible controls and implemented behaviors | Boundaries |
| --- | --- | --- |
| Files | Remote list and item selection, directory enter/refresh, connection status | Rooted in an active FTP/FTPS/SFTP connection, not two full Windows-style desktop panes |
| Sites | Connection form, protocol, host, port, username, password, host-key fingerprint, SFTP keep-alive, remote start path, open/disconnect/test | Session-focused rather than full desktop saved-site folder/tag/bookmark inventory |
| Transfers | Local file selection, remote target/name, upload, download, rename, delete, create directory, activity log | Not a persistent multi-transfer scheduler with concurrency/bandwidth preferences |
| Sync & Backup | Export and restore **non-secret connection settings** through Android documents | It is not automatic folder sync or cloud backup of user files |
| Cloud storage | Masked signed HTTPS URL inputs, upload document selection, download document destination and status | One-object S3-compatible/GCS/Azure SAS link operations in 0.93.0 candidate; not Google Drive OAuth or S3 listing |
| Settings | App settings and supported local actions | Do not assume Windows/Linux settings inventory is mirrored here |
| Help & About | Product/version/support/privacy links, protocol disclosure | External links must be official allow-listed URLs |

The activity has a compact navigation rail or an overlay according to screen width. Android 15 system bar insets are handled explicitly; keyboard overlap, rotation, foldables, tablets, landscape and TalkBack still require physical/emulator QA.

## 2. Connection form and transfer actions

ConnectionProtocol currently supports exactly **FTP, explicit FTPS and SFTP** in the ordinary server connection form. FTP/FTPS default port 21; SFTP defaults to 22. ConnectionProfile holds the protocol, host, port, username, volatile password, host-key fingerprint, remote path and keep-alive setting.

- **Open connection:** validate host and port; use the protocol-specific transport with TLS certificate checks and SSH host-key verification. A reachable TCP socket alone is not proof of authenticated server identity.
- **Remote browse:** list entries and refresh; select a file or folder; entering a directory updates the active target; handle permission errors without dropping the session into an inconsistent state.
- **Upload:** choose a source via ACTION_OPEN_DOCUMENT; read only the granted URI; use temporary/staged remote path where applicable and protect the original remote file on failure.
- **Download:** choose a local application-private output with safe filename handling; verify transfer success before marking complete; do not silently overwrite unrelated files.
- **New folder:** require a safe path component and confirm the active remote directory. Never allow dangerous path traversal.
- **Rename and delete:** require a selected target and explicit in-app confirmation for destructive operations. Nonempty folder removal is not equivalent to delete-file.
- **Disconnect:** cancel active operation and clear in-memory password and transient connection state.
- **Connection setting backups:** use Android Storage Access Framework documents, omit secrets, cap imported data, validate format and never change stored credentials on malformed restores.

Android connection errors have a redaction helper which removes userinfo, bearer tokens and known password/token assignments. The app uses FLAG_SECURE to reduce screenshots/recents exposure and disables cleartext traffic in the manifest. These controls are defense in depth, not proof that every server/library error is free of secrets.

## 3. New Cloud storage workspace — exact steps and limitations

### Upload

1. In the provider's authorized console or backend, create a **short-lived signed HTTPS PUT URL** with upload permission to exactly one object.
2. Open Cloud storage in Ghost FTP, paste the URL into its masked input. Do not paste an AWS account secret, ordinary cloud sharing page, password or permanent API key.
3. Select a file using Android's document picker; the app obtains read access to that URI only for the selection.
4. The implementation validates the HTTPS scheme, signature parameter, username/fragment/port, public-looking hostname and DNS resolution; sends bytes with a bounded HttpsURLConnection PUT; checks HTTP response code.
5. On completion/failure, the signed URL input is cleared. The URL must not enter Android saved instance state, connection backup or activity logs.

### Download

1. Generate a short-lived, object-scoped signed HTTPS GET URL from the provider.
2. Paste it and choose an output destination using ACTION_CREATE_DOCUMENT.
3. The app sends GET and streams the response into its private cache, checks status and size, then writes bytes to the user-selected SAF document.
4. It deletes the private cache staging file afterward. Interrupted copy **may leave a partial user-selected SAF document**, because SAF generally does not provide atomic replace; this is a known reliability limitation requiring an additional recovery UI.

**Current support:** S3/R2/B2-compatible signed PUT/GET, Google Cloud Storage signed URLs and Azure SAS links when the provider's required signed method/headers match this client's request. Individual objects are limited to **1 GiB**. No multipart/resumable cloud upload, Drive OAuth account navigation, OneDrive/Dropbox/Box sign-in, cloud directory browsing, cross-cloud copy or multi-file queue is implemented in this Android workspace.

**Security:** Signed URLs are bearer credentials; generating a link with broad write permission is dangerous. The client requires HTTPS and system certificate trust, rejects redirects, URL-embedded username/password, private-looking names and literal IPs, and pre-checks DNS addresses. DNS rebinding/proxy routing remains a potential gap; additional transport-level IP pinning or trusted-provider allowlisting is a release-hardening item. The app does not transmit content to any Ghost FTP relay.

## 4. Private session model

| Data | Storage intent | Audit requirement |
| --- | --- | --- |
| FTP/SFTP password | Activity/session RAM only; not saved in Bundle | Clear on disconnect/destroy; inspect heap dumps and third-party protocol library errors |
| SFTP trusted host key | User-supplied fingerprint verification | Reject unexpected/mismatched key, never trust silently |
| Cloud signed URL | Masked text field and temporary in-memory transfer reference | Clear on picker cancel, completion, lifecycle destruction; no URLs in logs/backups |
| File source | Scoped document URI read | Use only selected grant, release when no longer necessary |
| Cloud download | App-private temporary cache then selected SAF URI | Enforce size limits, clean cache on every exit path, acknowledge non-atomic SAF commit |
| Recent activity | Bounded non-secret visible rows | Avoid secret-in-name/URL errors and provide clear action |
| Connection settings backup | JSON/settings without passwords | Size/schema/type validation, atomic restore and duplicate handling |
| Diagnostic messages | Redacted user-visible strings | Test adversarial multiline input, URL credentials, tokens and long secrets |

The app currently calls the platform document picker rather than requesting broad ALL_FILES_ACCESS. Its manifest has INTERNET and disables Android backup/data extraction for app-private state. Do not add unnecessary location, contacts, photos or background service permissions.

## 5. Security and privacy backlog (explicit priorities)

| Priority | Work | Test / acceptance gate |
| --- | --- | --- |
| P0 | Signed cloud URL SSRF hardening | IPv4/v6, IDNA, DNS rebinding, redirects, metadata hosts and proxy paths blocked |
| P0 | Atomic SAF download UX | On canceled/failed download, never present partial target as complete; offer safe retry/cleanup and final digest checks |
| P0 | Cloud provider header/signature compatibility | Signed S3/GCS PUT and Azure BlockBlob GET/PUT integration against disposable accounts |
| P0 | Lifecycle and cancellation | Reliable cancel button, Activity rotation/recreation, low-memory kill and no leaked active transfer thread |
| P0 | Sensitive field hygiene | No passwords or signed URLs in Bundle, clipboard by default, diagnostics, notifications or backup |
| P0 | FTP/FTPS/SFTP identity | Strict TLS hostname/cert and SFTP pinned host key, no trust-all or silent downgrade |
| P0 | App packages | Installable APK/AAB, correct release signing/configuration, cold start and click-by-click emulator smoke |
| P1 | Cloud accounts | OAuth Google Drive using PKCE and least-privilege scopes; Android Keystore-backed token handling, revoke/logout |
| P1 | S3 full capability | Region, bucket list, object pagination, streaming operations and safe endpoint credential handling |
| P1 | Transfer manager parity | Queue, progress bytes/ETA, cancel/retry, resumable transfer, conflicts, bandwidth/battery constraints |
| P1 | Background execution | Explicit foreground service only for user-initiated long transfers with notification and cancellation, scoped storage |
| P1 | Better storage and privacy | Encrypted vault option, user-facing data deletion and retention policy, secure temporary files |
| P1 | I18n and accessibility | Full translations, Android TalkBack descriptions, large fonts, minimum 48dp targets, rotations |
| P2 | Broader cloud ecosystem | Optional Dropbox, OneDrive, Box, WebDAV after auth/privacy model and provider-specific tests |
| P2 | Full desktop parity | Two-pane browser, site groups/tags/bookmarks, sync, diff, search and cloud multi-account UX |

## 6. Test cases for each Android operation

1. Install both debug/preview and correctly signed release APK, launch repeatedly on Android 8 through recent target, verify no startup crash.
2. Validate every protocol through connect/disconnect, bad credentials, TLS and host-key mismatch, server permission error and network loss.
3. Browse empty/nested/Unicode directories, select file/folder, create, rename, delete and protected root; ensure every visible button has a handler.
4. Transfer zero-byte and large files; disconnect/cancel during upload/download; check previously good remote object was preserved and no leftover staged objects remain unexpectedly.
5. Test signed cloud PUT/GET with provider expiring URL, wrong method, 403, 404, 429, 500, server redirect, content length mismatch, cloud DNS/IP tricks.
6. Test the SAF picker canceled, read grant denied, output grant revoked, full disk, app in background, rotation and low-memory process recreation.
7. Verify password masking, no secret in saved instance state, clipboard/notification redaction, app-private cache cleanup and correct backup omissions.
8. Run unit tests and connected emulator screenshot/accessibility tests on the exact commit, not just a linted source tree.

## 7. Build/release evidence and source references

- android/app/src/main/java/com/ghostftp/android/MainActivity.kt — all Android workspace UI, forms and actions.
- ConnectionModel.kt — protocol connection model, handlers, fingerprint security and safe remote replacements.
- SignedCloudTransfer.kt — signed URL validation, bounded streaming and SAF cloud transfers.
- android/app/src/test and android/app/src/androidTest — JVM and instrumented tests.
- android/app/build.gradle.kts and AndroidManifest.xml — SDK level, packaging, permissions and backup controls.
- .github/workflows/ghostftp-android.yml — Android CI, package artifacts and emulator smoke checks.

A successful CI job does not replace physical-device sign-off. Treat this candidate as **in progress** until the latest commit's tests/builds finish and public release asset checks complete.
