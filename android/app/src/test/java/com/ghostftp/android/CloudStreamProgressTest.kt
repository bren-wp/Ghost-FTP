package com.ghostftp.android

import java.io.ByteArrayInputStream
import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.InputStream
import java.io.InterruptedIOException
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.fail
import org.junit.Test

/** Regression coverage for hostile/contract-violating SAF and network streams. */
class CloudStreamProgressTest {
    private fun copy(
        input: InputStream,
        maxBytes: Long,
        expected: Long? = null,
        cancellation: CloudTransferCancellation = CloudTransferCancellation()
    ): Pair<Long, ByteArray> {
        val output = ByteArrayOutputStream()
        val count = SignedCloudTransfer.streamWithBoundedProgress(
            input, maxBytes, expected, cancellation
        ) { bytes, length -> output.write(bytes, 0, length) }
        return count to output.toByteArray()
    }

    private fun shouldReject(block: () -> Unit) {
        try {
            block()
            fail("A corrupt/blocked source must not report a successful transfer.")
        } catch (_: IOException) {
            // Fail closed rather than accepting incomplete or unbounded reads.
        }
    }

    @Test fun exactSizeAndEmptyDocumentsSucceed() {
        val source = ByteArray(65537) { (it % 251).toByte() }
        val (count, saved) = copy(ByteArrayInputStream(source), source.size.toLong(), source.size.toLong())
        assertEquals(source.size.toLong(), count)
        assertArrayEquals(source, saved)
        assertEquals(0L, copy(ByteArrayInputStream(byteArrayOf()), 0L, 0L).first)
    }

    @Test fun truncatedAndExpandedDocumentsFailClosed() {
        shouldReject { copy(ByteArrayInputStream(byteArrayOf(1, 2)), 5, 3) }
        shouldReject { copy(ByteArrayInputStream(byteArrayOf(1, 2, 3, 4)), 3, 3) }
        shouldReject { copy(ByteArrayInputStream(byteArrayOf(1)), 0, 0) }
    }

    @Test fun unboundedZeroReadsAreRejected() {
        val source = object : InputStream() {
            override fun read(): Int = 0
            override fun read(buffer: ByteArray, offset: Int, length: Int): Int = 0
        }
        shouldReject { copy(source, 32) }
    }

    @Test fun transientZeroReadsRecoverWithinBound() {
        val original = byteArrayOf(11, 22, 33)
        val source = object : InputStream() {
            var zeros = 0
            val wrapped = ByteArrayInputStream(original)
            override fun read(): Int = wrapped.read()
            override fun read(buffer: ByteArray, offset: Int, length: Int): Int {
                if (zeros++ < 8) return 0
                return wrapped.read(buffer, offset, length)
            }
        }
        val (count, saved) = copy(source, 3, 3)
        assertEquals(3L, count)
        assertArrayEquals(original, saved)
    }

    @Test fun cancelledStreamCannotSucceed() {
        val cancellation = CloudTransferCancellation()
        cancellation.cancel()
        shouldReject { copy(ByteArrayInputStream(byteArrayOf(1)), 1, 1, cancellation) }
        assertEquals(true, cancellation.isCancelled)
    }

    @Test fun invalidDeclaredSizeIsRejected() {
        shouldReject { copy(ByteArrayInputStream(byteArrayOf()), -1, 0) }
        shouldReject { copy(ByteArrayInputStream(byteArrayOf()), 1, 2) }
        shouldReject { copy(ByteArrayInputStream(byteArrayOf()), 1_073_741_825L) }
    }

    @Test fun sinkFailurePropagates() {
        val source = ByteArrayInputStream(byteArrayOf(1, 2, 3))
        shouldReject {
            SignedCloudTransfer.streamWithBoundedProgress(
                source, 3, 3, CloudTransferCancellation()
            ) { _, _ -> throw IOException("provider write failure") }
        }
    }
}
