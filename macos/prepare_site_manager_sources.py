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
        '''private enum Palette {
    static let workspace = NSColor(rgb: 0xEEF1F5)
    static let panel = NSColor(rgb: 0xF6F8FB)
    static let list = NSColor(rgb: 0xFAFBFD)
    static let text = NSColor(rgb: 0x111827)
    static let muted = NSColor(rgb: 0x667085)
    static let accent = NSColor(rgb: 0x2563EB)
    static let border = NSColor(rgb: 0xD7DDE6)
}
''',
        '''private enum Palette {
    static var isDark: Bool {
        NSApp.effectiveAppearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
    }
    static var workspace: NSColor { NSColor(rgb: isDark ? 0x0B0F17 : 0xEEF1F5) }
    static var panel: NSColor { NSColor(rgb: isDark ? 0x121824 : 0xF6F8FB) }
    static var list: NSColor { NSColor(rgb: isDark ? 0x161D2A : 0xFAFBFD) }
    static let text = NSColor.labelColor
    static let muted = NSColor.secondaryLabelColor
    static let accent = NSColor.controlAccentColor
    static var border: NSColor { NSColor(rgb: isDark ? 0x273244 : 0xD7DDE6) }
}
''',
        "dynamic-palette",
    )
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
        '    private let transferQueueButton = NSButton(title: "Transfers", target: nil, action: nil)\n'
        '    private let siteManagerButton = NSButton(title: "Site Manager", target: nil, action: nil)\n'
        '    private let bookmarksButton = NSButton(title: "Bookmarks", target: nil, action: nil)\n'
        '    private let settingsButton = NSButton(title: "Settings", target: nil, action: nil)\n'
        '    private let aboutButton = NSButton(title: "About", target: nil, action: nil)\n'
        '    private let diagnosticsButton = NSButton(title: "Diagnostics", target: nil, action: nil)\n',
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
        "        let buttonRow = NSStackView(views: [connectButton, disconnectButton, siteManagerButton, bookmarksButton, settingsButton, statusLabel])\n"
        "        buttonRow.orientation = .horizontal\n"
        "        buttonRow.alignment = .centerY\n"
        "        buttonRow.spacing = 10\n"
        "        let toolsRow = NSStackView(views: [directoryCompareButton, transferQueueButton, diagnosticsButton, aboutButton])\n"
        "        toolsRow.orientation = .horizontal\n"
        "        toolsRow.alignment = .centerY\n"
        "        toolsRow.spacing = 10\n"
        "        statusLabel.textColor = Palette.muted\n"
        "        statusLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)\n\n"
        "        let connectionStack = NSStackView(views: [heading, form, rememberFingerprint, buttonRow, toolsRow])\n",
        "main-button-rows",
    )
    text = replace_once(
        text,
        "\n\n    private func startTransferQueuePolling() {\n",
        '''

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
                self.statusLabel.stringValue = remote ? "Remote bookmark opened: \(path)" : "Local bookmark opened: \(path)"
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
