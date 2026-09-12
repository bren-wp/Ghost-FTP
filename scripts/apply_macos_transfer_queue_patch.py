from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SWIFT = ROOT / "macos/Sources/GhostFTPApp/main.swift"
PARITY = ROOT / "macos/PARITY.md"
GLOBAL_PARITY = ROOT / "scripts/test_macos_windows_parity_contract.py"


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly one anchor, found {count}")
    return text.replace(old, new, 1)


swift = SWIFT.read_text(encoding="utf-8")

types = r'''
private enum TransferQueueMove {
    case top
    case up
    case down
    case bottom
}

private struct TransferQueueEntry {
    let id: String
    let direction: String
    let localPath: String
    let remotePath: String
    let status: String
    let progress: Double
    let bytesTransferred: Int64
    let bytesTotal: Int64
    let bytesPerSecond: Double
    let etaSeconds: Int64
    let attempts: Int
    let error: String
}

'''
swift = replace_once(
    swift,
    'private func bridgeString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {\n',
    types + 'private func bridgeString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {\n',
    "transfer queue types",
)

swift = replace_once(
    swift,
    '    private var directoryCompareController: DirectoryCompareWindowController?\n',
    '''    private var directoryCompareController: DirectoryCompareWindowController?
    private var transferQueueEntries: [TransferQueueEntry] = []
    private var transferQueuePaused = false
    private var transferQueueBusy = false
    private var transferQueueTimer: Timer?
    private var transferQueueController: TransferQueueWindowController?
    private var seenDoneTransferIDs: Set<String> = []
''',
    "transfer queue state",
)

swift = replace_once(
    swift,
    '    private let directoryCompareButton = NSButton(title: "Compare", target: nil, action: nil)\n    private let statusLabel = NSTextField(labelWithString: "Not connected")\n',
    '''    private let directoryCompareButton = NSButton(title: "Compare", target: nil, action: nil)
    private let transferQueueButton = NSButton(title: "Transfers", target: nil, action: nil)
    private let statusLabel = NSTextField(labelWithString: "Not connected")
''',
    "transfer queue button",
)

swift = replace_once(
    swift,
    '''        refreshConnectionState()
        window.makeKeyAndOrderFront(nil)
''',
    '''        refreshConnectionState()
        startTransferQueuePolling()
        window.makeKeyAndOrderFront(nil)
''',
    "start transfer queue polling",
)

swift = replace_once(
    swift,
    '''        GhostFTPCancelPendingTrust()
        GhostFTPShutdown()
''',
    '''        GhostFTPCancelPendingTrust()
        transferQueueTimer?.invalidate()
        transferQueueTimer = nil
        transferQueueController?.closeSilently()
        transferQueueController = nil
        GhostFTPClearTransferQueueSnapshot()
        GhostFTPShutdown()
''',
    "transfer queue termination cleanup",
)

swift = replace_once(
    swift,
    '''        directoryCompareButton.target = self
        directoryCompareButton.action = #selector(directoryCompareTapped)

        configureWorkspaceActions()
''',
    '''        directoryCompareButton.target = self
        directoryCompareButton.action = #selector(directoryCompareTapped)
        transferQueueButton.target = self
        transferQueueButton.action = #selector(transferQueueTapped)

        configureWorkspaceActions()
''',
    "transfer queue action wiring",
)

swift = replace_once(
    swift,
    '        let buttonRow = NSStackView(views: [connectButton, disconnectButton, directoryCompareButton, statusLabel])\n',
    '        let buttonRow = NSStackView(views: [connectButton, disconnectButton, directoryCompareButton, transferQueueButton, statusLabel])\n',
    "transfer queue button row",
)

swift = replace_once(
    swift,
    '''                self.statusLabel.stringValue = text
                if !firstError.isEmpty { self.showError(firstError) }
                self.updateWorkspaceControls()
''',
    '''                self.statusLabel.stringValue = text
                if !firstError.isEmpty { self.showError(firstError) }
                self.refreshTransferQueue(preserveSelection: true, silent: true)
                self.updateWorkspaceControls()
''',
    "queue transfer immediate refresh",
)

