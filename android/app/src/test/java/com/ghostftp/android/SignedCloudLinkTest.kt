package com.ghostftp.android

import org.junit.Assert.*
import org.junit.Test

class SignedCloudLinkTest {
    @Test fun acceptsSignedPublicHttpsLinks() {
        assertEquals("bucket.s3.amazonaws.com",
            SignedCloudLink("https://bucket.s3.amazonaws.com/key?X-Amz-Signature=abc").publicHost)
        SignedCloudLink("https://storage.googleapis.com/bucket/file?X-Goog-Signature=abc")
        SignedCloudLink("https://account.blob.core.windows.net/files/test?sig=abc")
        SignedCloudLink("https://account.r2.cloudflarestorage.com/bucket/object?X-Amz-Signature=abc")
    }

    @Test fun rejectsUnsignedPrivateAndInsecureEndpoints() {
        val rejected = listOf(
            "http://bucket.s3.amazonaws.com/a?X-Amz-Signature=x",
            "https://bucket.s3.amazonaws.com/a",
            "https://user:pass@bucket.s3.amazonaws.com/a?sig=x",
            "https://bucket.s3.amazonaws.com/a?sig=x#fragment",
            "https://bucket.s3.amazonaws.com:444/a?sig=x",
            "https://127.0.0.1/a?sig=x",
            "https://localhost/a?sig=x",
            "https://192.168.1.9/a?sig=x",
            "https://10.1.2.3/a?sig=x",
            "https://172.16.0.1/a?sig=x",
            "https://169.254.2.3/a?sig=x",
            "https://metadata.internal/a?sig=x",
            "https://example.com/a?sig=x",
            "https://bucket.s3.amazonaws.com.attacker.example/a?X-Amz-Signature=x",
            "https://bucket.s3.amazonaws.com/a?X-Amz-Signature=",
            "https://bucket.s3.amazonaws.com/a?X-Amz-Signature=x&X-Amz-Signature=y",
            "https://bucket.s3.amazonaws.com/a?sig=x",
            "https://storage.googleapis.com/bucket/file?X-Amz-Signature=x",
            "https://account.blob.core.windows.net/a/b?X-Goog-Signature=x",
            "https://account.blob.core.windows.net/a/b?sig=",
            "https://account.blob.core.windows.net/a/b?sig=x&sig=y"
        )
        rejected.forEach { value ->
            try {
                SignedCloudLink(value)
                fail("Unsafe signed cloud link was accepted.")
            } catch (_: IllegalArgumentException) {
                // Expected: no network request is made.
            }
        }
    }
}
