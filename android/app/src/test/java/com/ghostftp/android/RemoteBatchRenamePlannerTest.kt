package com.ghostftp.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
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
    @Test fun rejectsOverlappingSourceDestinationAndOversizedBatch() {
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan(listOf("/dir/file_01.txt", "/dir/a.txt"), "file_##", 1)
        }
        assertThrows(IllegalArgumentException::class.java) {
            RemoteBatchRenamePlanner.plan((1..13).map { "/dir/x$it.txt" }, "file_##", 1)
        }
    }
}
