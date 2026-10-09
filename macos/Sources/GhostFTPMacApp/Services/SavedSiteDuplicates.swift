import Foundation

/// Advisory saved-site identity consistent with Windows/Linux Site Manager.
/// Site name, remote path, password and other authentication data do not
/// participate in comparison; no profile or Keychain record is ever mutated.
enum SavedSiteDuplicates {
    private struct Endpoint: Hashable {
        let connectionProtocol: ConnectionProtocol
        let hostname: String
        let port: UInt16
        let username: String
    }

    private static func endpoint(for profile: ConnectionProfile) -> Endpoint? {
        let host = profile.host.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let normalizedHost = host.hasSuffix(".") ? String(host.dropLast()) : host
        let username = profile.username.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedHost.isEmpty, !username.isEmpty, profile.port > 0 else {
            return nil
        }
        return Endpoint(
            connectionProtocol: profile.protocolKind,
            hostname: normalizedHost,
            port: profile.port,
            username: username
        )
    }

    static func matchingProfiles(
        for candidate: ConnectionProfile,
        in profiles: [ConnectionProfile]
    ) -> [ConnectionProfile] {
        guard let identity = endpoint(for: candidate) else { return [] }
        return profiles.filter {
            $0.id != candidate.id && endpoint(for: $0) == identity
        }
    }

    static func duplicateIDs(in profiles: [ConnectionProfile]) -> Set<UUID> {
        var firstByEndpoint: [Endpoint: UUID] = [:]
        var duplicates = Set<UUID>()
        for profile in profiles {
            guard let identity = endpoint(for: profile) else { continue }
            if let first = firstByEndpoint[identity] {
                duplicates.insert(first)
                duplicates.insert(profile.id)
            } else {
                firstByEndpoint[identity] = profile.id
            }
        }
        return duplicates
    }
}
