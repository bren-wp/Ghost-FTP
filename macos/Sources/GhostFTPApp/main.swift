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
    private var localMutationBusy = false
    private var remoteMutationBusy = false
    private var localFilterBusy = false
    private var remoteFilterBusy = false

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
    private let uploadButton = NSButton(title: "Upload", target: nil, action: nil)
    private let remoteUpButton = NSButton(title: "Up", target: nil, action: nil)
    private let remoteRefreshButton = NSButton(title: "Refresh", target: nil, action: nil)
    private let remoteNewFolderButton = NSButton(title: "New Folder", target: nil, action: nil)
    private let remoteRenameButton = NSButton(title: "Rename", target: nil, action: nil)
    private let remoteDeleteButton = NSButton(title: "Delete", target: nil, action: nil)
    private let remotePermissionsButton = NSButton(title: "Permissions", target: nil, action: nil)
    private let remoteFilterButton = NSButton(title: "Filter", target: nil, action: nil)
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
        let buttonRow = NSStackView(views: [connectButton, disconnectButton, statusLabel])
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
            buttons: [localChooseButton, localUpButton, localRefreshButton, localNewFolderButton, localRenameButton, localDeleteButton, localFilterButton, uploadButton]
        ))
        split.addArrangedSubview(makeFilePane(
            title: "Remote",
            table: remoteTable,
            pathLabel: remotePathLabel,
            buttons: [remoteUpButton, remoteRefreshButton, remoteNewFolderButton, remoteRenameButton, remoteDeleteButton, remotePermissionsButton, remoteFilterButton, downloadButton]
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
        remoteFilterButton.target = self
        remoteFilterButton.action = #selector(remoteFilterTapped)
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
        setBusy(true, status: "Disconnecting…")
        remoteNavigationGeneration += 1
        remoteMutationGeneration += 1
        engineQueue.async { [weak self] in
            let ok = GhostFTPDisconnect() == 1
            let message = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                self?.remoteMutationBusy = false
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
        guard !remoteMutationBusy, !remoteFilterBusy, GhostFTPIsConnected() == 1, let name = promptName(title: "New remote folder", initial: "New Folder") else { return }
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
        guard !remoteMutationBusy, !remoteFilterBusy, GhostFTPIsConnected() == 1, rows.count == 1, let row = rows.first, row < remoteItems.count else { return }
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
        guard !remoteMutationBusy, !remoteFilterBusy, GhostFTPIsConnected() == 1, !selected.isEmpty, confirmDelete(selected) else { return }
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
        guard !remoteMutationBusy, !remoteFilterBusy, GhostFTPIsConnected() == 1, !selected.isEmpty else { return }
        if selected.count > 1000 {
            statusLabel.stringValue = "Permissions: too many selected items."
            return
        }
        guard let mode = promptPermissionMode() else { return }
        guard isValidPermissionMode(mode) else {
            showError("Permissions must contain exactly 3 or 4 octal digits (0–7).")
            return
        }
        let base = remoteCurrent
        runRemoteMutation(status: "Changing remote permissions…") {
            var changed = 0
            var failed = 0
            var skipped = 0
            var firstError = ""
            for item in selected {
                if item.isSymlink {
                    skipped += 1
                    continue
                }
                let baseValue = CStringBox(base)
                let nameValue = CStringBox(item.name)
                let modeValue = CStringBox(mode)
                if GhostFTPRemoteChmod(baseValue.pointer, nameValue.pointer, modeValue.pointer) == 1 {
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

    @objc private func localFilterTapped() {
        guard engineReady, !localMutationBusy, !localFilterBusy else { return }
        guard let query = promptCurrentFolderFilter(title: "Ghost FTP — Local", current: localFilterQuery) else { return }
        applyCurrentFolderFilter(remote: false, query: query)
    }

    @objc private func remoteFilterTapped() {
        guard engineReady, GhostFTPIsConnected() == 1, !remoteMutationBusy, !remoteFilterBusy else { return }
        guard let query = promptCurrentFolderFilter(title: "Ghost FTP — Remote", current: remoteFilterQuery) else { return }
        applyCurrentFolderFilter(remote: true, query: query)
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
        guard engineReady, GhostFTPIsConnected() == 1, !remoteMutationBusy, !remoteFilterBusy else { return }
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
        guard !remoteFilterBusy, row >= 0, row < remoteItems.count else { return }
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
        guard !remoteFilterBusy else { return }
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
        guard engineReady, GhostFTPIsConnected() == 1 else { return }
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
        if (direction == "upload" && localFilterBusy) || (direction == "download" && remoteFilterBusy) {
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
        let remoteSelectionCount = remoteTable.selectedRowIndexes.count
        let localReady = engineReady && !localMutationBusy && !localFilterBusy
        let remoteReady = connected && !remoteMutationBusy && !remoteFilterBusy

        updateFilterButtonLabels()
        localChooseButton.isEnabled = localReady
        localUpButton.isEnabled = localReady && localCurrent != "/"
        localRefreshButton.isEnabled = localReady
        localNewFolderButton.isEnabled = localReady
        localRenameButton.isEnabled = localReady && localSelectionCount == 1
        localDeleteButton.isEnabled = localReady && localSelectionCount > 0
        localFilterButton.isEnabled = localReady
        uploadButton.isEnabled = connected && localReady && localSelectionCount > 0
        localTable.isEnabled = localReady

        remoteUpButton.isEnabled = remoteReady && remoteCurrent != "/" && remoteCurrent != "."
        remoteRefreshButton.isEnabled = remoteReady
        remoteNewFolderButton.isEnabled = remoteReady
        remoteRenameButton.isEnabled = remoteReady && remoteSelectionCount == 1
        remoteDeleteButton.isEnabled = remoteReady && remoteSelectionCount > 0
        remotePermissionsButton.isEnabled = remoteReady && remoteSelectionCount > 0
        remoteFilterButton.isEnabled = remoteReady
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

private let app = NSApplication.shared
private let delegate = AppDelegate()
app.delegate = delegate
app.run()
