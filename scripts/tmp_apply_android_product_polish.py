from pathlib import Path

PATH = Path('android/app/src/main/java/app/ghostftp/client/MainActivity.java')
text = PATH.read_text(encoding='utf-8')


def replace_exact(old: str, new: str) -> None:
    global text
    count = text.count(old)
    if count != 1:
        raise SystemExit(f'Expected exactly one match, found {count}: {old!r}')
    text = text.replace(old, new, 1)


replacements = [
    (
        '        super.onCreate(state);\n        SharedPreferences preferences = getSharedPreferences(PREFS, MODE_PRIVATE);',
        '        super.onCreate(state);\n        GhostTheme.apply(this);\n        GhostTheme.applySystemBars(this);\n        SharedPreferences preferences = getSharedPreferences(PREFS, MODE_PRIVATE);',
    ),
    ('Local SAF storage and the active FTP/FTPS server. File management mirrors the desktop create, rename, delete and remote-permission workflow while preserving Android scoped storage.', 'Browse local files and your connected server from one workspace.'),
    ('Transfer only the selected file. Staged upload/download and cancellation safety remain authoritative.', 'Move the selected file safely and follow its progress while the transfer is active.'),
    ('Android Storage Access Framework only; Ghost FTP never requests broad all-files access. Long-press a directory to select it without opening it.', 'Choose a folder, browse files and manage only the locations you allow Ghost FTP to use. Long-press a folder to select it without opening it.'),
    ('Fresh MLSD listings over the active FTP/FTPS session; FTPS keeps strict TLS and hostname verification. Long-press a directory to select it without opening it.', 'Browse and manage files on the connected server. Secure connections are verified before use. Long-press a folder to select it without opening it.'),
    ('Quick Connect stays transient. Saved sites contain non-secret connection and navigation metadata only.', 'Connect quickly or save the server details you use often.'),
    ('Password is memory-only. Android currently exposes FTP and strict explicit FTPS; SFTP stays hidden until strict host-key verification exists.', 'Passwords are never saved. FTP and secure explicit FTPS are available on Android.'),
    ('Load, create, update or delete an explicit saved site. Quick Connect never creates hidden profiles.', 'Save only the connection details you choose. Passwords are never stored.'),
    ('Navigation state is explicit and account-bound. Quick Connect does not create hidden bookmarks.', 'Save frequently used local folders and remote paths for faster navigation.'),
    ('Local starts/bookmarks are SAF capability URIs and are freshly revalidated before navigation.', 'Saved local folders remain limited to locations you selected.'),
    ('Remote paths are bound to protocol, canonical host, port and exact username, and are freshly listed before visible commit.', 'Saved remote paths stay linked to the matching server and account.'),
    ('This surface shows the real active transfer lifecycle. No decorative queue or fake history is displayed.', 'Follow the current transfer and cancel it while cancellation is still safe.'),
    ('Progress comes from actual bytes read/written. Cancellation is available only before the irreversible final-name commit gate.', 'Progress reflects the current file transfer.'),
    ('Only settings with a real Android runtime owner are interactive.', 'Choose how Ghost FTP behaves on this device.'),
    ('Ghost FTP Android uses the canonical dark brand palette. These options change actual local runtime behavior.', 'Adjust local preferences for browsing and quick connections.'),
    ('Runtime security policy is informational here and cannot be weakened from the UI.', 'Security protections stay enforced automatically.'),
    ('Platform trust store + strict hostname verification', 'Certificate and hostname verification enabled'),
    ('Memory-only; never stored in site JSON/preferences', 'Kept in memory only and never saved'),
    ('Android SAF grants only; no all-files permission', 'Access limited to folders you select'),
    ('Hidden until strict Android host-key identity verification exists', 'Unavailable until strict server identity verification is enabled'),
    ('No telemetry, analytics, ads, fingerprinting or Ghost FTP cloud', 'No telemetry, analytics, ads or Ghost FTP cloud'),
    ('Build identity and privacy/security status for this Android app.', 'Version, supported protocols and privacy information.'),
    ('FTP + strict explicit FTPS on Android source line', 'FTP and explicit FTPS'),
    ('None: no telemetry, analytics, ads or hidden backend', 'No telemetry, analytics or ads'),
    ('Quick Connect metadata will be remembered. Passwords remain memory-only.', 'Quick Connect details will be remembered. Passwords are never saved.'),
    ('Quick Connect metadata persistence disabled and stored endpoint metadata cleared.', 'Saved Quick Connect details were cleared.'),
    ('Quick Connect mode. Connection details are not a saved site until you press Save / update.', 'Quick Connect ready. Use Save / update if you want to keep these server details.'),
    ('Site loaded. Password remains blank; connect to validate the saved server start directory.', 'Site loaded. Enter your password to connect.'),
    ('Site identity updated. Server start path and server bookmarks were cleared to prevent cross-server inheritance.', 'Site updated. Saved server paths and bookmarks were cleared because the connection details changed.'),
    ('Saved site deleted. Quick Connect settings were not converted into another profile.', 'Saved site deleted.'),
    ('Loaded site identity was edited. Save/update it first or switch to Quick Connect; saved server paths will not be reused across identities.', 'Connection details changed. Save the site or switch to Quick Connect before connecting.'),
    ('Connecting with strict FTPS TLS verification…', 'Connecting securely…'),
    ('Connecting with unencrypted FTP…', 'Connecting with FTP…'),
    ('FTPS connected. Certificate/hostname verified and server start directory freshly listed.', 'FTPS connected. Secure server identity verified.'),
    ('FTP connected. Warning: transport is unencrypted; server start directory freshly listed.', 'FTP connected. Warning: this connection is not encrypted.'),
    ('Finalizing transfer. The final-name commit has started and cannot be cancelled safely.', 'Finalizing transfer. It can no longer be cancelled safely.'),
    ('Cancellation accepted. Cleaning staged data before closing the session.', 'Cancellation accepted. Cleaning temporary data before closing the connection.'),
    ('Refreshing server directory…', 'Refreshing server folder…'),
    ('Server directory freshly listed.', 'Server folder refreshed.'),
    ('Saved server start directory updated after a successful listing: ', 'Server start folder saved: '),
    ('Server bookmark added for this site identity only.', 'Server bookmark added.'),
    ('Load or save a site first. Quick Connect does not create hidden profiles or bookmarks.', 'Load or save a site first.'),
    ('Local folder selected with persistent SAF permission.', 'Local folder selected.'),
    ('Local folder opened for this session only; persistent permission was not granted, so it cannot become a saved site start/bookmark.', 'Local folder opened for this session only.'),
    ('Selected SAF folder', 'Selected folder'),
    ('Local site start folder saved as a SAF capability URI; no filesystem-wide permission was added.', 'Local start folder saved.'),
    ('Local SAF bookmark added. It contains no credentials.', 'Local bookmark added.'),
    ('Local bookmark is stale or its persisted permission is unavailable. Re-select the folder to restore access.', 'Local bookmark is no longer available. Re-select the folder to restore access.'),
    ('Local bookmark opened after persisted SAF permission and directory listing were revalidated.', 'Local bookmark opened.'),
    ('Local folder permission or provider is no longer available. Choose the folder again.', 'Local folder is no longer available. Choose the folder again.'),
    ('Local current-folder filter', 'Local filter'),
    ('Server current-folder filter', 'Server filter'),
    ('Paired navigation was not committed: ', 'Could not open the matching folders: '),
    ('Downloading " + entry.name + " to a staged local document…', 'Downloading " + entry.name + "…'),
    ('Could not create staged local download document.', 'Could not prepare the local download.'),
    ('Could not open staged local download document.', 'Could not open the local download destination.'),
    ('A local item with the destination name appeared during download; staged data was not committed.', 'A file with that name appeared during download. No existing file was replaced.'),
    ('Finalizing download name…', 'Finalizing download…'),
    ('Storage provider rejected the final download name commit.', 'The download could not be finalized.'),
    ('Storage provider changed the requested final download name; commit was rejected.', 'The downloaded file name could not be verified.'),
    ('Download committed, but the connection was lost during finalization. Reconnect before another transfer.', 'Download completed, but the connection closed. Reconnect before another transfer.'),
    ('Download completed and committed: ', 'Download completed: '),
    ('Transfer cancelled before final-name commit.', 'Transfer cancelled.'),
    ('Storage provider could not verify the committed download name.', 'The downloaded file name could not be verified.'),
    ('Storage provider returned an empty committed download name.', 'The downloaded file name could not be verified.'),
    ('Storage permission was lost while verifying the committed download.', 'Storage access was lost while finishing the download.'),
    ('Dot-segment names are not allowed.', 'Choose a different name.'),
    ('Name must be a single safe item name without separators or control characters.', 'Names cannot contain slashes or control characters.'),
    ('Folder provider returned no directory listing.', 'This folder could not be opened.'),
    ('Local folder permission is no longer available.', 'Local folder access is no longer available.'),
]

