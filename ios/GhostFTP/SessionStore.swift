import Foundation
import Combine
import Security

@MainActor
final class SessionStore: ObservableObject {
    @Published var protocolKind: TransferProtocol = .ftpsImplicit
    @Published var host = ""
    @Published var port = ""
    @Published var username = ""
    @Published var password = ""

    @Published private(set) var connected = false
    @Published private(set) var busy = false
    @Published private(set) var currentPath = "/"
    @Published private(set) var entries: [RemoteEntry] = []
    @Published private(set) var status = "Ready"
    @Published var errorMessage: String?
    @Published private(set) var downloadedFile: URL?
    @Published private(set) var hasSavedConnection = false
    @Published private(set) var transferFraction: Double?
    @Published private(set) var transferDetail: String?
    @Published private(set) var canStopAfterCurrent = false
    @Published private(set) var hostingDiagnostics: SharedHostingDiagnostics?

    private var client: FTPRemoteClient?
    private var connectingClient: FTPRemoteClient?
    private var generation: UInt64 = 0
    private var stopAfterCurrentRequested = false

    init() {
        guard let preset = ConnectionPresetKeychain.load() else { return }
        do {
            let config = try preset.validatedConfig()
            protocolKind = config.protocolKind
            host = config.host
            port = String(config.port)
            username = config.username
            hasSavedConnection = true
        } catch {
            ConnectionPresetKeychain.clear()
        }
    }

    func connect() {
        guard !busy, !connected else { return }
        let config: ConnectionConfig
        do {
            config = try ConnectionConfig.make(
                protocolKind: protocolKind,
                host: host,
                port: port,
                username: username,
                password: password
            )
        } catch {
            password = ""
            present(error)
            return
        }
        let preset = ConnectionPreset(config: config)
        let diagnosticProtocol = config.protocolKind

        generation &+= 1
        let token = generation
        let next = FTPRemoteClient(config: config)
        connectingClient = next
        busy = true
        status = "Connecting…"
        errorMessage = nil
        hostingDiagnostics = nil

        Task {
            do {
                try await next.connect()
                let initial = try await next.list("/")
                let diagnostics = SharedHostingDiagnostics.analyze(protocolKind: diagnosticProtocol, entries: initial)
                guard token == generation else {
                    await next.close()
                    return
                }
                connectingClient = nil
                client = next
                password = ""
                connected = true
                currentPath = "/"
                entries = sortedEntries(initial)
                hostingDiagnostics = diagnostics
                hasSavedConnection = ConnectionPresetKeychain.save(preset)
                busy = false
                status = "Connected"
            } catch {
                await next.close()
                guard token == generation else { return }
                connectingClient = nil
                password = ""
                hostingDiagnostics = nil
                busy = false
                status = "Ready"
                present(error)
            }
        }
    }

    func disconnect() {
        generation &+= 1
        let current = client
        let pending = connectingClient
        client = nil
        connectingClient = nil
        connected = false
        busy = false
        currentPath = "/"
        entries = []
        hostingDiagnostics = nil
        password = ""
        finishTransfer()
        clearDownloadedFile()
        status = "Disconnected"
        Task {
            await current?.close()
            await pending?.close()
        }
    }

    func forgetSavedConnection() {
        ConnectionPresetKeychain.clear()
        hasSavedConnection = false
        status = connected ? "Connected · saved connection cleared" : "Saved connection cleared"
    }

    func refresh() {
        openDirectory(currentPath)
    }

    func openDirectory(_ path: String) {
        guard let client, !busy else { return }
        let normalized: String
        do { normalized = try RemotePath.normalizeDirectory(path) }
        catch { present(error); return }

        perform(statusText: "Loading…") {
            let next = try await client.list(normalized)
            return { [weak self] in
                self?.currentPath = normalized
                self?.entries = self?.sortedEntries(next) ?? next
            }
        }
    }

    func goUp() {
        guard currentPath != "/" else { return }
        do { openDirectory(try RemotePath.parent(currentPath)) }
        catch { present(error) }
    }

    func upload(_ url: URL) {
        upload([url])
    }

