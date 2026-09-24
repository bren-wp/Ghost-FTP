package com.ghostftp.android

import com.jcraft.jsch.ChannelSftp
import com.jcraft.jsch.JSch
import org.apache.commons.net.ftp.FTP
import org.apache.commons.net.ftp.FTPClient
import org.apache.commons.net.ftp.FTPFile
import org.apache.commons.net.ftp.FTPReply
import org.apache.commons.net.ftp.FTPSClient
import java.net.IDN
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
    val detail: String
)

class ConnectionController {
    fun listRemote(profile: ConnectionProfile): ConnectionProbeResult {
        val normalizedHost = normalizeHost(profile.host)
        require(normalizedHost.isNotBlank()) { "Host is required." }
        require(profile.port in 1..65535) { "Port must be between 1 and 65535." }

        return when (profile.protocol) {
            ConnectionProtocol.FTP -> listFtp(profile.copy(host = normalizedHost), secure = false)
            ConnectionProtocol.EXPLICIT_FTPS -> listFtp(profile.copy(host = normalizedHost), secure = true)
            ConnectionProtocol.SFTP -> listSftp(profile.copy(host = normalizedHost))
        }
    }

    private fun listFtp(profile: ConnectionProfile, secure: Boolean): ConnectionProbeResult {
        val client = if (secure) FTPSClient(false) else FTPClient()
        client.connectTimeout = CONNECT_TIMEOUT_MS
        client.defaultTimeout = CONNECT_TIMEOUT_MS
        client.dataTimeout = CONNECT_TIMEOUT_MS
        try {
            client.connect(profile.host, profile.port)
            require(FTPReply.isPositiveCompletion(client.replyCode)) {
                "Server rejected connection: ${client.replyString.trim()}"
            }
            val username = profile.username.ifBlank { "anonymous" }
            val password = profile.password.ifBlank { "ghostftp@local" }
            require(client.login(username, password)) { "Login failed for ${profile.protocol.label}." }

            if (secure && client is FTPSClient) {
                client.execPBSZ(0)
                client.execPROT("P")
            }

            client.enterLocalPassiveMode()
            client.setFileType(FTP.BINARY_FILE_TYPE)

            val requestedPath = profile.remotePath.ifBlank { "/" }
            if (requestedPath != "/") {
                require(client.changeWorkingDirectory(requestedPath)) {
                    "Remote path is not available: $requestedPath"
                }
            }
            val workingPath = client.printWorkingDirectory() ?: requestedPath
            val files = client.listFiles()
            return ConnectionProbeResult(
                reachable = true,
                title = "Connected",
                detail = "${profile.protocol.label} session opened on ${redactHost(profile.host)}:${profile.port}.",
                rows = ftpRows(workingPath, files)
            )
        } finally {
            runCatching { if (client.isConnected) client.logout() }
            runCatching { if (client.isConnected) client.disconnect() }
        }
    }

    private fun listSftp(profile: ConnectionProfile): ConnectionProbeResult {
        require(profile.username.isNotBlank()) { "SFTP username is required." }
        require(profile.password.isNotBlank()) { "SFTP password is required for this Android session." }

        val jsch = JSch()
        val session = jsch.getSession(profile.username, profile.host, profile.port)
        session.setPassword(profile.password)
        session.setConfig("StrictHostKeyChecking", "no")
        session.timeout = CONNECT_TIMEOUT_MS

        var channel: ChannelSftp? = null
        try {
            session.connect(CONNECT_TIMEOUT_MS)
            channel = session.openChannel("sftp") as ChannelSftp
            channel.connect(CONNECT_TIMEOUT_MS)
            val requestedPath = profile.remotePath.ifBlank { "/" }
            if (requestedPath != "/") {
                channel.cd(requestedPath)
            }
            val workingPath = channel.pwd()
            val entries = channel.ls(".") as Vector<*>
            return ConnectionProbeResult(
                reachable = true,
                title = "Connected",
                detail = "SFTP session opened on ${redactHost(profile.host)}:${profile.port}.",
                rows = sftpRows(workingPath, entries)
            )
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
                RemoteRow(
                    name = if (file.isDirectory) "📁 ${file.name}" else "📄 ${file.name}",
                    detail = when {
                        file.isDirectory -> "Folder · $path"
                        file.size >= 0L -> "File · ${formatBytes(file.size)}"
                        else -> "File"
                    }
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
                RemoteRow(
                    name = if (entry.attrs.isDir) "📁 ${entry.filename}" else "📄 ${entry.filename}",
                    detail = if (entry.attrs.isDir) "Folder · $path" else "File · ${formatBytes(entry.attrs.size)}"
                )
            }
            .toList()
        return listOf(RemoteRow("Remote path", path)) + rows.ifEmpty {
            listOf(RemoteRow("Remote folder", "No visible files returned by the server."))
        }
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

    private companion object {
        const val CONNECT_TIMEOUT_MS = 10000
        const val MAX_ROWS = 12
    }
}
