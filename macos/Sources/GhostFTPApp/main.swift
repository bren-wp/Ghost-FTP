import AppKit
import Darwin
import Foundation
import GhostFTPEngine

private enum Palette {
    static let workspace = NSColor(rgb: 0xEEF1F5)
    static let panel = NSColor(rgb: 0xF6F8FB)
    static let list = NSColor(rgb: 0xFAFBFD)
    static let text = NSColor(rgb: 0x111827)
    static let muted = NSColor(rgb: 0x667085)
    static let accent = NSColor(rgb: 0x2563EB)
    static let border = NSColor(rgb: 0xD7DDE6)
}

private extension NSColor {
    convenience init(rgb: Int) {
        self.init(
            calibratedRed: CGFloat((rgb >> 16) & 0xff) / 255.0,
            green: CGFloat((rgb >> 8) & 0xff) / 255.0,
            blue: CGFloat(rgb & 0xff) / 255.0,
            alpha: 1
        )
    }
}

private final class CStringBox {
    let pointer: UnsafeMutablePointer<CChar>?
    init(_ value: String) { pointer = strdup(value) }
    deinit { free(pointer) }
}

private struct ConnectionInput {
    var protocolName: String
    var host: String
    var port: Int32
    var username: String
    var password: String
    var privateKeyPath: String
    var passphrase: String
    var rememberFingerprint: Bool

    mutating func clearSecrets() {
        password.removeAll(keepingCapacity: false)
        passphrase.removeAll(keepingCapacity: false)
    }
}

private struct FileItem {
    let name: String
    let size: Int64
    let isDirectory: Bool
    let isSymlink: Bool
    let modifiedUnix: Int64
    let permissions: String
}

private struct MutationResult {
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

private struct SavedProfileView {
    let id: String
    let name: String
    let protocolName: String
    let host: String
    let port: Int32
    let username: String
    let HasPassword: Bool
    let privateKeyPath: String
    let HasPassphrase: Bool
    let remotePath: String
    let localPath: String
}

private func bridgeString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {
    guard let pointer else { return "" }
    defer { GhostFTPFreeCString(pointer) }
    return String(cString: pointer)
}

private final class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate, NSTableViewDataSource, NSTableViewDelegate {
    private var window: NSWindow!
    private let engineQueue = DispatchQueue(label: "app.ghostftp.engine", qos: .userInitiated)
    private var engineReady = false
    private var connectionBusy = false
    private var localNavigationGeneration = 0
    private var remoteNavigationGeneration = 0
    private var localMutationGeneration = 0
    private var remoteMutationGeneration = 0
    private var remoteEditGeneration = 0
    private var localMutationBusy = false
    private var remoteMutationBusy = false
    private var remoteEditBusy = false
    private var localFilterBusy = false
    private var remoteFilterBusy = false
    private var recursiveSearchGeneration = 0
    private var recursiveSearchBusy = false
    private var recursiveSearchController: RecursiveSearchWindowController?
    private var directoryCompareGeneration = 0
    private var directoryCompareBusy = false
    private var directoryCompareController: DirectoryCompareWindowController?
    private var transferQueueEntries: [TransferQueueEntry] = []
    private var transferQueuePaused = false
    private var transferQueueBusy = false
    private var transferQueueTimer: Timer?
    private var transferQueueController: TransferQueueWindowController?
    private var siteManagerController: SiteManagerWindowController?
    private var seenDoneTransferIDs: Set<String> = []

    private let protocolPopup = NSPopUpButton(frame: .zero, pullsDown: false)
    private let hostField = NSTextField(string: "")
    private let portField = NSTextField(string: "21")
    private let usernameField = NSTextField(string: "")
    private let passwordField = NSSecureTextField(string: "")
    private let privateKeyField = NSTextField(string: "")
    private let passphraseField = NSSecureTextField(string: "")
    private let privateKeyLabel = NSTextField(labelWithString: "Private key")
    private let passphraseLabel = NSTextField(labelWithString: "Passphrase")
    private let chooseKeyButton = NSButton(title: "Browse…", target: nil, action: nil)
    private let rememberFingerprint = NSButton(checkboxWithTitle: "Remember trusted SFTP host key", target: nil, action: nil)
    private let connectButton = NSButton(title: "Connect", target: nil, action: nil)
    private let disconnectButton = NSButton(title: "Disconnect", target: nil, action: nil)
    private let directoryCompareButton = NSButton(title: "Compare", target: nil, action: nil)
    private let transferQueueButton = NSButton(title: "Transfers", target: nil, action: nil)
    private let siteManagerButton = NSButton(title: "Site Manager", target: nil, action: nil)
    private let statusLabel = NSTextField(labelWithString: "Not connected")

    private let localTable = NSTableView(frame: .zero)
    private let remoteTable = NSTableView(frame: .zero)
    private let localPathLabel = NSTextField(labelWithString: "")
    private let remotePathLabel = NSTextField(labelWithString: "")
    private let localChooseButton = NSButton(title: "Choose Folder…", target: nil, action: nil)
    private let localUpButton = NSButton(title: "Up", target: nil, action: nil)
    private let localRefreshButton = NSButton(title: "Refresh", target: nil, action: nil)
    private let localNewFolderButton = NSButton(title: "New Folder", target: nil, action: nil)
    private let localRenameButton = NSButton(title: "Rename", target: nil, action: nil)
    private let localDeleteButton = NSButton(title: "Delete", target: nil, action: nil)
    private let localFilterButton = NSButton(title: "Filter", target: nil, action: nil)
    private let localSearchButton = NSButton(title: "Search", target: nil, action: nil)
    private let uploadButton = NSButton(title: "Upload", target: nil, action: nil)
    private let remoteUpButton = NSButton(title: "Up", target: nil, action: nil)
    private let remoteRefreshButton = NSButton(title: "Refresh", target: nil, action: nil)
    private let remoteNewFolderButton = NSButton(title: "New Folder", target: nil, action: nil)
    private let remoteRenameButton = NSButton(title: "Rename", target: nil, action: nil)
    private let remoteDeleteButton = NSButton(title: "Delete", target: nil, action: nil)
    private let remotePermissionsButton = NSButton(title: "Permissions", target: nil, action: nil)
    private let remoteEditButton = NSButton(title: "Edit", target: nil, action: nil)
    private let remoteFilterButton = NSButton(title: "Filter", target: nil, action: nil)
    private let remoteSearchButton = NSButton(title: "Search", target: nil, action: nil)
    private let downloadButton = NSButton(title: "Download", target: nil, action: nil)

    private var localCurrent = NSHomeDirectory()
    private var remoteCurrent = "/"
    private var localItems: [FileItem] = []
    private var remoteItems: [FileItem] = []
    private var localFilterQuery = ""
    private var remoteFilterQuery = ""
    private var localSourceCount = 0
    private var remoteSourceCount = 0

    private lazy var byteFormatter: ByteCountFormatter = {
        let formatter = ByteCountFormatter()
        formatter.countStyle = .file
        formatter.allowedUnits = [.useKB, .useMB, .useGB, .useTB]
        return formatter
    }()

    private lazy var dateFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .short
        return formatter
    }()

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        buildWindow()
        engineReady = GhostFTPCreateEngine() == 1
        if !engineReady {
            statusLabel.stringValue = bridgeString(GhostFTPLastError())
            connectButton.isEnabled = false
        } else {
            refreshLocal(localCurrent)
        }
        refreshConnectionState()
        startTransferQueuePolling()
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    func applicationWillTerminate(_ notification: Notification) {
        directoryCompareGeneration += 1
        directoryCompareBusy = false
        GhostFTPCancelDirectoryCompare()
        GhostFTPClearDirectoryCompare()
        directoryCompareController?.closeSilently()
        directoryCompareController = nil
        recursiveSearchGeneration += 1
        recursiveSearchBusy = false
        GhostFTPCancelRecursiveSearch()
        GhostFTPClearRecursiveSearch()
        recursiveSearchController?.closeSilently()
        recursiveSearchController = nil
        remoteEditGeneration += 1
        remoteEditBusy = false
        GhostFTPRemoteEditClear()
        GhostFTPCancelRemoteChmodBatch()
        GhostFTPCancelPendingTrust()
        transferQueueTimer?.invalidate()
        transferQueueTimer = nil
        transferQueueController?.closeSilently()
        transferQueueController = nil
        siteManagerController?.close()
        siteManagerController = nil
        GhostFTPClearProfilesSnapshot()
        GhostFTPClearTransferQueueSnapshot()
        GhostFTPShutdown()
    }

    private func buildWindow() {
        window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1440, height: 840),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Ghost FTP"
        window.minSize = NSSize(width: 1120, height: 700)
        window.center()
        window.delegate = self
        window.contentView?.wantsLayer = true
        window.contentView?.layer?.backgroundColor = Palette.workspace.cgColor

        protocolPopup.addItems(withTitles: ["FTP", "FTPS", "SFTP"])
        protocolPopup.target = self
        protocolPopup.action = #selector(protocolChanged)
        chooseKeyButton.target = self
        chooseKeyButton.action = #selector(choosePrivateKey)
        connectButton.target = self
        connectButton.action = #selector(connectTapped)
        connectButton.bezelStyle = .rounded
        connectButton.keyEquivalent = "\r"
        disconnectButton.target = self
        disconnectButton.action = #selector(disconnectTapped)
        directoryCompareButton.target = self
        directoryCompareButton.action = #selector(directoryCompareTapped)
        transferQueueButton.target = self
        transferQueueButton.action = #selector(transferQueueTapped)
        siteManagerButton.target = self
        siteManagerButton.action = #selector(siteManagerTapped)

        configureWorkspaceActions()
        configureTable(localTable, remote: false)
        configureTable(remoteTable, remote: true)

        let title = NSTextField(labelWithString: "Ghost FTP")
        title.font = .systemFont(ofSize: 25, weight: .bold)
        title.textColor = Palette.text
        let subtitle = NSTextField(labelWithString: "Quick Connect")
        subtitle.font = .systemFont(ofSize: 13, weight: .medium)
        subtitle.textColor = Palette.muted

        let heading = NSStackView(views: [title, subtitle])
        heading.orientation = .vertical
        heading.alignment = .leading
        heading.spacing = 3

