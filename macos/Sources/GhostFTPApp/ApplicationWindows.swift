import AppKit
import Darwin
import Foundation
import GhostFTPEngine

private final class ApplicationCStringBox {
    let pointer: UnsafeMutablePointer<CChar>?
    init(_ value: String) { pointer = strdup(value) }
    deinit { free(pointer) }
}

private func applicationBridgeString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {
    guard let pointer else { return "" }
    defer { GhostFTPFreeCString(pointer) }
    return String(cString: pointer)
}

private func applicationLabel(_ value: String, size: CGFloat = 12, weight: NSFont.Weight = .regular) -> NSTextField {
    let field = NSTextField(labelWithString: value)
    field.font = .systemFont(ofSize: size, weight: weight)
    field.textColor = .labelColor
    field.lineBreakMode = .byWordWrapping
    field.maximumNumberOfLines = 0
    return field
}

private struct BookmarkView {
    let id: String
    let name: String
    let kind: String
    let path: String
    let host: String
    let username: String
}

final class BookmarksWindowController: NSWindowController, NSTableViewDataSource, NSTableViewDelegate {
    var onNavigation: ((String, String) -> Void)?

    private let engineQueue = DispatchQueue(label: "app.ghostftp.bookmarks", qos: .userInitiated)
    private let table = NSTableView(frame: .zero)
    private let statusLabel = NSTextField(labelWithString: "")
    private let openButton = NSButton(title: "Open", target: nil, action: nil)
    private let addLocalButton = NSButton(title: "Add Local", target: nil, action: nil)
    private let addRemoteButton = NSButton(title: "Add Remote", target: nil, action: nil)
    private let deleteButton = NSButton(title: "Delete", target: nil, action: nil)
    private var items: [BookmarkView] = []
    private var busy = false

    init() {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 760, height: 500),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Bookmarks"
        window.minSize = NSSize(width: 620, height: 400)
        super.init(window: window)
        buildUI()
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    func showAndRefresh() {
        reloadBookmarks()
        showWindow(nil)
        window?.center()
        window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    private func buildUI() {
        guard let content = window?.contentView else { return }
        let column = NSTableColumn(identifier: NSUserInterfaceItemIdentifier("bookmark"))
        column.title = "Bookmarks"
        column.width = 690
        table.addTableColumn(column)
        table.headerView = nil
        table.delegate = self
        table.dataSource = self
        table.allowsEmptySelection = true
        table.usesAlternatingRowBackgroundColors = true
        table.target = self
        table.doubleAction = #selector(openSelected)

        let scroll = NSScrollView()
        scroll.documentView = table
        scroll.hasVerticalScroller = true
        scroll.borderType = .bezelBorder

        openButton.target = self
        openButton.action = #selector(openSelected)
        addLocalButton.target = self
        addLocalButton.action = #selector(addLocal)
        addRemoteButton.target = self
        addRemoteButton.action = #selector(addRemote)
        deleteButton.target = self
        deleteButton.action = #selector(deleteSelected)
        let closeButton = NSButton(title: "Close", target: self, action: #selector(closeTapped))

        let actions = NSStackView(views: [openButton, addLocalButton, addRemoteButton, deleteButton, closeButton])
        actions.orientation = .horizontal
        actions.spacing = 8
        statusLabel.textColor = .secondaryLabelColor
        statusLabel.lineBreakMode = .byTruncatingTail
        let stack = NSStackView(views: [scroll, actions, statusLabel])
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 10
        stack.edgeInsets = NSEdgeInsets(top: 16, left: 16, bottom: 16, right: 16)
        stack.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            stack.topAnchor.constraint(equalTo: content.topAnchor),
            stack.bottomAnchor.constraint(equalTo: content.bottomAnchor),
            scroll.widthAnchor.constraint(equalTo: stack.widthAnchor, constant: -32),
            scroll.heightAnchor.constraint(greaterThanOrEqualToConstant: 330)
        ])
        updateButtons()
    }

