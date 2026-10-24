import Foundation
import Darwin

enum ConnectionValidationError: LocalizedError, Equatable {
    case missingName
    case missingHost
    case invalidHost
    case invalidPort
    case missingUsername
    case invalidKeepAlive
    case invalidReconnectAttempts

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
        case .invalidKeepAlive:
            return "Choose an SFTP keep-alive interval between 5 and 300 seconds."
        case .invalidReconnectAttempts:
            return "Choose between 0 and 3 automatic reconnect attempts."
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
        if host.contains("://") || host.contains("/") || host.contains("@") {
            throw ConnectionValidationError.invalidHost
        }
        // Bracketed and unbracketed literals must parse as real IPv6.
        // Merely counting colons or checking brackets allowed malformed
        // addresses through profile import and into native DNS routines.
        if host.contains(":") || host.hasPrefix("[") || host.hasSuffix("]") {
            let literal: String
            if host.hasPrefix("[") && host.hasSuffix("]") {
                literal = String(host.dropFirst().dropLast())
            } else if !host.contains("[") && !host.contains("]") {
                literal = host
            } else {
                throw ConnectionValidationError.invalidHost
            }
            // Darwin inet_pton accepts scoped IPv6 suffixes such as %en0.
            // Reject them here: interface scopes are not portable saved
            // connection identities and must not bypass strict host parsing.
            guard !literal.contains("%") else {
                throw ConnectionValidationError.invalidHost
            }
            var address = in6_addr()
            let valid = literal.withCString { value in
                inet_pton(AF_INET6, value, &address) == 1
            }
            guard valid else {
                throw ConnectionValidationError.invalidHost
            }
        }

        if profile.port == 0 {
            throw ConnectionValidationError.invalidPort
        }

        if profile.username.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            throw ConnectionValidationError.missingUsername
        }

        if profile.protocolKind == .sftp && !(5...300).contains(profile.keepAliveSeconds) {
            throw ConnectionValidationError.invalidKeepAlive
        }

        if profile.reconnectAttempts > 3 {
            throw ConnectionValidationError.invalidReconnectAttempts
        }
    }
}