        let form = makeConnectionForm()
        let buttonRow = NSStackView(views: [connectButton, disconnectButton, siteManagerButton, directoryCompareButton, transferQueueButton, statusLabel])
        buttonRow.orientation = .horizontal
        buttonRow.alignment = .centerY
        buttonRow.spacing = 10
        statusLabel.textColor = Palette.muted
        statusLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)

        let connectionStack = NSStackView(views: [heading, form, rememberFingerprint, buttonRow])
        connectionStack.orientation = .vertical
        connectionStack.alignment = .leading
        connectionStack.spacing = 14
        connectionStack.edgeInsets = NSEdgeInsets(top: 18, left: 20, bottom: 18, right: 20)
        connectionStack.wantsLayer = true
        connectionStack.layer?.backgroundColor = Palette.panel.cgColor
        connectionStack.layer?.cornerRadius = 10
        connectionStack.layer?.borderWidth = 1
        connectionStack.layer?.borderColor = Palette.border.cgColor

        let workspace = makeWorkspace()
        let root = NSStackView(views: [connectionStack, workspace])
        root.orientation = .vertical
        root.spacing = 14
        root.edgeInsets = NSEdgeInsets(top: 18, left: 18, bottom: 18, right: 18)
        root.translatesAutoresizingMaskIntoConstraints = false
        window.contentView?.addSubview(root)
        guard let content = window.contentView else { return }
        NSLayoutConstraint.activate([
            root.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            root.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            root.topAnchor.constraint(equalTo: content.topAnchor),
            root.bottomAnchor.constraint(equalTo: content.bottomAnchor),
            workspace.heightAnchor.constraint(greaterThanOrEqualToConstant: 380),
            connectionStack.widthAnchor.constraint(equalTo: root.widthAnchor, constant: -36)
        ])
        protocolChanged()
        updateWorkspaceControls()
    }

    private func makeConnectionForm() -> NSGridView {
        let keyRow = NSStackView(views: [privateKeyField, chooseKeyButton])
        keyRow.orientation = .horizontal
        keyRow.spacing = 8
        privateKeyField.widthAnchor.constraint(greaterThanOrEqualToConstant: 280).isActive = true

        let grid = NSGridView(views: [
            [fieldLabel("Protocol"), protocolPopup, fieldLabel("Host"), hostField],
            [fieldLabel("Port"), portField, fieldLabel("Username"), usernameField],
            [fieldLabel("Password"), passwordField, privateKeyLabel, keyRow],
            [passphraseLabel, passphraseField, NSView(), NSView()]
        ])
        grid.rowSpacing = 10
        grid.columnSpacing = 10
        grid.column(at: 0).xPlacement = .trailing
        grid.column(at: 2).xPlacement = .trailing
        hostField.widthAnchor.constraint(greaterThanOrEqualToConstant: 300).isActive = true
        usernameField.widthAnchor.constraint(greaterThanOrEqualToConstant: 300).isActive = true
        passwordField.widthAnchor.constraint(greaterThanOrEqualToConstant: 220).isActive = true
        passphraseField.widthAnchor.constraint(greaterThanOrEqualToConstant: 220).isActive = true
        return grid
    }

    private func makeWorkspace() -> NSView {
        let split = NSSplitView(frame: .zero)
        split.isVertical = true
        split.dividerStyle = .thin
        split.translatesAutoresizingMaskIntoConstraints = false
        split.addArrangedSubview(makeFilePane(
            title: "Local",
            table: localTable,
            pathLabel: localPathLabel,
            buttons: [localChooseButton, localUpButton, localRefreshButton, localNewFolderButton, localRenameButton, localDeleteButton, localFilterButton, localSearchButton, uploadButton]
        ))
        split.addArrangedSubview(makeFilePane(
            title: "Remote",
            table: remoteTable,
            pathLabel: remotePathLabel,
            buttons: [remoteUpButton, remoteRefreshButton, remoteNewFolderButton, remoteRenameButton, remoteDeleteButton, remotePermissionsButton, remoteEditButton, remoteFilterButton, remoteSearchButton, downloadButton]
        ))

        let container = NSView()
        container.wantsLayer = true
        container.layer?.backgroundColor = Palette.panel.cgColor
        container.layer?.cornerRadius = 10
        container.layer?.borderWidth = 1
        container.layer?.borderColor = Palette.border.cgColor
        container.addSubview(split)
        NSLayoutConstraint.activate([
            split.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: 10),
            split.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -10),
            split.topAnchor.constraint(equalTo: container.topAnchor, constant: 10),
            split.bottomAnchor.constraint(equalTo: container.bottomAnchor, constant: -10)
        ])
        return container
    }

    private func makeFilePane(title: String, table: NSTableView, pathLabel: NSTextField, buttons: [NSButton]) -> NSView {
        let titleLabel = NSTextField(labelWithString: title)
        titleLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        titleLabel.textColor = Palette.text

        pathLabel.font = .monospacedSystemFont(ofSize: 11, weight: .regular)
        pathLabel.textColor = Palette.muted
        pathLabel.lineBreakMode = .byTruncatingMiddle
        pathLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        let heading = NSStackView(views: [titleLabel, pathLabel])
        heading.orientation = .horizontal
        heading.alignment = .centerY
        heading.spacing = 10
        pathLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)

        let toolbar = NSStackView(views: buttons)
        toolbar.orientation = .horizontal
        toolbar.alignment = .centerY
        toolbar.spacing = 6

        let scroll = NSScrollView()
        scroll.documentView = table
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = false
        scroll.autohidesScrollers = true
        scroll.borderType = .bezelBorder
        scroll.drawsBackground = true
        scroll.backgroundColor = Palette.list

        let stack = NSStackView(views: [heading, toolbar, scroll])
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 8
        stack.edgeInsets = NSEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
        stack.translatesAutoresizingMaskIntoConstraints = false

        let pane = NSView()
        pane.wantsLayer = true
        pane.layer?.backgroundColor = Palette.list.cgColor
        pane.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: pane.leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: pane.trailingAnchor),
            stack.topAnchor.constraint(equalTo: pane.topAnchor),
            stack.bottomAnchor.constraint(equalTo: pane.bottomAnchor),
            heading.widthAnchor.constraint(equalTo: stack.widthAnchor, constant: -24),
            scroll.widthAnchor.constraint(equalTo: stack.widthAnchor, constant: -24),
            scroll.heightAnchor.constraint(greaterThanOrEqualToConstant: 300)
        ])
        return pane
    }

    private func configureWorkspaceActions() {
        localChooseButton.target = self
        localChooseButton.action = #selector(chooseLocalFolder)
        localUpButton.target = self
        localUpButton.action = #selector(localUp)
        localRefreshButton.target = self
        localRefreshButton.action = #selector(refreshLocalTapped)
        localNewFolderButton.target = self
        localNewFolderButton.action = #selector(localNewFolderTapped)
        localRenameButton.target = self
        localRenameButton.action = #selector(localRenameTapped)
        localDeleteButton.target = self
        localDeleteButton.action = #selector(localDeleteTapped)
        localFilterButton.target = self
        localFilterButton.action = #selector(localFilterTapped)
        localSearchButton.target = self
        localSearchButton.action = #selector(localRecursiveSearchTapped)
        uploadButton.target = self
        uploadButton.action = #selector(uploadTapped)

        remoteUpButton.target = self
        remoteUpButton.action = #selector(remoteUp)
        remoteRefreshButton.target = self
        remoteRefreshButton.action = #selector(refreshRemoteTapped)
        remoteNewFolderButton.target = self
        remoteNewFolderButton.action = #selector(remoteNewFolderTapped)
        remoteRenameButton.target = self
        remoteRenameButton.action = #selector(remoteRenameTapped)
        remoteDeleteButton.target = self
        remoteDeleteButton.action = #selector(remoteDeleteTapped)
        remotePermissionsButton.target = self
        remotePermissionsButton.action = #selector(remotePermissionsTapped)
        remoteEditButton.target = self
        remoteEditButton.action = #selector(remoteEditTapped)
        remoteFilterButton.target = self
        remoteFilterButton.action = #selector(remoteFilterTapped)
        remoteSearchButton.target = self
        remoteSearchButton.action = #selector(remoteRecursiveSearchTapped)
        downloadButton.target = self
        downloadButton.action = #selector(downloadTapped)
    }

    private func configureTable(_ table: NSTableView, remote: Bool) {
        table.dataSource = self
        table.delegate = self
        table.allowsMultipleSelection = true
        table.allowsEmptySelection = true
        table.usesAlternatingRowBackgroundColors = false
        table.backgroundColor = Palette.list
        table.rowSizeStyle = .medium
        table.intercellSpacing = NSSize(width: 8, height: 2)
        table.headerView = NSTableHeaderView()
        table.target = self
        table.doubleAction = remote ? #selector(remoteDoubleClicked) : #selector(localDoubleClicked)

        addColumn(table, id: "name", title: "Name", width: 260)
        addColumn(table, id: "size", title: "Size", width: 100)
        addColumn(table, id: "modified", title: "Modified", width: 150)
        if remote {
            addColumn(table, id: "permissions", title: "Permissions", width: 105)
        }
    }

    private func addColumn(_ table: NSTableView, id: String, title: String, width: CGFloat) {
        let column = NSTableColumn(identifier: NSUserInterfaceItemIdentifier(id))
        column.title = title
        column.width = width
        column.minWidth = id == "name" ? 160 : 80
        column.resizingMask = id == "name" ? [.autoresizingMask, .userResizingMask] : [.userResizingMask]
        table.addTableColumn(column)
    }

    private func fieldLabel(_ title: String) -> NSTextField {
        let label = NSTextField(labelWithString: title)
        label.textColor = Palette.text
        label.font = .systemFont(ofSize: 12, weight: .medium)
        return label
    }

    @objc private func protocolChanged() {
        let sftp = protocolPopup.titleOfSelectedItem == "SFTP"
        privateKeyLabel.isHidden = !sftp
        privateKeyField.isHidden = !sftp
        chooseKeyButton.isHidden = !sftp
        passphraseLabel.isHidden = !sftp
        passphraseField.isHidden = !sftp
        rememberFingerprint.isHidden = !sftp
        if sftp && portField.stringValue == "21" { portField.stringValue = "22" }
        if !sftp && portField.stringValue == "22" { portField.stringValue = "21" }
    }

    @objc private func choosePrivateKey() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false
        panel.title = "Choose private key"
        if panel.runModal() == .OK, let url = panel.url {
            privateKeyField.stringValue = url.path
        }
    }

    private func currentInput() -> ConnectionInput? {
        let host = hostField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let username = usernameField.stringValue
        guard !host.isEmpty else {
            showError("Host is required.")
            return nil
        }
        guard let port = Int32(portField.stringValue), port > 0, port <= 65535 else {
            showError("Port must be between 1 and 65535.")
            return nil
        }
        return ConnectionInput(
            protocolName: (protocolPopup.titleOfSelectedItem ?? "FTP").lowercased(),
            host: host,
            port: port,
            username: username,
            password: passwordField.stringValue,
            privateKeyPath: privateKeyField.stringValue,
            passphrase: passphraseField.stringValue,
            rememberFingerprint: rememberFingerprint.state == .on
        )
    }

    @objc private func connectTapped() {
        guard let input = currentInput() else { return }
        passwordField.stringValue = ""
        passphraseField.stringValue = ""
        setBusy(true, status: "Connecting…")
        engineQueue.async { [weak self] in
            let result = self?.bridgeConnect(input: input, trustFingerprint: "") ?? 0
            DispatchQueue.main.async {
                self?.handleConnectResult(result, original: input)
            }
        }
    }

    private func bridgeConnect(input: ConnectionInput, trustFingerprint: String) -> Int32 {
        let p = CStringBox(input.protocolName)
        let h = CStringBox(input.host)
        let u = CStringBox(input.username)
        let pw = CStringBox(input.password)
        let key = CStringBox(input.privateKeyPath)
        let pass = CStringBox(input.passphrase)
        let trust = CStringBox(trustFingerprint)
        return Int32(GhostFTPConnect(p.pointer, h.pointer, CInt(input.port), u.pointer, pw.pointer, key.pointer, pass.pointer, trust.pointer, input.rememberFingerprint ? 1 : 0))
    }

    private func handleConnectResult(_ result: Int32, original: ConnectionInput) {
        if result == 1 {
            var finished = original
            finished.clearSecrets()
            remoteCurrent = original.protocolName == "sftp" ? "." : "/"
            setBusy(false, status: "Connected")
            refreshConnectionState()
            refreshLocal(localCurrent)
            refreshRemote(remoteCurrent)
            return
        }
        if result == 2 {
            let fingerprint = bridgeString(GhostFTPPendingFingerprint())
            guard !fingerprint.isEmpty else {
                var finished = original
                finished.clearSecrets()
                setBusy(false, status: "Connection failed")
                showError("Ghost FTP did not receive a valid SFTP host fingerprint.")
                return
            }
            let alert = NSAlert()
            alert.messageText = "Trust this SFTP host key?"
            alert.informativeText = fingerprint
            alert.alertStyle = .warning
            alert.addButton(withTitle: "Trust and connect")
            alert.addButton(withTitle: "Cancel")
            if alert.runModal() == .alertFirstButtonReturn {
                var trusted = original
                setBusy(true, status: "Verifying host key…")
                engineQueue.async { [weak self] in
                    let second = self?.bridgeConnect(input: trusted, trustFingerprint: fingerprint) ?? 0
                    trusted.clearSecrets()
                    DispatchQueue.main.async { self?.handleConnectResult(second, original: trusted) }
                }
            } else {
                var finished = original
                finished.clearSecrets()
                GhostFTPCancelPendingTrust()
                setBusy(false, status: "Not connected")
                refreshConnectionState()
            }
            return
        }
        var finished = original
        finished.clearSecrets()
        setBusy(false, status: "Connection failed")
        showError(bridgeString(GhostFTPLastError()))
        refreshConnectionState()
    }

    @objc private func siteManagerTapped() {
        guard engineReady, !connectionBusy else { return }
        if siteManagerController == nil {
            siteManagerController = SiteManagerWindowController(engineQueue: engineQueue) { [weak self] localPath, remotePath in
                guard let self else { return }
                self.setBusy(false, status: "Connected")
                self.refreshConnectionState()
                if !localPath.isEmpty {
                    self.refreshLocal(localPath)
                }
                let targetRemote = remotePath.isEmpty ? "/" : remotePath
                self.remoteCurrent = targetRemote
                self.refreshRemote(targetRemote)
            }
        }
        siteManagerController?.showAndRefresh()
    }

    @objc private func disconnectTapped() {
        GhostFTPCancelDirectoryCompare()
        directoryCompareGeneration += 1
        directoryCompareBusy = false
        directoryCompareController?.closeSilently()
        directoryCompareController = nil
        GhostFTPClearDirectoryCompare()
        GhostFTPCancelRecursiveSearch()
        recursiveSearchGeneration += 1
        recursiveSearchBusy = false
        recursiveSearchController?.closeSilently()
        recursiveSearchController = nil
        GhostFTPClearRecursiveSearch()
        remoteEditGeneration += 1
        remoteEditBusy = false
        GhostFTPRemoteEditClear()
        GhostFTPCancelRemoteChmodBatch()
        setBusy(true, status: "Disconnecting…")
        remoteNavigationGeneration += 1
        remoteMutationGeneration += 1
        engineQueue.async { [weak self] in
            let ok = GhostFTPDisconnect() == 1
            let message = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                self?.remoteMutationBusy = false
                self?.remoteEditBusy = false
                self?.remoteFilterBusy = false
                self?.remoteSourceCount = 0
                self?.setBusy(false, status: ok ? "Not connected" : "Disconnect failed")
                if !ok { self?.showError(message) }
                self?.remoteItems.removeAll(keepingCapacity: true)
                self?.remoteTable.reloadData()
                self?.remotePathLabel.stringValue = ""
                self?.refreshConnectionState()
            }
        }
    }

    @objc private func chooseLocalFolder() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.canCreateDirectories = false
        panel.title = "Choose local folder"
        panel.directoryURL = URL(fileURLWithPath: localCurrent, isDirectory: true)
        if panel.runModal() == .OK, let url = panel.url {
            refreshLocal(url.path)
        }
    }

    @objc private func localUp() {
        let currentURL = URL(fileURLWithPath: localCurrent, isDirectory: true)
        refreshLocal(currentURL.deletingLastPathComponent().path)
    }

    @objc private func refreshLocalTapped() {
        refreshLocal(localCurrent)
    }

    @objc private func remoteUp() {
        refreshRemote(remoteParent(remoteCurrent))
    }

    @objc private func refreshRemoteTapped() {
        refreshRemote(remoteCurrent)
    }

    @objc private func localNewFolderTapped() {
        guard !localMutationBusy, !localFilterBusy, let name = promptName(title: "New local folder", initial: "New Folder") else { return }
        let base = localCurrent
        runLocalMutation(status: "Creating local folder…") {
            let baseValue = CStringBox(base)
            let nameValue = CStringBox(name)
            let ok = GhostFTPLocalMkdir(baseValue.pointer, nameValue.pointer) == 1
            return MutationResult(changed: ok, status: ok ? "Created local folder: \(name)" : "Local folder creation failed", error: ok ? "" : bridgeString(GhostFTPLastError()))
        }
    }

    @objc private func localRenameTapped() {
        let rows = localTable.selectedRowIndexes
        guard !localMutationBusy, !localFilterBusy, rows.count == 1, let row = rows.first, row < localItems.count else { return }
        let item = localItems[row]
        guard let newName = promptName(title: "Rename local item", initial: item.name), newName != item.name else { return }
        let base = localCurrent
        runLocalMutation(status: "Renaming local item…") {
            let baseValue = CStringBox(base)
            let oldValue = CStringBox(item.name)
            let newValue = CStringBox(newName)
            let ok = GhostFTPLocalRename(baseValue.pointer, oldValue.pointer, newValue.pointer) == 1
            return MutationResult(changed: ok, status: ok ? "Renamed: \(newName)" : "Local rename failed", error: ok ? "" : bridgeString(GhostFTPLastError()))
        }
    }

    @objc private func localDeleteTapped() {
        let selected = selectedItems(table: localTable, items: localItems)
        guard !localMutationBusy, !localFilterBusy, !selected.isEmpty, confirmDelete(selected) else { return }
        let base = localCurrent
        runLocalMutation(status: "Deleting local items…") {
            var deleted = 0
            var firstError = ""
            for item in selected {
                let baseValue = CStringBox(base)
                let nameValue = CStringBox(item.name)
                if GhostFTPLocalDelete(baseValue.pointer, nameValue.pointer) == 1 {
                    deleted += 1
                } else if firstError.isEmpty {
                    firstError = bridgeString(GhostFTPLastError())
                }
            }
            let failed = selected.count - deleted
            var status = "Deleted: \(deleted)"
            if failed > 0 { status += " • Failed: \(failed)" }
            return MutationResult(changed: deleted > 0, status: status, error: firstError)
        }
    }

    @objc private func remoteNewFolderTapped() {
        guard !remoteMutationBusy, !remoteEditBusy, !remoteFilterBusy, GhostFTPIsConnected() == 1, let name = promptName(title: "New remote folder", initial: "New Folder") else { return }
        let base = remoteCurrent
        runRemoteMutation(status: "Creating remote folder…") {
            let baseValue = CStringBox(base)
            let nameValue = CStringBox(name)
            let ok = GhostFTPRemoteMkdir(baseValue.pointer, nameValue.pointer) == 1
            return MutationResult(changed: ok, status: ok ? "Created remote folder: \(name)" : "Remote folder creation failed", error: ok ? "" : bridgeString(GhostFTPLastError()))
        }
    }

    @objc private func remoteRenameTapped() {
        let rows = remoteTable.selectedRowIndexes
        guard !remoteMutationBusy, !remoteEditBusy, !remoteFilterBusy, GhostFTPIsConnected() == 1, rows.count == 1, let row = rows.first, row < remoteItems.count else { return }
        let item = remoteItems[row]
        guard let newName = promptName(title: "Rename remote item", initial: item.name), newName != item.name else { return }
        let base = remoteCurrent
        runRemoteMutation(status: "Renaming remote item…") {
            let baseValue = CStringBox(base)
            let oldValue = CStringBox(item.name)
            let newValue = CStringBox(newName)
            let ok = GhostFTPRemoteRename(baseValue.pointer, oldValue.pointer, newValue.pointer) == 1
            return MutationResult(changed: ok, status: ok ? "Renamed: \(newName)" : "Remote rename failed", error: ok ? "" : bridgeString(GhostFTPLastError()))
        }
    }

    @objc private func remoteDeleteTapped() {
        let selected = selectedItems(table: remoteTable, items: remoteItems)
        guard !remoteMutationBusy, !remoteEditBusy, !remoteFilterBusy, GhostFTPIsConnected() == 1, !selected.isEmpty, confirmDelete(selected) else { return }
        let base = remoteCurrent
        runRemoteMutation(status: "Deleting remote items…") {
            var deleted = 0
            var firstError = ""
            for item in selected {
                let baseValue = CStringBox(base)
                let nameValue = CStringBox(item.name)
                if GhostFTPRemoteDelete(baseValue.pointer, nameValue.pointer) == 1 {
                    deleted += 1
                } else if firstError.isEmpty {
                    firstError = bridgeString(GhostFTPLastError())
                }
            }
            let failed = selected.count - deleted
            var status = "Deleted: \(deleted)"
            if failed > 0 { status += " • Failed: \(failed)" }
            return MutationResult(changed: deleted > 0, status: status, error: firstError)
        }
    }

    @objc private func remotePermissionsTapped() {
        let selected = selectedItems(table: remoteTable, items: remoteItems)
        guard !remoteMutationBusy, !remoteEditBusy, !remoteFilterBusy, GhostFTPIsConnected() == 1, !selected.isEmpty else { return }
        if selected.count > 1000 {
            statusLabel.stringValue = "Permissions: too many selected items."
            return
        }
        let actionable = selected.filter { !$0.isSymlink }
        let initiallySkipped = selected.count - actionable.count
        guard !actionable.isEmpty else {
            statusLabel.stringValue = "Skipped links: \(initiallySkipped)"
            return
        }
        guard let mode = promptPermissionMode() else { return }
        guard isValidPermissionMode(mode) else {
            showError("Permissions must contain exactly 3 or 4 octal digits (0–7).")
            return
        }
        let base = remoteCurrent
        runRemoteMutation(status: "Changing remote permissions…") {
            let baseValue = CStringBox(base)
            let modeValue = CStringBox(mode)
            guard GhostFTPBeginRemoteChmodBatch(baseValue.pointer, modeValue.pointer, CInt(actionable.count)) == 1 else {
                return MutationResult(changed: false, status: "Remote permissions change failed", error: bridgeString(GhostFTPLastError()))
            }
            defer { GhostFTPEndRemoteChmodBatch() }

            var changed = 0
            var failed = 0
            var skipped = initiallySkipped
            var firstError = ""
            for item in actionable {
                if item.isSymlink {
                    skipped += 1
                    continue
                }
                let nameValue = CStringBox(item.name)
                if GhostFTPRemoteChmodBatchItem(nameValue.pointer) == 1 {
                    changed += 1
                } else {
                    failed += 1
                    if firstError.isEmpty {
                        firstError = bridgeString(GhostFTPLastError())
                    }
                }
            }
            var status = "Changed: \(changed)"
            if failed > 0 { status += " • Failed: \(failed)" }
            if skipped > 0 { status += " • Skipped links: \(skipped)" }
            return MutationResult(changed: changed > 0, status: status, error: firstError)
        }
    }

    @objc private func remoteEditTapped() {
        let rows = remoteTable.selectedRowIndexes
        guard engineReady,
              GhostFTPIsConnected() == 1,
              !remoteMutationBusy,
              !remoteEditBusy,
              !remoteFilterBusy,
              rows.count == 1,
              let row = rows.first,
              row >= 0,
              row < remoteItems.count else { return }
        let item = remoteItems[row]
        guard !item.isDirectory, !item.isSymlink else {
            statusLabel.stringValue = "Select one regular remote file to edit."
            return
        }

        remoteEditGeneration += 1
        let editGeneration = remoteEditGeneration
        let navigationGeneration = remoteNavigationGeneration
        let base = remoteCurrent
        let name = item.name
        remoteEditBusy = true
        statusLabel.stringValue = "Opening remote file…"
        updateWorkspaceControls()
        loadRemoteEdit(base: base, name: name, navigationGeneration: navigationGeneration, editGeneration: editGeneration)
    }

    private func loadRemoteEdit(base: String, name: String, navigationGeneration: Int, editGeneration: Int) {
        engineQueue.async { [weak self] in
            let baseValue = CStringBox(base)
            let nameValue = CStringBox(name)
            let ok = GhostFTPRemoteEditOpen(baseValue.pointer, nameValue.pointer) == 1
            let text = ok ? bridgeString(GhostFTPRemoteEditText()) : ""
            let revision = ok ? bridgeString(GhostFTPRemoteEditRevision()) : ""
            let message = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self,
                      editGeneration == self.remoteEditGeneration,
                      navigationGeneration == self.remoteNavigationGeneration,
                      self.remoteCurrent == base,
                      GhostFTPIsConnected() == 1 else { return }
                if !ok {
                    self.remoteEditBusy = false
                    self.statusLabel.stringValue = "Remote edit unavailable"
                    self.showError(message)
                    self.updateWorkspaceControls()
                    return
                }
                self.statusLabel.stringValue = "Editing: \(name)"
                self.presentRemoteEditor(
                    base: base,
                    name: name,
                    text: text,
                    originalText: text,
                    expectedRevision: revision,
                    navigationGeneration: navigationGeneration,
                    editGeneration: editGeneration,
                    notice: "",
                    conflict: false
                )
            }
        }
    }

    private func presentRemoteEditor(
        base: String,
        name: String,
        text: String,
        originalText: String,
        expectedRevision: String,
        navigationGeneration: Int,
        editGeneration: Int,
        notice: String,
        conflict: Bool
    ) {
        guard editGeneration == remoteEditGeneration,
              navigationGeneration == remoteNavigationGeneration,
              remoteCurrent == base,
              GhostFTPIsConnected() == 1 else {
            finishRemoteEditSession(status: "Remote edit was cancelled because the folder changed.")
            return
        }

        let alert = NSAlert()
        alert.messageText = "Remote Edit — \(name)"
        let safety = "Built-in editor • UTF-8 text • conflict-checked • read-back verified"
        alert.informativeText = notice.isEmpty ? safety : "\(notice)\n\n\(safety)"
        alert.alertStyle = conflict ? .warning : .informational
        if conflict {
            alert.addButton(withTitle: "Reload Remote")
            alert.addButton(withTitle: "Cancel")
        } else {
            alert.addButton(withTitle: "Save")
            alert.addButton(withTitle: "Reload Remote")
            alert.addButton(withTitle: "Cancel")
        }

        let scroll = NSScrollView(frame: NSRect(x: 0, y: 0, width: 760, height: 470))
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = true
        scroll.autohidesScrollers = false
        scroll.borderType = .bezelBorder
        let textView = NSTextView(frame: scroll.contentView.bounds)
        textView.isRichText = false
        textView.importsGraphics = false
        textView.allowsUndo = true
        textView.isAutomaticQuoteSubstitutionEnabled = false
        textView.isAutomaticDashSubstitutionEnabled = false
        textView.isAutomaticTextReplacementEnabled = false
        textView.isAutomaticSpellingCorrectionEnabled = false
        textView.isContinuousSpellCheckingEnabled = false
        textView.font = .monospacedSystemFont(ofSize: 13, weight: .regular)
        textView.string = text
        textView.isEditable = !conflict
        textView.isSelectable = true
        textView.minSize = NSSize(width: 0, height: scroll.contentSize.height)
        textView.maxSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: CGFloat.greatestFiniteMagnitude)
        textView.isVerticallyResizable = true
        textView.isHorizontallyResizable = true
        textView.autoresizingMask = [.width]
        textView.textContainer?.containerSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: CGFloat.greatestFiniteMagnitude)
        textView.textContainer?.widthTracksTextView = false
        textView.setAccessibilityLabel("Remote file contents")
        scroll.documentView = textView
        alert.accessoryView = scroll
        window.makeFirstResponder(textView)

        let result = alert.runModal()
        if conflict {
            if result == .alertFirstButtonReturn {
                reloadRemoteEdit(
                    base: base,
                    name: name,
                    navigationGeneration: navigationGeneration,
                    editGeneration: editGeneration
                )
            } else {
                finishRemoteEditSession(status: "Remote edit cancelled; remote changes were preserved.")
            }
            return
        }

        if result == .alertFirstButtonReturn {
            saveRemoteEdit(
                text: textView.string,
                base: base,
                name: name,
                expectedRevision: expectedRevision,
                navigationGeneration: navigationGeneration,
                editGeneration: editGeneration
            )
            return
        }
        if result == .alertSecondButtonReturn {
            if textView.string != originalText {
                let confirm = NSAlert()
                confirm.messageText = "Discard local edits and reload?"
                confirm.informativeText = "Reloading fetches the current remote file and discards the unsaved text in this editor."
                confirm.alertStyle = .warning
                confirm.addButton(withTitle: "Reload")
                confirm.addButton(withTitle: "Keep Editing")
                if confirm.runModal() != .alertFirstButtonReturn {
                    presentRemoteEditor(
                        base: base,
                        name: name,
                        text: textView.string,
                        originalText: originalText,
                        expectedRevision: expectedRevision,
                        navigationGeneration: navigationGeneration,
                        editGeneration: editGeneration,
                        notice: "Unsaved changes kept.",
                        conflict: false
                    )
                    return
                }
            }
            reloadRemoteEdit(
                base: base,
                name: name,
                navigationGeneration: navigationGeneration,
                editGeneration: editGeneration
            )
            return
        }
        finishRemoteEditSession(status: "Remote edit cancelled.")
    }

    private func reloadRemoteEdit(base: String, name: String, navigationGeneration: Int, editGeneration: Int) {
        guard editGeneration == remoteEditGeneration,
              navigationGeneration == remoteNavigationGeneration,
              remoteCurrent == base,
              GhostFTPIsConnected() == 1 else {
            finishRemoteEditSession(status: "Remote edit was cancelled because the folder changed.")
            return
        }
        statusLabel.stringValue = "Reloading remote file…"
        loadRemoteEdit(base: base, name: name, navigationGeneration: navigationGeneration, editGeneration: editGeneration)
    }

    private func saveRemoteEdit(text: String, base: String, name: String, expectedRevision: String, navigationGeneration: Int, editGeneration: Int) {
        guard editGeneration == remoteEditGeneration,
              navigationGeneration == remoteNavigationGeneration,
              remoteCurrent == base,
              GhostFTPIsConnected() == 1 else {
            finishRemoteEditSession(status: "Remote edit was cancelled because the folder changed.")
            return
        }
        let encoded = Data(text.utf8)
        let maximum = Int64(GhostFTPRemoteEditMaxBytes())
        guard Int64(encoded.count) <= maximum else {
            presentRemoteEditor(
                base: base,
                name: name,
                text: text,
                originalText: text,
                expectedRevision: expectedRevision,
                navigationGeneration: navigationGeneration,
                editGeneration: editGeneration,
                notice: "The edited text exceeds the 4 MiB safe editor limit.",
                conflict: false
            )
            return
        }

        remoteEditBusy = true
        statusLabel.stringValue = "Saving remote file…"
        updateWorkspaceControls()
        engineQueue.async { [weak self] in
            let baseValue = CStringBox(base)
            let nameValue = CStringBox(name)
            let revisionValue = CStringBox(expectedRevision)
            let result: Int32 = encoded.withUnsafeBytes { bytes in
                let pointer = bytes.baseAddress.map { UnsafeMutableRawPointer(mutating: $0) }
                return Int32(GhostFTPRemoteEditSave(
                    baseValue.pointer,
                    nameValue.pointer,
                    revisionValue.pointer,
                    pointer,
                    CLongLong(encoded.count)
                ))
            }
            let nextRevision = result == 1 ? bridgeString(GhostFTPRemoteEditRevision()) : expectedRevision
            let message = result == 1 ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self, editGeneration == self.remoteEditGeneration else { return }
                if result == 1 {
                    self.remoteEditBusy = false
                    self.statusLabel.stringValue = "Saved: \(name)"
                    self.remoteEditGeneration += 1
                    GhostFTPRemoteEditClear()
                    if navigationGeneration == self.remoteNavigationGeneration,
                       self.remoteCurrent == base,
                       GhostFTPIsConnected() == 1 {
                        self.refreshRemote(base)
                    } else {
                        self.updateWorkspaceControls()
                    }
                    _ = nextRevision
                    return
                }
                if result == 2 {
                    self.statusLabel.stringValue = "Remote edit conflict"
                    self.presentRemoteEditor(
                        base: base,
                        name: name,
                        text: text,
                        originalText: text,
                        expectedRevision: expectedRevision,
                        navigationGeneration: navigationGeneration,
                        editGeneration: editGeneration,
                        notice: message,
                        conflict: true
                    )
                    return
                }
                self.statusLabel.stringValue = "Remote save failed"
                self.presentRemoteEditor(
                    base: base,
                    name: name,
                    text: text,
                    originalText: text,
                    expectedRevision: expectedRevision,
                    navigationGeneration: navigationGeneration,
                    editGeneration: editGeneration,
                    notice: message.isEmpty ? "The remote file could not be saved safely." : message,
                    conflict: false
                )
            }
        }
    }

    private func finishRemoteEditSession(status: String) {
        remoteEditGeneration += 1
        remoteEditBusy = false
        statusLabel.stringValue = status
        engineQueue.async { GhostFTPRemoteEditClear() }
        updateWorkspaceControls()
    }

    @objc private func localFilterTapped() {
        guard engineReady, !localMutationBusy, !localFilterBusy else { return }
        guard let query = promptCurrentFolderFilter(title: "Ghost FTP — Local", current: localFilterQuery) else { return }
        applyCurrentFolderFilter(remote: false, query: query)
    }

    @objc private func remoteFilterTapped() {
        guard engineReady, GhostFTPIsConnected() == 1, !remoteMutationBusy, !remoteEditBusy, !remoteFilterBusy else { return }
        guard let query = promptCurrentFolderFilter(title: "Ghost FTP — Remote", current: remoteFilterQuery) else { return }
        applyCurrentFolderFilter(remote: true, query: query)
    }

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

    private func promptCurrentFolderFilter(title: String, current: String) -> String? {
        let alert = NSAlert()
        alert.messageText = title
        alert.informativeText = "Filter names in the current folder. Multiple terms must all match."
        alert.addButton(withTitle: "Apply")
        alert.addButton(withTitle: "Cancel")
        let input = NSTextField(string: current)
        input.frame = NSRect(x: 0, y: 0, width: 340, height: 24)
        alert.accessoryView = input
        window.makeFirstResponder(input)
        guard alert.runModal() == .alertFirstButtonReturn else { return nil }
        return input.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func applyCurrentFolderFilter(remote: Bool, query: String) {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        let table = remote ? remoteTable : localTable
        let items = remote ? remoteItems : localItems
        let selectedItemNames = Set(selectedItems(table: table, items: items).map(\.name))
        let generation = remote ? remoteNavigationGeneration : localNavigationGeneration
        if remote {
            remoteFilterQuery = trimmed
            remoteFilterBusy = true
        } else {
            localFilterQuery = trimmed
            localFilterBusy = true
        }
        updateWorkspaceControls()

        engineQueue.async { [weak self] in
            let value = CStringBox(trimmed)
            let ok: Bool
            let visible: [FileItem]
            let sourceCount: Int
            if remote {
                ok = GhostFTPRemoteFilter(value.pointer) == 1
                visible = ok ? self?.readRemoteFilteredSnapshot() ?? [] : []
                sourceCount = ok ? max(0, Int(GhostFTPRemoteItemCount())) : 0
            } else {
                ok = GhostFTPLocalFilter(value.pointer) == 1
                visible = ok ? self?.readLocalFilteredSnapshot() ?? [] : []
                sourceCount = ok ? max(0, Int(GhostFTPLocalItemCount())) : 0
            }
            let message = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self else { return }
                if remote {
                    self.remoteFilterBusy = false
                    guard generation == self.remoteNavigationGeneration,
                          trimmed == self.remoteFilterQuery,
                          GhostFTPIsConnected() == 1 else {
                        self.updateWorkspaceControls()
                        return
                    }
                    if ok {
                        self.remoteSourceCount = sourceCount
                        self.remoteItems = visible
                        self.remoteTable.reloadData()
                        self.restoreSelection(table: self.remoteTable, items: visible, names: selectedItemNames)
                    }
                } else {
                    self.localFilterBusy = false
                    guard generation == self.localNavigationGeneration,
                          trimmed == self.localFilterQuery else {
                        self.updateWorkspaceControls()
                        return
                    }
                    if ok {
                        self.localSourceCount = sourceCount
                        self.localItems = visible
                        self.localTable.reloadData()
                        self.restoreSelection(table: self.localTable, items: visible, names: selectedItemNames)
                    }
                }
                if !ok { self.showError(message) }
                self.updateWorkspaceControls()
            }
        }
    }


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
        let operationToken = UInt64(GhostFTPPrepareDirectoryCompare())
        guard operationToken != 0 else {
            showError("Directory comparison could not be prepared safely.")
            return
        }
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
            let code = Int32(GhostFTPCompareDirectories(CUnsignedLongLong(operationToken), local.pointer, remote.pointer))
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
              controller.entry(at: index) != nil,
              GhostFTPDirectoryCompareCanOpenBoth(CInt(index)) == 1,
              GhostFTPIsConnected() == 1 else { return }
        let operationToken = UInt64(GhostFTPPrepareDirectoryCompareOpen())
        guard operationToken != 0 else {
            showError("Compared directory navigation could not be prepared safely.")
            return
        }
        directoryCompareGeneration += 1
        let generation = directoryCompareGeneration
        directoryCompareBusy = true
        statusLabel.stringValue = "Opening synchronized directory…"
        updateWorkspaceControls()

        engineQueue.async { [weak self] in
            let code = Int32(GhostFTPOpenComparedDirectoryBoth(CUnsignedLongLong(operationToken), CInt(index)))
            let localResolved = code == 1 ? bridgeString(GhostFTPLocalPath()) : ""
            let remoteResolved = code == 1 ? bridgeString(GhostFTPRemotePath()) : ""
            let localSourceCount = code == 1 ? max(0, Int(GhostFTPLocalItemCount())) : 0
            let remoteSourceCount = code == 1 ? max(0, Int(GhostFTPRemoteItemCount())) : 0
            let localItems = code == 1 ? self?.readLocalFilteredSnapshot() ?? [] : []
            let remoteItems = code == 1 ? self?.readRemoteFilteredSnapshot() ?? [] : []
            let message = code == 0 ? bridgeString(GhostFTPLastError()) : ""
            DispatchQueue.main.async {
                guard let self, generation == self.directoryCompareGeneration else { return }
                self.directoryCompareBusy = false
                switch code {
                case 1:
                    guard !localResolved.isEmpty,
                          !remoteResolved.isEmpty,
                          GhostFTPIsConnected() == 1 else {
                        self.statusLabel.stringValue = "Compared directory navigation was discarded."
                        self.updateWorkspaceControls()
                        return
                    }
                    self.localNavigationGeneration += 1
                    self.remoteNavigationGeneration += 1
                    self.localFilterQuery = ""
                    self.remoteFilterQuery = ""
                    self.localCurrent = localResolved
                    self.remoteCurrent = remoteResolved
                    self.localSourceCount = localSourceCount
                    self.remoteSourceCount = remoteSourceCount
                    self.localItems = localItems
                    self.remoteItems = remoteItems
                    self.localPathLabel.stringValue = localResolved
                    self.remotePathLabel.stringValue = remoteResolved
                    self.localTable.reloadData()
                    self.remoteTable.reloadData()
                    controller.closeSilently()
                    self.directoryCompareController = nil
                    GhostFTPClearDirectoryCompare()
                    self.statusLabel.stringValue = "Opened synchronized directory on both sides."
                case 2:
                    self.statusLabel.stringValue = "Compared directory navigation cancelled."
                default:
                    self.statusLabel.stringValue = "Could not open the compared directory."
                    self.showError(message.isEmpty ? "Both folders must remain available before either pane can navigate." : message)
                }
                self.updateWorkspaceControls()
            }
        }
    }

    private func promptPermissionMode() -> String? {
        let alert = NSAlert()
        alert.messageText = "Remote permissions"
        alert.informativeText = "Permissions (644 / 755):"
        alert.addButton(withTitle: "Apply")
        alert.addButton(withTitle: "Cancel")
        let input = NSTextField(string: "644")
        input.frame = NSRect(x: 0, y: 0, width: 180, height: 24)
        alert.accessoryView = input
        window.makeFirstResponder(input)
        guard alert.runModal() == .alertFirstButtonReturn else { return nil }
        return input.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func isValidPermissionMode(_ value: String) -> Bool {
        guard value.count == 3 || value.count == 4 else { return false }
        return value.allSatisfy { $0 >= "0" && $0 <= "7" }
    }

    private func promptName(title: String, initial: String) -> String? {
        let alert = NSAlert()
        alert.messageText = title
        alert.informativeText = "Name:"
        alert.addButton(withTitle: "OK")
        alert.addButton(withTitle: "Cancel")
        let input = NSTextField(string: initial)
        input.frame = NSRect(x: 0, y: 0, width: 320, height: 24)
        alert.accessoryView = input
        window.makeFirstResponder(input)
        guard alert.runModal() == .alertFirstButtonReturn else { return nil }
        let value = input.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty else {
            showError("Name cannot be empty.")
            return nil
        }
        return value
    }

    private func confirmDelete(_ items: [FileItem]) -> Bool {
        let alert = NSAlert()
        alert.messageText = "Delete selected item?"
        if items.count == 1 {
            alert.informativeText = items[0].name
        } else {
            alert.informativeText = "\(items.count) selected items will be deleted."
        }
        alert.alertStyle = .warning
        alert.addButton(withTitle: "Delete")
        alert.addButton(withTitle: "Cancel")
        return alert.runModal() == .alertFirstButtonReturn
    }

    private func selectedItems(table: NSTableView, items: [FileItem]) -> [FileItem] {
        table.selectedRowIndexes.compactMap { index in
            guard index >= 0, index < items.count else { return nil }
            return items[index]
        }
    }

    private func restoreSelection(table: NSTableView, items: [FileItem], names: Set<String>) {
        var indexes = IndexSet()
        for (index, item) in items.enumerated() where names.contains(item.name) {
            indexes.insert(index)
        }
        table.selectRowIndexes(indexes, byExtendingSelection: false)
    }

    private func runLocalMutation(status: String, operation: @escaping () -> MutationResult) {
        guard engineReady, !localMutationBusy, !localFilterBusy else { return }
        localMutationBusy = true
        localMutationGeneration += 1
        let mutation = localMutationGeneration
        let generation = localNavigationGeneration
        let base = localCurrent
        statusLabel.stringValue = status
        updateWorkspaceControls()
        engineQueue.async { [weak self] in
            let result = operation()
            DispatchQueue.main.async {
                guard let self, mutation == self.localMutationGeneration else { return }
                self.localMutationBusy = false
                self.statusLabel.stringValue = result.status
                if !result.error.isEmpty { self.showError(result.error) }
                if result.changed && generation == self.localNavigationGeneration {
                    self.refreshLocal(base)
                } else {
                    self.updateWorkspaceControls()
                }
            }
        }
    }

    private func runRemoteMutation(status: String, operation: @escaping () -> MutationResult) {
        guard engineReady, GhostFTPIsConnected() == 1, !remoteMutationBusy, !remoteEditBusy, !remoteFilterBusy else { return }
        remoteMutationBusy = true
        remoteMutationGeneration += 1
        let mutation = remoteMutationGeneration
        let generation = remoteNavigationGeneration
        let base = remoteCurrent
        statusLabel.stringValue = status
        updateWorkspaceControls()
        engineQueue.async { [weak self] in
            let result = operation()
            DispatchQueue.main.async {
                guard let self, mutation == self.remoteMutationGeneration else { return }
                self.remoteMutationBusy = false
                self.statusLabel.stringValue = result.status
                if !result.error.isEmpty { self.showError(result.error) }
                if result.changed && generation == self.remoteNavigationGeneration && GhostFTPIsConnected() == 1 {
                    self.refreshRemote(base)
                } else {
                    self.updateWorkspaceControls()
                }
            }
        }
    }

    @objc private func localDoubleClicked() {
        let row = localTable.clickedRow
        guard !localFilterBusy, row >= 0, row < localItems.count else { return }
        let item = localItems[row]
        if item.isDirectory && !item.isSymlink {
            refreshLocal(URL(fileURLWithPath: localCurrent, isDirectory: true).appendingPathComponent(item.name, isDirectory: true).path)
        } else if !item.isSymlink && GhostFTPIsConnected() == 1 {
            queueTransfer(direction: "upload", rows: IndexSet(integer: row))
        }
    }

    @objc private func remoteDoubleClicked() {
        let row = remoteTable.clickedRow
        guard !remoteFilterBusy, !remoteEditBusy, row >= 0, row < remoteItems.count else { return }
        let item = remoteItems[row]
        if item.isDirectory && !item.isSymlink {
            refreshRemote(remoteChild(remoteCurrent, item.name))
        } else if !item.isSymlink {
            queueTransfer(direction: "download", rows: IndexSet(integer: row))
        }
    }

    @objc private func uploadTapped() {
        guard !localFilterBusy else { return }
        queueTransfer(direction: "upload", rows: localTable.selectedRowIndexes)
    }

    @objc private func downloadTapped() {
        guard !remoteFilterBusy, !remoteEditBusy else { return }
        queueTransfer(direction: "download", rows: remoteTable.selectedRowIndexes)
    }

    private func refreshLocal(_ requestedPath: String) {
        guard engineReady else { return }
        localNavigationGeneration += 1
        let generation = localNavigationGeneration
        let target = requestedPath
        let filterQuery = localFilterQuery
        localRefreshButton.isEnabled = false
        engineQueue.async { [weak self] in
            let path = CStringBox(target)
            var ok = GhostFTPLocalList(path.pointer) == 1
            var message = ok ? "" : bridgeString(GhostFTPLastError())
            let resolved = ok ? bridgeString(GhostFTPLocalPath()) : ""
            let sourceCount = ok ? max(0, Int(GhostFTPLocalItemCount())) : 0
            var items: [FileItem] = []
            if ok {
                let query = CStringBox(filterQuery)
                ok = GhostFTPLocalFilter(query.pointer) == 1
                if ok {
                    items = self?.readLocalFilteredSnapshot() ?? []
                } else {
                    message = bridgeString(GhostFTPLastError())
                }
            }
            DispatchQueue.main.async {
                guard let self, generation == self.localNavigationGeneration else { return }
                self.localRefreshButton.isEnabled = true
                if !ok {
                    self.statusLabel.stringValue = "Local folder unavailable"
                    self.showError(message)
                    self.updateWorkspaceControls()
                    return
                }
                self.localCurrent = resolved
                self.localSourceCount = sourceCount
                if filterQuery == self.localFilterQuery {
                    self.localItems = items
                    self.localTable.reloadData()
                }
                self.localPathLabel.stringValue = resolved
                self.statusLabel.stringValue = GhostFTPIsConnected() == 1 ? "Connected" : "Not connected"
                self.updateWorkspaceControls()
            }
        }
    }

    private func refreshRemote(_ requestedPath: String) {
        guard engineReady, GhostFTPIsConnected() == 1, !remoteEditBusy else { return }
        remoteNavigationGeneration += 1
        let generation = remoteNavigationGeneration
        let target = requestedPath
        let filterQuery = remoteFilterQuery
        remoteRefreshButton.isEnabled = false
        engineQueue.async { [weak self] in
            let path = CStringBox(target)
            var ok = GhostFTPRemoteList(path.pointer) == 1
            var message = ok ? "" : bridgeString(GhostFTPLastError())
            let resolved = ok ? bridgeString(GhostFTPRemotePath()) : ""
            let sourceCount = ok ? max(0, Int(GhostFTPRemoteItemCount())) : 0
            var items: [FileItem] = []
            if ok {
                let query = CStringBox(filterQuery)
                ok = GhostFTPRemoteFilter(query.pointer) == 1
                if ok {
                    items = self?.readRemoteFilteredSnapshot() ?? []
                } else {
                    message = bridgeString(GhostFTPLastError())
                }
            }
            DispatchQueue.main.async {
                guard let self, generation == self.remoteNavigationGeneration else { return }
                self.remoteRefreshButton.isEnabled = true
                if !ok {
                    self.statusLabel.stringValue = "Remote folder unavailable"
                    self.showError(message)
                    self.updateWorkspaceControls()
                    return
                }
                self.remoteCurrent = resolved
                self.remoteSourceCount = sourceCount
                if filterQuery == self.remoteFilterQuery {
                    self.remoteItems = items
                    self.remoteTable.reloadData()
                }
                self.remotePathLabel.stringValue = resolved
                self.statusLabel.stringValue = "Connected"
                self.updateWorkspaceControls()
            }
        }
    }

    private func readLocalFilteredSnapshot() -> [FileItem] {
        let count = max(0, Int(GhostFTPLocalFilteredItemCount()))
        return (0..<count).map { index in
            let cIndex = CInt(index)
            return FileItem(
                name: bridgeString(GhostFTPLocalFilteredItemName(cIndex)),
                size: Int64(GhostFTPLocalFilteredItemSize(cIndex)),
                isDirectory: GhostFTPLocalFilteredItemIsDirectory(cIndex) == 1,
                isSymlink: GhostFTPLocalFilteredItemIsSymlink(cIndex) == 1,
                modifiedUnix: Int64(GhostFTPLocalFilteredItemModifiedUnix(cIndex)),
                permissions: ""
            )
        }
    }

    private func readRemoteFilteredSnapshot() -> [FileItem] {
        let count = max(0, Int(GhostFTPRemoteFilteredItemCount()))
        return (0..<count).map { index in
            let cIndex = CInt(index)
            return FileItem(
                name: bridgeString(GhostFTPRemoteFilteredItemName(cIndex)),
                size: Int64(GhostFTPRemoteFilteredItemSize(cIndex)),
                isDirectory: GhostFTPRemoteFilteredItemIsDirectory(cIndex) == 1,
                isSymlink: GhostFTPRemoteFilteredItemIsSymlink(cIndex) == 1,
                modifiedUnix: Int64(GhostFTPRemoteFilteredItemModifiedUnix(cIndex)),
                permissions: bridgeString(GhostFTPRemoteFilteredItemPermissions(cIndex))
            )
        }
    }

    private func queueTransfer(direction: String, rows: IndexSet) {
        guard GhostFTPIsConnected() == 1 else {
            showError("Connect to a server before transferring files.")
            return
        }
        if (direction == "upload" && localFilterBusy) || (direction == "download" && (remoteFilterBusy || remoteEditBusy)) {
            return
        }
        let items = direction == "upload" ? localItems : remoteItems
        let selected = rows.compactMap { index -> FileItem? in
            guard index >= 0, index < items.count else { return nil }
            return items[index]
        }
        guard !selected.isEmpty else {
            statusLabel.stringValue = direction == "upload" ? "Select local items to upload." : "Select remote items to download."
            return
        }
        let localBase = localCurrent
        let remoteBase = remoteCurrent
        statusLabel.stringValue = direction == "upload" ? "Queueing upload…" : "Queueing download…"
        uploadButton.isEnabled = false
        downloadButton.isEnabled = false
        engineQueue.async { [weak self] in
            var queued = 0
            var skipped = 0
            var firstError = ""
            for item in selected {
                if item.isSymlink {
                    skipped += 1
                    continue
                }
                let local = CStringBox(localBase)
                let remote = CStringBox(remoteBase)
                let name = CStringBox(item.name)
                let ok: Bool
                if direction == "upload" {
                    ok = GhostFTPUpload(local.pointer, remote.pointer, name.pointer) == 1
                } else {
                    ok = GhostFTPDownload(local.pointer, remote.pointer, name.pointer) == 1
                }
                if ok {
                    queued += 1
                } else if firstError.isEmpty {
                    firstError = bridgeString(GhostFTPLastError())
                }
            }
            DispatchQueue.main.async {
                guard let self else { return }
                var text = "Queued: \(queued)"
                if skipped > 0 { text += " • Skipped links: \(skipped)" }
                self.statusLabel.stringValue = text
                if !firstError.isEmpty { self.showError(firstError) }
                self.refreshTransferQueue(preserveSelection: true, silent: true)
                self.updateWorkspaceControls()
            }
        }
    }


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

    private func remoteChild(_ base: String, _ name: String) -> String {
        if base == "." { return name }
        if base == "/" { return "/" + name }
        let trimmed = base.hasSuffix("/") ? String(base.dropLast()) : base
        return trimmed + "/" + name
    }

    private func remoteParent(_ value: String) -> String {
        if value.isEmpty || value == "." { return "." }
        if value == "/" { return "/" }
        var clean = value
        while clean.count > 1 && clean.hasSuffix("/") { clean.removeLast() }
        guard let slash = clean.lastIndex(of: "/") else { return "." }
        if slash == clean.startIndex { return "/" }
        return String(clean[..<slash])
    }

    private func setBusy(_ busy: Bool, status: String) {
        connectionBusy = busy
        connectButton.isEnabled = !busy && engineReady && GhostFTPIsConnected() == 0
        disconnectButton.isEnabled = !busy && GhostFTPIsConnected() == 1
        protocolPopup.isEnabled = !busy
        hostField.isEnabled = !busy
        portField.isEnabled = !busy
        usernameField.isEnabled = !busy
        passwordField.isEnabled = !busy
        privateKeyField.isEnabled = !busy
        passphraseField.isEnabled = !busy
        chooseKeyButton.isEnabled = !busy
        statusLabel.stringValue = status
        updateWorkspaceControls()
    }

    private func refreshConnectionState() {
        let connected = GhostFTPIsConnected() == 1
        connectButton.isEnabled = engineReady && !connectionBusy && !connected
        disconnectButton.isEnabled = !connectionBusy && connected
        statusLabel.stringValue = connected ? "Connected" : "Not connected"
        updateWorkspaceControls()
    }

    private func updateFilterButtonLabels() {
        let localQuery = localFilterQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        let remoteQuery = remoteFilterQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        localFilterButton.title = localQuery.isEmpty ? "Filter" : "Filter · \(localItems.count)/\(localSourceCount)"
        remoteFilterButton.title = remoteQuery.isEmpty ? "Filter" : "Filter · \(remoteItems.count)/\(remoteSourceCount)"
    }

    private func updateWorkspaceControls() {
        let connected = engineReady && GhostFTPIsConnected() == 1 && !connectionBusy
        let localSelectionCount = localTable.selectedRowIndexes.count
        let remoteSelection = remoteTable.selectedRowIndexes
        let remoteSelectionCount = remoteSelection.count
        let selectedRemoteFile: FileItem? = {
            guard remoteSelectionCount == 1,
                  let index = remoteSelection.first,
                  index >= 0,
                  index < remoteItems.count else { return nil }
            return remoteItems[index]
        }()
        let localReady = engineReady && !localMutationBusy && !localFilterBusy && !recursiveSearchBusy && !directoryCompareBusy
        let remoteReady = connected && !remoteMutationBusy && !remoteEditBusy && !remoteFilterBusy && !recursiveSearchBusy && !directoryCompareBusy

        updateFilterButtonLabels()
        transferQueueButton.isEnabled = engineReady && !connectionBusy
        siteManagerButton.isEnabled = engineReady && !connectionBusy
        directoryCompareButton.isEnabled = connected && localReady && remoteReady && directoryCompareController == nil
        localChooseButton.isEnabled = localReady
        localUpButton.isEnabled = localReady && localCurrent != "/"
        localRefreshButton.isEnabled = localReady
        localNewFolderButton.isEnabled = localReady
        localRenameButton.isEnabled = localReady && localSelectionCount == 1
        localDeleteButton.isEnabled = localReady && localSelectionCount > 0
        localFilterButton.isEnabled = localReady
        localSearchButton.isEnabled = localReady
        uploadButton.isEnabled = connected && localReady && localSelectionCount > 0
        localTable.isEnabled = localReady

        remoteUpButton.isEnabled = remoteReady && remoteCurrent != "/" && remoteCurrent != "."
        remoteRefreshButton.isEnabled = remoteReady
        remoteNewFolderButton.isEnabled = remoteReady
        remoteRenameButton.isEnabled = remoteReady && remoteSelectionCount == 1
        remoteDeleteButton.isEnabled = remoteReady && remoteSelectionCount > 0
        remotePermissionsButton.isEnabled = remoteReady && remoteSelectionCount > 0
        remoteEditButton.isEnabled = remoteReady && selectedRemoteFile.map { !$0.isDirectory && !$0.isSymlink } == true
        remoteFilterButton.isEnabled = remoteReady
        remoteSearchButton.isEnabled = remoteReady
        downloadButton.isEnabled = remoteReady && remoteSelectionCount > 0
        remoteTable.isEnabled = remoteReady
    }

    func numberOfRows(in tableView: NSTableView) -> Int {
        tableView === localTable ? localItems.count : remoteItems.count
    }

    func tableView(_ tableView: NSTableView, viewFor tableColumn: NSTableColumn?, row: Int) -> NSView? {
        let items = tableView === localTable ? localItems : remoteItems
        guard row >= 0, row < items.count, let column = tableColumn else { return nil }
        let item = items[row]
        let value: String
        switch column.identifier.rawValue {
        case "name":
            let prefix = item.isDirectory ? "[Folder] " : item.isSymlink ? "[Link] " : ""
            value = prefix + item.name
        case "size":
            value = item.isDirectory ? "—" : byteFormatter.string(fromByteCount: item.size)
        case "modified":
            value = item.modifiedUnix > 0 ? dateFormatter.string(from: Date(timeIntervalSince1970: TimeInterval(item.modifiedUnix))) : ""
        case "permissions":
            value = item.permissions
        default:
            value = ""
        }

        let cell = NSTableCellView()
        let text = NSTextField(labelWithString: value)
        text.textColor = Palette.text
        text.lineBreakMode = column.identifier.rawValue == "name" ? .byTruncatingMiddle : .byTruncatingTail
        text.translatesAutoresizingMaskIntoConstraints = false
        cell.addSubview(text)
        NSLayoutConstraint.activate([
            text.leadingAnchor.constraint(equalTo: cell.leadingAnchor, constant: 5),
            text.trailingAnchor.constraint(equalTo: cell.trailingAnchor, constant: -5),
            text.centerYAnchor.constraint(equalTo: cell.centerYAnchor)
        ])
        return cell
    }

    func tableViewSelectionDidChange(_ notification: Notification) {
        updateWorkspaceControls()
    }

    private func showError(_ text: String) {
        let alert = NSAlert()
        alert.messageText = "Ghost FTP"
        alert.informativeText = text.isEmpty ? "The operation could not be completed." : text
        alert.alertStyle = .critical
        alert.runModal()
    }
}


