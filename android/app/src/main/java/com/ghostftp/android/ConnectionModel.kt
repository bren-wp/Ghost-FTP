package com.ghostftp.android

import com.jcraft.jsch.ChannelSftp
import com.jcraft.jsch.HostKey
import com.jcraft.jsch.HostKeyRepository
import com.jcraft.jsch.JSch
import com.jcraft.jsch.UserInfo
import org.apache.commons.net.ftp.FTP
import org.apache.commons.net.ftp.FTPClient
import org.apache.commons.net.ftp.FTPFile
import org.apache.commons.net.ftp.FTPReply
import org.apache.commons.net.ftp.FTPSClient
import java.io.File
import java.io.InputStream
import java.net.IDN
import java.security.MessageDigest
import java.time.Duration
import java.util.Base64
import java.util.Vector

enum class ConnectionProtocol(val label: String, val defaultPort: Int) {
    FTP("FTP", 21),
    EXPLICIT_FTPS("Explicit FTPS", 21),
    SFTP("SFTP", 22);

    companion object {
        fun fromIndex(index: Int): ConnectionProtocol = entries.getOrElse(index) { FTP }
    }
}

data class ConnectionProfile(
    val protocol: ConnectionProtocol,
    val host: String,
    val port: Int,
    val username: String,
    val password: String,
    val hostKeyFingerprint: String,
    val remotePath: String
)

data class ConnectionProbeResult(
    val reachable: Boolean,
    val title: String,
    val detail: String,
    val rows: List<RemoteRow>
)

data class RemoteRow(
    val name: String,
    val detail: String,
    val remotePath: String? = null,
    val isDirectory: Boolean = false,
    val isFile: Boolean = false
)

data class TransferResult(
    val title: String,
    val detail: String,
    val remotePath: String
)

