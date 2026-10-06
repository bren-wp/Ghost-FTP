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

    @Published private(set) var state: State = .idle
    @Published private(set) var entries: [FTPDirectoryEntry] = []
    @Published private(set) var isListing = false

    private let session = FTPControlSession()
    private var operation: Task<Void, Never>?

    var isConnected: Bool {
        if case .connected = state {
            return true
        }
        return false
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
        operation = nil
        state = .idle
        entries = []
        isListing = false

        Task {
            await session.disconnect()
        }
    }

    func cancel() {
        operation?.cancel()
        operation = nil
        state = .idle
        entries = []
        isListing = false

        Task {
            await session.cancel()
        }
    }
}