swift = replace_once(
    swift,
    '''        updateFilterButtonLabels()
        directoryCompareButton.isEnabled = connected && localReady && remoteReady && directoryCompareController == nil
''',
    '''        updateFilterButtonLabels()
        transferQueueButton.isEnabled = engineReady && !connectionBusy
        directoryCompareButton.isEnabled = connected && localReady && remoteReady && directoryCompareController == nil
''',
    "queue button enablement",
)

queue_methods = r'''
    private func startTransferQueuePolling() {
        guard engineReady else { return }
        transferQueueTimer?.invalidate()
        transferQueueTimer = Timer.scheduledTimer(
            timeInterval: 1.0,
            target: self,
            selector: #selector(transferQueuePollTimerFired(_:)),
            userInfo: nil,
            repeats: true
        )
        refreshTransferQueue(preserveSelection: true, silent: true)
    }

    @objc private func transferQueuePollTimerFired(_ timer: Timer) {
        guard timer.isValid else { return }
        refreshTransferQueue(preserveSelection: true, silent: true)
    }

    @objc private func transferQueueTapped() {
        guard engineReady else { return }
        if let controller = transferQueueController {
            controller.showWindow(nil)
            controller.window?.makeKeyAndOrderFront(nil)
            refreshTransferQueue(preserveSelection: true, silent: true)
            return
        }
        let controller = TransferQueueWindowController(
            onPause: { [weak self] in self?.pauseTransferQueue() },
            onResume: { [weak self] in self?.resumeTransferQueue() },
            onCancel: { [weak self] rows in self?.cancelTransferRows(rows) },
            onRetry: { [weak self] rows in self?.retryTransferRows(rows) },
            onClearFinished: { [weak self] in self?.clearFinishedTransfers() },
            onMove: { [weak self] move in self?.moveSelectedTransfer(move) },
            onClose: { [weak self] in self?.transferQueueController = nil }
        )
        transferQueueController = controller
        controller.apply(
            entries: transferQueueEntries,
            paused: transferQueuePaused,
            connected: GhostFTPIsConnected() == 1,
            preserveIDs: []
        )
        controller.showWindow(nil)
        controller.window?.center()
        controller.window?.makeKeyAndOrderFront(nil)
        refreshTransferQueue(preserveSelection: true, silent: true)
    }

    private func readTransferQueueSnapshot() -> ([TransferQueueEntry], Bool) {
        let count = max(0, Int(GhostFTPTransferCount()))
        let entries = (0..<count).map { index -> TransferQueueEntry in
            let cIndex = CInt(index)
            return TransferQueueEntry(
                id: bridgeString(GhostFTPTransferID(cIndex)),
                direction: bridgeString(GhostFTPTransferDirection(cIndex)),
                localPath: bridgeString(GhostFTPTransferLocalPath(cIndex)),
                remotePath: bridgeString(GhostFTPTransferRemotePath(cIndex)),
                status: bridgeString(GhostFTPTransferStatus(cIndex)),
                progress: Double(GhostFTPTransferProgress(cIndex)),
                bytesTransferred: Int64(GhostFTPTransferBytesTransferred(cIndex)),
                bytesTotal: Int64(GhostFTPTransferBytesTotal(cIndex)),
                bytesPerSecond: Double(GhostFTPTransferBytesPerSecond(cIndex)),
                etaSeconds: Int64(GhostFTPTransferETASeconds(cIndex)),
                attempts: Int(GhostFTPTransferAttempts(cIndex)),
                error: bridgeString(GhostFTPTransferError(cIndex))
            )
        }
        return (entries, GhostFTPTransferQueuePaused() == 1)
    }

    private func selectedTransferIDs(_ rows: IndexSet) -> Set<String> {
        Set(rows.compactMap { index in
            guard index >= 0, index < transferQueueEntries.count else { return nil }
            let id = transferQueueEntries[index].id
            return id.isEmpty ? nil : id
        })
    }

    private func restoreTransferQueueSelection(_ ids: Set<String>) {
        transferQueueController?.restoreTransferQueueSelection(ids)
    }

    private func refreshTransferQueue(preserveSelection: Bool, silent: Bool) {
        guard engineReady, !transferQueueBusy else { return }
        let preserveIDs = preserveSelection ? (transferQueueController?.selectedTransferIDs() ?? []) : []
        transferQueueBusy = true
        engineQueue.async { [weak self] in
            let ok = GhostFTPRefreshTransferQueue() == 1
            let snapshot = ok ? self?.readTransferQueueSnapshot() : nil
            let message = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self else { return }
                self.transferQueueBusy = false
                guard let snapshot else {
                    if !silent && !message.isEmpty { self.showError(message) }
                    return
                }
                self.applyTransferQueueSnapshot(
                    entries: snapshot.0,
                    paused: snapshot.1,
                    preserveIDs: preserveIDs,
                    silent: silent
                )
            }
        }
    }

    private func applyTransferQueueSnapshot(
        entries: [TransferQueueEntry],
        paused: Bool,
        preserveIDs: Set<String>,
        silent: Bool
    ) {
        let currentIDs = Set(entries.map(\.id))
        seenDoneTransferIDs.formIntersection(currentIDs)
        let newlyDone = entries.filter { entry in
            entry.status == "done" && !seenDoneTransferIDs.contains(entry.id)
        }
        for entry in newlyDone where !entry.id.isEmpty {
            seenDoneTransferIDs.insert(entry.id)
        }

        transferQueueEntries = entries
        transferQueuePaused = paused
        transferQueueController?.apply(
            entries: entries,
            paused: paused,
            connected: GhostFTPIsConnected() == 1,
            preserveIDs: preserveIDs
        )
        restoreTransferQueueSelection(preserveIDs)

        if !newlyDone.isEmpty {
            refreshLocal(localCurrent)
            if GhostFTPIsConnected() == 1 {
                refreshRemote(remoteCurrent)
            }
        }
        if !silent {
            statusLabel.stringValue = paused ? "Transfer queue paused." : "Transfer queue updated."
        }
    }

    private func finishTransferQueueAction(ok: Bool, preserveIDs: Set<String>, successStatus: String) {
        let message = ok ? "" : bridgeString(GhostFTPLastError())
        if !ok { _ = GhostFTPRefreshTransferQueue() }
        let snapshot = readTransferQueueSnapshot()
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            self.transferQueueBusy = false
            self.transferQueueController?.setActionBusy(false)
            self.applyTransferQueueSnapshot(
                entries: snapshot.0,
                paused: snapshot.1,
                preserveIDs: preserveIDs,
                silent: true
            )
            if ok {
                self.statusLabel.stringValue = successStatus
            } else {
                self.statusLabel.stringValue = "Transfer queue action failed."
                self.showError(message.isEmpty ? "The transfer queue changed before the action could complete." : message)
            }
        }
    }

    private func beginTransferQueueAction() -> Bool {
        guard engineReady, !transferQueueBusy else { return false }
        transferQueueBusy = true
        transferQueueController?.setActionBusy(true)
        return true
    }

    private func pauseTransferQueue() {
        guard beginTransferQueueAction() else { return }
        let preserveIDs = transferQueueController?.selectedTransferIDs() ?? []
        engineQueue.async { [weak self] in
            let ok = GhostFTPPauseTransferQueue() == 1
            self?.finishTransferQueueAction(ok: ok, preserveIDs: preserveIDs, successStatus: "Transfer queue paused.")
        }
    }

    private func resumeTransferQueue() {
        guard beginTransferQueueAction() else { return }
        let preserveIDs = transferQueueController?.selectedTransferIDs() ?? []
        engineQueue.async { [weak self] in
            let ok = GhostFTPResumeTransferQueue() == 1
            self?.finishTransferQueueAction(ok: ok, preserveIDs: preserveIDs, successStatus: "Transfer queue resumed.")
        }
    }

    private func cancelTransferRows(_ rows: IndexSet) {
        let selected = rows.compactMap { index -> TransferQueueEntry? in
            guard index >= 0, index < transferQueueEntries.count else { return nil }
            return transferQueueEntries[index]
        }
        guard selected.count == rows.count, !selected.isEmpty else { return }
        let canCancel = selected.allSatisfy { entry in
            let status = entry.status
            return status == "queued" || status == "running"
        }
        guard canCancel, beginTransferQueueAction() else { return }
        let preserveIDs = selectedTransferIDs(rows)
        let rawRows = rows.map { CInt($0) }
        engineQueue.async { [weak self] in
            var cRows = rawRows
            let ok = cRows.withUnsafeMutableBufferPointer { buffer in
                GhostFTPCancelTransferRows(buffer.baseAddress, CInt(buffer.count)) == 1
            }
            self?.finishTransferQueueAction(ok: ok, preserveIDs: preserveIDs, successStatus: "Selected transfer(s) cancelled.")
        }
    }

    private func retryTransferRows(_ rows: IndexSet) {
        guard GhostFTPIsConnected() == 1 else {
            statusLabel.stringValue = "Connect before retrying a transfer."
            return
        }
        let selected = rows.compactMap { index -> TransferQueueEntry? in
            guard index >= 0, index < transferQueueEntries.count else { return nil }
            return transferQueueEntries[index]
        }
        guard selected.count == rows.count, !selected.isEmpty else { return }
        let canRetry = selected.allSatisfy { entry in
            let status = entry.status
            return status == "failed" || status == "cancelled"
        }
        guard canRetry, beginTransferQueueAction() else { return }
        let preserveIDs = selectedTransferIDs(rows)
        let rawRows = rows.map { CInt($0) }
        engineQueue.async { [weak self] in
            var cRows = rawRows
            let ok = cRows.withUnsafeMutableBufferPointer { buffer in
                GhostFTPRetryTransferRows(buffer.baseAddress, CInt(buffer.count)) == 1
            }
            self?.finishTransferQueueAction(ok: ok, preserveIDs: preserveIDs, successStatus: "Selected transfer(s) queued for retry.")
        }
    }

    private func clearFinishedTransfers() {
        guard beginTransferQueueAction() else { return }
        let preserveIDs = transferQueueController?.selectedTransferIDs() ?? []
        engineQueue.async { [weak self] in
            let ok = GhostFTPClearFinishedTransfers() == 1
            self?.finishTransferQueueAction(ok: ok, preserveIDs: preserveIDs, successStatus: "Finished transfers cleared.")
        }
    }

    private func selectedQueueReorderCandidate() -> (row: Int, id: String, queuedBefore: Int, queuedAfter: Int)? {
        guard let rows = transferQueueController?.selectedRows(), rows.count == 1, let row = rows.first,
              row >= 0, row < transferQueueEntries.count else { return nil }
        let entry = transferQueueEntries[row]
        let status = entry.status
        guard status == "queued", !entry.id.isEmpty else { return nil }
        let queuedBefore = transferQueueEntries[..<row].filter { $0.status == "queued" }.count
        let queuedAfter = transferQueueEntries[(row + 1)...].filter { $0.status == "queued" }.count
        return (row, entry.id, queuedBefore, queuedAfter)
    }

    private func moveSelectedTransfer(_ move: TransferQueueMove) {
        guard let candidate = selectedQueueReorderCandidate() else { return }
        switch move {
        case .top, .up:
            guard candidate.queuedBefore > 0 else { return }
        case .down, .bottom:
            guard candidate.queuedAfter > 0 else { return }
        }
        guard beginTransferQueueAction() else { return }
        let preserveIDs: Set<String> = [candidate.id]
        engineQueue.async { [weak self] in
            let ok: Bool
            switch move {
            case .top:
                ok = GhostFTPMoveTransferTop(CInt(candidate.row)) == 1
            case .up:
                ok = GhostFTPMoveTransferUp(CInt(candidate.row)) == 1
            case .down:
                ok = GhostFTPMoveTransferDown(CInt(candidate.row)) == 1
            case .bottom:
                ok = GhostFTPMoveTransferBottom(CInt(candidate.row)) == 1
            }
            self?.finishTransferQueueAction(ok: ok, preserveIDs: preserveIDs, successStatus: "Transfer queue priority updated.")
        }
    }

'''
swift = replace_once(
    swift,
    '    private func remoteChild(_ base: String, _ name: String) -> String {\n',
    queue_methods + '    private func remoteChild(_ base: String, _ name: String) -> String {\n',
    "transfer queue methods",
)

