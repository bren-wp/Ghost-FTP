from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MAIN = ROOT / "macos/Sources/GhostFTPApp/main.swift"
PARITY = ROOT / "macos/PARITY.md"


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly one match, found {count}")
    return text.replace(old, new, 1)


source = MAIN.read_text(encoding="utf-8")

source = replace_once(
    source,
    '''private struct MutationResult {
    let changed: Bool
    let status: String
    let error: String
}
''',
    '''private struct MutationResult {
    let changed: Bool
    let status: String
    let error: String
}

private struct RecursiveSearchResult {
    let name: String
    let parent: String
    let path: String
    let size: Int64
    let isDirectory: Bool
    let isSymlink: Bool
    let modifiedUnix: Int64
}
''',
    "recursive search result model",
)

source = replace_once(
    source,
    '''    private var remoteEditBusy = false
    private var localFilterBusy = false
    private var remoteFilterBusy = false
''',
    '''    private var remoteEditBusy = false
    private var localFilterBusy = false
    private var remoteFilterBusy = false
    private var recursiveSearchGeneration = 0
    private var recursiveSearchBusy = false
    private var recursiveSearchController: RecursiveSearchWindowController?
''',
    "recursive search state",
)

source = replace_once(
    source,
    '''    private let localDeleteButton = NSButton(title: "Delete", target: nil, action: nil)
    private let localFilterButton = NSButton(title: "Filter", target: nil, action: nil)
    private let uploadButton = NSButton(title: "Upload", target: nil, action: nil)
''',
    '''    private let localDeleteButton = NSButton(title: "Delete", target: nil, action: nil)
    private let localFilterButton = NSButton(title: "Filter", target: nil, action: nil)
    private let localSearchButton = NSButton(title: "Search", target: nil, action: nil)
    private let uploadButton = NSButton(title: "Upload", target: nil, action: nil)
''',
    "local search button",
)

source = replace_once(
    source,
    '''    private let remoteEditButton = NSButton(title: "Edit", target: nil, action: nil)
    private let remoteFilterButton = NSButton(title: "Filter", target: nil, action: nil)
    private let downloadButton = NSButton(title: "Download", target: nil, action: nil)
''',
    '''    private let remoteEditButton = NSButton(title: "Edit", target: nil, action: nil)
    private let remoteFilterButton = NSButton(title: "Filter", target: nil, action: nil)
    private let remoteSearchButton = NSButton(title: "Search", target: nil, action: nil)
    private let downloadButton = NSButton(title: "Download", target: nil, action: nil)
''',
    "remote search button",
)

source = replace_once(
    source,
    '''    func applicationWillTerminate(_ notification: Notification) {
        remoteEditGeneration += 1
        remoteEditBusy = false
        GhostFTPRemoteEditClear()
''',
    '''    func applicationWillTerminate(_ notification: Notification) {
        recursiveSearchGeneration += 1
        recursiveSearchBusy = false
        GhostFTPCancelRecursiveSearch()
        GhostFTPClearRecursiveSearch()
        recursiveSearchController?.closeSilently()
        recursiveSearchController = nil
        remoteEditGeneration += 1
        remoteEditBusy = false
        GhostFTPRemoteEditClear()
''',
    "termination cancellation",
)

source = replace_once(
    source,
    '''            buttons: [localChooseButton, localUpButton, localRefreshButton, localNewFolderButton, localRenameButton, localDeleteButton, localFilterButton, uploadButton]
''',
    '''            buttons: [localChooseButton, localUpButton, localRefreshButton, localNewFolderButton, localRenameButton, localDeleteButton, localFilterButton, localSearchButton, uploadButton]
''',
    "local toolbar",
)

source = replace_once(
    source,
    '''            buttons: [remoteUpButton, remoteRefreshButton, remoteNewFolderButton, remoteRenameButton, remoteDeleteButton, remotePermissionsButton, remoteEditButton, remoteFilterButton, downloadButton]
''',
    '''            buttons: [remoteUpButton, remoteRefreshButton, remoteNewFolderButton, remoteRenameButton, remoteDeleteButton, remotePermissionsButton, remoteEditButton, remoteFilterButton, remoteSearchButton, downloadButton]
''',
    "remote toolbar",
)

