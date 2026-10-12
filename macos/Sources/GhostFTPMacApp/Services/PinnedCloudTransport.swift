import Darwin
import Foundation
import CryptoKit

/// Public address selection for signed cloud links. Each approved numeric
/// address is supplied to curl's --resolve, which binds its CONNECT to the
/// chosen peer without changing the URL host used for SNI and certificate
/// verification. A separate DNS preflight followed by URLSession would not.
enum PinnedCloudPublicAddress {
    static func allowed(_ bytes: [UInt8]) -> Bool {
        if bytes.count == 4 {
            let a = Int(bytes[0]), b = Int(bytes[1]), c = Int(bytes[2])
            guard a >= 1 && a <= 223 else { return false }
            if a == 10 || a == 127 || (a == 169 && b == 254) ||
                (a == 172 && (16...31).contains(b)) ||
                (a == 192 && b == 168) { return false }
            if a == 100 && (64...127).contains(b) { return false }
            if a == 192 && b == 0 && c == 0 { return false }
            if a == 192 && b == 0 && c == 2 { return false }
            if a == 192 && b == 88 && c == 99 { return false }
            if a == 198 && (18...19).contains(b) { return false }
            if a == 198 && b == 51 && c == 100 { return false }
            if a == 203 && b == 0 && c == 113 { return false }
            return true
        }
        if bytes.count == 16 {
            guard (0x20...0x3f).contains(bytes[0]) else { return false }
            if bytes[0] == 0x20 && bytes[1] == 0x01 &&
                bytes[2] == 0x0d && bytes[3] == 0xb8 { return false }
            if bytes[0] == 0x20 && bytes[1] == 0x02 { return false }
            if bytes[0] == 0x20 && bytes[1] == 0x01 &&
                bytes[2] == 0 && bytes[3] == 0 { return false }
            // IANA: benchmark 2001:2::/48, deprecated 2001:10::/28,
            // and RFC 9637 documentation 3fff::/20 aren't public peers.
            if bytes[0] == 0x20 && bytes[1] == 0x01 &&
                bytes[2] == 0 && bytes[3] == 2 &&
                bytes[4] == 0 && bytes[5] == 0 { return false }
            if bytes[0] == 0x20 && bytes[1] == 0x01 &&
                bytes[2] == 0 && (bytes[3] & 0xf0) == 0x10 { return false }
            if bytes[0] == 0x3f && bytes[1] == 0xff &&
                (bytes[2] & 0xf0) == 0 { return false }
            return true
        }
        return false
    }

    static func lookup(_ host: String) throws -> [String] {
        guard SignedCloudLink.isPublicHost(host),
              SignedCloudLink.isKnownProviderHost(host) else {
            throw SignedCloudLinkError.privateEndpoint
        }
        var hints = addrinfo()
        hints.ai_family = AF_UNSPEC
        hints.ai_socktype = SOCK_STREAM
        hints.ai_protocol = IPPROTO_TCP
        var result: UnsafeMutablePointer<addrinfo>?
        guard getaddrinfo(host, "443", &hints, &result) == 0, let first = result else {
            throw SignedCloudLinkError.privateEndpoint
        }
        defer { freeaddrinfo(first) }
        var addresses: [String] = []
        var current: UnsafeMutablePointer<addrinfo>? = first
        while let pointer = current {
            let info = pointer.pointee
            guard let address = info.ai_addr else {
                throw SignedCloudLinkError.privateEndpoint
            }
            let bytes: [UInt8]
            if info.ai_family == AF_INET {
                let sin = UnsafeRawPointer(address).assumingMemoryBound(to: sockaddr_in.self)
                bytes = withUnsafeBytes(of: sin.pointee.sin_addr) { Array($0) }
            } else if info.ai_family == AF_INET6 {
                let sin = UnsafeRawPointer(address).assumingMemoryBound(to: sockaddr_in6.self)
                bytes = withUnsafeBytes(of: sin.pointee.sin6_addr) { Array($0) }
            } else {
                throw SignedCloudLinkError.privateEndpoint
            }
            // Do not fall back to a "good" address when the same hostname
            // also has loopback/metadata/private answers.
            guard allowed(bytes) else { throw SignedCloudLinkError.privateEndpoint }
            var buffer = [CChar](repeating: 0, count: Int(NI_MAXHOST))
            let numeric = buffer.withUnsafeMutableBufferPointer { chars in
                getnameinfo(address, info.ai_addrlen, chars.baseAddress,
                            socklen_t(chars.count), nil, 0, NI_NUMERICHOST)
            }
            guard numeric == 0 else { throw SignedCloudLinkError.privateEndpoint }
            let text = String(cString: buffer)
            let value = info.ai_family == AF_INET6 ? "[\(text)]" : text
            if !addresses.contains(value) { addresses.append(value) }
            current = info.ai_next
        }
        guard !addresses.isEmpty else { throw SignedCloudLinkError.privateEndpoint }
        return addresses
    }
}

