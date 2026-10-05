import Combine
import Foundation

@MainActor
final class ProfileStore: ObservableObject {
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
