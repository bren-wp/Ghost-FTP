import AppKit
import Darwin
import Foundation
import GhostFTPEngine

private enum Palette {
    static let workspace = NSColor(rgb: 0xEEF1F5)
    static let panel = NSColor(rgb: 0xF6F8FB)
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

private func bridgeString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {
    guard let pointer else { return "" }
    defer { GhostFTPFreeCString(pointer) }
    return String(cString: pointer)
}

private final class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate {
    private var window: NSWindow!
    private let engineQueue = DispatchQueue(label: "app.ghostftp.engine", qos: .userInitiated)

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
    private let workspaceLabel = NSTextField(labelWithString: "Connect to a server to open the local and remote file workspace.")

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        buildWindow()
        if GhostFTPCreateEngine() != 1 {
            statusLabel.stringValue = bridgeString(GhostFTPLastError())
            connectButton.isEnabled = false
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
            contentRect: NSRect(x: 0, y: 0, width: 1320, height: 820),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Ghost FTP"
        window.minSize = NSSize(width: 1040, height: 680)
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

        workspaceLabel.font = .systemFont(ofSize: 15, weight: .medium)
        workspaceLabel.textColor = Palette.muted
        workspaceLabel.alignment = .center
        let workspace = NSView()
        workspace.wantsLayer = true
        workspace.layer?.backgroundColor = NSColor.white.cgColor
        workspace.layer?.cornerRadius = 10
        workspace.layer?.borderWidth = 1
        workspace.layer?.borderColor = Palette.border.cgColor
        workspace.addSubview(workspaceLabel)
        workspaceLabel.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            workspaceLabel.centerXAnchor.constraint(equalTo: workspace.centerXAnchor),
            workspaceLabel.centerYAnchor.constraint(equalTo: workspace.centerYAnchor)
        ])

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
            workspace.heightAnchor.constraint(greaterThanOrEqualToConstant: 360),
            connectionStack.widthAnchor.constraint(equalTo: root.widthAnchor, constant: -36)
        ])
        protocolChanged()
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
        guard var input = currentInput() else { return }
        passwordField.stringValue = ""
        passphraseField.stringValue = ""
        setBusy(true, status: "Connecting…")
        engineQueue.async { [weak self] in
            let result = self?.bridgeConnect(input: input, trustFingerprint: "") ?? 0
            input.clearSecrets()
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
            setBusy(false, status: "Connected")
            workspaceLabel.stringValue = "Connected. File workspace parity is the next implementation layer."
            refreshConnectionState()
            return
        }
        if result == 2 {
            let fingerprint = bridgeString(GhostFTPPendingFingerprint())
            guard !fingerprint.isEmpty else {
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
                trusted.password = ""
                trusted.passphrase = ""
                setBusy(true, status: "Verifying host key…")
                engineQueue.async { [weak self] in
                    let second = self?.bridgeConnect(input: trusted, trustFingerprint: fingerprint) ?? 0
                    trusted.clearSecrets()
                    DispatchQueue.main.async { self?.handleConnectResult(second, original: trusted) }
                }
            } else {
                GhostFTPCancelPendingTrust()
                setBusy(false, status: "Not connected")
            }
            return
        }
        setBusy(false, status: "Connection failed")
        showError(bridgeString(GhostFTPLastError()))
        refreshConnectionState()
    }

    @objc private func disconnectTapped() {
        setBusy(true, status: "Disconnecting…")
        engineQueue.async { [weak self] in
            let ok = GhostFTPDisconnect() == 1
            let message = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                self?.setBusy(false, status: ok ? "Not connected" : "Disconnect failed")
                if !ok { self?.showError(message) }
                self?.workspaceLabel.stringValue = "Connect to a server to open the local and remote file workspace."
                self?.refreshConnectionState()
            }
        }
    }

    private func setBusy(_ busy: Bool, status: String) {
        connectButton.isEnabled = !busy && GhostFTPIsConnected() == 0
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
    }

    private func refreshConnectionState() {
        let connected = GhostFTPIsConnected() == 1
        connectButton.isEnabled = !connected
        disconnectButton.isEnabled = connected
        statusLabel.stringValue = connected ? "Connected" : "Not connected"
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
