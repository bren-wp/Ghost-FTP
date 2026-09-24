package com.ghostftp.android

import java.net.IDN
import java.net.InetSocketAddress
import java.net.Socket

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
    fun probe(profile: ConnectionProfile): ConnectionProbeResult {
        val normalizedHost = normalizeHost(profile.host)
        require(normalizedHost.isNotBlank()) { "Host is required." }
        require(profile.port in 1..65535) { "Port must be between 1 and 65535." }

        Socket().use { socket ->
            socket.soTimeout = CONNECT_TIMEOUT_MS
            socket.connect(InetSocketAddress(normalizedHost, profile.port), CONNECT_TIMEOUT_MS)
        }

        val path = profile.remotePath.ifBlank { "/" }
        return ConnectionProbeResult(
            reachable = true,
            title = "Endpoint reachable",
            detail = "${profile.protocol.label} endpoint ${redactHost(normalizedHost)}:${profile.port} accepted a network session.",
            rows = listOf(
                RemoteRow("Remote path", path),
                RemoteRow("Protocol", profile.protocol.label),
                RemoteRow("Account", profile.username.ifBlank { "Ask on connect" }),
                RemoteRow("Privacy", "Credentials are not stored by this Android build.")
            )
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

    private fun redactHost(host: String): String {
        if (host.length <= 6) return host
        val first = host.take(3)
        val last = host.takeLast(3)
        return "$first…$last"
    }

    private companion object {
        const val CONNECT_TIMEOUT_MS = 7000
    }
}
