import XCTest
@testable import GhostFTPMacApp

final class ConnectionValidatorTests: XCTestCase {
    func testDefaultPortsMatchProtocols() {
        XCTAssertEqual(ConnectionProtocol.ftp.defaultPort, 21)
        XCTAssertEqual(ConnectionProtocol.ftps.defaultPort, 21)
        XCTAssertEqual(ConnectionProtocol.sftp.defaultPort, 22)
    }

    func testValidProfilePassesValidation() {
        let profile = ConnectionProfile(
            name: "Production",
            protocolKind: .sftp,
            host: "sftp.example.com",
            username: "deploy"
        )

        XCTAssertNoThrow(try ConnectionValidator.validate(profile))
    }

    func testBlankHostIsRejected() {
        let profile = ConnectionProfile(
            name: "Production",
            host: "   ",
            username: "deploy"
        )

        XCTAssertThrowsError(try ConnectionValidator.validate(profile)) { error in
            XCTAssertEqual(error as? ConnectionValidationError, .missingHost)
        }
    }

    func testWhitespaceInsideHostIsRejected() {
        let profile = ConnectionProfile(
            name: "Production",
            host: "bad host.example",
            username: "deploy"
        )

        XCTAssertThrowsError(try ConnectionValidator.validate(profile)) { error in
            XCTAssertEqual(error as? ConnectionValidationError, .invalidHost)
        }
    }

    func testBlankUsernameIsRejected() {
        let profile = ConnectionProfile(
            name: "Production",
            host: "server.example",
            username: ""
        )

        XCTAssertThrowsError(try ConnectionValidator.validate(profile)) { error in
            XCTAssertEqual(error as? ConnectionValidationError, .missingUsername)
        }
    }

    func testPersistedProfileHasNoPasswordField() throws {
        let profile = ConnectionProfile(
            name: "Production",
            host: "server.example",
            username: "deploy"
        )

        let json = String(decoding: try JSONEncoder().encode(profile), as: UTF8.self)
        XCTAssertFalse(json.localizedCaseInsensitiveContains("password"))
    }
}
