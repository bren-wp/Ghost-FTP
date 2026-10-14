import XCTest
import Foundation
import CryptoKit
@testable import GhostFTPMacApp

final class CloudOAuthPKCETests: XCTestCase {
    private let client = "123456789-example.apps.googleusercontent.com"
    private let redirect = "com.googleusercontent.apps.123456789-example:/oauth2redirect"

    func testPKCEChallengeAndLeastPrivilegeRequest() throws {
        let first = try CloudOAuthPKCEAttempt(googleClientID: client, registeredRedirect: redirect)
        let second = try CloudOAuthPKCEAttempt(googleClientID: client, registeredRedirect: redirect)
        let url = try XCTUnwrap(URLComponents(url: first.authorizationURL, resolvingAgainstBaseURL: false))
        XCTAssertEqual(url.scheme, "https")
        XCTAssertEqual(url.host, "accounts.google.com")
        let args = Dictionary(uniqueKeysWithValues: try XCTUnwrap(url.queryItems).map { ($0.name, $0.value ?? "") })
        XCTAssertEqual(args["scope"], "https://www.googleapis.com/auth/drive.file")
        XCTAssertEqual(args["code_challenge_method"], "S256")
        XCTAssertEqual(args["response_type"], "code")
        XCTAssertNil(args["client_secret"])
        XCTAssertFalse(first.authorizationURL.absoluteString.contains(first.verifier))
        XCTAssertNotEqual(first.verifier, second.verifier)
        XCTAssertNotEqual(args["state"], nil)
        let digest = SHA256.hash(data: Data(first.verifier.utf8))
        let expectedChallenge = Data(digest).base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
        XCTAssertEqual(args["code_challenge"], expectedChallenge)
    }

    func testStateValidationExactRedirectAndSingleUseCallback() throws {
        let attempt = try CloudOAuthPKCEAttempt(googleClientID: client, registeredRedirect: redirect)
        let query = try XCTUnwrap(URLComponents(url: attempt.authorizationURL, resolvingAgainstBaseURL: false))
        let state = try XCTUnwrap(query.queryItems?.first(where: { $0.name == "state" })?.value)
        let callback = try XCTUnwrap(URL(string: redirect + "?code=oneTimeCode&state=" + state))
        XCTAssertEqual(try attempt.consume(callback: callback), "oneTimeCode")
        XCTAssertThrowsError(try attempt.consume(callback: callback))
    }

    func testDuplicateStateAndWrongRedirectAreRejected() throws {
        let attempt = try CloudOAuthPKCEAttempt(googleClientID: client, registeredRedirect: redirect)
        let callback = try XCTUnwrap(URL(string: redirect + "?code=abc&state=bad&state=bad"))
        XCTAssertThrowsError(try attempt.consume(callback: callback))
        let other = try CloudOAuthPKCEAttempt(googleClientID: client, registeredRedirect: redirect)
        let wrong = try XCTUnwrap(URL(string: "com.attacker.example:/oauth2redirect?code=x&state=bad"))
        XCTAssertThrowsError(try other.consume(callback: wrong))
    }

    func testMalformedClientAndRedirectAreRejected() {
        XCTAssertThrowsError(try CloudOAuthPKCEAttempt(googleClientID: "not-registered", registeredRedirect: redirect))
        XCTAssertThrowsError(try CloudOAuthPKCEAttempt(googleClientID: client, registeredRedirect: "https://example.com/path?token=secret"))
    }
}
