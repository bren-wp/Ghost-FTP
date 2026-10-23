package com.ghostftp.android

import java.io.File
import java.nio.file.Files
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LocalBrowserListingReaderTest {
    @Test fun filesAreSortedWithDirectoriesFirstAndNeverEscapeRoot() {
        val root = Files.createTempDirectory("ghostftp-listing").toFile()
        try {
            File(root, "z.txt").writeText("z")
            File(root, "a.txt").writeText("a")
            File(root, "folder").mkdir()
            val result = LocalBrowserListingReader.list(root, root)
            assertFalse(result.unavailable)
            assertFalse(result.truncated)
            assertEquals(listOf("folder", "a.txt", "z.txt"), result.files.map { it.name })
        } finally {
            root.deleteRecursively()
        }
    }

    @Test fun symlinkCannotLeakOutsideAppRoot() {
        val root = Files.createTempDirectory("ghostftp-root").toFile()
        val outside = Files.createTempDirectory("ghostftp-outside").toFile()
        try {
            File(outside, "private.txt").writeText("do not expose")
            Files.createSymbolicLink(File(root, "external").toPath(), outside.toPath())
            val result = LocalBrowserListingReader.list(root, root)
            assertFalse(result.unavailable)
            assertFalse(result.files.any { it.name == "external" })
            assertTrue(LocalBrowserListingReader.list(root, outside).unavailable)
        } finally {
            root.deleteRecursively()
            outside.deleteRecursively()
        }
    }

    @Test fun boundedScanAndRowsStaySmallForHugeDirectories() {
        val root = Files.createTempDirectory("ghostftp-bounded").toFile()
        try {
            repeat(48) { File(root, "item-%03d.txt".format(it)).writeText("x") }
            val result = LocalBrowserListingReader.list(root, root, displayedLimit = 8, scanLimit = 16)
            assertFalse(result.unavailable)
            assertTrue(result.truncated)
            assertEquals(8, result.files.size)
        } finally {
            root.deleteRecursively()
        }
    }

    @Test fun revokedOrMissingFolderReturnsSafeUnavailableState() {
        val root = Files.createTempDirectory("ghostftp-missing").toFile()
        try {
            assertTrue(LocalBrowserListingReader.list(root, File(root, "missing")).unavailable)
        } finally {
            root.deleteRecursively()
        }
    }
}
