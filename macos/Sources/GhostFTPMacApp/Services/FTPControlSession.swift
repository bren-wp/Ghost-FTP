import Foundation
import Network

enum FTPControlError: LocalizedError {
    case unsupportedProtocol
    case invalidPort
    case timedOut
    case connectionFailed(String)
    case disconnected
    case malformedReply
    case unexpectedReply(expected: String, actual: Int, message: String)
    case unsafeCommandArgument
    case invalidWorkingDirectory

    var errorDescription: String? {
        switch self {
        case .unsupportedProtocol:
            return "This session engine currently supports unencrypted FTP only."
        case .invalidPort:
            return "The selected FTP port is invalid."
        case .timedOut:
            return "The FTP server did not respond before the timeout."
        case .connectionFailed:
            return "The FTP connection could not be established."
        case .disconnected:
            return "The FTP server closed the connection."
        case .malformedReply:
            return "The FTP server returned a malformed control reply."
        case .unexpectedReply(let expected, let actual, _):
            return "The FTP server returned reply code \(actual); expected \(expected)."
        case .unsafeCommandArgument:
            return "The FTP command contains unsupported control characters."
        case .invalidWorkingDirectory:
            return "The FTP server did not return a valid working directory."
        }
    }

    var diagnosticDescription: String {
        switch self {
        case .connectionFailed(let detail):
            return detail
        case .unexpectedReply(_, _, let message):
            return message
        default:
            return errorDescription ?? "FTP control error."
        }
    }
}

struct FTPReply: Equatable {
    let code: Int
    let lines: [String]

    var message: String {
        lines.joined(separator: "\n")
    }
}

enum FTPControlCodec {
    private static let crlf = Data([13, 10])

    static func takeReply(from buffer: inout Data) throws -> FTPReply? {
        let bytes = [UInt8](buffer)

        func nextLine(from offset: Int) -> (line: String, nextOffset: Int)? {
            guard offset < bytes.count else { return nil }
            var index = offset
            while index + 1 < bytes.count {
                if bytes[index] == 13 && bytes[index + 1] == 10 {
                    let line = String(decoding: bytes[offset..<index], as: UTF8.self)
                    return (line, index + 2)
                }
                index += 1
            }
            return nil
        }

        guard let first = nextLine(from: 0) else { return nil }
        guard first.line.count >= 3,
              let code = Int(first.line.prefix(3)) else {
            throw FTPControlError.malformedReply
        }

        let separator: Character? = first.line.count > 3
            ? first.line[first.line.index(first.line.startIndex, offsetBy: 3)]
            : nil

        if separator == nil || separator == " " {
            buffer.removeFirst(first.nextOffset)
            return FTPReply(code: code, lines: [first.line])
        }

        guard separator == "-" else {
            throw FTPControlError.malformedReply
        }

        var lines = [first.line]
        var offset = first.nextOffset
        let terminator = "\(code) "

        while let item = nextLine(from: offset) {
            lines.append(item.line)
            offset = item.nextOffset
            if item.line.hasPrefix(terminator) {
                buffer.removeFirst(offset)
                return FTPReply(code: code, lines: lines)
            }
        }

        return nil
    }

    static func workingDirectory(from reply: FTPReply) throws -> String {
        guard reply.code == 257, let firstLine = reply.lines.first else {
            throw FTPControlError.invalidWorkingDirectory
        }

        guard let firstQuote = firstLine.firstIndex(of: "\"") else {
            throw FTPControlError.invalidWorkingDirectory
        }

        var result = ""
        var index = firstLine.index(after: firstQuote)

        while index < firstLine.endIndex {
            let character = firstLine[index]
            if character == "\"" {
                let next = firstLine.index(after: index)
                if next < firstLine.endIndex, firstLine[next] == "\"" {
                    result.append("\"")
                    index = firstLine.index(after: next)
                    continue
                }
                return result
            }
            result.append(character)
            index = firstLine.index(after: index)
        }

        throw FTPControlError.invalidWorkingDirectory
    }