    func upload(_ urls: [URL]) {
        guard let client, !busy, !urls.isEmpty else { return }
        var jobs: [(url: URL, remotePath: String, name: String)] = []
        var remoteNames = Set<String>()
        do {
            for url in urls {
                let name = try RemotePath.validateName(url.lastPathComponent)
                guard remoteNames.insert(name).inserted else {
                    throw ValidationError("Two selected files have the same remote name: \(name)")
                }
                jobs.append((url, try RemotePath.child(currentPath, name), name))
            }
        } catch {
            present(error)
            return
        }

        beginTransfer(canStop: jobs.count > 1)
        let statusText = jobs.count == 1 ? "Uploading…" : "Uploading \(jobs.count) files…"
        perform(statusText: statusText, refreshAfter: true, endsTransfer: true) { [weak self] in
            guard let self else { return nil }
            for (index, job) in jobs.enumerated() {
                let granted = job.url.startAccessingSecurityScopedResource()
                defer { if granted { job.url.stopAccessingSecurityScopedResource() } }
                let totalBytes = localFileSize(job.url)
                renderTransferProgress(
                    upload: true,
                    fileIndex: index + 1,
                    fileCount: jobs.count,
                    name: job.name,
                    transferred: 0,
                    totalBytes: totalBytes
                )
                let progress: @Sendable (Int64) -> Void = { [weak self] bytes in
                    Task { @MainActor [weak self] in
                        self?.renderTransferProgress(
                            upload: true,
                            fileIndex: index + 1,
                            fileCount: jobs.count,
                            name: job.name,
                            transferred: bytes,
                            totalBytes: totalBytes
                        )
                    }
                }
                try await client.upload(remotePath: job.remotePath, localURL: job.url, progress: progress)
                renderTransferProgress(
                    upload: true,
                    fileIndex: index + 1,
                    fileCount: jobs.count,
                    name: job.name,
                    transferred: totalBytes ?? 0,
                    totalBytes: totalBytes
                )
                if stopAfterCurrentRequested && index + 1 < jobs.count { break }
            }
            return nil
        }
    }

    func requestStopAfterCurrent() {
        guard busy, canStopAfterCurrent else { return }
        stopAfterCurrentRequested = true
        canStopAfterCurrent = false
        transferDetail = "Stopping after the current file…"
    }

    func download(_ entry: RemoteEntry) {
        guard let client, !busy, !entry.isDirectory else { return }
        let remotePath: String
        do { remotePath = try RemotePath.child(currentPath, entry.name) }
        catch { present(error); return }

        clearDownloadedFile()
        let destination = FileManager.default.temporaryDirectory
            .appendingPathComponent("GhostFTP-\(UUID().uuidString)", isDirectory: true)
            .appendingPathComponent(entry.name, isDirectory: false)
        let temporaryParent = destination.deletingLastPathComponent()
        do {
            try FileManager.default.createDirectory(at: temporaryParent, withIntermediateDirectories: true)
        } catch {
            present(error)
            return
        }

        beginTransfer(canStop: false)
        renderTransferProgress(
            upload: false,
            fileIndex: 1,
            fileCount: 1,
            name: entry.name,
            transferred: 0,
            totalBytes: max(0, entry.size)
        )
        perform(
            statusText: "Downloading…",
            discard: { try? FileManager.default.removeItem(at: temporaryParent) },
            endsTransfer: true
        ) { [weak self] in
            guard let self else { return nil }
            let totalBytes = max(0, entry.size)
            let progress: @Sendable (Int64) -> Void = { [weak self] bytes in
                Task { @MainActor [weak self] in
                    self?.renderTransferProgress(
                        upload: false,
                        fileIndex: 1,
                        fileCount: 1,
                        name: entry.name,
                        transferred: bytes,
                        totalBytes: totalBytes
                    )
                }
            }
            try await client.download(remotePath: remotePath, localURL: destination, progress: progress)
            return { [weak self] in self?.downloadedFile = destination }
        }
    }

    func createFolder(named rawName: String) {
        guard let client, !busy else { return }
        let remotePath: String
        do { remotePath = try RemotePath.child(currentPath, rawName) }
        catch { present(error); return }
        perform(statusText: "Creating folder…", refreshAfter: true) {
            try await client.makeDirectory(remotePath)
            return nil
        }
    }

    func rename(_ entry: RemoteEntry, to rawName: String) {
        guard let client, !busy else { return }
        let source: String
        let destination: String
        do {
            source = try RemotePath.child(currentPath, entry.name)
            destination = try RemotePath.child(currentPath, rawName)
        } catch {
            present(error)
            return
        }
        perform(statusText: "Renaming…", refreshAfter: true) {
            try await client.rename(from: source, to: destination)
            return nil
        }
    }

    func delete(_ entry: RemoteEntry) {
        guard let client, !busy else { return }
        let remotePath: String
        do { remotePath = try RemotePath.child(currentPath, entry.name) }
        catch { present(error); return }
        perform(statusText: "Deleting…", refreshAfter: true) {
            try await client.delete(remotePath, directory: entry.isDirectory)
            return nil
        }
    }

    func clearDownloadedFile() {
        guard let file = downloadedFile else { return }
        let parent = file.deletingLastPathComponent()
        downloadedFile = nil
        try? FileManager.default.removeItem(at: parent)
    }

