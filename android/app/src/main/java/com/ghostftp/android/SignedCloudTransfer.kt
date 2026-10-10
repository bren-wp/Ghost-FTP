package com.ghostftp.android

import android.content.ContentResolver
import android.net.Uri
import android.provider.OpenableColumns
import java.io.File
import java.io.FileOutputStream
import java.io.IOException
import java.io.InterruptedIOException
import java.net.InetAddress
import java.net.URI
import java.net.URL
import javax.net.ssl.HttpsURLConnection

/**
 * Signed S3/R2/GCS object URLs and Azure SAS links, without account passwords,
 * embedded API keys, telemetry or a Ghost FTP server in the data path.
 * URLs are short-lived bearer credentials and must never be persisted or logged.
 */
internal class SignedCloudLink(raw: String) {
    val destination: URL
    val publicHost: String

    init {
        require(raw.toByteArray(Charsets.UTF_8).size <= 8192) { "Signed cloud link is too long." }
        val uri = try { URI(raw) } catch (_: Exception) { throw IllegalArgumentException("Invalid signed cloud URL.") }
        val host = uri.host?.lowercase()
        val signed = uri.rawQuery?.split('&')?.any {
            it.substringBefore('=').lowercase() in setOf("x-amz-signature", "x-goog-signature", "sig")
        } ?: false
        require(uri.scheme?.lowercase() == "https" &&
            !host.isNullOrBlank() && uri.userInfo == null && uri.fragment == null &&
            (uri.port == -1 || uri.port == 443) && signed) {
            "Use a short-lived HTTPS PUT/GET link signed by your cloud provider."
        }
        require(isPublicHost(host)) { "Local or private cloud endpoints are not permitted." }
        destination = uri.toURL()
        publicHost = host
    }

    companion object {
        internal fun isPublicHost(host: String): Boolean {
            if (host == "localhost" || host.endsWith(".localhost") ||
                host.endsWith(".local") || host.endsWith(".internal") ||
                host == "::1" || host.startsWith("fe80:") ||
                host.startsWith("fc") || host.startsWith("fd")) return false
            // Public cloud endpoints use DNS names; never accept direct IPs.
            if (host.contains(':') || host.all { it.isDigit() || it == '.' }) return false
            return host.contains('.')
        }
    }

    /** Verify DNS resolution before cloud data is sent to a user-supplied host. */
    fun verifyResolvedAddresses() {
        val addresses = InetAddress.getAllByName(publicHost)
        val safe = addresses.isNotEmpty() && addresses.all { address ->
            !address.isAnyLocalAddress && !address.isLoopbackAddress &&
                !address.isLinkLocalAddress && !address.isSiteLocalAddress &&
                !address.isMulticastAddress
        }
        require(safe) { "Cloud URL resolved to a private network address." }
    }
}

internal object SignedCloudTransfer {
    private const val MAX_BYTES = 1024L * 1024L * 1024L

    private fun open(link: SignedCloudLink, verb: String): HttpsURLConnection {
        link.verifyResolvedAddresses()
        return (link.destination.openConnection() as HttpsURLConnection).apply {
            requestMethod = verb
            instanceFollowRedirects = false
            connectTimeout = 20000
            readTimeout = 120000
            useCaches = false
            setRequestProperty("Cache-Control", "no-store")
        }
    }

    private fun ensureSuccess(connection: HttpsURLConnection) {
        val status = connection.responseCode
        // A signed GET must return the complete object. HTTP 206 without an
        // explicit Range request is not a safe replacement for an existing file.
        if (status !in 200..299 || (connection.requestMethod == "GET" && status != 200)) {
            throw IOException("Cloud provider rejected the request (HTTP $status). Check link expiry and permissions.")
        }
    }

    fun upload(resolver: ContentResolver, source: Uri, link: SignedCloudLink) {
        val length = resolver.query(source, arrayOf(OpenableColumns.SIZE), null, null, null)?.use {
            if (it.moveToFirst() && !it.isNull(0)) it.getLong(0) else null
        } ?: throw IOException("Selected document size is unavailable; a fixed-length signed upload is required.")
        require(length in 0..MAX_BYTES) { "Cloud uploads are limited to 1 GiB per signed URL." }
        val connection = open(link, "PUT")
        try {
            connection.doOutput = true
            connection.setFixedLengthStreamingMode(length)
            connection.setRequestProperty("Content-Type", "application/octet-stream")
            if (link.publicHost.endsWith(".blob.core.windows.net")) {
                connection.setRequestProperty("x-ms-blob-type", "BlockBlob")
            }
            var copied = 0L
            resolver.openInputStream(source)?.use { input ->
                connection.outputStream.use { output ->
                    val buffer = ByteArray(64 * 1024)
                    while (true) {
                        if (Thread.currentThread().isInterrupted) throw InterruptedIOException("Cloud upload canceled.")
                        val read = input.read(buffer)
                        if (read == -1) break
                        copied += read
                        if (copied > length) throw IOException("Selected document changed during upload.")
                        output.write(buffer, 0, read)
                    }
                }
            } ?: throw IOException("The selected document is unavailable.")
            if (copied != length) throw IOException("Selected document changed during upload.")
            ensureSuccess(connection)
        } finally {
            connection.disconnect()
        }
    }

    fun download(resolver: ContentResolver, target: Uri, link: SignedCloudLink, cache: File) {
        val connection = open(link, "GET")
        val stage = File.createTempFile("ghostftp-cloud-", ".part", cache)
        try {
            ensureSuccess(connection)
            if (connection.contentLengthLong > MAX_BYTES) {
                throw IOException("Cloud downloads are limited to 1 GiB per signed URL.")
            }
            var count = 0L
            connection.inputStream.use { input ->
                FileOutputStream(stage).use { output ->
                    val buffer = ByteArray(64 * 1024)
                    while (true) {
                        if (Thread.currentThread().isInterrupted) throw InterruptedIOException("Cloud download canceled.")
                        val read = input.read(buffer)
                        if (read == -1) break
                        count += read
                        if (count > MAX_BYTES) throw IOException("Cloud download exceeded its 1 GiB safety limit.")
                        output.write(buffer, 0, read)
                    }
                }
            }
            if (connection.contentLengthLong >= 0 && count != connection.contentLengthLong) {
                throw IOException("Cloud download was incomplete.")
            }
            resolver.openOutputStream(target, "w")?.use { output ->
                stage.inputStream().use { input -> input.copyTo(output, 64 * 1024) }
            } ?: throw IOException("Selected download destination is not writable.")
        } finally {
            stage.delete()
            connection.disconnect()
        }
    }
}
