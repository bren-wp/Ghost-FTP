#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FTP = ROOT / "android/app/src/main/java/app/ghostftp/client/FtpSession.java"
MAIN = ROOT / "android/app/src/main/java/app/ghostftp/client/MainActivity.java"


def once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly one marker, found {count}")
    return text.replace(old, new, 1)


ftp = FTP.read_text(encoding="utf-8")
ftp = once(
    ftp,
    "    synchronized void upload(String remotePath, InputStream input, TransferCommitGate gate) throws IOException {\n        ensureConnected();\n",
    "    synchronized void upload(String remotePath, InputStream input, TransferCommitGate gate) throws IOException {\n"
    "        uploadInternal(remotePath, input, gate, \"\");\n"
    "    }\n\n"
    "    synchronized void uploadPreservingMode(String remotePath, InputStream input, TransferCommitGate gate, String mode) throws IOException {\n"
    "        uploadInternal(remotePath, input, gate, normalizeChmodMode(mode));\n"
    "    }\n\n"
    "    private void uploadInternal(String remotePath, InputStream input, TransferCommitGate gate, String preserveMode) throws IOException {\n"
    "        ensureConnected();\n",
    "upload overload",
)
ftp = once(
    ftp,
    "        if (!gate.markReadyToCommit()) {\n",
    "        if (!preserveMode.isEmpty()) {\n"
    "            try {\n"
    "                expect(mutationCommand(\"SITE CHMOD \" + preserveMode + \" \" + sanitizeArgument(tempPath)), 200);\n"
    "                verifyRemoteMode(tempPath, preserveMode);\n"
    "            } catch (IOException e) {\n"
    "                deleteRemoteBestEffort(tempPath);\n"
    "                throw new IOException(\"Remote Edit staging permissions could not be applied and verified; final name was not committed.\", e);\n"
    "            }\n"
    "        }\n\n"
    "        if (!gate.markReadyToCommit()) {\n",
    "staged mode verification",
)
ftp = once(
    ftp,
    "    synchronized void chmod(String remotePath, String mode) throws IOException {\n        String path = requireMutableRemotePath(remotePath);\n        String safeMode = normalizeChmodMode(mode);\n        expect(mutationCommand(\"SITE CHMOD \" + safeMode + \" \" + sanitizeArgument(path)), 200);\n    }\n",
    "    synchronized void chmod(String remotePath, String mode) throws IOException {\n"
    "        String path = requireMutableRemotePath(remotePath);\n"
    "        String safeMode = normalizeChmodMode(mode);\n"
    "        expect(mutationCommand(\"SITE CHMOD \" + safeMode + \" \" + sanitizeArgument(path)), 200);\n"
    "    }\n\n"
    "    synchronized void requireRemoteModeUnchanged(String remotePath, String expectedMode) throws IOException {\n"
    "        String safeExpected = expectedMode == null ? \"\" : expectedMode.trim();\n"
    "        if (safeExpected.isEmpty()) return;\n"
    "        verifyRemoteMode(requireMutableRemotePath(remotePath), normalizeChmodMode(safeExpected));\n"
    "    }\n\n"
    "    private void verifyRemoteMode(String remotePath, String expectedMode) throws IOException {\n"
    "        String path = requireMutableRemotePath(remotePath);\n"
    "        String parent = parentRemote(path);\n"
    "        String name = path.substring(path.lastIndexOf('/') + 1);\n"
    "        for (RemoteEntry entry : list(parent)) {\n"
    "            if (!name.equals(entry.name)) continue;\n"
    "            if (entry.permissions.isEmpty() || !sameOctalMode(entry.permissions, expectedMode)) {\n"
    "                throw new IOException(\"Remote permissions changed or could not be verified.\");\n"
    "            }\n"
    "            return;\n"
    "        }\n"
    "        throw new IOException(\"Remote file disappeared while verifying permissions.\");\n"
    "    }\n\n"
    "    private static boolean sameOctalMode(String left, String right) {\n"
    "        try {\n"
    "            return Integer.parseInt(normalizeChmodMode(left), 8) == Integer.parseInt(normalizeChmodMode(right), 8);\n"
    "        } catch (IOException | NumberFormatException e) {\n"
    "            return false;\n"
    "        }\n"
    "    }\n",
    "mode verification helpers",
)
ftp = once(
    ftp,
    "        boolean directory = facts.contains(\"type=dir\");\n        long size = 0L;\n        long modified = 0L;\n        String permissions = \"\";\n        for (String fact : facts.split(\";\")) {\n            if (fact.startsWith(\"size=\")) {\n",
    "        String type = \"\";\n"
    "        long size = 0L;\n"
    "        long modified = 0L;\n"
    "        String permissions = \"\";\n"
    "        for (String fact : facts.split(\";\")) {\n"
    "            if (fact.startsWith(\"type=\")) {\n"
    "                type = fact.substring(5).trim();\n"
    "            } else if (fact.startsWith(\"size=\")) {\n",
    "mlsd type parse",
)
ftp = once(
    ftp,
    "        return new RemoteEntry(name, directory, size, modified, permissions);\n",
    "        boolean directory = \"dir\".equals(type);\n"
    "        return new RemoteEntry(name, directory, size, modified, permissions, type);\n",
    "mlsd typed entry",
)
FTP.write_text(ftp, encoding="utf-8")

