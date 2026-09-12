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
source = replace_once(source,
'''private struct RecursiveSearchResult {
    let name: String
    let parent: String
    let path: String
    let size: Int64
    let isDirectory: Bool
    let isSymlink: Bool
    let modifiedUnix: Int64
}
''',
'''private struct RecursiveSearchResult {
    let name: String
    let parent: String
    let path: String
    let size: Int64
    let isDirectory: Bool
    let isSymlink: Bool
    let modifiedUnix: Int64
}

private struct DirectoryCompareEntry {
    let name: String
    let status: String
    let hasLocal: Bool
    let hasRemote: Bool
    let localSize: Int64
    let remoteSize: Int64
    let localModifiedUnix: Int64
    let remoteModifiedUnix: Int64
    let canOpenBoth: Bool
}
''', 'compare result model')

source = replace_once(source,
'''    private var recursiveSearchGeneration = 0
    private var recursiveSearchBusy = false
    private var recursiveSearchController: RecursiveSearchWindowController?
''',
'''    private var recursiveSearchGeneration = 0
    private var recursiveSearchBusy = false
    private var recursiveSearchController: RecursiveSearchWindowController?
    private var directoryCompareGeneration = 0
    private var directoryCompareBusy = false
    private var directoryCompareController: DirectoryCompareWindowController?
''', 'compare state')

source = replace_once(source,
'''    private let disconnectButton = NSButton(title: "Disconnect", target: nil, action: nil)
    private let statusLabel = NSTextField(labelWithString: "Not connected")
''',
'''    private let disconnectButton = NSButton(title: "Disconnect", target: nil, action: nil)
    private let directoryCompareButton = NSButton(title: "Compare", target: nil, action: nil)
    private let statusLabel = NSTextField(labelWithString: "Not connected")
''', 'compare button')

source = replace_once(source,
'''    func applicationWillTerminate(_ notification: Notification) {
        recursiveSearchGeneration += 1
''',
'''    func applicationWillTerminate(_ notification: Notification) {
        directoryCompareGeneration += 1
        directoryCompareBusy = false
        GhostFTPCancelDirectoryCompare()
        GhostFTPClearDirectoryCompare()
        directoryCompareController?.closeSilently()
        directoryCompareController = nil
        recursiveSearchGeneration += 1
''', 'terminate compare')

source = replace_once(source,
'''        disconnectButton.target = self
        disconnectButton.action = #selector(disconnectTapped)

        configureWorkspaceActions()
''',
'''        disconnectButton.target = self
        disconnectButton.action = #selector(disconnectTapped)
        directoryCompareButton.target = self
        directoryCompareButton.action = #selector(directoryCompareTapped)

        configureWorkspaceActions()
''', 'compare action binding')

source = replace_once(source,
'''        let buttonRow = NSStackView(views: [connectButton, disconnectButton, statusLabel])
''',
'''        let buttonRow = NSStackView(views: [connectButton, disconnectButton, directoryCompareButton, statusLabel])
''', 'compare button row')

source = replace_once(source,
'''    @objc private func disconnectTapped() {
        GhostFTPCancelRecursiveSearch()
''',
'''    @objc private func disconnectTapped() {
        GhostFTPCancelDirectoryCompare()
        directoryCompareGeneration += 1
        directoryCompareBusy = false
        directoryCompareController?.closeSilently()
        directoryCompareController = nil
        GhostFTPClearDirectoryCompare()
        GhostFTPCancelRecursiveSearch()
''', 'disconnect compare cancellation')