source = replace_once(
    source,
    '''        localFilterButton.target = self
        localFilterButton.action = #selector(localFilterTapped)
        uploadButton.target = self
''',
    '''        localFilterButton.target = self
        localFilterButton.action = #selector(localFilterTapped)
        localSearchButton.target = self
        localSearchButton.action = #selector(localRecursiveSearchTapped)
        uploadButton.target = self
''',
    "local search action",
)

source = replace_once(
    source,
    '''        remoteFilterButton.target = self
        remoteFilterButton.action = #selector(remoteFilterTapped)
        downloadButton.target = self
''',
    '''        remoteFilterButton.target = self
        remoteFilterButton.action = #selector(remoteFilterTapped)
        remoteSearchButton.target = self
        remoteSearchButton.action = #selector(remoteRecursiveSearchTapped)
        downloadButton.target = self
''',
    "remote search action",
)

source = replace_once(
    source,
    '''    @objc private func disconnectTapped() {
        remoteEditGeneration += 1
''',
    '''    @objc private func disconnectTapped() {
        GhostFTPCancelRecursiveSearch()
        recursiveSearchGeneration += 1
        recursiveSearchBusy = false
        recursiveSearchController?.closeSilently()
        recursiveSearchController = nil
        GhostFTPClearRecursiveSearch()
        remoteEditGeneration += 1
''',
    "disconnect search cancellation",
)