controller = r'''
private final class TransferQueueWindowController: NSWindowController, NSWindowDelegate, NSTableViewDataSource, NSTableViewDelegate {
    private var entries: [TransferQueueEntry] = []
    private var transferQueuePaused = false
    private var connected = false
    private var actionBusy = false
    private var closingSilently = false

    private let transferQueueTable = NSTableView(frame: .zero)
    private let summaryLabel = NSTextField(labelWithString: "No transfers")
    private let pauseButton = NSButton(title: "Pause", target: nil, action: nil)
    private let resumeButton = NSButton(title: "Resume", target: nil, action: nil)
    private let cancelButton = NSButton(title: "Cancel", target: nil, action: nil)
    private let retryButton = NSButton(title: "Retry", target: nil, action: nil)
    private let clearFinishedButton = NSButton(title: "Clear Finished", target: nil, action: nil)
    private let topButton = NSButton(title: "Top", target: nil, action: nil)
    private let upButton = NSButton(title: "Up", target: nil, action: nil)
    private let downButton = NSButton(title: "Down", target: nil, action: nil)
    private let bottomButton = NSButton(title: "Bottom", target: nil, action: nil)

    private let onPause: () -> Void
    private let onResume: () -> Void
    private let onCancel: (IndexSet) -> Void
    private let onRetry: (IndexSet) -> Void
    private let onClearFinished: () -> Void
    private let onMove: (TransferQueueMove) -> Void
    private let onClose: () -> Void

    init(
        onPause: @escaping () -> Void,
        onResume: @escaping () -> Void,
        onCancel: @escaping (IndexSet) -> Void,
        onRetry: @escaping (IndexSet) -> Void,
        onClearFinished: @escaping () -> Void,
        onMove: @escaping (TransferQueueMove) -> Void,
        onClose: @escaping () -> Void
    ) {
        self.onPause = onPause
        self.onResume = onResume
        self.onCancel = onCancel
        self.onRetry = onRetry
        self.onClearFinished = onClearFinished
        self.onMove = onMove
        self.onClose = onClose

        let panel = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1180, height: 520),
            styleMask: [.titled, .closable, .resizable],
            backing: .buffered,
            defer: false
        )
        panel.title = "Ghost FTP — Transfer Queue"
        panel.minSize = NSSize(width: 900, height: 420)
        super.init(window: panel)
        panel.delegate = self

        let heading = NSTextField(labelWithString: "Transfer Queue")
        heading.font = .systemFont(ofSize: 18, weight: .semibold)
        summaryLabel.textColor = .secondaryLabelColor
        summaryLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)

        transferQueueTable.dataSource = self
        transferQueueTable.delegate = self
        transferQueueTable.allowsMultipleSelection = true
        transferQueueTable.allowsEmptySelection = true
        transferQueueTable.usesAlternatingRowBackgroundColors = true
        addColumn(id: "direction", title: "Direction", width: 80)
        addColumn(id: "local", title: "Local", width: 250)
        addColumn(id: "remote", title: "Remote", width: 250)
        addColumn(id: "status", title: "Status", width: 105)
        addColumn(id: "progress", title: "Progress", width: 180)
        addColumn(id: "speed", title: "Speed", width: 110)
        addColumn(id: "eta", title: "ETA", width: 90)

        let scroll = NSScrollView()
        scroll.documentView = transferQueueTable
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = true
        scroll.borderType = .bezelBorder

        for button in [pauseButton, resumeButton, cancelButton, retryButton, clearFinishedButton, topButton, upButton, downButton, bottomButton] {
            button.target = self
        }
        pauseButton.action = #selector(pauseTapped)
        resumeButton.action = #selector(resumeTapped)
        cancelButton.action = #selector(cancelTapped)
        retryButton.action = #selector(retryTapped)
        clearFinishedButton.action = #selector(clearFinishedTapped)
        topButton.action = #selector(topTapped)
        upButton.action = #selector(upTapped)
        downButton.action = #selector(downTapped)
        bottomButton.action = #selector(bottomTapped)

        let actions = NSStackView(views: [
            pauseButton, resumeButton, cancelButton, retryButton, clearFinishedButton,
            topButton, upButton, downButton, bottomButton
        ])
        actions.orientation = .horizontal
        actions.alignment = .centerY
        actions.spacing = 7

        let header = NSStackView(views: [heading, summaryLabel])
        header.orientation = .horizontal
        header.alignment = .centerY
        header.spacing = 12

        let stack = NSStackView(views: [header, scroll, actions])
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
                header.widthAnchor.constraint(equalTo: stack.widthAnchor, constant: -28),
                scroll.widthAnchor.constraint(equalTo: stack.widthAnchor, constant: -28),
                scroll.heightAnchor.constraint(greaterThanOrEqualToConstant: 330),
                actions.widthAnchor.constraint(lessThanOrEqualTo: stack.widthAnchor, constant: -28),
            ])
        }
        updateActionControls()
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func apply(entries: [TransferQueueEntry], paused: Bool, connected: Bool, preserveIDs: Set<String>) {
        self.entries = entries
        transferQueuePaused = paused
        self.connected = connected
        transferQueueTable.reloadData()
        restoreTransferQueueSelection(preserveIDs)
        updateSummary()
        updateActionControls()
    }

    func setActionBusy(_ busy: Bool) {
        actionBusy = busy
        updateActionControls()
    }

    func selectedRows() -> IndexSet {
        transferQueueTable.selectedRowIndexes
    }

    func selectedTransferIDs() -> Set<String> {
        Set(transferQueueTable.selectedRowIndexes.compactMap { index in
            guard index >= 0, index < entries.count else { return nil }
            let id = entries[index].id
            return id.isEmpty ? nil : id
        })
    }

    func restoreTransferQueueSelection(_ ids: Set<String>) {
        guard !ids.isEmpty else {
            transferQueueTable.deselectAll(nil)
            return
        }
        var rows = IndexSet()
        for (index, entry) in entries.enumerated() where ids.contains(entry.id) {
            rows.insert(index)
        }
        transferQueueTable.selectRowIndexes(rows, byExtendingSelection: false)
    }

    func closeSilently() {
        closingSilently = true
        window?.close()
    }

    func numberOfRows(in tableView: NSTableView) -> Int {
        entries.count
    }

    func tableViewSelectionDidChange(_ notification: Notification) {
        updateActionControls()
    }

    private func updateSummary() {
        let running = entries.filter { $0.status == "running" }.count
        let queued = entries.filter { $0.status == "queued" }.count
        let done = entries.filter { $0.status == "done" }.count
        let failed = entries.filter { $0.status == "failed" || $0.status == "cancelled" }.count
        let pauseText = transferQueuePaused ? " • paused" : ""
        summaryLabel.stringValue = "\(running) running • \(queued) queued • \(done) done • \(failed) failed/cancelled\(pauseText)"
    }

    private func updateActionControls() {
        let rows = transferQueueTable.selectedRowIndexes
        let selected = rows.compactMap { index -> TransferQueueEntry? in
            guard index >= 0, index < entries.count else { return nil }
            return entries[index]
        }

        let canCancel = !selected.isEmpty && selected.count == rows.count && selected.allSatisfy { entry in
            let status = entry.status
            return status == "queued" || status == "running"
        }
        let canRetry = connected && !selected.isEmpty && selected.count == rows.count && selected.allSatisfy { entry in
            let status = entry.status
            return status == "failed" || status == "cancelled"
        }
        let terminalExists = entries.contains { entry in
            ["done", "failed", "cancelled", "skipped"].contains(entry.status)
        }

        var queuedBefore = 0
        var queuedAfter = 0
        var reorderCandidate = false
        if rows.count == 1, let row = rows.first, row >= 0, row < entries.count {
            let status = entries[row].status
            if status == "queued" {
                reorderCandidate = true
                queuedBefore = entries[..<row].filter { $0.status == "queued" }.count
                queuedAfter = entries[(row + 1)...].filter { $0.status == "queued" }.count
            }
        }

        pauseButton.isEnabled = !actionBusy && !transferQueuePaused
        resumeButton.isEnabled = !actionBusy && transferQueuePaused
        cancelButton.isEnabled = !actionBusy && canCancel
        retryButton.isEnabled = !actionBusy && canRetry
        clearFinishedButton.isEnabled = !actionBusy && terminalExists
        topButton.isEnabled = !actionBusy && reorderCandidate && queuedBefore > 0
        upButton.isEnabled = !actionBusy && reorderCandidate && queuedBefore > 0
        downButton.isEnabled = !actionBusy && reorderCandidate && queuedAfter > 0
        bottomButton.isEnabled = !actionBusy && reorderCandidate && queuedAfter > 0
        transferQueueTable.isEnabled = !actionBusy
    }

    func tableView(_ tableView: NSTableView, viewFor tableColumn: NSTableColumn?, row: Int) -> NSView? {
        guard row >= 0, row < entries.count, let column = tableColumn else { return nil }
        let entry = entries[row]
        let value: String
        switch column.identifier.rawValue {
        case "direction":
            value = entry.direction.capitalized
        case "local":
            value = entry.localPath
        case "remote":
            value = entry.remotePath
        case "status":
            value = entry.error.isEmpty ? entry.status : "\(entry.status) • \(entry.error)"
        case "progress":
            value = progressText(entry)
        case "speed":
            value = entry.bytesPerSecond > 0
                ? ByteCountFormatter.string(fromByteCount: Int64(entry.bytesPerSecond), countStyle: .file) + "/s"
                : "—"
        case "eta":
            value = etaText(entry.etaSeconds)
        default:
            value = ""
        }

        let cell = NSTableCellView()
        let text = NSTextField(labelWithString: value)
        text.lineBreakMode = ["local", "remote"].contains(column.identifier.rawValue) ? .byTruncatingMiddle : .byTruncatingTail
        text.toolTip = value
        text.translatesAutoresizingMaskIntoConstraints = false
        cell.addSubview(text)
        NSLayoutConstraint.activate([
            text.leadingAnchor.constraint(equalTo: cell.leadingAnchor, constant: 5),
            text.trailingAnchor.constraint(equalTo: cell.trailingAnchor, constant: -5),
            text.centerYAnchor.constraint(equalTo: cell.centerYAnchor),
        ])
        return cell
    }

    private func progressText(_ entry: TransferQueueEntry) -> String {
        if entry.bytesTotal > 0 {
            let percent = max(0, min(100, entry.progress))
            let transferred = ByteCountFormatter.string(fromByteCount: entry.bytesTransferred, countStyle: .file)
            let total = ByteCountFormatter.string(fromByteCount: entry.bytesTotal, countStyle: .file)
            return String(format: "%.1f%% • %@ / %@", percent, transferred, total)
        }
        if entry.bytesTransferred > 0 {
            return ByteCountFormatter.string(fromByteCount: entry.bytesTransferred, countStyle: .file)
        }
        return entry.status == "done" ? "100%" : "—"
    }

    private func etaText(_ seconds: Int64) -> String {
        guard seconds > 0 else { return "—" }
        let minutes = seconds / 60
        let remainder = seconds % 60
        if minutes > 0 { return "\(minutes)m \(remainder)s" }
        return "\(seconds)s"
    }

    @objc private func pauseTapped() { onPause() }
    @objc private func resumeTapped() { onResume() }
    @objc private func cancelTapped() { onCancel(transferQueueTable.selectedRowIndexes) }
    @objc private func retryTapped() { onRetry(transferQueueTable.selectedRowIndexes) }
    @objc private func clearFinishedTapped() { onClearFinished() }
    @objc private func topTapped() { onMove(.top) }
    @objc private func upTapped() { onMove(.up) }
    @objc private func downTapped() { onMove(.down) }
    @objc private func bottomTapped() { onMove(.bottom) }

    func windowWillClose(_ notification: Notification) {
        if closingSilently { return }
        onClose()
    }

    private func addColumn(id: String, title: String, width: CGFloat) {
        let column = NSTableColumn(identifier: NSUserInterfaceItemIdentifier(id))
        column.title = title
        column.width = width
        column.minWidth = 70
        column.resizingMask = [.autoresizingMask, .userResizingMask]
        transferQueueTable.addTableColumn(column)
    }
}

'''
swift = replace_once(
    swift,
    '\nprivate let app = NSApplication.shared\n',
    '\n' + controller + 'private let app = NSApplication.shared\n',
    "transfer queue window controller",
)

