import Foundation

enum ConnectionProtocol: String, Codable, CaseIterable, Identifiable {
    case ftp
    case ftps
    case sftp

    var id: String { rawValue }

    var title: String {
        switch self {
        case .ftp: return "FTP"
        case .ftps: return "Explicit FTPS"
        case .sftp: return "SFTP"
        }
    }

    var defaultPort: UInt16 {
        switch self {
        case .ftp, .ftps: return 21
        case .sftp: return 22
        }
    }

    var securitySummary: String {
        switch self {
        case .ftp:
            return "FTP traffic is not encrypted. Credentials and file contents can be exposed in transit."
        case .ftps:
            return "Explicit FTPS upgrades the FTP control channel with TLS and must validate the server certificate and hostname."
        case .sftp:
            return "SFTP runs over SSH and must verify the server host key before credentials or file operations are allowed."
        }
    }
}

struct ConnectionProfile: Identifiable, Codable, Equatable {
    var id: UUID
    var name: String
    var protocolKind: ConnectionProtocol
    var host: String
    var port: UInt16
    var username: String
    var keepAliveSeconds: UInt16
    var reconnectAttempts: UInt8

    init(
        id: UUID = UUID(),
        name: String = "",
        protocolKind: ConnectionProtocol = .sftp,
        host: String = "",
        port: UInt16? = nil,
        username: String = "",
        keepAliveSeconds: UInt16 = 15,
        reconnectAttempts: UInt8 = 1
    ) {
        self.id = id
        self.name = name
        self.protocolKind = protocolKind
        self.host = host
        self.port = port ?? protocolKind.defaultPort
        self.username = username
        self.keepAliveSeconds = keepAliveSeconds
        self.reconnectAttempts = reconnectAttempts
    }
}