methods = r'''
    @objc private func directoryCompareTapped() {
        guard engineReady,
              GhostFTPIsConnected() == 1,
              !connectionBusy,
              !directoryCompareBusy,
              !recursiveSearchBusy,
              !localMutationBusy,
              !remoteMutationBusy,
              !remoteEditBusy,
              !localFilterBusy,
              !remoteFilterBusy else { return }
        startDirectoryCompare()
    }

    private func startDirectoryCompare() {
        let localBase = localCurrent
        let remoteBase = remoteCurrent
        let localGeneration = localNavigationGeneration
        let remoteGeneration = remoteNavigationGeneration
        directoryCompareGeneration += 1
        let generation = directoryCompareGeneration
        directoryCompareBusy = true
        statusLabel.stringValue = "Comparing local and remote folders…"

        let controller = DirectoryCompareWindowController(
            localBase: localBase,
            remoteBase: remoteBase,
            onCancel: { [weak self] in
                guard let self, generation == self.directoryCompareGeneration else { return }
                GhostFTPCancelDirectoryCompare()
                self.statusLabel.stringValue = "Cancelling directory comparison…"
            },
            onClose: { [weak self] in
                self?.closeDirectoryCompareSession()
            },
            onOpenBoth: { [weak self] index in
                self?.openComparedDirectoryBoth(index: index)
            }
        )
        directoryCompareController = controller
        controller.showWindow(nil)
        controller.window?.center()
        controller.window?.makeKeyAndOrderFront(nil)
        updateWorkspaceControls()

        engineQueue.async { [weak self] in
            let local = CStringBox(localBase)
            let remote = CStringBox(remoteBase)
            let code = Int32(GhostFTPCompareDirectories(local.pointer, remote.pointer))
            let entries = self?.readDirectoryCompareEntries() ?? []
            let message = code == 0 ? bridgeString(GhostFTPLastError()) : ""
            DispatchQueue.main.async {
                guard let self, generation == self.directoryCompareGeneration else { return }
                self.directoryCompareBusy = false
                let snapshotIsCurrent = localGeneration == self.localNavigationGeneration
                    && remoteGeneration == self.remoteNavigationGeneration
                    && self.localCurrent == localBase
                    && self.remoteCurrent == remoteBase
                    && GhostFTPIsConnected() == 1
                if !snapshotIsCurrent {
                    self.directoryCompareController?.finish(entries: [], status: "Folders changed; comparison was discarded.")
                    self.statusLabel.stringValue = "Directory comparison discarded after navigation."
                    GhostFTPClearDirectoryCompare()
                    self.updateWorkspaceControls()
                    return
                }
                switch code {
                case 1:
                    self.directoryCompareController?.finish(entries: entries, status: "Compared \(entries.count) item(s).")
                    self.statusLabel.stringValue = "Directory comparison ready: \(entries.count) item(s)."
                case 2:
                    self.directoryCompareController?.finish(entries: [], status: "Directory comparison cancelled.")
                    self.statusLabel.stringValue = "Directory comparison cancelled."
                default:
                    let error = message.isEmpty ? "Directory comparison failed." : message
                    self.directoryCompareController?.finish(entries: [], status: error)
                    self.statusLabel.stringValue = "Directory comparison failed."
                }
                self.updateWorkspaceControls()
            }
        }
    }

    private func readDirectoryCompareEntries() -> [DirectoryCompareEntry] {
        let count = max(0, Int(GhostFTPDirectoryCompareCount()))
        return (0..<count).map { index in
            let cIndex = CInt(index)
            return DirectoryCompareEntry(
                name: bridgeString(GhostFTPDirectoryCompareName(cIndex)),
                status: bridgeString(GhostFTPDirectoryCompareStatus(cIndex)),
                hasLocal: GhostFTPDirectoryCompareHasLocal(cIndex) == 1,
                hasRemote: GhostFTPDirectoryCompareHasRemote(cIndex) == 1,
                localSize: Int64(GhostFTPDirectoryCompareLocalSize(cIndex)),
                remoteSize: Int64(GhostFTPDirectoryCompareRemoteSize(cIndex)),
                localModifiedUnix: Int64(GhostFTPDirectoryCompareLocalModifiedUnix(cIndex)),
                remoteModifiedUnix: Int64(GhostFTPDirectoryCompareRemoteModifiedUnix(cIndex)),
                canOpenBoth: GhostFTPDirectoryCompareCanOpenBoth(cIndex) == 1
            )
        }
    }

    private func closeDirectoryCompareSession() {
        if directoryCompareBusy {
            GhostFTPCancelDirectoryCompare()
        }
        directoryCompareGeneration += 1
        directoryCompareBusy = false
        GhostFTPClearDirectoryCompare()
        directoryCompareController = nil
        statusLabel.stringValue = GhostFTPIsConnected() == 1 ? "Connected" : "Not connected"
        updateWorkspaceControls()
    }

    private func openComparedDirectoryBoth(index: Int) {
        guard !directoryCompareBusy,
              let controller = directoryCompareController,
              let entry = controller.entry(at: index),
              GhostFTPDirectoryCompareCanOpenBoth(CInt(index)) == 1 else { return }
        let localBase = bridgeString(GhostFTPDirectoryCompareLocalBase())
        let remoteBase = bridgeString(GhostFTPDirectoryCompareRemoteBase())
        guard !localBase.isEmpty, !remoteBase.isEmpty, GhostFTPIsConnected() == 1 else { return }
        let localTarget = URL(fileURLWithPath: localBase, isDirectory: true).appendingPathComponent(entry.name, isDirectory: true).path
        let remoteTarget = remoteChild(remoteBase, entry.name)
        directoryCompareGeneration += 1
        directoryCompareBusy = false
        controller.closeSilently()
        directoryCompareController = nil
        GhostFTPClearDirectoryCompare()
        localFilterQuery = ""
        remoteFilterQuery = ""
        statusLabel.stringValue = "Opening synchronized directory…"
        refreshLocal(localTarget)
        refreshRemote(remoteTarget)
    }

'''
source = replace_once(source,
'''    private func promptPermissionMode() -> String? {
''', methods + '''    private func promptPermissionMode() -> String? {
''', 'compare methods')

