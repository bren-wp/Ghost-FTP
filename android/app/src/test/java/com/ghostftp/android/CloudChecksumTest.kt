package com.ghostftp.android

import java.io.IOException
import java.security.MessageDigest
import java.util.Base64
import org.junit.Assert.assertThrows
import org.junit.Test

/** No provider credentials or internet access needed for checksum contract. */
class CloudChecksumTest {
    @Test fun acceptsMatchingS3Sha256Checksum() {
        val bytes = "Ghost FTP verified bytes".toByteArray()
        val digest = MessageDigest.getInstance("SHA-256").digest(bytes)
        SignedCloudTransfer.requireMatchingSha256Header(Base64.getEncoder().encodeToString(digest), digest)
    }

    @Test fun acceptsMissingOptionalSha256HeaderWithoutClaimingVerification() {
        val digest = MessageDigest.getInstance("SHA-256").digest(byteArrayOf())
        SignedCloudTransfer.requireMatchingSha256Header(null, digest)
    }

    @Test fun rejectsMismatchBeforeSafDestinationIsOpened() {
        val actual = MessageDigest.getInstance("SHA-256").digest("object".toByteArray())
        val expected = MessageDigest.getInstance("SHA-256").digest("tampered".toByteArray())
        assertThrows(IOException::class.java) {
            SignedCloudTransfer.requireMatchingSha256Header(
                Base64.getEncoder().encodeToString(expected), actual
            )
        }
    }

    @Test fun rejectsMalformedAndTruncatedProviderChecksum() {
        val digest = MessageDigest.getInstance("SHA-256").digest(byteArrayOf(0))
        for (header in listOf("not base64???", "AA==", "")) {
            assertThrows(IOException::class.java) {
                SignedCloudTransfer.requireMatchingSha256Header(header, digest)
            }
        }
    }
}