search_methods = r'''
    @objc private func localRecursiveSearchTapped() {
        guard engineReady, !localMutationBusy, !localFilterBusy, !recursiveSearchBusy else { return }
        guard let query = promptRecursiveSearchQuery(title: "Ghost FTP — Local Recursive Search") else { return }
        startRecursiveSearch(remote: false, query: query)
    }

    @objc private func remoteRecursiveSearchTapped() {
        guard engineReady,
              GhostFTPIsConnected() == 1,
              !remoteMutationBusy,
              !remoteEditBusy,
              !remoteFilterBusy,
              !recursiveSearchBusy else { return }
        guard let query = promptRecursiveSearchQuery(title: "Ghost FTP — Remote Recursive Search") else { return }
        startRecursiveSearch(remote: true, query: query)
    }

    private func promptRecursiveSearchQuery(title: String) -> String? {
        let alert = NSAlert()
        alert.messageText = title
        alert.informativeText = "Search recursively from the currently visible folder. The shared engine enforces bounded depth, item, result and time limits."
        alert.addButton(withTitle: "Search")
        alert.addButton(withTitle: "Cancel")
        let input = NSTextField(string: "")
        input.placeholderString = "Name contains…"
        input.frame = NSRect(x: 0, y: 0, width: 360, height: 24)
        alert.accessoryView = input
        window.makeFirstResponder(input)
        guard alert.runModal() == .alertFirstButtonReturn else { return nil }
        let query = input.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !query.isEmpty else {
            showError("Search query cannot be empty.")
            return nil
        }
        return query
    }

    private func startRecursiveSearch(remote: Bool, query: String) {
        guard !recursiveSearchBusy else { return }
        let base = remote ? remoteCurrent : localCurrent
        let navigationGeneration = remote ? remoteNavigationGeneration : localNavigationGeneration
        recursiveSearchGeneration += 1
        let searchGeneration = recursiveSearchGeneration
        recursiveSearchBusy = true
        statusLabel.stringValue = remote ? "Searching remote folders…" : "Searching local folders…"

        let controller = RecursiveSearchWindowController(
            title: remote ? "Remote Recursive Search" : "Local Recursive Search",
            root: base,
            query: query,
            onCancel: { [weak self] in
                guard let self, searchGeneration == self.recursiveSearchGeneration else { return }
                GhostFTPCancelRecursiveSearch()
                self.statusLabel.stringValue = "Cancelling recursive search…"
            },
            onClose: { [weak self] in
                guard let self, searchGeneration == self.recursiveSearchGeneration else { return }
                if self.recursiveSearchBusy {
                    GhostFTPCancelRecursiveSearch()
                    self.statusLabel.stringValue = "Cancelling recursive search…"
                } else {
                    GhostFTPClearRecursiveSearch()
                }
                self.recursiveSearchController = nil
                self.updateWorkspaceControls()
            },
            onNavigate: { [weak self] index in
                self?.navigateRecursiveSearchResult(remote: remote, index: index)
            }
        )
        recursiveSearchController = controller
        controller.showWindow(nil)
        controller.window?.center()
        controller.window?.makeKeyAndOrderFront(nil)
        updateWorkspaceControls()

        engineQueue.async { [weak self] in
            let baseValue = CStringBox(base)
            let queryValue = CStringBox(query)
            let code: Int32
            if remote {
                code = Int32(GhostFTPSearchRemoteRecursive(baseValue.pointer, queryValue.pointer))
            } else {
                code = Int32(GhostFTPSearchLocalRecursive(baseValue.pointer, queryValue.pointer))
            }
            let results = self?.readRecursiveSearchResults() ?? []
            let visited = max(0, Int(GhostFTPSearchVisited()))
            let stopReason = bridgeString(GhostFTPSearchStopReason())
            let message = code == 0 ? bridgeString(GhostFTPLastError()) : ""
            DispatchQueue.main.async {
                guard let self, searchGeneration == self.recursiveSearchGeneration else { return }
                self.recursiveSearchBusy = false
                let snapshotIsCurrent = remote
                    ? navigationGeneration == self.remoteNavigationGeneration && self.remoteCurrent == base && GhostFTPIsConnected() == 1
                    : navigationGeneration == self.localNavigationGeneration && self.localCurrent == base
                if !snapshotIsCurrent {
                    self.recursiveSearchController?.finish(results: [], status: "Folder changed; search results were discarded.")
                    self.statusLabel.stringValue = "Recursive search discarded after navigation."
                    GhostFTPClearRecursiveSearch()
                    self.updateWorkspaceControls()
                    return
                }
                switch code {
                case 1:
                    var status = "Found \(results.count) result(s) after visiting \(visited) item(s)."
                    if !stopReason.isEmpty { status += " Limit: \(stopReason)." }
                    self.recursiveSearchController?.finish(results: results, status: status)
                    self.statusLabel.stringValue = "Recursive search complete: \(results.count) result(s)."
                case 2:
                    self.recursiveSearchController?.finish(results: results, status: "Search cancelled. Partial results: \(results.count).")
                    self.statusLabel.stringValue = "Recursive search cancelled."
                default:
                    let error = message.isEmpty ? "Recursive search failed." : message
                    self.recursiveSearchController?.finish(results: [], status: error)
                    self.statusLabel.stringValue = "Recursive search failed."
                }
                self.updateWorkspaceControls()
            }
        }
    }

    private func readRecursiveSearchResults() -> [RecursiveSearchResult] {
        let count = max(0, Int(GhostFTPSearchResultCount()))
        return (0..<count).map { index in
            let cIndex = CInt(index)
            return RecursiveSearchResult(
                name: bridgeString(GhostFTPSearchResultName(cIndex)),
                parent: bridgeString(GhostFTPSearchResultParent(cIndex)),
                path: bridgeString(GhostFTPSearchResultPath(cIndex)),
                size: Int64(GhostFTPSearchResultSize(cIndex)),
                isDirectory: GhostFTPSearchResultIsDirectory(cIndex) == 1,
                isSymlink: GhostFTPSearchResultIsSymlink(cIndex) == 1,
                modifiedUnix: Int64(GhostFTPSearchResultModifiedUnix(cIndex))
            )
        }
    }

    private func navigateRecursiveSearchResult(remote: Bool, index: Int) {
        guard !recursiveSearchBusy,
              let controller = recursiveSearchController,
              let result = controller.result(at: index) else { return }
        let parent = result.parent
        recursiveSearchGeneration += 1
        controller.closeSilently()
        recursiveSearchController = nil
        GhostFTPClearRecursiveSearch()
        if remote {
            guard GhostFTPIsConnected() == 1 else {
                statusLabel.stringValue = "Connection changed; search result was not opened."
                updateWorkspaceControls()
                return
            }
            remoteFilterQuery = ""
            statusLabel.stringValue = "Navigating to search result…"
            refreshRemote(parent)
        } else {
            localFilterQuery = ""
            statusLabel.stringValue = "Navigating to search result…"
            refreshLocal(parent)
        }
    }
'''