private final class SiteManagerWindowController: NSWindowController, NSWindowDelegate, NSTableViewDataSource, NSTableViewDelegate {
    private let engineQueue: DispatchQueue
    private let onConnected: (String, String) -> Void
    private var profiles: [SavedProfileView] = []
    private var selectedProfileID = ""
    private var busy = false

    private let table = NSTableView(frame: .zero)
    private let nameField = NSTextField(string: "")
    private let protocolPopup = NSPopUpButton(frame: .zero, pullsDown: false)
    private let hostField = NSTextField(string: "")
    private let portField = NSTextField(string: "21")
    private let usernameField = NSTextField(string: "")
    private let passwordField = NSSecureTextField(string: "")
    private let privateKeyField = NSTextField(string: "")
    private let passphraseField = NSSecureTextField(string: "")
    private let remotePathField = NSTextField(string: "/")
    private let localPathField = NSTextField(string: NSHomeDirectory())
    private let passwordState = NSTextField(labelWithString: "")
    private let passphraseState = NSTextField(labelWithString: "")
    private let statusLabel = NSTextField(labelWithString: "")
    private let newButton = NSButton(title: "New", target: nil, action: nil)
    private let saveButton = NSButton(title: "Save Profile", target: nil, action: nil)
    private let removeButton = NSButton(title: "Remove Profile", target: nil, action: nil)
    private let connectButton = NSButton(title: "Connect", target: nil, action: nil)
    private let browseKeyButton = NSButton(title: "Browse…", target: nil, action: nil)
    private let closeButton = NSButton(title: "Close", target: nil, action: nil)