    func numberOfRows(in tableView: NSTableView) -> Int { items.count }

    func tableView(_ tableView: NSTableView, viewFor tableColumn: NSTableColumn?, row: Int) -> NSView? {
        guard row >= 0, row < items.count else { return nil }
        let identifier = NSUserInterfaceItemIdentifier("bookmark-cell")
        let cell: NSTableCellView
        if let reused = tableView.makeView(withIdentifier: identifier, owner: self) as? NSTableCellView {
            cell = reused
        } else {
            cell = NSTableCellView()
            cell.identifier = identifier
            let text = NSTextField(labelWithString: "")
            text.translatesAutoresizingMaskIntoConstraints = false
            text.lineBreakMode = .byTruncatingMiddle
            cell.textField = text
            cell.addSubview(text)
            NSLayoutConstraint.activate([
                text.leadingAnchor.constraint(equalTo: cell.leadingAnchor, constant: 6),
                text.trailingAnchor.constraint(equalTo: cell.trailingAnchor, constant: -6),
                text.centerYAnchor.constraint(equalTo: cell.centerYAnchor)
            ])
        }
        let item = items[row]
        let scope = item.kind == "remote" ? "Remote" : "Local"
        let account = item.kind == "remote" && !item.host.isEmpty ? "  ·  \(item.username)@\(item.host)" : ""
        cell.textField?.stringValue = "[\(scope)]  \(item.name)  ·  \(item.path)\(account)"
        return cell
    }

    func tableViewSelectionDidChange(_ notification: Notification) { updateButtons() }

    private func readSnapshot() -> [BookmarkView] {
        let count = max(0, Int(GhostFTPBookmarkCount()))
        return (0..<count).map { index in
            let i = CInt(index)
            return BookmarkView(
                id: applicationBridgeString(GhostFTPBookmarkID(i)),
                name: applicationBridgeString(GhostFTPBookmarkName(i)),
                kind: applicationBridgeString(GhostFTPBookmarkKind(i)),
                path: applicationBridgeString(GhostFTPBookmarkPath(i)),
                host: applicationBridgeString(GhostFTPBookmarkHost(i)),
                username: applicationBridgeString(GhostFTPBookmarkUsername(i))
            )
        }
    }

    private func reloadBookmarks(selectID: String = "") {
        guard GhostFTPRefreshBookmarks() == 1 else {
            statusLabel.stringValue = applicationBridgeString(GhostFTPLastError())
            return
        }
        items = readSnapshot()
        table.reloadData()
        if !selectID.isEmpty, let index = items.firstIndex(where: { $0.id == selectID }) {
            table.selectRowIndexes(IndexSet(integer: index), byExtendingSelection: false)
            table.scrollRowToVisible(index)
        } else if table.selectedRow >= items.count {
            table.deselectAll(nil)
        }
        statusLabel.stringValue = items.isEmpty ? "No bookmarks yet." : "Bookmarks: \(items.count)"
        updateButtons()
    }

    private func updateButtons() {
        let selected = table.selectedRow >= 0 && table.selectedRow < items.count
        openButton.isEnabled = selected && !busy
        deleteButton.isEnabled = selected && !busy
        addLocalButton.isEnabled = !busy
        addRemoteButton.isEnabled = !busy && GhostFTPIsConnected() == 1
    }

    private func promptBookmarkName(title: String, suggested: String) -> String? {
        let alert = NSAlert()
        alert.messageText = title
        alert.informativeText = "Name:"
        alert.addButton(withTitle: "Save")
        alert.addButton(withTitle: "Cancel")
        let field = NSTextField(string: suggested)
        field.frame = NSRect(x: 0, y: 0, width: 340, height: 24)
        alert.accessoryView = field
        guard alert.runModal() == .alertFirstButtonReturn else { return nil }
        let value = field.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        return value.isEmpty ? nil : value
    }

