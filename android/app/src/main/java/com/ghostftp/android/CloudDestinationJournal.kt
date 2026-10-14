package com.ghostftp.android

import java.io.File
import java.io.FileOutputStream
import java.io.IOException
import java.net.URI
import java.nio.file.Files

/**
 * Records ONLY the SAF content URI for a newly created download document.
 * Stored in the app's noBackupFilesDir, never Bundle, logs or Android Backup.
 * No signed cloud URL, token, source path, or credentials are persisted.
 *
 * After process death the SAF provider may have persisted only a partial
 * document. Never automatically delete an uncertain file: require the user
 * to choose cleanup or to keep it. This journal is a recovery hint, not a
 * promise that third-party providers will grant access after restart.
 */
internal object CloudDestinationJournal {
    private const val FILE_NAME = "pending-cloud-download.uri"
    private const val TEMP_NAME = "pending-cloud-download.tmp"
    private const val MAX_URI_BYTES = 4096

    internal fun validDocumentUri(value: String): Boolean {
        if (value.toByteArray(Charsets.UTF_8).size !in 1..MAX_URI_BYTES) return false
        val uri = try { URI(value) } catch (_: Exception) { return false }
        return uri.scheme == "content" &&
            !uri.host.isNullOrBlank() &&
            uri.userInfo == null && uri.rawQuery == null && uri.rawFragment == null &&
            uri.rawPath?.isNotBlank() == true &&
            !value.contains('\n') && !value.contains('\r')
    }

    @Synchronized
    @Throws(IOException::class)
    fun record(noBackupDir: File, documentUri: String) {
        if (!validDocumentUri(documentUri)) {
            throw IOException("Cloud recovery cannot safely record this document.")
        }
        if (!noBackupDir.isDirectory && !noBackupDir.mkdirs()) {
            throw IOException("Cloud recovery storage is unavailable.")
        }
        val target = File(noBackupDir, FILE_NAME)
        val temporary = File(noBackupDir, TEMP_NAME)
        // Never replace an unresolved journal: two overlapping SAF results
        // could otherwise cause the first incomplete document to be lost.
        if (target.exists() || Files.isSymbolicLink(target.toPath()) ||
            Files.isSymbolicLink(temporary.toPath())) {
            throw IOException("Previous cloud recovery must be resolved first.")
        }
        try {
            FileOutputStream(temporary).use { output ->
                output.write(documentUri.toByteArray(Charsets.UTF_8))
                output.fd.sync()
            }
            if (!temporary.renameTo(target)) {
                throw IOException("Cloud recovery record could not be committed.")
            }
        } finally {
            temporary.delete()
        }
    }

    /**
     * A corrupt, truncated or tampered journal is still an unresolved
     * recovery condition. Do not silently allow a new transfer to replace it.
     */
    @Synchronized
    fun hasUnresolved(noBackupDir: File): Boolean {
        val file = File(noBackupDir, FILE_NAME)
        return file.exists() || Files.isSymbolicLink(file.toPath())
    }

    @Synchronized
    fun pending(noBackupDir: File): String? {
        val file = File(noBackupDir, FILE_NAME)
        if (!file.isFile || Files.isSymbolicLink(file.toPath()) ||
            file.length() !in 1..MAX_URI_BYTES.toLong()) return null
        val content = runCatching { file.readText(Charsets.UTF_8) }.getOrNull() ?: return null
        return content.takeIf(::validDocumentUri)
    }

    @Synchronized
    fun clear(noBackupDir: File): Boolean {
        val file = File(noBackupDir, FILE_NAME)
        return !file.exists() || (!Files.isSymbolicLink(file.toPath()) && file.delete())
    }
}
