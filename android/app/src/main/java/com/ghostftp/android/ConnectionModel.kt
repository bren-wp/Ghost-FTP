package com.ghostftp.android

import com.jcraft.jsch.ChannelSftp
import com.jcraft.jsch.HostKey
import com.jcraft.jsch.HostKeyRepository
import com.jcraft.jsch.JSch
import com.jcraft.jsch.SftpException
import com.jcraft.jsch.UserInfo
import org.apache.commons.net.ftp.FTP
import org.apache.commons.net.ftp.FTPClient
import org.apache.commons.net.ftp.FTPFile
import org.apache.commons.net.ftp.FTPReply
import org.apache.commons.net.ftp.FTPSClient
import java.io.File
import java.io.IOException
import java.io.InputStream
import java.io.OutputStream
import java.net.IDN
import java.security.MessageDigest
import java.time.Duration
import java.util.Base64
import java.util.UUID
import java.util.Vector
import java.util.concurrent.atomic.AtomicBoolean

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
    val remotePath: String,
    val keepAliveSeconds: Int = 15
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

// Compile redaction patterns once instead of recreating them for every
// connection error. User/server diagnostics are untrusted input.
private val REDACT_URL_USER_INFO =
    Regex("""(?i)\\b([a-z][a-z0-9+.-]*://)([^/\\s:@]+):([^/\\s@]+)@""")
private val REDACT_SECRET_ASSIGNMENT =
    Regex("""(?i)\\b(password|passwd|pwd|token|secret|authorization)\\s*[:=]\\s*([^\\s,;]+)""")
private val REDACT_BEARER_TOKEN =
    Regex("""(?i)\\b(Bearer)\\s+[A-Za-z0-9._~+/=-]+""")

internal fun redactSensitiveErrorText(
    raw: String,
    secrets: Iterable<String> = emptyList()
): String {
    // Redact before the 600-character UI limit. Truncating first can expose
    // a password/token prefix when a secret straddles the display boundary.
    // Bound regex work even when a server sends a massive error message.
    val uniqueSecrets = secrets.filter { it.isNotEmpty() }.distinct()
    if (uniqueSecrets.any { it.length > 4096 }) {
        return "Connection error details hidden for privacy."
    }
    val inspectionLimit = (600 + (uniqueSecrets.maxOfOrNull { it.length } ?: 0) + 512)
        .coerceAtMost(8192)
    var redacted = raw.take(inspectionLimit)

    uniqueSecrets.forEach { secret -> redacted = redacted.replace(secret, "••••") }

    redacted = REDACT_URL_USER_INFO.replace(redacted) { match ->
        "${match.groupValues[1]}${match.groupValues[2]}:••••@"
    }
    redacted = REDACT_SECRET_ASSIGNMENT.replace(redacted) { match ->
        "${match.groupValues[1]}=••••"
    }
    redacted = REDACT_BEARER_TOKEN.replace(redacted) { match ->
        "${match.groupValues[1]} ••••"
    }

    return redacted.take(600)
}

class OperationCancellation {
    private val canceled = AtomicBoolean(false)

    fun cancel() {
        canceled.set(true)
    }

    fun throwIfCanceled() {
        if (canceled.get()) throw OperationCanceledException()
    }

    fun isCanceled(): Boolean = canceled.get()
}

class OperationCanceledException : IOException("Operation canceled.")