    @objc private func addLocal() {
        guard let name = promptBookmarkName(title: "Add Local Bookmark", suggested: "Local folder") else { return }
        let value = ApplicationCStringBox(name)
        if GhostFTPSaveLocalBookmark(value.pointer) != 1 {
            statusLabel.stringValue = applicationBridgeString(GhostFTPLastError())
            return
        }
        let id = applicationBridgeString(GhostFTPLastSavedBookmarkID())
        reloadBookmarks(selectID: id)
        statusLabel.stringValue = "Local bookmark saved."
    }

    @objc private func addRemote() {
        guard GhostFTPIsConnected() == 1 else {
            statusLabel.stringValue = "Connect before saving a remote bookmark."
            updateButtons()
            return
        }
        guard let name = promptBookmarkName(title: "Add Remote Bookmark", suggested: "Remote folder") else { return }
        let value = ApplicationCStringBox(name)
        if GhostFTPSaveRemoteBookmark(value.pointer) != 1 {
            statusLabel.stringValue = applicationBridgeString(GhostFTPLastError())
            return
        }
        let id = applicationBridgeString(GhostFTPLastSavedBookmarkID())
        reloadBookmarks(selectID: id)
        statusLabel.stringValue = "Remote bookmark saved for the active server account."
    }

    @objc private func deleteSelected() {
        let row = table.selectedRow
        guard row >= 0, row < items.count else { return }
        let item = items[row]
        let alert = NSAlert()
        alert.messageText = "Delete Bookmark"
        alert.informativeText = "Delete \"\(item.name)\"?\n\(item.path)"
        alert.alertStyle = .warning
        alert.addButton(withTitle: "Delete")
        alert.addButton(withTitle: "Cancel")
        guard alert.runModal() == .alertFirstButtonReturn else { return }
        let id = ApplicationCStringBox(item.id)
        if GhostFTPRemoveBookmark(id.pointer) != 1 {
            statusLabel.stringValue = applicationBridgeString(GhostFTPLastError())
            return
        }
        reloadBookmarks()
        statusLabel.stringValue = "Bookmark removed."
    }

    @objc private func openSelected() {
        let row = table.selectedRow
        guard !busy, row >= 0, row < items.count else { return }
        let item = items[row]
        busy = true
        statusLabel.stringValue = "Opening bookmark…"
        updateButtons()
        let idValue = item.id
        engineQueue.async { [weak self] in
            let id = ApplicationCStringBox(idValue)
            let ok = GhostFTPNavigateBookmark(id.pointer) == 1
            let kind = ok ? applicationBridgeString(GhostFTPLastBookmarkKind()) : ""
            let path = ok ? applicationBridgeString(GhostFTPLastBookmarkPath()) : ""
            let message = ok ? "" : applicationBridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self else { return }
                self.busy = false
                self.updateButtons()
                if ok {
                    self.statusLabel.stringValue = "Bookmark opened."
                    self.onNavigation?(kind, path)
                    self.close()
                } else {
                    self.statusLabel.stringValue = message.isEmpty ? "Bookmark could not be opened safely." : message
                }
            }
        }
    }

    @objc private func closeTapped() { close() }
}

private struct LanguageView {
    let code: String
    let nativeName: String
    let englishName: String
}

final class SettingsWindowController: NSWindowController {
    var onAppearanceChanged: ((String) -> Void)?

    private var languages: [LanguageView] = []
    private let languagePopup = NSPopUpButton(frame: .zero, pullsDown: false)
    private let appearancePopup = NSPopUpButton(frame: .zero, pullsDown: false)
    private let parallelismField = NSTextField(string: "2")
    private let uploadLimitField = NSTextField(string: "0")
    private let downloadLimitField = NSTextField(string: "0")
    private let connectionTimeoutField = NSTextField(string: "15")
    private let autoRetryField = NSTextField(string: "0")
    private let retryDelayField = NSTextField(string: "3")
    private let conflictPopup = NSPopUpButton(frame: .zero, pullsDown: false)
    private let confirmDeleteButton = NSButton(checkboxWithTitle: "Confirm delete", target: nil, action: nil)
    private let statusLabel = NSTextField(labelWithString: "")

