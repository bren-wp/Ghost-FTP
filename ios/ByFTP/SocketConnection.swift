import Foundation
import Network

final class SocketConnection {
    private let connection: NWConnection
    private let queue = DispatchQueue(label: "com.byftp.ios.socket", qos: .userInitiated)
    private var readBuffer = Data()

    init(host: String, port: UInt16, tls: Bool) throws {
        guard let endpointPort = NWEndpoint.Port(rawValue: port) else {
            throw ValidationError("Invalid network port.")
        }
        let parameters: NWParameters
        if tls {
            parameters = NWParameters(tls: NWProtocolTLS.Options(), tcp: NWProtocolTCP.Options())
        } else {
            parameters = .tcp
        }
        parameters.includePeerToPeer = false
        connection = NWConnection(host: NWEndpoint.Host(host), port: endpointPort, using: parameters)
    }

    func start() async throws {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            let lock = NSLock()
            var pending: CheckedContinuation<Void, Error>? = continuation
            connection.stateUpdateHandler = { state in
                lock.lock()
                defer { lock.unlock() }
                guard let current = pending else { return }
                switch state {
                case .ready:
                    pending = nil
                    current.resume()
                case .failed(let error):
                    pending = nil
                    current.resume(throwing: error)
                case .cancelled:
                    pending = nil
                    current.resume(throwing: NetworkError.connectionClosed)
                default:
                    break
                }
            }
            connection.start(queue: queue)
        }
    }

    func sendLine(_ line: String) async throws {
        guard !line.contains("\r"), !line.contains("\n"), !line.contains("\0") else {
            throw ValidationError("Network command contains an unsafe control character.")
        }
        guard let data = (line + "\r\n").data(using: .utf8) else {
            throw NetworkError.encodingFailed
        }
        try await send(data)
    }

    func send(_ data: Data) async throws {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            connection.send(content: data, completion: .contentProcessed { error in
                if let error { continuation.resume(throwing: error) }
                else { continuation.resume() }
            })
        }
    }

    func sendFile(_ url: URL, chunkSize: Int = 64 * 1024) async throws {
        let handle = try FileHandle(forReadingFrom: url)
        defer { try? handle.close() }
        while true {
            let chunk = try handle.read(upToCount: chunkSize) ?? Data()
            if chunk.isEmpty { break }
            try await send(chunk)
        }
    }

    func finishSending() async throws {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            connection.send(content: nil, contentContext: .finalMessage, isComplete: true, completion: .contentProcessed { error in
                if let error { continuation.resume(throwing: error) }
                else { continuation.resume() }
            })
        }
    }

    func readLine(maxBytes: Int = 16 * 1024) async throws -> String {
        while true {
            if let range = readBuffer.range(of: Data([13, 10])) {
                let lineData = readBuffer[..<range.lowerBound]
                readBuffer.removeSubrange(..<range.upperBound)
                guard let line = String(data: lineData, encoding: .utf8) else { throw NetworkError.encodingFailed }
                return line
            }
            guard readBuffer.count < maxBytes else { throw NetworkError.responseTooLarge }
            let (chunk, complete) = try await receiveChunk(maximumLength: min(64 * 1024, maxBytes - readBuffer.count))
            if !chunk.isEmpty { readBuffer.append(chunk) }
            if complete && chunk.isEmpty { throw NetworkError.connectionClosed }
        }
    }

    func receiveAll(maxBytes: Int = 8 * 1024 * 1024) async throws -> Data {
        var result = Data()
        while true {
            guard result.count < maxBytes else { throw NetworkError.responseTooLarge }
            let (chunk, complete) = try await receiveChunk(maximumLength: min(64 * 1024, maxBytes - result.count))
            if !chunk.isEmpty { result.append(chunk) }
            if complete { return result }
        }
    }

    func receiveToFile(_ url: URL, maxBytes: Int64 = 4 * 1024 * 1024 * 1024) async throws {
        FileManager.default.createFile(atPath: url.path, contents: nil)
        let handle = try FileHandle(forWritingTo: url)
        defer { try? handle.close() }
        var total: Int64 = 0
        while true {
            let (chunk, complete) = try await receiveChunk(maximumLength: 64 * 1024)
            if !chunk.isEmpty {
                total += Int64(chunk.count)
                guard total <= maxBytes else { throw NetworkError.responseTooLarge }
                try handle.write(contentsOf: chunk)
            }
            if complete { return }
        }
    }

    func cancel() {
        connection.stateUpdateHandler = nil
        connection.cancel()
        readBuffer.removeAll(keepingCapacity: false)
    }

    private func receiveChunk(maximumLength: Int) async throws -> (Data, Bool) {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<(Data, Bool), Error>) in
            connection.receive(minimumIncompleteLength: 1, maximumLength: max(1, maximumLength)) { data, _, isComplete, error in
                if let error { continuation.resume(throwing: error) }
                else { continuation.resume(returning: (data ?? Data(), isComplete)) }
            }
        }
    }
}

enum NetworkError: LocalizedError {
    case connectionClosed
    case encodingFailed
    case responseTooLarge

    var errorDescription: String? {
        switch self {
        case .connectionClosed: return "The server closed the connection."
        case .encodingFailed: return "The server returned text that could not be decoded."
        case .responseTooLarge: return "The server response exceeded the safety limit."
        }
    }
}
