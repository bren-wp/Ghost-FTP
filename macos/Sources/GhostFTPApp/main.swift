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
        let buttonRow = NSStackView(views: [connectButton, disconnectButton, directoryCompareButton, statusLabel])
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
                self.updateWorkspaceControls()
            }
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


private let app = NSApplication.shared
private let delegate = AppDelegate()
app.delegate = delegate
app.run()
