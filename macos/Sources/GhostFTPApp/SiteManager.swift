import AppKit
import Darwin
import Foundation
import GhostFTPEngine

private final class SiteCStringBox {
    let pointer: UnsafeMutablePointer<CChar>?
    init(_ value: String) { pointer = strdup(value) }
    deinit { free(pointer) }
}

private func siteBridgeString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {
    guard let pointer else { return "" }
    defer { GhostFTPFreeCString(pointer) }
    return String(cString: pointer)
}

private struct SiteProfileView {
    let id: String
    let name: String
    let protocolName: String
    let host: String
    let port: Int32
    let username: String
    let hasPassword: Bool
    let privateKeyPath: String
    let hasPassphrase: Bool
    let fingerprint: String
    let remotePath: String
    let localPath: String
}

final class SiteManagerWindowController: NSWindowController, NSTableViewDataSource, NSTableViewDelegate {
    var onConnected: (() -> Void)?

    private let engineQueue = DispatchQueue(label: "app.ghostftp.site-manager", qos: .userInitiated)
    private let table = NSTableView(frame: .zero)
    private var profiles: [SiteProfileView] = []
    private var selectedProfile: SiteProfileView?
    private var connectionBusy = false

    private let nameField = NSTextField(string: "")
    private let protocolPopup = NSPopUpButton(frame: .zero, pullsDown: false)
    private let hostField = NSTextField(string: "")
    private let portField = NSTextField(string: "21")
    private let usernameField = NSTextField(string: "")
    private let passwordField = NSSecureTextField(string: "")
    private let privateKeyField = NSTextField(string: "")
    private let passphraseField = NSSecureTextField(string: "")
    private let remotePathField = NSTextField(string: "/")
    private let localPathField = NSTextField(string: "")
    private let saveCredentialsButton = NSButton(checkboxWithTitle: "Save credentials on this computer?", target: nil, action: nil)
    private let rememberFingerprintButton = NSButton(checkboxWithTitle: "Remember trusted SFTP host key", target: nil, action: nil)
    private let statusLabel = NSTextField(labelWithString: "")
    private let saveButton = NSButton(title: "Save Profile", target: nil, action: nil)
    private let removeButton = NSButton(title: "Remove Profile", target: nil, action: nil)
    private let connectButton = NSButton(title: "Connect", target: nil, action: nil)

