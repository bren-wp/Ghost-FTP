package com.ghostftp.android

import java.io.File
import java.io.IOException
import java.nio.file.Files
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CloudDestinationJournalTest {
    private fun withDirectory(block: (File) -> Unit) {
        val root = Files.createTempDirectory("ghostftp-recovery-").toFile()
        try { block(root) } finally { root.deleteRecursively() }
    }

    @Test fun interruptedDownloadSurvivesRestartWithoutSavingSignedUrl() = withDirectory { root ->
        val uri = "content://com.example.documents/document/transfer%3A42"
        CloudDestinationJournal.record(root, uri)
        assertEquals(uri, CloudDestinationJournal.pending(root))
        val bytes = File(root, "pending-cloud-download.uri").readText()
        assertEquals(uri, bytes)
        assertFalse("never store HTTPS signed URL", bytes.contains("X-Amz-Signature"))
        assertTrue(CloudDestinationJournal.clear(root))
        assertNull(CloudDestinationJournal.pending(root))
    }

    @Test fun rejectsBearerUrlsQueriesFragmentsAndMalformedContentUris() = withDirectory { root ->
        for (bad in listOf(
            "https://storage.googleapis.com/bucket?X-Goog-Signature=secret",
            "content://com.example/doc?token=secret",
            "content://com.example/doc#fragment",
            "content://user:password@com.example/doc",
            "content:///doc",
            "file:///tmp/file",
            "content://com.example",
            "content://com.example/doc\nInjected"
        )) {
            assertFalse("reject untrusted URI form", CloudDestinationJournal.validDocumentUri(bad))
            try {
                CloudDestinationJournal.record(root, bad)
                throw AssertionError("record unexpectedly accepted invalid URI")
            } catch (_: IOException) {
                assertNull(CloudDestinationJournal.pending(root))
            }
        }
    }

    @Test fun refusesOversizedAndTruncatedRecoveryEntries() = withDirectory { root ->
        assertFalse(CloudDestinationJournal.validDocumentUri("content://example.com/doc/" + "x".repeat(4096)))
        val file = File(root, "pending-cloud-download.uri")
        file.writeText("corrupted")
        assertNull(CloudDestinationJournal.pending(root))
        file.writeText("x".repeat(5000))
        assertNull(CloudDestinationJournal.pending(root))
        assertTrue(CloudDestinationJournal.clear(root))
    }

    @Test fun incompleteWriteDoesNotReplacePreviouslyCommittedRecoveryHint() = withDirectory { root ->
        val prior = "content://com.example.documents/document/prior"
        CloudDestinationJournal.record(root, prior)
        val temp = File(root, "pending-cloud-download.tmp")
        val outside = Files.createTempFile("ghostftp-recovery-outside", ".txt")
        try {
            Files.createSymbolicLink(temp.toPath(), outside)
            try {
                CloudDestinationJournal.record(root, "content://com.example.documents/document/new")
                throw AssertionError("must not overwrite a symlink")
            } catch (_: IOException) {
                assertEquals(prior, CloudDestinationJournal.pending(root))
                assertEquals(0L, Files.size(outside))
            }
        } catch (_: UnsupportedOperationException) {
            org.junit.Assume.assumeTrue("Symlink unavailable", false)
        } finally {
            Files.deleteIfExists(outside)
        }
    }

    @Test fun clearingMissingJournalIsSafeAndIdempotent() = withDirectory { root ->
        assertTrue(CloudDestinationJournal.clear(root))
        assertNull(CloudDestinationJournal.pending(root))
    }
}