    init(engineQueue: DispatchQueue, onConnected: @escaping (String, String) -> Void) {
        self.engineQueue = engineQueue
        self.onConnected = onConnected
        let panel = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1040, height: 610),
            styleMask: [.titled, .closable, .resizable],
            backing: .buffered,
            defer: false
        )
        panel.title = "Ghost FTP — Site Manager"
        panel.minSize = NSSize(width: 900, height: 540)
        super.init(window: panel)
        panel.delegate = self
        buildUI()
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    private func label(_ value: String) -> NSTextField {
        let field = NSTextField(labelWithString: value)
        field.textColor = Palette.text
        field.font = .systemFont(ofSize: 12, weight: .medium)
        return field
    }

    private func addColumn(_ id: String, _ title: String, _ width: CGFloat) {
        let column = NSTableColumn(identifier: NSUserInterfaceItemIdentifier(id))
        column.title = title
        column.width = width
        column.minWidth = 80
        column.resizingMask = id == "name" ? [.autoresizingMask, .userResizingMask] : [.userResizingMask]
        table.addTableColumn(column)
    }

    private func buildUI() {
        guard let content = window?.contentView else { return }
        content.wantsLayer = true
        content.layer?.backgroundColor = Palette.workspace.cgColor

        protocolPopup.addItems(withTitles: ["FTP", "FTPS", "SFTP"])
        protocolPopup.target = self
        protocolPopup.action = #selector(protocolChanged)

        table.dataSource = self
        table.delegate = self
        table.allowsEmptySelection = true
        table.allowsMultipleSelection = false
        table.headerView = NSTableHeaderView()
        table.backgroundColor = Palette.list
        table.rowSizeStyle = .medium
        addColumn("name", "Name", 170)
        addColumn("protocol", "Protocol", 75)
        addColumn("host", "Host", 180)
        addColumn("username", "Username", 130)

        let scroll = NSScrollView()
        scroll.documentView = table
        scroll.hasVerticalScroller = true
        scroll.borderType = .bezelBorder
        scroll.translatesAutoresizingMaskIntoConstraints = false

        passwordField.placeholderString = "Enter a new password to replace the saved one"
        passphraseField.placeholderString = "Enter a new passphrase to replace the saved one"
        passwordState.textColor = Palette.muted
        passphraseState.textColor = Palette.muted
        statusLabel.textColor = Palette.muted
        statusLabel.lineBreakMode = .byTruncatingTail

        let keyRow = NSStackView(views: [privateKeyField, browseKeyButton])
        keyRow.orientation = .horizontal
        keyRow.spacing = 6
        privateKeyField.widthAnchor.constraint(greaterThanOrEqualToConstant: 250).isActive = true

        let form = NSGridView(views: [
            [label("Name"), nameField],
            [label("Protocol"), protocolPopup],
            [label("Host"), hostField],
            [label("Port"), portField],
            [label("Username"), usernameField],
            [label("Password"), passwordField],
            [NSView(), passwordState],
            [label("Private key"), keyRow],
            [label("Passphrase"), passphraseField],
            [NSView(), passphraseState],
            [label("Remote path"), remotePathField],
            [label("Local path"), localPathField],
        ])
        form.rowSpacing = 8
        form.columnSpacing = 10
        form.column(at: 0).xPlacement = .trailing
        form.translatesAutoresizingMaskIntoConstraints = false

        newButton.target = self
        newButton.action = #selector(newTapped)
        saveButton.target = self
        saveButton.action = #selector(saveTapped)
        removeButton.target = self
        removeButton.action = #selector(removeTapped)
        connectButton.target = self
        connectButton.action = #selector(connectTapped)
        browseKeyButton.target = self
        browseKeyButton.action = #selector(browseKeyTapped)
        closeButton.target = self
        closeButton.action = #selector(closeTapped)

        let actions = NSStackView(views: [newButton, saveButton, removeButton, connectButton, closeButton])
        actions.orientation = .horizontal
        actions.spacing = 8

        let right = NSStackView(views: [form, statusLabel, actions])
        right.orientation = .vertical
        right.alignment = .leading
        right.spacing = 12
        right.translatesAutoresizingMaskIntoConstraints = false

        let split = NSStackView(views: [scroll, right])
        split.orientation = .horizontal
        split.alignment = .top
        split.spacing = 16
        split.edgeInsets = NSEdgeInsets(top: 16, left: 16, bottom: 16, right: 16)
        split.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(split)
        NSLayoutConstraint.activate([
            split.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            split.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            split.topAnchor.constraint(equalTo: content.topAnchor),
            split.bottomAnchor.constraint(equalTo: content.bottomAnchor),
            scroll.widthAnchor.constraint(greaterThanOrEqualToConstant: 390),
            scroll.heightAnchor.constraint(equalTo: split.heightAnchor, constant: -32),
            form.widthAnchor.constraint(greaterThanOrEqualToConstant: 500),
        ])
        protocolChanged()
        updateControls()
    }

    func showAndRefresh() {
        showWindow(nil)
        window?.center()
        window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        refreshProfiles(preserveID: selectedProfileID)
    }

    private func setBusy(_ value: Bool, status: String) {
        busy = value
        statusLabel.stringValue = status
        updateControls()
    }

    private func updateControls() {
        let hasSelection = !selectedProfileID.isEmpty
        newButton.isEnabled = !busy
        saveButton.isEnabled = !busy
        removeButton.isEnabled = !busy && hasSelection
        connectButton.isEnabled = !busy && hasSelection && GhostFTPIsConnected() == 0
        browseKeyButton.isEnabled = !busy && protocolPopup.titleOfSelectedItem == "SFTP"
        table.isEnabled = !busy
    }

    @objc private func protocolChanged() {
        let sftp = protocolPopup.titleOfSelectedItem == "SFTP"
        privateKeyField.isEnabled = sftp && !busy
        passphraseField.isEnabled = sftp && !busy
        passphraseState.isHidden = !sftp
        if sftp && portField.stringValue == "21" { portField.stringValue = "22" }
        if !sftp && portField.stringValue == "22" { portField.stringValue = "21" }
        if remotePathField.stringValue.isEmpty || remotePathField.stringValue == "/" || remotePathField.stringValue == "." {
            remotePathField.stringValue = sftp ? "." : "/"
        }
        updateControls()
    }

    @objc private func browseKeyTapped() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false
        panel.title = "Choose private key"
        if panel.runModal() == .OK, let url = panel.url { privateKeyField.stringValue = url.path }
    }

    private func readProfilesSnapshot() -> [SavedProfileView] {
        let count = max(0, Int(GhostFTPProfileCount()))
        return (0..<count).map { index in
            SavedProfileView(
                id: bridgeString(GhostFTPProfileID(CInt(index))),
                name: bridgeString(GhostFTPProfileName(CInt(index))),
                protocolName: bridgeString(GhostFTPProfileProtocol(CInt(index))),
                host: bridgeString(GhostFTPProfileHost(CInt(index))),
                port: Int32(GhostFTPProfilePort(CInt(index))),
                username: bridgeString(GhostFTPProfileUsername(CInt(index))),
                HasPassword: GhostFTPProfileHasPassword(CInt(index)) == 1,
                privateKeyPath: bridgeString(GhostFTPProfilePrivateKeyPath(CInt(index))),
                HasPassphrase: GhostFTPProfileHasPassphrase(CInt(index)) == 1,
                remotePath: bridgeString(GhostFTPProfileRemotePath(CInt(index))),
                localPath: bridgeString(GhostFTPProfileLocalPath(CInt(index)))
            )
        }
    }

    private func refreshProfiles(preserveID: String) {
        guard !busy else { return }
        setBusy(true, status: "Loading saved profiles…")
        engineQueue.async { [weak self] in
            let ok = GhostFTPRefreshProfiles() == 1
            let snapshot = ok ? self?.readProfilesSnapshot() : nil
            let error = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self else { return }
                self.busy = false
                guard let snapshot else {
                    self.statusLabel.stringValue = "Saved profiles could not be loaded."
                    self.showError(error)
                    self.updateControls()
                    return
                }
                self.profiles = snapshot
                self.table.reloadData()
                if let row = snapshot.firstIndex(where: { $0.id == preserveID }) {
                    self.table.selectRowIndexes(IndexSet(integer: row), byExtendingSelection: false)
                    self.applyProfile(snapshot[row])
                } else if let first = snapshot.first {
                    self.table.selectRowIndexes(IndexSet(integer: 0), byExtendingSelection: false)
                    self.applyProfile(first)
                } else {
                    self.clearEditorForNewProfile()
                }
                self.statusLabel.stringValue = "\(snapshot.count) saved profile(s)."
                self.updateControls()
            }
        }
    }

    private func applyProfile(_ profile: SavedProfileView) {
        selectedProfileID = profile.id
        nameField.stringValue = profile.name
        protocolPopup.selectItem(withTitle: profile.protocolName.uppercased())
        hostField.stringValue = profile.host
        portField.stringValue = String(profile.port)
        usernameField.stringValue = profile.username
        passwordField.stringValue = ""
        passwordState.stringValue = profile.HasPassword ? "Password saved securely in macOS Keychain." : "No saved password."
        privateKeyField.stringValue = profile.privateKeyPath
        passphraseField.stringValue = ""
        passphraseState.stringValue = profile.HasPassphrase ? "Passphrase saved securely in macOS Keychain." : "No saved passphrase."
        remotePathField.stringValue = profile.remotePath
        localPathField.stringValue = profile.localPath
        protocolChanged()
    }

    private func clearEditorForNewProfile() {
        selectedProfileID = ""
        table.deselectAll(nil)
        nameField.stringValue = ""
        protocolPopup.selectItem(withTitle: "FTP")
        hostField.stringValue = ""
        portField.stringValue = "21"
        usernameField.stringValue = ""
        passwordField.stringValue = ""
        passwordState.stringValue = "No saved password."
        privateKeyField.stringValue = ""
        passphraseField.stringValue = ""
        passphraseState.stringValue = "No saved passphrase."
        remotePathField.stringValue = "/"
        localPathField.stringValue = NSHomeDirectory()
        protocolChanged()
    }

    @objc private func newTapped() { clearEditorForNewProfile(); updateControls() }

    private enum CredentialDecision { case keepOrSave, remove, cancel }

    private func credentialDecision(existing: SavedProfileView?, hasNewSecret: Bool) -> CredentialDecision {
        if hasNewSecret {
            let alert = NSAlert()
            alert.messageText = "Save credentials on this computer?"
            alert.informativeText = "Ghost FTP will encrypt the credential with a key protected by your macOS Keychain. Choose Don't Save to store only the profile metadata."
            alert.alertStyle = .warning
            alert.addButton(withTitle: "Save Securely")
            alert.addButton(withTitle: "Don't Save")
            alert.addButton(withTitle: "Cancel")
            switch alert.runModal() {
            case .alertFirstButtonReturn: return .keepOrSave
            case .alertSecondButtonReturn: return .remove
            default: return .cancel
            }
        }
        if existing?.HasPassword == true || existing?.HasPassphrase == true {
            let alert = NSAlert()
            alert.messageText = "Keep saved credentials on this computer?"
            alert.informativeText = "The existing Keychain-protected credential can be retained for this profile or removed now."
            alert.alertStyle = .warning
            alert.addButton(withTitle: "Keep Securely")
            alert.addButton(withTitle: "Remove Saved Credentials")
            alert.addButton(withTitle: "Cancel")
            switch alert.runModal() {
            case .alertFirstButtonReturn: return .keepOrSave
            case .alertSecondButtonReturn: return .remove
            default: return .cancel
            }
        }
        return .remove
    }

    @objc private func saveTapped() {
        guard !busy else { return }
        let name = nameField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let host = hostField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty, !host.isEmpty else { showError("Profile name and host are required."); return }
        guard let port = Int32(portField.stringValue), port > 0, port <= 65535 else { showError("Port must be between 1 and 65535."); return }

        let existing = profiles.first(where: { $0.id == selectedProfileID })
        var password = passwordField.stringValue
        var passphrase = passphraseField.stringValue
        let hasNewSecret = !password.isEmpty || !passphrase.isEmpty
        let decision = credentialDecision(existing: existing, hasNewSecret: hasNewSecret)
        if decision == .cancel { password.removeAll(); passphrase.removeAll(); return }
        if decision == .remove { password.removeAll(); passphrase.removeAll() }
        let clearPassword = decision == .remove
        let clearPassphrase = decision == .remove

        let profileID = selectedProfileID
        let protocolName = (protocolPopup.titleOfSelectedItem ?? "FTP").lowercased()
        let username = usernameField.stringValue
        let privateKey = protocolName == "sftp" ? privateKeyField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines) : ""
        let remotePath = remotePathField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let localPath = localPathField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        passwordField.stringValue = ""
        passphraseField.stringValue = ""
        setBusy(true, status: "Saving profile…")

        engineQueue.async { [weak self] in
            let cID = CStringBox(profileID)
            let cName = CStringBox(name)
            let cProtocol = CStringBox(protocolName)
            let cHost = CStringBox(host)
            let cUser = CStringBox(username)
            let cPassword = CStringBox(password)
            let cKey = CStringBox(privateKey)
            let cPassphrase = CStringBox(passphrase)
            let cRemote = CStringBox(remotePath)
            let cLocal = CStringBox(localPath)
            let ok = GhostFTPSaveProfile(
                cID.pointer, cName.pointer, cProtocol.pointer, cHost.pointer, CInt(port), cUser.pointer,
                cPassword.pointer, clearPassword ? 1 : 0, cKey.pointer, cPassphrase.pointer,
                clearPassphrase ? 1 : 0, cRemote.pointer, cLocal.pointer
            ) == 1
            password.removeAll(keepingCapacity: false)
            passphrase.removeAll(keepingCapacity: false)
            let snapshot = ok ? self?.readProfilesSnapshot() : nil
            let error = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self else { return }
                self.busy = false
                if let snapshot {
                    self.profiles = snapshot
                    self.table.reloadData()
                    let candidate = snapshot.first(where: { $0.id == profileID }) ?? snapshot.first(where: { $0.name == name && $0.host == host })
                    if let candidate, let row = snapshot.firstIndex(where: { $0.id == candidate.id }) {
                        self.table.selectRowIndexes(IndexSet(integer: row), byExtendingSelection: false)
                        self.applyProfile(candidate)
                    }
                    self.statusLabel.stringValue = "Profile saved."
                } else {
                    self.statusLabel.stringValue = "Profile save failed."
                    self.showError(error)
                }
                self.updateControls()
            }
        }
    }

    @objc private func removeTapped() {
        guard !busy, !selectedProfileID.isEmpty else { return }
        let profileID = selectedProfileID
        let alert = NSAlert()
        alert.messageText = "Remove Profile"
        alert.informativeText = "Remove this saved profile and its stored credential references?"
        alert.alertStyle = .warning
        alert.addButton(withTitle: "Remove")
        alert.addButton(withTitle: "Cancel")
        guard alert.runModal() == .alertFirstButtonReturn else { return }
        setBusy(true, status: "Removing profile…")
        engineQueue.async { [weak self] in
            let cID = CStringBox(profileID)
            let ok = GhostFTPRemoveProfile(cID.pointer) == 1
            let snapshot = ok ? self?.readProfilesSnapshot() : nil
            let error = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self else { return }
                self.busy = false
                if let snapshot {
                    self.profiles = snapshot
                    self.table.reloadData()
                    if let first = snapshot.first {
                        self.table.selectRowIndexes(IndexSet(integer: 0), byExtendingSelection: false)
                        self.applyProfile(first)
                    } else {
                        self.clearEditorForNewProfile()
                    }
                    self.statusLabel.stringValue = "Profile removed."
                } else {
                    self.statusLabel.stringValue = "Profile removal failed."
                    self.showError(error)
                }
                self.updateControls()
            }
        }
    }

    @objc private func connectTapped() {
        guard !busy, let profile = profiles.first(where: { $0.id == selectedProfileID }) else { return }
        guard GhostFTPIsConnected() == 0 else { showError("Disconnect the current server before connecting a saved profile."); return }
        setBusy(true, status: "Connecting saved profile…")
        connect(profile: profile, trustFingerprint: "")
    }

    private func connect(profile: SavedProfileView, trustFingerprint: String) {
        engineQueue.async { [weak self] in
            let cID = CStringBox(profile.id)
            let cTrust = CStringBox(trustFingerprint)
            let result = Int32(GhostFTPConnectProfile(cID.pointer, cTrust.pointer, 1))
            let fingerprint = result == 2 ? bridgeString(GhostFTPPendingFingerprint()) : ""
            let error = result == 0 ? bridgeString(GhostFTPLastError()) : ""
            DispatchQueue.main.async {
                guard let self else { return }
                if result == 1 {
                    self.busy = false
                    self.statusLabel.stringValue = "Connected."
                    self.updateControls()
                    self.onConnected(profile.localPath, profile.remotePath)
                    self.close()
                    return
                }
                if result == 2, !fingerprint.isEmpty {
                    let alert = NSAlert()
                    alert.messageText = "Trust this SFTP host key?"
                    alert.informativeText = fingerprint
                    alert.alertStyle = .warning
                    alert.addButton(withTitle: "Trust, remember and connect")
                    alert.addButton(withTitle: "Cancel")
                    if alert.runModal() == .alertFirstButtonReturn {
                        self.statusLabel.stringValue = "Verifying host key…"
                        self.connect(profile: profile, trustFingerprint: fingerprint)
                    } else {
                        GhostFTPCancelPendingTrust()
                        self.busy = false
                        self.statusLabel.stringValue = "Connection cancelled."
                        self.updateControls()
                    }
                    return
                }
                self.busy = false
                self.statusLabel.stringValue = "Connection failed."
                self.showError(error.isEmpty ? "The saved profile could not connect." : error)
                self.updateControls()
            }
        }
    }

    @objc private func closeTapped() { close() }

    func numberOfRows(in tableView: NSTableView) -> Int { profiles.count }

    func tableView(_ tableView: NSTableView, viewFor tableColumn: NSTableColumn?, row: Int) -> NSView? {
        guard row >= 0, row < profiles.count, let column = tableColumn else { return nil }
        let profile = profiles[row]
        let value: String
        switch column.identifier.rawValue {
        case "name": value = profile.name
        case "protocol": value = profile.protocolName.uppercased()
        case "host": value = profile.host + ":" + String(profile.port)
        case "username": value = profile.username
        default: value = ""
        }
        let cell = NSTableCellView()
        let text = NSTextField(labelWithString: value)
        text.textColor = Palette.text
        text.lineBreakMode = .byTruncatingMiddle
        text.translatesAutoresizingMaskIntoConstraints = false
        cell.addSubview(text)
        NSLayoutConstraint.activate([
            text.leadingAnchor.constraint(equalTo: cell.leadingAnchor, constant: 5),
            text.trailingAnchor.constraint(equalTo: cell.trailingAnchor, constant: -5),
            text.centerYAnchor.constraint(equalTo: cell.centerYAnchor),
        ])
        return cell
    }

    func tableViewSelectionDidChange(_ notification: Notification) {
        let row = table.selectedRow
        guard row >= 0, row < profiles.count else { selectedProfileID = ""; updateControls(); return }
        applyProfile(profiles[row])
        updateControls()
    }

    private func showError(_ message: String) {
        let alert = NSAlert()
        alert.messageText = "Ghost FTP"
        alert.informativeText = message.isEmpty ? "The operation could not be completed." : message
        alert.alertStyle = .critical
        alert.runModal()
    }
}

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

private let app = NSApplication.shared
private let delegate = AppDelegate()
app.delegate = delegate
app.run()