/// Shares cancellation state between SwiftUI's parent Task and a detached
/// background process waiter without letting curl block the main actor.
final class PinnedTransferCancellation: @unchecked Sendable {
    private let lock = NSLock()
    private var cancelled = false

    /// Serializes cancellation with Process.run so a cancellation delivered
    /// immediately before launch cannot leave an untracked curl process alive.
    func start(_ process: Process) throws {
        lock.lock()
        defer { lock.unlock() }
        if cancelled { throw CancellationError() }
        try process.run()
    }

    func cancel(_ process: Process) {
        lock.lock()
        cancelled = true
        if process.isRunning {
            process.terminate()
        }
        lock.unlock()
    }

    func ensureActive() throws {
        lock.lock()
        let stopped = cancelled
        lock.unlock()
        if stopped { throw CancellationError() }
    }
}

/// Fail-closed signed-object HTTPS transport for macOS. No caller-provided
/// URL is placed in process arguments, logs, environment or on disk. Apple's
/// /usr/bin/curl receives an escaped config through its private stdin pipe.
/// -q suppresses ~/.curlrc; --proxy "" overrides proxy environment; no
/// --location, --insecure or custom certificate verifier is ever used.
enum PinnedCloudTransport {
    /// Curl can emit interim HTTP 100 Continue headers followed by the actual
    /// provider response. Inspect only the last complete HTTP header block;
    /// duplicate security headers in that response are rejected.
    static func parseFinalHeaders(_ raw: String) throws -> [String: String] {
        let normalized = raw.replacingOccurrences(of: "\r\n", with: "\n")
        let blocks = normalized.components(separatedBy: "\n\n")
        guard let last = blocks.last(where: { $0.hasPrefix("HTTP/") }) else {
            throw SignedCloudLinkError.invalidResponse
        }
        var headers: [String: String] = [:]
        for line in last.components(separatedBy: "\n").dropFirst() {
            guard let colon = line.firstIndex(of: ":") else { continue }
            let name = String(line[..<colon]).lowercased()
            let value = line[line.index(after: colon)...].trimmingCharacters(in: .whitespaces)
            if headers[name] != nil { throw SignedCloudLinkError.invalidResponse }
            headers[name] = value
        }
        return headers
    }

