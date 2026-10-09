import XCTest
@testable import GhostFTPMacApp

@MainActor
final class WorkspaceStoreTests: XCTestCase {
    private func isolatedDefaults(_ suffix: String) -> UserDefaults {
        let suite = "com.brendigo.ghostftp.tests.\(suffix).\(UUID().uuidString)"
        let defaults = UserDefaults(suiteName: suite)!
        defaults.removePersistentDomain(forName: suite)
        return defaults
    }

    func testProfileBackupRoundTripExcludesPasswords() throws {
        let sourceDefaults = isolatedDefaults("profile-source")
        let source = ProfileStore(defaults: sourceDefaults)
        let original = ConnectionProfile(
            name: "Production",
            protocolKind: .ftp,
            host: "ftp.example.com",
            port: 21,
            username: "deploy",
            keepAliveSeconds: 15,
            reconnectAttempts: 1
        )
        source.save(original)

        let data = try source.exportProfiles()
        let text = String(decoding: data, as: UTF8.self)
        XCTAssertFalse(text.lowercased().contains("password"))

        let destination = ProfileStore(defaults: isolatedDefaults("profile-destination"))
        let count = try destination.importProfiles(from: data)

        XCTAssertEqual(count, 1)
        XCTAssertEqual(destination.profiles, [original])
    }

    func testProfileImportMergesByIdentifier() throws {
        let defaults = isolatedDefaults("profile-merge")
        let store = ProfileStore(defaults: defaults)
        let id = UUID()

        store.save(
            ConnectionProfile(
                id: id,
                name: "Old",
                protocolKind: .ftp,
                host: "old.example.com",
                username: "user"
            )
        )

        let replacement = ConnectionProfile(
            id: id,
            name: "Updated",
            protocolKind: .ftp,
            host: "new.example.com",
            username: "user"
        )
        let data = try JSONEncoder().encode([replacement])

        XCTAssertEqual(try store.importProfiles(from: data), 1)
        XCTAssertEqual(store.profiles.count, 1)
        XCTAssertEqual(store.profiles.first?.name, "Updated")
        XCTAssertEqual(store.profiles.first?.host, "new.example.com")
    }

    func testProfileImportRejectsTooManyProfiles() throws {
        let store = ProfileStore(defaults: isolatedDefaults("profile-count-limit"))
        let manyProfiles = (0..<513).map { index in
            ConnectionProfile(
                name: "Site \(index)",
                protocolKind: .ftp,
                host: "host-\(index).example.com",
                username: "user"
            )
        }
        let data = try JSONEncoder().encode(manyProfiles)

        XCTAssertThrowsError(try store.importProfiles(from: data))
        XCTAssertTrue(store.profiles.isEmpty)
    }

    func testProfileImportCannotOverflowTotalProfileLimit() throws {
        let defaults = isolatedDefaults("profile-merge-limit")
        let store = ProfileStore(defaults: defaults)
        let existing = (0..<512).map { index in
            ConnectionProfile(
                name: "Saved \(index)",
                protocolKind: .ftp,
                host: "existing-\(index).example.com",
                username: "user"
            )
        }
        let initialBackup = try JSONEncoder().encode(existing)
        XCTAssertEqual(try store.importProfiles(from: initialBackup), 512)
        let original = store.profiles
        let extra = ConnectionProfile(
            name: "Extra",
            protocolKind: .ftp,
            host: "new.example.com",
            username: "user"
        )
        let backup = try JSONEncoder().encode([extra])
        XCTAssertThrowsError(try store.importProfiles(from: backup))
        XCTAssertEqual(store.profiles, original)
        XCTAssertEqual(ProfileStore(defaults: defaults).profiles, original)
    }

    func testProfileImportRejectsControlCharacters() throws {
        let store = ProfileStore(defaults: isolatedDefaults("profile-control-limit"))
        let profile = ConnectionProfile(
            name: "Unsafe\u{0000}Site",
            protocolKind: .ftp,
            host: "ftp.example.com",
            username: "user"
        )
        let data = try JSONEncoder().encode([profile])

        XCTAssertThrowsError(try store.importProfiles(from: data))
        XCTAssertTrue(store.profiles.isEmpty)
    }

    func testTransferHistoryTracksCompletionCancellationAndPersistence() {
        let defaults = isolatedDefaults("transfer-history")
        let history = TransferHistoryStore(defaults: defaults)

        let upload = history.begin(direction: .upload, fileName: "release.zip")
        history.complete(upload)

        let download = history.begin(direction: .download, fileName: "backup.sql")
        history.cancel(download)

        XCTAssertEqual(history.records.count, 2)
        XCTAssertEqual(history.records[0].status, .cancelled)
        XCTAssertEqual(history.records[1].status, .completed)

        let reloaded = TransferHistoryStore(defaults: defaults)
        XCTAssertEqual(reloaded.records, history.records)
    }

    func testRunningTransferIsRecoveredAsCancelledAfterRestart() {
        let defaults = isolatedDefaults("transfer-recovery")
        let history = TransferHistoryStore(defaults: defaults)
        _ = history.begin(direction: .download, fileName: "interrupted.zip")

        let reloaded = TransferHistoryStore(defaults: defaults)

        XCTAssertEqual(reloaded.records.count, 1)
        XCTAssertEqual(reloaded.records.first?.status, .cancelled)
        XCTAssertNotNil(reloaded.records.first?.finishedAt)
    }

    func testTransferHistoryDoesNotPersistDirectoriesOrControlCharacters() {
        let defaults = isolatedDefaults("history-privacy")
        let history = TransferHistoryStore(defaults: defaults)
        _ = history.begin(
            direction: .download,
            fileName: "/Users/secret/Documents/report\u{000A}.pdf"
        )
        XCTAssertEqual(history.records.first?.fileName, "report.pdf")
        let persisted = defaults.data(forKey: "ghostftp.macos.transfer-history.v1")!
        let raw = String(decoding: persisted, as: UTF8.self)
        XCTAssertFalse(raw.contains("/Users/secret/"))
        XCTAssertFalse(raw.contains("Documents"))
        XCTAssertEqual(TransferHistoryStore(defaults: defaults).records.first?.fileName, "report.pdf")
    }

    func testTransferHistoryIsBoundedToTwoHundredRecords() {
        let history = TransferHistoryStore(defaults: isolatedDefaults("transfer-limit"))

        for index in 0..<205 {
            _ = history.begin(direction: .upload, fileName: "file-\(index).dat")
        }

        XCTAssertEqual(history.records.count, 200)
        XCTAssertEqual(history.records.first?.fileName, "file-204.dat")
        XCTAssertEqual(history.records.last?.fileName, "file-5.dat")
    }
}
