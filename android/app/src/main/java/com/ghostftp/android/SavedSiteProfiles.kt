package com.ghostftp.android

import android.content.Context
import java.util.Locale
import org.json.JSONArray
import org.json.JSONObject

/**
 * Saved site metadata only. Credentials are never serialized. If stored data
 * cannot be fully decoded, mutations fail closed rather than deleting entries.
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

    internal fun ordered(sites: List<SavedSiteDefinition>): List<SavedSiteDefinition> =
        sites.sortedWith(compareBy({ it.name.lowercase(Locale.ROOT) }, { it.name }))

    private fun decode(raw: String): List<SavedSiteDefinition>? {
        if (raw.toByteArray(Charsets.UTF_8).size > MAX_JSON_BYTES) return null
        return runCatching {
            val array = JSONArray(raw)
            if (array.length() > MAX_SITES) return@runCatching null
            val saved = mutableListOf<SavedSiteDefinition>()
            for (index in 0 until array.length()) {
                val item = array.getJSONObject(index)
                val protocol = ConnectionProtocol.entries.firstOrNull {
                    it.name == item.optString("protocol")
                } ?: return@runCatching null
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
                if (!valid(site) || saved.any { it.name.equals(site.name, ignoreCase = true) }) {
                    return@runCatching null
                }
                saved.add(site)
            }
            saved.toList()
        }.getOrNull()
    }

    private fun load(context: Context): List<SavedSiteDefinition>? {
        val raw = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY, null) ?: return emptyList()
        return decode(raw)
    }

    fun read(context: Context): List<SavedSiteDefinition> =
        ordered(load(context) ?: emptyList())

    private fun fromProfile(name: String, profile: ConnectionProfile) = SavedSiteDefinition(
        name = name.trim(), protocol = profile.protocol,
        host = profile.host.trim(), port = profile.port,
        username = profile.username.trim(),
        fingerprint = profile.hostKeyFingerprint.trim(),
        remotePath = profile.remotePath.trim().ifEmpty { "/" },
        keepAliveSeconds = profile.keepAliveSeconds
    )

    private fun persist(context: Context, sites: List<SavedSiteDefinition>): Boolean {
        if (sites.size > MAX_SITES || sites.any { !valid(it) }) return false
        val array = JSONArray()
        for (site in ordered(sites)) {
            array.put(JSONObject().apply {
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
        val encoded = array.toString()
        if (encoded.toByteArray(Charsets.UTF_8).size > MAX_JSON_BYTES) return false
        val editor = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit()
        if (sites.isEmpty()) editor.remove(KEY) else editor.putString(KEY, encoded)
        return editor.commit()
    }

    fun save(context: Context, name: String, profile: ConnectionProfile): Boolean {
        val candidate = fromProfile(name, profile)
        if (!valid(candidate)) return false
        val saved = load(context) ?: return false
        if (saved.size >= MAX_SITES ||
            saved.any { it.name.equals(candidate.name, ignoreCase = true) }) return false
        return persist(context, saved + candidate)
    }

    /** Replace exactly one chosen profile, rejecting ambiguous or duplicate names. */
    internal fun updated(
        saved: List<SavedSiteDefinition>,
        originalName: String,
        candidate: SavedSiteDefinition
    ): List<SavedSiteDefinition>? {
        if (!valid(candidate)) return null
        val matches = saved.indices.filter { saved[it].name.equals(originalName, ignoreCase = true) }
        if (matches.size != 1) return null
        val index = matches.single()
        if (saved.indices.any {
                it != index && saved[it].name.equals(candidate.name, ignoreCase = true)
            }) return null
        return saved.toMutableList().apply { this[index] = candidate }
    }

    internal fun without(
        saved: List<SavedSiteDefinition>,
        name: String
    ): List<SavedSiteDefinition>? {
        val matches = saved.indices.filter { saved[it].name.equals(name, ignoreCase = true) }
        if (matches.size != 1) return null
        return saved.filterIndexed { index, _ -> index != matches.single() }
    }

    fun update(context: Context, originalName: String, name: String, profile: ConnectionProfile): Boolean {
        val saved = load(context) ?: return false
        val next = updated(saved, originalName, fromProfile(name, profile)) ?: return false
        return persist(context, next)
    }

    fun remove(context: Context, name: String): Boolean {
        val saved = load(context) ?: return false
        val next = without(saved, name) ?: return false
        return persist(context, next)
    }

    private fun valid(site: SavedSiteDefinition): Boolean =
        site.name.length in 1..100 && site.host.length in 1..255 &&
            site.username.length <= 255 && site.port in 1..65535 &&
            site.fingerprint.length <= 255 && site.remotePath.length <= 1024 &&
            site.keepAliveSeconds in 5..300 &&
            listOf(site.name, site.host, site.username, site.fingerprint, site.remotePath)
                .all { field -> field.none { char -> char.code < 32 || char.code == 127 } }
}