SWIFT.write_text(swift, encoding="utf-8")

parity = PARITY.read_text(encoding="utf-8")
for action in (
    "Pause Queue",
    "Resume Queue",
    "Cancel Transfer",
    "Retry Transfer",
    "Clear Finished",
    "Move Top",
    "Move Up",
    "Move Down",
    "Move Bottom",
):
    parity = replace_once(parity, f"- [ ] {action}", f"- [x] {action}", f"parity {action}")

paragraph_anchor = (
    "The macOS bridge keeps each transfer action bound to the currently visible engine snapshot so stale UI names cannot be used after navigation. "
    "Download targets are derived with the shared safe-local-child validation and remote names are validated before transfer. "
    "The bridge remains typed C ABI only: no JSON dispatcher, localhost server, browser IPC or credential-bearing generic payload was added.\n"
)
queue_paragraph = (
    paragraph_anchor
    + "\n"
    + "The native AppKit Transfer Queue now renders the authoritative shared transfer-manager snapshot and paused state. "
      "Pause/Resume, multi-select Cancel/Retry, Clear Finished and Top/Up/Down/Bottom priority controls call the existing typed `internal/api.Engine` queue APIs; "
      "macOS does not run a second scheduler or protocol stack. The queue refreshes at a bounded one-second cadence, preserves selection by stable transfer ID, "
      "allows reordering only for one queued job, and uses the shared `TransferJob` byte/progress/speed/ETA fields without fabricating unsupported metrics. "
      "Retry remains connection-bound, and terminal completion refreshes the file panes without changing queue authority.\n"
)
parity = replace_once(parity, paragraph_anchor, queue_paragraph, "transfer queue parity paragraph")
PARITY.write_text(parity, encoding="utf-8")

global_parity = GLOBAL_PARITY.read_text(encoding="utf-8")
anchor = '''    "Upload",
    "Download",
}
'''
replacement = '''    "Upload",
    "Download",
    "Pause Queue",
    "Resume Queue",
    "Cancel Transfer",
    "Retry Transfer",
    "Clear Finished",
    "Move Top",
    "Move Up",
    "Move Down",
    "Move Bottom",
}
'''
global_parity = replace_once(global_parity, anchor, replacement, "global parity implemented set")
GLOBAL_PARITY.write_text(global_parity, encoding="utf-8")
