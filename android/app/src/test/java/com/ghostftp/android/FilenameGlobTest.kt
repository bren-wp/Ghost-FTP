package com.ghostftp.android

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class FilenameGlobTest {
    @Test fun wildcardMatchingPreservesLiteralNames() {
        assertTrue(FilenameGlob("*.txt").matches("notes.txt"))
        assertFalse(FilenameGlob("*.txt").matches("notes.jpg"))
        assertTrue(FilenameGlob("readme.?d").matches("readme.md"))
        assertFalse(FilenameGlob("readme.?d").matches("readme.mmd"))
        assertTrue(FilenameGlob("[abc]").matches("[abc]"))
        assertFalse(FilenameGlob("[abc]").matches("a"))
        assertTrue(FilenameGlob("*").matches(""))
    }

    @Test fun matchingHonorsCaseSetting() {
        assertTrue(FilenameGlob("RÉSUMÉ.*", ignoreCase = true).matches("résumé.PDF"))
        assertFalse(FilenameGlob("Report.*", ignoreCase = false).matches("report.txt"))
        assertTrue(FilenameGlob("Report.*", ignoreCase = false).matches("Report.txt"))
    }

    @Test fun ambiguousWildcardsAreBoundedAndDoNotBecomeRegex() {
        val dangerous = "*a".repeat(38) + "*b"
        assertFalse(FilenameGlob(dangerous).matches("a".repeat(250)))
        assertTrue(FilenameGlob("*".repeat(80)).matches("x".repeat(10000)))
        assertFalse(FilenameGlob("file.(a+)+").matches("file.aaaa"))
    }

    @Test(expected = IllegalArgumentException::class)
    fun rejectsExcessivelyLongPattern() {
        FilenameGlob("x".repeat(81))
    }
}
