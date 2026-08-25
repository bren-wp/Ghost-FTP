import Foundation

enum TransferProtocol: String, CaseIterable, Identifiable, Sendable {
    case ftp = "FTP"
    case ftpsImplicit = "FTPS (implicit)"

    var id: String { rawValue }

    var defaultPort: UInt16 {
        switch self {
        case .ftp: return 21
        case .ftpsImplicit: return 990
        }
    }

    var usesTLS: Bool { self == .ftpsImplicit }
}

struct ConnectionConfig: Equatable, Sendable {
    let protocolKind: TransferProtocol
    let host: String
    let port: UInt16
    let username: String
    let password: String

    static func make(
        protocolKind: TransferProtocol,
        host rawHost: String,
        port rawPort: String,
        username rawUsername: String,
        password rawPassword: String
    ) throws -> ConnectionConfig {
        let host = try normalizeHost(rawHost)

        // Validate raw credentials before normalization. Otherwise CR/LF at an
        // edge could be removed by trimming instead of being rejected.
        try rejectControlCharacters(rawUsername, field: "Username")
        try rejectControlCharacters(rawPassword, field: "Password")

        let username = rawUsername.trimmingCharacters(in: .whitespaces)
        guard !username.isEmpty else { throw ValidationError("Username is required.") }

        return ConnectionConfig(
            protocolKind: protocolKind,
            host: host,
            port: try parsePort(rawPort, fallback: protocolKind.defaultPort),
            username: username,
            password: rawPassword
        )
    }

    static func normalizeHost(_ raw: String) throws -> String {
        var host = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !host.isEmpty else { throw ValidationError("Host is required.") }
        guard !host.contains("://"), !host.contains("/"), !host.contains("\\"), !host.unicodeScalars.contains(where: { $0.value == 0 }) else {
            throw ValidationError("Enter a host name or IP address, not a URL or path.")
        }
        guard host.unicodeScalars.allSatisfy({ !CharacterSet.whitespacesAndNewlines.contains($0) }) else {
            throw ValidationError("Host cannot contain whitespace.")
        }
        if host.hasPrefix("["), host.hasSuffix("]"), host.count > 2 {
            host.removeFirst()
            host.removeLast()
        }
        guard !host.isEmpty else { throw ValidationError("Host is required.") }
        return host
    }

    private static func parsePort(_ raw: String, fallback: UInt16) throws -> UInt16 {
        let value = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if value.isEmpty { return fallback }
        guard let parsed = UInt16(value), parsed > 0 else {
            throw ValidationError("Port must be between 1 and 65535.")
        }
        return parsed
    }

    private static func rejectControlCharacters(_ value: String, field: String) throws {
        let hasProtocolControl = value.unicodeScalars.contains { scalar in
            scalar.value == 0 || scalar.value == 10 || scalar.value == 13
        }
        guard !hasProtocolControl else {
            throw ValidationError("\(field) contains an unsafe control character.")
        }
    }
}

struct ValidationError: LocalizedError, Equatable, Sendable {
    let message: String

    init(_ message: String) { self.message = message }

    var errorDescription: String? { message }
}
