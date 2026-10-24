package com.ghostftp.android

import java.io.File
import java.nio.file.Files
import java.nio.file.LinkOption
import java.util.ArrayDeque
import java.util.Locale

internal data class LocalSearchResult(
    val matches: List<File>,
    val visited: Int,
    val truncated: Boolean,
    val unavailable: Boolean,
    val cancelled: Boolean
)

/**
 * Search only the app-owned local root, without following symlinks or
 * allocating an unbounded File[] per directory. This can be slow on mounted
 * storage and MUST be called off Android's main thread.
 */
internal object LocalScopedSearch {
    fun find(
        base: File,
        pattern: Regex,
        recursive: Boolean,
        maxVisited: Int = 1000,
        maxMatches: Int = 150,
        maxDepth: Int = 4,
        isCancelled: () -> Boolean = { false }
    ): LocalSearchResult {
        require(maxVisited in 1..10000 && maxMatches in 1..500 && maxDepth in 0..8)
        fun cancelled() = LocalSearchResult(emptyList(), 0, false, false, true)
        fun unavailable() = LocalSearchResult(emptyList(), 0, false, true, false)
        if (isCancelled()) return cancelled()

        return runCatching {
            if (Files.isSymbolicLink(base.toPath())) return unavailable()
            val root = base.canonicalFile
            if (!root.isDirectory || !root.canRead()) return unavailable()
            val prefix = root.path + File.separator
            val queue = ArrayDeque<Pair<File, Int>>()
            val seen = HashSet<String>()
            val matches = ArrayList<File>(minOf(maxMatches, 150))
            queue.add(root to 0)
            seen.add(root.path)
            var visited = 0
            var truncated = false

            while (queue.isNotEmpty() && visited < maxVisited && matches.size < maxMatches) {
                if (isCancelled()) return cancelled()
                val (directory, depth) = queue.removeFirst()
                Files.newDirectoryStream(directory.toPath()).use { entries ->
                    for (entry in entries) {
                        if (isCancelled()) return cancelled()
                        if (visited >= maxVisited || matches.size >= maxMatches) {
                            truncated = true
                            break
                        }
                        visited++
                        if (Files.isSymbolicLink(entry)) continue
                        val file = entry.toFile()
                        val canonical = file.canonicalFile
                        if (!canonical.path.startsWith(prefix)) continue
                        val isDirectory = Files.isDirectory(entry, LinkOption.NOFOLLOW_LINKS)
                        val isFile = Files.isRegularFile(entry, LinkOption.NOFOLLOW_LINKS)
                        if (!isDirectory && !isFile) continue
                        if (isDirectory && recursive && depth < maxDepth && seen.add(canonical.path)) {
                            queue.add(canonical to depth + 1)
                        }
                        if (pattern.matches(file.name)) matches.add(canonical)
                    }
                }
            }
            if (queue.isNotEmpty() || visited >= maxVisited || matches.size >= maxMatches) {
                truncated = true
            }
            matches.sortWith(compareBy({ it.name.lowercase(Locale.ROOT) }, { it.name }))
            LocalSearchResult(matches, visited, truncated, unavailable = false, cancelled = false)
        }.getOrElse {
            // An unreadable/revoked directory must never be presented as a
            // complete empty result or leak partially traversed paths.
            unavailable()
        }
    }
}