    static func quoted(_ text: String) throws -> String {
        guard !text.unicodeScalars.contains(where: { $0.value < 0x20 || $0.value == 0x7f }) else {
            throw SignedCloudLinkError.invalidLink
        }
        return "\"" + text.replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "\"", with: "\\\"") + "\""
    }

    static func transfer(
        from link: SignedCloudLink,
        localFile: URL,
        uploading: Bool
    ) async throws -> (status: Int, headers: [String: String]) {
        let addresses = try PinnedCloudPublicAddress.lookup(link.host)
        try Task.checkCancellation()
        let directory = FileManager.default.temporaryDirectory
            .appendingPathComponent("ghostftp-pinned-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(
            at: directory, withIntermediateDirectories: false,
            attributes: [.posixPermissions: NSNumber(value: 0o700)]
        )
        defer { try? FileManager.default.removeItem(at: directory) }

        let headersFile = directory.appendingPathComponent("headers")
        let outputFile = directory.appendingPathComponent("response")
        var arguments = [
            "-q", "--silent", "--fail", "--proto", "=https",
            "--proxy", "", "--noproxy", "*", "--max-redirs", "0",
            "--connect-timeout", "20", "--max-time", "3600",
            "--http1.1", "--dump-header", headersFile.path,
            "--output", outputFile.path, "--write-out", "%{http_code}",
            "--config", "-"
        ]
        for address in addresses {
            arguments += ["--resolve", "\(link.host):443:\(address)"]
        }
        if uploading {
            arguments += ["--upload-file", localFile.path,
                          "--header", "Content-Type: application/octet-stream"]
            if link.host.hasSuffix(".blob.core.windows.net") {
                arguments += ["--header", "x-ms-blob-type: BlockBlob"]
            }
        } else {
            arguments += ["--header", "Accept-Encoding: identity",
                          "--max-filesize", "1073741824"]
        }
        // Config is private to stdin and only supplies the URL. The escaping
        // disallows curlrc directives, newlines and arbitrary shell syntax.
        let config = "url = \(try quoted(link.url.absoluteString))\n"
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/curl")
        process.arguments = arguments
        let input = Pipe(), output = Pipe()
        process.standardInput = input
        process.standardOutput = output
        process.standardError = FileHandle.nullDevice

        let cancellation = PinnedTransferCancellation()
        return try await withTaskCancellationHandler(operation: {
            let result = try await Task.detached(priority: .userInitiated) { () throws -> (status: Int, headers: [String: String]) in
            try cancellation.start(process)
            try cancellation.ensureActive()
            do {
                try input.fileHandleForWriting.write(contentsOf: Data(config.utf8))
                try input.fileHandleForWriting.close()
            } catch {
                if process.isRunning { process.terminate() }
                process.waitUntilExit()
                try cancellation.ensureActive()
                throw SignedCloudLinkError.invalidResponse
            }
            // Curl only sends a 3-digit status to stdout, never the signed URL.
            let responseText = String(
                decoding: output.fileHandleForReading.readDataToEndOfFile(),
                as: UTF8.self
            ).trimmingCharacters(in: .whitespacesAndNewlines)
            process.waitUntilExit()
            try cancellation.ensureActive()
            guard process.terminationStatus == 0,
                  let status = Int(responseText),
                  uploading ? (200...299).contains(status) : status == 200 else {
                throw SignedCloudLinkError.invalidResponse
            }
            guard let headerSize = try? FileManager.default.attributesOfItem(
                atPath: headersFile.path
            )[.size] as? NSNumber, headerSize.intValue <= 65536 else {
                throw SignedCloudLinkError.invalidResponse
            }
            let data = try Data(contentsOf: headersFile)
            guard let headerText = String(data: data, encoding: .utf8) else {
                throw SignedCloudLinkError.invalidResponse
            }
            let headers = try parseFinalHeaders(headerText)
            if !uploading {
                let manager = FileManager.default
                guard let size = try? manager.attributesOfItem(atPath: outputFile.path)[.size] as? NSNumber,
                      size.int64Value <= 1_073_741_824 else {
                    throw SignedCloudLinkError.invalidFile
                }
                if let length = headers["content-length"] {
                    guard let declared = Int64(length), declared == size.int64Value else {
                        throw SignedCloudLinkError.invalidResponse
                    }
                }
                if let encoding = headers["content-encoding"],
                   encoding.lowercased() != "identity" {
                    throw SignedCloudLinkError.invalidResponse
                }
                if let checksum = headers["x-amz-checksum-sha256"] {
                    let handle = try FileHandle(forReadingFrom: outputFile)
                    defer { try? handle.close() }
                    var hasher = SHA256()
                    while true {
                        try cancellation.ensureActive()
                        let bytes = try handle.read(upToCount: 64 * 1024) ?? Data()
                        if bytes.isEmpty { break }
                        hasher.update(data: bytes)
                    }
                    try SignedCloudTransfer.verifySHA256(
                        header: checksum, digest: Data(hasher.finalize())
                    )
                }
                try cancellation.ensureActive()
                // Preserve the destination until response and checksum are
                // accepted. Caller owns the user-selected path; this is an
                // ordinary local file, not a cloud account token.
                let staged = localFile.deletingLastPathComponent()
                    .appendingPathComponent(".ghostftp-\(UUID().uuidString).part")
                do {
                    try manager.copyItem(at: outputFile, to: staged)
                    try cancellation.ensureActive()
                    if manager.fileExists(atPath: localFile.path) {
                        _ = try manager.replaceItemAt(localFile, withItemAt: staged)
                    } else {
                        try manager.moveItem(at: staged, to: localFile)
                    }
                } catch is CancellationError {
                    try? manager.removeItem(at: staged)
                    throw CancellationError()
                } catch {
                    try? manager.removeItem(at: staged)
                    throw SignedCloudLinkError.invalidFile
                }
            }
            return (status, headers)
            }.value
            try Task.checkCancellation()
            return result
        }, onCancel: {
            cancellation.cancel(process)
        })
    }
}
