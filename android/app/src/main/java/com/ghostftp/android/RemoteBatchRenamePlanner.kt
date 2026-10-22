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
            "Pattern needs one group of 1–6 # number placeholders."
        }
        val digits = groups.single().value.length
        val canon = HashSet<String>()
        val proposals = paths.sorted().mapIndexed { offset, from ->
            require(from.startsWith('/') && from != "/" && !from.contains('\u0000')) {
                "Refusing unsafe remote source path."
            }
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
            val normalized = Normalizer.normalize(to, Normalizer.Form.NFC)
                .lowercase(Locale.ROOT)
            require(canon.add(normalized)) { "Two files would have the same destination." }
            RemoteBatchRename(from, to)
        }
        // Avoid dependency/order cycles; protect even servers where RNFR/RNTO
        // accepts renaming over a just-renamed source from this same batch.
        val sources = paths.toSet()
        require(proposals.none { it.to in sources }) {
            "Destination overlaps a selected source; choose another pattern."
        }
        return proposals
    }
}
