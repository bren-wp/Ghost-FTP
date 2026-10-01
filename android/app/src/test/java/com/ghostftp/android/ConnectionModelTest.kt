package com.ghostftp.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test

class ConnectionModelTest {
    private val controller = ConnectionController()

    @Test
    fun protocolIndexUsesSafeFallback() {
        assertEquals(ConnectionProtocol.FTP, ConnectionProtocol.fromIndex(-1))
        assertEquals(ConnectionProtocol.FTP, ConnectionProtocol.fromIndex(99))
        assertEquals(ConnectionProtocol.SFTP, ConnectionProtocol.fromIndex(ConnectionProtocol.SFTP.ordinal))
    }

    @Test
    fun cancellationBecomesStickyAndThrows() {
        val cancellation = OperationCancellation()
        assertFalse(cancellation.isCanceled())

        cancellation.cancel()

        assertTrue(cancellation.isCanceled())
        assertThrows(OperationCanceledException::class.java) {
            cancellation.throwIfCanceled()
        }
    }

    @Test
    fun hostNormalizationAcceptsSupportedForms() {
        assertEquals("example.com", controller.normalizeHost(" ftp://example.com/path "))
        assertEquals("xn--bcher-kva.example", controller.normalizeHost("bücher.example"))
        assertEquals("2001:db8::1", controller.normalizeHost("[2001:db8::1]"))
    }

    @Test
    fun hostNormalizationRejectsEmbeddedCredentialsPortsAndSchemes() {
        for (value in listOf("user@example.com", "example.com:2121", "https://example.com")) {
            assertThrows(IllegalArgumentException::class.java) {
                controller.normalizeHost(value)
            }
        }
    }

    @Test
    fun remoteTargetsRejectDotSegmentsAndRootDestruction() {
        assertEquals("/folder/file.txt", controller.normalizeRemoteTarget("folder/file.txt"))
        assertEquals("/folder", controller.normalizeRemoteDirectory("folder"))

        for (value in listOf("../secret", "/folder/../secret", "/folder/./file.txt")) {
            assertThrows(IllegalArgumentException::class.java) {
                controller.normalizeRemoteTarget(value)
            }
            assertThrows(IllegalArgumentException::class.java) {
                controller.normalizeRemoteDirectory(value)
            }
        }

        assertThrows(IllegalArgumentException::class.java) {
            controller.normalizeRemoteDeleteTarget("/")
        }
    }

    @Test
    fun normalizedProfileRejectsInvalidPortAndNormalizesRemotePath() {
        val base = ConnectionProfile(
            protocol = ConnectionProtocol.FTP,
            host = "ftp://example.com/",
            port = 21,
            username = "user",
            password = "secret",
            hostKeyFingerprint = "",
            remotePath = "incoming"
        )

        val normalized = controller.normalizedProfile(base)
        assertEquals("example.com", normalized.host)
        assertEquals("/incoming", normalized.remotePath)

        assertThrows(IllegalArgumentException::class.java) {
            controller.normalizedProfile(base.copy(port = 0))
        }
    }

    @Test
    fun remotePathJoinNeverDuplicatesSeparators() {
        assertEquals("/file.txt", controller.joinRemotePath("/", "file.txt"))
        assertEquals("/folder/file.txt", controller.joinRemotePath("/folder/", "/file.txt"))
    }
}
