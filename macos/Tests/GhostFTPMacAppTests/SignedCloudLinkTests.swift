import XCTest
@testable import GhostFTPMacApp

final class SignedCloudLinkTests: XCTestCase {
    func testAcceptsS3AndGcsSignedLinksWithoutPersistingToken() throws {
        let s3 = try SignedCloudLink("https://bucket.s3.amazonaws.com/object?X-Amz-Signature=abc")
        XCTAssertEqual(s3.host, "bucket.s3.amazonaws.com")
        XCTAssertNotNil(try SignedCloudLink("https://storage.googleapis.com/bucket/object?X-Goog-Signature=abc"))
        XCTAssertNotNil(try SignedCloudLink("https://account.blob.core.windows.net/container/blob?sig=abc"))
    }

    func testRejectsInsecureOrUnsignedAndCredentialedLinks() {
        let bad = [
            "http://bucket.s3.amazonaws.com/a?X-Amz-Signature=abc",
            "https://bucket.s3.amazonaws.com/a",
            "https://user:password@bucket.s3.amazonaws.com/a?sig=abc",
            "https://bucket.s3.amazonaws.com/a?sig=abc#fragment",
            "https://bucket.s3.amazonaws.com:8443/a?sig=abc",
            "https://localhost/a?sig=abc",
            "https://127.0.0.1/a?sig=abc",
            "https://192.168.1.5/a?sig=abc",
            "https://10.0.0.1/a?sig=abc",
            "https://172.20.0.1/a?sig=abc",
            "https://169.254.0.1/a?sig=abc",
            "https://private.local/a?sig=abc",
            "https://[::1]/a?sig=abc",
        ]
        for value in bad {
            XCTAssertThrowsError(try SignedCloudLink(value))
        }
    }
}
