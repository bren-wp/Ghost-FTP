#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MAIN = ROOT / "android/app/src/main/java/app/ghostftp/client/MainActivity.java"
FTP = ROOT / "android/app/src/main/java/app/ghostftp/client/FtpSession.java"


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly one marker, found {count}")
    return text.replace(old, new, 1)


main = MAIN.read_text(encoding="utf-8")
main = replace_once(
    main,
    "import android.text.InputType;\n",
    "import android.text.Editable;\nimport android.text.InputType;\nimport android.text.TextWatcher;\n",
    "android text imports",
)
main = replace_once(
    main,
    "import java.util.Deque;\nimport java.util.List;\nimport java.util.UUID;\n",
    "import java.util.Deque;\nimport java.util.HashSet;\nimport java.util.List;\nimport java.util.Set;\nimport java.util.UUID;\n",
    "java util imports",
)
main = replace_once(
    main,
    "    private final List<Button> navigationButtons = new ArrayList<>();\n",
    "    private final List<Button> navigationButtons = new ArrayList<>();\n"
    "    private final List<WorkspaceOps.Item> localVisibleItems = new ArrayList<>();\n"
    "    private final List<WorkspaceOps.Item> remoteVisibleItems = new ArrayList<>();\n",
    "visible workspace fields",
)
main = replace_once(
    main,
    "    private Button remoteDelete;\n    private Button remoteChmod;\n",
    "    private Button remoteDelete;\n    private Button remoteChmod;\n"
    "    private Button localFilter;\n    private Button localSort;\n    private Button localSearch;\n"
    "    private Button remoteFilter;\n    private Button remoteSort;\n    private Button remoteSearch;\n"
    "    private Button directoryCompare;\n    private Button remoteEdit;\n",
    "advanced workspace buttons",
)
main = replace_once(
    main,
    "    private boolean rememberEndpoint = true;\n    private boolean showFileSizes = true;\n",
    "    private boolean rememberEndpoint = true;\n    private boolean showFileSizes = true;\n"
    "    private String localFilterQuery = \"\";\n    private String remoteFilterQuery = \"\";\n"
    "    private WorkspaceOps.SortKey localSortKey = WorkspaceOps.SortKey.NAME;\n"
    "    private WorkspaceOps.SortKey remoteSortKey = WorkspaceOps.SortKey.NAME;\n"
    "    private boolean localSortAscending = true;\n    private boolean remoteSortAscending = true;\n"
    "    private volatile long advancedOperationGeneration;\n"
    "    private RemoteEditorState remoteEditorState;\n",
    "advanced workspace state",
)
main = replace_once(
    main,
    "        transferGeneration++;\n        transferActive = false;\n",
    "        transferGeneration++;\n        advancedOperationGeneration++;\n"
    "        RemoteEditorState editorState = remoteEditorState;\n"
    "        remoteEditorState = null;\n"
    "        if (editorState != null && editorState.dialog != null) editorState.dialog.dismiss();\n"
    "        transferActive = false;\n",
    "destroy advanced operation invalidation",
)
main = replace_once(
    main,
    "        refresh.setOnClickListener(v -> refreshLocal());\n\n        LinearLayout fileActions = row();\n",
    "        refresh.setOnClickListener(v -> refreshLocal());\n\n"
    "        LinearLayout viewActions = row();\n"
    "        localFilter = button(\"Filter\");\n"
    "        localSort = button(\"Sort: Name ↑\");\n"
    "        localSearch = button(\"Search\");\n"
    "        viewActions.addView(localFilter, weightedSpaced());\n"
    "        viewActions.addView(localSort, weightedSpaced());\n"
    "        viewActions.addView(localSearch, weightedSpaced());\n"
    "        card.addView(viewActions, matchWrap());\n"
    "        localFilter.setOnClickListener(v -> editLocalFilter());\n"
    "        localSort.setOnClickListener(v -> cycleLocalSort());\n"
    "        localSort.setOnLongClickListener(v -> { toggleLocalSortDirection(); return true; });\n"
    "        localSearch.setOnClickListener(v -> promptLocalRecursiveSearch());\n\n"
    "        LinearLayout fileActions = row();\n",
    "local filter sort search controls",
)
main = replace_once(
    main,
    "        localList.setOnItemLongClickListener((parent, view, position, id) -> {\n"
    "            if (busy || position < 0 || position >= localEntries.size()) return true;\n"
    "            selectedLocal = position;\n"
    "            renderLocal();\n"
    "            setStatus(\"Local item selected for file management: \" + localEntries.get(position).name);\n"
    "            return true;\n"
    "        });\n",
    "        localList.setOnItemLongClickListener((parent, view, position, id) -> {\n"
    "            int sourceIndex = localSourceIndex(position);\n"
    "            if (busy || sourceIndex < 0 || sourceIndex >= localEntries.size()) return true;\n"
    "            selectedLocal = sourceIndex;\n"
    "            renderLocal();\n"
    "            setStatus(\"Local item selected for file management: \" + localEntries.get(sourceIndex).name);\n"
    "            return true;\n"
    "        });\n",
    "local long press visible mapping",
)
main = replace_once(
    main,
    "        refresh.setOnClickListener(v -> refreshRemote(currentRemotePath));\n\n        LinearLayout primaryActions = row();\n",
    "        refresh.setOnClickListener(v -> refreshRemote(currentRemotePath));\n\n"
    "        LinearLayout viewActions = row();\n"
    "        remoteFilter = button(\"Filter\");\n"
    "        remoteSort = button(\"Sort: Name ↑\");\n"
    "        remoteSearch = button(\"Search\");\n"
    "        viewActions.addView(remoteFilter, weightedSpaced());\n"
    "        viewActions.addView(remoteSort, weightedSpaced());\n"
    "        viewActions.addView(remoteSearch, weightedSpaced());\n"
    "        card.addView(viewActions, matchWrap());\n"
    "        remoteFilter.setOnClickListener(v -> editRemoteFilter());\n"
    "        remoteSort.setOnClickListener(v -> cycleRemoteSort());\n"
    "        remoteSort.setOnLongClickListener(v -> { toggleRemoteSortDirection(); return true; });\n"
    "        remoteSearch.setOnClickListener(v -> promptRemoteRecursiveSearch());\n\n"
    "        LinearLayout primaryActions = row();\n",
    "remote filter sort search controls",
)
main = replace_once(
    main,
    "        remoteChmod = button(\"Permissions / CHMOD\");\n"
    "        remoteChmod.setOnClickListener(v -> chmodRemoteSelected());\n"
    "        card.addView(remoteChmod, matchWrapSpaced());\n\n"
    "        remoteList = new ListView(this);\n",
    "        LinearLayout advancedActions = row();\n"
    "        remoteChmod = button(\"Permissions / CHMOD\");\n"
    "        directoryCompare = button(\"Compare folders\");\n"
    "        remoteEdit = primaryButton(\"Remote Edit\");\n"
    "        advancedActions.addView(remoteChmod, weightedSpaced());\n"
    "        advancedActions.addView(directoryCompare, weightedSpaced());\n"
    "        advancedActions.addView(remoteEdit, weightedSpaced());\n"
    "        card.addView(advancedActions, matchWrap());\n"
    "        remoteChmod.setOnClickListener(v -> chmodRemoteSelected());\n"
    "        directoryCompare.setOnClickListener(v -> showDirectoryComparison());\n"
    "        remoteEdit.setOnClickListener(v -> openRemoteEditorSelected());\n\n"
    "        remoteList = new ListView(this);\n",
    "remote compare edit controls",
)
main = replace_once(
    main,
    "        remoteList.setOnItemLongClickListener((parent, view, position, id) -> {\n"
    "            if (busy || position < 0 || position >= remoteEntries.size()) return true;\n"
    "            selectedRemote = position;\n"
    "            renderRemote();\n"
    "            setStatus(\"Server item selected for file management: \" + remoteEntries.get(position).name);\n"
    "            return true;\n"
    "        });\n",
    "        remoteList.setOnItemLongClickListener((parent, view, position, id) -> {\n"
    "            int sourceIndex = remoteSourceIndex(position);\n"
    "            if (busy || sourceIndex < 0 || sourceIndex >= remoteEntries.size()) return true;\n"
    "            selectedRemote = sourceIndex;\n"
    "            renderRemote();\n"
    "            setStatus(\"Server item selected for file management: \" + remoteEntries.get(sourceIndex).name);\n"
    "            return true;\n"
    "        });\n",
    "remote long press visible mapping",
)
main = replace_once(
    main,
    "        String[] projection = {DocumentsContract.Document.COLUMN_DOCUMENT_ID, DocumentsContract.Document.COLUMN_DISPLAY_NAME,\n"
    "                DocumentsContract.Document.COLUMN_MIME_TYPE, DocumentsContract.Document.COLUMN_SIZE};\n",
    "        String[] projection = {DocumentsContract.Document.COLUMN_DOCUMENT_ID, DocumentsContract.Document.COLUMN_DISPLAY_NAME,\n"
    "                DocumentsContract.Document.COLUMN_MIME_TYPE, DocumentsContract.Document.COLUMN_SIZE,\n"
    "                DocumentsContract.Document.COLUMN_LAST_MODIFIED};\n",
    "local modified projection",
)
main = replace_once(
    main,
    "                long size = cursor.isNull(3) ? 0L : cursor.getLong(3);\n"
    "                result.add(new LocalEntry(id, name, DocumentsContract.Document.MIME_TYPE_DIR.equals(mime), size));\n",
    "                long size = cursor.isNull(3) ? 0L : cursor.getLong(3);\n"
    "                long modified = cursor.isNull(4) ? 0L : cursor.getLong(4);\n"
    "                result.add(new LocalEntry(id, name, DocumentsContract.Document.MIME_TYPE_DIR.equals(mime), size, modified));\n",
    "local modified entry",
)
main = replace_once(
    main,
    "    private void selectLocal(int position) {\n"
    "        if (busy || position < 0 || position >= localEntries.size()) return;\n"
    "        LocalEntry entry = localEntries.get(position);\n",
    "    private void selectLocal(int position) {\n"
    "        int sourceIndex = localSourceIndex(position);\n"
    "        if (busy || sourceIndex < 0 || sourceIndex >= localEntries.size()) return;\n"
    "        LocalEntry entry = localEntries.get(sourceIndex);\n",
    "local click visible mapping",
)
main = replace_once(
    main,
    "            selectedLocal = position;\n            renderLocal();\n        }\n    }\n\n    private void localUp()",
    "            selectedLocal = sourceIndex;\n            renderLocal();\n        }\n    }\n\n    private void localUp()",
    "local file selection source index",
)
main = replace_once(
    main,
    "    private void selectRemote(int position) {\n"
    "        if (busy || position < 0 || position >= remoteEntries.size()) return;\n"
    "        RemoteEntry entry = remoteEntries.get(position);\n",
    "    private void selectRemote(int position) {\n"
    "        int sourceIndex = remoteSourceIndex(position);\n"
    "        if (busy || sourceIndex < 0 || sourceIndex >= remoteEntries.size()) return;\n"
    "        RemoteEntry entry = remoteEntries.get(sourceIndex);\n",
    "remote click visible mapping",
)
main = replace_once(
    main,
    "            selectedRemote = position;\n            renderRemote();\n        }\n    }\n\n    private void remoteUp()",
    "            selectedRemote = sourceIndex;\n            renderRemote();\n        }\n    }\n\n    private void remoteUp()",
    "remote file selection source index",
)

