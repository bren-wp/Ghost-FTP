import Foundation
import Security

enum KeychainStoreError: Error {
    case unexpectedStatus(OSStatus)
    case invalidData
}

struct KeychainStore {
    private let service = "com.brendigo.ghostftp.macos"

    func setPassword(_ password: String, for profileID: UUID) throws {
        let account = profileID.uuidString
        let encoded = Data(password.utf8)

        let baseQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecAttrSynchronizable as String: kCFBooleanFalse as Any,
        ]

        // Update in place: deleting first could lose the existing password
        // if a subsequent Keychain insertion fails or the device is locked.
        let attributes: [String: Any] = [
            kSecValueData as String: encoded,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
        ]
        try Self.updateOrInsert(
            update: {
                SecItemUpdate(baseQuery as CFDictionary, attributes as CFDictionary)
            },
            insert: {
                var item = baseQuery
                for (key, value) in attributes {
                    item[key] = value
                }
                return SecItemAdd(item as CFDictionary, nil)
            }
        )
    }

    // Injectable OSStatus operations keep the failure handling executable
    // in unit tests without touching a user's or CI runner's actual Keychain.
    static func updateOrInsert(
        update: () -> OSStatus,
        insert: () -> OSStatus
    ) throws {
        let updateStatus = update()
        if updateStatus == errSecSuccess { return }
        guard updateStatus == errSecItemNotFound else {
            throw KeychainStoreError.unexpectedStatus(updateStatus)
        }

        let insertStatus = insert()
        if insertStatus == errSecSuccess { return }
        // Another process may have created the same item after the first
        // lookup. Retry an in-place update without deleting that item.
        if insertStatus == errSecDuplicateItem {
            let retryStatus = update()
            if retryStatus == errSecSuccess { return }
            throw KeychainStoreError.unexpectedStatus(retryStatus)
        }
        throw KeychainStoreError.unexpectedStatus(insertStatus)
    }

    func password(for profileID: UUID) throws -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: profileID.uuidString,
            kSecAttrSynchronizable as String: kCFBooleanFalse as Any,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]

        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        if status == errSecItemNotFound {
            return nil
        }
        guard status == errSecSuccess else {
            throw KeychainStoreError.unexpectedStatus(status)
        }
        guard let data = result as? Data, let password = String(data: data, encoding: .utf8) else {
            throw KeychainStoreError.invalidData
        }
        return password
    }

    func removePassword(for profileID: UUID) throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: profileID.uuidString,
            kSecAttrSynchronizable as String: kCFBooleanFalse as Any,
        ]

        let status = SecItemDelete(query as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainStoreError.unexpectedStatus(status)
        }
    }
}