for old, new in replacements:
    replace_exact(old, new)

for old in [
    '        card.addView(infoLine("Package", BuildConfig.APPLICATION_ID), matchWrapSpaced());\n',
    '        card.addView(infoLine("Release status", "Repository build " + BuildConfig.VERSION_NAME + (BuildConfig.DEBUG ? "; development package; not the production-signed public APK" : "; release package; official publication requires verified publisher-signature evidence")), matchWrapSpaced());\n',
]:
    replace_exact(old, '')

replace_exact(
    '''    private String displayLocalPath() {\n        if (treeUri == null || currentDocumentId == null) return "No folder selected";\n        return currentDocumentId.equals(rootDocumentId) ? "Selected folder" : currentDocumentId;\n    }''',
    '''    private String displayLocalPath() {\n        if (treeUri == null || currentDocumentId == null) return "No folder selected";\n        try {\n            Uri current = DocumentsContract.buildDocumentUriUsingTree(treeUri, currentDocumentId);\n            String[] projection = {DocumentsContract.Document.COLUMN_DISPLAY_NAME};\n            try (Cursor cursor = getContentResolver().query(current, projection, null, null, null)) {\n                if (cursor != null && cursor.moveToFirst()) {\n                    String name = cursor.getString(0);\n                    if (name != null && !name.trim().isEmpty()) return name;\n                }\n            }\n        } catch (RuntimeException ignored) {\n            // Fall back to a non-sensitive label when the provider cannot resolve a display name.\n        }\n        return "Selected folder";\n    }'''
)

