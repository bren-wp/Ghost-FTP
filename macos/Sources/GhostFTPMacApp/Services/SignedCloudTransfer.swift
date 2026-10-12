import Foundation
import CryptoKit

/// Transfers one object through an HTTPS URL signed by the user's own cloud provider.
/// The signed URL is a short-lived credential: never persist, log, or include it in an error.
enum SignedCloudLinkError: LocalizedError {
    case invalidLink
    case privateEndpoint
    case unsupportedProvider
    case failedResponse(Int)
    case invalidResponse
    case invalidFile
    case invalidChecksum

    var errorDescription: String? {
        switch self {
        case .invalidLink:
            return "Paste an HTTPS object URL signed by S3 or Google Cloud Storage (or an Azure SAS URL)."
        case .privateEndpoint:
            return "Local and private network addresses are not allowed for signed cloud transfers."
        case .unsupportedProvider:
            return "Use a signed URL from a supported cloud storage provider. Custom S3 endpoints require a separately trusted connection."
        case .failedResponse(let status):
            return "Cloud provider rejected the request (HTTP \(status)). Check link expiry and permissions."
        case .invalidResponse:
            return "The cloud provider returned an invalid response."
        case .invalidFile:
            return "The selected file could not be read or saved."
        case .invalidChecksum:
            return "Cloud object checksum verification failed; the destination was not changed."
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
              let validURL = components.url else {
            throw SignedCloudLinkError.invalidLink
        }
        guard Self.isPublicHost(hostname) else {
            throw SignedCloudLinkError.privateEndpoint
        }
        guard Self.isKnownProviderHost(hostname) else {
            throw SignedCloudLinkError.unsupportedProvider
        }
        // Match precisely one nonempty signature field to its cloud provider.
        let signatureKey: String
        if hostname.hasSuffix(".blob.core.windows.net") {
            signatureKey = "sig"
        } else if hostname == "storage.googleapis.com" ||
                    hostname.hasSuffix(".storage.googleapis.com") {
            signatureKey = "x-goog-signature"
        } else {
            signatureKey = "x-amz-signature"
        }
        let signatures = components.queryItems?.filter {
            $0.name.lowercased() == signatureKey
        } ?? []
        guard signatures.count == 1, let signature = signatures.first?.value,
              !signature.isEmpty else {
            throw SignedCloudLinkError.invalidLink
        }
        url = validURL
        host = hostname
    }

    /// A fake signature query on an arbitrary HTTPS host is not sufficient
    /// authorization to send a user's file. Start with known public provider
    /// domains; custom S3 endpoints require an explicit trust flow in the future.
    static func isKnownProviderHost(_ host: String) -> Bool {
        if host == "storage.googleapis.com" ||
            host.hasSuffix(".storage.googleapis.com") ||
            host.hasSuffix(".blob.core.windows.net") ||
            host.hasSuffix(".r2.cloudflarestorage.com") ||
            host.hasSuffix(".digitaloceanspaces.com") {
            return true
        }
        if host.hasSuffix(".amazonaws.com") ||
            host.hasSuffix(".backblazeb2.com") ||
            host.hasSuffix(".wasabisys.com") ||
            host.hasSuffix(".scw.cloud") {
            return host.split(separator: ".").contains {
                $0 == "s3" || $0.hasPrefix("s3-")
            }
        }
        return false
    }

    static func isPublicHost(_ host: String) -> Bool {
        // Restrict signed provider hostnames to unambiguous ASCII DNS labels.
        // IDNA/punycode or unusual Unicode labels must not expand the
        // provider suffix allowlist through inconsistent normalization.
        let bytes = Array(host.utf8)
        guard (4...253).contains(bytes.count),
              bytes.allSatisfy({ byte in
                  (48...57).contains(byte) || (97...122).contains(byte) ||
                  byte == 45 || byte == 46
              }) else { return false }
        let labels = host.split(separator: ".", omittingEmptySubsequences: false)
        guard labels.count >= 2,
              labels.allSatisfy({ label in
                  !label.isEmpty && label.count <= 63 &&
                  label.first != "-" && label.last != "-" &&
                  !label.hasPrefix("xn--")
              }) else { return false }
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

/// Signed object transfers use only peers bound to approved public IPs.
/// The transport preserves the original hostname for OS certificate checking
/// and keeps signed URLs out of process arguments and filesystem config.
enum SignedCloudTransfer {
    static func upload(file: URL, to link: SignedCloudLink) async throws {
        guard FileManager.default.isReadableFile(atPath: file.path),
              let size = try? FileManager.default.attributesOfItem(atPath: file.path)[.size] as? NSNumber,
              size.int64Value <= 1_073_741_824 else {
            throw SignedCloudLinkError.invalidFile
        }
        _ = try await PinnedCloudTransport.transfer(
            from: link, localFile: file, uploading: true
        )
    }

    static func download(from link: SignedCloudLink, to destination: URL) async throws {
        _ = try await PinnedCloudTransport.transfer(
            from: link, localFile: destination, uploading: false
        )
    }

    /// The S3 checksum header is optional. Never claim ETag is a checksum:
    /// multipart objects use ETags that are not the SHA-256 of the content.
    /// Invalid/mismatched headers fail before replacing an existing file.
    static func verifySHA256(header: String?, digest: Data) throws {
        guard let header else { return }
        guard let expected = Data(base64Encoded: header.trimmingCharacters(in: .whitespacesAndNewlines)),
              expected.count == 32, digest.count == 32 else {
            throw SignedCloudLinkError.invalidChecksum
        }
        var mismatch: UInt8 = 0
        for index in 0..<32 {
            mismatch |= expected[index] ^ digest[index]
        }
        guard mismatch == 0 else { throw SignedCloudLinkError.invalidChecksum }
    }

}
