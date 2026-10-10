package com.ghostftp.android

import android.content.ContentResolver
import android.net.Uri
import android.provider.OpenableColumns
import java.io.File
import java.io.FileOutputStream
import java.io.IOException
import java.net.URI
import java.net.URL
import java.util.Locale
import okhttp3.MediaType
import okhttp3.Request
import okhttp3.RequestBody
import okio.BufferedSink
import okhttp3.MediaType.Companion.toMediaType

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
        val host = uri.host?.lowercase(Locale.ROOT)
        require(uri.scheme?.lowercase(Locale.ROOT) == "https" &&
            !host.isNullOrBlank() && uri.userInfo == null && uri.fragment == null &&
            (uri.port == -1 || uri.port == 443)) {
            "Use a short-lived HTTPS PUT/GET link signed by your cloud provider."
        }
        require(isPublicHost(host)) { "Local or private cloud endpoints are not permitted." }
        require(isKnownProviderHost(host)) {
            "Signed cloud link must use a supported cloud storage provider domain."
        }
        val expectedSignature = when {
            host.endsWith(".blob.core.windows.net") -> "sig"
            host == "storage.googleapis.com" || host.endsWith(".storage.googleapis.com") -> "x-goog-signature"
            else -> "x-amz-signature"
        }
        val signatures = uri.rawQuery?.split('&')?.mapNotNull { field ->
            if (field.substringBefore('=').lowercase(Locale.ROOT) == expectedSignature) {
                field.substringAfter('=', "")
            } else null
        } ?: emptyList()
        require(signatures.size == 1 && signatures[0].isNotBlank()) {
            "The signed cloud link is missing a valid provider-specific signature."
        }
        destination = uri.toURL()
        publicHost = host
    }

    companion object {
        internal fun isKnownProviderHost(host: String): Boolean {
            if (host == "storage.googleapis.com" ||
                host.endsWith(".storage.googleapis.com") ||
                host.endsWith(".blob.core.windows.net") ||
                host.endsWith(".r2.cloudflarestorage.com") ||
                host.endsWith(".digitaloceanspaces.com")) return true
            if (host.endsWith(".amazonaws.com") ||
                host.endsWith(".backblazeb2.com") ||
                host.endsWith(".wasabisys.com") ||
                host.endsWith(".scw.cloud")) {
                return host.split('.').any { it == "s3" || it.startsWith("s3-") }
            }
            return false
        }

        internal fun isPublicHost(host: String): Boolean {
            if (host == "localhost" || host.endsWith(".localhost") ||
                host.endsWith(".local") || host.endsWith(".internal") ||
                host == "::1" || host.startsWith("fe80:") ||
                host.startsWith("fc") || host.startsWith("fd")) return false
            if (host.contains(':') || host.all { it.isDigit() || it == '.' }) return false
            return host.contains('.')
        }
    }
}

internal object SignedCloudTransfer {
    private const val MAX_BYTES = 1024L * 1024L * 1024L
    private val BINARY_TYPE: MediaType = "application/octet-stream".toMediaType()

    private fun checkResponse(status: Int, isDownload: Boolean) {
        if (status !in 200..299 || (isDownload && status != 200)) {
            throw IOException("Cloud provider rejected the request (HTTP $status). Check link expiry and permissions.")
        }
    }

