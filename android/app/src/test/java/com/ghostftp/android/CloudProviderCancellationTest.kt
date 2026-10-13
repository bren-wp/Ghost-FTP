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

    @Test fun blockedProviderCloseDoesNotHoldUpAnotherProviderStream() {
        val firstCloseEntered = CountDownLatch(1)
        val unblockFirst = CountDownLatch(1)
        val secondClosed = CountDownLatch(1)
        val cancellation = CloudTransferCancellation()
        cancellation.watchStream(object : Closeable {
            override fun close() {
                firstCloseEntered.countDown()
                unblockFirst.await(5, TimeUnit.SECONDS)
            }
        })
        cancellation.watchStream(object : Closeable {
            override fun close() { secondClosed.countDown() }
        })
        try {
            cancellation.cancel()
            assertTrue("first provider close should begin",
                firstCloseEntered.await(2, TimeUnit.SECONDS))
            assertTrue("blocked close must not starve other SAF provider handles",
                secondClosed.await(2, TimeUnit.SECONDS))
        } finally {
            unblockFirst.countDown()
        }
    }

    @Test fun streamOpenedAfterCancellationIsClosedAndRejected() {
        val cancellation = CloudTransferCancellation()
        cancellation.cancel()
        val closed = CountDownLatch(1)
        val stream = object : Closeable {
            override fun close() { closed.countDown() }
        }
        try {
            cancellation.watchStream(stream)
            fail("An already cancelled transfer must not accept a new provider stream")
        } catch (_: InterruptedIOException) {
            assertTrue("late SAF stream must receive off-thread close", closed.await(2, TimeUnit.SECONDS))
        }
    }

    @Test fun lateStreamRegistrationDoesNotBlockOnHostileProviderClose() {
        val cancellation = CloudTransferCancellation()
        cancellation.cancel()
        val closeEntered = CountDownLatch(1)
        val unblockProvider = CountDownLatch(1)
        val registrationReturned = CountDownLatch(1)
        val lateStream = object : Closeable {
            override fun close() {
                closeEntered.countDown()
                unblockProvider.await(5, TimeUnit.SECONDS)
            }
        }
        try {
            val registrar = Thread({
                try {
                    cancellation.watchStream(lateStream)
                    fail("A cancelled transfer must reject late streams")
                } catch (_: InterruptedIOException) {
                    registrationReturned.countDown()
                }
            }, "test-cloud-late-registration")
            registrar.start()
            assertTrue("late registration must reject without waiting for SAF close",
                registrationReturned.await(2, TimeUnit.SECONDS))
            assertTrue("provider close request must still be dispatched",
                closeEntered.await(2, TimeUnit.SECONDS))
            registrar.join(1000)
        } finally {
            unblockProvider.countDown()
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
