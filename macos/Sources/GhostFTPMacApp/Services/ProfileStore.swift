import Combine
import Foundation

enum ProfileStoreError: Error, Equatable {
    case backupTooLarge
    case tooManyProfiles
    case invalidProfileShape
    case credentialIdentityConflict
}

@MainActor
final class ProfileStore: ObservableObject {
    private static let maximumBackupBytes = 256 * 1024
    private static let maximumImportedProfiles = 512
    private static let maximumNameBytes = 256
    private static let maximumHostBytes = 512
    private static let maximumUsernameBytes = 512
    @Published private(set) var profiles: [ConnectionProfile] = []

    private let defaults: UserDefaults
    private let storageKey = "ghostftp.macos.profiles.v1"
    private let recoveryKey = "ghostftp.macos.profiles.recovery.v1"

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        load()
    }

    func addProfile() -> ConnectionProfile {
        let profile = ConnectionProfile()
        profiles.append(profile)
        persist()
        return profile
    }

    func save(_ profile: ConnectionProfile) {
        if let index = profiles.firstIndex(where: { $0.id == profile.id }) {
            profiles[index] = profile
        } else {
            profiles.append(profile)
        }

        profiles.sort { lhs, rhs in
            lhs.name.localizedCaseInsensitiveCompare(rhs.name) == .orderedAscending
        }
        persist()
    }

    func delete(_ id: UUID) {
        profiles.removeAll { $0.id == id }
        persist()
    }

    func exportProfiles() throws -> Data {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        return try encoder.encode(profiles)
    }

    @discardableResult
    func importProfiles(from data: Data) throws -> Int {
        guard data.count <= Self.maximumBackupBytes else {
            throw ProfileStoreError.backupTooLarge
        }

        let imported = try JSONDecoder().decode([ConnectionProfile].self, from: data)
        guard imported.count <= Self.maximumImportedProfiles else {
            throw ProfileStoreError.tooManyProfiles
        }
        try imported.forEach(validateBackupProfile)
        // Reject duplicate UUIDs inside one backup instead of silently losing
        // entries when a dictionary keeps only the final profile for a key.
        guard Set(imported.map(\.id)).count == imported.count else {
            throw ProfileStoreError.invalidProfileShape
        }

        let existingByID = Dictionary(uniqueKeysWithValues: profiles.map { ($0.id, $0) })
        // Keychain passwords are indexed by UUID. A backup must not silently
        // redirect an existing saved password to a different server/account.
        // Reject the entire import before changing the saved-site inventory.
        for profile in imported {
            guard let existing = existingByID[profile.id] else { continue }
            guard existing.protocolKind == profile.protocolKind,
                  existing.host == profile.host,
                  existing.port == profile.port,
                  existing.username == profile.username else {
                throw ProfileStoreError.credentialIdentityConflict
            }
        }

        var merged = existingByID
        for profile in imported {
            merged[profile.id] = profile
        }
        // Imported backups must not bypass the total saved-profile cap.
        // Fail before mutating in-memory or persisted profiles.
        guard merged.count <= Self.maximumImportedProfiles else {
            throw ProfileStoreError.tooManyProfiles
        }
        profiles = merged.values.sorted {
            $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending
        }
        persist()
        return imported.count
    }

    private func validateBackupProfile(_ profile: ConnectionProfile) throws {
        let fields = [
            (profile.name, Self.maximumNameBytes),
            (profile.host, Self.maximumHostBytes),
            (profile.username, Self.maximumUsernameBytes),
        ]
        let forbidden = CharacterSet.controlCharacters
        guard fields.allSatisfy({ field in
            let (value, maximumBytes) = field
            return value.utf8.count <= maximumBytes
                && !value.unicodeScalars.contains(where: forbidden.contains)
        }) else {
            throw ProfileStoreError.invalidProfileShape
        }
    }

    private func load() {
        guard let data = defaults.data(forKey: storageKey) else {
            profiles = []
            return
        }

        do {
            guard data.count <= Self.maximumBackupBytes else {
                throw ProfileStoreError.backupTooLarge
            }
            let stored = try JSONDecoder().decode([ConnectionProfile].self, from: data)
            guard stored.count <= Self.maximumImportedProfiles,
                  Set(stored.map(\.id)).count == stored.count else {
                throw ProfileStoreError.invalidProfileShape
            }
            try stored.forEach(validateBackupProfile)
            profiles = stored
        } catch {
            // Keep the unreadable original bytes for manual recovery before
            // the next site edit writes a valid replacement settings value.
            // Do not overwrite an older recovery copy on subsequent launches.
            if defaults.data(forKey: recoveryKey) == nil {
                defaults.set(data, forKey: recoveryKey)
            }
            profiles = []
        }
    }

    private func persist() {
        guard let data = try? JSONEncoder().encode(profiles) else { return }
        defaults.set(data, forKey: storageKey)
    }
}