    fun upload(
        resolver: ContentResolver,
        source: Uri,
        link: SignedCloudLink,
        cancellation: CloudTransferCancellation = CloudTransferCancellation()
    ) {
        cancellation.ensureActive()
        val length = resolver.query(source, arrayOf(OpenableColumns.SIZE), null, null, null)?.use {
            if (it.moveToFirst() && !it.isNull(0)) it.getLong(0) else null
        } ?: throw IOException("Selected document size is unavailable; a fixed-length signed upload is required.")
        require(length in 0..MAX_BYTES) { "Cloud uploads are limited to 1 GiB per signed URL." }
        val body = object : RequestBody() {
            override fun contentType(): MediaType = BINARY_TYPE
            override fun contentLength(): Long = length

            override fun writeTo(sink: BufferedSink) {
                cancellation.ensureActive()
                var copied = 0L
                resolver.openInputStream(source)?.use { input ->
                    val buffer = ByteArray(64 * 1024)
                    while (true) {
                        cancellation.ensureActive()
                        val read = input.read(buffer)
                        if (read == -1) break
                        copied += read
                        if (copied > length) throw IOException("Selected document changed during upload.")
                        sink.write(buffer, 0, read)
                    }
                } ?: throw IOException("Selected document is unavailable.")
                if (copied != length) throw IOException("Selected document changed during upload.")
                cancellation.ensureActive()
            }
        }
        val builder = Request.Builder()
            .url(link.destination)
            .header("Cache-Control", "no-store")
            .header("Accept-Encoding", "identity")
            .put(body)
        if (link.publicHost.endsWith(".blob.core.windows.net")) {
            builder.header("x-ms-blob-type", "BlockBlob")
        }
        val client = CloudHttpClient.create()
        val call = client.newCall(builder.build())
        cancellation.attach(call)
        try {
            cancellation.ensureActive()
            call.execute().use { response ->
                cancellation.ensureActive()
                checkResponse(response.code, isDownload = false)
            }
        } finally {
            cancellation.clear(call)
            client.connectionPool.evictAll()
        }
    }

    fun download(
        resolver: ContentResolver,
        target: Uri,
        link: SignedCloudLink,
        cache: File,
        cancellation: CloudTransferCancellation = CloudTransferCancellation()
    ) {
        cancellation.ensureActive()
        val stage = File.createTempFile("ghostftp-cloud-", ".part", cache)
        val request = Request.Builder()
            .url(link.destination)
            .header("Cache-Control", "no-store")
            .header("Accept-Encoding", "identity")
            .get()
            .build()
        val client = CloudHttpClient.create()
        val call = client.newCall(request)
        try {
            cancellation.attach(call)
            call.execute().use { response ->
                cancellation.ensureActive()
                checkResponse(response.code, isDownload = true)
                if (response.header("Content-Encoding")?.let { !it.equals("identity", ignoreCase = true) } == true) {
                    throw IOException("Cloud provider returned an unexpected encoded object.")
                }
                val body = response.body ?: throw IOException("Cloud provider returned no response body.")
                if (body.contentLength() > MAX_BYTES) {
                    throw IOException("Cloud downloads are limited to 1 GiB per signed URL.")
                }
                var received = 0L
                body.byteStream().use { input ->
                    FileOutputStream(stage).use { output ->
                        val buffer = ByteArray(64 * 1024)
                        while (true) {
                            cancellation.ensureActive()
                            val read = input.read(buffer)
                            if (read == -1) break
                            received += read
                            if (received > MAX_BYTES) throw IOException("Cloud download exceeded its 1 GiB safety limit.")
                            output.write(buffer, 0, read)
                        }
                    }
                }
                cancellation.ensureActive()
                if (body.contentLength() >= 0 && received != body.contentLength()) {
                    throw IOException("Cloud download was incomplete.")
                }
            }
            // SAF destinations are generally not atomic. The caller must
            // remove a newly created destination if this final copy fails.
            resolver.openOutputStream(target, "w")?.use { output ->
                stage.inputStream().use { input ->
                    val buffer = ByteArray(64 * 1024)
                    while (true) {
                        cancellation.ensureActive()
                        val read = input.read(buffer)
                        if (read == -1) break
                        output.write(buffer, 0, read)
                    }
                    output.flush()
                }
            } ?: throw IOException("Selected download destination is not writable.")
            cancellation.ensureActive()
        } finally {
            cancellation.clear(call)
            stage.delete()
            client.connectionPool.evictAll()
        }
    }
}
