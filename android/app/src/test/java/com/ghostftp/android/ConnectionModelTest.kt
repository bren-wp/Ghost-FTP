package com.ghostftp.android

import com.jcraft.jsch.ChannelSftp
import com.jcraft.jsch.SftpException
import java.io.IOException
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
    fun remoteParentAndNameSplitsRootAndNestedTargets() {
        assertEquals("/" to "file.txt", controller.remoteParentAndName("/file.txt"))
        assertEquals("/folder" to "file.txt", controller.remoteParentAndName("/folder/file.txt"))
        assertThrows(IllegalArgumentException::class.java) {
            controller.remoteParentAndName("/")
        }
    }

    @Test
    fun remotePathJoinNeverDuplicatesSeparators() {
        assertEquals("/file.txt", controller.joinRemotePath("/", "file.txt"))
        assertEquals("/folder/file.txt", controller.joinRemotePath("/folder/", "/file.txt"))
    }

    @Test
    fun stagedReplacementPromotesWhenTargetIsAbsent() {
        var existenceChecks = 0
        var promoted = false
        var temporaryCleaned = false
        var backupTouched = false

        commitStagedRemoteReplacement(
            remoteFilePath = "/incoming/file.txt",
            backupPath = "/incoming/.backup",
            targetExists = {
                existenceChecks += 1
                false
            },
            backupTarget = { backupTouched = true },
            promoteTemporary = { promoted = true },
            restoreBackup = { throw IOException("restore should not run") },
            cleanupTemporary = { temporaryCleaned = true },
            cleanupBackup = { backupTouched = true }
        )

        assertEquals(2, existenceChecks)
        assertTrue(promoted)
        assertFalse(temporaryCleaned)
        assertFalse(backupTouched)
    }

    @Test
    fun stagedReplacementBacksUpExistingTargetBeforePromotion() {
        var existenceChecks = 0
        var backedUp = false
        var promoted = false
        var backupCleaned = false

        commitStagedRemoteReplacement(
            remoteFilePath = "/incoming/file.txt",
            backupPath = "/incoming/.backup",
            targetExists = {
                existenceChecks += 1
                true
            },
            backupTarget = { backedUp = true },
            promoteTemporary = { promoted = true },
            restoreBackup = { throw IOException("restore should not run") },
            cleanupTemporary = { throw IOException("cleanup should not run") },
            cleanupBackup = { backupCleaned = true }
        )

        assertEquals(1, existenceChecks)
        assertTrue(backedUp)
        assertTrue(promoted)
        assertTrue(backupCleaned)
    }

    @Test
    fun stagedReplacementFailsClosedWhenExistenceCheckFails() {
        var promoted = false
        var temporaryCleaned = false

        assertThrows(IOException::class.java) {
            commitStagedRemoteReplacement(
                remoteFilePath = "/incoming/file.txt",
                backupPath = "/incoming/.backup",
                targetExists = { throw IOException("target lookup denied") },
                backupTarget = { throw IOException("backup should not run") },
                promoteTemporary = { promoted = true },
                restoreBackup = { throw IOException("restore should not run") },
                cleanupTemporary = { temporaryCleaned = true },
                cleanupBackup = { throw IOException("backup cleanup should not run") }
            )
        }

        assertFalse(promoted)
        assertTrue(temporaryCleaned)
    }

    @Test
    fun stagedReplacementCleansTemporaryWhenBackupRenameFails() {
        var promoted = false
        var restored = false
        var temporaryCleaned = false

        val error = assertThrows(IOException::class.java) {
            commitStagedRemoteReplacement(
                remoteFilePath = "/incoming/file.txt",
                backupPath = "/incoming/.backup",
                targetExists = { true },
                backupTarget = { throw IOException("rename denied") },
                promoteTemporary = { promoted = true },
                restoreBackup = { restored = true },
                cleanupTemporary = { temporaryCleaned = true },
                cleanupBackup = { throw IOException("backup cleanup should not run") }
            )
        }

        assertTrue(error.message!!.contains("could not preserve"))
        assertFalse(promoted)
        assertFalse(restored)
        assertTrue(temporaryCleaned)
    }

    @Test
    fun stagedReplacementRestoresOriginalWhenPromotionFails() {
        var restored = false
        var temporaryCleaned = false
        var backupCleaned = false

        val error = assertThrows(IOException::class.java) {
            commitStagedRemoteReplacement(
                remoteFilePath = "/incoming/file.txt",
                backupPath = "/incoming/.backup",
                targetExists = { true },
                backupTarget = { },
                promoteTemporary = { throw IOException("promotion denied") },
                restoreBackup = { restored = true },
                cleanupTemporary = { temporaryCleaned = true },
                cleanupBackup = { backupCleaned = true }
            )
        }

        assertTrue(error.message!!.contains("could not promote"))
        assertTrue(restored)
        assertTrue(temporaryCleaned)
        assertFalse(backupCleaned)
    }

    @Test
    fun stagedReplacementPreservesBackupWhenRollbackFails() {
        var temporaryCleaned = false

        val error = assertThrows(IOException::class.java) {
            commitStagedRemoteReplacement(
                remoteFilePath = "/incoming/file.txt",
                backupPath = "/incoming/.backup",
                targetExists = { true },
                backupTarget = { },
                promoteTemporary = { throw IOException("promotion denied") },
                restoreBackup = { throw IOException("restore denied") },
                cleanupTemporary = { temporaryCleaned = true },
                cleanupBackup = { throw IOException("backup cleanup should not run") }
            )
        }

        assertTrue(temporaryCleaned)
        assertTrue(error.message!!.contains("/incoming/.backup"))
        assertEquals(1, error.suppressed.size)
    }

    @Test
    fun stagedReplacementRejectsTargetThatAppearsDuringTransfer() {
        var existenceChecks = 0
        var promoted = false
        var temporaryCleaned = false

        val error = assertThrows(IOException::class.java) {
            commitStagedRemoteReplacement(
                remoteFilePath = "/incoming/file.txt",
                backupPath = "/incoming/.backup",
                targetExists = {
                    existenceChecks += 1
                    existenceChecks > 1
                },
                backupTarget = { throw IOException("backup should not run") },
                promoteTemporary = { promoted = true },
                restoreBackup = { throw IOException("restore should not run") },
                cleanupTemporary = { temporaryCleaned = true },
                cleanupBackup = { throw IOException("backup cleanup should not run") }
            )
        }

        assertTrue(error.message!!.contains("appeared during transfer"))
        assertEquals(2, existenceChecks)
        assertFalse(promoted)
        assertTrue(temporaryCleaned)
    }

    @Test
    fun sensitiveErrorTextRedactsPasswordsUserInfoAndTokens() {
        val raw = "password=supersecret sftp://deploy:supersecret@example.com Authorization=abc123 Bearer token.value"
        val redacted = redactSensitiveErrorText(raw, listOf("supersecret"))

        assertFalse(redacted.contains("supersecret"))
        assertFalse(redacted.contains("abc123"))
        assertFalse(redacted.contains("token.value"))
        assertTrue(redacted.contains("password=••••"))
        assertTrue(redacted.contains("sftp://deploy:••••@example.com"))
        assertTrue(redacted.contains("Authorization=••••"))
        assertTrue(redacted.contains("Bearer ••••"))
    }

    @Test
    fun sensitiveErrorTextKeepsUsefulNonSecretDiagnostics() {
        assertEquals(
            "Permission denied for /incoming/report.csv",
            redactSensitiveErrorText("Permission denied for /incoming/report.csv")
        )
    }

    @Test
    fun sftpNoSuchFileMeansTargetAbsent() {
        val missing = SftpException(ChannelSftp.SSH_FX_NO_SUCH_FILE, "missing")
        assertFalse(controller.sftpTargetExistsFromLookupFailure(missing))
    }

    @Test
    fun sftpPermissionAndProtocolErrorsFailClosed() {
        for (id in listOf(ChannelSftp.SSH_FX_PERMISSION_DENIED, ChannelSftp.SSH_FX_FAILURE)) {
            val error = SftpException(id, "lookup failed")
            assertThrows(IOException::class.java) {
                controller.sftpTargetExistsFromLookupFailure(error)
            }
        }
    }

}