    static func validateCommandArgument(_ value: String) throws {
        if value.unicodeScalars.contains(where: { scalar in
            scalar.value == 0 || scalar.value == 10 || scalar.value == 13
        }) {
            throw FTPControlError.unsafeCommandArgument
        }
    }
}

private final class OneShotGate: @unchecked Sendable {
    private let lock = NSLock()
    private var claimed = false

    func claim() -> Bool {
        lock.lock()
        defer { lock.unlock() }
        guard !claimed else { return false }
        claimed = true
        return true
    }
}

actor FTPControlSession {
    private let queue = DispatchQueue(label: "com.brendigo.ghostftp.macos.ftp-control")
    private var connection: NWConnection?
    private var receiveBuffer = Data()

    func connect(
        profile: ConnectionProfile,
        password: String,
        timeoutSeconds: TimeInterval = 8
    ) async throws -> String {
        guard profile.protocolKind == .ftp else {
            throw FTPControlError.unsupportedProtocol
        }
        guard let port = NWEndpoint.Port(rawValue: profile.port) else {
            throw FTPControlError.invalidPort
        }

        try FTPControlCodec.validateCommandArgument(profile.username)
        try FTPControlCodec.validateCommandArgument(password)

        closeTransport()

        let connection = NWConnection(
            host: NWEndpoint.Host(profile.host),
            port: port,
            using: .tcp
        )
        self.connection = connection
        receiveBuffer.removeAll(keepingCapacity: true)

        do {
            try await waitUntilReady(connection, timeoutSeconds: timeoutSeconds)

            let greeting = try await readReply(timeoutSeconds: timeoutSeconds)
            try expect(greeting, accepted: [220], description: "220 service ready")

            let userReply = try await command(
                "USER \(profile.username)",
                timeoutSeconds: timeoutSeconds
            )
            switch userReply.code {
            case 230:
                break
            case 331:
                let passwordReply = try await command(
                    "PASS \(password)",
                    timeoutSeconds: timeoutSeconds
                )
                try expect(passwordReply, accepted: [230], description: "230 login successful")
            default:
                try expect(userReply, accepted: [230, 331], description: "230 or 331 authentication reply")
            }

            let typeReply = try await command("TYPE I", timeoutSeconds: timeoutSeconds)
            try expect(typeReply, accepted: [200], description: "200 binary transfer mode")

            return try await currentDirectory(timeoutSeconds: timeoutSeconds)
        } catch {
            closeTransport()
            throw error
        }
    }

    func currentDirectory(timeoutSeconds: TimeInterval = 8) async throws -> String {
        let reply = try await command("PWD", timeoutSeconds: timeoutSeconds)
        try expect(reply, accepted: [257], description: "257 current directory")
        return try FTPControlCodec.workingDirectory(from: reply)
    }

    func changeDirectory(
        to path: String,
        timeoutSeconds: TimeInterval = 8
    ) async throws -> String {
        try FTPControlCodec.validateCommandArgument(path)
        guard !path.isEmpty else {
            throw FTPControlError.invalidWorkingDirectory
        }

        let reply = try await command("CWD \(path)", timeoutSeconds: timeoutSeconds)
        try expect(reply, accepted: [250], description: "250 directory changed")
        return try await currentDirectory(timeoutSeconds: timeoutSeconds)
    }

    func noop(timeoutSeconds: TimeInterval = 8) async throws {
        let reply = try await command("NOOP", timeoutSeconds: timeoutSeconds)
        try expect(reply, accepted: [200], description: "200 NOOP")
    }

    func disconnect() async {
        guard connection != nil else { return }
        _ = try? await command("QUIT", timeoutSeconds: 2)
        closeTransport()
    }

    private func command(
        _ command: String,
        timeoutSeconds: TimeInterval
    ) async throws -> FTPReply {
        try FTPControlCodec.validateCommandArgument(command)
        try await sendLine(command, timeoutSeconds: timeoutSeconds)
        return try await readReply(timeoutSeconds: timeoutSeconds)
    }

    private func expect(
        _ reply: FTPReply,
        accepted: Set<Int>,
        description: String
    ) throws {
        guard accepted.contains(reply.code) else {
            throw FTPControlError.unexpectedReply(
                expected: description,
                actual: reply.code,
                message: reply.message
            )
        }
    }

    private func waitUntilReady(
        _ connection: NWConnection,
        timeoutSeconds: TimeInterval
    ) async throws {
        try await withCheckedThrowingContinuation { continuation in
            let gate = OneShotGate()

            connection.stateUpdateHandler = { state in
                switch state {
                case .ready:
                    if gate.claim() {
                        continuation.resume()
                    }
                case .failed(let error):
                    if gate.claim() {
                        continuation.resume(
                            throwing: FTPControlError.connectionFailed(error.localizedDescription)
                        )
                    }
                case .cancelled:
                    if gate.claim() {
                        continuation.resume(throwing: FTPControlError.disconnected)
                    }
                default:
                    break
                }
            }

            queue.asyncAfter(deadline: .now() + timeoutSeconds) {
                if gate.claim() {
                    connection.cancel()
                    continuation.resume(throwing: FTPControlError.timedOut)
                }
            }

            connection.start(queue: queue)
        }
    }

    private func sendLine(
        _ line: String,
        timeoutSeconds: TimeInterval
    ) async throws {
        guard let connection else {
            throw FTPControlError.disconnected
        }

        let data = Data((line + "\r\n").utf8)

        try await withCheckedThrowingContinuation { continuation in
            let gate = OneShotGate()

            queue.asyncAfter(deadline: .now() + timeoutSeconds) {
                if gate.claim() {
                    connection.cancel()
                    continuation.resume(throwing: FTPControlError.timedOut)
                }
            }

            connection.send(content: data, completion: .contentProcessed { error in
                guard gate.claim() else { return }
                if let error {
                    continuation.resume(
                        throwing: FTPControlError.connectionFailed(error.localizedDescription)
                    )
                } else {
                    continuation.resume()
                }
            })
        }
    }

    private func readReply(timeoutSeconds: TimeInterval) async throws -> FTPReply {
        while true {
            if let reply = try FTPControlCodec.takeReply(from: &receiveBuffer) {
                return reply
            }

            let chunk = try await receiveChunk(timeoutSeconds: timeoutSeconds)
            receiveBuffer.append(chunk)
        }
    }

    private func receiveChunk(timeoutSeconds: TimeInterval) async throws -> Data {
        guard let connection else {
            throw FTPControlError.disconnected
        }

        return try await withCheckedThrowingContinuation { continuation in
            let gate = OneShotGate()

            queue.asyncAfter(deadline: .now() + timeoutSeconds) {
                if gate.claim() {
                    connection.cancel()
                    continuation.resume(throwing: FTPControlError.timedOut)
                }
            }

            connection.receive(
                minimumIncompleteLength: 1,
                maximumLength: 64 * 1024
            ) { data, _, isComplete, error in
                guard gate.claim() else { return }

                if let error {
                    continuation.resume(
                        throwing: FTPControlError.connectionFailed(error.localizedDescription)
                    )
                    return
                }

                if let data, !data.isEmpty {
                    continuation.resume(returning: data)
                    return
                }

                if isComplete {
                    continuation.resume(throwing: FTPControlError.disconnected)
                    return
                }

                continuation.resume(throwing: FTPControlError.disconnected)
            }
        }
    }

    private func closeTransport() {
        connection?.stateUpdateHandler = nil
        connection?.cancel()
        connection = nil
        receiveBuffer.removeAll(keepingCapacity: false)
    }
}
