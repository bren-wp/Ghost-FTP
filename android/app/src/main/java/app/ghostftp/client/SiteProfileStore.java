package app.ghostftp.client;

import android.content.SharedPreferences;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

final class SiteProfileStore {
    private static final String KEY = "siteProfilesV1";
    private static final int MAX_PROFILES = 50;
    private static final int MAX_BOOKMARKS_PER_KIND = 50;

    private final SharedPreferences preferences;

    SiteProfileStore(SharedPreferences preferences) {
        this.preferences = preferences;
    }

    List<SiteProfile> load() {
        String raw = preferences.getString(KEY, "");
        if (raw == null || raw.trim().isEmpty()) return Collections.emptyList();
        try {
            JSONArray array = new JSONArray(raw);
            List<SiteProfile> result = new ArrayList<>();
            int count = Math.min(array.length(), MAX_PROFILES);
            for (int i = 0; i < count; i++) {
                JSONObject object = array.getJSONObject(i);
                result.add(fromJson(object));
            }
            return result;
        } catch (JSONException | IllegalArgumentException e) {
            return Collections.emptyList();
        }
    }

    void save(List<SiteProfile> profiles) {
        JSONArray array = new JSONArray();
        int count = Math.min(profiles == null ? 0 : profiles.size(), MAX_PROFILES);
        try {
            for (int i = 0; i < count; i++) {
                array.put(toJson(profiles.get(i)));
            }
        } catch (JSONException e) {
            throw new IllegalStateException("Could not encode site profiles.", e);
        }
        preferences.edit().putString(KEY, array.toString()).apply();
    }

    private static JSONObject toJson(SiteProfile profile) throws JSONException {
        JSONObject object = new JSONObject();
        object.put("id", profile.id);
        object.put("name", profile.name);
        object.put("protocol", profile.protocol);
        object.put("host", profile.host);
        object.put("port", profile.port);
        object.put("username", profile.username);
        object.put("localStartTreeUri", profile.localStartTreeUri);
        object.put("remoteStartPath", profile.remoteStartPath);
        object.put("localBookmarks", toJsonArray(profile.localBookmarks));
        object.put("remoteBookmarks", toJsonArray(profile.remoteBookmarks));
        return object;
    }

    private static SiteProfile fromJson(JSONObject object) throws JSONException {
        return new SiteProfile(
                object.getString("id"),
                object.getString("name"),
                object.getString("protocol"),
                object.getString("host"),
                object.getInt("port"),
                object.optString("username", ""),
                object.optString("localStartTreeUri", ""),
                object.optString("remoteStartPath", "/"),
                strings(object.optJSONArray("localBookmarks")),
                strings(object.optJSONArray("remoteBookmarks")));
    }

    private static JSONArray toJsonArray(List<String> values) {
        JSONArray array = new JSONArray();
        int count = Math.min(values == null ? 0 : values.size(), MAX_BOOKMARKS_PER_KIND);
        for (int i = 0; i < count; i++) array.put(values.get(i));
        return array;
    }

    private static List<String> strings(JSONArray array) throws JSONException {
        if (array == null) return Collections.emptyList();
        List<String> result = new ArrayList<>();
        int count = Math.min(array.length(), MAX_BOOKMARKS_PER_KIND);
        for (int i = 0; i < count; i++) result.add(array.getString(i));
        return result;
    }
}