    private func sortedEntries(_ values: [RemoteEntry]) -> [RemoteEntry] {
        values.sorted { left, right in
            if left.isDirectory != right.isDirectory { return left.isDirectory }
            let lhs = left.name.lowercased()
            let rhs = right.name.lowercased()
            if lhs != rhs { return lhs < rhs }
            return left.name < right.name
        }
    }

    private func beginTransfer(canStop: Bool) {
        transferFraction = nil
        transferDetail = nil
        canStopAfterCurrent = canStop
        stopAfterCurrentRequested = false
    }

    private func finishTransfer() {
        transferFraction = nil
        transferDetail = nil
        canStopAfterCurrent = false
        stopAfterCurrentRequested = false
    }

    private func localFileSize(_ url: URL) -> Int64? {
        guard let size = try? url.resourceValues(forKeys: [.fileSizeKey]).fileSize else { return nil }
        return Int64(max(0, size))
    }

    private func renderTransferProgress(
        upload: Bool,
        fileIndex: Int,
        fileCount: Int,
        name: String,
        transferred: Int64,
        totalBytes: Int64?
    ) {
        let safeTransferred = max(0, transferred)
        if let totalBytes {
            let safeTotal = max(0, totalBytes)
            let fraction = safeTotal == 0 ? 1.0 : min(1.0, Double(safeTransferred) / Double(safeTotal))
            transferFraction = fraction
            let percent = Int((fraction * 100.0).rounded(.down))
            if stopAfterCurrentRequested && upload {
                transferDetail = "Stopping after the current file…"
            } else if upload {
                transferDetail = "Uploading \(fileIndex)/\(fileCount) · \(percent)% · \(name)"
            } else {
                transferDetail = "Downloading · \(percent)% · \(ByteCountFormatter.string(fromByteCount: safeTransferred, countStyle: .file))"
            }
        } else {
            transferFraction = nil
            if stopAfterCurrentRequested && upload {
                transferDetail = "Stopping after the current file…"
            } else if upload {
                transferDetail = "Uploading \(fileIndex)/\(fileCount) · \(ByteCountFormatter.string(fromByteCount: safeTransferred, countStyle: .file)) transferred · \(name)"
            } else {
                transferDetail = "Downloading · \(ByteCountFormatter.string(fromByteCount: safeTransferred, countStyle: .file)) transferred"
            }
        }
    }

    private func perform(
        statusText: String,
        refreshAfter: Bool = false,
        discard: (() -> Void)? = nil,
        endsTransfer: Bool = false,
        operation: @escaping () async throws -> (@MainActor () -> Void)?
    ) {
        guard !busy else { return }
        generation &+= 1
        let token = generation
        busy = true
        status = statusText
        errorMessage = nil

        Task {
            do {
                let apply = try await operation()
                guard token == generation else {
                    discard?()
                    if endsTransfer { finishTransfer() }
                    return
                }
                apply?()
                if refreshAfter, let client {
                    let refreshed = try await client.list(currentPath)
                    guard token == generation else {
                        discard?()
                        if endsTransfer { finishTransfer() }
                        return
                    }
                    entries = sortedEntries(refreshed)
                }
                if endsTransfer { finishTransfer() }
                busy = false
                status = connected ? "Connected" : "Ready"
            } catch {
                discard?()
                guard token == generation else {
                    if endsTransfer { finishTransfer() }
                    return
                }
                if endsTransfer { finishTransfer() }
                busy = false
                status = connected ? "Connected" : "Ready"
                present(error)
            }
        }
    }

    private func present(_ error: Error) {
        let raw = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        let clean = raw.replacingOccurrences(of: "\r", with: " ")
            .replacingOccurrences(of: "\n", with: " ")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        errorMessage = String((clean.isEmpty ? "Unknown error." : clean).prefix(320))
    }
}

private enum ConnectionPresetKeychain {
    private static let service = "com.GhostFTP.client.connection-preset"
    private static let account = "last-connection"

    static func load() -> ConnectionPreset? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecMatchLimit as String: kSecMatchLimitOne,
            kSecReturnData as String: true
        ]
        var result: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data else {
            return nil
        }
        return try? JSONDecoder().decode(ConnectionPreset.self, from: data)
    }

    @discardableResult
    static func save(_ preset: ConnectionPreset) -> Bool {
        guard let data = try? JSONEncoder().encode(preset) else { return false }
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
        let update: [String: Any] = [
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
            kSecValueData as String: data
        ]
        let updateStatus = SecItemUpdate(query as CFDictionary, update as CFDictionary)
        if updateStatus == errSecSuccess {
            return true
        }
        guard updateStatus == errSecItemNotFound else { return false }

        var attributes = query
        attributes[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        attributes[kSecValueData as String] = data
        return SecItemAdd(attributes as CFDictionary, nil) == errSecSuccess
    }

    static func clear() {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
        SecItemDelete(query as CFDictionary)
    }
}
