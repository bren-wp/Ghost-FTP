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
    let fileName: String
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
        let id = UUID()
        records.insert(
            TransferHistoryRecord(
                id: id,
                direction: direction,
                fileName: fileName,
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

    private func load() {
        guard let data = defaults.data(forKey: storageKey),
              let decoded = try? JSONDecoder().decode([TransferHistoryRecord].self, from: data) else {
            records = []
            return
        }
        records = decoded.prefix(maximumRecords).map { $0 }
    }

    private func persist() {
        guard let data = try? JSONEncoder().encode(records) else { return }
        defaults.set(data, forKey: storageKey)
    }
}
