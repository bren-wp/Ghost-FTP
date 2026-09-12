#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def replace_once(path: str, old: str, new: str) -> None:
    p = ROOT / path
    source = p.read_text(encoding="utf-8")
    count = source.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one anchor, found {count}: {old[:100]!r}")
    p.write_text(source.replace(old, new, 1), encoding="utf-8")


# Darwin now has a real persistent credential capability. Runtime credentials
# remain the existing short-lived broker; this flag only advertises durable
# profile support to shared capability checks.
replace_once(
    "internal/security/dpapi_darwin.go",
    "func PersistentSecretStorageAvailable() bool { return false }",
    "func PersistentSecretStorageAvailable() bool { return true }",
)

# A saved macOS profile credential is converted to a fresh runtime capability
# only after shared profile/account/private-key binding has matched.
replace_once(
    "internal/remote/manager.go",
    '''\t\tif in.Password == "" && profileAccountMatches(p, resolved.Config) {\n\t\t\tresolved.PasswordBlob = p.PasswordBlob\n\t\t}\n\t\tif in.Passphrase == "" && profilePrivateKeyMatches(p, resolved.Config) {\n\t\t\tresolved.PassphraseBlob = p.PassphraseBlob\n\t\t}\n''',
    '''\t\tif in.Password == "" && p.PasswordBlob != "" && profileAccountMatches(p, resolved.Config) {\n\t\t\truntimeBlob, convertErr := security.PersistentProfileSecretToRuntime(p.PasswordBlob)\n\t\t\tif convertErr != nil {\n\t\t\t\treturn resolved, profile, convertErr\n\t\t\t}\n\t\t\tresolved.PasswordBlob = runtimeBlob\n\t\t\tresolved.ownsPasswordBlob = runtime.GOOS == "darwin" && runtimeBlob != ""\n\t\t}\n\t\tif in.Passphrase == "" && p.PassphraseBlob != "" && profilePrivateKeyMatches(p, resolved.Config) {\n\t\t\truntimeBlob, convertErr := security.PersistentProfileSecretToRuntime(p.PassphraseBlob)\n\t\t\tif convertErr != nil {\n\t\t\t\tresolved.forgetOwnedSecrets()\n\t\t\t\treturn resolved, profile, convertErr\n\t\t\t}\n\t\t\tresolved.PassphraseBlob = runtimeBlob\n\t\t\tresolved.ownsPassphraseBlob = runtime.GOOS == "darwin" && runtimeBlob != ""\n\t\t}\n''',
)

# Transfer ownership to the transport that actually keeps using the ephemeral
# runtime handle after Connect returns.
replace_once(
    "internal/remote/manager.go",
    '''func transferResolvedSecretOwnershipToSFTP(resolved *resolvedConnection, s *SFTP) {\n\tif resolved == nil || s == nil {\n\t\treturn\n\t}\n\tif resolved.ownsPasswordBlob && resolved.PasswordBlob != "" && s.passwordBlob == resolved.PasswordBlob {\n\t\ts.ownsPasswordBlob = true\n\t\tresolved.ownsPasswordBlob = false\n\t}\n\tif resolved.ownsPassphraseBlob && resolved.PassphraseBlob != "" && s.passphraseBlob == resolved.PassphraseBlob {\n\t\ts.ownsPassphraseBlob = true\n\t\tresolved.ownsPassphraseBlob = false\n\t}\n}\n''',
    '''func transferResolvedSecretOwnershipToSFTP(resolved *resolvedConnection, s *SFTP) {\n\tif resolved == nil || s == nil {\n\t\treturn\n\t}\n\tif resolved.ownsPasswordBlob && resolved.PasswordBlob != "" && s.passwordBlob == resolved.PasswordBlob {\n\t\ts.ownsPasswordBlob = true\n\t\tresolved.ownsPasswordBlob = false\n\t}\n\tif resolved.ownsPassphraseBlob && resolved.PassphraseBlob != "" && s.passphraseBlob == resolved.PassphraseBlob {\n\t\ts.ownsPassphraseBlob = true\n\t\tresolved.ownsPassphraseBlob = false\n\t}\n}\n\nfunc transferResolvedSecretOwnershipToCurl(resolved *resolvedConnection, s *CurlFTP) {\n\tif resolved == nil || s == nil {\n\t\treturn\n\t}\n\tif resolved.ownsPasswordBlob && resolved.PasswordBlob != "" && s.passwordBlob == resolved.PasswordBlob {\n\t\tresolved.ownsPasswordBlob = false\n\t}\n}\n''',
)