source = replace_once(
    source,
    '''    @objc private func remoteFilterTapped() {
        guard engineReady, GhostFTPIsConnected() == 1, !remoteMutationBusy, !remoteEditBusy, !remoteFilterBusy else { return }
        guard let query = promptCurrentFolderFilter(title: "Ghost FTP — Remote", current: remoteFilterQuery) else { return }
        applyCurrentFolderFilter(remote: true, query: query)
    }

    private func promptCurrentFolderFilter(title: String, current: String) -> String? {
''',
    '''    @objc private func remoteFilterTapped() {
        guard engineReady, GhostFTPIsConnected() == 1, !remoteMutationBusy, !remoteEditBusy, !remoteFilterBusy else { return }
        guard let query = promptCurrentFolderFilter(title: "Ghost FTP — Remote", current: remoteFilterQuery) else { return }
        applyCurrentFolderFilter(remote: true, query: query)
    }
''' + search_methods + '''
    private func promptCurrentFolderFilter(title: String, current: String) -> String? {
''',
    "recursive search actions",
)

source = replace_once(
    source,
    '''        let localReady = engineReady && !localMutationBusy && !localFilterBusy
        let remoteReady = connected && !remoteMutationBusy && !remoteEditBusy && !remoteFilterBusy
''',
    '''        let localReady = engineReady && !localMutationBusy && !localFilterBusy && !recursiveSearchBusy
        let remoteReady = connected && !remoteMutationBusy && !remoteEditBusy && !remoteFilterBusy && !recursiveSearchBusy
''',
    "recursive search readiness",
)

source = replace_once(
    source,
    '''        localFilterButton.isEnabled = localReady
        uploadButton.isEnabled = connected && localReady && localSelectionCount > 0
''',
    '''        localFilterButton.isEnabled = localReady
        localSearchButton.isEnabled = localReady
        uploadButton.isEnabled = connected && localReady && localSelectionCount > 0
''',
    "local search enablement",
)

source = replace_once(
    source,
    '''        remoteFilterButton.isEnabled = remoteReady
        downloadButton.isEnabled = remoteReady && remoteSelectionCount > 0
''',
    '''        remoteFilterButton.isEnabled = remoteReady
        remoteSearchButton.isEnabled = remoteReady
        downloadButton.isEnabled = remoteReady && remoteSelectionCount > 0
''',
    "remote search enablement",
)

