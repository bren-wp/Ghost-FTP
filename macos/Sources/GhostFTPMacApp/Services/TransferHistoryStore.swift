import Combine
import Foundation

struct TransferHistoryRecord: Codable, Identifiable, Equatable {
    enum Direction: String, Codable {
        case upload
        case download

        var title: String {
            switch self {
            case .upload: return "Upload"
            case .download: return "Download"
            }
        }
    }

    enum Status: String, Codable {
        case running
        case completed
        case failed
        case cancelled

        var title: String {
            switch self {
            case .running: return "In progress"
            case .completed: return "Completed"
            case .failed: return "Failed"
            case .cancelled: return "Cancelled"
            }
        }
    }

    let id: UUID
    let direction: Direction
    var fileName: String
    let startedAt: Date
    var finishedAt: Date?
    var status: Status
}

@MainActor
final class TransferHistoryStore: ObservableObject {
    static let shared = TransferHistoryStore()

    @Published private(set) var records: [TransferHistoryRecord] = []

    private let defaults: UserDefaults
    private let storageKey = "ghostftp.macos.transfer-history.v1"
    private let maximumRecords = 200

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        load()
    }

    @discardableResult
    func begin(direction: TransferHistoryRecord.Direction, fileName: String) -> UUID {
        // Do not persist complete local/remote paths in UserDefaults history.
        // File names may be untrusted: strip control characters and bound size.
        let safeName = sanitizedFileName(fileName)
        let id = UUID()
        records.insert(
            TransferHistoryRecord(
                id: id,
                direction: direction,
                fileName: safeName,
                startedAt: Date(),
                finishedAt: nil,
                status: .running
            ),
            at: 0
        )
        trimAndPersist()
        return id
    }

    func complete(_ id: UUID) {
        finish(id, status: .completed)
    }

    func fail(_ id: UUID) {
        finish(id, status: .failed)
    }

    func cancel(_ id: UUID) {
        finish(id, status: .cancelled)
    }

    func clear() {
        records.removeAll()
        persist()
    }

    private func finish(_ id: UUID, status: TransferHistoryRecord.Status) {
        guard let index = records.firstIndex(where: { $0.id == id }) else { return }
        records[index].status = status
        records[index].finishedAt = Date()
        trimAndPersist()
    }

    private func trimAndPersist() {
        if records.count > maximumRecords {
            records.removeLast(records.count - maximumRecords)
        }
        persist()
    }

    private func sanitizedFileName(_ fileName: String) -> String {
        let base = fileName.replacingOccurrences(of: "\\", with: "/")
            .split(separator: "/", omittingEmptySubsequences: true)
            .last.map(String.init) ?? ""
        let clean = String(base.filter { character in
            character.unicodeScalars.allSatisfy { !CharacterSet.controlCharacters.contains($0) }
        }.prefix(180))
        return clean.isEmpty ? "File" : clean
    }

    private func load() {
        guard let data = defaults.data(forKey: storageKey),
              let decoded = try? JSONDecoder().decode([TransferHistoryRecord].self, from: data) else {
            records = []
            return
        }
        records = decoded.prefix(maximumRecords).map { $0 }

        var needsPrivateHistoryMigration = false
        let recoveredAt = Date()
        for index in records.indices {
            // Upgrade entries persisted by earlier versions, which may carry
            // full private directory paths. Rewrite UserDefaults atomically.
            let clean = sanitizedFileName(records[index].fileName)
            if clean != records[index].fileName {
                records[index].fileName = clean
                needsPrivateHistoryMigration = true
            }
            if records[index].status == .running {
                records[index].status = .cancelled
                records[index].finishedAt = recoveredAt
                needsPrivateHistoryMigration = true
            }
        }
        if needsPrivateHistoryMigration {
            persist()
        }
    }

    private func persist() {
        guard let data = try? JSONEncoder().encode(records) else { return }
        defaults.set(data, forKey: storageKey)
    }
}
