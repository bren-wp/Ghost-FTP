package com.ghostftp.android

import com.jcraft.jsch.ChannelSftp
import org.apache.commons.net.ftp.FTPFile
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
    fun boundedRemoteEditorCopyRejectsOverLimitWithoutWritingExtraBytes() {
        val source = ByteArray(524_289) { 65 }
        val sink = java.io.ByteArrayOutputStream()
        assertThrows(IOException::class.java) {
            controller.copyCancelable(
                java.io.ByteArrayInputStream(source), sink,
                OperationCancellation(), maxBytes = 524_288
            )
        }
        assertTrue("No bytes beyond the editor limit may be written.",
            sink.size() <= 524_288)
    }

    @Test
    fun boundedRemoteEditorCopyAcceptsExactLimitAndRejectsNegativeLimits() {
        val data = "hello".toByteArray(Charsets.UTF_8)
        val sink = java.io.ByteArrayOutputStream()
        assertEquals(5L, controller.copyCancelable(
            java.io.ByteArrayInputStream(data), sink,
            OperationCancellation(), maxBytes = 5
        ))
        assertEquals("hello", sink.toString("UTF-8"))
        assertThrows(IllegalArgumentException::class.java) {
            controller.copyCancelable(
                java.io.ByteArrayInputStream(data), java.io.ByteArrayOutputStream(),
                OperationCancellation(), maxBytes = -1L
            )
        }
    }

    @Test
    fun boundedCopyHonorsCancellationBeforeWritingData() {
        val canceled = OperationCancellation().apply { cancel() }
        val output = java.io.ByteArrayOutputStream()
        assertThrows(OperationCanceledException::class.java) {
            controller.copyCancelable(
                java.io.ByteArrayInputStream(byteArrayOf(1, 2, 3)), output,
                canceled, maxBytes = 512 * 1024L
            )
        }
        assertEquals(0, output.size())
    }

    @Test
    fun protocolIndexUsesSafeFallback() {
        assertEquals(ConnectionProtocol.FTP, ConnectionProtocol.fromIndex(-1))
        assertEquals(ConnectionProtocol.FTP, ConnectionProtocol.fromIndex(99))
        assertEquals(ConnectionProtocol.SFTP, ConnectionProtocol.fromIndex(ConnectionProtocol.SFTP.ordinal))
    }

    @Test
    fun loadingSavedProtocolNeverOverwritesCustomPortWhenSelectionIsUnchanged() {
        // Android Spinner may omit onItemSelected if setSelection is a no-op.
        // Subsequent callbacks for the current selection must preserve the port.
        assertEquals(null, portOnProtocolChange(ConnectionProtocol.FTP, ConnectionProtocol.FTP, "2121"))
        assertEquals(null, portOnProtocolChange(ConnectionProtocol.SFTP, ConnectionProtocol.SFTP, "21"))
        assertEquals(null, portOnProtocolChange(ConnectionProtocol.EXPLICIT_FTPS, ConnectionProtocol.EXPLICIT_FTPS, "22"))
    }

    @Test
    fun userProtocolChangesOnlyReplaceOldDefaultAndNeverChangeCustomPorts() {
        assertEquals("22", portOnProtocolChange(ConnectionProtocol.FTP, ConnectionProtocol.SFTP, "21"))
        assertEquals("21", portOnProtocolChange(ConnectionProtocol.SFTP, ConnectionProtocol.FTP, "22"))
        assertEquals("21", portOnProtocolChange(ConnectionProtocol.SFTP, ConnectionProtocol.EXPLICIT_FTPS, " "))
        assertEquals(null, portOnProtocolChange(ConnectionProtocol.FTP, ConnectionProtocol.SFTP, "2121"))
        assertEquals(null, portOnProtocolChange(ConnectionProtocol.SFTP, ConnectionProtocol.FTP, "21"))
    }

    @Test
    fun switchingAwayAndBackCannotResurrectPreviouslyLoadedPort() {
        // A loaded FTP:2121 profile is edited to 2021. The user changes the
        // protocol twice; neither callback can resurrect the old saved port.
        val edited = "2021"
        assertEquals(null, portOnProtocolChange(ConnectionProtocol.FTP, ConnectionProtocol.SFTP, edited))
        assertEquals(null, portOnProtocolChange(ConnectionProtocol.SFTP, ConnectionProtocol.FTP, edited))
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
        for (value in listOf(
            "user@example.com",
            "example.com:2121",
            "https://example.com",
            "bad host.example",
            "bad\n.example"
        )) {
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

        val sftp = base.copy(
            protocol = ConnectionProtocol.SFTP,
            port = 22,
            hostKeyFingerprint = "SHA256:test",
            keepAliveSeconds = 15
        )
        assertEquals(15, controller.normalizedProfile(sftp).keepAliveSeconds)
        assertThrows(IllegalArgumentException::class.java) {
            controller.normalizedProfile(sftp.copy(keepAliveSeconds = 4))
        }
        assertThrows(IllegalArgumentException::class.java) {
            controller.normalizedProfile(sftp.copy(keepAliveSeconds = 301))
        }
    }

    @Test
    fun ftpListingRequiresSuccessfulFinalControlReply() {
        controller.requireSuccessfulFtpListing(226)
        assertThrows(IOException::class.java) {
            controller.requireSuccessfulFtpListing(550)
        }
        assertThrows(IOException::class.java) {
            controller.requireSuccessfulFtpListing(421)
        }
    }

    @Test
    fun remoteListingRejectsPathInjectionAndAmbiguousNames() {
        for (invalid in listOf("", ".", "..", "../secret", "nested/file", "bad\nname", "bad\rname", "bad\u0000name")) {
            assertThrows(IOException::class.java) {
                controller.validateRemoteInventoryName(invalid)
            }
        }
        controller.validateRemoteInventoryName("Račun 2026.txt")

        val duplicated = arrayOf(
            FTPFile().apply { name = "same.txt"; type = FTPFile.FILE_TYPE; size = 3L },
            FTPFile().apply { name = "same.txt"; type = FTPFile.DIRECTORY_TYPE },
        )
        assertThrows(IOException::class.java) {
            controller.ftpRows("/docs", duplicated)
        }
    }

    @Test
    fun remoteListingDisclosesDisplayLimitInsteadOfPretendingToShowAllFiles() {
        val files = Array(27) { index ->
            FTPFile().apply {
                name = "file-$index.txt"
                type = FTPFile.FILE_TYPE
                size = 2L
            }
        }
        val rows = controller.ftpRows("/docs", files)
        assertEquals(26, rows.size)
        assertEquals("Remote path", rows.first().name)
        assertTrue(rows.last().detail.contains("first 24 of 27"))
        assertFalse(rows.last().isFile)
        assertFalse(rows.last().isDirectory)
    }

    @Test
    fun remoteSymlinksAreVisibleButCannotBeMistakenForRegularFiles() {
        val link = FTPFile().apply {
            name = "target-link"
            type = FTPFile.SYMBOLIC_LINK_TYPE
        }
        val rows = controller.ftpRows("/docs", arrayOf(link))
        assertEquals("[LINK] target-link", rows[1].name)
        assertFalse(rows[1].isFile)
        assertFalse(rows[1].isDirectory)
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
        assertTrue(redacted.contains("sftp://••••@example.com"))
        assertFalse(redacted.contains("deploy"))
        assertTrue(redacted.contains("Authorization=••••"))
        assertTrue(redacted.contains("Bearer ••••"))
    }

    @Test
    fun sensitiveErrorTextRedactsUsernamesWithoutPasswordsAndEncodedCredentials() {
        val diagnostic = "ftp://deploy@example.com sftp://bob:p%40ss@host.example.com " +
            "https://status.example.com/help and ftp://host.example.com:21/public"
        val redacted = redactSensitiveErrorText(diagnostic)
        assertFalse(redacted.contains("deploy"))
        assertFalse(redacted.contains("bob"))
        assertFalse(redacted.contains("p%40ss"))
        assertTrue(redacted.contains("ftp://••••@example.com"))
        assertTrue(redacted.contains("sftp://••••@host.example.com"))
        assertTrue(redacted.contains("https://status.example.com/help"))
        assertTrue(redacted.contains("ftp://host.example.com:21/public"))
    }

    @Test
    fun sensitiveErrorTextHidesSecretCrossingDisplayLimit() {
        val prefix = "x".repeat(596)
        val secret = "dontLeakThisPassword"
        val raw = prefix + " " + secret + " network unreachable"
        val redacted = redactSensitiveErrorText(raw, listOf(secret))
        assertTrue(redacted.length <= 600)
        assertFalse(redacted.contains(secret.take(3)))
        assertTrue(redacted.endsWith("••"))
    }

    @Test
    fun sensitiveErrorTextTruncatesLargeUntrustedServerError() {
        val raw = "error ".repeat(5000)
        assertEquals(600, redactSensitiveErrorText(raw).length)
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
