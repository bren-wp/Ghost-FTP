package com.ghostftp.android

import java.net.URI
import java.security.MessageDigest
import java.util.Base64
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CloudOAuthPkceTest {
    private val client = "123456789-example.apps.googleusercontent.com"
    private val redirect = "com.googleusercontent.apps.123456789-example:/oauth2redirect"

    @Test fun googleDriveUsesS256PkceRandomStateAndNarrowScope() {
        val first = CloudOAuthPkce.googleDrive(client, redirect)
        val second = CloudOAuthPkce.googleDrive(client, redirect)
        val url = first.authorizationUrl.toString()
        assertEquals("https", first.authorizationUrl.scheme)
        assertEquals("accounts.google.com", first.authorizationUrl.host)
        assertTrue(url.contains("code_challenge_method=S256"))
        assertTrue(url.contains("drive.file"))
        assertTrue(url.contains("response_type=code"))
        assertTrue(url.contains("access_type=offline"))
        assertFalse(url.contains("client_secret"))
        assertFalse(url.contains(first.codeVerifier))
        assertTrue(first.codeVerifier.length >= 43)
        assertFalse(first.codeVerifier == second.codeVerifier)
        assertFalse(first.authorizationUrl == second.authorizationUrl)
        val args = first.authorizationUrl.rawQuery.split('&')
            .associate { it.substringBefore('=') to java.net.URLDecoder.decode(it.substringAfter('='), "UTF-8") }
        val expected = Base64.getUrlEncoder().withoutPadding().encodeToString(
            MessageDigest.getInstance("SHA-256")
                .digest(first.codeVerifier.toByteArray(Charsets.US_ASCII))
        )
        assertEquals(expected, args["code_challenge"])
        assertEquals(redirect, args["redirect_uri"])
    }

    @Test fun callbackIsSingleUseAndExactState() {
        val auth = CloudOAuthPkce.googleDrive(client, redirect)
        val state = auth.authorizationUrl.rawQuery.split('&')
            .first { it.startsWith("state=") }.substringAfter('=')
        val code = auth.consumeCallback("$redirect?code=authorizedOnce&state=$state")
        assertEquals("authorizedOnce", code)
        try {
            auth.consumeCallback("$redirect?code=replay&state=$state")
            throw AssertionError("callback replay was permitted")
        } catch (_: IllegalStateException) {}
    }

    @Test fun mismatchedStateAndDuplicateParametersAreRejected() {
        for (suffix in listOf(
            "?code=x&state=wrong",
            "?code=x",
            "?code=x&code=two&state=wrong",
            "?code=x&state=wrong&state=wrong",
            "?error=access_denied&state=wrong",
            "#fragment"
        )) {
            val auth = CloudOAuthPkce.googleDrive(client, redirect)
            try {
                auth.consumeCallback("$redirect$suffix")
                throw AssertionError("invalid callback accepted")
            } catch (_: IllegalArgumentException) {}
        }
    }

    @Test fun rejectsWrongRedirectAndInvalidUnregisteredClientId() {
        for (badRedirect in listOf(
            "com.attacker.app:/oauth2redirect",
            "https://accounts.google.com/redirect",
            "com.googleusercontent.apps.123456789-example:/different"
        )) {
            val auth = CloudOAuthPkce.googleDrive(client, redirect)
            try {
                auth.consumeCallback("$badRedirect?code=x&state=wrong")
                throw AssertionError("wrong redirect accepted")
            } catch (_: IllegalArgumentException) {}
        }
        for (bad in listOf("", "secret", "evil.com", "abc.apps.googleusercontent.com")) {
            try {
                CloudOAuthPkce.googleDrive(bad, redirect)
                throw AssertionError("bad client accepted")
            } catch (_: IllegalArgumentException) {}
        }
    }
}