    init() {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 880, height: 570),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Site Manager"
        window.minSize = NSSize(width: 760, height: 500)
        super.init(window: window)
        buildUI()
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    func showAndRefresh() {
        reloadProfiles()
        showWindow(nil)
        window?.center()
        window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    private func buildUI() {
        guard let content = window?.contentView else { return }
        protocolPopup.addItems(withTitles: ["FTP", "FTPS", "SFTP"])
        protocolPopup.target = self
        protocolPopup.action = #selector(protocolChanged)
        saveCredentialsButton.toolTip = "Saved credentials stay on this Mac and are protected by macOS Keychain-backed encryption."
        rememberFingerprintButton.state = .on

        let nameColumn = NSTableColumn(identifier: NSUserInterfaceItemIdentifier("name"))
        nameColumn.title = "Saved sites"
        nameColumn.width = 220
        table.addTableColumn(nameColumn)
        table.headerView = nil
        table.delegate = self
        table.dataSource = self
        table.usesAlternatingRowBackgroundColors = true
        table.allowsEmptySelection = true

        let scroll = NSScrollView()
        scroll.documentView = table
        scroll.hasVerticalScroller = true
        scroll.borderType = .bezelBorder

        let newButton = NSButton(title: "New", target: self, action: #selector(newProfile))
        let refreshButton = NSButton(title: "Refresh", target: self, action: #selector(refreshTapped))
        let listButtons = NSStackView(views: [newButton, refreshButton])
        listButtons.orientation = .horizontal
        listButtons.spacing = 8
        let listStack = NSStackView(views: [scroll, listButtons])
        listStack.orientation = .vertical
        listStack.spacing = 8
        listStack.widthAnchor.constraint(equalToConstant: 240).isActive = true

        let browseKeyButton = NSButton(title: "Browse…", target: self, action: #selector(browsePrivateKey))
        let keyRow = NSStackView(views: [privateKeyField, browseKeyButton])
        keyRow.orientation = .horizontal
        keyRow.spacing = 8

        let grid = NSGridView(views: [
            [label("Name"), nameField],
            [label("Protocol"), protocolPopup],
            [label("Host"), hostField],
            [label("Port"), portField],
            [label("Username"), usernameField],
            [label("Password"), passwordField],
            [label("Private key"), keyRow],
            [label("Passphrase"), passphraseField],
            [label("Remote start"), remotePathField],
            [label("Local start"), localPathField]
        ])
        grid.rowSpacing = 9
        grid.columnSpacing = 12
        grid.column(at: 0).xPlacement = .trailing
        grid.column(at: 1).width = 430

        saveButton.target = self
        saveButton.action = #selector(saveProfile)
        removeButton.target = self
        removeButton.action = #selector(removeProfile)
        connectButton.target = self
        connectButton.action = #selector(connectProfile)
        connectButton.keyEquivalent = "\r"
        let actionRow = NSStackView(views: [saveButton, removeButton, connectButton])
        actionRow.orientation = .horizontal
        actionRow.spacing = 8

        statusLabel.textColor = .secondaryLabelColor
        statusLabel.lineBreakMode = .byTruncatingTail
        let formStack = NSStackView(views: [grid, saveCredentialsButton, rememberFingerprintButton, actionRow, statusLabel])
        formStack.orientation = .vertical
        formStack.alignment = .leading
        formStack.spacing = 12

        let root = NSStackView(views: [listStack, formStack])
        root.orientation = .horizontal
        root.alignment = .top
        root.spacing = 18
        root.edgeInsets = NSEdgeInsets(top: 18, left: 18, bottom: 18, right: 18)
        root.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(root)
        NSLayoutConstraint.activate([
            root.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            root.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            root.topAnchor.constraint(equalTo: content.topAnchor),
            root.bottomAnchor.constraint(equalTo: content.bottomAnchor),
            scroll.heightAnchor.constraint(greaterThanOrEqualToConstant: 430),
            hostField.widthAnchor.constraint(greaterThanOrEqualToConstant: 380)
        ])
        clearEditor()
    }

    private func label(_ value: String) -> NSTextField {
        let field = NSTextField(labelWithString: value)
        field.textColor = .secondaryLabelColor
        return field
    }

    func numberOfRows(in tableView: NSTableView) -> Int { profiles.count }

    func tableView(_ tableView: NSTableView, viewFor tableColumn: NSTableColumn?, row: Int) -> NSView? {
        guard row >= 0 && row < profiles.count else { return nil }
        let id = NSUserInterfaceItemIdentifier("site-cell")
        let cell: NSTableCellView
        if let reused = tableView.makeView(withIdentifier: id, owner: self) as? NSTableCellView {
            cell = reused
        } else {
            cell = NSTableCellView()
            cell.identifier = id
            let text = NSTextField(labelWithString: "")
            text.translatesAutoresizingMaskIntoConstraints = false
            cell.textField = text
            cell.addSubview(text)
            NSLayoutConstraint.activate([
                text.leadingAnchor.constraint(equalTo: cell.leadingAnchor, constant: 6),
                text.trailingAnchor.constraint(equalTo: cell.trailingAnchor, constant: -6),
                text.centerYAnchor.constraint(equalTo: cell.centerYAnchor)
            ])
        }
        cell.textField?.stringValue = profiles[row].name
        return cell
    }

    func tableViewSelectionDidChange(_ notification: Notification) {
        let row = table.selectedRow
        guard row >= 0 && row < profiles.count else {
            selectedProfile = nil
            clearEditor()
            return
        }
        loadEditor(profiles[row])
    }

    private func readProfile(_ index: Int32) -> SiteProfileView {
        SiteProfileView(
            id: siteBridgeString(GhostFTPProfileID(index)),
            name: siteBridgeString(GhostFTPProfileName(index)),
            protocolName: siteBridgeString(GhostFTPProfileProtocol(index)),
            host: siteBridgeString(GhostFTPProfileHost(index)),
            port: GhostFTPProfilePort(index),
            username: siteBridgeString(GhostFTPProfileUsername(index)),
            hasPassword: GhostFTPProfileHasPassword(index) == 1,
            privateKeyPath: siteBridgeString(GhostFTPProfilePrivateKeyPath(index)),
            hasPassphrase: GhostFTPProfileHasPassphrase(index) == 1,
            fingerprint: siteBridgeString(GhostFTPProfileFingerprint(index)),
            remotePath: siteBridgeString(GhostFTPProfileRemotePath(index)),
            localPath: siteBridgeString(GhostFTPProfileLocalPath(index))
        )
    }

    private func reloadProfiles(selectID: String? = nil) {
        guard GhostFTPRefreshProfiles() == 1 else {
            statusLabel.stringValue = siteBridgeString(GhostFTPLastError())
            return
        }
        let count = max(0, Int(GhostFTPProfileCount()))
        profiles = (0..<count).map { readProfile(Int32($0)) }
        table.reloadData()
        if let selectID, let index = profiles.firstIndex(where: { $0.id == selectID }) {
            table.selectRowIndexes(IndexSet(integer: index), byExtendingSelection: false)
            table.scrollRowToVisible(index)
            loadEditor(profiles[index])
        } else if table.selectedRow >= profiles.count {
            table.deselectAll(nil)
            clearEditor()
        }
        statusLabel.stringValue = profiles.isEmpty ? "No saved sites yet." : "Saved sites: \(profiles.count)"
    }

    private func clearEditor() {
        selectedProfile = nil
        nameField.stringValue = ""
        protocolPopup.selectItem(withTitle: "FTP")
        hostField.stringValue = ""
        portField.stringValue = "21"
        usernameField.stringValue = ""
        passwordField.stringValue = ""
        privateKeyField.stringValue = ""
        passphraseField.stringValue = ""
        remotePathField.stringValue = "/"
        localPathField.stringValue = ""
        saveCredentialsButton.title = "Save credentials on this computer?"
        saveCredentialsButton.state = .off
        removeButton.isEnabled = false
        connectButton.isEnabled = false
        protocolChanged()
    }

    private func loadEditor(_ profile: SiteProfileView) {
        selectedProfile = profile
        nameField.stringValue = profile.name
        protocolPopup.selectItem(withTitle: profile.protocolName.uppercased())
        hostField.stringValue = profile.host
        portField.stringValue = String(profile.port)
        usernameField.stringValue = profile.username
        passwordField.stringValue = ""
        privateKeyField.stringValue = profile.privateKeyPath
        passphraseField.stringValue = ""
        remotePathField.stringValue = profile.remotePath
        localPathField.stringValue = profile.localPath
        let hasSavedCredentials = profile.hasPassword || profile.hasPassphrase
        saveCredentialsButton.title = hasSavedCredentials ? "Keep saved credentials on this computer?" : "Save credentials on this computer?"
        saveCredentialsButton.state = hasSavedCredentials ? .on : .off
        removeButton.isEnabled = true
        connectButton.isEnabled = true
        protocolChanged()
    }

    @objc private func protocolChanged() {
        let sftp = protocolPopup.titleOfSelectedItem == "SFTP"
        privateKeyField.isEnabled = sftp
        passphraseField.isEnabled = sftp
        rememberFingerprintButton.isHidden = !sftp
        if !sftp {
            privateKeyField.stringValue = ""
            passphraseField.stringValue = ""
        }
        if selectedProfile == nil {
            portField.stringValue = sftp ? "22" : "21"
            remotePathField.stringValue = sftp ? "." : "/"
        }
    }

    @objc private func newProfile() {
        table.deselectAll(nil)
        clearEditor()
        nameField.becomeFirstResponder()
    }

    @objc private func refreshTapped() { reloadProfiles(selectID: selectedProfile?.id) }

    @objc private func browsePrivateKey() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false
        if panel.runModal() == .OK, let url = panel.url {
            privateKeyField.stringValue = url.path
        }
    }

    private func fingerprintForSave(protocolName: String, host: String, port: Int32) -> String {
        guard let existing = selectedProfile else { return "" }
        guard protocolName == "sftp",
              existing.protocolName.lowercased() == protocolName,
              existing.host.caseInsensitiveCompare(host) == .orderedSame,
              existing.port == port else { return "" }
        return existing.fingerprint
    }

    @objc private func saveProfile() {
        let name = nameField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let protocolName = (protocolPopup.titleOfSelectedItem ?? "FTP").lowercased()
        let host = hostField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let port = Int32(portField.stringValue), port > 0 && port <= 65535 else {
            statusLabel.stringValue = "Enter a valid port from 1 to 65535."
            return
        }
        let fingerprint = fingerprintForSave(protocolName: protocolName, host: host, port: port)
        let id = SiteCStringBox(selectedProfile?.id ?? "")
        let cName = SiteCStringBox(name)
        let cProtocol = SiteCStringBox(protocolName)
        let cHost = SiteCStringBox(host)
        let cUsername = SiteCStringBox(usernameField.stringValue)
        let cPassword = SiteCStringBox(passwordField.stringValue)
        let cKey = SiteCStringBox(privateKeyField.stringValue)
        let cPassphrase = SiteCStringBox(passphraseField.stringValue)
        let cFingerprint = SiteCStringBox(fingerprint)
        let cRemote = SiteCStringBox(remotePathField.stringValue)
        let cLocal = SiteCStringBox(localPathField.stringValue)
        let keep = saveCredentialsButton.state == .on ? Int32(1) : Int32(0)
        let ok = GhostFTPSaveProfile(
            id.pointer, cName.pointer, cProtocol.pointer, cHost.pointer, port,
            cUsername.pointer, cPassword.pointer, keep, cKey.pointer, cPassphrase.pointer,
            cFingerprint.pointer, cRemote.pointer, cLocal.pointer
        )
        passwordField.stringValue = ""
        passphraseField.stringValue = ""
        if ok != 1 {
            statusLabel.stringValue = siteBridgeString(GhostFTPLastError())
            return
        }
        reloadProfiles()
        if let saved = profiles.first(where: { $0.name == name && $0.host.caseInsensitiveCompare(host) == .orderedSame && $0.port == port }) {
            reloadProfiles(selectID: saved.id)
        }
        statusLabel.stringValue = "Profile saved securely on this Mac."
    }

    @objc private func removeProfile() {
        guard let profile = selectedProfile else { return }
        let alert = NSAlert()
        alert.messageText = "Remove Profile"
        alert.informativeText = "Remove \"\(profile.name)\" from this Mac?"
        alert.addButton(withTitle: "Remove")
        alert.addButton(withTitle: "Cancel")
        guard alert.runModal() == .alertFirstButtonReturn else { return }
        let id = SiteCStringBox(profile.id)
        if GhostFTPRemoveProfile(id.pointer) != 1 {
            statusLabel.stringValue = siteBridgeString(GhostFTPLastError())
            return
        }
        reloadProfiles()
        clearEditor()
        statusLabel.stringValue = "Profile removed."
    }

    @objc private func connectProfile() {
        guard let profile = selectedProfile, !connectionBusy else { return }
        connectionBusy = true
        connectButton.isEnabled = false
        statusLabel.stringValue = "Connecting…"
        let profileID = profile.id
        let password = passwordField.stringValue
        let passphrase = passphraseField.stringValue
        let remember = rememberFingerprintButton.state == .on
        passwordField.stringValue = ""
        passphraseField.stringValue = ""
        runConnect(profileID: profileID, password: password, passphrase: passphrase, trust: "", remember: remember)
    }

    private func runConnect(profileID: String, password: String, passphrase: String, trust: String, remember: Bool) {
        engineQueue.async { [weak self] in
            let id = SiteCStringBox(profileID)
            let pwd = SiteCStringBox(password)
            let phrase = SiteCStringBox(passphrase)
            let trustValue = SiteCStringBox(trust)
            let result = GhostFTPConnectProfile(id.pointer, pwd.pointer, phrase.pointer, trustValue.pointer, remember ? 1 : 0)
            let error = result == 0 ? siteBridgeString(GhostFTPLastError()) : ""
            let fingerprint = result == 2 ? siteBridgeString(GhostFTPPendingFingerprint()) : ""
            DispatchQueue.main.async {
                guard let self else { return }
                if result == 2 {
                    self.confirmFingerprint(profileID: profileID, password: password, passphrase: passphrase, fingerprint: fingerprint, remember: remember)
                    return
                }
                self.connectionBusy = false
                self.connectButton.isEnabled = self.selectedProfile != nil
                if result == 1 {
                    self.statusLabel.stringValue = "Connected."
                    self.onConnected?()
                    self.close()
                } else {
                    self.statusLabel.stringValue = error.isEmpty ? "Connection failed." : error
                }
            }
        }
    }

    private func confirmFingerprint(profileID: String, password: String, passphrase: String, fingerprint: String, remember: Bool) {
        let alert = NSAlert()
        alert.messageText = "Verify SFTP server identity"
        alert.informativeText = "Verify this fingerprint through an independent trusted channel (for example, your hosting control panel or administrator). Accept only if it matches exactly.\n\n\(fingerprint)"
        alert.addButton(withTitle: "Trust and Connect")
        alert.addButton(withTitle: "Cancel")
        if alert.runModal() == .alertFirstButtonReturn {
            runConnect(profileID: profileID, password: password, passphrase: passphrase, trust: fingerprint, remember: remember)
        } else {
            GhostFTPCancelPendingTrust()
            connectionBusy = false
            connectButton.isEnabled = selectedProfile != nil
            statusLabel.stringValue = "Connection cancelled."
        }
    }
}
