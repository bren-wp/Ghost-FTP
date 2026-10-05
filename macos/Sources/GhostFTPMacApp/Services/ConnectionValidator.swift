import Foundation

enum ConnectionValidationError: LocalizedError, Equatable {
    case missingName
    case missingHost
    case invalidHost
    case invalidPort
    case missingUsername

    var errorDescription: String? {
        switch self {
        case .missingName:
            return "Enter a site name."
        case .missingHost:
            return "Enter a server hostname or IP address."
        case .invalidHost:
            return "The server address contains unsupported whitespace or control characters."
        case .invalidPort:
            return "Choose a port between 1 and 65535."
        case .missingUsername:
            return "Enter a username."
        }
    }
}

enum ConnectionValidator {
    static func validate(_ profile: ConnectionProfile) throws {
        if profile.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            throw ConnectionValidationError.missingName
        }

        let host = profile.host.trimmingCharacters(in: .whitespacesAndNewlines)
        if host.isEmpty {
            throw ConnectionValidationError.missingHost
        }

        let forbidden = CharacterSet.whitespacesAndNewlines.union(.controlCharacters)
        if host.unicodeScalars.contains(where: { forbidden.contains($0) }) {
            throw ConnectionValidationError.invalidHost
        }

        if profile.port == 0 {
            throw ConnectionValidationError.invalidPort
        }

        if profile.username.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            throw ConnectionValidationError.missingUsername
        }
    }
}