advanced_methods = r'''
    private void editLocalFilter() {
        promptText("Local current-folder filter", localFilterQuery, "Name contains… (blank clears)", false, value -> {
            localFilterQuery = value == null ? "" : value.trim();
            selectedLocal = -1;
            renderLocal();
            setStatus(localFilterQuery.isEmpty() ? "Local filter cleared." : "Local filter applied: " + localFilterQuery);
        });
    }

    private void editRemoteFilter() {
        promptText("Server current-folder filter", remoteFilterQuery, "Name contains… (blank clears)", false, value -> {
            remoteFilterQuery = value == null ? "" : value.trim();
            selectedRemote = -1;
            renderRemote();
            setStatus(remoteFilterQuery.isEmpty() ? "Server filter cleared." : "Server filter applied: " + remoteFilterQuery);
        });
    }

    private void cycleLocalSort() {
        if (busy) return;
        localSortKey = WorkspaceOps.nextLocalSortKey(localSortKey);
        renderLocal();
        setStatus("Local sort: " + WorkspaceOps.sortLabel(localSortKey, localSortAscending));
    }

    private void toggleLocalSortDirection() {
        if (busy) return;
        localSortAscending = !localSortAscending;
        renderLocal();
        setStatus("Local sort: " + WorkspaceOps.sortLabel(localSortKey, localSortAscending));
    }

    private void cycleRemoteSort() {
        if (busy) return;
        remoteSortKey = WorkspaceOps.nextRemoteSortKey(remoteSortKey);
        renderRemote();
        setStatus("Server sort: " + WorkspaceOps.sortLabel(remoteSortKey, remoteSortAscending));
    }

    private void toggleRemoteSortDirection() {
        if (busy) return;
        remoteSortAscending = !remoteSortAscending;
        renderRemote();
        setStatus("Server sort: " + WorkspaceOps.sortLabel(remoteSortKey, remoteSortAscending));
    }

    private int localSourceIndex(int visiblePosition) {
        if (visiblePosition < 0 || visiblePosition >= localVisibleItems.size()) return -1;
        return localVisibleItems.get(visiblePosition).sourceIndex;
    }

    private int remoteSourceIndex(int visiblePosition) {
        if (visiblePosition < 0 || visiblePosition >= remoteVisibleItems.size()) return -1;
        return remoteVisibleItems.get(visiblePosition).sourceIndex;
    }

    private List<WorkspaceOps.Item> localWorkspaceItems() {
        List<WorkspaceOps.Item> result = new ArrayList<>();
        for (int i = 0; i < localEntries.size(); i++) {
            LocalEntry entry = localEntries.get(i);
            result.add(new WorkspaceOps.Item(i, entry.name, entry.directory, entry.size, entry.modifiedEpochMillis, ""));
        }
        return result;
    }

    private List<WorkspaceOps.Item> remoteWorkspaceItems() {
        List<WorkspaceOps.Item> result = new ArrayList<>();
        for (int i = 0; i < remoteEntries.size(); i++) {
            RemoteEntry entry = remoteEntries.get(i);
            result.add(new WorkspaceOps.Item(i, entry.name, entry.directory, entry.size, entry.modifiedEpochMillis, entry.permissions));
        }
        return result;
    }

    private void promptLocalRecursiveSearch() {
        if (busy || treeUri == null || rootDocumentId == null) return;
        promptText("Search local folders", "", "Name contains…", false, value -> {
            String query = value == null ? "" : value.trim();
            if (query.isEmpty()) {
                setStatus("Search text is required.");
                return;
            }
            runLocalRecursiveSearch(query);
        });
    }

    private void runLocalRecursiveSearch(String query) {
        Uri searchTree = treeUri;
        String searchRoot = rootDocumentId;
        if (busy || searchTree == null || searchRoot == null) return;
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Searching local folders within bounded safety limits…");
        io.execute(() -> {
            try {
                List<LocalSearchResult> results = new ArrayList<>();
                ArrayDeque<LocalSearchNode> queue = new ArrayDeque<>();
                queue.add(new LocalSearchNode(searchRoot, "", new ArrayList<>(), 0));
                int directories = 0;
                while (!queue.isEmpty()
                        && directories < WorkspaceOps.MAX_SEARCH_DIRECTORIES
                        && results.size() < WorkspaceOps.MAX_SEARCH_RESULTS) {
                    LocalSearchNode node = queue.removeFirst();
                    directories++;
                    List<LocalEntry> children = queryChildren(searchTree, node.documentId);
                    for (LocalEntry entry : children) {
                        String display = node.displayPath.isEmpty() ? entry.name : node.displayPath + "/" + entry.name;
                        if (WorkspaceOps.matchesSearch(entry.name, query)) {
                            results.add(new LocalSearchResult(node.documentId, node.ancestors, entry, display));
                            if (results.size() >= WorkspaceOps.MAX_SEARCH_RESULTS) break;
                        }
                        if (entry.directory && node.depth < WorkspaceOps.MAX_SEARCH_DEPTH) {
                            List<String> ancestors = new ArrayList<>(node.ancestors);
                            ancestors.add(node.documentId);
                            queue.addLast(new LocalSearchNode(entry.documentId, display, ancestors, node.depth + 1));
                        }
                    }
                }
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || treeUri == null || !treeUri.equals(searchTree)) return;
                    busy = false;
                    refreshButtons();
                    showLocalSearchResults(query, results);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Local recursive search failed: " + safeMessage(e));
                });
            }
        });
    }

    private void showLocalSearchResults(String query, List<LocalSearchResult> results) {
        if (results.isEmpty()) {
            setStatus("No local recursive-search results for: " + query);
            return;
        }
        String[] labels = new String[results.size()];
        for (int i = 0; i < results.size(); i++) {
            LocalSearchResult result = results.get(i);
            labels[i] = (result.entry.directory ? "DIR   " : "FILE  ") + result.displayPath;
        }
        new AlertDialog.Builder(this)
                .setTitle("Local search · " + results.size() + " result(s)")
                .setItems(labels, (dialog, which) -> navigateLocalSearchResult(results.get(which)))
                .setNegativeButton("Close", null)
                .show();
    }

    private void navigateLocalSearchResult(LocalSearchResult result) {
        Uri targetTree = treeUri;
        if (busy || targetTree == null) return;
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Opening local search result…");
        io.execute(() -> {
            try {
                List<LocalEntry> fresh = queryChildren(targetTree, result.parentDocumentId);
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || treeUri == null || !treeUri.equals(targetTree)) return;
                    currentDocumentId = result.parentDocumentId;
                    localParents.clear();
                    for (String ancestor : result.ancestors) localParents.push(ancestor);
                    localEntries.clear();
                    localEntries.addAll(fresh);
                    selectedLocal = findLocalByDocumentId(result.entry.documentId);
                    localFilterQuery = "";
                    renderLocal();
                    setBusy(false, "Opened local search result: " + result.displayPath);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Local search result is no longer available: " + safeMessage(e));
                });
            }
        });
    }

    private int findLocalByDocumentId(String documentId) {
        for (int i = 0; i < localEntries.size(); i++) {
            if (localEntries.get(i).documentId.equals(documentId)) return i;
        }
        return -1;
    }

    private void promptRemoteRecursiveSearch() {
        FtpSession current = session;
        if (busy || current == null || !current.isConnected()) return;
        promptText("Search server folders", "", "Name contains…", false, value -> {
            String query = value == null ? "" : value.trim();
            if (query.isEmpty()) {
                setStatus("Search text is required.");
                return;
            }
            runRemoteRecursiveSearch(current, query);
        });
    }

    private void runRemoteRecursiveSearch(FtpSession owner, String query) {
        if (busy || owner == null || session != owner || !owner.isConnected()) return;
        String searchRoot = currentRemotePath;
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Searching server folders within bounded safety limits…");
        io.execute(() -> {
            try {
                List<RemoteSearchResult> results = new ArrayList<>();
                ArrayDeque<RemoteSearchNode> queue = new ArrayDeque<>();
                Set<String> visited = new HashSet<>();
                queue.add(new RemoteSearchNode(searchRoot, 0));
                int directories = 0;
                while (!queue.isEmpty()
                        && directories < WorkspaceOps.MAX_SEARCH_DIRECTORIES
                        && results.size() < WorkspaceOps.MAX_SEARCH_RESULTS) {
                    if (session != owner || !owner.isConnected()) throw new IOException("Server connection changed during recursive search.");
                    RemoteSearchNode node = queue.removeFirst();
                    if (!visited.add(node.path)) continue;
                    directories++;
                    List<RemoteEntry> children = owner.list(node.path);
                    for (RemoteEntry entry : children) {
                        String childPath = FtpSession.joinRemote(node.path, entry.name);
                        if (WorkspaceOps.matchesSearch(entry.name, query)) {
                            results.add(new RemoteSearchResult(node.path, entry, childPath));
                            if (results.size() >= WorkspaceOps.MAX_SEARCH_RESULTS) break;
                        }
                        if (entry.directory && node.depth < WorkspaceOps.MAX_SEARCH_DEPTH) {
                            queue.addLast(new RemoteSearchNode(childPath, node.depth + 1));
                        }
                    }
                }
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || session != owner || !owner.isConnected()) return;
                    busy = false;
                    refreshButtons();
                    showRemoteSearchResults(owner, query, results);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    if (session == owner && !owner.isConnected()) {
                        session = null;
                        connectedIdentityKey = null;
                        remoteEntries.clear();
                        selectedRemote = -1;
                        currentRemotePath = "/";
                        renderRemote();
                    }
                    setBusy(false, "Server recursive search failed: " + safeMessage(e));
                });
            }
        });
    }

    private void showRemoteSearchResults(FtpSession owner, String query, List<RemoteSearchResult> results) {
        if (results.isEmpty()) {
            setStatus("No server recursive-search results for: " + query);
            return;
        }
        String[] labels = new String[results.size()];
        for (int i = 0; i < results.size(); i++) {
            RemoteSearchResult result = results.get(i);
            labels[i] = (result.entry.directory ? "DIR   " : "FILE  ") + result.displayPath;
        }
        new AlertDialog.Builder(this)
                .setTitle("Server search · " + results.size() + " result(s)")
                .setItems(labels, (dialog, which) -> navigateRemoteSearchResult(owner, results.get(which)))
                .setNegativeButton("Close", null)
                .show();
    }

    private void navigateRemoteSearchResult(FtpSession owner, RemoteSearchResult result) {
        if (busy || session != owner || !owner.isConnected()) return;
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Opening server search result…");
        io.execute(() -> {
            try {
                List<RemoteEntry> fresh = owner.list(result.parentPath);
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || session != owner || !owner.isConnected()) return;
                    currentRemotePath = result.parentPath;
                    remoteEntries.clear();
                    remoteEntries.addAll(fresh);
                    selectedRemote = findRemoteByName(result.entry.name);
                    remoteFilterQuery = "";
                    renderRemote();
                    setBusy(false, "Opened server search result: " + result.displayPath);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Server search result is no longer available: " + safeMessage(e));
                });
            }
        });
    }

    private int findRemoteByName(String name) {
        for (int i = 0; i < remoteEntries.size(); i++) {
            if (remoteEntries.get(i).name.equals(name)) return i;
        }
        return -1;
    }

    private void showDirectoryComparison() {
        if (busy || treeUri == null || currentDocumentId == null || session == null || !session.isConnected()) return;
        List<WorkspaceOps.Comparison> rows = WorkspaceOps.compareDirectories(localWorkspaceItems(), remoteWorkspaceItems());
        if (rows.isEmpty()) {
            setStatus("Both current folders are empty.");
            return;
        }
        String[] labels = new String[rows.size()];
        for (int i = 0; i < rows.size(); i++) {
            WorkspaceOps.Comparison row = rows.get(i);
            String marker;
            switch (row.difference) {
                case ONLY_LOCAL:
                    marker = "LOCAL ONLY";
                    break;
                case ONLY_REMOTE:
                    marker = "SERVER ONLY";
                    break;
                case DIFFERENT:
                    marker = "DIFFERENT";
                    break;
                case SAME:
                default:
                    marker = "SAME";
                    break;
            }
            labels[i] = marker + "   " + row.name + (row.canSynchronizeDirectoryNavigation() ? "   › open both" : "");
        }
        new AlertDialog.Builder(this)
                .setTitle("Directory comparison")
                .setItems(labels, (dialog, which) -> {
                    WorkspaceOps.Comparison row = rows.get(which);
                    if (row.canSynchronizeDirectoryNavigation()) {
                        synchronizedNavigateInto(row.name);
                    } else {
                        setStatus("Comparison: " + row.difference.name().replace('_', ' ') + " · " + row.name);
                    }
                })
                .setNegativeButton("Close", null)
                .show();
    }

    private void synchronizedNavigateInto(String name) {
        FtpSession owner = session;
        Uri ownerTree = treeUri;
        if (busy || owner == null || ownerTree == null || !owner.isConnected()) return;
        LocalEntry localDirectory = null;
        RemoteEntry remoteDirectory = null;
        for (LocalEntry entry : localEntries) if (entry.directory && entry.name.equals(name)) localDirectory = entry;
        for (RemoteEntry entry : remoteEntries) if (entry.directory && entry.name.equals(name)) remoteDirectory = entry;
        if (localDirectory == null || remoteDirectory == null) {
            setStatus("Synchronized navigation is unavailable because the paired directories changed.");
            return;
        }
        final LocalEntry localTarget = localDirectory;
        final String localParent = currentDocumentId;
        final String remoteParent = currentRemotePath;
        final String remoteTarget;
        try {
            remoteTarget = FtpSession.joinRemote(remoteParent, name);
        } catch (IOException e) {
            setStatus(e.getMessage());
            return;
        }
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Opening paired directories…");
        io.execute(() -> {
            try {
                List<LocalEntry> localFresh = queryChildren(ownerTree, localTarget.documentId);
                List<RemoteEntry> remoteFresh = owner.list(remoteTarget);
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || session != owner || treeUri == null || !treeUri.equals(ownerTree)) return;
                    if (!localParent.equals(currentDocumentId) || !remoteParent.equals(currentRemotePath)) return;
                    localParents.push(localParent);
                    currentDocumentId = localTarget.documentId;
                    currentRemotePath = remoteTarget;
                    localEntries.clear();
                    localEntries.addAll(localFresh);
                    remoteEntries.clear();
                    remoteEntries.addAll(remoteFresh);
                    selectedLocal = -1;
                    selectedRemote = -1;
                    localFilterQuery = "";
                    remoteFilterQuery = "";
                    renderLocal();
                    renderRemote();
                    setBusy(false, "Opened paired local/server directory: " + name);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Paired navigation was not committed: " + safeMessage(e));
                });
            }
        });
    }

    private void openRemoteEditorSelected() {
        FtpSession owner = session;
        if (busy || owner == null || selectedRemote < 0 || selectedRemote >= remoteEntries.size() || !owner.isConnected()) return;
        RemoteEntry entry = remoteEntries.get(selectedRemote);
        if (entry.directory) {
            setStatus("Remote Edit supports regular text files only.");
            return;
        }
        if (entry.size > WorkspaceOps.MAX_REMOTE_EDIT_BYTES) {
            setStatus("Remote Edit supports text files up to 1 MiB.");
            return;
        }
        final String path;
        try {
            path = FtpSession.joinRemote(currentRemotePath, entry.name);
        } catch (IOException e) {
            setStatus(e.getMessage());
            return;
        }
        long generation = ++advancedOperationGeneration;
        setBusy(true, "Opening Remote Edit with conflict-safe snapshot…");
        io.execute(() -> {
            try {
                RemoteTextDocument.Snapshot snapshot = RemoteEditIo.open(owner, path);
                runOnUiThread(() -> {
                    if (lifecycleDestroyed || generation != advancedOperationGeneration || session != owner || !owner.isConnected()) return;
                    showRemoteEditor(owner, path, entry.name, generation, snapshot);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (generation != advancedOperationGeneration || lifecycleDestroyed) return;
                    setBusy(false, "Remote Edit could not open file: " + safeMessage(e));
                });
            }
        });
    }

    private void showRemoteEditor(FtpSession owner, String path, String name, long generation, RemoteTextDocument.Snapshot snapshot) {
        EditText editor = field("Remote UTF-8 text", false);
        editor.setSingleLine(false);
        editor.setGravity(Gravity.TOP | Gravity.START);
        editor.setMinLines(16);
        editor.setText(snapshot.text);

        FrameLayout holder = new FrameLayout(this);
        int pad = dp(16);
        holder.setPadding(pad, dp(8), pad, 0);
        holder.addView(editor, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(420)));

        RemoteEditorState state = new RemoteEditorState(owner, path, name, generation, snapshot, editor);
        AlertDialog dialog = new AlertDialog.Builder(this)
                .setTitle("Remote Edit — " + name)
                .setView(holder)
                .setPositiveButton("Save", null)
                .setNeutralButton("Reload", null)
                .setNegativeButton("Close", null)
                .create();
        state.dialog = dialog;
        remoteEditorState = state;

        editor.addTextChangedListener(new TextWatcher() {
            @Override public void beforeTextChanged(CharSequence s, int start, int count, int after) { }
            @Override public void onTextChanged(CharSequence s, int start, int before, int count) { }
            @Override public void afterTextChanged(Editable s) {
                if (state.applying || remoteEditorState != state) return;
                state.dirty = true;
                state.dialog.setTitle("Remote Edit • modified — " + state.name);
            }
        });

        dialog.setOnShowListener(ignored -> {
            dialog.getButton(AlertDialog.BUTTON_POSITIVE).setOnClickListener(v -> saveRemoteEditor(state));
            dialog.getButton(AlertDialog.BUTTON_NEUTRAL).setOnClickListener(v -> reloadRemoteEditor(state));
            dialog.getButton(AlertDialog.BUTTON_NEGATIVE).setOnClickListener(v -> closeRemoteEditor(state));
            setStatus("Remote Edit opened with SHA-256 conflict detection and read-back verification.");
            refreshButtons();
        });
        dialog.setOnDismissListener(ignored -> {
            if (remoteEditorState != state) return;
            remoteEditorState = null;
            advancedOperationGeneration++;
            busy = false;
            setStatus("Remote Edit closed.");
            refreshButtons();
        });
        dialog.show();
    }

    private void saveRemoteEditor(RemoteEditorState state) {
        if (!remoteEditorUsable(state) || state.running) return;
        state.running = true;
        setRemoteEditorButtonsEnabled(state, false);
        String text = state.editor.getText().toString();
        setStatus("Remote Edit is checking for conflicts and saving…");
        io.execute(() -> {
            try {
                RemoteTextDocument.Snapshot saved = RemoteEditIo.save(
                        state.owner, state.path, state.snapshot.sha256, state.snapshot.lineEnding, text);
                runOnUiThread(() -> {
                    if (!remoteEditorUsable(state)) return;
                    applyRemoteEditorSnapshot(state, saved);
                    state.running = false;
                    setRemoteEditorButtonsEnabled(state, true);
                    setStatus("Remote Edit saved and verified by read-back: " + state.name);
                    refreshRemoteAfterEditor(state.owner);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (remoteEditorState != state || lifecycleDestroyed) return;
                    state.running = false;
                    setRemoteEditorButtonsEnabled(state, true);
                    setStatus("Remote Edit save blocked: " + safeMessage(e));
                });
            }
        });
    }

    private void reloadRemoteEditor(RemoteEditorState state) {
        if (!remoteEditorUsable(state) || state.running) return;
        state.running = true;
        setRemoteEditorButtonsEnabled(state, false);
        setStatus("Reloading Remote Edit from server…");
        io.execute(() -> {
            try {
                RemoteTextDocument.Snapshot fresh = RemoteEditIo.reload(state.owner, state.path);
                runOnUiThread(() -> {
                    if (!remoteEditorUsable(state)) return;
                    applyRemoteEditorSnapshot(state, fresh);
                    state.running = false;
                    setRemoteEditorButtonsEnabled(state, true);
                    setStatus("Remote Edit reloaded from server: " + state.name);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (remoteEditorState != state || lifecycleDestroyed) return;
                    state.running = false;
                    setRemoteEditorButtonsEnabled(state, true);
                    setStatus("Remote Edit reload failed: " + safeMessage(e));
                });
            }
        });
    }

    private void closeRemoteEditor(RemoteEditorState state) {
        if (remoteEditorState != state || state.dialog == null || state.running) return;
        if (!state.dirty) {
            state.dialog.dismiss();
            return;
        }
        new AlertDialog.Builder(this)
                .setTitle("Discard Remote Edit changes?")
                .setMessage("Unsaved changes to “" + state.name + "” will be discarded.")
                .setNegativeButton("Keep editing", null)
                .setPositiveButton("Discard", (dialog, which) -> state.dialog.dismiss())
                .show();
    }

    private boolean remoteEditorUsable(RemoteEditorState state) {
        return !lifecycleDestroyed
                && remoteEditorState == state
                && state.generation == advancedOperationGeneration
                && session == state.owner
                && state.owner.isConnected();
    }

    private void applyRemoteEditorSnapshot(RemoteEditorState state, RemoteTextDocument.Snapshot snapshot) {
        state.applying = true;
        state.snapshot = snapshot;
        state.editor.setText(snapshot.text);
        state.editor.setSelection(state.editor.length());
        state.dirty = false;
        state.dialog.setTitle("Remote Edit — " + state.name);
        state.applying = false;
    }

    private void setRemoteEditorButtonsEnabled(RemoteEditorState state, boolean enabled) {
        if (state.dialog == null) return;
        if (state.dialog.getButton(AlertDialog.BUTTON_POSITIVE) != null) state.dialog.getButton(AlertDialog.BUTTON_POSITIVE).setEnabled(enabled);
        if (state.dialog.getButton(AlertDialog.BUTTON_NEUTRAL) != null) state.dialog.getButton(AlertDialog.BUTTON_NEUTRAL).setEnabled(enabled);
        if (state.dialog.getButton(AlertDialog.BUTTON_NEGATIVE) != null) state.dialog.getButton(AlertDialog.BUTTON_NEGATIVE).setEnabled(enabled);
    }

    private void refreshRemoteAfterEditor(FtpSession owner) {
        if (remoteEditorState == null || session != owner || !owner.isConnected()) return;
        String directory = currentRemotePath;
        io.execute(() -> {
            try {
                List<RemoteEntry> fresh = owner.list(directory);
                runOnUiThread(() -> {
                    if (session != owner || !owner.isConnected() || !directory.equals(currentRemotePath)) return;
                    String selectedName = selectedRemote >= 0 && selectedRemote < remoteEntries.size() ? remoteEntries.get(selectedRemote).name : "";
                    remoteEntries.clear();
                    remoteEntries.addAll(fresh);
                    selectedRemote = findRemoteByName(selectedName);
                    renderRemote();
                });
            } catch (Exception ignored) {
                // Save/read-back verification already succeeded; metadata refresh is best-effort.
            }
        });
    }
'''
main = replace_once(
    main,
    "    private void createLocalDirectory() {\n",
    advanced_methods + "\n    private void createLocalDirectory() {\n",
    "advanced workspace methods",
)

