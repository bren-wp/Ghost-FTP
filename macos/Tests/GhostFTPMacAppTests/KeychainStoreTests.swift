import Security
import XCTest
@testable import GhostFTPMacApp

final class KeychainStoreTests: XCTestCase {
    func testUpdatingExistingPasswordNeverDeletesOrInserts() throws {
        var updates = 0
        var inserts = 0
        try KeychainStore.updateOrInsert(
            update: {
                updates += 1
                return errSecSuccess
            },
            insert: {
                inserts += 1
                return errSecSuccess
            }
        )
        XCTAssertEqual(updates, 1)
        XCTAssertEqual(inserts, 0)
    }

    func testFailedExistingItemUpdateDoesNotInsertOrDelete() {
        var insertCalled = false
        XCTAssertThrowsError(try KeychainStore.updateOrInsert(
            update: { errSecAuthFailed },
            insert: {
                insertCalled = true
                return errSecSuccess
            }
        )) { error in
            guard case KeychainStoreError.unexpectedStatus(let status) = error else {
                return XCTFail("Expected Keychain authorization failure")
            }
            XCTAssertEqual(status, errSecAuthFailed)
        }
        XCTAssertFalse(insertCalled)
    }

    func testMissingPasswordIsInserted() throws {
        var inserted = false
        try KeychainStore.updateOrInsert(
            update: { errSecItemNotFound },
            insert: {
                inserted = true
                return errSecSuccess
            }
        )
        XCTAssertTrue(inserted)
    }

    func testConcurrentInsertRetriesUpdateWithoutDeleting() throws {
        var updates = 0
        var inserts = 0
        try KeychainStore.updateOrInsert(
            update: {
                updates += 1
                return updates == 1 ? errSecItemNotFound : errSecSuccess
            },
            insert: {
                inserts += 1
                return errSecDuplicateItem
            }
        )
        XCTAssertEqual(updates, 2)
        XCTAssertEqual(inserts, 1)
    }

    func testFailedInsertReportsError() {
        XCTAssertThrowsError(try KeychainStore.updateOrInsert(
            update: { errSecItemNotFound },
            insert: { errSecNotAvailable }
        )) { error in
            guard case KeychainStoreError.unexpectedStatus(let status) = error else {
                return XCTFail("Expected unavailable Keychain status")
            }
            XCTAssertEqual(status, errSecNotAvailable)
        }
    }

    func testConcurrentInsertRetryFailureReportsError() {
        var attempts = 0
        XCTAssertThrowsError(try KeychainStore.updateOrInsert(
            update: {
                attempts += 1
                return attempts == 1 ? errSecItemNotFound : errSecAuthFailed
            },
            insert: { errSecDuplicateItem }
        )) { error in
            guard case KeychainStoreError.unexpectedStatus(let status) = error else {
                return XCTFail("Expected retry failure")
            }
            XCTAssertEqual(status, errSecAuthFailed)
        }
        XCTAssertEqual(attempts, 2)
    }
}
