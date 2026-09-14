package app.ghostftp.client;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

final class WorkspaceOps {
    static final int MAX_SEARCH_RESULTS = 500;
    static final int MAX_SEARCH_DIRECTORIES = 2000;
    static final int MAX_SEARCH_DEPTH = 32;
    static final long MAX_REMOTE_SEARCH_MILLIS = 45_000L;
    static final int MAX_REMOTE_EDIT_BYTES = 1024 * 1024;

    enum SortKey {
        NAME,
        TYPE,
        SIZE,
        MODIFIED,
        PERMISSIONS
    }

    enum Difference {
        SAME,
        ONLY_LOCAL,
        ONLY_REMOTE,
        DIFFERENT
    }

    static final class Item {
        final int sourceIndex;
        final String name;
        final boolean directory;
        final long size;
        final long modifiedEpochMillis;
        final String permissions;

        Item(int sourceIndex, String name, boolean directory, long size, long modifiedEpochMillis, String permissions) {
            this.sourceIndex = sourceIndex;
            this.name = requireName(name);
            this.directory = directory;
            this.size = Math.max(0L, size);
            this.modifiedEpochMillis = Math.max(0L, modifiedEpochMillis);
            this.permissions = permissions == null ? "" : permissions.trim();
        }
    }

    static final class Comparison {
        final String name;
        final Item local;
        final Item remote;
        final Difference difference;

        Comparison(String name, Item local, Item remote, Difference difference) {
            this.name = requireName(name);
            this.local = local;
            this.remote = remote;
            this.difference = difference;
        }

        boolean canSynchronizeDirectoryNavigation() {
            return local != null
                    && remote != null
                    && local.directory
                    && remote.directory
                    && local.name.equals(remote.name);
        }
    }

    private WorkspaceOps() {
    }

    static List<Item> filterAndSort(List<Item> source, String query, SortKey key, boolean ascending) {
        if (source == null || source.isEmpty()) {
            return Collections.emptyList();
        }
        String needle = query == null ? "" : query.trim().toLowerCase(Locale.ROOT);
        List<Item> visible = new ArrayList<>();
        for (Item item : source) {
            if (item == null) {
                continue;
            }
            if (needle.isEmpty() || item.name.toLowerCase(Locale.ROOT).contains(needle)) {
                visible.add(item);
            }
        }
        SortKey safeKey = key == null ? SortKey.NAME : key;
        Comparator<Item> comparator = (left, right) -> {
            if (left.directory != right.directory) {
                return left.directory ? -1 : 1;
            }
            int value;
            switch (safeKey) {
                case TYPE:
                    value = compareType(left, right);
                    break;
                case SIZE:
                    value = Long.compare(left.size, right.size);
                    break;
                case MODIFIED:
                    value = Long.compare(left.modifiedEpochMillis, right.modifiedEpochMillis);
                    break;
                case PERMISSIONS:
                    value = compareText(left.permissions, right.permissions);
                    break;
                case NAME:
                default:
                    value = compareText(left.name, right.name);
                    break;
            }
            if (!ascending) {
                value = -value;
            }
            if (value != 0) {
                return value;
            }
            return compareText(left.name, right.name);
        };
        visible.sort(comparator);
        return visible;
    }

    static List<Comparison> compareDirectories(List<Item> local, List<Item> remote) {
        Map<String, Item> localByName = byName(local);
        Map<String, Item> remoteByName = byName(remote);
        List<String> names = new ArrayList<>();
        names.addAll(localByName.keySet());
        for (String name : remoteByName.keySet()) {
            if (!localByName.containsKey(name)) {
                names.add(name);
            }
        }
        names.sort(WorkspaceOps::compareText);

        List<Comparison> rows = new ArrayList<>(names.size());
        for (String name : names) {
            Item left = localByName.get(name);
            Item right = remoteByName.get(name);
            Difference difference;
            if (left == null) {
                difference = Difference.ONLY_REMOTE;
            } else if (right == null) {
                difference = Difference.ONLY_LOCAL;
            } else if (equivalent(left, right)) {
                difference = Difference.SAME;
            } else {
                difference = Difference.DIFFERENT;
            }
            rows.add(new Comparison(name, left, right, difference));
        }
        return rows;
    }

    static boolean matchesSearch(String name, String query) {
        if (name == null) {
            return false;
        }
        String needle = query == null ? "" : query.trim().toLowerCase(Locale.ROOT);
        return !needle.isEmpty() && name.toLowerCase(Locale.ROOT).contains(needle);
    }

    static SortKey nextLocalSortKey(SortKey key) {
        SortKey safe = key == null ? SortKey.NAME : key;
        switch (safe) {
            case NAME:
                return SortKey.TYPE;
            case TYPE:
                return SortKey.SIZE;
            case SIZE:
                return SortKey.MODIFIED;
            case MODIFIED:
            case PERMISSIONS:
            default:
                return SortKey.NAME;
        }
    }

    static SortKey nextRemoteSortKey(SortKey key) {
        SortKey safe = key == null ? SortKey.NAME : key;
        switch (safe) {
            case NAME:
                return SortKey.TYPE;
            case TYPE:
                return SortKey.SIZE;
            case SIZE:
                return SortKey.MODIFIED;
            case MODIFIED:
                return SortKey.PERMISSIONS;
            case PERMISSIONS:
            default:
                return SortKey.NAME;
        }
    }

    static String sortLabel(SortKey key, boolean ascending) {
        SortKey safe = key == null ? SortKey.NAME : key;
        String label = safe.name().charAt(0) + safe.name().substring(1).toLowerCase(Locale.ROOT);
        return label + (ascending ? " ↑" : " ↓");
    }

    private static Map<String, Item> byName(List<Item> items) {
        Map<String, Item> result = new LinkedHashMap<>();
        if (items == null) {
            return result;
        }
        for (Item item : items) {
            if (item != null) {
                result.put(item.name, item);
            }
        }
        return result;
    }

    private static boolean equivalent(Item left, Item right) {
        if (left.directory != right.directory) {
            return false;
        }
        if (left.directory) {
            return true;
        }
        if (left.size != right.size) {
            return false;
        }
        return left.modifiedEpochMillis > 0L
                && right.modifiedEpochMillis > 0L
                && left.modifiedEpochMillis == right.modifiedEpochMillis;
    }

    private static int compareType(Item left, Item right) {
        String leftType = extension(left.name);
        String rightType = extension(right.name);
        return compareText(leftType, rightType);
    }

    private static String extension(String name) {
        int dot = name.lastIndexOf('.');
        if (dot <= 0 || dot == name.length() - 1) {
            return "";
        }
        return name.substring(dot + 1);
    }

    private static int compareText(String left, String right) {
        int value = left.compareToIgnoreCase(right);
        return value != 0 ? value : left.compareTo(right);
    }

    private static String requireName(String value) {
        if (value == null || value.isEmpty()) {
            throw new IllegalArgumentException("Item name is required.");
        }
        return value;
    }
}
