package com.ghostftp.android

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject

/**
 * Saved site metadata only. No password, OAuth token or key passphrase is
 * serialized; a user explicitly supplies credentials for each new session.
 */
internal data class SavedSiteDefinition(
    val name: String,
    val protocol: ConnectionProtocol,
    val host: String,
    val port: Int,
    val username: String,
    val fingerprint: String,
    val remotePath: String,
    val keepAliveSeconds: Int
)

internal object SavedSiteProfiles {
    private const val PREFS = "ghostftp.saved-sites.v1"
    private const val KEY = "sites"
    private const val MAX_SITES = 100
    private const val MAX_JSON_BYTES = 128 * 1024

    fun read(context: Context): List<SavedSiteDefinition> {
        val raw = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY, null) ?: return emptyList()
        if (raw.toByteArray(Charsets.UTF_8).size > MAX_JSON_BYTES) return emptyList()
        return runCatching {
            val array = JSONArray(raw)
            if (array.length() > MAX_SITES) return emptyList()
            val saved = mutableListOf<SavedSiteDefinition>()
            for (index in 0 until array.length()) {
                val item = array.getJSONObject(index)
                val protocol = ConnectionProtocol.entries.firstOrNull {
                    it.name == item.optString("protocol")
                } ?: continue
                val site = SavedSiteDefinition(
                    name = item.optString("name"),
                    protocol = protocol,
                    host = item.optString("host"),
                    port = item.optInt("port"),
                    username = item.optString("username"),
                    fingerprint = item.optString("fingerprint"),
                    remotePath = item.optString("remotePath", "/"),
                    keepAliveSeconds = item.optInt("keepAliveSeconds", 15)
                )
                if (valid(site)) saved.add(site)
            }
            saved.toList()
        }.getOrDefault(emptyList())
    }

    fun save(context: Context, name: String, profile: ConnectionProfile): Boolean {
        val candidate = SavedSiteDefinition(
            name = name.trim(),
            protocol = profile.protocol,
            host = profile.host.trim(),
            port = profile.port,
            username = profile.username.trim(),
            fingerprint = profile.hostKeyFingerprint.trim(),
            remotePath = profile.remotePath.trim().ifEmpty { "/" },
            keepAliveSeconds = profile.keepAliveSeconds
        )
        if (!valid(candidate)) return false
        val saved = read(context)
        if (saved.size >= MAX_SITES ||
            saved.any { it.name.equals(candidate.name, ignoreCase = true) }) return false
        val result = JSONArray()
        for (site in saved + candidate) {
            result.put(JSONObject().apply {
                put("name", site.name)
                put("protocol", site.protocol.name)
                put("host", site.host)
                put("port", site.port)
                put("username", site.username)
                put("fingerprint", site.fingerprint)
                put("remotePath", site.remotePath)
                put("keepAliveSeconds", site.keepAliveSeconds)
            })
        }
        val encoded = result.toString()
        if (encoded.toByteArray(Charsets.UTF_8).size > MAX_JSON_BYTES) return false
        return context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit().putString(KEY, encoded).commit()
    }

    private fun valid(site: SavedSiteDefinition): Boolean =
        site.name.length in 1..100 && site.host.length in 1..255 &&
            site.username.length <= 255 && site.port in 1..65535 &&
            site.fingerprint.length <= 255 && site.remotePath.length <= 1024 &&
            site.keepAliveSeconds in 5..300 &&
            listOf(site.name, site.host, site.username, site.fingerprint, site.remotePath)
                .all { field -> field.none(Char::isISOControl) }
}
