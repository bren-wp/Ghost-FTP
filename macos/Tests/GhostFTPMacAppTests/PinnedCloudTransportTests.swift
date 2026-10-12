import Foundation
import XCTest
@testable import GhostFTPMacApp

final class PinnedCloudTransportTests: XCTestCase {
    func testOnlyGlobalPublicIPv4CanBecomePeerCandidates() {
        XCTAssertTrue(PinnedCloudPublicAddress.allowed([1, 1, 1, 1]))
        XCTAssertTrue(PinnedCloudPublicAddress.allowed([8, 8, 8, 8]))
        let blockedIPv4: [[UInt8]] = [
            [0, 0, 0, 0], [10, 0, 0, 1], [100, 64, 0, 1],
            [127, 0, 0, 1], [169, 254, 169, 254], [172, 20, 1, 2],
            [192, 168, 0, 1], [192, 0, 2, 1], [198, 18, 0, 1],
            [198, 51, 100, 2], [203, 0, 113, 2], [224, 0, 0, 1]
        ]
        for address in blockedIPv4 {
            XCTAssertFalse(PinnedCloudPublicAddress.allowed(address), "\(address)")
        }
    }

    func testOnlyReviewedGlobalUnicastIPv6CanBecomePeerCandidates() {
        XCTAssertTrue(PinnedCloudPublicAddress.allowed(
            [0x26, 0x06, 0x47, 0x00] + Array(repeating: 0, count: 12)
        ))
        let blockedIPv6Prefixes: [[UInt8]] = [
            [0, 0, 0, 0], [0xfc, 0, 0, 0], [0xfe, 0x80, 0, 0],
            [0x20, 0x01, 0x0d, 0xb8], [0x20, 0x02, 0, 0],
            [0x20, 0x01, 0, 0],
            [0x20, 0x01, 0, 2], // IANA benchmarking /48
            [0x20, 0x01, 0, 0x10], // deprecated ORCHID /28
            [0x3f, 0xff, 0, 0] // documentation /20 (RFC 9637)
        ]
        for prefix in blockedIPv6Prefixes {
            XCTAssertFalse(PinnedCloudPublicAddress.allowed(
                prefix + Array(repeating: 0, count: 12)
            ))
        }
    }

    func testCurlConfigEscapesBearerURLAndRejectsNewlines() throws {
        let ordinary = try PinnedCloudTransport.quoted("https://bucket.s3.amazonaws.com/a?X-Amz-Signature=a%2Bb")
        XCTAssertEqual(ordinary, "\"https://bucket.s3.amazonaws.com/a?X-Amz-Signature=a%2Bb\"")
        XCTAssertEqual(try PinnedCloudTransport.quoted("alpha\"beta"), "\"alpha\\\"beta\"")
        XCTAssertThrowsError(try PinnedCloudTransport.quoted("https://bucket.s3.amazonaws.com/\nproxy = evil"))
        XCTAssertThrowsError(try PinnedCloudTransport.quoted("a\rb"))
    }
    func testParsesLastHttpHeadersWithoutConfusing100Continue() throws {
        let raw = "HTTP/1.1 100 Continue\r\nServer: intermediary\r\n\r\n" +
            "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n" +
            "x-amz-checksum-sha256: checksum\r\n\r\n"
        let parsed = try PinnedCloudTransport.parseFinalHeaders(raw)
        XCTAssertEqual(parsed["content-length"], "5")
        XCTAssertEqual(parsed["x-amz-checksum-sha256"], "checksum")
        XCTAssertNil(parsed["server"])
    }

    func testRejectsDuplicateProviderHeadersAndMissingResponseBlock() {
        XCTAssertThrowsError(try PinnedCloudTransport.parseFinalHeaders(
            "HTTP/1.1 200 OK\r\nx-amz-checksum-sha256: one\r\n" +
            "X-Amz-Checksum-Sha256: two\r\n\r\n"
        ))
        XCTAssertThrowsError(try PinnedCloudTransport.parseFinalHeaders("Content-Length: 99"))
    }

    func testCancellationBeforeLaunchPreventsProcessStart() {
        let cancellation = PinnedTransferCancellation()
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/true")

        cancellation.cancel(process)
        XCTAssertThrowsError(try cancellation.start(process)) { error in
            XCTAssertTrue(error is CancellationError)
        }
        XCTAssertFalse(process.isRunning)
    }

    func testCancellationAfterLaunchTerminatesActiveProcess() throws {
        let cancellation = PinnedTransferCancellation()
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/bin/sleep")
        process.arguments = ["5"]

        try cancellation.start(process)
        cancellation.cancel(process)
        process.waitUntilExit()

        XCTAssertFalse(process.isRunning)
        XCTAssertThrowsError(try cancellation.ensureActive()) { error in
            XCTAssertTrue(error is CancellationError)
        }
        // A repeated user cancel must not signal an already exited process.
        cancellation.cancel(process)
    }
}