controller = r'''

private final class RecursiveSearchWindowController: NSWindowController, NSWindowDelegate, NSTableViewDataSource, NSTableViewDelegate {
    private var results: [RecursiveSearchResult] = []
    private let statusLabel = NSTextField(labelWithString: "Searching…")
    private let table = NSTableView(frame: .zero)
    private let cancelButton = NSButton(title: "", target: nil, action: nil)
    private let closeButton = NSButton(title: "", target: nil, action: nil)
    private let navigateButton = NSButton(title: "", target: nil, action: nil)
    private let onCancel: () -> Void
    private let onClose: () -> Void
    private let onNavigate: (Int) -> Void
    private var closingSilently = false

    init(title: String, root: String, query: String, onCancel: @escaping () -> Void, onClose: @escaping () -> Void, onNavigate: @escaping (Int) -> Void) {
        self.onCancel = onCancel
        self.onClose = onClose
        self.onNavigate = onNavigate
        let panel = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 860, height: 540),
            styleMask: [.titled, .closable, .resizable],
            backing: .buffered,
            defer: false
        )
        panel.title = "Ghost FTP — \(title)"
        panel.minSize = NSSize(width: 680, height: 420)
        super.init(window: panel)
        panel.delegate = self

        let heading = NSTextField(labelWithString: title)
        heading.font = .systemFont(ofSize: 18, weight: .semibold)
        let detail = NSTextField(labelWithString: "\(root)  •  \(query)")
        detail.font = .monospacedSystemFont(ofSize: 11, weight: .regular)
        detail.textColor = .secondaryLabelColor
        detail.lineBreakMode = .byTruncatingMiddle

        table.dataSource = self
        table.delegate = self
        table.allowsEmptySelection = true
        table.allowsMultipleSelection = false
        table.usesAlternatingRowBackgroundColors = true
        addColumn(id: "name", title: "Name", width: 240)
        addColumn(id: "folder", title: "Folder", width: 360)
        addColumn(id: "size", title: "Size", width: 90)
        addColumn(id: "modified", title: "Modified", width: 140)
        let scroll = NSScrollView()
        scroll.documentView = table
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = true
        scroll.borderType = .bezelBorder

        cancelButton.title = "Cancel"
        closeButton.title = "Close"
        navigateButton.title = "Navigate"
        cancelButton.target = self
        cancelButton.action = #selector(cancelTapped)
        closeButton.target = self
        closeButton.action = #selector(closeTapped)
        navigateButton.target = self
        navigateButton.action = #selector(navigateTapped)
        closeButton.isEnabled = false
        navigateButton.isEnabled = false

        statusLabel.textColor = .secondaryLabelColor
        statusLabel.lineBreakMode = .byTruncatingTail
        let buttons = NSStackView(views: [statusLabel, navigateButton, cancelButton, closeButton])
        buttons.orientation = .horizontal
        buttons.alignment = .centerY
        buttons.spacing = 8
        statusLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)

        let stack = NSStackView(views: [heading, detail, scroll, buttons])
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 10
        stack.edgeInsets = NSEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        stack.translatesAutoresizingMaskIntoConstraints = false
        panel.contentView?.addSubview(stack)
        if let content = panel.contentView {
            NSLayoutConstraint.activate([
                stack.leadingAnchor.constraint(equalTo: content.leadingAnchor),
                stack.trailingAnchor.constraint(equalTo: content.trailingAnchor),
                stack.topAnchor.constraint(equalTo: content.topAnchor),
                stack.bottomAnchor.constraint(equalTo: content.bottomAnchor),
                scroll.widthAnchor.constraint(equalTo: stack.widthAnchor, constant: -28),
                scroll.heightAnchor.constraint(greaterThanOrEqualToConstant: 330),
                buttons.widthAnchor.constraint(equalTo: stack.widthAnchor, constant: -28),
            ])
        }
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func finish(results: [RecursiveSearchResult], status: String) {
        self.results = results
        statusLabel.stringValue = status
        table.reloadData()
        cancelButton.isEnabled = false
        closeButton.isEnabled = true
        navigateButton.isEnabled = !results.isEmpty
        if !results.isEmpty {
            table.selectRowIndexes(IndexSet(integer: 0), byExtendingSelection: false)
        }
    }

    func result(at index: Int) -> RecursiveSearchResult? {
        guard index >= 0, index < results.count else { return nil }
        return results[index]
    }

    func closeSilently() {
        closingSilently = true
        window?.close()
    }

    @objc private func cancelTapped() {
        cancelButton.isEnabled = false
        statusLabel.stringValue = "Cancelling…"
        onCancel()
    }

    @objc private func closeTapped() {
        window?.close()
    }

    @objc private func navigateTapped() {
        let index = table.selectedRow
        guard index >= 0, index < results.count else { return }
        onNavigate(index)
    }

    func windowWillClose(_ notification: Notification) {
        if closingSilently { return }
        if cancelButton.isEnabled { onCancel() }
        onClose()
    }

    func numberOfRows(in tableView: NSTableView) -> Int {
        results.count
    }

    func tableViewSelectionDidChange(_ notification: Notification) {
        navigateButton.isEnabled = !cancelButton.isEnabled && table.selectedRow >= 0
    }

    func tableView(_ tableView: NSTableView, viewFor tableColumn: NSTableColumn?, row: Int) -> NSView? {
        guard row >= 0, row < results.count, let column = tableColumn else { return nil }
        let result = results[row]
        let value: String
        switch column.identifier.rawValue {
        case "name":
            let prefix = result.isDirectory ? "[Folder] " : result.isSymlink ? "[Link] " : ""
            value = prefix + result.name
        case "folder":
            value = result.parent
        case "size":
            value = result.isDirectory ? "—" : ByteCountFormatter.string(fromByteCount: result.size, countStyle: .file)
        case "modified":
            if result.modifiedUnix > 0 {
                let formatter = DateFormatter()
                formatter.dateStyle = .short
                formatter.timeStyle = .short
                value = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(result.modifiedUnix)))
            } else {
                value = ""
            }
        default:
            value = ""
        }
        let cell = NSTableCellView()
        let text = NSTextField(labelWithString: value)
        text.lineBreakMode = column.identifier.rawValue == "folder" ? .byTruncatingMiddle : .byTruncatingTail
        text.translatesAutoresizingMaskIntoConstraints = false
        cell.addSubview(text)
        NSLayoutConstraint.activate([
            text.leadingAnchor.constraint(equalTo: cell.leadingAnchor, constant: 5),
            text.trailingAnchor.constraint(equalTo: cell.trailingAnchor, constant: -5),
            text.centerYAnchor.constraint(equalTo: cell.centerYAnchor),
        ])
        return cell
    }

    private func addColumn(id: String, title: String, width: CGFloat) {
        let column = NSTableColumn(identifier: NSUserInterfaceItemIdentifier(id))
        column.title = title
        column.width = width
        column.minWidth = 80
        column.resizingMask = [.autoresizingMask, .userResizingMask]
        table.addTableColumn(column)
    }
}
'''

