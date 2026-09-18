#!/usr/bin/env python3
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"macOS integration anchor {label!r} matched {count} times; refusing to build")
    return text.replace(old, new, 1)


def integrate_main(text: str) -> str:
    text = replace_once(
        text,
        "    private var transferQueueController: TransferQueueWindowController?\n",
        "    private var transferQueueController: TransferQueueWindowController?\n"
        "    private var siteManagerController: SiteManagerWindowController?\n"
        "    private var bookmarksController: BookmarksWindowController?\n"
        "    private var settingsController: SettingsWindowController?\n"
        "    private var aboutController: AboutWindowController?\n"
        "    private var diagnosticsController: DiagnosticsWindowController?\n",
        "main-controller-properties",
    )
    text = replace_once(
        text,
        '    private let transferQueueButton = NSButton(title: "Transfers", target: nil, action: nil)\n',
        '    private let filesNavButton = NSButton(title: "Files", target: nil, action: nil)\n'
        '    private let transferQueueButton = NSButton(title: "Transfer Queue", target: nil, action: nil)\n'
        '    private let siteManagerButton = NSButton(title: "Connections", target: nil, action: nil)\n'
        '    private let bookmarksButton = NSButton(title: "Bookmarks", target: nil, action: nil)\n'
        '    private let settingsButton = NSButton(title: "Settings", target: nil, action: nil)\n'
        '    private let aboutButton = NSButton(title: "About", target: nil, action: nil)\n'
        '    private let diagnosticsButton = NSButton(title: "Connection info", target: nil, action: nil)\n',
        "main-button-properties",
    )
    text = replace_once(
        text,
        '''    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        buildWindow()
        engineReady = GhostFTPCreateEngine() == 1
        if !engineReady {
            statusLabel.stringValue = bridgeString(GhostFTPLastError())
            connectButton.isEnabled = false
        } else {
            refreshLocal(localCurrent)
        }
''',
        '''    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        engineReady = GhostFTPCreateEngine() == 1
        if engineReady {
            applyInitialAppearance()
        }
        buildWindow()
        if !engineReady {
            statusLabel.stringValue = bridgeString(GhostFTPLastError())
            connectButton.isEnabled = false
        } else {
            refreshLocal(localCurrent)
        }
''',
        "launch-appearance-before-window",
    )
    text = replace_once(
        text,
        "        transferQueueController?.closeSilently()\n"
        "        transferQueueController = nil\n"
        "        GhostFTPClearTransferQueueSnapshot()\n",
        "        transferQueueController?.closeSilently()\n"
        "        transferQueueController = nil\n"
        "        siteManagerController?.close()\n"
        "        siteManagerController = nil\n"
        "        bookmarksController?.close()\n"
        "        bookmarksController = nil\n"
        "        settingsController?.close()\n"
        "        settingsController = nil\n"
        "        aboutController?.close()\n"
        "        aboutController = nil\n"
        "        diagnosticsController?.close()\n"
        "        diagnosticsController = nil\n"
        "        GhostFTPClearTransferQueueSnapshot()\n",
        "main-termination-cleanup",
    )
    text = replace_once(
        text,
        "        transferQueueButton.target = self\n"
        "        transferQueueButton.action = #selector(transferQueueTapped)\n",
        "        filesNavButton.target = self\n"
        "        filesNavButton.action = #selector(filesNavigationTapped)\n"
        "        transferQueueButton.target = self\n"
        "        transferQueueButton.action = #selector(transferQueueTapped)\n"
        "        siteManagerButton.target = self\n"
        "        siteManagerButton.action = #selector(siteManagerTapped)\n"
        "        bookmarksButton.target = self\n"
        "        bookmarksButton.action = #selector(bookmarksTapped)\n"
        "        settingsButton.target = self\n"
        "        settingsButton.action = #selector(settingsTapped)\n"
        "        aboutButton.target = self\n"
        "        aboutButton.action = #selector(aboutTapped)\n"
        "        diagnosticsButton.target = self\n"
        "        diagnosticsButton.action = #selector(diagnosticsTapped)\n",
        "main-button-actions",
    )
    text = replace_once(
        text,
        "        let buttonRow = NSStackView(views: [connectButton, disconnectButton, directoryCompareButton, transferQueueButton, statusLabel])\n"
        "        buttonRow.orientation = .horizontal\n"
        "        buttonRow.alignment = .centerY\n"
        "        buttonRow.spacing = 10\n"
        "        statusLabel.textColor = Palette.muted\n"
        "        statusLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)\n\n"
        "        let connectionStack = NSStackView(views: [heading, form, rememberFingerprint, buttonRow])\n",
        "        let buttonRow = NSStackView(views: [connectButton, disconnectButton, directoryCompareButton, statusLabel])\n"
        "        buttonRow.orientation = .horizontal\n"
        "        buttonRow.alignment = .centerY\n"
        "        buttonRow.spacing = 10\n"
        "        statusLabel.textColor = Palette.muted\n"
        "        statusLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)\n\n"
        "        let connectionStack = NSStackView(views: [heading, form, rememberFingerprint, buttonRow])\n",
        "main-button-rows",
    )
    text = replace_once(
        text,
        "        let workspace = makeWorkspace()\n"
        "        let root = NSStackView(views: [connectionStack, workspace])\n"
        "        root.orientation = .vertical\n"
        "        root.spacing = 14\n"
        "        root.edgeInsets = NSEdgeInsets(top: 18, left: 18, bottom: 18, right: 18)\n"
        "        root.translatesAutoresizingMaskIntoConstraints = false\n"
        "        window.contentView?.addSubview(root)\n"
        "        guard let content = window.contentView else { return }\n"
        "        NSLayoutConstraint.activate([\n"
        "            root.leadingAnchor.constraint(equalTo: content.leadingAnchor),\n"
        "            root.trailingAnchor.constraint(equalTo: content.trailingAnchor),\n"
        "            root.topAnchor.constraint(equalTo: content.topAnchor),\n"
        "            root.bottomAnchor.constraint(equalTo: content.bottomAnchor),\n"
        "            workspace.heightAnchor.constraint(greaterThanOrEqualToConstant: 380),\n"
        "            connectionStack.widthAnchor.constraint(equalTo: root.widthAnchor, constant: -36)\n"
        "        ])\n",
        "        let workspace = makeWorkspace()\n"
        "        let embeddedQueue = makeEmbeddedTransferQueue()\n"
        "        let navigationRail = makeMasterNavigationRail()\n"
        "        let mainColumn = NSStackView(views: [connectionStack, workspace, embeddedQueue])\n"
        "        mainColumn.orientation = .vertical\n"
        "        mainColumn.spacing = 12\n"
        "        mainColumn.alignment = .leading\n"
        "        let root = NSStackView(views: [navigationRail, mainColumn])\n"
        "        root.orientation = .horizontal\n"
        "        root.alignment = .top\n"
        "        root.spacing = 14\n"
        "        root.edgeInsets = NSEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)\n"
        "        root.translatesAutoresizingMaskIntoConstraints = false\n"
        "        window.contentView?.addSubview(root)\n"
        "        guard let content = window.contentView else { return }\n"
        "        NSLayoutConstraint.activate([\n"
        "            root.leadingAnchor.constraint(equalTo: content.leadingAnchor),\n"
        "            root.trailingAnchor.constraint(equalTo: content.trailingAnchor),\n"
        "            root.topAnchor.constraint(equalTo: content.topAnchor),\n"
        "            root.bottomAnchor.constraint(equalTo: content.bottomAnchor),\n"
        "            navigationRail.widthAnchor.constraint(equalToConstant: 188),\n"
        "            navigationRail.heightAnchor.constraint(equalTo: root.heightAnchor, constant: -28),\n"
        "            mainColumn.heightAnchor.constraint(equalTo: root.heightAnchor, constant: -28),\n"
        "            connectionStack.widthAnchor.constraint(equalTo: mainColumn.widthAnchor),\n"
        "            workspace.widthAnchor.constraint(equalTo: mainColumn.widthAnchor),\n"
        "            embeddedQueue.widthAnchor.constraint(equalTo: mainColumn.widthAnchor),\n"
        "            workspace.heightAnchor.constraint(greaterThanOrEqualToConstant: 300),\n"
        "            embeddedQueue.heightAnchor.constraint(greaterThanOrEqualToConstant: 180)\n"
        "        ])\n",
        "main-master-rail-layout",
    )
    text = replace_once(
        text,
        "\n\n    private func startTransferQueuePolling() {\n",
        '''

    private func styleMasterRailButton(_ button: NSButton, active: Bool) {
        button.isBordered = false
        button.font = .systemFont(ofSize: 13, weight: active ? .semibold : .medium)
        button.alignment = .left
        button.contentTintColor = active ? Palette.accentStrong : Palette.text
        button.wantsLayer = true
        button.layer?.cornerRadius = 8
        button.layer?.borderWidth = active ? 1 : 0
        button.layer?.borderColor = active ? Palette.accent.cgColor : NSColor.clear.cgColor
        button.layer?.backgroundColor = active ? Palette.selection.cgColor : NSColor.clear.cgColor
    }

    private func makeMasterNavigationRail() -> NSView {
        let brand = NSTextField(labelWithString: "Ghost FTP")
        brand.font = .systemFont(ofSize: 20, weight: .bold)
        brand.textColor = Palette.text

        let platform = NSTextField(labelWithString: "macOS")
        platform.font = .systemFont(ofSize: 11, weight: .medium)
        platform.textColor = Palette.muted

        for button in [filesNavButton, siteManagerButton, transferQueueButton, settingsButton, bookmarksButton, diagnosticsButton, aboutButton] {
            styleMasterRailButton(button, active: button === filesNavButton)
            button.heightAnchor.constraint(equalToConstant: 38).isActive = true
        }

        let primary = NSStackView(views: [filesNavButton, siteManagerButton, transferQueueButton, settingsButton])
        primary.orientation = .vertical
        primary.alignment = .leading
        primary.spacing = 6

        let utility = NSStackView(views: [bookmarksButton, diagnosticsButton, aboutButton])
        utility.orientation = .vertical
        utility.alignment = .leading
        utility.spacing = 6

        for stack in [primary, utility] {
            for view in stack.views {
                view.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true
            }
        }

        let spacer = NSView()
        spacer.setContentHuggingPriority(.defaultLow, for: .vertical)
        let railStack = NSStackView(views: [brand, platform, primary, spacer, utility])
        railStack.orientation = .vertical
        railStack.alignment = .leading
        railStack.spacing = 10
        railStack.edgeInsets = NSEdgeInsets(top: 18, left: 14, bottom: 14, right: 14)
        railStack.translatesAutoresizingMaskIntoConstraints = false

        let rail = NSView()
        rail.wantsLayer = true
        rail.layer?.backgroundColor = Palette.panel.cgColor
        rail.layer?.cornerRadius = 12
        rail.layer?.borderWidth = 1
        rail.layer?.borderColor = Palette.border.cgColor
        rail.addSubview(railStack)
        NSLayoutConstraint.activate([
            railStack.leadingAnchor.constraint(equalTo: rail.leadingAnchor),
            railStack.trailingAnchor.constraint(equalTo: rail.trailingAnchor),
            railStack.topAnchor.constraint(equalTo: rail.topAnchor),
            railStack.bottomAnchor.constraint(equalTo: rail.bottomAnchor)
        ])
        return rail
    }

    @objc private func filesNavigationTapped() {
        window?.makeKeyAndOrderFront(nil)
        window?.makeFirstResponder(localTable)
    }

    @objc private func siteManagerTapped() {
        guard engineReady, !connectionBusy else { return }
        if let controller = siteManagerController {
            controller.showAndRefresh()
            return
        }
        let controller = SiteManagerWindowController()
        controller.onConnected = { [weak self] protocolName, remoteStart, localStart in
            guard let self else { return }
            let local = localStart.trimmingCharacters(in: .whitespacesAndNewlines)
            let remote = remoteStart.trimmingCharacters(in: .whitespacesAndNewlines)
            let remoteTarget = remote.isEmpty ? (protocolName.lowercased() == "sftp" ? "." : "/") : remote
            self.remoteCurrent = remoteTarget
            self.refreshConnectionState()
            self.refreshLocal(local.isEmpty ? self.localCurrent : local)
            self.refreshRemote(remoteTarget)
        }
        siteManagerController = controller
        controller.showAndRefresh()
    }

    @objc private func bookmarksTapped() {
        guard engineReady,
              !connectionBusy,
              !localMutationBusy,
              !remoteMutationBusy,
              !remoteEditBusy,
              !recursiveSearchBusy,
              !directoryCompareBusy else { return }
        if let controller = bookmarksController {
            controller.showAndRefresh()
            return
        }
        let controller = BookmarksWindowController()
        controller.onNavigation = { [weak self] kind, path in
            self?.bookmarkNavigationApplied(kind: kind, path: path)
        }
        bookmarksController = controller
        controller.showAndRefresh()
    }

    private func bookmarkNavigationApplied(kind: String, path: String) {
        guard engineReady else { return }
        let remote = kind == "remote"
        if remote {
            remoteNavigationGeneration += 1
            remoteFilterQuery = ""
        } else {
            localNavigationGeneration += 1
            localFilterQuery = ""
        }
        let generation = remote ? remoteNavigationGeneration : localNavigationGeneration
        statusLabel.stringValue = "Opening bookmark…"
        engineQueue.async { [weak self] in
            let query = CStringBox("")
            let ok = remote ? GhostFTPRemoteFilter(query.pointer) == 1 : GhostFTPLocalFilter(query.pointer) == 1
            let resolved = ok ? bridgeString(remote ? GhostFTPRemotePath() : GhostFTPLocalPath()) : ""
            let sourceCount = ok ? max(0, Int(remote ? GhostFTPRemoteItemCount() : GhostFTPLocalItemCount())) : 0
            let items = ok ? (remote ? self?.readRemoteFilteredSnapshot() ?? [] : self?.readLocalFilteredSnapshot() ?? []) : []
            let message = ok ? "" : bridgeString(GhostFTPLastError())
            DispatchQueue.main.async {
                guard let self else { return }
                let currentGeneration = remote ? self.remoteNavigationGeneration : self.localNavigationGeneration
                guard generation == currentGeneration else { return }
                if !ok || resolved.isEmpty {
                    self.statusLabel.stringValue = "Bookmark view could not be applied."
                    if !message.isEmpty { self.showError(message) }
                    self.updateWorkspaceControls()
                    return
                }
                if remote {
                    self.remoteCurrent = resolved
                    self.remoteSourceCount = sourceCount
                    self.remoteItems = items
                    self.remotePathLabel.stringValue = resolved
                    self.remoteTable.reloadData()
                } else {
                    self.localCurrent = resolved
                    self.localSourceCount = sourceCount
                    self.localItems = items
                    self.localPathLabel.stringValue = resolved
                    self.localTable.reloadData()
                }
                self.statusLabel.stringValue = remote ? "Remote bookmark opened: \\(path)" : "Local bookmark opened: \\(path)"
                self.updateWorkspaceControls()
            }
        }
    }

    @objc private func settingsTapped() {
        guard engineReady else { return }
        if let controller = settingsController {
            controller.showAndRefresh()
            return
        }
        let controller = SettingsWindowController()
        controller.onAppearanceChanged = { [weak self] appearance in
            self?.settingsAppearanceChanged(appearance)
        }
        settingsController = controller
        controller.showAndRefresh()
    }

    private func applyInitialAppearance() {
        guard GhostFTPRefreshSettings() == 1 else { return }
        let appearance = bridgeString(GhostFTPSettingsAppearance())
        let name: NSAppearance.Name = appearance == "dark" ? .darkAqua : .aqua
        NSApp.appearance = NSAppearance(named: name)
    }

    private func settingsAppearanceChanged(_ appearance: String) {
        let name: NSAppearance.Name = appearance == "dark" ? .darkAqua : .aqua
        NSApp.appearance = NSAppearance(named: name)
        window?.appearance = NSAppearance(named: name)
        reapplyPalette()
        statusLabel.stringValue = "Settings saved."
    }

    private func reapplyPalette() {
        guard let content = window?.contentView else { return }
        content.layer?.backgroundColor = Palette.workspace.cgColor
        func visit(_ view: NSView) {
            if let layer = view.layer, layer.cornerRadius >= 9 {
                layer.backgroundColor = Palette.panel.cgColor
                layer.borderColor = Palette.border.cgColor
            }
            if view.superview is NSSplitView {
                view.layer?.backgroundColor = Palette.list.cgColor
            }
            if let table = view as? NSTableView {
                table.backgroundColor = Palette.list
            }
            if let scroll = view as? NSScrollView {
                scroll.backgroundColor = Palette.list
            }
            for child in view.subviews { visit(child) }
        }
        visit(content)
        styleMasterRailButton(filesNavButton, active: true)
        for button in [siteManagerButton, transferQueueButton, settingsButton, bookmarksButton, diagnosticsButton, aboutButton] {
            styleMasterRailButton(button, active: false)
        }
        content.needsDisplay = true
    }

    @objc private func aboutTapped() {
        if let controller = aboutController {
            controller.showAbout()
            return
        }
        let controller = AboutWindowController()
        aboutController = controller
        controller.showAbout()
    }

    @objc private func diagnosticsTapped() {
        guard engineReady else { return }
        if let controller = diagnosticsController {
            controller.showAndRefresh()
            return
        }
        let controller = DiagnosticsWindowController()
        diagnosticsController = controller
        controller.showAndRefresh()
    }

    private func startTransferQueuePolling() {
''',
        "main-application-actions",
    )
    text = replace_once(
        text,
        "        transferQueueButton.isEnabled = engineReady && !connectionBusy\n",
        "        siteManagerButton.isEnabled = engineReady && !connectionBusy\n"
        "        bookmarksButton.isEnabled = engineReady && !connectionBusy && !localMutationBusy && !remoteMutationBusy && !remoteEditBusy && !recursiveSearchBusy && !directoryCompareBusy\n"
        "        settingsButton.isEnabled = engineReady\n"
        "        aboutButton.isEnabled = true\n"
        "        diagnosticsButton.isEnabled = engineReady\n"
        "        transferQueueButton.isEnabled = engineReady && !connectionBusy\n",
        "main-control-state",
    )
    text = replace_once(
        text,
        "        transferQueueEntries = entries\n"
        "        transferQueuePaused = paused\n"
        "        updateEmbeddedTransferQueue()\n"
        "        transferQueueController?.apply(\n",
        "        transferQueueEntries = entries\n"
        "        let actionableCount = entries.filter { entry in\n"
        "            entry.status == \"queued\" || entry.status == \"running\" || entry.status == \"failed\" || entry.status == \"cancelled\"\n"
        "        }.count\n"
        "        transferQueueButton.title = actionableCount > 0 ? \"Transfer Queue (\\(min(actionableCount, 99)))\" : \"Transfer Queue\"\n"
        "        transferQueuePaused = paused\n"
        "        updateEmbeddedTransferQueue()\n"
        "        transferQueueController?.apply(\n",
        "main-transfer-queue-badge",
    )
    text = replace_once(
        text,
        "    private func confirmDelete(_ items: [FileItem]) -> Bool {\n"
        "        let alert = NSAlert()\n",
        "    private func confirmDelete(_ items: [FileItem]) -> Bool {\n"
        "        if GhostFTPRefreshSettings() == 1 && GhostFTPSettingsConfirmDelete() == 0 {\n"
        "            return true\n"
        "        }\n"
        "        let alert = NSAlert()\n",
        "main-confirm-delete-setting",
    )
    return text


