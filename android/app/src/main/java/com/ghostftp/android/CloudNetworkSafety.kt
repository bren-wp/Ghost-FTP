package com.ghostftp.android

import java.io.InterruptedIOException
import java.net.InetAddress
import java.net.Proxy
import java.net.UnknownHostException
import java.util.Locale
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicReference
import okhttp3.Call
import okhttp3.ConnectionPool
import okhttp3.Dns
import okhttp3.OkHttpClient

/**
 * Enforces the public-address policy in the HTTP client's actual DNS selection
 * path. Unlike a separate DNS preflight, these are the addresses OkHttp uses
 * to open the connection. Direct-only networking prevents proxy-side DNS
 * resolution from silently bypassing this guard.
 */
internal class PublicCloudDns(private val upstream: Dns = Dns.SYSTEM) : Dns {
    override fun lookup(hostname: String): List<InetAddress> {
        // Revalidate canonical host syntax inside the DNS hook itself.
        // A later caller must not bypass URL validation by using OkHttp
        // directly with an ambiguous IDNA/hostname spelling.
        val canonicalHost = hostname.lowercase(Locale.ROOT)
        if (!SignedCloudLink.isPublicHost(canonicalHost) ||
            !SignedCloudLink.isKnownProviderHost(canonicalHost)) {
            throw UnknownHostException("Cloud storage provider hostname is not permitted.")
        }
        val addresses = upstream.lookup(hostname)
        if (addresses.isEmpty() || addresses.any { !isPublicAddress(it) }) {
            throw UnknownHostException("Cloud storage host did not resolve exclusively to public addresses.")
        }
        return addresses
    }

    companion object {
        internal fun isPublicAddress(address: InetAddress): Boolean {
            if (address.isAnyLocalAddress || address.isLoopbackAddress ||
                address.isLinkLocalAddress || address.isSiteLocalAddress ||
                address.isMulticastAddress) return false
            val bytes = address.address.map { it.toInt() and 255 }
            return when (bytes.size) {
                4 -> {
                    val a = bytes[0]
                    val b = bytes[1]
                    val c = bytes[2]
                    a in 1..223 &&
                        !(a == 100 && b in 64..127) && // shared/CGNAT (includes metadata endpoints)
                        !(a == 192 && b == 0 && c == 0) &&
                        !(a == 192 && b == 0 && c == 2) &&
                        !(a == 192 && b == 88 && c == 99) &&
                        !(a == 198 && b in 18..19) &&
                        !(a == 198 && b == 51 && c == 100) &&
                        !(a == 203 && b == 0 && c == 113)
                }
                16 -> {
                    // Global-unicast IPv6 only. Exclude documentation and
                    // transition/tunnel ranges until explicitly reviewed.
                    bytes[0] in 0x20..0x3f &&
                        !(bytes[0] == 0x20 && bytes[1] == 0x01 &&
                          bytes[2] == 0x0d && bytes[3] == 0xb8) &&
                        !(bytes[0] == 0x20 && bytes[1] == 0x02) &&
                        !(bytes[0] == 0x20 && bytes[1] == 0x01 &&
                          bytes[2] == 0x00 && bytes[3] == 0x00) &&
                        // IANA non-global IPv6 special-purpose blocks:
                        // 2001:2::/48 benchmarking, 2001:10::/28 deprecated
                        // ORCHID, and 3fff::/20 documentation (RFC 9637).
                        !(bytes[0] == 0x20 && bytes[1] == 0x01 &&
                          bytes[2] == 0 && bytes[3] == 2 &&
                          bytes[4] == 0 && bytes[5] == 0) &&
                        !(bytes[0] == 0x20 && bytes[1] == 0x01 &&
                          bytes[2] == 0 && (bytes[3] and 0xf0) == 0x10) &&
                        !(bytes[0] == 0x3f && bytes[1] == 0xff &&
                          (bytes[2] and 0xf0) == 0)
                }
                else -> false
            }
        }
    }
}

/** Cancel the active request and its I/O, not merely the UI worker thread. */
internal class CloudTransferCancellation {
    private val cancelled = AtomicBoolean(false)
    private val active = AtomicReference<Call?>(null)

    fun attach(call: Call) {
        check(active.compareAndSet(null, call)) { "A cloud transfer is already in progress." }
        if (cancelled.get()) call.cancel()
    }

    fun cancel() {
        cancelled.set(true)
        active.get()?.cancel()
    }

    fun ensureActive() {
        if (cancelled.get() || Thread.currentThread().isInterrupted) {
            throw InterruptedIOException("Cloud transfer canceled.")
        }
    }

    fun clear(call: Call) {
        active.compareAndSet(call, null)
    }

    val isCancelled: Boolean get() = cancelled.get()
}

internal object CloudHttpClient {
    fun create(): OkHttpClient = OkHttpClient.Builder()
        .dns(PublicCloudDns())
        .proxy(Proxy.NO_PROXY)
        .followRedirects(false)
        .followSslRedirects(false)
        .retryOnConnectionFailure(false)
        .connectTimeout(20, TimeUnit.SECONDS)
        .readTimeout(120, TimeUnit.SECONDS)
        .writeTimeout(120, TimeUnit.SECONDS)
        .callTimeout(60, TimeUnit.MINUTES)
        .connectionPool(ConnectionPool(0, 1, TimeUnit.SECONDS))
        // Keep the platform's default certificate chain and hostname verifier.
        // Never add unsafe trust managers or HTTP fallbacks.
        .build()
}