source = replace_once(
    source,
    '''}

private let app = NSApplication.shared
''',
    '''}''' + controller + '''

private let app = NSApplication.shared
''',
    "recursive search window controller",
)

MAIN.write_text(source, encoding="utf-8")

parity = PARITY.read_text(encoding="utf-8")
parity = replace_once(parity, "- [ ] Local Recursive Search", "- [x] Local Recursive Search", "local parity")
parity = replace_once(parity, "- [ ] Remote Recursive Search", "- [x] Remote Recursive Search", "remote parity")
needle = "Local and Remote Filter now use the same shared `internal/itemlist.Filter` implementation as Windows/Linux."
if needle not in parity:
    raise SystemExit("parity narrative anchor missing")
search_note = (
    "Local and Remote Recursive Search are wired to the shared `Engine.SearchLocalRecursive` / "
    "`Engine.SearchRemoteRecursive` APIs. The AppKit search window exposes Search, Cancel, Close and Navigate; "
    "the bridge requires the visible Local/Remote snapshot to still match the requested root, and the shared engine "
    "retains bounded depth, visited-item, result, batch and timeout limits. Remote search holds one generation-bound "
    "remote operation for the scan, Disconnect cancels search before entering the serialized engine queue, and stale "
    "navigation generations discard results rather than applying them to another folder. Navigate opens the containing "
    "Local/Remote folder through the normal listing path; recursive search does not introduce a second filesystem or "
    "FTP/SFTP traversal implementation.\n\n"
)
parity = parity.replace(needle, search_note + needle, 1)
PARITY.write_text(parity, encoding="utf-8")

print("Applied guarded macOS recursive search UI integration.")
