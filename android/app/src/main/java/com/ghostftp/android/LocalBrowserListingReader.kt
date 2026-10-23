package com.ghostftp.android

import java.io.File
import java.nio.file.Files
import java.util.Locale

internal data class LocalBrowserListing(
    val files: List<File>,
    val truncated: Boolean,
    val unavailable: Boolean
)

/**
 * Listing an app-private directory must not enumerate unbounded entries on
 * Android's main thread. Bound BOTH the memory retained and number of
 * DirectoryStream entries visited. Never follow symlink directories or an
 * entry whose canonical path escapes the app-owned root.
 */
internal object LocalBrowserListingReader {
    private val filenameOrder = compareBy<File>(
        { !it.isDirectory },
        { it.name.lowercase(Locale.ROOT) },
        { it.name }
    )

    fun list(
        base: File,
        requested: File,
        displayedLimit: Int = 150,
        scanLimit: Int = 2048
    ): LocalBrowserListing {
        require(displayedLimit in 1..500)
        require(scanLimit >= displayedLimit && scanLimit <= 10000)
        return runCatching {
            val root = base.canonicalFile
            val dir = requested.canonicalFile
            val prefix = root.path + File.separator
            if (dir.path != root.path && !dir.path.startsWith(prefix)) {
                return LocalBrowserListing(emptyList(), truncated = false, unavailable = true)
            }
            if (!dir.isDirectory || !dir.canRead() || Files.isSymbolicLink(requested.toPath())) {
                return LocalBrowserListing(emptyList(), truncated = false, unavailable = true)
            }
            val candidates = ArrayList<File>(minOf(displayedLimit, 150))
            var visited = 0
            var truncated = false
            Files.newDirectoryStream(dir.toPath()).use { entries ->
                for (entry in entries) {
                    if (visited >= scanLimit) {
                        truncated = true
                        break
                    }
                    visited++
                    val file = entry.toFile()
                    val safe = runCatching {
                        !Files.isSymbolicLink(entry) &&
                            !file.isHidden && file.canonicalPath.startsWith(prefix) &&
                            (file.isDirectory || file.isFile)
                    }.getOrDefault(false)
                    if (!safe) continue
                    candidates.add(file)
                }
            }
            candidates.sortWith(filenameOrder)
            if (candidates.size > displayedLimit) truncated = true
            LocalBrowserListing(candidates.take(displayedLimit), truncated, unavailable = false)
        }.getOrElse {
            // Revoked storage permissions, unmounted media and DirectoryStream
            // I/O races must not unwind into an Android Activity crash.
            LocalBrowserListing(emptyList(), truncated = false, unavailable = true)
        }
    }
}
