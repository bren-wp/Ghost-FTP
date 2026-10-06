import Combine
import Foundation

@MainActor
final class FTPConnectionController: ObservableObject {
    enum State: Equatable {
        case idle
        case connecting
        case connected(String)
        case failed(String)
    }

    enum TransferState: Equatable {
        case idle
        case uploading(String)
        case downloading(String)
        case completed(String)
        case failed(String)
    }

    @Published private(set) var state: State = .idle
    @Published private(set) var entries: [FTPDirectoryEntry] = []
    @Published private(set) var isListing = false
    @Published private(set) var transferState: TransferState = .idle

    private let session = FTPControlSession()
    private let history = TransferHistoryStore.shared
    private var operation: Task<Void, Never>?
    private var activeTransferID: UUID?

    var isConnected: Bool {
        if case .connected = state {
            return true
        }
        return false
    }

    var isTransferring: Bool {
        switch transferState {
        case .uploading, .downloading:
            return true
        default:
            return false
        }
    }

    var workingDirectory: String? {
        if case .connected(let path) = state {
            return path
        }
        return nil
    }

    func connect(profile: ConnectionProfile, password: String) {
        operation?.cancel()
        state = .connecting

        operation = Task { [weak self] in
            guard let self else { return }

            do {
                let path = try await session.connect(profile: profile, password: password)
                guard !Task.isCancelled else {
                    await session.cancel()
                    return
                }
                state = .connected(path)
                await refreshDirectoryInternal()
            } catch {
                guard !Task.isCancelled else {
                    state = .idle
                    return
                }
                state = .failed(error.localizedDescription)
            }
        }
    }

    func changeDirectory(to path: String) {
        guard isConnected else { return }

        operation?.cancel()
        operation = Task { [weak self] in
            guard let self else { return }

            do {
                let resolved = try await session.changeDirectory(to: path)
                guard !Task.isCancelled else { return }
                state = .connected(resolved)
                await refreshDirectoryInternal()
            } catch {
                guard !Task.isCancelled else { return }
                state = .failed(error.localizedDescription)
            }
        }
    }

    func refreshWorkingDirectory() {
        guard isConnected else { return }

        operation?.cancel()
        operation = Task { [weak self] in
            guard let self else { return }

            do {
                let path = try await session.currentDirectory()
                guard !Task.isCancelled else { return }
                state = .connected(path)
            } catch {
                guard !Task.isCancelled else { return }
                state = .failed(error.localizedDescription)
            }
        }
    }


    func refreshDirectory() {
        guard isConnected else { return }

        operation?.cancel()
        operation = Task { [weak self] in
            guard let self else { return }
            await refreshDirectoryInternal()
        }
    }

    private func refreshDirectoryInternal() async {
        isListing = true
        defer { isListing = false }

        do {
            let result = try await session.listDirectory()
            guard !Task.isCancelled else { return }
            entries = result
        } catch {
            guard !Task.isCancelled else { return }
            state = .failed(error.localizedDescription)
        }
    }

    func uploadFile(from localURL: URL) {
        guard isConnected, !isTransferring else { return }
        let remoteName = localURL.lastPathComponent
        guard !remoteName.isEmpty else { return }

        transferState = .uploading(remoteName)
        let transferID = history.begin(direction: .upload, fileName: remoteName)
        activeTransferID = transferID
        operation?.cancel()
        operation = Task { [weak self] in
            guard let self else { return }

            let scoped = localURL.startAccessingSecurityScopedResource()
            defer {
                if scoped {
                    localURL.stopAccessingSecurityScopedResource()
                }
            }

            do {
                try await session.uploadFile(from: localURL, remoteName: remoteName)
                guard !Task.isCancelled else { return }
                history.complete(transferID)
                activeTransferID = nil
                transferState = .completed("Uploaded \(remoteName)")
                await refreshDirectoryInternal()
            } catch is CancellationError {
                history.cancel(transferID)
                activeTransferID = nil
                transferState = .idle
            } catch {
                history.fail(transferID)
                activeTransferID = nil
                transferState = .failed(error.localizedDescription)
                state = .failed(error.localizedDescription)
            }
        }
    }

    func downloadFile(_ entry: FTPDirectoryEntry, to localURL: URL) {
        guard isConnected, !isTransferring, !entry.isDirectory else { return }

        transferState = .downloading(entry.name)
        let transferID = history.begin(direction: .download, fileName: entry.name)
        activeTransferID = transferID
        operation?.cancel()
        operation = Task { [weak self] in
            guard let self else { return }

            let scoped = localURL.startAccessingSecurityScopedResource()
            defer {
                if scoped {
                    localURL.stopAccessingSecurityScopedResource()
                }
            }

            do {
                try await session.downloadFile(remoteName: entry.name, to: localURL)
                guard !Task.isCancelled else { return }
                history.complete(transferID)
                activeTransferID = nil
                transferState = .completed("Downloaded \(entry.name)")
            } catch is CancellationError {
                history.cancel(transferID)
                activeTransferID = nil
                transferState = .idle
            } catch {
                history.fail(transferID)
                activeTransferID = nil
                transferState = .failed(error.localizedDescription)
                state = .failed(error.localizedDescription)
            }
        }
    }

    func verifyConnection() {
        guard isConnected else { return }

        operation?.cancel()
        operation = Task { [weak self] in
            guard let self else { return }

            do {
                try await session.noop()
                guard !Task.isCancelled else { return }
                if let path = workingDirectory {
                    state = .connected(path)
                }
            } catch {
                guard !Task.isCancelled else { return }
                state = .failed(error.localizedDescription)
            }
        }
    }

    func disconnect() {
        operation?.cancel()
        if let activeTransferID {
            history.cancel(activeTransferID)
            self.activeTransferID = nil
        }
        operation = nil
        state = .idle
        entries = []
        isListing = false
        transferState = .idle

        Task {
            await session.disconnect()
        }
    }

    func cancel() {
        operation?.cancel()
        if let activeTransferID {
            history.cancel(activeTransferID)
            self.activeTransferID = nil
        }
        operation = nil
        state = .idle
        entries = []
        isListing = false
        transferState = .idle

        Task {
            await session.cancel()
        }
    }
}