def integrate_site_manager(text: str) -> str:
    text = replace_once(
        text,
        "    var onConnected: (() -> Void)?\n",
        "    var onConnected: ((String, String, String) -> Void)?\n",
        "site-manager-connected-callback-type",
    )
    text = replace_once(
        text,
        "        connectButton.isEnabled = true\n"
        "        protocolChanged()\n",
        "        connectButton.isEnabled = GhostFTPIsConnected() == 0\n"
        "        protocolChanged()\n",
        "site-manager-connected-session-gate",
    )
    text = replace_once(
        text,
        "                self.connectButton.isEnabled = self.selectedProfile != nil\n"
        "                if result == 1 {\n"
        "                    self.statusLabel.stringValue = \"Connected.\"\n"
        "                    self.onConnected?()\n"
        "                    self.close()\n"
        "                } else {\n",
        "                self.connectButton.isEnabled = self.selectedProfile != nil && GhostFTPIsConnected() == 0\n"
        "                if result == 1 {\n"
        "                    self.statusLabel.stringValue = \"Connected.\"\n"
        "                    if let connected = self.profiles.first(where: { $0.id == profileID }) {\n"
        "                        self.onConnected?(connected.protocolName, connected.remotePath, connected.localPath)\n"
        "                    } else {\n"
        "                        self.onConnected?(\"\", \"\", \"\")\n"
        "                    }\n"
        "                    self.close()\n"
        "                } else {\n",
        "site-manager-connected-callback",
    )
    text = replace_once(
        text,
        "            connectButton.isEnabled = selectedProfile != nil\n"
        "            statusLabel.stringValue = \"Connection cancelled.\"\n",
        "            connectButton.isEnabled = selectedProfile != nil && GhostFTPIsConnected() == 0\n"
        "            statusLabel.stringValue = \"Connection cancelled.\"\n",
        "site-manager-cancel-gate",
    )
    return text


def main() -> None:
    if len(sys.argv) != 5:
        raise SystemExit("usage: prepare_site_manager_sources.py MAIN.swift SiteManager.swift OUT_MAIN.swift OUT_SITE.swift")
    main_source, site_source, out_main, out_site = map(Path, sys.argv[1:])
    main_text = integrate_main(main_source.read_text(encoding="utf-8"))
    site_text = integrate_site_manager(site_source.read_text(encoding="utf-8"))
    Path(out_main).write_text(main_text, encoding="utf-8")
    Path(out_site).write_text(site_text, encoding="utf-8")


if __name__ == "__main__":
    main()
