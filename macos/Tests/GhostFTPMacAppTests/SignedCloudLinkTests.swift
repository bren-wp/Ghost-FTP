import CryptoKit
import XCTest
@testable import GhostFTPMacApp

final class SignedCloudLinkTests: XCTestCase {
    func testAcceptsS3AndGcsSignedLinksWithoutPersistingToken() throws {
        let s3 = try SignedCloudLink("https://bucket.s3.amazonaws.com/object?X-Amz-Signature=abc")
        XCTAssertEqual(s3.host, "bucket.s3.amazonaws.com")
        XCTAssertNotNil(try SignedCloudLink("https://storage.googleapis.com/bucket/object?X-Goog-Signature=abc"))
        XCTAssertNotNil(try SignedCloudLink("https://account.blob.core.windows.net/container/blob?sig=abc"))
        XCTAssertNotNil(try SignedCloudLink("https://account.r2.cloudflarestorage.com/bucket/object?X-Amz-Signature=abc"))
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
            "https://8.8.8.8/a?sig=abc",
            "https://192.168.1.5/a?sig=abc",
            "https://10.0.0.1/a?sig=abc",
            "https://172.20.0.1/a?sig=abc",
            "https://169.254.0.1/a?sig=abc",
            "https://private.local/a?sig=abc",
            "https://example.com/a?sig=abc",
            "https://bucket.s3.amazonaws.com.attacker.example/a?X-Amz-Signature=abc",
            "https://[::1]/a?sig=abc",
        ]
        for value in bad {
            XCTAssertThrowsError(try SignedCloudLink(value))
        }
    }
    func testRejectsEmptyDuplicateAndWrongProviderSignatures() {
        let invalid = [
            "https://bucket.s3.amazonaws.com/object?X-Amz-Signature=",
            "https://bucket.s3.amazonaws.com/object?X-Amz-Signature=a&X-Amz-Signature=b",
            "https://bucket.s3.amazonaws.com/object?sig=a",
            "https://storage.googleapis.com/bucket/object?X-Amz-Signature=a",
            "https://account.blob.core.windows.net/container/object?X-Goog-Signature=a",
            "https://account.blob.core.windows.net/container/object?sig="
        ]
        for url in invalid {
            XCTAssertThrowsError(try SignedCloudLink(url), "Accepted an invalid cloud signature")
        }
    }

    func testRejectsPunycodeIDNAAndAmbiguousCloudProviderHosts() {
        let unsafe = [
            "https://xn--bcher-kva.s3.amazonaws.com/item?X-Amz-Signature=x",
            "https://under_score.s3.amazonaws.com/item?X-Amz-Signature=x",
            "https://a..s3.amazonaws.com/item?X-Amz-Signature=x",
            "https://-bucket.s3.amazonaws.com/item?X-Amz-Signature=x",
            "https://bucket-.s3.amazonaws.com/item?X-Amz-Signature=x",
            "https://bucket.s3.amazonaws.com./item?X-Amz-Signature=x"
        ]
        for link in unsafe {
            XCTAssertThrowsError(try SignedCloudLink(link), "Unsafe host should be rejected.")
        }
        XCTAssertTrue(SignedCloudLink.isPublicHost("bucket.s3.amazonaws.com"))
        XCTAssertFalse(SignedCloudLink.isPublicHost("xn--bcher-kva.s3.amazonaws.com"))
        XCTAssertFalse(SignedCloudLink.isPublicHost("bucket-.s3.amazonaws.com"))
    }

    func testS3SHA256AcceptsMatchingAndMissingOptionalHeader() throws {
        let bytes = Data("Ghost FTP streamed data".utf8)
        let digest = Data(SHA256.hash(data: bytes))
        XCTAssertNoThrow(try SignedCloudTransfer.verifySHA256(header: digest.base64EncodedString(), digest: digest))
        XCTAssertNoThrow(try SignedCloudTransfer.verifySHA256(header: nil, digest: digest))
    }

    func testS3SHA256RejectsWrongMalformedAndShortHeadersBeforeReplacingFile() throws {
        let actual = Data(SHA256.hash(data: Data("trusted".utf8)))
        let wrong = Data(SHA256.hash(data: Data("untrusted".utf8)))
        for bad in [wrong.base64EncodedString(), "%%%bad", "AA==", ""] {
            XCTAssertThrowsError(try SignedCloudTransfer.verifySHA256(header: bad, digest: actual))
        }
    }

}