# Pending host-key trust must adopt the converted runtime handles rather than
# leave them owned by the first Connect call's deferred cleanup.
replace_once(
    "internal/remote/manager.go",
    '''func (m *Manager) stashPendingTrust(cfg model.ConnectionConfig, resolved resolvedConnection, fingerprint string) error {\n\tm.clearPendingTrustLocked()\n\tpasswordBlob := resolved.PasswordBlob\n\tpassphraseBlob := resolved.PassphraseBlob\n\townsPasswordBlob := false\n\townsPassphraseBlob := false\n''',
    '''func (m *Manager) stashPendingTrust(cfg model.ConnectionConfig, resolved *resolvedConnection, fingerprint string) error {\n\tm.clearPendingTrustLocked()\n\tpasswordBlob := ""\n\tpassphraseBlob := ""\n\tadoptPasswordBlob := false\n\tadoptPassphraseBlob := false\n\tif resolved != nil {\n\t\tpasswordBlob = resolved.PasswordBlob\n\t\tpassphraseBlob = resolved.PassphraseBlob\n\t\tadoptPasswordBlob = cfg.Password == "" && resolved.ownsPasswordBlob && passwordBlob != ""\n\t\tadoptPassphraseBlob = cfg.Passphrase == "" && resolved.ownsPassphraseBlob && passphraseBlob != ""\n\t}\n\townsPasswordBlob := adoptPasswordBlob\n\townsPassphraseBlob := adoptPassphraseBlob\n''',
)
replace_once(
    "internal/remote/manager.go",
    '''\tm.pendingTrust = pendingTrustState{\n\t\tendpointKey:        profilebinding.EndpointKey(cfg.Protocol, cfg.Host, cfg.Port),\n\t\tusername:           cfg.Username,\n\t\tkeyPath:            cfg.PrivateKeyPath,\n\t\tfingerprint:        fingerprint,\n\t\tpasswordBlob:       passwordBlob,\n\t\tpassphraseBlob:     passphraseBlob,\n\t\townsPasswordBlob:   ownsPasswordBlob,\n\t\townsPassphraseBlob: ownsPassphraseBlob,\n\t\texpires:            time.Now().Add(2 * time.Minute),\n\t}\n\treturn nil\n}\n''',
    '''\tm.pendingTrust = pendingTrustState{\n\t\tendpointKey:        profilebinding.EndpointKey(cfg.Protocol, cfg.Host, cfg.Port),\n\t\tusername:           cfg.Username,\n\t\tkeyPath:            cfg.PrivateKeyPath,\n\t\tfingerprint:        fingerprint,\n\t\tpasswordBlob:       passwordBlob,\n\t\tpassphraseBlob:     passphraseBlob,\n\t\townsPasswordBlob:   ownsPasswordBlob,\n\t\townsPassphraseBlob: ownsPassphraseBlob,\n\t\texpires:            time.Now().Add(2 * time.Minute),\n\t}\n\tif resolved != nil {\n\t\tif adoptPasswordBlob {\n\t\t\tresolved.ownsPasswordBlob = false\n\t\t}\n\t\tif adoptPassphraseBlob {\n\t\t\tresolved.ownsPassphraseBlob = false\n\t\t}\n\t}\n\treturn nil\n}\n''',
)
replace_once(
    "internal/remote/manager.go",
    "if err := m.stashPendingTrust(cfg, resolved, fp); err != nil {",
    "if err := m.stashPendingTrust(cfg, &resolved, fp); err != nil {",
)
replace_once(
    "internal/remote/manager.go",
    '''\t} else {\n\t\ts, err = newCurlFTPWithProtectedSecret(cfg.Protocol, cfg.Host, cfg.Port, cfg.Username, cfg.Password, resolved.PasswordBlob, connectTimeout)\n\t\tif err != nil {\n\t\t\treturn ConnectResult{}, err\n\t\t}\n\t}\n''',
    '''\t} else {\n\t\tcurlSession, curlErr := newCurlFTPWithProtectedSecret(cfg.Protocol, cfg.Host, cfg.Port, cfg.Username, cfg.Password, resolved.PasswordBlob, connectTimeout)\n\t\tif curlErr != nil {\n\t\t\treturn ConnectResult{}, curlErr\n\t\t}\n\t\ttransferResolvedSecretOwnershipToCurl(&resolved, curlSession)\n\t\ts = curlSession\n\t}\n''',
)

