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
    case invalidPassiveEndpoint
    case invalidDirectoryListing
    case directoryListingTooLarge
    case invalidRemoteFileName
    case localFileFailure(String)

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
        case .invalidPassiveEndpoint:
            return "The FTP server did not return a valid passive data endpoint."
        case .invalidDirectoryListing:
            return "The FTP server returned a directory listing that Ghost FTP could not parse."
        case .directoryListingTooLarge:
            return "The FTP directory listing exceeded the safe 8 MiB limit."
        case .invalidRemoteFileName:
            return "The FTP remote file name is invalid."
        case .localFileFailure:
            return "Ghost FTP could not read or write the selected local file."
        }
    }

    var diagnosticDescription: String {
        switch self {
        case .connectionFailed(let detail):
            return detail
        case .unexpectedReply(_, _, let message):
            return message
        case .localFileFailure(let detail):
            return detail
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


struct FTPDirectoryEntry: Identifiable, Equatable {
    let name: String
    let isDirectory: Bool
    let size: UInt64?
    let modified: String?

    var id: String { name }
}

enum FTPControlCodec {
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


    static func validateRemoteFileName(_ value: String) throws {
        try validateCommandArgument(value)
        guard !value.isEmpty, value != ".", value != "..", !value.contains("/") else {
            throw FTPControlError.invalidRemoteFileName
        }
    }


    static func extendedPassivePort(from reply: FTPReply) throws -> UInt16 {
        guard reply.code == 229, let line = reply.lines.first,
              let open = line.firstIndex(of: "("),
              let close = line[open...].firstIndex(of: ")") else {
            throw FTPControlError.invalidPassiveEndpoint
        }

        let body = String(line[line.index(after: open)..<close])
        guard let delimiter = body.first else {
            throw FTPControlError.invalidPassiveEndpoint
        }

        let fields = body.split(separator: delimiter, omittingEmptySubsequences: false)
        guard fields.count >= 5,
              let port = UInt16(fields[3]),
              port > 0 else {
            throw FTPControlError.invalidPassiveEndpoint
        }
        return port
    }

    static func appendListingChunk(
        _ chunk: Data,
        to output: inout Data,
        maximumBytes: Int
    ) throws {
        guard maximumBytes > 0,
              chunk.count <= maximumBytes - output.count else {
            throw FTPControlError.directoryListingTooLarge
        }
        output.append(chunk)
    }

    static func parseMLSD(_ data: Data) throws -> [FTPDirectoryEntry] {
        guard let text = String(data: data, encoding: .utf8) else {
            throw FTPControlError.invalidDirectoryListing
        }

        var entries: [FTPDirectoryEntry] = []
        for rawLine in text.split(whereSeparator: \.isNewline) {
            let line = String(rawLine)
            guard let split = line.firstIndex(of: " ") else {
                continue
            }

            let factsText = line[..<split]
            let name = String(line[line.index(after: split)...])
            if name.isEmpty || name == "." || name == ".." {
                continue
            }

            var facts: [String: String] = [:]
            for fact in factsText.split(separator: ";") {
                guard let equals = fact.firstIndex(of: "=") else { continue }
                let key = fact[..<equals].lowercased()
                let value = String(fact[fact.index(after: equals)...])
                facts[key] = value
            }

            let type = facts["type"]?.lowercased()
            if type == "cdir" || type == "pdir" {
                continue
            }

            let size = facts["size"].flatMap(UInt64.init)
            entries.append(
                FTPDirectoryEntry(
                    name: name,
                    isDirectory: type == "dir",
                    size: size,
                    modified: facts["modify"]
                )
            )
        }

        return entries.sorted {
            if $0.isDirectory != $1.isDirectory {
                return $0.isDirectory && !$1.isDirectory
            }
            return $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending
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
    private var controlHost: String?

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
        controlHost = profile.host
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


    func downloadFile(
        remoteName: String,
        to localURL: URL,
        timeoutSeconds: TimeInterval = 20
    ) async throws {
        try FTPControlCodec.validateRemoteFileName(remoteName)

        let dataConnection = try await openPassiveDataConnection(timeoutSeconds: timeoutSeconds)
        let directoryURL = localURL.deletingLastPathComponent()
        let temporaryURL = directoryURL.appendingPathComponent(
            ".(localURL.lastPathComponent).ghostftp-(UUID().uuidString).part"
        )

        guard FileManager.default.createFile(atPath: temporaryURL.path, contents: nil) else {
            dataConnection.cancel()
            throw FTPControlError.localFileFailure("Could not create a temporary download file.")
        }

        do {
            let reply = try await command("RETR (remoteName)", timeoutSeconds: timeoutSeconds)
            try expect(
                reply,
                accepted: [125, 150],
                description: "125 or 150 download start"
            )

            let handle = try FileHandle(forWritingTo: temporaryURL)
            defer { try? handle.close() }

            while true {
                try Task.checkCancellation()
                let (chunk, complete) = try await receiveDataChunk(
                    dataConnection,
                    timeoutSeconds: timeoutSeconds
                )
                if !chunk.isEmpty {
                    try handle.write(contentsOf: chunk)
                }
                if complete {
                    break
                }
            }

            try handle.synchronize()
            dataConnection.cancel()

            let completion = try await readReply(timeoutSeconds: timeoutSeconds)
            try expect(
                completion,
                accepted: [226, 250],
                description: "226 or 250 download complete"
            )

            if FileManager.default.fileExists(atPath: localURL.path) {
                _ = try FileManager.default.replaceItemAt(localURL, withItemAt: temporaryURL)
            } else {
                try FileManager.default.moveItem(at: temporaryURL, to: localURL)
            }
        } catch {
            dataConnection.cancel()
            try? FileManager.default.removeItem(at: temporaryURL)
            closeTransport()
            if error is CancellationError {
                throw error
            }
            if let ftpError = error as? FTPControlError {
                throw ftpError
            }
            throw FTPControlError.localFileFailure(error.localizedDescription)
        }
    }

    func uploadFile(
        from localURL: URL,
        remoteName: String,
        timeoutSeconds: TimeInterval = 20
    ) async throws {
        try FTPControlCodec.validateRemoteFileName(remoteName)

        let handle: FileHandle
        do {
            handle = try FileHandle(forReadingFrom: localURL)
        } catch {
            throw FTPControlError.localFileFailure(error.localizedDescription)
        }
        defer { try? handle.close() }

        let dataConnection = try await openPassiveDataConnection(timeoutSeconds: timeoutSeconds)

        do {
            let reply = try await command("STOR (remoteName)", timeoutSeconds: timeoutSeconds)
            try expect(
                reply,
                accepted: [125, 150],
                description: "125 or 150 upload start"
            )

            while true {
                try Task.checkCancellation()
                guard let chunk = try handle.read(upToCount: 64 * 1024), !chunk.isEmpty else {
                    break
                }
                try await sendData(
                    chunk,
                    through: dataConnection,
                    timeoutSeconds: timeoutSeconds
                )
            }

            try await finishSending(
                dataConnection,
                timeoutSeconds: timeoutSeconds
            )

            let completion = try await readReply(timeoutSeconds: timeoutSeconds)
            try expect(
                completion,
                accepted: [226, 250],
                description: "226 or 250 upload complete"
            )
            dataConnection.cancel()
        } catch {
            dataConnection.cancel()
            closeTransport()
            if error is CancellationError {
                throw error
            }
            if let ftpError = error as? FTPControlError {
                throw ftpError
            }
            throw FTPControlError.localFileFailure(error.localizedDescription)
        }
    }

    func listDirectory(timeoutSeconds: TimeInterval = 8) async throws -> [FTPDirectoryEntry] {
        let dataConnection = try await openPassiveDataConnection(
            timeoutSeconds: timeoutSeconds
        )

        do {
            try await sendLine("MLSD", timeoutSeconds: timeoutSeconds)
            let preliminary = try await readReply(timeoutSeconds: timeoutSeconds)
            try expect(
                preliminary,
                accepted: [125, 150],
                description: "125 or 150 directory transfer start"
            )

            let payload = try await readDataUntilClosed(
                dataConnection,
                timeoutSeconds: timeoutSeconds
            )
            dataConnection.cancel()

            let completion = try await readReply(timeoutSeconds: timeoutSeconds)
            try expect(
                completion,
                accepted: [226, 250],
                description: "226 or 250 directory transfer complete"
            )

            return try FTPControlCodec.parseMLSD(payload)
        } catch {
            dataConnection.cancel()
            throw error
        }
    }

    func disconnect() async {
        guard connection != nil else { return }
        _ = try? await command("QUIT", timeoutSeconds: 2)
        closeTransport()
    }

    func cancel() {
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

        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
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

    private func openPassiveDataConnection(
        timeoutSeconds: TimeInterval
    ) async throws -> NWConnection {
        guard let controlHost else {
            throw FTPControlError.disconnected
        }

        let passiveReply = try await command("EPSV", timeoutSeconds: timeoutSeconds)
        try expect(passiveReply, accepted: [229], description: "229 extended passive mode")
        let dataPort = try FTPControlCodec.extendedPassivePort(from: passiveReply)
        guard let port = NWEndpoint.Port(rawValue: dataPort) else {
            throw FTPControlError.invalidPassiveEndpoint
        }

        let dataConnection = NWConnection(
            host: NWEndpoint.Host(controlHost),
            port: port,
            using: .tcp
        )
        try await waitUntilReady(dataConnection, timeoutSeconds: timeoutSeconds)
        return dataConnection
    }

    private func sendData(
        _ data: Data,
        through connection: NWConnection,
        timeoutSeconds: TimeInterval
    ) async throws {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
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

    private func finishSending(
        _ connection: NWConnection,
        timeoutSeconds: TimeInterval
    ) async throws {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            let gate = OneShotGate()

            queue.asyncAfter(deadline: .now() + timeoutSeconds) {
                if gate.claim() {
                    connection.cancel()
                    continuation.resume(throwing: FTPControlError.timedOut)
                }
            }

            connection.send(
                content: nil,
                contentContext: .finalMessage,
                isComplete: true,
                completion: .contentProcessed { error in
                    guard gate.claim() else { return }
                    if let error {
                        continuation.resume(
                            throwing: FTPControlError.connectionFailed(error.localizedDescription)
                        )
                    } else {
                        continuation.resume()
                    }
                }
            )
        }
    }

    private func readDataUntilClosed(
        _ connection: NWConnection,
        timeoutSeconds: TimeInterval,
        maximumBytes: Int = 8 * 1024 * 1024
    ) async throws -> Data {
        var output = Data()

        while true {
            let (chunk, complete) = try await receiveDataChunk(
                connection,
                timeoutSeconds: timeoutSeconds
            )
            try FTPControlCodec.appendListingChunk(
                chunk,
                to: &output,
                maximumBytes: maximumBytes
            )
            if complete {
                return output
            }
        }
    }

    private func receiveDataChunk(
        _ connection: NWConnection,
        timeoutSeconds: TimeInterval
    ) async throws -> (Data, Bool) {
        try await withCheckedThrowingContinuation { continuation in
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

                continuation.resume(returning: (data ?? Data(), isComplete))
            }
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
        controlHost = nil
        receiveBuffer.removeAll(keepingCapacity: false)
    }
}
