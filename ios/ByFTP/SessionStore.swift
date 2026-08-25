import Foundation
import Combine

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

    private var client: FTPRemoteClient?
    private var connectingClient: FTPRemoteClient?
    private var generation: UInt64 = 0

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

        generation &+= 1
        let token = generation
        let next = FTPRemoteClient(config: config)
        connectingClient = next
        busy = true
        status = "Connecting…"
        errorMessage = nil

        Task {
            do {
                try await next.connect()
                let initial = try await next.list("/")
                guard token == generation else {
                    await next.close()
                    return
                }
                connectingClient = nil
                client = next
                password = ""
                connected = true
                currentPath = "/"
                entries = initial
                busy = false
                status = "Connected"
            } catch {
                await next.close()
                guard token == generation else { return }
                connectingClient = nil
                password = ""
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
        password = ""
        clearDownloadedFile()
        status = "Disconnected"
        Task {
            await current?.close()
            await pending?.close()
        }
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
                self?.entries = next
            }
        }
    }

    func goUp() {
        guard currentPath != "/" else { return }
        do { openDirectory(try RemotePath.parent(currentPath)) }
        catch { present(error) }
    }

    func upload(_ url: URL) {
        guard let client, !busy else { return }
        let remotePath: String
        do { remotePath = try RemotePath.child(currentPath, url.lastPathComponent) }
        catch { present(error); return }

        perform(statusText: "Uploading…", refreshAfter: true) {
            let granted = url.startAccessingSecurityScopedResource()
            defer { if granted { url.stopAccessingSecurityScopedResource() } }
            try await client.upload(remotePath: remotePath, localURL: url)
            return nil
        }
    }

    func download(_ entry: RemoteEntry) {
        guard let client, !busy, !entry.isDirectory else { return }
        let remotePath: String
        do { remotePath = try RemotePath.child(currentPath, entry.name) }
        catch { present(error); return }

        clearDownloadedFile()
        let destination = FileManager.default.temporaryDirectory
            .appendingPathComponent("ByFTP-\(UUID().uuidString)", isDirectory: true)
            .appendingPathComponent(entry.name, isDirectory: false)
        let temporaryParent = destination.deletingLastPathComponent()
        do {
            try FileManager.default.createDirectory(at: temporaryParent, withIntermediateDirectories: true)
        } catch {
            present(error)
            return
        }

        perform(
            statusText: "Downloading…",
            discard: { try? FileManager.default.removeItem(at: temporaryParent) }
        ) {
            try await client.download(remotePath: remotePath, localURL: destination)
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

    private func perform(
        statusText: String,
        refreshAfter: Bool = false,
        discard: (() -> Void)? = nil,
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
                    return
                }
                apply?()
                if refreshAfter, let client {
                    let refreshed = try await client.list(currentPath)
                    guard token == generation else {
                        discard?()
                        return
                    }
                    entries = refreshed
                }
                busy = false
                status = connected ? "Connected" : "Ready"
            } catch {
                discard?()
                guard token == generation else { return }
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