    init() {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 650, height: 560),
            styleMask: [.titled, .closable, .miniaturizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Settings"
        super.init(window: window)
        buildUI()
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    func showAndRefresh() {
        reloadSettings()
        showWindow(nil)
        window?.center()
        window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    private func buildUI() {
        guard let content = window?.contentView else { return }
        appearancePopup.addItems(withTitles: ["Classic Light", "Dark"])
        conflictPopup.addItems(withTitles: ["Skip existing", "Replace", "Replace + Backup"])
        for field in [parallelismField, uploadLimitField, downloadLimitField, connectionTimeoutField, autoRetryField, retryDelayField] {
            field.alignment = .right
            field.widthAnchor.constraint(equalToConstant: 150).isActive = true
        }

        let grid = NSGridView(views: [
            [applicationLabel("Language"), languagePopup],
            [applicationLabel("Appearance"), appearancePopup],
            [applicationLabel("Parallel transfers"), parallelismField],
            [applicationLabel("Upload limit (KiB/s, 0 = unlimited)"), uploadLimitField],
            [applicationLabel("Download limit (KiB/s, 0 = unlimited)"), downloadLimitField],
            [applicationLabel("Connection timeout (seconds)"), connectionTimeoutField],
            [applicationLabel("Automatic retries"), autoRetryField],
            [applicationLabel("Retry delay (seconds)"), retryDelayField],
            [applicationLabel("Conflict policy"), conflictPopup],
            [applicationLabel("Delete safety"), confirmDeleteButton]
        ])
        grid.rowSpacing = 10
        grid.columnSpacing = 14
        grid.column(at: 0).xPlacement = .trailing

        let saveButton = NSButton(title: "Save", target: self, action: #selector(saveTapped))
        saveButton.keyEquivalent = "\r"
        let closeButton = NSButton(title: "Close", target: self, action: #selector(closeTapped))
        let actions = NSStackView(views: [saveButton, closeButton])
        actions.orientation = .horizontal
        actions.spacing = 8
        statusLabel.textColor = .secondaryLabelColor
        statusLabel.maximumNumberOfLines = 2
        let note = applicationLabel("Language is stored in the shared Ghost FTP settings used by all desktop surfaces.", size: 11)
        note.textColor = .secondaryLabelColor

        let stack = NSStackView(views: [applicationLabel("Ghost FTP Settings", size: 20, weight: .semibold), grid, note, actions, statusLabel])
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 14
        stack.edgeInsets = NSEdgeInsets(top: 20, left: 20, bottom: 20, right: 20)
        stack.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            stack.topAnchor.constraint(equalTo: content.topAnchor),
            stack.bottomAnchor.constraint(equalTo: content.bottomAnchor),
            grid.widthAnchor.constraint(greaterThanOrEqualToConstant: 560)
        ])
    }

    private func loadLanguages() {
        let count = max(0, Int(GhostFTPLanguageCount()))
        languages = (0..<count).map { index in
            let i = CInt(index)
            return LanguageView(
                code: applicationBridgeString(GhostFTPLanguageCode(i)),
                nativeName: applicationBridgeString(GhostFTPLanguageNativeName(i)),
                englishName: applicationBridgeString(GhostFTPLanguageEnglishName(i))
            )
        }
        languagePopup.removeAllItems()
        languagePopup.addItems(withTitles: languages.map { language in
            language.nativeName == language.englishName ? language.nativeName : "\(language.nativeName) — \(language.englishName)"
        })
    }

    private func reloadSettings() {
        loadLanguages()
        guard GhostFTPRefreshSettings() == 1 else {
            statusLabel.stringValue = applicationBridgeString(GhostFTPLastError())
            return
        }
        let language = applicationBridgeString(GhostFTPSettingsLanguage())
        if let index = languages.firstIndex(where: { $0.code == language }) {
            languagePopup.selectItem(at: index)
        }
        appearancePopup.selectItem(at: applicationBridgeString(GhostFTPSettingsAppearance()) == "dark" ? 1 : 0)
        parallelismField.stringValue = String(GhostFTPSettingsParallelism())
        uploadLimitField.stringValue = String(GhostFTPSettingsUploadLimit())
        downloadLimitField.stringValue = String(GhostFTPSettingsDownloadLimit())
        connectionTimeoutField.stringValue = String(GhostFTPSettingsConnectionTimeout())
        autoRetryField.stringValue = String(GhostFTPSettingsAutoRetryCount())
        retryDelayField.stringValue = String(GhostFTPSettingsRetryDelay())
        switch applicationBridgeString(GhostFTPSettingsConflictPolicy()) {
        case "skip": conflictPopup.selectItem(at: 0)
        case "replace": conflictPopup.selectItem(at: 1)
        default: conflictPopup.selectItem(at: 2)
        }
        confirmDeleteButton.state = GhostFTPSettingsConfirmDelete() == 1 ? .on : .off
        statusLabel.stringValue = ""
    }

    private func validatedInteger(_ field: NSTextField, name: String, minimum: Int, maximum: Int) -> Int? {
        guard let value = Int(field.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)), value >= minimum, value <= maximum else {
            statusLabel.stringValue = "\(name) must be between \(minimum) and \(maximum)."
            return nil
        }
        return value
    }

    @objc private func saveTapped() {
        guard !languages.isEmpty, languagePopup.indexOfSelectedItem >= 0, languagePopup.indexOfSelectedItem < languages.count else {
            statusLabel.stringValue = "Select a supported language."
            return
        }
        guard let parallelism = validatedInteger(parallelismField, name: "Parallel transfers", minimum: 1, maximum: 8),
              let upload = validatedInteger(uploadLimitField, name: "Upload limit", minimum: 0, maximum: 1_048_576),
              let download = validatedInteger(downloadLimitField, name: "Download limit", minimum: 0, maximum: 1_048_576),
              let timeout = validatedInteger(connectionTimeoutField, name: "Connection timeout", minimum: 5, maximum: 60),
              let retries = validatedInteger(autoRetryField, name: "Automatic retries", minimum: 0, maximum: 3),
              let retryDelay = validatedInteger(retryDelayField, name: "Retry delay", minimum: 1, maximum: 30) else { return }

        let language = languages[languagePopup.indexOfSelectedItem].code
        let appearance = appearancePopup.indexOfSelectedItem == 1 ? "dark" : "light"
        let conflict: String
        switch conflictPopup.indexOfSelectedItem {
        case 0: conflict = "skip"
        case 1: conflict = "replace"
        default: conflict = "replace_backup"
        }
        let languageValue = ApplicationCStringBox(language)
        let appearanceValue = ApplicationCStringBox(appearance)
        let conflictValue = ApplicationCStringBox(conflict)
        let ok = GhostFTPSaveSettings(
            languageValue.pointer,
            appearanceValue.pointer,
            CInt(parallelism),
            CInt(upload),
            CInt(download),
            CInt(timeout),
            CInt(retries),
            CInt(retryDelay),
            conflictValue.pointer,
            confirmDeleteButton.state == .on ? 1 : 0
        ) == 1
        if !ok {
            statusLabel.stringValue = applicationBridgeString(GhostFTPLastError())
            return
        }
        statusLabel.stringValue = "Settings saved."
        onAppearanceChanged?(appearance)
    }

    @objc private func closeTapped() { close() }
}

final class AboutWindowController: NSWindowController {
    init() {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 520, height: 390),
            styleMask: [.titled, .closable],
            backing: .buffered,
            defer: false
        )
        window.title = "About Ghost FTP"
        super.init(window: window)
        buildUI()
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    func showAbout() {
        showWindow(nil)
        window?.center()
        window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    private func buildUI() {
        guard let content = window?.contentView else { return }
        let version = applicationBridgeString(GhostFTPAboutVersion())
        let publisher = applicationBridgeString(GhostFTPAboutPublisher())
        let website = applicationBridgeString(GhostFTPAboutWebsite())
        let author = applicationBridgeString(GhostFTPAboutAuthorWebsite())
        let support = applicationBridgeString(GhostFTPAboutSupport())
        let rows = [
            applicationLabel("Ghost FTP", size: 25, weight: .bold),
            applicationLabel("Version \(version)", size: 13, weight: .medium),
            applicationLabel("FTP · explicit FTPS · SFTP", size: 12, weight: .medium),
            applicationLabel("Publisher: \(publisher)"),
            applicationLabel("Product: \(website)"),
            applicationLabel("Author: \(author)"),
            applicationLabel("Support: \(support)"),
            applicationLabel("Your servers. Your files. No cloud middleman."),
            applicationLabel("No telemetry or tracking.")
        ]
        rows.last?.textColor = .secondaryLabelColor
        let closeButton = NSButton(title: "Close", target: self, action: #selector(closeTapped))
        let stack = NSStackView(views: rows + [closeButton])
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 12
        stack.edgeInsets = NSEdgeInsets(top: 24, left: 24, bottom: 24, right: 24)
        stack.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            stack.topAnchor.constraint(equalTo: content.topAnchor),
            stack.bottomAnchor.constraint(equalTo: content.bottomAnchor)
        ])
    }

