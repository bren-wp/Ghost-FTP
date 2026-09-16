#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
activity_path = ROOT / "android/app/src/main/java/app/ghostftp/client/MainActivity.java"
test_path = ROOT / "scripts/test_android_contract.py"

activity = activity_path.read_text(encoding="utf-8")
tests = test_path.read_text(encoding="utf-8")

replacements = {
    'button("Permissions / CHMOD")': 'button("Permissions")',
    'card("QUICK CONNECT / CONNECTION",': 'card("QUICK CONNECT",',
    'field("Password (memory only)", true)': 'field("Password (never saved)", true)',
    'card("LOCAL SAF BOOKMARKS",': 'card("LOCAL BOOKMARKS",',
    'card("UI / LOCAL PREFERENCES",': 'card("APP PREFERENCES",',
    'infoLine("FTPS", "Certificate and hostname verification enabled")': 'infoLine("FTPS", "Secure certificate checks are enabled")',
    'infoLine("Passwords", "Kept in memory only and never saved")': 'infoLine("Passwords", "Never saved")',
    'infoLine("SFTP", "Unavailable until strict server identity verification is enabled")': 'infoLine("SFTP", "Not available in the Android app")',
    '"Searching local folders within bounded safety limits…"': '"Searching local folders…"',
    '"Local recursive search failed: "': '"Local search failed: "',
    '"No local recursive-search results for: "': '"No local search results for: "',
    '(result.entry.directory ? "DIR   " : "FILE  ") + result.displayPath': '(result.entry.directory ? "Folder · " : "File · ") + result.displayPath',
    '"Searching server folders within bounded safety limits…"': '"Searching server folders…"',
    '.setTitle("Server recursive search")': '.setTitle("Search server folders")',
    '.setMessage("Searching with directory, depth, result and 45-second safety bounds. Cancelling closes this FTP/FTPS session immediately.")': '.setMessage("Searching server folders. Cancelling the search will close the current connection.")',
    '"Server recursive search cancelled. Connection closed; reconnect before continuing."': '"Search cancelled. Connection closed; reconnect before continuing."',
    'new IOException("Server recursive search was cancelled.")': 'new IOException("Search was cancelled.")',
    'new IOException("Server recursive search reached the 45-second safety deadline.")': 'new IOException("Search took too long and was stopped.")',
    'new IOException("Server connection changed during recursive search.")': 'new IOException("The server connection changed during search.")',
    '"Server recursive search failed: "': '"Server search failed: "',
    '"No server recursive-search results for: "': '"No server search results for: "',
    'marker = "LOCAL ONLY";': 'marker = "Only on this device";',
    'marker = "SERVER ONLY";': 'marker = "Only on server";',
    'marker = "DIFFERENT";': 'marker = "Different";',
    'marker = "SAME";': 'marker = "Same";',
    'setStatus("Comparison: " + row.difference.name().replace(\'_\', \' \') + " · " + row.name);': 'setStatus("Comparison selected: " + row.name);',
    '"Synchronized navigation is unavailable because the paired directories changed."': '"The matching folders changed. Refresh and try again."',
    '"Opening paired directories…"': '"Opening matching folders…"',
    '"Opened paired local/server directory: "': '"Opened matching folders: "',
    '"Remote Edit supports explicitly reported regular text files only; links and special entries are not editable."': '"Select a regular text file to edit. Links and special entries cannot be edited."',
    '"Opening Remote Edit with conflict-safe snapshot…"': '"Opening file for editing…"',
    'field("Remote UTF-8 text", false)': 'field("File contents", false)',
    '"Remote Edit opened with SHA-256 conflict detection and read-back verification."': '"Remote file opened for editing."',
    '"Remote Edit is checking for conflicts and saving…"': '"Checking for changes and saving…"',
    '"Remote Edit saved and verified by read-back: "': '"Remote file saved: "',
    '"Load or save a site first. Quick Connect does not create hidden site state."': '"Save or load a site before setting a start folder."',
    '"Choose a folder with persistent Android permission before setting the site start folder."': '"Choose a folder that Ghost FTP can reopen before setting it as the start folder."',
    '"Load or save a site first. Quick Connect does not create hidden bookmarks."': '"Save or load a site before adding a local bookmark."',
    '"Choose a folder with persistent Android permission before bookmarking it."': '"Choose a folder that Ghost FTP can reopen before adding it as a bookmark."',
    'String type = e.directory ? "DIR   " : "FILE  ";': 'String type = e.directory ? "Folder · " : "File · ";',
}

for old, new in replacements.items():
    if old not in activity:
        raise SystemExit(f"Expected Android source marker not found: {old}")
    activity = activity.replace(old, new)

# Contract expectations must follow the production-facing copy while preserving
# the underlying privacy and fail-closed behavior.
test_replacements = {
    'self.assertIn("Quick Connect does not create hidden site state.", activity)': 'self.assertIn("Save or load a site before setting a start folder.", activity)',
    'self.assertIn("Quick Connect does not create hidden bookmarks.", activity)': 'self.assertIn("Save or load a site before adding a local bookmark.", activity)',
    "'infoLine(\"SFTP\", \"Unavailable until strict server identity verification is enabled\")',": "'infoLine(\"SFTP\", \"Not available in the Android app\")',",
}
for old, new in test_replacements.items():
    if old not in tests:
        raise SystemExit(f"Expected Android contract marker not found: {old}")
    tests = tests.replace(old, new)

needle = '        for marker in ("developer", "development", "debug", "demo", "mock", "staging"):\n            self.assertNotIn(marker, shipping_copy)\n'
replacement = '''        for marker in ("developer", "development", "debug", "demo", "mock", "staging"):\n            self.assertNotIn(marker, shipping_copy)\n\n        for marker in (\n            "permissions / chmod",\n            "quick connect / connection",\n            "password (memory only)",\n            "local saf bookmarks",\n            "ui / local preferences",\n            "bounded safety limits",\n            "safety bounds",\n            "recursive-search",\n            "server recursive search",\n            "local recursive search",\n            "sha-256 conflict detection",\n            "read-back verification",\n            "hidden site state",\n            "hidden bookmarks",\n            "persistent android permission",\n            "strict server identity verification",\n        ):\n            self.assertNotIn(marker, shipping_copy)\n'''
if needle not in tests:
    raise SystemExit("Expected shipping-copy regression block not found")
tests = tests.replace(needle, replacement)

activity_path.write_text(activity, encoding="utf-8")
test_path.write_text(tests, encoding="utf-8")
