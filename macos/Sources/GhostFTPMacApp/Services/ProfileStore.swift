import Combine
import Foundation

enum ProfileStoreError: Error {
    case backupTooLarge
    case tooManyProfiles
    case invalidProfileShape
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

        var merged = Dictionary(uniqueKeysWithValues: profiles.map { ($0.id, $0) })
        for profile in imported {
            merged[profile.id] = profile
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
        guard fields.allSatisfy({ value, maximumBytes in
            value.utf8.count <= maximumBytes
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
            profiles = try JSONDecoder().decode([ConnectionProfile].self, from: data)
        } catch {
            profiles = []
        }
    }

    private func persist() {
        guard let data = try? JSONEncoder().encode(profiles) else { return }
        defaults.set(data, forKey: storageKey)
    }
}
