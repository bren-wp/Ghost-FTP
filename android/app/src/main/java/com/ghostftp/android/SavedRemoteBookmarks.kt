package com.ghostftp.android

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject
import java.util.Locale

/**
 * Strictly local, non-secret bookmarks for actual saved site profiles.
 * Bookmarks do NOT reopen unauthenticated sessions or persist credentials.
 * Malformed stored data causes all mutations to fail closed.
 */
internal data class SavedRemoteBookmark(
    val title: String,
    val siteName: String,
    val remoteDirectory: String
)

internal object SavedRemoteBookmarks {
    private const val PREFS = "ghostftp.remote-bookmarks.v1"
    private const val KEY = "bookmarks"
    private const val MAX_ITEMS = 80
    private const val MAX_JSON_BYTES = 64 * 1024

    internal fun valid(item: SavedRemoteBookmark): Boolean {
        if (item.title.length !in 1..80 ||
            item.siteName.length !in 1..100 ||
            item.remoteDirectory.length !in 1..1024 ||
            !item.remoteDirectory.startsWith('/') ||
            item.remoteDirectory.contains('\\') ||
            item.remoteDirectory.contains("//") ||
            item.remoteDirectory.split('/').any { it == "." || it == ".." }) {
            return false
        }
        return listOf(item.title, item.siteName, item.remoteDirectory).all {
            it.isNotBlank() && it.none { ch -> ch.code < 32 || ch.code == 127 }
        }
    }

    internal fun added(
        current: List<SavedRemoteBookmark>,
        bookmark: SavedRemoteBookmark
    ): List<SavedRemoteBookmark>? {
        if (!valid(bookmark) || current.size >= MAX_ITEMS || current.any { !valid(it) }) {
            return null
        }
        if (current.any {
                it.siteName.equals(bookmark.siteName, ignoreCase = true) &&
                    it.remoteDirectory == bookmark.remoteDirectory
            }) return null
        return current + bookmark
    }

    internal fun without(
        current: List<SavedRemoteBookmark>,
        bookmark: SavedRemoteBookmark
    ): List<SavedRemoteBookmark>? {
        if (current.any { !valid(it) }) return null
        val matches = current.indices.filter { current[it] == bookmark }
        if (matches.size != 1) return null
        return current.filterIndexed { index, _ -> index != matches.single() }
    }

    private fun load(context: Context): List<SavedRemoteBookmark>? {
        val value = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY, null) ?: return emptyList()
        if (value.toByteArray(Charsets.UTF_8).size > MAX_JSON_BYTES) return null
        return runCatching {
            val array = JSONArray(value)
            if (array.length() > MAX_ITEMS) return@runCatching null
            val rows = (0 until array.length()).map { index ->
                val entry = array.getJSONObject(index)
                SavedRemoteBookmark(
                    title = entry.getString("title"),
                    siteName = entry.getString("siteName"),
                    remoteDirectory = entry.getString("remoteDirectory")
                )
            }
            if (rows.any { !valid(it) }) return@runCatching null
            if (rows.any { item -> rows.count {
                    it.siteName.equals(item.siteName, ignoreCase = true) &&
                        it.remoteDirectory == item.remoteDirectory
                } != 1 }) return@runCatching null
            rows
        }.getOrNull()
    }

    fun read(context: Context): List<SavedRemoteBookmark> =
        (load(context) ?: emptyList()).sortedWith(
            compareBy({ it.title.lowercase(Locale.ROOT) }, { it.siteName.lowercase(Locale.ROOT) })
        )

    private fun persist(context: Context, items: List<SavedRemoteBookmark>): Boolean {
        if (items.size > MAX_ITEMS || items.any { !valid(it) }) return false
        val array = JSONArray()
        items.forEach { item ->
            array.put(JSONObject().apply {
                put("title", item.title)
                put("siteName", item.siteName)
                put("remoteDirectory", item.remoteDirectory)
            })
        }
        val raw = array.toString()
        if (raw.toByteArray(Charsets.UTF_8).size > MAX_JSON_BYTES) return false
        return context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit().putString(KEY, raw).commit()
    }

    fun add(context: Context, item: SavedRemoteBookmark): Boolean {
        val current = load(context) ?: return false
        val changed = added(current, item) ?: return false
        return persist(context, changed)
    }

    fun remove(context: Context, item: SavedRemoteBookmark): Boolean {
        val current = load(context) ?: return false
        val changed = without(current, item) ?: return false
        return persist(context, changed)
    }
}
