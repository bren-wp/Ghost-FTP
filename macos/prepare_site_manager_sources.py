#!/usr/bin/env python3
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"Site Manager integration anchor {label!r} matched {count} times; refusing to build")
    return text.replace(old, new, 1)


def integrate_main(text: str) -> str:
    text = replace_once(
        text,
        "    private var transferQueueController: TransferQueueWindowController?\n",
        "    private var transferQueueController: TransferQueueWindowController?\n"
        "    private var siteManagerController: SiteManagerWindowController?\n",
        "main-controller-property",
    )
    text = replace_once(
        text,
        '    private let transferQueueButton = NSButton(title: "Transfers", target: nil, action: nil)\n',
        '    private let transferQueueButton = NSButton(title: "Transfers", target: nil, action: nil)\n'
        '    private let siteManagerButton = NSButton(title: "Site Manager", target: nil, action: nil)\n',
        "main-button-property",
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
        "        siteManagerButton.action = #selector(siteManagerTapped)\n",
        "main-button-action",
    )
    text = replace_once(
        text,
        "        let buttonRow = NSStackView(views: [connectButton, disconnectButton, directoryCompareButton, transferQueueButton, statusLabel])\n",
        "        let buttonRow = NSStackView(views: [connectButton, disconnectButton, siteManagerButton, directoryCompareButton, transferQueueButton, statusLabel])\n",
        "main-button-row",
    )
    text = replace_once(
        text,
        "\n\n    private func startTransferQueuePolling() {\n",
        "\n\n    @objc private func siteManagerTapped() {\n"
        "        guard engineReady, !connectionBusy else { return }\n"
        "        if let controller = siteManagerController {\n"
        "            controller.showAndRefresh()\n"
        "            return\n"
        "        }\n"
        "        let controller = SiteManagerWindowController()\n"
        "        controller.onConnected = { [weak self] protocolName, remoteStart, localStart in\n"
        "            guard let self else { return }\n"
        "            let local = localStart.trimmingCharacters(in: .whitespacesAndNewlines)\n"
        "            let remote = remoteStart.trimmingCharacters(in: .whitespacesAndNewlines)\n"
        "            let remoteTarget = remote.isEmpty ? (protocolName.lowercased() == \"sftp\" ? \".\" : \"/\") : remote\n"
        "            self.remoteCurrent = remoteTarget\n"
        "            self.refreshConnectionState()\n"
        "            self.refreshLocal(local.isEmpty ? self.localCurrent : local)\n"
        "            self.refreshRemote(remoteTarget)\n"
        "        }\n"
        "        siteManagerController = controller\n"
        "        controller.showAndRefresh()\n"
        "    }\n"
        "\n"
        "    private func startTransferQueuePolling() {\n",
        "main-site-manager-action",
    )
    text = replace_once(
        text,
        "        transferQueueButton.isEnabled = engineReady && !connectionBusy\n",
        "        siteManagerButton.isEnabled = engineReady && !connectionBusy\n"
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
