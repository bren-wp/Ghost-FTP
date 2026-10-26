package com.ghostftp.android

/**
 * Match user-entered filename globs without compiling untrusted input into a
 * regular expression. '*' matches any sequence; '?' matches one character.
 * The bounded greedy/backtracking matcher uses O(pattern.length * name.length)
 * time in the worst case, avoiding catastrophic regex backtracking.
 */
internal class FilenameGlob(pattern: String, private val ignoreCase: Boolean = true) {
    private val wildcard: String = buildString(pattern.length) {
        for (ch in pattern) {
            if (ch != '*' || lastOrNull() != '*') append(ch)
        }
    }

    init {
        require(pattern.isNotBlank() && pattern.length <= 80) {
            "Filename search pattern must have 1 to 80 characters"
        }
    }

    fun matches(name: String): Boolean {
        var nameIndex = 0
        var patternIndex = 0
        var lastStarIndex = -1
        var retryNameIndex = 0

        while (nameIndex < name.length) {
            when {
                patternIndex < wildcard.length &&
                    (wildcard[patternIndex] == '?' ||
                        wildcard[patternIndex].equals(name[nameIndex], ignoreCase)) -> {
                    patternIndex++
                    nameIndex++
                }
                patternIndex < wildcard.length && wildcard[patternIndex] == '*' -> {
                    lastStarIndex = patternIndex++
                    retryNameIndex = nameIndex
                }
                lastStarIndex != -1 -> {
                    patternIndex = lastStarIndex + 1
                    nameIndex = ++retryNameIndex
                }
                else -> return false
            }
        }
        while (patternIndex < wildcard.length && wildcard[patternIndex] == '*') {
            patternIndex++
        }
        return patternIndex == wildcard.length
    }
}
