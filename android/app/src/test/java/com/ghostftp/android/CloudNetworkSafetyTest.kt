package com.ghostftp.android

import java.io.InterruptedIOException
import java.net.InetAddress
import java.net.Proxy
import java.net.UnknownHostException
import okhttp3.Dns
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test

class CloudNetworkSafetyTest {
    private fun ip(vararg octets: Int): InetAddress =
        InetAddress.getByAddress(octets.map { it.toByte() }.toByteArray())

    private fun dns(vararg addresses: InetAddress): Dns = object : Dns {
        override fun lookup(hostname: String): List<InetAddress> = addresses.toList()
    }

    @Test fun acceptsOnlyGloballyRoutableAddresses() {
        val good = listOf(
            ip(8, 8, 8, 8),
            ip(1, 1, 1, 1),
            InetAddress.getByName("2606:4700:4700::1111"),
            InetAddress.getByName("2001:4860:4860::8888")
        )
        good.forEach { assertTrue("Rejected public address ${it.hostAddress}", PublicCloudDns.isPublicAddress(it)) }

        val unsafe = listOf(
            ip(0, 0, 0, 1), ip(10, 0, 0, 1),
            ip(100, 64, 0, 1), ip(100, 100, 100, 200),
            ip(127, 0, 0, 1), ip(169, 254, 169, 254),
            ip(172, 16, 0, 1), ip(192, 168, 1, 1),
            ip(192, 0, 2, 5), ip(192, 0, 0, 1),
            ip(198, 18, 0, 1), ip(198, 51, 100, 1),
            ip(203, 0, 113, 7), ip(224, 0, 0, 1),
            ip(240, 0, 0, 1),
            InetAddress.getByName("::1"),
            InetAddress.getByName("fc00::1"),
            InetAddress.getByName("fe80::1"),
            InetAddress.getByName("2001:db8::1"),
            InetAddress.getByName("2001:2::1234"),  // IANA benchmarking
            InetAddress.getByName("2001:10::1"),   // deprecated ORCHID
            InetAddress.getByName("2001:20::1"),   // RFC 7343 ORCHIDv2
            InetAddress.getByName("2001:2f::1"),   // last /28 nibble
            InetAddress.getByName("3fff::1"),      // RFC 9637 documentation
            InetAddress.getByName("2002::1")
        )
        unsafe.forEach { assertFalse("Accepted nonpublic address ${it.hostAddress}", PublicCloudDns.isPublicAddress(it)) }
    }

    @Test fun refusesEntireDnsAnswerIfAnyAddressIsUnsafe() {
        val host = "bucket.s3.amazonaws.com"
        val safe = ip(8, 8, 4, 4)
        val mixed = PublicCloudDns(dns(safe, ip(100, 100, 100, 200)))
        try {
            mixed.lookup(host)
            fail("Mixed safe/private resolution must never be used for cloud traffic.")
        } catch (_: UnknownHostException) {
            // OkHttp receives no connectable addresses.
        }
        assertEquals(listOf(safe), PublicCloudDns(dns(safe)).lookup(host))
    }

    @Test fun rejectsArbitraryHostsEvenWithPublicDnsAnswers() {
        val lookup = PublicCloudDns(dns(ip(8, 8, 8, 8)))
        try {
            lookup.lookup("attacker.example")
            fail("DNS check must not authorize arbitrary upload destinations.")
        } catch (_: UnknownHostException) {
            // Correct.
        }
        try {
            PublicCloudDns(dns()).lookup("bucket.s3.amazonaws.com")
            fail("Empty DNS answer must be rejected.")
        } catch (_: UnknownHostException) {
            // Correct.
        }
        for (invalid in listOf(
            "xn--bcher-kva.s3.amazonaws.com",
            "under_score.s3.amazonaws.com",
            "bucket-.s3.amazonaws.com"
        )) {
            try {
                PublicCloudDns(dns(ip(8, 8, 8, 8))).lookup(invalid)
                fail("Cloud DNS hook accepted ambiguous provider hostname: $invalid")
            } catch (_: UnknownHostException) {
                // Reject before upstream DNS resolution can be used.
            }
        }
    }

    @Test fun disablesRedirectsAutomaticRetriesAndProxyRouting() {
        val client = CloudHttpClient.create()
        try {
            assertEquals(Proxy.NO_PROXY, client.proxy)
            assertFalse(client.followRedirects)
            assertFalse(client.followSslRedirects)
            assertFalse(client.retryOnConnectionFailure)
            assertTrue(client.dns is PublicCloudDns)
        } finally {
            client.connectionPool.evictAll()
        }
    }

    @Test fun cancellationIsIdempotentAndStopsNewWork() {
        val cancellation = CloudTransferCancellation()
        cancellation.ensureActive()
        cancellation.cancel()
        cancellation.cancel()
        assertTrue(cancellation.isCancelled)
        try {
            cancellation.ensureActive()
            fail("Canceled cloud work must not continue.")
        } catch (_: InterruptedIOException) {
            // Correct.
        }
    }
}