main = replace_once(
    main,
    "    private void renderLocal() {\n"
    "        if (localPath != null) localPath.setText(displayLocalPath());\n"
    "        if (bookmarkLocalCurrent != null) bookmarkLocalCurrent.setText(displayLocalPath());\n"
    "        List<String> labels = new ArrayList<>();\n"
    "        for (int i = 0; i < localEntries.size(); i++) {\n"
    "            LocalEntry e = localEntries.get(i);\n"
    "            String marker = i == selectedLocal ? \"●  \" : \"   \";\n"
    "            String type = e.directory ? \"DIR   \" : \"FILE  \";\n"
    "            String size = !e.directory && showFileSizes ? \"   \" + TransferProgress.formatBytes(e.size) : \"\";\n"
    "            labels.add(marker + type + e.name + size);\n"
    "        }\n"
    "        if (localList != null) localList.setAdapter(GhostTheme.listAdapter(this, labels));\n"
    "        refreshButtons();\n"
    "    }\n",
    "    private void renderLocal() {\n"
    "        if (localPath != null) localPath.setText(displayLocalPath());\n"
    "        if (bookmarkLocalCurrent != null) bookmarkLocalCurrent.setText(displayLocalPath());\n"
    "        localVisibleItems.clear();\n"
    "        localVisibleItems.addAll(WorkspaceOps.filterAndSort(localWorkspaceItems(), localFilterQuery, localSortKey, localSortAscending));\n"
    "        List<String> labels = new ArrayList<>();\n"
    "        for (WorkspaceOps.Item visible : localVisibleItems) {\n"
    "            LocalEntry e = localEntries.get(visible.sourceIndex);\n"
    "            String marker = visible.sourceIndex == selectedLocal ? \"●  \" : \"   \";\n"
    "            String type = e.directory ? \"DIR   \" : \"FILE  \";\n"
    "            String size = !e.directory && showFileSizes ? \"   \" + TransferProgress.formatBytes(e.size) : \"\";\n"
    "            labels.add(marker + type + e.name + size);\n"
    "        }\n"
    "        if (localList != null) localList.setAdapter(GhostTheme.listAdapter(this, labels));\n"
    "        if (localFilter != null) localFilter.setText(localFilterQuery.isEmpty() ? \"Filter\" : \"Filter: \" + localFilterQuery);\n"
    "        if (localSort != null) localSort.setText(\"Sort: \" + WorkspaceOps.sortLabel(localSortKey, localSortAscending));\n"
    "        refreshButtons();\n"
    "    }\n",
    "render local visible model",
)
main = replace_once(
    main,
    "    private void renderRemote() {\n"
    "        if (remotePath != null) remotePath.setText(currentRemotePath);\n"
    "        if (bookmarkRemoteCurrent != null) bookmarkRemoteCurrent.setText(currentRemotePath);\n"
    "        List<String> labels = new ArrayList<>();\n"
    "        for (int i = 0; i < remoteEntries.size(); i++) {\n"
    "            RemoteEntry e = remoteEntries.get(i);\n"
    "            String marker = i == selectedRemote ? \"●  \" : \"   \";\n"
    "            String type = e.directory ? \"DIR   \" : \"FILE  \";\n"
    "            String size = !e.directory && showFileSizes ? \"   \" + TransferProgress.formatBytes(e.size) : \"\";\n"
    "            labels.add(marker + type + e.name + size);\n"
    "        }\n"
    "        if (remoteList != null) remoteList.setAdapter(GhostTheme.listAdapter(this, labels));\n"
    "        refreshButtons();\n"
    "    }\n",
    "    private void renderRemote() {\n"
    "        if (remotePath != null) remotePath.setText(currentRemotePath);\n"
    "        if (bookmarkRemoteCurrent != null) bookmarkRemoteCurrent.setText(currentRemotePath);\n"
    "        remoteVisibleItems.clear();\n"
    "        remoteVisibleItems.addAll(WorkspaceOps.filterAndSort(remoteWorkspaceItems(), remoteFilterQuery, remoteSortKey, remoteSortAscending));\n"
    "        List<String> labels = new ArrayList<>();\n"
    "        for (WorkspaceOps.Item visible : remoteVisibleItems) {\n"
    "            RemoteEntry e = remoteEntries.get(visible.sourceIndex);\n"
    "            String marker = visible.sourceIndex == selectedRemote ? \"●  \" : \"   \";\n"
    "            String type = e.directory ? \"DIR   \" : \"FILE  \";\n"
    "            String size = !e.directory && showFileSizes ? \"   \" + TransferProgress.formatBytes(e.size) : \"\";\n"
    "            String permissions = e.permissions.isEmpty() ? \"\" : \"   [\" + e.permissions + \"]\";\n"
    "            labels.add(marker + type + e.name + size + permissions);\n"
    "        }\n"
    "        if (remoteList != null) remoteList.setAdapter(GhostTheme.listAdapter(this, labels));\n"
    "        if (remoteFilter != null) remoteFilter.setText(remoteFilterQuery.isEmpty() ? \"Filter\" : \"Filter: \" + remoteFilterQuery);\n"
    "        if (remoteSort != null) remoteSort.setText(\"Sort: \" + WorkspaceOps.sortLabel(remoteSortKey, remoteSortAscending));\n"
    "        refreshButtons();\n"
    "    }\n",
    "render remote visible model",
)
main = replace_once(
    main,
    "        remoteChmod.setEnabled(!busy && connected && selectedRemoteItem);\n"
    "        saveSite.setEnabled(!busy && !connected);\n",
    "        remoteChmod.setEnabled(!busy && connected && selectedRemoteItem);\n"
    "        localFilter.setEnabled(!busy && treeUri != null);\n"
    "        localSort.setEnabled(!busy && !localEntries.isEmpty());\n"
    "        localSearch.setEnabled(!busy && treeUri != null && rootDocumentId != null);\n"
    "        remoteFilter.setEnabled(!busy && connected);\n"
    "        remoteSort.setEnabled(!busy && connected && !remoteEntries.isEmpty());\n"
    "        remoteSearch.setEnabled(!busy && connected);\n"
    "        directoryCompare.setEnabled(!busy && connected && treeUri != null && currentDocumentId != null);\n"
    "        remoteEdit.setEnabled(!busy && connected && selectedRemoteFile\n"
    "                && remoteEntries.get(selectedRemote).size <= WorkspaceOps.MAX_REMOTE_EDIT_BYTES);\n"
    "        saveSite.setEnabled(!busy && !connected);\n",
    "advanced workspace button states",
)
main = replace_once(
    main,
    "                remoteCreateDirectory, remoteRename, remoteDelete, remoteChmod,\n"
    "                saveSite, deleteSite,\n",
    "                remoteCreateDirectory, remoteRename, remoteDelete, remoteChmod,\n"
    "                localFilter, localSort, localSearch, remoteFilter, remoteSort, remoteSearch,\n"
    "                directoryCompare, remoteEdit, saveSite, deleteSite,\n",
    "advanced enabled alpha",
)
main = replace_once(
    main,
    "    private static final class TransferAttempt {\n",
    r'''    private static final class LocalSearchNode {
        final String documentId;
        final String displayPath;
        final List<String> ancestors;
        final int depth;

        LocalSearchNode(String documentId, String displayPath, List<String> ancestors, int depth) {
            this.documentId = documentId;
            this.displayPath = displayPath;
            this.ancestors = new ArrayList<>(ancestors);
            this.depth = depth;
        }
    }

    private static final class LocalSearchResult {
        final String parentDocumentId;
        final List<String> ancestors;
        final LocalEntry entry;
        final String displayPath;

        LocalSearchResult(String parentDocumentId, List<String> ancestors, LocalEntry entry, String displayPath) {
            this.parentDocumentId = parentDocumentId;
            this.ancestors = new ArrayList<>(ancestors);
            this.entry = entry;
            this.displayPath = displayPath;
        }
    }

    private static final class RemoteSearchNode {
        final String path;
        final int depth;

        RemoteSearchNode(String path, int depth) {
            this.path = path;
            this.depth = depth;
        }
    }

    private static final class RemoteSearchResult {
        final String parentPath;
        final RemoteEntry entry;
        final String displayPath;

        RemoteSearchResult(String parentPath, RemoteEntry entry, String displayPath) {
            this.parentPath = parentPath;
            this.entry = entry;
            this.displayPath = displayPath;
        }
    }

    private static final class RemoteEditorState {
        final FtpSession owner;
        final String path;
        final String name;
        final long generation;
        final EditText editor;
        RemoteTextDocument.Snapshot snapshot;
        AlertDialog dialog;
        boolean applying;
        boolean dirty;
        boolean running;

        RemoteEditorState(FtpSession owner, String path, String name, long generation,
                          RemoteTextDocument.Snapshot snapshot, EditText editor) {
            this.owner = owner;
            this.path = path;
            this.name = name;
            this.generation = generation;
            this.snapshot = snapshot;
            this.editor = editor;
        }
    }

    private static final class TransferAttempt {
''',
    "advanced nested state",
)
main = replace_once(
    main,
    "    private static final class LocalEntry {\n"
    "        final String documentId;\n"
    "        final String name;\n"
    "        final boolean directory;\n"
    "        final long size;\n\n"
    "        LocalEntry(String documentId, String name, boolean directory, long size) {\n"
    "            this.documentId = documentId;\n"
    "            this.name = name;\n"
    "            this.directory = directory;\n"
    "            this.size = size;\n"
    "        }\n"
    "    }\n",
    "    private static final class LocalEntry {\n"
    "        final String documentId;\n"
    "        final String name;\n"
    "        final boolean directory;\n"
    "        final long size;\n"
    "        final long modifiedEpochMillis;\n\n"
    "        LocalEntry(String documentId, String name, boolean directory, long size, long modifiedEpochMillis) {\n"
    "            this.documentId = documentId;\n"
    "            this.name = name;\n"
    "            this.directory = directory;\n"
    "            this.size = size;\n"
    "            this.modifiedEpochMillis = Math.max(0L, modifiedEpochMillis);\n"
    "        }\n"
    "    }\n",
    "local entry modified metadata",
)
MAIN.write_text(main, encoding="utf-8")

