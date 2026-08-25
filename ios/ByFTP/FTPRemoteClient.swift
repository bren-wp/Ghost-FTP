import Foundation

actor FTPRemoteClient {
    private let protocolKind: TransferProtocol
    private let host: String
    private let port: UInt16
    private let username: String
    private var password: String
    private var control: SocketConnection?
    private var loginRoot = ""

    init(config: ConnectionConfig) {
        protocolKind = config.protocolKind
        host = config.host
        port = config.port
        username = config.username
        password = config.password
    }

    func connect() async throws {
        guard control == nil else { return }
        let loginPassword = password
        defer { password = "" }

        let next = try SocketConnection(host: host, port: port, tls: protocolKind.usesTLS)
        do {
            try await next.start()
            let greeting = try await readResponse(from: next)
            guard greeting.code == 220 else { throw FTPError.unexpected(greeting) }

            let user = try await command("USER", username, on: next)
            switch user.code {
            case 230:
                break
            case 331, 332:
                let passwordReply = try await command("PASS", loginPassword, on: next)
                guard passwordReply.code == 230 else { throw FTPError.unexpected(passwordReply) }
            default:
                throw FTPError.unexpected(user)
            }

            if protocolKind.usesTLS {
                try expect2xx(try await command("PBSZ", "0", on: next))
                try expect2xx(try await command("PROT", "P", on: next))
            }
            try expect2xx(try await command("TYPE", "I", on: next))
            _ = try? await command("OPTS", "UTF8 ON", on: next)

            let pwd = try? await command("PWD", nil, on: next)
            loginRoot = try FTPPathMapper.normalizeLoginRoot(pwd.flatMap(parseWorkingDirectory))
            control = next
        } catch {
            next.cancel()
            control = nil
            loginRoot = ""
            throw error
        }
    }

    func close() async {
        password = ""
        guard let current = control else { return }
        _ = try? await command("QUIT", nil, on: current)
        current.cancel()
        control = nil
        loginRoot = ""
    }

    func list(_ directory: String) async throws -> [RemoteEntry] {
        let path = try FTPPathMapper.map(loginRoot: loginRoot, uiPath: directory)
        do {
            let data = try await receiveData(commandName: "MLSD", argument: path, maxBytes: 8 * 1024 * 1024)
            return parseMLSD(data)
        } catch let error as FTPError where error.isUnsupportedCommand {
            let data = try await receiveData(commandName: "LIST", argument: path, maxBytes: 8 * 1024 * 1024)
            return parseLIST(data)
        }
    }

    func upload(remotePath: String, localURL: URL) async throws {
        let path = try FTPPathMapper.map(loginRoot: loginRoot, uiPath: remotePath)
        let dataSocket = try await openPassiveDataSocket()
        do {
            let preliminary = try await command("STOR", path)
            try expectPreliminary(preliminary)
            try await dataSocket.sendFile(localURL)
            try await dataSocket.finishSending()
            dataSocket.cancel()
            try expect2xx(try await readResponse())
        } catch {
            dataSocket.cancel()
            throw error
        }
    }

    func download(remotePath: String, localURL: URL) async throws {
        let path = try FTPPathMapper.map(loginRoot: loginRoot, uiPath: remotePath)
        let dataSocket = try await openPassiveDataSocket()
        do {
            let preliminary = try await command("RETR", path)
            try expectPreliminary(preliminary)
            try await dataSocket.receiveToFile(localURL)
            dataSocket.cancel()
            try expect2xx(try await readResponse())
        } catch {
            dataSocket.cancel()
            try? FileManager.default.removeItem(at: localURL)
            throw error
        }
    }

    func makeDirectory(_ remotePath: String) async throws {
        let path = try FTPPathMapper.map(loginRoot: loginRoot, uiPath: remotePath)
        try expect2xx(try await command("MKD", path))
    }

    func rename(from: String, to: String) async throws {
        let source = try FTPPathMapper.map(loginRoot: loginRoot, uiPath: from)
        let destination = try FTPPathMapper.map(loginRoot: loginRoot, uiPath: to)
        let first = try await command("RNFR", source)
        guard first.code == 350 else { throw FTPError.unexpected(first) }
        try expect2xx(try await command("RNTO", destination))
    }

    func delete(_ remotePath: String, directory: Bool) async throws {
        let path = try FTPPathMapper.map(loginRoot: loginRoot, uiPath: remotePath)
        try expect2xx(try await command(directory ? "RMD" : "DELE", path))
    }

    private func receiveData(commandName: String, argument: String, maxBytes: Int) async throws -> Data {
        let dataSocket = try await openPassiveDataSocket()
        do {
            let preliminary = try await command(commandName, argument)
            try expectPreliminary(preliminary)
            let data = try await dataSocket.receiveAll(maxBytes: maxBytes)
            dataSocket.cancel()
            try expect2xx(try await readResponse())
            return data
        } catch {
            dataSocket.cancel()
            throw error
        }
    }

    private func openPassiveDataSocket() async throws -> SocketConnection {
        let dataPort = try await passivePort()
        let socket = try SocketConnection(host: host, port: dataPort, tls: protocolKind.usesTLS)
        do {
            try await socket.start()
            return socket
        } catch {
            socket.cancel()
            throw error
        }
    }

    private func passivePort() async throws -> UInt16 {
        let epsv = try await command("EPSV")
        if epsv.code == 229, let parsed = parseEPSVPort(epsv.message) { return parsed }

        let pasv = try await command("PASV")
        guard pasv.code == 227, let parsed = parsePASVPort(pasv.message) else {
            throw FTPError.unexpected(pasv)
        }
        // Intentionally ignore the server-supplied PASV host. Shared-hosting/NAT
        // servers often return private addresses, and trusting that address could
        // redirect the data connection away from the user-selected endpoint.
        return parsed
    }

    private func command(_ name: String, _ argument: String? = nil) async throws -> FTPResponse {
        try await command(name, argument, on: requireControl())
    }

    private func command(_ name: String, _ argument: String?, on socket: SocketConnection) async throws -> FTPResponse {
        guard name.allSatisfy({ $0.isASCII && ($0.isLetter || $0.isNumber) }) else {
            throw ValidationError("Invalid FTP command name.")
        }
        var line = name
        if let argument {
            guard !argument.contains("\r"), !argument.contains("\n"), !argument.contains("\0") else {
                throw ValidationError("FTP command argument contains an unsafe control character.")
            }
            line += " " + argument
        }
        try await socket.sendLine(line)
        return try await readResponse(from: socket)
    }

    private func readResponse() async throws -> FTPResponse {
        try await readResponse(from: requireControl())
    }

    private func readResponse(from socket: SocketConnection) async throws -> FTPResponse {
        let first = try await socket.readLine()
        guard first.count >= 3, let code = Int(first.prefix(3)) else {
            throw FTPError.malformedResponse(first)
        }
        var lines = [first]
        let markerIndex = first.index(first.startIndex, offsetBy: 3, limitedBy: first.endIndex)
        let multiline = markerIndex.map { first[$0] == "-" } ?? false
        if multiline {
            let terminal = "\(code) "
            for _ in 0..<99 {
                let line = try await socket.readLine()
                lines.append(line)
                if line.hasPrefix(terminal) { break }
            }
            guard lines.last?.hasPrefix(terminal) == true else {
                throw FTPError.malformedResponse("FTP multiline response did not terminate.")
            }
        }
        return FTPResponse(code: code, lines: lines)
    }

    private func expectPreliminary(_ response: FTPResponse) throws {
        guard response.code == 125 || response.code == 150 else { throw FTPError.unexpected(response) }
    }

    private func expect2xx(_ response: FTPResponse) throws {
        guard (200..<300).contains(response.code) else { throw FTPError.unexpected(response) }
    }

    private func requireControl() throws -> SocketConnection {
        guard let control else { throw FTPError.notConnected }
        return control
    }

    private func parseWorkingDirectory(_ response: FTPResponse) -> String? {
        guard response.code == 257 else { return nil }
        let text = response.message
        guard let first = text.firstIndex(of: "\"") else { return nil }
        var index = text.index(after: first)
        var value = ""
        while index < text.endIndex {
            let character = text[index]
            if character == "\"" {
                let next = text.index(after: index)
                if next < text.endIndex, text[next] == "\"" {
                    value.append("\"")
                    index = text.index(after: next)
                    continue
                }
                return value
            }
            value.append(character)
            index = text.index(after: index)
        }
        return nil
    }

    private func parseEPSVPort(_ text: String) -> UInt16? {
        guard let open = text.lastIndex(of: "("), let close = text[open...].firstIndex(of: ")") else { return nil }
        let inside = text[text.index(after: open)..<close]
        guard let delimiter = inside.first else { return nil }
        let parts = inside.split(separator: delimiter, omittingEmptySubsequences: false)
        guard parts.count >= 5 else { return nil }
        return UInt16(parts[3])
    }

    private func parsePASVPort(_ text: String) -> UInt16? {
        let pattern = #"(?:\d{1,3},){5}\d{1,3}"#
        guard let regex = try? NSRegularExpression(pattern: pattern),
              let match = regex.firstMatch(in: text, range: NSRange(text.startIndex..., in: text)),
              let range = Range(match.range, in: text) else { return nil }
        let values = text[range].split(separator: ",").compactMap { Int($0) }
        guard values.count == 6, values.allSatisfy({ (0...255).contains($0) }) else { return nil }
        let parsedPort = values[4] * 256 + values[5]
        guard (1...65535).contains(parsedPort) else { return nil }
        return UInt16(parsedPort)
    }

    private func parseMLSD(_ data: Data) -> [RemoteEntry] {
        guard let text = decodeListing(data) else { return [] }
        var result: [RemoteEntry] = []
        for rawLine in text.split(whereSeparator: { $0.isNewline }) {
            let line = String(rawLine)
            guard let space = line.firstIndex(of: " ") else { continue }
            let factsText = line[..<space]
            let rawName = String(line[line.index(after: space)...])
            guard let name = safeRemoteName(rawName) else { continue }

            var facts: [String: String] = [:]
            for fact in factsText.split(separator: ";") {
                guard let equals = fact.firstIndex(of: "=") else { continue }
                let key = fact[..<equals].lowercased()
                let value = String(fact[fact.index(after: equals)...])
                facts[key] = value
            }
            let type = facts["type"]?.lowercased() ?? "file"
            if type == "cdir" || type == "pdir" { continue }
            result.append(RemoteEntry(
                name: name,
                isDirectory: type == "dir",
                size: Int64(facts["size"] ?? "") ?? 0,
                modifiedAt: parseMLSDate(facts["modify"])
            ))
        }
        return sorted(result)
    }

    private func parseLIST(_ data: Data) -> [RemoteEntry] {
        guard let text = decodeListing(data) else { return [] }
        var result: [RemoteEntry] = []
        for rawLine in text.split(whereSeparator: { $0.isNewline }) {
            let line = String(rawLine).trimmingCharacters(in: .whitespacesAndNewlines)
            if line.isEmpty || line.lowercased().hasPrefix("total ") { continue }

            let unix = line.split(maxSplits: 8, whereSeparator: { $0.isWhitespace })
            if unix.count == 9, let kind = unix[0].first, kind == "d" || kind == "-" || kind == "l" {
                guard kind != "l", let name = safeRemoteName(String(unix[8])) else { continue }
                result.append(RemoteEntry(
                    name: name,
                    isDirectory: kind == "d",
                    size: Int64(unix[4]) ?? 0,
                    modifiedAt: nil
                ))
                continue
            }

            let windows = line.split(maxSplits: 3, whereSeparator: { $0.isWhitespace })
            if windows.count == 4, windows[0].contains("-"), windows[1].uppercased().contains("M") {
                guard let name = safeRemoteName(String(windows[3])) else { continue }
                let directory = windows[2].uppercased() == "<DIR>"
                result.append(RemoteEntry(
                    name: name,
                    isDirectory: directory,
                    size: directory ? 0 : (Int64(windows[2]) ?? 0),
                    modifiedAt: nil
                ))
            }
        }
        return sorted(result)
    }

    private func decodeListing(_ data: Data) -> String? {
        String(data: data, encoding: .utf8) ?? String(data: data, encoding: .isoLatin1)
    }

    private func safeRemoteName(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed == raw, !trimmed.isEmpty, trimmed != ".", trimmed != "..",
              !trimmed.contains("/"), !trimmed.contains("\\"), !trimmed.contains("\0") else { return nil }
        return trimmed
    }

    private func parseMLSDate(_ raw: String?) -> Date? {
        guard let raw, raw.count >= 14 else { return nil }
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.dateFormat = "yyyyMMddHHmmss"
        return formatter.date(from: String(raw.prefix(14)))
    }

    private func sorted(_ entries: [RemoteEntry]) -> [RemoteEntry] {
        entries.sorted {
            if $0.isDirectory != $1.isDirectory { return $0.isDirectory && !$1.isDirectory }
            return $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending
        }
    }
}

private struct FTPResponse: Sendable {
    let code: Int
    let lines: [String]

    var message: String {
        lines.joined(separator: " ").replacingOccurrences(of: "\r", with: " ").replacingOccurrences(of: "\n", with: " ")
    }
}

private enum FTPError: LocalizedError {
    case notConnected
    case malformedResponse(String)
    case unexpected(FTPResponse)

    var isUnsupportedCommand: Bool {
        if case .unexpected(let response) = self {
            return response.code == 500 || response.code == 501 || response.code == 502 || response.code == 504
        }
        return false
    }

    var errorDescription: String? {
        switch self {
        case .notConnected:
            return "Not connected."
        case .malformedResponse(let text):
            return "FTP server returned a malformed response: \(sanitize(text))"
        case .unexpected(let response):
            return "FTP server rejected the operation (\(response.code)): \(sanitize(response.message))"
        }
    }

    private func sanitize(_ value: String) -> String {
        let clean = value.replacingOccurrences(of: "\r", with: " ")
            .replacingOccurrences(of: "\n", with: " ")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return String(clean.prefix(240))
    }
}
