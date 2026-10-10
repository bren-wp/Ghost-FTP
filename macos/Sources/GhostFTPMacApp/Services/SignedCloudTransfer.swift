import Foundation

/// Transfers one object through an HTTPS URL signed by the user's own cloud provider.
/// The signed URL is a short-lived credential: never persist, log, or include it in an error.
enum SignedCloudLinkError: LocalizedError {
    case invalidLink
    case privateEndpoint
    case failedResponse(Int)
    case invalidResponse
    case invalidFile

    var errorDescription: String? {
        switch self {
        case .invalidLink:
            return "Paste an HTTPS object URL signed by S3 or Google Cloud Storage (or an Azure SAS URL)."
        case .privateEndpoint:
            return "Local and private network addresses are not allowed for signed cloud transfers."
        case .failedResponse(let status):
            return "Cloud provider rejected the request (HTTP \(status)). Check link expiry and permissions."
        case .invalidResponse:
            return "The cloud provider returned an invalid response."
        case .invalidFile:
            return "The selected file could not be read or saved."
        }
    }
}

struct SignedCloudLink {
    let url: URL
    let host: String

    init(_ text: String) throws {
        guard text.utf8.count <= 8192,
              let components = URLComponents(string: text),
              components.scheme?.lowercased() == "https",
              let hostname = components.host?.lowercased(), !hostname.isEmpty,
              components.user == nil, components.password == nil,
              components.fragment == nil,
              components.port == nil || components.port == 443,
              let queryNames = components.queryItems?.map({ $0.name.lowercased() }),
              queryNames.contains(where: { ["x-amz-signature", "x-goog-signature", "sig"].contains($0) }),
              let validURL = components.url else {
            throw SignedCloudLinkError.invalidLink
        }
        guard Self.isPublicHost(hostname) else {
            throw SignedCloudLinkError.privateEndpoint
        }
        url = validURL
        host = hostname
    }

    static func isPublicHost(_ host: String) -> Bool {
        if host == "localhost" || host.hasSuffix(".localhost") ||
            host.hasSuffix(".local") || host.hasSuffix(".internal") ||
            host == "::1" || host.hasPrefix("fe80:") ||
            host.hasPrefix("fc") || host.hasPrefix("fd") {
            return false
        }
        // Signed public cloud links use DNS hosts. Reject literal IPs to reduce
        // accidental uploads to local services and bypasses via IP notation.
        if host.contains(":") || host.allSatisfy({ $0.isNumber || $0 == "." }) {
            return false
        }
        return host.contains(".")
    }
}

private final class SignedCloudNoRedirects: NSObject, URLSessionTaskDelegate {
    func urlSession(_ session: URLSession, task: URLSessionTask,
                    willPerformHTTPRedirection response: HTTPURLResponse,
                    newRequest request: URLRequest,
                    completionHandler: @escaping (URLRequest?) -> Void) {
        completionHandler(nil)
    }
}

enum SignedCloudTransfer {
    private static func session() -> URLSession {
        let config = URLSessionConfiguration.ephemeral
        config.urlCache = nil
        config.httpCookieStorage = nil
        config.httpShouldSetCookies = false
        config.timeoutIntervalForRequest = 120
        config.timeoutIntervalForResource = 3600
        return URLSession(configuration: config, delegate: SignedCloudNoRedirects(), delegateQueue: nil)
    }

    static func upload(file: URL, to link: SignedCloudLink) async throws {
        guard FileManager.default.isReadableFile(atPath: file.path),
              let size = try? FileManager.default.attributesOfItem(atPath: file.path)[.size] as? NSNumber,
              size.int64Value <= 1024 * 1024 * 1024 else {
            throw SignedCloudLinkError.invalidFile
        }
        var request = URLRequest(url: link.url)
        request.httpMethod = "PUT"
        let client = session()
        defer { client.finishTasksAndInvalidate() }
        let (_, response) = try await client.upload(for: request, fromFile: file)
        try verify(response, isDownload: false)
    }

    static func download(from link: SignedCloudLink, to destination: URL) async throws {
        var request = URLRequest(url: link.url)
        request.httpMethod = "GET"
        let client = session()
        defer { client.finishTasksAndInvalidate() }
        let (temporary, response) = try await client.download(for: request)
        try verify(response, isDownload: true)
        let manager = FileManager.default
        guard let size = try? manager.attributesOfItem(atPath: temporary.path)[.size] as? NSNumber,
              size.int64Value <= 1024 * 1024 * 1024 else {
            throw SignedCloudLinkError.invalidFile
        }
        let staged = destination.deletingLastPathComponent()
            .appendingPathComponent(".ghostftp-cloud-\(UUID().uuidString).part")
        do {
            try manager.moveItem(at: temporary, to: staged)
            if manager.fileExists(atPath: destination.path) {
                _ = try manager.replaceItemAt(destination, withItemAt: staged)
            } else {
                try manager.moveItem(at: staged, to: destination)
            }
        } catch {
            try? manager.removeItem(at: staged)
            throw SignedCloudLinkError.invalidFile
        }
    }

    private static func verify(_ response: URLResponse, isDownload: Bool) throws {
        guard let http = response as? HTTPURLResponse else {
            throw SignedCloudLinkError.invalidResponse
        }
        // A GET without a Range request must not silently install a partial
        // object returned as HTTP 206 in place of the complete destination.
        guard isDownload ? http.statusCode == 200 : (200...299).contains(http.statusCode) else {
            throw SignedCloudLinkError.failedResponse(http.statusCode)
        }
    }
}
