package com.ghostftp.android

import java.text.Normalizer
import java.util.Locale

internal data class RemoteBatchRename(val from: String, val to: String)

/**
 * No I/O: produce a complete preview before any destructive remote rename.
 * The UI must confirm the exact mappings, and the transport must refuse an
 * already-present destination. Never silently overwrite an existing target.
 */
internal object RemoteBatchRenamePlanner {
    const val MAX_FILES = 12

    private fun comparablePath(path: String): String =
        Normalizer.normalize(path, Normalizer.Form.NFC).lowercase(Locale.ROOT)

    /**
     * Check every remote object reported by the fresh directory listing,
     * including directories, before starting a destructive rename. Servers
     * may compare paths without case or Unicode normalization distinctions.
     */
    fun hasExistingDestinationCollision(
        plan: List<RemoteBatchRename>,
        listedPaths: Collection<String>
    ): Boolean {
        val occupied = listedPaths.mapTo(HashSet<String>()) { comparablePath(it) }
        return plan.any { comparablePath(it.to) in occupied }
    }

    fun plan(paths: List<String>, template: String, start: Int): List<RemoteBatchRename> {
        require(paths.isNotEmpty() && paths.size <= MAX_FILES) {
            "Select between 1 and $MAX_FILES real remote files."
        }
        require(start in 0..999_999) { "Start number must be between 0 and 999999." }
        require(template.length in 2..80 && template.none {
            it == '/' || it == '\\' || it.isISOControl()
        }) { "Pattern must be a safe filename without directory separators." }
        val groups = Regex("#+").findAll(template).toList()
        require(groups.size == 1 && groups.single().value.length in 1..6) {
            "Pattern must contain exactly one group of 1–6 number markers (#)."
        }
        val digits = groups.single().value.length
        // '#' is an explicit fixed-width number field. Refuse a batch when
        // its last number would exceed that width rather than silently
        // producing a longer, surprising filename mid-operation.
        val finalOrdinal = start.toLong() + paths.size - 1
        val maxOrdinal = (1..digits).fold(1L) { total, _ -> total * 10L } - 1
        require(finalOrdinal <= maxOrdinal) {
            "The batch numbering exceeds the # field width; add more # markers."
        }
        val sources = HashSet<String>()
        require(paths.all { from ->
            from.startsWith('/') && from != "/" && from.split('/').drop(1).all { segment ->
                segment.isNotEmpty() && segment != "." && segment != ".." &&
                    segment.none { it.isISOControl() || it == '\\' }
            }
        }) { "Refusing unsafe remote source path." }
        require(paths.all { sources.add(comparablePath(it)) }) {
            "Duplicate or equivalent remote source paths are not allowed."
        }
        val canon = HashSet<String>()
        val proposals = paths.sorted().mapIndexed { offset, from ->
            val name = from.substringAfterLast('/')
            require(name.isNotBlank() && name !in setOf(".", "..")) {
                "Refusing invalid remote filename."
            }
            val ext = name.lastIndexOf('.').takeIf { it > 0 }
                ?.let { name.substring(it) }.orEmpty()
            val ordinal = (start.toLong() + offset).toString().padStart(digits, '0')
            val generated = template.replaceRange(groups.single().range, ordinal)
            val destinationName = if ('.' in generated) generated else generated + ext
            require(destinationName.isNotBlank() && destinationName !in setOf(".", "..") &&
                ".." !in destinationName && destinationName.length <= 255) {
                "Generated remote file name is unsafe."
            }
            val parent = from.substringBeforeLast('/')
            val to = (if (parent.isBlank()) "" else parent) + "/" + destinationName
            require(to != from) { "At least one generated name is unchanged." }
            require(canon.add(comparablePath(to))) { "Two files would have the same destination." }
            RemoteBatchRename(from, to)
        }
        // Avoid dependency/order cycles; protect even servers where RNFR/RNTO
        // accepts renaming over a just-renamed source from this same batch.
        // On case-insensitive or Unicode-normalizing FTP/SFTP volumes, a
        // destination may overwrite another selected source even though the
        // raw Unicode codepoints or filename case differ. Reject conservatively.
        val sourceNames = paths.map(::comparablePath).toSet()
        require(proposals.none { comparablePath(it.to) in sourceNames }) {
            "Destination overlaps a selected source; choose another pattern."
        }
        return proposals
    }
}
