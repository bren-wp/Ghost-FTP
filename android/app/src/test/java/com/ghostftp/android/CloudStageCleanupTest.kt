package com.ghostftp.android

import java.io.File
import java.nio.file.Files
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CloudStageCleanupTest {
    @Test fun cleansOnlyMatchingOrphanStagesWithoutTouchingUserFilesOrFolders() {
        val cache = Files.createTempDirectory("ghostftp-stage-qa-").toFile()
        try {
            val orphan = File.createTempFile("ghostftp-cloud-", ".part", cache)
            orphan.writeBytes(byteArrayOf(1, 2, 3))
            val unrelated = File(cache, "unrelated.part").apply { writeText("preserve") }
            val similar = File(cache, "ghostftp-cloud-download.tmp").apply { writeText("preserve") }
            val directory = File(cache, "ghostftp-cloud-123456.part").apply { mkdir() }
            val nested = File(directory, "nested.part").apply { writeText("preserve") }
            val tooShort = File(cache, "ghostftp-cloud-x.part").apply { writeText("preserve") }
            val unrelatedExtension = File(cache, "ghostftp-cloud-abcdef.other").apply { writeText("preserve") }

            assertEquals(1, CloudStageCleanup.purge(cache))
            assertFalse(orphan.exists())
            assertTrue(unrelated.exists())
            assertEquals("preserve", unrelated.readText())
            assertTrue(similar.exists())
            assertTrue(directory.isDirectory)
            assertTrue(nested.exists())
            assertTrue(tooShort.exists())
            assertTrue(unrelatedExtension.exists())
            assertEquals(0, CloudStageCleanup.purge(cache))
        } finally {
            cache.deleteRecursively()
        }
    }

    @Test fun missingCacheDirectoryDoesNotCrashStartup() {
        val missing = File(Files.createTempDirectory("ghostftp-no-cache-").toFile(), "absent")
        try {
            assertEquals(0, CloudStageCleanup.purge(missing))
        } finally {
            missing.parentFile?.deleteRecursively()
        }
    }
}