internal fun commitStagedRemoteReplacement(
    remoteFilePath: String,
    backupPath: String,
    targetExists: () -> Boolean,
    backupTarget: () -> Unit,
    promoteTemporary: () -> Unit,
    restoreBackup: () -> Unit,
    cleanupTemporary: () -> Unit,
    cleanupBackup: () -> Unit,
    checkpoint: () -> Unit = {}
) {
    var backupCreated = false
    try {
        checkpoint()
        if (targetExists()) {
            try {
                backupTarget()
            } catch (error: Throwable) {
                throw IOException(
                    "Upload completed but could not preserve the existing remote file before replacement.",
                    error
                )
            }
            backupCreated = true
        }

        checkpoint()
        if (!backupCreated && targetExists()) {
            throw IOException(
                "Remote upload target appeared during transfer; refusing to overwrite it."
            )
        }

        checkpoint()
        try {
            promoteTemporary()
        } catch (error: Throwable) {
            throw IOException(
                "Upload completed but could not promote the temporary file to $remoteFilePath.",
                error
            )
        }

        if (backupCreated) {
            runCatching { cleanupBackup() }
        }
    } catch (error: Throwable) {
        runCatching { cleanupTemporary() }
        if (backupCreated) {
            val restored = runCatching {
                restoreBackup()
                true
            }.getOrDefault(false)
            if (!restored) {
                val recoveryError = IOException(
                    "Upload failed and the original remote file could not be restored automatically. " +
                        "Original remote file preserved at $backupPath for manual recovery."
                )
                recoveryError.addSuppressed(error)
                throw recoveryError
            }
        }
        throw error
    }
}

class ConnectionController {
    fun checkConnectionHealth(
        profile: ConnectionProfile,
        cancellation: OperationCancellation = OperationCancellation()
    ): ConnectionProbeResult {
        cancellation.throwIfCanceled()
        val normalized = normalizedProfile(profile)
        return when (normalized.protocol) {
            ConnectionProtocol.FTP ->
                healthFtp(normalized, secure = false, cancellation = cancellation)
            ConnectionProtocol.EXPLICIT_FTPS ->
                healthFtp(normalized, secure = true, cancellation = cancellation)
            ConnectionProtocol.SFTP ->
                healthSftp(normalized, cancellation)
        }
    }

