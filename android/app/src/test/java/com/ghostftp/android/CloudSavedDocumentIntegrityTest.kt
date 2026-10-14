package com.ghostftp.android

import java.io.ByteArrayInputStream
import java.io.IOException
import java.security.MessageDigest
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test

/** Exercises post-write SAF integrity without any real cloud credential or provider. */
class CloudSavedDocumentIntegrityTest {
    private fun hash(data: ByteArray): ByteArray =
        MessageDigest.getInstance("SHA-256").digest(data)

    private fun verify(
        actual: ByteArray,
        expected: ByteArray,
        cancellation: CloudTransferCancellation = CloudTransferCancellation()
    ) = SignedCloudTransfer.verifyPersistedCopy(
        ByteArrayInputStream(actual),
        expected.size.toLong(),
        hash(expected),
        cancellation
    )

    private fun mustReject(actual: ByteArray, expected: ByteArray) {
        try {
            verify(actual, expected)
            fail("SAF read-back changed or truncated content but reported success.")
        } catch (_: IOException) {
            // Provider lied about persisted bytes: fail closed.
        }
    }

    @Test fun acceptsExactDocumentAndZeroByteObject() {
        verify(ByteArray(0), ByteArray(0))
        val objectBytes = ByteArray(32769) { (it % 239).toByte() }
        verify(objectBytes.copyOf(), objectBytes)
    }

    @Test fun rejectsProviderTruncation() {
        val expected = ByteArray(64000) { 0x31 }
        mustReject(expected.copyOfRange(0, 63000), expected)
    }

    @Test fun rejectsSameLengthSilentDataCorruption() {
        val expected = ByteArray(8192) { (it % 251).toByte() }
        val corrupted = expected.copyOf()
        corrupted[8120] = (corrupted[8120].toInt() xor 0x40).toByte()
        mustReject(corrupted, expected)
    }

    @Test fun rejectsProviderExtraBytes() {
        mustReject(byteArrayOf(1, 2, 3, 4), byteArrayOf(1, 2, 3))
    }

    @Test fun refusesCancelledReadBackEvenForMatchingBytes() {
        val objectBytes = byteArrayOf(4, 5, 6)
        val cancellation = CloudTransferCancellation()
        cancellation.cancel()
        try {
            verify(objectBytes, objectBytes, cancellation)
            fail("Canceled SAF verification must not signal successful transfer.")
        } catch (_: java.io.InterruptedIOException) {
            assertTrue(cancellation.isCancelled)
        }
    }

    @Test fun rejectsNonProgressingDocumentProviderStream() {
        val input = object : java.io.InputStream() {
            override fun read(): Int = 0
            override fun read(buffer: ByteArray, offset: Int, length: Int): Int = 0
        }
        try {
            SignedCloudTransfer.verifyPersistedCopy(
                input, 1L, hash(byteArrayOf(1)), CloudTransferCancellation()
            )
            fail("A SAF provider returning endless zero-byte reads must fail closed.")
        } catch (_: IOException) {
            // A stalled readback must not hang the app or claim success.
        }
    }

    @Test fun readBackRejectsCancellationDuringBlockedProviderRead() {
        val cancellation = CloudTransferCancellation()
        val input = object : java.io.InputStream() {
            override fun read(): Int = 7
            override fun read(buffer: ByteArray, offset: Int, length: Int): Int {
                buffer[offset] = 7
                cancellation.cancel()
                return 1
            }
        }
        try {
            SignedCloudTransfer.verifyPersistedCopy(
                input, 1L, hash(byteArrayOf(7)), cancellation
            )
            fail("A canceled SAF provider read must never report a verified document.")
        } catch (_: java.io.InterruptedIOException) {
            assertTrue(cancellation.isCancelled)
        }
    }

    @Test fun rejectsInvalidExpectedChecksumMetadata() {
        try {
            SignedCloudTransfer.verifyPersistedCopy(
                ByteArrayInputStream(byteArrayOf(1)),
                1,
                byteArrayOf(42),
                CloudTransferCancellation()
            )
            fail("Malformed expected checksum must fail.")
        } catch (_: IOException) {
            // Correct.
        }
    }
}
