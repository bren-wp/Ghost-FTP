package com.ghostftp.android

import android.content.ContentResolver
import android.net.Uri
import android.provider.OpenableColumns
import java.io.File
import java.io.FileOutputStream
import java.io.IOException
import java.io.InputStream
import java.net.URI
import java.net.URL
import java.security.MessageDigest
import java.util.Base64
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
            // Do not let IDNA/punycode, unusual Unicode or malformed DNS
            // labels broaden the signed provider allowlist. This does not
            // replace PublicCloudDns validation on the actual HTTP lookup.
            if (host.length !in 4..253 ||
                host.any { ch -> ch !in 'a'..'z' && ch !in '0'..'9' && ch != '-' && ch != '.' } ||
                host.split('.').any { label ->
                    label.isEmpty() || label.length > 63 || label.startsWith('-') ||
                        label.endsWith('-') || label.startsWith("xn--")
                }) return false
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
    private const val MAX_EMPTY_READS = 8
    private val BINARY_TYPE: MediaType = "application/octet-stream".toMediaType()

    /**
     * Copy data from a potentially untrusted provider/network stream. Some
     * broken SAF providers return zero bytes indefinitely instead of EOF,
     * while others report a file size that changes mid-copy. Neither case
     * may produce an apparent successful transfer or an unbounded loop.
     *
     * This bounds *zero-progress returns*, not a read that blocks inside
     * the provider; real-device cancellation and lifecycle proof remains
     * separately required by Issue #127.
     */
    internal fun streamWithBoundedProgress(
        input: InputStream,
        maximumBytes: Long,
        expectedBytes: Long?,
        cancellation: CloudTransferCancellation,
        writeChunk: (ByteArray, Int) -> Unit
    ): Long {
        if (maximumBytes !in 0..MAX_BYTES ||
            (expectedBytes != null && expectedBytes !in 0..maximumBytes)) {
            throw IOException("Cloud transfer size metadata is invalid.")
        }
        val buffer = ByteArray(64 * 1024)
        var copied = 0L
        var emptyReads = 0
        while (true) {
            cancellation.ensureActive()
            val read = input.read(buffer)
            // A blocking SAF read can return bytes after Cancel was pressed.
            // Never forward those late bytes into a destination or HTTP sink.
            cancellation.ensureActive()
            if (read < 0) break
            if (read == 0) {
                if (++emptyReads > MAX_EMPTY_READS) {
                    throw IOException("Cloud transfer source did not make read progress.")
                }
                continue
            }
            emptyReads = 0
            if (read.toLong() > maximumBytes - copied) {
                throw IOException("Cloud transfer source exceeded its permitted size.")
            }
            writeChunk(buffer, read)
            copied += read
        }
        cancellation.ensureActive()
        if (expectedBytes != null && copied != expectedBytes) {
            throw IOException("Cloud transfer source changed or was truncated.")
        }
        return copied
    }

    // S3 can return its SHA-256 object checksum with a presigned GET.
    // Verify before opening the SAF destination. ETag is not an object
    // checksum for multipart uploads and must not be treated as one.
    internal fun requireMatchingSha256Header(header: String?, computedDigest: ByteArray) {
        if (header == null) return
        val expected = try {
            Base64.getDecoder().decode(header.trim())
        } catch (_: IllegalArgumentException) {
            throw IOException("Cloud provider returned an invalid SHA-256 checksum.")
        }
        if (expected.size != 32 || !MessageDigest.isEqual(expected, computedDigest)) {
            throw IOException("Cloud download checksum mismatch; destination was not written.")
        }
    }

    /**
     * SAF providers may acknowledge a successful OutputStream.close() while
     * silently truncating or altering the persisted document. Read the exact
     * destination back after close before reporting success. Refuse providers
     * that cannot offer a readable committed document: integrity takes
     * precedence over an optimistic success notification.
     */
    internal fun verifyPersistedCopy(
        input: InputStream,
        expectedBytes: Long,
        expectedDigest: ByteArray,
        cancellation: CloudTransferCancellation
    ) {
        if (expectedBytes !in 0..MAX_BYTES || expectedDigest.size != 32) {
            throw IOException("Cloud download integrity metadata is invalid.")
        }
        val digest = MessageDigest.getInstance("SHA-256")
        // Share the exact same bounded, cancel-aware stream policy used by
        // uploads and downloads. SAF read-back must not silently diverge.
        streamWithBoundedProgress(input, expectedBytes, expectedBytes, cancellation) { bytes, count ->
            digest.update(bytes, 0, count)
        }
        if (!MessageDigest.isEqual(digest.digest(), expectedDigest)) {
            throw IOException("Saved cloud document failed the read-back integrity check.")
        }
    }

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
                resolver.openInputStream(source)?.use { input ->
                    cancellation.watchStream(input)
                    try {
                        streamWithBoundedProgress(
                            input, length, length, cancellation
                        ) { bytes, count -> sink.write(bytes, 0, count) }
                    } finally {
                        cancellation.unwatchStream(input)
                    }
                } ?: throw IOException("Selected document is unavailable.")
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
        // The stage must be removed even if request/client setup fails before
        // a network call has been attached to the cancellation token.
        try {
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
                val (expectedBytes, expectedDigest) = call.execute().use { response ->
                    cancellation.ensureActive()
                    checkResponse(response.code, isDownload = true)
                    if (response.header("Content-Encoding")?.let { !it.equals("identity", ignoreCase = true) } == true) {
                        throw IOException("Cloud provider returned an unexpected encoded object.")
                    }
                    val body = response.body ?: throw IOException("Cloud provider returned no response body.")
                    if (body.contentLength() > MAX_BYTES) {
                        throw IOException("Cloud downloads are limited to 1 GiB per signed URL.")
                    }
                    val contentDigest = MessageDigest.getInstance("SHA-256")
                    val received = body.byteStream().use { input ->
                        FileOutputStream(stage).use { output ->
                            streamWithBoundedProgress(
                                input, MAX_BYTES, body.contentLength().takeIf { it >= 0 }, cancellation
                            ) { bytes, count ->
                                output.write(bytes, 0, count)
                                contentDigest.update(bytes, 0, count)
                            }
                        }
                    }
                    val digest = contentDigest.digest()
                    requireMatchingSha256Header(response.header("x-amz-checksum-sha256"), digest)
                    received to digest
                }
                // SAF destinations are generally not atomic. The caller must
                // remove a newly created destination if this final copy fails.
                resolver.openOutputStream(target, "w")?.use { output ->
                    cancellation.watchStream(output)
                    try {
                        stage.inputStream().use { input ->
                            streamWithBoundedProgress(
                                input, expectedBytes, expectedBytes, cancellation
                            ) { bytes, count -> output.write(bytes, 0, count) }
                            output.flush()
                        }
                    } finally {
                        cancellation.unwatchStream(output)
                    }
                } ?: throw IOException("Selected download destination is not writable.")
                cancellation.ensureActive()
                resolver.openInputStream(target)?.use { saved ->
                    cancellation.watchStream(saved)
                    try {
                        verifyPersistedCopy(saved, expectedBytes, expectedDigest, cancellation)
                    } finally {
                        cancellation.unwatchStream(saved)
                    }
                } ?: throw IOException("Cloud destination cannot be read back for integrity verification.")
                cancellation.ensureActive()
            } finally {
                cancellation.clear(call)
                client.connectionPool.evictAll()
            }
        } finally {
            stage.delete()
        }
    }
}
