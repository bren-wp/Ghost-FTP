package com.ghostftp.android

import java.io.File
import java.nio.file.Files
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Remove private cloud download staging residue left if Android kills the
 * process before SignedCloudTransfer.download() reaches its finally block.
 *
 * Called once, synchronously at process startup before transfers are allowed.
 * Activity recreation in the same process must never delete a live transfer's
 * stage. Only the app's own narrowly named, regular cache files are eligible.
 * No SAF document or user-chosen destination is modified here.
 */
internal object CloudStageCleanup {
    private val cleanedThisProcess = AtomicBoolean(false)
    private val stageName = Regex("""^ghostftp-cloud-[A-Za-z0-9_-]{3,64}\.part$""")

    fun purgeOnProcessStart(cacheDir: File) {
        if (cleanedThisProcess.compareAndSet(false, true)) {
            purge(cacheDir)
        }
    }

    /** Intentionally package-private for deterministic JVM coverage. */
    internal fun purge(cacheDir: File): Int {
        val children = runCatching { cacheDir.listFiles() }.getOrNull() ?: return 0
        var removed = 0
        for (candidate in children) {
            if (!stageName.matches(candidate.name)) continue
            val deleted = runCatching {
                candidate.isFile &&
                    !Files.isSymbolicLink(candidate.toPath()) &&
                    candidate.delete()
            }.getOrDefault(false)
            if (deleted) removed += 1
        }
        return removed
    }
}
