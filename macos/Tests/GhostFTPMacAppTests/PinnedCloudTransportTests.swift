import Foundation
import XCTest
@testable import GhostFTPMacApp

final class PinnedCloudTransportTests: XCTestCase {
    func testOnlyGlobalPublicIPv4CanBecomePeerCandidates() {
        XCTAssertTrue(PinnedCloudPublicAddress.allowed([1, 1, 1, 1]))
        XCTAssertTrue(PinnedCloudPublicAddress.allowed([8, 8, 8, 8]))
        for address: [UInt8] in [
            [0, 0, 0, 0], [10, 0, 0, 1], [100, 64, 0, 1],
            [127, 0, 0, 1], [169, 254, 169, 254], [172, 20, 1, 2],
            [192, 168, 0, 1], [192, 0, 2, 1], [198, 18, 0, 1],
            [198, 51, 100, 2], [203, 0, 113, 2], [224, 0, 0, 1]
        ] {
            XCTAssertFalse(PinnedCloudPublicAddress.allowed(address), "\(address)")
        }
    }

    func testOnlyReviewedGlobalUnicastIPv6CanBecomePeerCandidates() {
        XCTAssertTrue(PinnedCloudPublicAddress.allowed(
            [0x26, 0x06, 0x47, 0x00] + Array(repeating: 0, count: 12)
        ))
        for prefix: [UInt8] in [
            [0, 0, 0, 0], [0xfc, 0, 0, 0], [0xfe, 0x80, 0, 0],
            [0x20, 0x01, 0x0d, 0xb8], [0x20, 0x02, 0, 0],
            [0x20, 0x01, 0, 0]
        ] {
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
}
