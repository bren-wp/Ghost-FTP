package com.ghostftp.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test

class RemoteBatchRenamePlannerTest {
    @Test fun buildsZeroPaddedPreviewWithOriginalExtensions() {
        val p = RemoteBatchRenamePlanner.plan(
            listOf("/var/www/html/z.php", "/var/www/html/a.txt"), "file_##", 1)
        assertEquals("/var/www/html/file_01.txt", p[0].to)
        assertEquals("/var/www/html/file_02.php", p[1].to)
    }
    @Test fun rejectsPathTraversalAndClashingPatterns() {
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(listOf("/safe/a.txt"), "../file_##", 1)
        }
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(listOf("/safe/a.txt"), "file", 1)
        }
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(listOf("/safe/a.txt","/safe/b.txt"), "file_#", 1_000_000)
        }
    }
    @Test fun rejectsNumberFieldOverflowBeforeAnyRemoteOperation() {
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(listOf("/dir/a.txt", "/dir/b.txt"), "file_##", 99)
        }
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(listOf("/dir/a.txt"), "file_##", 100)
        }
        assertEquals("/dir/file_099.txt",
            RemoteBatchRenamePlanner.plan(listOf("/dir/a.txt"), "file_###", 99).single().to)
    }

    @Test fun rejectsCaseInsensitiveAndUnicodeNormalizingSourceCollisions() {
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(
                listOf("/dir/FILE_01.TXT", "/dir/z.txt"), "file_##", 1)
        }
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(
                listOf("/dir/caf\u00e9_01.txt", "/dir/z.txt"), "cafe\u0301_##", 1)
        }
    }

    @Test fun rejectsInvalidAndEquivalentRemoteSourcesBeforeRename() {
        for (path in listOf("/dir/../secret.txt", "/dir/./file.txt",
            "/dir//file.txt", "/dir/trailing/", "/dir/bad\\name.txt",
            "/dir/bad\nname.txt")) {
            assertThrows(IllegalArgumentException::class.java) {
                RemoteBatchRenamePlanner.plan(listOf(path), "file_##", 1)
            }
        }
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(
                listOf("/dir/Résumé.txt", "/dir/Résumé.txt"), "new_##", 1)
        }
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(
                listOf("/dir/same.txt", "/dir/same.txt"), "new_##", 1)
        }
    }

    @Test fun preflightRejectsOccupiedNamesAcrossCaseUnicodeAndDirectories() {
        val plan = RemoteBatchRenamePlanner.plan(listOf("/dir/source.txt"), "report_##", 1)
        assertTrue(RemoteBatchRenamePlanner.hasExistingDestinationCollision(
            plan, listOf("/dir/REPORT_01.txt")))
        assertTrue(RemoteBatchRenamePlanner.hasExistingDestinationCollision(
            plan, listOf("/dir/report_01.txt"))) // May be a directory rather than a file.
        assertFalse(RemoteBatchRenamePlanner.hasExistingDestinationCollision(
            plan, listOf("/dir/other.txt", "/dir/source.txt")))

        val unicodePlan = RemoteBatchRenamePlanner.plan(
            listOf("/dir/source.txt"), "café_##", 1)
        assertTrue(RemoteBatchRenamePlanner.hasExistingDestinationCollision(
            unicodePlan, listOf("/dir/café_01.txt")))
    }

    @Test fun rejectsOverlappingSourceDestinationAndOversizedBatch() {
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(listOf("/dir/file_01.txt", "/dir/a.txt"), "file_##", 1)
        }
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan((1..13).map { "/dir/x$it.txt" }, "file_##", 1)
        }
    }
}
