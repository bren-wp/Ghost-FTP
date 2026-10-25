package com.ghostftp.android

import java.io.File
import java.nio.file.Files
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LocalScopedSearchTest {
    @Test fun nonRecursiveSearchFindsOnlyImmediateChildren() {
        val root = Files.createTempDirectory("ghostftp-local-search").toFile()
        try {
            File(root, "example.txt").writeText("a")
            File(root, "other.jpg").writeText("b")
            File(root, "folder").mkdir()
            File(root, "folder/nested.txt").writeText("c")
            val regex = FilenameGlob("*.txt")
            val shallow = LocalScopedSearch.find(root, regex, recursive = false)
            assertFalse(shallow.unavailable)
            assertFalse(shallow.cancelled)
            assertEquals(listOf("example.txt"), shallow.matches.map { it.name })
            val deep = LocalScopedSearch.find(root, regex, recursive = true)
            assertEquals(listOf("example.txt", "nested.txt"), deep.matches.map { it.name })
        } finally {
            root.deleteRecursively()
        }
    }

    @Test fun neverSearchesThroughSymlinksEvenInsideAppRoot() {
        val root = Files.createTempDirectory("ghostftp-local-root").toFile()
        val outside = Files.createTempDirectory("ghostftp-secret-root").toFile()
        try {
            File(outside, "private.txt").writeText("secret")
            File(root, "ok.txt").writeText("safe")
            Files.createSymbolicLink(File(root, "outside").toPath(), outside.toPath())
            val result = LocalScopedSearch.find(root, FilenameGlob("*.txt"), recursive = true)
            assertEquals(listOf("ok.txt"), result.matches.map { it.name })
            assertFalse(result.unavailable)
            assertTrue(LocalScopedSearch.find(File(root, "outside"),
                FilenameGlob("*"), recursive = true).unavailable)
        } finally {
            root.deleteRecursively()
            outside.deleteRecursively()
        }
    }

    @Test fun boundsVisitedEntriesAndResultCount() {
        val root = Files.createTempDirectory("ghostftp-search-bounds").toFile()
        try {
            repeat(80) { File(root, "file-%03d.txt".format(it)).writeText("x") }
            val result = LocalScopedSearch.find(root, FilenameGlob("*.txt"),
                recursive = true, maxVisited = 12, maxMatches = 8)
            assertTrue(result.truncated)
            assertTrue(result.visited <= 12)
            assertTrue(result.matches.size <= 8)
            assertFalse(result.unavailable)
        } finally {
            root.deleteRecursively()
        }
    }

    @Test fun cancelledAndMissingRootsFailClosed() {
        val root = Files.createTempDirectory("ghostftp-local-cancel").toFile()
        try {
            File(root, "test.txt").writeText("x")
            assertTrue(LocalScopedSearch.find(root, FilenameGlob("*"), recursive = true) { true }.cancelled)
            val unavailable = LocalScopedSearch.find(File(root, "missing"), FilenameGlob("*"),
                recursive = true)
            assertTrue(unavailable.unavailable)
            assertTrue(unavailable.matches.isEmpty())
        } finally {
            root.deleteRecursively()
        }
    }
}