source = replace_once(source,
'''        let localReady = engineReady && !localMutationBusy && !localFilterBusy && !recursiveSearchBusy
        let remoteReady = connected && !remoteMutationBusy && !remoteEditBusy && !remoteFilterBusy && !recursiveSearchBusy
''',
'''        let localReady = engineReady && !localMutationBusy && !localFilterBusy && !recursiveSearchBusy && !directoryCompareBusy
        let remoteReady = connected && !remoteMutationBusy && !remoteEditBusy && !remoteFilterBusy && !recursiveSearchBusy && !directoryCompareBusy
''', 'compare busy readiness')

source = replace_once(source,
'''        updateFilterButtonLabels()
        localChooseButton.isEnabled = localReady
''',
'''        updateFilterButtonLabels()
        directoryCompareButton.isEnabled = connected && localReady && remoteReady && directoryCompareController == nil
        localChooseButton.isEnabled = localReady
''', 'compare enablement')

controller = r'''

private final class DirectoryCompareWindowController: NSWindowController, NSWindowDelegate, NSTableViewDataSource, NSTableViewDelegate {
    private var entries: [DirectoryCompareEntry] = []
    private let statusLabel = NSTextField(labelWithString: "Comparing…")
    private let table = NSTableView(frame: .zero)
    private let cancelButton = NSButton(title: "", target: nil, action: nil)
    private let closeButton = NSButton(title: "", target: nil, action: nil)
    private let openBothButton = NSButton(title: "", target: nil, action: nil)
    private let onCancel: () -> Void
    private let onClose: () -> Void
    private let onOpenBoth: (Int) -> Void
    private var closingSilently = false

    init(localBase: String, remoteBase: String, onCancel: @escaping () -> Void, onClose: @escaping () -> Void, onOpenBoth: @escaping (Int) -> Void) {
        self.onCancel = onCancel
        self.onClose = onClose
        self.onOpenBoth = onOpenBoth
        let panel = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 980, height: 560), styleMask: [.titled, .closable, .resizable], backing: .buffered, defer: false)
        panel.title = "Ghost FTP — Directory Compare"
        panel.minSize = NSSize(width: 760, height: 430)
        super.init(window: panel)
        panel.delegate = self

        let heading = NSTextField(labelWithString: "Directory Compare")
        heading.font = .systemFont(ofSize: 18, weight: .semibold)
        let detail = NSTextField(labelWithString: "Local: \(localBase)   •   Remote: \(remoteBase)")
        detail.font = .monospacedSystemFont(ofSize: 11, weight: .regular)
        detail.textColor = .secondaryLabelColor
        detail.lineBreakMode = .byTruncatingMiddle

        table.dataSource = self
        table.delegate = self
        table.allowsEmptySelection = true
        table.allowsMultipleSelection = false
        table.usesAlternatingRowBackgroundColors = true
        addColumn(id: "name", title: "Name", width: 250)
        addColumn(id: "status", title: "Status", width: 150)
        addColumn(id: "local", title: "Local", width: 250)
        addColumn(id: "remote", title: "Remote", width: 250)
        let scroll = NSScrollView()
        scroll.documentView = table
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = true
        scroll.borderType = .bezelBorder

        cancelButton.title = "Cancel"
        closeButton.title = "Close"
        openBothButton.title = "Open Both"
        cancelButton.target = self
        cancelButton.action = #selector(cancelTapped)
        closeButton.target = self
        closeButton.action = #selector(closeTapped)
        openBothButton.target = self
        openBothButton.action = #selector(openBothTapped)
        closeButton.isEnabled = false
        openBothButton.isEnabled = false
        statusLabel.textColor = .secondaryLabelColor
        statusLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)
        let buttons = NSStackView(views: [statusLabel, openBothButton, cancelButton, closeButton])
        buttons.orientation = .horizontal
        buttons.alignment = .centerY
        buttons.spacing = 8

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
                scroll.heightAnchor.constraint(greaterThanOrEqualToConstant: 340),
                buttons.widthAnchor.constraint(equalTo: stack.widthAnchor, constant: -28),
            ])
        }
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    func finish(entries: [DirectoryCompareEntry], status: String) {
        self.entries = entries
        statusLabel.stringValue = status
        table.reloadData()
        cancelButton.isEnabled = false
        closeButton.isEnabled = true
        if !entries.isEmpty {
            table.selectRowIndexes(IndexSet(integer: 0), byExtendingSelection: false)
        }
        updateOpenBoth()
    }

    func entry(at index: Int) -> DirectoryCompareEntry? {
        guard index >= 0, index < entries.count else { return nil }
        return entries[index]
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

    @objc private func closeTapped() { window?.close() }

    @objc private func openBothTapped() {
        let index = table.selectedRow
        guard index >= 0, index < entries.count, entries[index].canOpenBoth else { return }
        onOpenBoth(index)
    }

    func windowWillClose(_ notification: Notification) {
        if closingSilently { return }
        if cancelButton.isEnabled { onCancel() }
        onClose()
    }

    func numberOfRows(in tableView: NSTableView) -> Int { entries.count }

    func tableViewSelectionDidChange(_ notification: Notification) { updateOpenBoth() }

    private func updateOpenBoth() {
        let row = table.selectedRow
        openBothButton.isEnabled = !cancelButton.isEnabled && row >= 0 && row < entries.count && entries[row].canOpenBoth
    }

    func tableView(_ tableView: NSTableView, viewFor tableColumn: NSTableColumn?, row: Int) -> NSView? {
        guard row >= 0, row < entries.count, let column = tableColumn else { return nil }
        let entry = entries[row]
        let value: String
        switch column.identifier.rawValue {
        case "name": value = entry.name
        case "status": value = statusTitle(entry.status)
        case "local": value = sideTitle(has: entry.hasLocal, size: entry.localSize, modified: entry.localModifiedUnix)
        case "remote": value = sideTitle(has: entry.hasRemote, size: entry.remoteSize, modified: entry.remoteModifiedUnix)
        default: value = ""
        }
        let cell = NSTableCellView()
        let text = NSTextField(labelWithString: value)
        text.lineBreakMode = .byTruncatingTail
        text.translatesAutoresizingMaskIntoConstraints = false
        cell.addSubview(text)
        NSLayoutConstraint.activate([
            text.leadingAnchor.constraint(equalTo: cell.leadingAnchor, constant: 5),
            text.trailingAnchor.constraint(equalTo: cell.trailingAnchor, constant: -5),
            text.centerYAnchor.constraint(equalTo: cell.centerYAnchor),
        ])
        return cell
    }

    private func statusTitle(_ value: String) -> String {
        switch value {
        case "same": return "Same"
        case "local_only": return "Local only"
        case "remote_only": return "Remote only"
        case "newer_local": return "Newer local"
        case "newer_remote": return "Newer remote"
        case "conflict": return "Conflict"
        default: return "Unknown"
        }
    }

    private func sideTitle(has: Bool, size: Int64, modified: Int64) -> String {
        guard has else { return "—" }
        let sizeText = ByteCountFormatter.string(fromByteCount: size, countStyle: .file)
        guard modified > 0 else { return sizeText }
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .short
        return "\(sizeText) • \(formatter.string(from: Date(timeIntervalSince1970: TimeInterval(modified))))"
    }

    private func addColumn(id: String, title: String, width: CGFloat) {
        let column = NSTableColumn(identifier: NSUserInterfaceItemIdentifier(id))
        column.title = title
        column.width = width
        column.minWidth = 90
        column.resizingMask = [.autoresizingMask, .userResizingMask]
        table.addTableColumn(column)
    }
}
'''
source = replace_once(source,
'''private final class RecursiveSearchWindowController: NSWindowController, NSWindowDelegate, NSTableViewDataSource, NSTableViewDelegate {
''', controller + '''
private final class RecursiveSearchWindowController: NSWindowController, NSWindowDelegate, NSTableViewDataSource, NSTableViewDelegate {
''', 'compare controller')

MAIN.write_text(source, encoding="utf-8")

parity = PARITY.read_text(encoding="utf-8")
parity = replace_once(parity, "- [ ] Directory Compare", "- [x] Directory Compare", "compare parity")
anchor = "Local and Remote Recursive Search are wired to the shared `Engine.SearchLocalRecursive` / `Engine.SearchRemoteRecursive` APIs."
note = ("Directory Compare is wired to the shared `Engine.CompareDirectoryItems` contract using fresh Local/Remote listings from the existing engine. The native comparison window shows conservative status for both sides and enables Open Both only when `Engine.SynchronizedDirectoryName` proves an ordinary directory exists on both sides. Compare is cancellable before queued disconnect, generation/path-bound, and reuses the normal Local/Remote navigation paths; it performs no independent filesystem or protocol traversal.\n\n")
if anchor not in parity:
    raise SystemExit("parity narrative anchor missing")
parity = parity.replace(anchor, note + anchor, 1)
PARITY.write_text(parity, encoding="utf-8")
print("Applied guarded macOS Directory Compare integration.")
