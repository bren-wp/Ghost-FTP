import Foundation
import CryptoKit
import Security

/// Google Drive public-client authorization request primitives.
/// No token is exchanged or persisted here; production sign-in additionally
/// requires a registered redirect, Keychain vault and authorized Drive API.
/// The short-lived verifier and state live only in this object.
final class CloudOAuthPKCEAttempt {
    enum AuthError: Error {
        case invalidClient
        case invalidRedirect
        case unavailableEntropy
        case unexpectedCallback
        case duplicateParameters
        case invalidState
        case authorizationDeclined
        case missingCode
        case replay
    }

    let authorizationURL: URL
    let redirectURL: URL
    let verifier: String
    private let state: String
    private var consumed = false
    private let lock = NSLock()

    private static func nonce() throws -> String {
        var bytes = [UInt8](repeating: 0, count: 32)
        guard SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes) == errSecSuccess else {
            throw AuthError.unavailableEntropy
        }
        return Data(bytes).base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }

    init(googleClientID: String, registeredRedirect: String) throws {
        let idPattern = #"^[0-9]+-[a-z0-9-]+\.apps\.googleusercontent\.com$"#
        guard googleClientID.range(of: idPattern, options: .regularExpression) != nil else {
            throw AuthError.invalidClient
        }
        guard let redirect = URL(string: registeredRedirect),
              let scheme = redirect.scheme, !scheme.isEmpty,
              !redirect.path.isEmpty,
              URLComponents(url: redirect, resolvingAgainstBaseURL: false)?.queryItems == nil,
              redirect.fragment == nil,
              redirect.user == nil,
              redirect.password == nil else {
            throw AuthError.invalidRedirect
        }

        let verifier = try Self.nonce()
        let state = try Self.nonce()
        let digest = SHA256.hash(data: Data(verifier.utf8))
        let challenge = Data(digest).base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
        var url = URLComponents()
        url.scheme = "https"
        url.host = "accounts.google.com"
        url.path = "/o/oauth2/v2/auth"
        url.queryItems = [
            URLQueryItem(name: "client_id", value: googleClientID),
            URLQueryItem(name: "redirect_uri", value: redirect.absoluteString),
            URLQueryItem(name: "response_type", value: "code"),
            URLQueryItem(name: "scope", value: "https://www.googleapis.com/auth/drive.file"),
            URLQueryItem(name: "state", value: state),
            URLQueryItem(name: "code_challenge", value: challenge),
            URLQueryItem(name: "code_challenge_method", value: "S256"),
            URLQueryItem(name: "access_type", value: "offline")
        ]
        guard let result = url.url else { throw AuthError.invalidRedirect }
        authorizationURL = result
        redirectURL = redirect
        self.verifier = verifier
        self.state = state
    }

    /// Exactly one callback may consume this attempt, even if that callback
    /// is malicious or malformed. The code is never logged or stored.
    func consume(callback: URL) throws -> String {
        lock.lock()
        defer { lock.unlock() }
        guard !consumed else { throw AuthError.replay }
        consumed = true

        guard callback.scheme == redirectURL.scheme,
              callback.host == redirectURL.host,
              callback.port == redirectURL.port,
              callback.path == redirectURL.path,
              callback.fragment == nil,
              callback.user == nil,
              callback.password == nil,
              let parts = URLComponents(url: callback, resolvingAgainstBaseURL: false)?.queryItems else {
            throw AuthError.unexpectedCallback
        }
        let keys = parts.map(\.name)
        guard Set(keys).count == keys.count else { throw AuthError.duplicateParameters }
        let fields = Dictionary(uniqueKeysWithValues: parts.map { ($0.name, $0.value ?? "") })
        guard let supplied = fields["state"],
              Data(supplied.utf8).count == Data(state.utf8).count else {
            throw AuthError.invalidState
        }
        var mismatch: UInt8 = 0
        for (a, b) in zip(supplied.utf8, state.utf8) { mismatch |= a ^ b }
        guard mismatch == 0 else { throw AuthError.invalidState }
        guard fields["error"] == nil else { throw AuthError.authorizationDeclined }
        guard let code = fields["code"], !code.isEmpty, code.count <= 4096 else {
            throw AuthError.missingCode
        }
        return code
    }
}