main = MAIN.read_text(encoding="utf-8")
main = once(
    main,
    "        if (entry.directory) {\n            setStatus(\"Remote Edit supports regular text files only.\");\n            return;\n        }\n",
    "        if (!entry.regularFile) {\n"
    "            setStatus(\"Remote Edit supports explicitly reported regular text files only; links and special entries are not editable.\");\n"
    "            return;\n"
    "        }\n",
    "regular file remote edit guard",
)
main = once(
    main,
    "                    showRemoteEditor(owner, path, entry.name, generation, snapshot);\n",
    "                    showRemoteEditor(owner, path, entry.name, entry.permissions, generation, snapshot);\n",
    "pass remote edit mode",
)
main = once(
    main,
    "    private void showRemoteEditor(FtpSession owner, String path, String name, long generation, RemoteTextDocument.Snapshot snapshot) {\n",
    "    private void showRemoteEditor(FtpSession owner, String path, String name, String originalMode, long generation, RemoteTextDocument.Snapshot snapshot) {\n",
    "remote editor signature",
)
main = once(
    main,
    "        RemoteEditorState state = new RemoteEditorState(owner, path, name, generation, snapshot, editor);\n",
    "        RemoteEditorState state = new RemoteEditorState(owner, path, name, originalMode, generation, snapshot, editor);\n",
    "remote editor state mode",
)
main = once(
    main,
    "                        state.owner, state.path, state.snapshot.sha256, state.snapshot.lineEnding, text);\n",
    "                        state.owner, state.path, state.snapshot.sha256, state.snapshot.lineEnding, state.originalMode, text);\n",
    "remote edit save mode",
)
main = once(
    main,
    "        final String name;\n        final long generation;\n        final EditText editor;\n",
    "        final String name;\n        final String originalMode;\n        final long generation;\n        final EditText editor;\n",
    "remote editor mode field",
)
main = once(
    main,
    "        RemoteEditorState(FtpSession owner, String path, String name, long generation,\n                          RemoteTextDocument.Snapshot snapshot, EditText editor) {\n            this.owner = owner;\n            this.path = path;\n            this.name = name;\n            this.generation = generation;\n",
    "        RemoteEditorState(FtpSession owner, String path, String name, String originalMode, long generation,\n"
    "                          RemoteTextDocument.Snapshot snapshot, EditText editor) {\n"
    "            this.owner = owner;\n"
    "            this.path = path;\n"
    "            this.name = name;\n"
    "            this.originalMode = originalMode == null ? \"\" : originalMode.trim();\n"
    "            this.generation = generation;\n",
    "remote editor mode constructor",
)
old_search_start = """        long generation = ++advancedOperationGeneration;
        setBusy(true, \"Searching server folders within bounded safety limits…\");
        io.execute(() -> {
            try {
                List<RemoteSearchResult> results = new ArrayList<>();
"""
new_search_start = """        long generation = ++advancedOperationGeneration;
        long deadlineNanos = System.nanoTime() + WorkspaceOps.MAX_REMOTE_SEARCH_MILLIS * 1_000_000L;
        setBusy(true, \"Searching server folders within bounded safety limits…\");
        AlertDialog searchDialog = new AlertDialog.Builder(this)
                .setTitle(\"Server recursive search\")
                .setMessage(\"Searching with directory, depth, result and 45-second safety bounds. Cancelling closes this FTP/FTPS session immediately.\")
                .setNegativeButton(\"Cancel search\", null)
                .setCancelable(false)
                .create();
        searchDialog.setOnShowListener(ignored -> searchDialog.getButton(AlertDialog.BUTTON_NEGATIVE).setOnClickListener(v -> {
            if (generation != advancedOperationGeneration || session != owner) return;
            advancedOperationGeneration++;
            owner.abort();
            session = null;
            connectedIdentityKey = null;
            remoteEntries.clear();
            selectedRemote = -1;
            currentRemotePath = \"/\";
            busy = false;
            renderRemote();
            setStatus(\"Server recursive search cancelled. Connection closed; reconnect before continuing.\");
            refreshButtons();
            searchDialog.dismiss();
        }));
        searchDialog.show();
        io.execute(() -> {
            try {
                List<RemoteSearchResult> results = new ArrayList<>();
"""
main = once(main, old_search_start, new_search_start, "remote search cancellation dialog")
main = once(
    main,
    "                    if (session != owner || !owner.isConnected()) throw new IOException(\"Server connection changed during recursive search.\");\n                    RemoteSearchNode node = queue.removeFirst();\n",
    "                    if (generation != advancedOperationGeneration) throw new IOException(\"Server recursive search was cancelled.\");\n"
    "                    if (System.nanoTime() > deadlineNanos) throw new IOException(\"Server recursive search reached the 45-second safety deadline.\");\n"
    "                    if (session != owner || !owner.isConnected()) throw new IOException(\"Server connection changed during recursive search.\");\n"
    "                    RemoteSearchNode node = queue.removeFirst();\n",
    "remote search deadline",
)
main = once(
    main,
    "                    busy = false;\n                    refreshButtons();\n                    showRemoteSearchResults(owner, query, results);\n",
    "                    searchDialog.dismiss();\n"
    "                    busy = false;\n"
    "                    refreshButtons();\n"
    "                    showRemoteSearchResults(owner, query, results);\n",
    "remote search success dialog close",
)
main = once(
    main,
    "                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;\n                    if (session == owner && !owner.isConnected()) {\n",
    "                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;\n"
    "                    searchDialog.dismiss();\n"
    "                    if (session == owner && !owner.isConnected()) {\n",
    "remote search failure dialog close",
)
MAIN.write_text(main, encoding="utf-8")
