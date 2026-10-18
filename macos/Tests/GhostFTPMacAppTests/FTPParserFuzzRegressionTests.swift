import Foundation
import XCTest
@testable import GhostFTPMacApp

/// Deterministic hostile-server corpus: a bounded regression budget, not a
/// substitute for libFuzzer, TLS/SSH handshake tests, or third-party servers.
final class FTPParserFuzzRegressionTests: XCTestCase {
    private struct Generator {
        var state: UInt64 = 0x4D41432D46545031

        mutating func byte() -> UInt8 {
            state = state &* 6364136223846793005 &+ 1442695040888963407
            return UInt8(truncatingIfNeeded: state >> 32)
        }
    }

    func testMalformedControlReplyUTF8FailsClosed() {
        var firstLine = Data([0x32, 0x32, 0x30, 0x20, 0xff, 0x0d, 0x0a])
        XCTAssertThrowsError(try FTPControlCodec.takeReply(from: &firstLine))

        var continuation = Data("211-Features\r\n".utf8)
        continuation.append(contentsOf: [0xff, 0xfe, 0x0d, 0x0a])
        continuation.append(Data("211 End\r\n".utf8))
        XCTAssertThrowsError(try FTPControlCodec.takeReply(from: &continuation))

        var valid = Data("220 Dostupno — računi\r\n".utf8)
        XCTAssertEqual(try? FTPControlCodec.takeReply(from: &valid)?.code, 220)
    }

    func testAlreadyOversizedBuffersNeverUnderflowOrTrap() {
        var control = Data(repeating: 65, count: FTPControlCodec.maximumControlReplyBytes + 1)
        XCTAssertThrowsError(try FTPControlCodec.appendControlChunk(Data(), to: &control))
        XCTAssertThrowsError(try FTPControlCodec.takeReply(from: &control))

        var listing = Data(repeating: 65, count: 9)
        XCTAssertThrowsError(try FTPControlCodec.appendListingChunk(Data(), to: &listing, maximumBytes: 8))
        XCTAssertThrowsError(try FTPControlCodec.appendListingChunk(Data([66]), to: &listing, maximumBytes: 8))
    }

    func testHostileFTPReplyAndListingCorpusIsCrashFree() {
        var generator = Generator()
        for i in 0..<4_096 {
            let length = i % 87
            var bytes = [UInt8]()
            bytes.reserveCapacity(length + 2)
            for _ in 0..<length {
                bytes.append(generator.byte())
            }
            if i % 3 == 0 {
                bytes.append(13)
                bytes.append(10)
            }
            var control = Data(bytes)
            do {
                _ = try FTPControlCodec.takeReply(from: &control)
            } catch is FTPControlError {
                // Correctly bounded rejection of untrusted control bytes.
            } catch {
                XCTFail("Unexpected control parser error at corpus case \(i)")
            }

            let inventory = Data(bytes)
            do {
                _ = try FTPControlCodec.parseMLSD(inventory)
            } catch is FTPControlError {
                // Invalid inventory must never be accepted partially.
            } catch {
                XCTFail("Unexpected MLSD parser error at corpus case \(i)")
            }
        }
    }

    func testFTPSAndSFTPStillFailBeforeCredentialsOrNetwork() async {
        for kind in [ConnectionProtocol.ftps, .sftp] {
            let session = FTPControlSession()
            let site = ConnectionProfile(
                name: "Unsupported secure transport",
                protocolKind: kind,
                host: "127.0.0.1",
                username: "test"
            )
            do {
                _ = try await session.connect(profile: site, password: "never-transmit")
                XCTFail("A secure profile must not be silently downgraded to FTP")
            } catch let error as FTPControlError {
                switch error {
                case .unsupportedProtocol:
                    break
                default:
                    XCTFail("Unexpected secure protocol error: \(error)")
                }
            } catch {
                XCTFail("Unexpected secure protocol error: \(error)")
            }
        }
    }
}