replace_exact(
    '''    private static String safeMessage(Exception e) {\n        String value = e.getMessage();\n        return value == null || value.trim().isEmpty() ? e.getClass().getSimpleName() : value.replace('\\n', ' ').replace('\\r', ' ');\n    }''',
    '''    private static String safeMessage(Exception e) {\n        String value = e == null ? null : e.getMessage();\n        if (value == null || value.trim().isEmpty()) return "The operation could not be completed.";\n        String safe = value.replace('\\n', ' ').replace('\\r', ' ').trim();\n        if (safe.length() > 180\n                || safe.contains("/home/")\n                || safe.contains("/data/user/")\n                || safe.contains("java.")\n                || safe.contains("javax.")\n                || safe.contains("android.")\n                || safe.contains("Exception")\n                || safe.contains("StackTrace")) {\n            return "The operation could not be completed. Check the connection and try again.";\n        }\n        return safe;\n    }'''
)

forbidden = [
    'Storage Access Framework', 'scoped storage', 'Fresh MLSD', 'remain authoritative',
    'commit gate', 'runtime owner', 'canonical dark brand palette', 'production-signature evidence',
    'development package', 'Repository build', 'Android source line', 'site JSON/preferences',
    'SAF capability URI', 'hidden profiles', 'fake history', 'visible commit',
    'persistent SAF permission', 'staged local document', 'staged data', 'final-name commit',
    'Storage provider', 'freshly listed', 'site identity only', 'current-folder filter',
]
found = []
for phrase in forbidden:
    for number, line in enumerate(text.splitlines(), 1):
        if phrase in line and not line.lstrip().startswith('//'):
            found.append(f'{number}: {phrase}: {line.strip()}')
if found:
    raise SystemExit('Forbidden user-facing development wording remains:\n' + '\n'.join(found))

PATH.write_text(text, encoding='utf-8')
