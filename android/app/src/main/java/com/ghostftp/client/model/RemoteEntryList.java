package com.GhostFTP.client.model;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;

/** Pure list presentation rules shared by the Android mobile file browser. */
public final class RemoteEntryList {
    private static final Comparator<RemoteEntry> DISPLAY_ORDER = Comparator
        .comparing(RemoteEntry::directory).reversed()
        .thenComparing(RemoteEntry::name, String.CASE_INSENSITIVE_ORDER)
        .thenComparing(RemoteEntry::name);

    private RemoteEntryList() {}

    public static List<RemoteEntry> sorted(List<RemoteEntry> source) {
        List<RemoteEntry> result = new ArrayList<>(source);
        result.sort(DISPLAY_ORDER);
        return result;
    }

    public static List<RemoteEntry> filtered(List<RemoteEntry> sortedSource, String rawQuery) {
        String query = rawQuery == null ? "" : rawQuery.trim().toLowerCase(Locale.ROOT);
        if (query.isEmpty()) return new ArrayList<>(sortedSource);
        List<RemoteEntry> result = new ArrayList<>();
        for (RemoteEntry entry : sortedSource) {
            if (entry.name().toLowerCase(Locale.ROOT).contains(query)) result.add(entry);
        }
        return result;
    }
}