    @objc private func closeTapped() { close() }
}

final class DiagnosticsWindowController: NSWindowController {
    private let connectionLabel = applicationLabel("")
    private let pathLabel = applicationLabel("")

    init() {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 600, height: 390),
            styleMask: [.titled, .closable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Diagnostics"
        window.minSize = NSSize(width: 500, height: 330)
        super.init(window: window)
        buildUI()
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    func showAndRefresh() {
        refreshStatus()
        showWindow(nil)
        window?.center()
        window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    private func buildUI() {
        guard let content = window?.contentView else { return }
        let refreshButton = NSButton(title: "Refresh", target: self, action: #selector(refreshTapped))
        let closeButton = NSButton(title: "Close", target: self, action: #selector(closeTapped))
        let actions = NSStackView(views: [refreshButton, closeButton])
        actions.orientation = .horizontal
        actions.spacing = 8
        pathLabel.lineBreakMode = .byTruncatingMiddle
        let privacy = applicationLabel("No telemetry or tracking. Saved profiles stay on this computer.")
        privacy.textColor = .secondaryLabelColor
        let note = applicationLabel("This view intentionally shows compact runtime status only. It does not expose connection secrets or raw diagnostic output.", size: 11)
        note.textColor = .secondaryLabelColor
        let stack = NSStackView(views: [
            applicationLabel("Ghost FTP Diagnostics", size: 20, weight: .semibold),
            connectionLabel,
            pathLabel,
            privacy,
            note,
            actions
        ])
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 14
        stack.edgeInsets = NSEdgeInsets(top: 22, left: 22, bottom: 22, right: 22)
        stack.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            stack.topAnchor.constraint(equalTo: content.topAnchor),
            stack.bottomAnchor.constraint(equalTo: content.bottomAnchor)
        ])
    }

    private func refreshStatus() {
        let connected = GhostFTPDiagnosticsConnected() == 1
        let proto = applicationBridgeString(GhostFTPDiagnosticsProtocol())
        let remotePath = applicationBridgeString(GhostFTPDiagnosticsRemotePath())
        connectionLabel.stringValue = connected ? "\(proto.isEmpty ? "Connection" : proto) · Connected" : "Not connected"
        pathLabel.stringValue = connected && !remotePath.isEmpty ? "Remote folder: \(remotePath)" : "Remote folder: —"
    }

    @objc private func refreshTapped() { refreshStatus() }
    @objc private func closeTapped() { close() }
}