    fun listRemote(
        profile: ConnectionProfile,
        cancellation: OperationCancellation = OperationCancellation()
    ): ConnectionProbeResult {
        cancellation.throwIfCanceled()
        val normalized = normalizedProfile(profile)
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> listFtp(normalized, secure = false, cancellation = cancellation)
            ConnectionProtocol.EXPLICIT_FTPS -> listFtp(normalized, secure = true, cancellation = cancellation)
            ConnectionProtocol.SFTP -> listSftp(normalized, cancellation)
        }
    }

    fun downloadRemote(
        profile: ConnectionProfile,
        remoteFilePath: String,
        outputFile: File,
        cancellation: OperationCancellation = OperationCancellation()
    ): TransferResult {
        cancellation.throwIfCanceled()
        val normalized = normalizedProfile(profile)
        val target = normalizeRemoteTarget(remoteFilePath)
        outputFile.parentFile?.let { parent ->
            require(parent.exists() || parent.mkdirs()) { "Unable to create the download directory." }
        }
        val temporaryFile = localTemporarySibling(outputFile, "download")
        runCatching { if (temporaryFile.exists()) temporaryFile.delete() }
        return try {
            val result = when (normalized.protocol) {
                ConnectionProtocol.FTP -> downloadFtp(normalized, secure = false, remoteFilePath = target, outputFile = temporaryFile, cancellation = cancellation)
                ConnectionProtocol.EXPLICIT_FTPS -> downloadFtp(normalized, secure = true, remoteFilePath = target, outputFile = temporaryFile, cancellation = cancellation)
                ConnectionProtocol.SFTP -> downloadSftp(normalized, remoteFilePath = target, outputFile = temporaryFile, cancellation = cancellation)
            }
            cancellation.throwIfCanceled()
            commitLocalReplacement(temporaryFile, outputFile)
            result.copy(
                detail = "Saved ${formatBytes(outputFile.length())} to ${outputFile.name}."
            )
        } catch (error: Throwable) {
            runCatching { if (temporaryFile.exists()) temporaryFile.delete() }
            throw error
        }
    }

    fun uploadRemote(
        profile: ConnectionProfile,
        input: InputStream,
        remoteFilePath: String,
        cancellation: OperationCancellation = OperationCancellation()
    ): TransferResult = input.use { source ->
        cancellation.throwIfCanceled()
        val normalized = normalizedProfile(profile)
        val target = normalizeRemoteTarget(remoteFilePath)
        when (normalized.protocol) {
            ConnectionProtocol.FTP -> uploadFtp(normalized, secure = false, input = source, remoteFilePath = target, cancellation = cancellation)
            ConnectionProtocol.EXPLICIT_FTPS -> uploadFtp(normalized, secure = true, input = source, remoteFilePath = target, cancellation = cancellation)
            ConnectionProtocol.SFTP -> uploadSftp(normalized, input = source, remoteFilePath = target, cancellation = cancellation)
        }
    }

    fun deleteRemoteFile(
        profile: ConnectionProfile,
        remoteFilePath: String,
        cancellation: OperationCancellation = OperationCancellation()
    ): TransferResult {
        cancellation.throwIfCanceled()
        val normalized = normalizedProfile(profile)
        val target = normalizeRemoteDeleteTarget(remoteFilePath)
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> deleteFtp(normalized, secure = false, remoteFilePath = target, cancellation = cancellation)
            ConnectionProtocol.EXPLICIT_FTPS -> deleteFtp(normalized, secure = true, remoteFilePath = target, cancellation = cancellation)
            ConnectionProtocol.SFTP -> deleteSftp(normalized, remoteFilePath = target, cancellation = cancellation)
        }
    }

    fun createRemoteDirectory(
        profile: ConnectionProfile,
        remoteDirectoryPath: String,
        cancellation: OperationCancellation = OperationCancellation()
    ): TransferResult {
        cancellation.throwIfCanceled()
        val normalized = normalizedProfile(profile)
        val target = normalizeRemoteTarget(remoteDirectoryPath)
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> mkdirFtp(normalized, secure = false, remoteDirectoryPath = target, cancellation = cancellation)
            ConnectionProtocol.EXPLICIT_FTPS -> mkdirFtp(normalized, secure = true, remoteDirectoryPath = target, cancellation = cancellation)
            ConnectionProtocol.SFTP -> mkdirSftp(normalized, remoteDirectoryPath = target, cancellation = cancellation)
        }
    }

    fun renameRemote(
        profile: ConnectionProfile,
        sourcePath: String,
        destinationPath: String,
        cancellation: OperationCancellation = OperationCancellation()
    ): TransferResult {
        cancellation.throwIfCanceled()
        val normalized = normalizedProfile(profile)
        val source = normalizeRemoteTarget(sourcePath)
        val destination = normalizeRemoteTarget(destinationPath)
        require(source.split('/').any { it.isNotBlank() }) {
            "Refusing to rename the remote root path."
        }
        require(destination.split('/').any { it.isNotBlank() }) {
            "Refusing to rename to the remote root path."
        }
        require(source != destination) { "Source and destination paths must differ." }
        return when (normalized.protocol) {
            ConnectionProtocol.FTP -> renameFtp(normalized, secure = false, sourcePath = source, destinationPath = destination, cancellation = cancellation)
            ConnectionProtocol.EXPLICIT_FTPS -> renameFtp(normalized, secure = true, sourcePath = source, destinationPath = destination, cancellation = cancellation)
            ConnectionProtocol.SFTP -> renameSftp(normalized, sourcePath = source, destinationPath = destination, cancellation = cancellation)
        }
    }

    private fun healthFtp(
        profile: ConnectionProfile,
        secure: Boolean,
        cancellation: OperationCancellation
    ): ConnectionProbeResult = withFtpClient(profile, secure, cancellation) { client ->
        cancellation.throwIfCanceled()
        require(client.sendNoOp()) {
            "${profile.protocol.label} health check was rejected by the server."
        }
        ConnectionProbeResult(
            reachable = true,
            title = "Connection healthy",
            detail = "${profile.protocol.label} authenticated and accepted NOOP on ${redactHost(profile.host)}:${profile.port}.",
            rows = emptyList()
        )
    }

    private fun healthSftp(
        profile: ConnectionProfile,
        cancellation: OperationCancellation
    ): ConnectionProbeResult = withSftpChannel(profile, cancellation) { channel ->
        cancellation.throwIfCanceled()
        val path = channel.pwd()
        ConnectionProbeResult(
            reachable = true,
            title = "Connection healthy",
            detail = "SFTP authenticated and resolved ${redactHost(profile.host)}:${profile.port}.",
            rows = listOf(
                RemoteRow(
                    name = path,
                    detail = "Current remote directory",
                    remotePath = path,
                    isDirectory = true
                )
            )
        )
    }

    private fun listFtp(
        profile: ConnectionProfile,
        secure: Boolean,
        cancellation: OperationCancellation
    ): ConnectionProbeResult = withFtpClient(profile, secure, cancellation) { client ->
        cancellation.throwIfCanceled()
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

    private fun listSftp(
        profile: ConnectionProfile,
        cancellation: OperationCancellation
    ): ConnectionProbeResult = withSftpChannel(profile, cancellation) { channel ->
        cancellation.throwIfCanceled()
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

    private fun downloadFtp(
        profile: ConnectionProfile,
        secure: Boolean,
        remoteFilePath: String,
        outputFile: File,
        cancellation: OperationCancellation
    ): TransferResult = withFtpClient(profile, secure, cancellation) { client ->
        cancellation.throwIfCanceled()
        val remote = client.retrieveFileStream(remoteFilePath)
            ?: throw IOException("Download failed for $remoteFilePath: server did not open a data stream.")
        try {
            remote.use { source ->
                outputFile.outputStream().use { output ->
                    copyCancelable(source, output, cancellation)
                }
            }
            cancellation.throwIfCanceled()
            require(client.completePendingCommand()) {
                "Download failed for $remoteFilePath."
            }
        } catch (error: Throwable) {
            runCatching { if (client.isConnected) client.abort() }
            throw error
        }
        TransferResult(
            title = "Download complete",
            detail = "Saved ${formatBytes(outputFile.length())} to ${outputFile.name}.",
            remotePath = remoteFilePath
        )
    }

    private fun uploadFtp(
        profile: ConnectionProfile,
        secure: Boolean,
        input: InputStream,
        remoteFilePath: String,
        cancellation: OperationCancellation
    ): TransferResult = withFtpClient(profile, secure, cancellation) { client ->
        cancellation.throwIfCanceled()
        val temporaryPath = remoteTemporarySibling(remoteFilePath, "upload")
        val backupPath = remoteTemporarySibling(remoteFilePath, "backup")
        var dataCommandCompleted = false

        try {
            val remote = client.storeFileStream(temporaryPath)
                ?: throw IOException("Upload failed for $remoteFilePath: server did not open a data stream.")
            remote.use { output ->
                copyCancelable(input, output, cancellation)
            }
            cancellation.throwIfCanceled()
            require(client.completePendingCommand()) {
                "Upload failed for $remoteFilePath."
            }
            dataCommandCompleted = true
            cancellation.throwIfCanceled()
        } catch (error: Throwable) {
            if (!dataCommandCompleted) {
                runCatching { if (client.isConnected) client.abort() }
            }
            runCatching { if (client.isConnected) client.deleteFile(temporaryPath) }
            throw error
        }

        commitStagedRemoteReplacement(
            remoteFilePath = remoteFilePath,
            backupPath = backupPath,
            targetExists = { ftpRemoteTargetExists(client, remoteFilePath) },
            backupTarget = {
                if (!client.rename(remoteFilePath, backupPath)) {
                    throw IOException("FTP server rejected backup rename.")
                }
            },
            promoteTemporary = {
                if (!client.rename(temporaryPath, remoteFilePath)) {
                    throw IOException("FTP server rejected temporary-file promotion.")
                }
            },
            restoreBackup = {
                if (!client.isConnected || !client.rename(backupPath, remoteFilePath)) {
                    throw IOException("FTP server rejected backup restoration.")
                }
            },
            cleanupTemporary = {
                if (client.isConnected) client.deleteFile(temporaryPath)
            },
            cleanupBackup = {
                if (client.isConnected) client.deleteFile(backupPath)
            },
            checkpoint = { cancellation.throwIfCanceled() }
        )

        TransferResult(
            title = "Upload complete",
            detail = "Uploaded file to $remoteFilePath.",
            remotePath = remoteFilePath
        )
    }

    private fun deleteFtp(
        profile: ConnectionProfile,
        secure: Boolean,
        remoteFilePath: String,
        cancellation: OperationCancellation
    ): TransferResult = withFtpClient(profile, secure, cancellation) { client ->
        cancellation.throwIfCanceled()
        val deletedAsFile = client.deleteFile(remoteFilePath)
        val deletedAsDirectory = !deletedAsFile && client.removeDirectory(remoteFilePath)
        require(deletedAsFile || deletedAsDirectory) {
            "Delete failed for $remoteFilePath. Non-empty folders are not removed recursively."
        }
        TransferResult(
            title = if (deletedAsDirectory) "Remote folder deleted" else "Remote file deleted",
            detail = "Deleted $remoteFilePath.",
            remotePath = remoteFilePath
        )
    }

    private fun renameFtp(
        profile: ConnectionProfile,
        secure: Boolean,
        sourcePath: String,
        destinationPath: String,
        cancellation: OperationCancellation
    ): TransferResult = withFtpClient(profile, secure, cancellation) { client ->
        cancellation.throwIfCanceled()
        require(client.rename(sourcePath, destinationPath)) {
            "Rename failed from $sourcePath to $destinationPath."
        }
        TransferResult(
            title = "Remote entry renamed",
            detail = "Renamed $sourcePath to $destinationPath.",
            remotePath = destinationPath
        )
    }

    private fun mkdirFtp(
        profile: ConnectionProfile,
        secure: Boolean,
        remoteDirectoryPath: String,
        cancellation: OperationCancellation
    ): TransferResult = withFtpClient(profile, secure, cancellation) { client ->
        cancellation.throwIfCanceled()
        require(client.makeDirectory(remoteDirectoryPath)) {
            "Folder creation failed for $remoteDirectoryPath."
        }
        TransferResult(
            title = "Remote folder created",
            detail = "Created $remoteDirectoryPath.",
            remotePath = remoteDirectoryPath
        )
    }

    private fun downloadSftp(
        profile: ConnectionProfile,
        remoteFilePath: String,
        outputFile: File,
        cancellation: OperationCancellation
    ): TransferResult = withSftpChannel(profile, cancellation) { channel ->
        cancellation.throwIfCanceled()
        channel.get(remoteFilePath).use { source ->
            outputFile.outputStream().use { output ->
                copyCancelable(source, output, cancellation)
            }
        }
        TransferResult(
            title = "Download complete",
            detail = "Saved ${formatBytes(outputFile.length())} to ${outputFile.name}.",
            remotePath = remoteFilePath
        )
    }

    private fun uploadSftp(
        profile: ConnectionProfile,
        input: InputStream,
        remoteFilePath: String,
        cancellation: OperationCancellation
    ): TransferResult = withSftpChannel(profile, cancellation) { channel ->
        val temporaryPath = remoteTemporarySibling(remoteFilePath, "upload")
        val backupPath = remoteTemporarySibling(remoteFilePath, "backup")

        try {
            channel.put(temporaryPath).use { output ->
                copyCancelable(input, output, cancellation)
            }
            cancellation.throwIfCanceled()
        } catch (error: Throwable) {
            runCatching { channel.rm(temporaryPath) }
            throw error
        }

        commitStagedRemoteReplacement(
            remoteFilePath = remoteFilePath,
            backupPath = backupPath,
            targetExists = { sftpRemoteTargetExists(channel, remoteFilePath) },
            backupTarget = { channel.rename(remoteFilePath, backupPath) },
            promoteTemporary = { channel.rename(temporaryPath, remoteFilePath) },
            restoreBackup = { channel.rename(backupPath, remoteFilePath) },
            cleanupTemporary = { channel.rm(temporaryPath) },
            cleanupBackup = { channel.rm(backupPath) },
            checkpoint = { cancellation.throwIfCanceled() }
        )

        TransferResult(
            title = "Upload complete",
            detail = "Uploaded file to $remoteFilePath.",
            remotePath = remoteFilePath
        )
    }

    private fun deleteSftp(
        profile: ConnectionProfile,
        remoteFilePath: String,
        cancellation: OperationCancellation
    ): TransferResult = withSftpChannel(profile, cancellation) { channel ->
        cancellation.throwIfCanceled()
        val deletedAsDirectory = runCatching {
            channel.rm(remoteFilePath)
            false
        }.getOrElse {
            cancellation.throwIfCanceled()
            channel.rmdir(remoteFilePath)
            true
        }
        TransferResult(
            title = if (deletedAsDirectory) "Remote folder deleted" else "Remote file deleted",
            detail = "Deleted $remoteFilePath.",
            remotePath = remoteFilePath
        )
    }

    private fun renameSftp(
        profile: ConnectionProfile,
        sourcePath: String,
        destinationPath: String,
        cancellation: OperationCancellation
    ): TransferResult = withSftpChannel(profile, cancellation) { channel ->
        cancellation.throwIfCanceled()
        channel.rename(sourcePath, destinationPath)
        TransferResult(
            title = "Remote entry renamed",
            detail = "Renamed $sourcePath to $destinationPath.",
            remotePath = destinationPath
        )
    }

    private fun mkdirSftp(
        profile: ConnectionProfile,
        remoteDirectoryPath: String,
        cancellation: OperationCancellation
    ): TransferResult = withSftpChannel(profile, cancellation) { channel ->
        cancellation.throwIfCanceled()
        channel.mkdir(remoteDirectoryPath)
        TransferResult(
            title = "Remote folder created",
            detail = "Created $remoteDirectoryPath.",
            remotePath = remoteDirectoryPath
        )
    }

    private fun <T> withFtpClient(
        profile: ConnectionProfile,
        secure: Boolean,
        cancellation: OperationCancellation,
        block: (FTPClient) -> T
    ): T {
        val client = if (secure) {
            FTPSClient(false).apply {
                // Commons Net's FTPS defaults only check certificate dates and
                // leave endpoint identification disabled. Use Android/JVM's
                // platform trust store and verify the requested host name.
                setTrustManager(null)
                setEndpointCheckingEnabled(true)
            }
        } else {
            FTPClient()
        }
        client.connectTimeout = CONNECT_TIMEOUT_MS
        client.defaultTimeout = CONNECT_TIMEOUT_MS
        client.dataTimeout = Duration.ofMillis(CONNECT_TIMEOUT_MS.toLong())
        try {
            cancellation.throwIfCanceled()
            client.connect(profile.host, profile.port)
            cancellation.throwIfCanceled()
            require(FTPReply.isPositiveCompletion(client.replyCode)) {
                "Server rejected connection: ${client.replyString.trim()}"
            }
            require(profile.username.isNotBlank()) { "Username is required." }
            require(profile.password.isNotBlank()) { "Password is required." }
            require(client.login(profile.username, profile.password)) { "Login failed for ${profile.protocol.label}." }
            cancellation.throwIfCanceled()

            if (secure && client is FTPSClient) {
                client.execPBSZ(0)
                client.execPROT("P")
            }

            client.enterLocalPassiveMode()
            client.setFileType(FTP.BINARY_FILE_TYPE)
            cancellation.throwIfCanceled()
            return block(client)
        } finally {
            runCatching { if (client.isConnected) client.logout() }
            runCatching { if (client.isConnected) client.disconnect() }
        }
    }

    private fun <T> withSftpChannel(
        profile: ConnectionProfile,
        cancellation: OperationCancellation,
        block: (ChannelSftp) -> T
    ): T {
        require(profile.username.isNotBlank()) { "SFTP username is required." }
        require(profile.password.isNotBlank()) { "SFTP password is required." }
        require(profile.hostKeyFingerprint.isNotBlank()) {
            "SFTP host key fingerprint is required. Use the server SHA-256 host key fingerprint."
        }

        val jsch = JSch()
        jsch.hostKeyRepository = FingerprintHostKeyRepository(profile.hostKeyFingerprint)
        val session = jsch.getSession(profile.username, profile.host, profile.port)
        val passwordBytes = profile.password.toByteArray(Charsets.UTF_8)
        try {
            session.setPassword(passwordBytes)
        } finally {
            passwordBytes.fill(0)
        }
        session.setConfig("StrictHostKeyChecking", "yes")
        // Android currently exposes password authentication only. Restrict JSch
        // to that method so an untrusted SSH server cannot steer pre-auth into
        // keyboard-interactive parsing that Ghost FTP does not need.
        session.setConfig("PreferredAuthentications", "password")
        session.timeout = CONNECT_TIMEOUT_MS
        session.setServerAliveInterval(profile.keepAliveSeconds * 1000)
        session.setServerAliveCountMax(3)

        var channel: ChannelSftp? = null
        try {
            cancellation.throwIfCanceled()
            session.connect(CONNECT_TIMEOUT_MS)
            cancellation.throwIfCanceled()
            channel = session.openChannel("sftp") as ChannelSftp
            channel.connect(CONNECT_TIMEOUT_MS)
            cancellation.throwIfCanceled()
            return block(channel)
        } finally {
            runCatching { channel?.disconnect() }
            runCatching { session.disconnect() }
        }
    }

    private fun localTemporarySibling(target: File, purpose: String): File {
        val parent = target.parentFile ?: throw IOException("Download target has no parent directory.")
        return File(parent, ".ghostftp-$purpose-${UUID.randomUUID()}.part")
    }

    private fun commitLocalReplacement(temporary: File, target: File) {
        val backup = localTemporarySibling(target, "backup")
        var backupCreated = false
        try {
            if (target.exists()) {
                if (!target.renameTo(backup)) {
                    throw IOException("Unable to preserve existing local file before replacement: ${target.name}")
                }
                backupCreated = true
            }
            if (!temporary.renameTo(target)) {
                if (backupCreated) {
                    runCatching { backup.renameTo(target) }
                }
                throw IOException("Unable to promote completed download to ${target.name}.")
            }
            if (backupCreated) {
                runCatching { backup.delete() }
            }
        } catch (error: Throwable) {
            runCatching { if (temporary.exists()) temporary.delete() }
            if (backupCreated && !target.exists()) {
                runCatching { backup.renameTo(target) }
            }
            throw error
        }
    }

    private fun remoteTemporarySibling(target: String, purpose: String): String {
        val slash = target.lastIndexOf('/')
        val parent = if (slash >= 0) target.substring(0, slash + 1) else ""
        return "$parent.ghostftp-$purpose-${UUID.randomUUID()}.part"
    }

    internal fun remoteParentAndName(target: String): Pair<String, String> {
        val slash = target.lastIndexOf('/')
        val parent = when {
            slash < 0 -> "/"
            slash == 0 -> "/"
            else -> target.substring(0, slash)
        }
        val name = target.substring(slash + 1)
        require(name.isNotBlank()) { "Remote file name is required." }
        return parent to name
    }

    private fun ftpRemoteTargetExists(client: FTPClient, remoteFilePath: String): Boolean {
        val (parent, name) = remoteParentAndName(remoteFilePath)
        val entries = client.listFiles(parent)
        if (!FTPReply.isPositiveCompletion(client.replyCode)) {
            throw IOException(
                "Unable to verify the existing remote upload target before replacement."
            )
        }
        val target = entries.firstOrNull { it.name == name } ?: return false
        if (!target.isFile) {
            throw IOException("Remote upload target is not a regular file: $remoteFilePath")
        }
        return true
    }

    private fun sftpRemoteTargetExists(channel: ChannelSftp, remoteFilePath: String): Boolean {
        return try {
            val attrs = channel.lstat(remoteFilePath)
            if (attrs.isDir || attrs.isLink) {
                throw IOException("Remote upload target is not a regular file: $remoteFilePath")
            }
            true
        } catch (error: SftpException) {
            sftpTargetExistsFromLookupFailure(error)
        }
    }

    internal fun sftpTargetExistsFromLookupFailure(error: SftpException): Boolean {
        if (error.id == ChannelSftp.SSH_FX_NO_SUCH_FILE) {
            return false
        }
        throw IOException(
            "Unable to verify the existing remote upload target before replacement.",
            error
        )
    }

    private fun copyCancelable(
        input: InputStream,
        output: OutputStream,
        cancellation: OperationCancellation
    ): Long {
        val buffer = ByteArray(64 * 1024)
        var total = 0L
        while (true) {
            cancellation.throwIfCanceled()
            val read = input.read(buffer)
            if (read < 0) break
            if (read == 0) continue
            output.write(buffer, 0, read)
            total += read
        }
        output.flush()
        cancellation.throwIfCanceled()
        return total
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

    internal fun normalizedProfile(profile: ConnectionProfile): ConnectionProfile {
        val normalizedHost = normalizeHost(profile.host)
        require(normalizedHost.isNotBlank()) { "Host is required." }
        require(profile.port in 1..65535) { "Port must be between 1 and 65535." }
        if (profile.protocol == ConnectionProtocol.SFTP) {
            require(profile.keepAliveSeconds in 5..300) {
                "SFTP keep-alive interval must be between 5 and 300 seconds."
            }
        }
        return profile.copy(
            host = normalizedHost,
            remotePath = normalizeRemoteDirectory(profile.remotePath)
        )
    }

    internal fun normalizeHost(input: String): String {
        var value = input.trim()
        if (value.isBlank()) return ""
        require(value.none { it.isWhitespace() || it.isISOControl() }) {
            "Host must not contain whitespace or control characters."
        }

        val schemeIndex = value.indexOf("://")
        if (schemeIndex >= 0) {
            val scheme = value.substring(0, schemeIndex).lowercase()
            require(scheme in setOf("ftp", "ftps", "sftp")) { "Unsupported host scheme." }
            value = value.substring(schemeIndex + 3)
        }

        value = value.substringBefore('/').trim()
        require('@' !in value) { "Credentials must not be embedded in the host." }

        val host = when {
            value.startsWith('[') -> {
                val end = value.indexOf(']')
                require(end > 1) { "Invalid IPv6 host." }
                val remainder = value.substring(end + 1)
                require(remainder.isBlank()) { "Enter the port in the Port field." }
                value.substring(1, end)
            }
            value.count { it == ':' } > 1 -> value
            ':' in value -> throw IllegalArgumentException("Enter the port in the Port field.")
            else -> value
        }.trim()

        if (host.isBlank()) return ""
        return if (':' in host) host else IDN.toASCII(host)
    }

    internal fun normalizeRemoteDirectory(input: String): String {
        val value = input.trim().ifBlank { "/" }
        val target = if (value.startsWith('/')) value else "/$value"
        requireSafeRemoteSegments(target)
        return target
    }

    internal fun normalizeRemoteTarget(input: String): String {
        val value = input.trim()
        require(value.isNotBlank()) { "Remote path is required." }
        val target = if (value.startsWith('/')) value else "/$value"
        requireSafeRemoteSegments(target)
        return target
    }

    internal fun normalizeRemoteDeleteTarget(input: String): String {
        val target = normalizeRemoteTarget(input)
        require(target.split('/').any { it.isNotBlank() }) {
            "Refusing to delete the remote root path."
        }
        return target
    }

    internal fun joinRemotePath(directory: String, child: String): String {
        val safeChild = child.trim().trimStart('/')
        val base = directory.ifBlank { "/" }.trimEnd('/')
        return if (base.isBlank()) "/$safeChild" else "$base/$safeChild"
    }

    private fun requireSafeRemoteSegments(path: String) {
        require(path.split('/').filter { it.isNotBlank() }.none { it == "." || it == ".." }) {
            "Remote path must not contain dot path segments."
        }
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
