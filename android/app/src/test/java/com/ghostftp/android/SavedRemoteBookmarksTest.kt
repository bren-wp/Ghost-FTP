package com.ghostftp.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class SavedRemoteBookmarksTest {
    private fun bookmark(path: String = "/var/www/html", site: String = "Production") =
        SavedRemoteBookmark("Website Root", site, path)

    @Test fun acceptsAbsoluteDirectoriesWithoutInventingAccounts() {
        assertTrue(SavedRemoteBookmarks.valid(bookmark()))
        assertTrue(SavedRemoteBookmarks.valid(bookmark("/")))
        assertTrue(SavedRemoteBookmarks.valid(bookmark("/Računi 2026/folder")))
    }

    @Test fun rejectsTraversalMalformedAndProtocolUrls() {
        for (path in listOf("", "relative/path", "../secrets", "/a/../b", "/./file",
            "/a//b", "\\\\server\\share", "/a\\b", "sftp://example.com/path",
            "/a\nbad")) {
            assertFalse("Rejected unsafe directory: $path", SavedRemoteBookmarks.valid(bookmark(path)))
        }
    }

    @Test fun addIsBoundedAndDeduplicatedPerSiteAndPath() {
        val original = listOf(bookmark())
        assertNull(SavedRemoteBookmarks.added(original, bookmark()))
        assertNull(SavedRemoteBookmarks.added(original, bookmark(site = "production")))
        val second = bookmark(path = "/another")
        assertEquals(listOf(bookmark(), second), SavedRemoteBookmarks.added(original, second))
        assertEquals(1, original.size)
        assertNull(SavedRemoteBookmarks.added(List(80) { bookmark("/path$it") }, second))
    }

    @Test fun removalIsExactAndNeverErasesUnrelatedEntries() {
        val original = listOf(bookmark(), bookmark(path = "/other", site = "Backup"))
        assertEquals(listOf(original.last()), SavedRemoteBookmarks.without(original, original.first()))
        assertNull(SavedRemoteBookmarks.without(original, bookmark("/unknown")))
        assertEquals(2, original.size)
    }

    @Test fun rejectUntrustedNamesAndMalformedExistingCollection() {
        assertFalse(SavedRemoteBookmarks.valid(bookmark().copy(title = "Bad\nlabel")))
        assertNull(SavedRemoteBookmarks.added(listOf(bookmark().copy(siteName = "")), bookmark("/new")))
        assertNull(SavedRemoteBookmarks.without(
            listOf(bookmark().copy(remoteDirectory = "/../secret")), bookmark()
        ))
    }
}
