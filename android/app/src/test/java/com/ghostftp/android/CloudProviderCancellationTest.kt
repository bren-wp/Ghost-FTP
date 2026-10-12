package com.ghostftp.android

import java.io.Closeable
import java.io.InterruptedIOException
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test

/** JVM regression tests for Android SAF provider close-on-cancel behavior. */
class CloudProviderCancellationTest {
    @Test fun cancelDoesNotWaitForBlockedProviderCloseOnCallerThread() {
        val enteredClose = CountDownLatch(1)
        val allowClose = CountDownLatch(1)
        val cancelReturned = CountDownLatch(1)
        val stream = object : Closeable {
            override fun close() {
                enteredClose.countDown()
                allowClose.await(5, TimeUnit.SECONDS)
            }
        }
        val cancellation = CloudTransferCancellation()
        cancellation.watchStream(stream)
        try {
            val caller = Thread({
                cancellation.cancel()
                cancelReturned.countDown()
            }, "test-cloud-cancel")
            caller.start()
            assertTrue("cancel should return while provider close is blocked",
                cancelReturned.await(2, TimeUnit.SECONDS))
            assertTrue("cancellation should close the registered provider stream",
                enteredClose.await(2, TimeUnit.SECONDS))
            assertTrue(cancellation.isCancelled)
            caller.join(1000)
        } finally {
            allowClose.countDown()
        }
    }

    @Test fun streamOpenedAfterCancellationIsClosedAndRejected() {
        val cancellation = CloudTransferCancellation()
        cancellation.cancel()
        val closed = AtomicInteger(0)
        val stream = object : Closeable {
            override fun close() { closed.incrementAndGet() }
        }
        try {
            cancellation.watchStream(stream)
            fail("An already cancelled transfer must not accept a new provider stream")
        } catch (_: InterruptedIOException) {
            assertEquals(1, closed.get())
        }
    }

    @Test fun completedProviderStreamIsNeverClosedByLaterCancellation() {
        val cancellation = CloudTransferCancellation()
        val closed = AtomicInteger(0)
        val stream = object : Closeable {
            override fun close() { closed.incrementAndGet() }
        }
        cancellation.watchStream(stream)
        cancellation.unwatchStream(stream)
        cancellation.cancel()
        cancellation.cancel()
        assertEquals(0, closed.get())
    }
}
