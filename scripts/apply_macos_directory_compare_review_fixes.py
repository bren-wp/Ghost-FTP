from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SWIFT = ROOT / "macos/Sources/GhostFTPApp/main.swift"

source = SWIFT.read_text(encoding="utf-8")

old_start = '''        directoryCompareGeneration += 1
        let generation = directoryCompareGeneration
        directoryCompareBusy = true
        statusLabel.stringValue = "Comparing local and remote folders…"

        let controller = DirectoryCompareWindowController(
'''
new_start = '''        let operationToken = UInt64(GhostFTPPrepareDirectoryCompare())
        guard operationToken != 0 else {
            showError("Directory comparison could not be prepared safely.")
            return
        }
        directoryCompareGeneration += 1
        let generation = directoryCompareGeneration
        directoryCompareBusy = true
        statusLabel.stringValue = "Comparing local and remote folders…"

        let controller = DirectoryCompareWindowController(
'''
if source.count(old_start) != 1:
    raise SystemExit("unexpected startDirectoryCompare preparation block")
source = source.replace(old_start, new_start, 1)

old_call = '''            let code = Int32(GhostFTPCompareDirectories(local.pointer, remote.pointer))
'''
new_call = '''            let code = Int32(GhostFTPCompareDirectories(CUnsignedLongLong(operationToken), local.pointer, remote.pointer))
'''
if source.count(old_call) != 1:
    raise SystemExit("unexpected GhostFTPCompareDirectories call")
source = source.replace(old_call, new_call, 1)

old_open = '''    private func openComparedDirectoryBoth(index: Int) {
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
'''
new_open = '''    private func openComparedDirectoryBoth(index: Int) {
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
'''
if source.count(old_open) != 1:
    raise SystemExit("unexpected openComparedDirectoryBoth implementation")
source = source.replace(old_open, new_open, 1)

SWIFT.write_text(source, encoding="utf-8")