ftp = FTP.read_text(encoding="utf-8")
ftp = replace_once(
    ftp,
    "import java.nio.charset.StandardCharsets;\n",
    "import java.nio.charset.StandardCharsets;\n"
    "import java.time.LocalDateTime;\nimport java.time.ZoneOffset;\n"
    "import java.time.format.DateTimeFormatter;\nimport java.time.format.DateTimeParseException;\n",
    "ftp time imports",
)
ftp = replace_once(
    ftp,
    "    private static final int MAX_DIRECTORY_ENTRIES = 10000;\n",
    "    private static final int MAX_DIRECTORY_ENTRIES = 10000;\n"
    "    private static final DateTimeFormatter MLSD_TIMESTAMP = DateTimeFormatter.ofPattern(\"yyyyMMddHHmmss\", Locale.ROOT);\n",
    "ftp timestamp formatter",
)
old_parse = '''    private static RemoteEntry parseMlsd(String line) {
        int split = line.indexOf(' ');
        if (split <= 0 || split + 1 >= line.length()) {
            return null;
        }
        String facts = line.substring(0, split).toLowerCase(Locale.ROOT);
        String name = line.substring(split + 1).trim();
        if (name.isEmpty() || ".".equals(name) || "..".equals(name) || facts.contains("type=cdir") || facts.contains("type=pdir")) {
            return null;
        }
        boolean directory = facts.contains("type=dir");
        long size = 0L;
        for (String fact : facts.split(";")) {
            if (fact.startsWith("size=")) {
                try {
                    size = Long.parseLong(fact.substring(5));
                } catch (NumberFormatException ignored) {
                    size = 0L;
                }
            }
        }
        return new RemoteEntry(name, directory, size);
    }
'''
new_parse = '''    static RemoteEntry parseMlsd(String line) {
        int split = line.indexOf(' ');
        if (split <= 0 || split + 1 >= line.length()) {
            return null;
        }
        String facts = line.substring(0, split).toLowerCase(Locale.ROOT);
        String name = line.substring(split + 1).trim();
        if (name.isEmpty() || ".".equals(name) || "..".equals(name) || facts.contains("type=cdir") || facts.contains("type=pdir")) {
            return null;
        }
        boolean directory = facts.contains("type=dir");
        long size = 0L;
        long modified = 0L;
        String permissions = "";
        for (String fact : facts.split(";")) {
            if (fact.startsWith("size=")) {
                try {
                    size = Long.parseLong(fact.substring(5));
                } catch (NumberFormatException ignored) {
                    size = 0L;
                }
            } else if (fact.startsWith("modify=")) {
                modified = parseMlsdTimestamp(fact.substring(7));
            } else if (fact.startsWith("unix.mode=")) {
                permissions = fact.substring(10).trim();
            }
        }
        return new RemoteEntry(name, directory, size, modified, permissions);
    }

    static long parseMlsdTimestamp(String value) {
        String text = value == null ? "" : value.trim();
        if (text.length() < 14) return 0L;
        String seconds = text.substring(0, 14);
        for (int i = 0; i < seconds.length(); i++) {
            if (seconds.charAt(i) < '0' || seconds.charAt(i) > '9') return 0L;
        }
        try {
            return LocalDateTime.parse(seconds, MLSD_TIMESTAMP).toInstant(ZoneOffset.UTC).toEpochMilli();
        } catch (DateTimeParseException e) {
            return 0L;
        }
    }
'''
ftp = replace_once(ftp, old_parse, new_parse, "ftp MLSD metadata parser")
FTP.write_text(ftp, encoding="utf-8")

print("Android advanced workspace patch applied successfully")