# Swift model and AppDelegate integration.
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    '''private struct TransferQueueEntry {\n    let id: String\n    let direction: String\n    let localPath: String\n    let remotePath: String\n    let status: String\n    let progress: Double\n    let bytesTransferred: Int64\n    let bytesTotal: Int64\n    let bytesPerSecond: Double\n    let etaSeconds: Int64\n    let attempts: Int\n    let error: String\n}\n\nprivate func bridgeString''',
    '''private struct TransferQueueEntry {\n    let id: String\n    let direction: String\n    let localPath: String\n    let remotePath: String\n    let status: String\n    let progress: Double\n    let bytesTransferred: Int64\n    let bytesTotal: Int64\n    let bytesPerSecond: Double\n    let etaSeconds: Int64\n    let attempts: Int\n    let error: String\n}\n\nprivate struct SavedProfileView {\n    let id: String\n    let name: String\n    let protocolName: String\n    let host: String\n    let port: Int32\n    let username: String\n    let HasPassword: Bool\n    let privateKeyPath: String\n    let HasPassphrase: Bool\n    let remotePath: String\n    let localPath: String\n}\n\nprivate func bridgeString''',
)
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    '''    private var transferQueueController: TransferQueueWindowController?\n    private var seenDoneTransferIDs: Set<String> = []\n''',
    '''    private var transferQueueController: TransferQueueWindowController?\n    private var siteManagerController: SiteManagerWindowController?\n    private var seenDoneTransferIDs: Set<String> = []\n''',
)
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    '''    private let transferQueueButton = NSButton(title: "Transfers", target: nil, action: nil)\n    private let statusLabel''',
    '''    private let transferQueueButton = NSButton(title: "Transfers", target: nil, action: nil)\n    private let siteManagerButton = NSButton(title: "Site Manager", target: nil, action: nil)\n    private let statusLabel''',
)
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    '''        transferQueueController?.closeSilently()\n        transferQueueController = nil\n        GhostFTPClearTransferQueueSnapshot()\n        GhostFTPShutdown()\n''',
    '''        transferQueueController?.closeSilently()\n        transferQueueController = nil\n        siteManagerController?.close()\n        siteManagerController = nil\n        GhostFTPClearProfilesSnapshot()\n        GhostFTPClearTransferQueueSnapshot()\n        GhostFTPShutdown()\n''',
)
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    '''        transferQueueButton.target = self\n        transferQueueButton.action = #selector(transferQueueTapped)\n\n        configureWorkspaceActions()''',
    '''        transferQueueButton.target = self\n        transferQueueButton.action = #selector(transferQueueTapped)\n        siteManagerButton.target = self\n        siteManagerButton.action = #selector(siteManagerTapped)\n\n        configureWorkspaceActions()''',
)
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    '''        let buttonRow = NSStackView(views: [connectButton, disconnectButton, directoryCompareButton, transferQueueButton, statusLabel])''',
    '''        let buttonRow = NSStackView(views: [connectButton, disconnectButton, siteManagerButton, directoryCompareButton, transferQueueButton, statusLabel])''',
)
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    '''    @objc private func disconnectTapped() {''',
    '''    @objc private func siteManagerTapped() {\n        guard engineReady, !connectionBusy else { return }\n        if siteManagerController == nil {\n            siteManagerController = SiteManagerWindowController(engineQueue: engineQueue) { [weak self] localPath, remotePath in\n                guard let self else { return }\n                self.setBusy(false, status: "Connected")\n                self.refreshConnectionState()\n                if !localPath.isEmpty {\n                    self.refreshLocal(localPath)\n                }\n                let targetRemote = remotePath.isEmpty ? "/" : remotePath\n                self.remoteCurrent = targetRemote\n                self.refreshRemote(targetRemote)\n            }\n        }\n        siteManagerController?.showAndRefresh()\n    }\n\n    @objc private func disconnectTapped() {''',
)
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    '''        transferQueueButton.isEnabled = engineReady && !connectionBusy\n        directoryCompareButton.isEnabled''',
    '''        transferQueueButton.isEnabled = engineReady && !connectionBusy\n        siteManagerButton.isEnabled = engineReady && !connectionBusy\n        directoryCompareButton.isEnabled''',
)

site_manager_class = r'''
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

'''
replace_once(
    "macos/Sources/GhostFTPApp/main.swift",
    "\n\n\n\nprivate final class DirectoryCompareWindowController",
    "\n\n" + site_manager_class + "private final class DirectoryCompareWindowController",
)

# Promote only genuinely implemented actions in this feature.
for action in ("Site Manager", "Save Profile", "Remove Profile"):
    replace_once("macos/PARITY.md", f"- [ ] {action}", f"- [x] {action}")

replace_once(
    "macos/PARITY.md",
    "The macOS client is currently a development source platform.",
    "The macOS client is currently a development source platform. Saved profiles use a native Security.framework Keychain-protected 256-bit wrapping key; durable profile credentials are converted to short-lived runtime capabilities only inside the shared engine after exact profile binding succeeds. Site Manager exposes PublicProfile metadata only, requires explicit credential persistence/retention consent, and never returns saved password or passphrase blobs to Swift.",
)

replace_once(
    "scripts/test_macos_windows_parity_contract.py",
    '''IMPLEMENTED_MACOS_ACTIONS = {\n    "Connect",\n    "Disconnect",\n    "Private Key",''',
    '''IMPLEMENTED_MACOS_ACTIONS = {\n    "Connect",\n    "Disconnect",\n    "Site Manager",\n    "Save Profile",\n    "Remove Profile",\n    "Private Key",''',
)

print("macOS Site Manager guarded patch applied")
