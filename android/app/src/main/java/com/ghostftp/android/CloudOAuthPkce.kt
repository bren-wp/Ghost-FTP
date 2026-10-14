package com.ghostftp.android

import java.net.URI
import java.net.URLEncoder
import java.security.MessageDigest
import java.security.SecureRandom
import java.util.Base64

/**
 * OAuth public-client request validation. Never persist verifier/state or log
 * the callback (the authorization code is a short-lived bearer secret).
 *
 * This is the protocol foundation, not a working Google Drive account flow:
 * a registered mobile client, redirect handler, Keystore token vault, token
 * exchange/refresh and real Drive API adapter are separately required.
 */
internal class CloudOAuthPkceAttempt internal constructor(
    val authorizationUrl: URI,
    val redirectUri: URI,
    val codeVerifier: String,
    private val expectedState: String
) {
    private var consumed = false

    @Synchronized
    fun consumeCallback(rawCallback: String): String {
        check(!consumed) { "OAuth authorization was already consumed." }
        consumed = true // Reject replay even when the first callback is invalid.
        val callback = try { URI(rawCallback) } catch (_: Exception) {
            throw IllegalArgumentException("Invalid OAuth callback.")
        }
        require(callback.scheme == redirectUri.scheme &&
            callback.rawAuthority == redirectUri.rawAuthority &&
            callback.rawPath == redirectUri.rawPath &&
            callback.rawFragment == null && callback.userInfo == null) {
            "Unexpected OAuth redirect."
        }
        val parts = callback.rawQuery?.split('&')?.map { pair ->
            val fields = pair.split('=', limit = 2)
            require(fields.size == 2) { "Invalid OAuth callback parameter." }
            fields[0] to java.net.URLDecoder.decode(fields[1], "UTF-8")
        } ?: emptyList()
        require(parts.size == parts.map { it.first }.toSet().size) {
            "Duplicate OAuth callback parameter."
        }
        val values = parts.toMap()
        val state = values["state"] ?: throw IllegalArgumentException("Missing OAuth state.")
        require(MessageDigest.isEqual(
            state.toByteArray(Charsets.UTF_8), expectedState.toByteArray(Charsets.UTF_8)
        )) { "OAuth state mismatch." }
        require(values["error"] == null) { "OAuth provider declined authorization." }
        return values["code"]?.takeIf { it.isNotBlank() && it.length <= 4096 }
            ?: throw IllegalArgumentException("Missing OAuth authorization code.")
    }
}

internal object CloudOAuthPkce {
    private val base64 = Base64.getUrlEncoder().withoutPadding()
    private const val DRIVE_SCOPE = "https://www.googleapis.com/auth/drive.file"
    private const val GOOGLE_AUTH = "https://accounts.google.com/o/oauth2/v2/auth"
    private val clientIdPattern = Regex("""^[0-9]+-[a-z0-9-]+\.apps\.googleusercontent\.com$""")

    private fun encode(value: String) =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    internal fun googleDrive(
        clientId: String,
        redirectRaw: String,
        random: SecureRandom = SecureRandom()
    ): CloudOAuthPkceAttempt {
        require(clientIdPattern.matches(clientId)) { "Registered Google OAuth client ID required." }
        val redirect = URI(redirectRaw)
        require(redirect.scheme != null && redirect.isAbsolute &&
            redirect.userInfo == null && redirect.rawQuery == null &&
            redirect.rawFragment == null && !redirect.rawPath.isNullOrBlank()) {
            "A registered, exact OAuth redirect URI is required."
        }
        val verifierBytes = ByteArray(32).also(random::nextBytes)
        val stateBytes = ByteArray(32).also(random::nextBytes)
        val verifier = base64.encodeToString(verifierBytes)
        val state = base64.encodeToString(stateBytes)
        val challenge = base64.encodeToString(
            MessageDigest.getInstance("SHA-256").digest(verifier.toByteArray(Charsets.US_ASCII))
        )
        val fields = linkedMapOf(
            "client_id" to clientId,
            "redirect_uri" to redirect.toASCIIString(),
            "response_type" to "code",
            "scope" to DRIVE_SCOPE,
            "state" to state,
            "code_challenge" to challenge,
            "code_challenge_method" to "S256",
            "access_type" to "offline"
        )
        val url = URI(GOOGLE_AUTH + "?" + fields.entries.joinToString("&") {
            encode(it.key) + "=" + encode(it.value)
        })
        return CloudOAuthPkceAttempt(url, redirect, verifier, state)
    }
}