class ConnectionController {
    fun listRemote(profile: ConnectionProfile): ConnectionProbeResult {
        val normalized = normalizedProfile(profile)
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> listFtp(normalized, secure = false)
            ConnectionProtocol.EXPLICIT_FTPS -> listFtp(normalized, secure = true)
            ConnectionProtocol.SFTP -> listSftp(normalized)
        }
    }

    fun downloadRemote(profile: ConnectionProfile, remoteFilePath: String, outputFile: File): TransferResult {
        val normalized = normalizedProfile(profile)
        val target = normalizeRemoteTarget(remoteFilePath)
        outputFile.parentFile?.mkdirs()
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> downloadFtp(normalized, secure = false, remoteFilePath = target, outputFile = outputFile)
            ConnectionProtocol.EXPLICIT_FTPS -> downloadFtp(normalized, secure = true, remoteFilePath = target, outputFile = outputFile)
            ConnectionProtocol.SFTP -> downloadSftp(normalized, remoteFilePath = target, outputFile = outputFile)
        }
    }

    fun uploadRemote(profile: ConnectionProfile, input: InputStream, remoteFilePath: String): TransferResult {
        val normalized = normalizedProfile(profile)
        val target = normalizeRemoteTarget(remoteFilePath)
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> uploadFtp(normalized, secure = false, input = input, remoteFilePath = target)
            ConnectionProtocol.EXPLICIT_FTPS -> uploadFtp(normalized, secure = true, input = input, remoteFilePath = target)
            ConnectionProtocol.SFTP -> uploadSftp(normalized, input = input, remoteFilePath = target)
        }
    }

    fun deleteRemoteFile(profile: ConnectionProfile, remoteFilePath: String): TransferResult {
        val normalized = normalizedProfile(profile)
        val target = normalizeRemoteDeleteTarget(remoteFilePath)
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> deleteFtp(normalized, secure = false, remoteFilePath = target)
            ConnectionProtocol.EXPLICIT_FTPS -> deleteFtp(normalized, secure = true, remoteFilePath = target)
            ConnectionProtocol.SFTP -> deleteSftp(normalized, remoteFilePath = target)
        }
    }

    fun createRemoteDirectory(profile: ConnectionProfile, remoteDirectoryPath: String): TransferResult {
        val normalized = normalizedProfile(profile)
        val target = normalizeRemoteTarget(remoteDirectoryPath)
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> mkdirFtp(normalized, secure = false, remoteDirectoryPath = target)
            ConnectionProtocol.EXPLICIT_FTPS -> mkdirFtp(normalized, secure = true, remoteDirectoryPath = target)
            ConnectionProtocol.SFTP -> mkdirSftp(normalized, remoteDirectoryPath = target)
        }
    }

    private fun listFtp(profile: ConnectionProfile, secure: Boolean): ConnectionProbeResult = withFtpClient(profile, secure) { client ->
        val requestedPath = profile.remotePath.ifBlank { "/" }
        if (requestedPath != "/") {
            require(client.changeWorkingDirectory(requestedPath)) {
                "Remote path is not available: $requestedPath"
            }
        }
        val workingPath = client.printWorkingDirectory() ?: requestedPath
        val files = client.listFiles()
        ConnectionProbeResult(
            reachable = true,
            title = "Connected",
            detail = "${profile.protocol.label} session opened on ${redactHost(profile.host)}:${profile.port}.",
            rows = ftpRows(workingPath, files)
        )
    }

    private fun listSftp(profile: ConnectionProfile): ConnectionProbeResult = withSftpChannel(profile) { channel ->
        val requestedPath = profile.remotePath.ifBlank { "/" }
        if (requestedPath != "/") {
            channel.cd(requestedPath)
        }
        val workingPath = channel.pwd()
        val entries = channel.ls(".") as Vector<*>
        ConnectionProbeResult(
            reachable = true,
            title = "Connected",
            detail = "SFTP session opened on ${redactHost(profile.host)}:${profile.port}.",
            rows = sftpRows(workingPath, entries)
        )
    }

    private fun downloadFtp(profile: ConnectionProfile, secure: Boolean, remoteFilePath: String, outputFile: File): TransferResult = withFtpClient(profile, secure) { client ->
        outputFile.outputStream().use { output ->
            require(client.retrieveFile(remoteFilePath, output)) {
                "Download failed for $remoteFilePath."
            }
        }
        TransferResult(
            title = "Download complete",
            detail = "Saved ${formatBytes(outputFile.length())} to ${outputFile.name}.",
            remotePath = remoteFilePath
        )
    }

    private fun uploadFtp(profile: ConnectionProfile, secure: Boolean, input: InputStream, remoteFilePath: String): TransferResult = withFtpClient(profile, secure) { client ->
        input.use { source ->
            require(client.storeFile(remoteFilePath, source)) {
                "Upload failed for $remoteFilePath."
            }
        }
        TransferResult(
            title = "Upload complete",
            detail = "Uploaded file to $remoteFilePath.",
            remotePath = remoteFilePath
        )
    }

    private fun deleteFtp(profile: ConnectionProfile, secure: Boolean, remoteFilePath: String): TransferResult = withFtpClient(profile, secure) { client ->
        require(client.deleteFile(remoteFilePath)) {
            "Delete failed for $remoteFilePath."
        }
        TransferResult(
            title = "Remote file deleted",
            detail = "Deleted $remoteFilePath.",
            remotePath = remoteFilePath
        )
    }

    private fun mkdirFtp(profile: ConnectionProfile, secure: Boolean, remoteDirectoryPath: String): TransferResult = withFtpClient(profile, secure) { client ->
        require(client.makeDirectory(remoteDirectoryPath)) {
            "Folder creation failed for $remoteDirectoryPath."
        }
        TransferResult(
            title = "Remote folder created",
            detail = "Created $remoteDirectoryPath.",
            remotePath = remoteDirectoryPath
        )
    }

    private fun downloadSftp(profile: ConnectionProfile, remoteFilePath: String, outputFile: File): TransferResult = withSftpChannel(profile) { channel ->
        channel.get(remoteFilePath, outputFile.absolutePath)
        TransferResult(
            title = "Download complete",
            detail = "Saved ${formatBytes(outputFile.length())} to ${outputFile.name}.",
            remotePath = remoteFilePath
        )
    }

    private fun uploadSftp(profile: ConnectionProfile, input: InputStream, remoteFilePath: String): TransferResult = withSftpChannel(profile) { channel ->
        input.use { source -> channel.put(source, remoteFilePath) }
        TransferResult(
            title = "Upload complete",
            detail = "Uploaded file to $remoteFilePath.",
            remotePath = remoteFilePath
        )
    }

    private fun deleteSftp(profile: ConnectionProfile, remoteFilePath: String): TransferResult = withSftpChannel(profile) { channel ->
        channel.rm(remoteFilePath)
        TransferResult(
            title = "Remote file deleted",
            detail = "Deleted $remoteFilePath.",
            remotePath = remoteFilePath
        )
    }

    private fun mkdirSftp(profile: ConnectionProfile, remoteDirectoryPath: String): TransferResult = withSftpChannel(profile) { channel ->
        channel.mkdir(remoteDirectoryPath)
        TransferResult(
            title = "Remote folder created",
            detail = "Created $remoteDirectoryPath.",
            remotePath = remoteDirectoryPath
        )
    }

    private fun <T> withFtpClient(profile: ConnectionProfile, secure: Boolean, block: (FTPClient) -> T): T {
        val client = if (secure) FTPSClient(false) else FTPClient()
        client.connectTimeout = CONNECT_TIMEOUT_MS
        client.defaultTimeout = CONNECT_TIMEOUT_MS
        client.dataTimeout = Duration.ofMillis(CONNECT_TIMEOUT_MS.toLong())
        try {
            client.connect(profile.host, profile.port)
            require(FTPReply.isPositiveCompletion(client.replyCode)) {
                "Server rejected connection: ${client.replyString.trim()}"
            }
            require(profile.username.isNotBlank()) { "Username is required." }
            require(profile.password.isNotBlank()) { "Password is required." }
            require(client.login(profile.username, profile.password)) { "Login failed for ${profile.protocol.label}." }

            if (secure && client is FTPSClient) {
                client.execPBSZ(0)
                client.execPROT("P")
            }

            client.enterLocalPassiveMode()
            client.setFileType(FTP.BINARY_FILE_TYPE)
            return block(client)
        } finally {
            runCatching { if (client.isConnected) client.logout() }
            runCatching { if (client.isConnected) client.disconnect() }
        }
    }

    private fun <T> withSftpChannel(profile: ConnectionProfile, block: (ChannelSftp) -> T): T {
        require(profile.username.isNotBlank()) { "SFTP username is required." }
        require(profile.password.isNotBlank()) { "SFTP password is required." }
        require(profile.hostKeyFingerprint.isNotBlank()) {
            "SFTP host key fingerprint is required. Use the server SHA-256 host key fingerprint."
        }

        val jsch = JSch()
        jsch.hostKeyRepository = FingerprintHostKeyRepository(profile.hostKeyFingerprint)
        val session = jsch.getSession(profile.username, profile.host, profile.port)
        session.setPassword(profile.password)
        session.setConfig("StrictHostKeyChecking", "yes")
        session.timeout = CONNECT_TIMEOUT_MS

        var channel: ChannelSftp? = null
        try {
            session.connect(CONNECT_TIMEOUT_MS)
            channel = session.openChannel("sftp") as ChannelSftp
            channel.connect(CONNECT_TIMEOUT_MS)
            return block(channel)
        } finally {
            runCatching { channel?.disconnect() }
            runCatching { session.disconnect() }
        }
    }

    private fun ftpRows(path: String, files: Array<FTPFile>): List<RemoteRow> {
        val entries = files
            .filter { it.name != "." && it.name != ".." }
            .take(MAX_ROWS)
            .map { file ->
                val remotePath = joinRemotePath(path, file.name)
                RemoteRow(
                    name = if (file.isDirectory) "[DIR] ${file.name}" else "[FILE] ${file.name}",
                    detail = when {
                        file.isDirectory -> "Folder · $path"
                        file.size >= 0L -> "File · ${formatBytes(file.size)}"
                        else -> "File"
                    },
                    remotePath = remotePath,
                    isDirectory = file.isDirectory,
                    isFile = file.isFile
                )
            }
        return listOf(RemoteRow("Remote path", path)) + entries.ifEmpty {
            listOf(RemoteRow("Remote folder", "No visible files returned by the server."))
        }
    }

    private fun sftpRows(path: String, entries: Vector<*>): List<RemoteRow> {
        val rows = entries
            .asSequence()
            .filterIsInstance<ChannelSftp.LsEntry>()
            .filter { it.filename != "." && it.filename != ".." }
            .take(MAX_ROWS)
            .map { entry ->
                val remotePath = joinRemotePath(path, entry.filename)
                RemoteRow(
                    name = if (entry.attrs.isDir) "[DIR] ${entry.filename}" else "[FILE] ${entry.filename}",
                    detail = if (entry.attrs.isDir) "Folder · $path" else "File · ${formatBytes(entry.attrs.size)}",
                    remotePath = remotePath,
                    isDirectory = entry.attrs.isDir,
                    isFile = !entry.attrs.isDir
                )
            }
            .toList()
        return listOf(RemoteRow("Remote path", path)) + rows.ifEmpty {
            listOf(RemoteRow("Remote folder", "No visible files returned by the server."))
        }
    }

    private fun normalizedProfile(profile: ConnectionProfile): ConnectionProfile {
        val normalizedHost = normalizeHost(profile.host)
        require(normalizedHost.isNotBlank()) { "Host is required." }
        require(profile.port in 1..65535) { "Port must be between 1 and 65535." }
        return profile.copy(
            host = normalizedHost,
            remotePath = normalizeRemoteDirectory(profile.remotePath)
        )
    }

    private fun normalizeHost(input: String): String {
        val value = input.trim()
            .removePrefix("ftp://")
            .removePrefix("ftps://")
            .removePrefix("sftp://")
            .substringBefore('/')
            .substringBefore(':')
            .trim()
        return if (value.isBlank()) "" else IDN.toASCII(value)
    }

    private fun normalizeRemoteDirectory(input: String): String {
        val value = input.trim().ifBlank { "/" }
        return if (value.startsWith('/')) value else "/$value"
    }

    private fun normalizeRemoteTarget(input: String): String {
        val value = input.trim()
        require(value.isNotBlank()) { "Remote path is required." }
        val target = if (value.startsWith('/')) value else "/$value"
        require(target.split('/').filter { it.isNotBlank() }.none { it == "." || it == ".." }) {
            "Remote path must not contain dot path segments."
        }
        return target
    }

    private fun normalizeRemoteDeleteTarget(input: String): String {
        val target = normalizeRemoteTarget(input)
        require(target.split('/').any { it.isNotBlank() }) {
            "Refusing to delete the remote root path."
        }
        return target
    }

    private fun joinRemotePath(directory: String, child: String): String {
        val safeChild = child.trim().trimStart('/')
        val base = directory.ifBlank { "/" }.trimEnd('/')
        return if (base.isBlank()) "/$safeChild" else "$base/$safeChild"
    }

    private fun redactHost(host: String): String {
        if (host.length <= 6) return host
        return "${host.take(3)}…${host.takeLast(3)}"
    }

    private fun formatBytes(bytes: Long): String {
        if (bytes < 1024) return "$bytes B"
        val units = arrayOf("KB", "MB", "GB", "TB")
        var value = bytes.toDouble() / 1024.0
        var unitIndex = 0
        while (value >= 1024.0 && unitIndex < units.lastIndex) {
            value /= 1024.0
            unitIndex += 1
        }
        return "%.1f %s".format(value, units[unitIndex])
    }

    private class FingerprintHostKeyRepository(
        expectedFingerprint: String
    ) : HostKeyRepository {
        private val expected = normalizeFingerprint(expectedFingerprint)

        override fun getKnownHostsRepositoryID(): String = "Ghost FTP Android host key verifier"

        override fun check(host: String?, key: ByteArray?): Int {
            if (key == null || expected.isBlank()) return HostKeyRepository.NOT_INCLUDED
            val actual = normalizeFingerprint(sha256Fingerprint(key))
            return if (actual == expected) HostKeyRepository.OK else HostKeyRepository.CHANGED
        }

        override fun add(hostkey: HostKey?, ui: UserInfo?) = Unit

        override fun remove(host: String?, type: String?) = Unit

        override fun remove(host: String?, type: String?, key: ByteArray?) = Unit

        override fun getHostKey(): Array<HostKey> = emptyArray()

        override fun getHostKey(host: String?, type: String?): Array<HostKey> = emptyArray()

        private companion object {
            fun normalizeFingerprint(value: String): String {
                val trimmed = value.trim().replace(" ", "")
                return if (trimmed.startsWith("SHA256:", ignoreCase = true)) {
                    "SHA256:${trimmed.substringAfter(':')}"
                } else {
                    "SHA256:$trimmed"
                }
            }

            fun sha256Fingerprint(key: ByteArray): String {
                val digest = MessageDigest.getInstance("SHA-256").digest(key)
                return "SHA256:${Base64.getEncoder().withoutPadding().encodeToString(digest)}"
            }
        }
    }

    private companion object {
        const val CONNECT_TIMEOUT_MS = 10000
        const val MAX_ROWS = 24
    }
}
