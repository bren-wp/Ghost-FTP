package app.ghostftp.client;

final class RemoteEntry {
    final String name;
    final boolean directory;
    final boolean regularFile;
    final String type;
    final long size;
    final long modifiedEpochMillis;
    final String permissions;

    RemoteEntry(String name, boolean directory, long size) {
        this(name, directory, size, 0L, "", directory ? "dir" : "file");
    }

    RemoteEntry(String name, boolean directory, long size, long modifiedEpochMillis, String permissions) {
        this(name, directory, size, modifiedEpochMillis, permissions, directory ? "dir" : "file");
    }

    RemoteEntry(String name, boolean directory, long size, long modifiedEpochMillis, String permissions, String type) {
        this.name = name;
        this.directory = directory;
        this.type = type == null ? "" : type.trim().toLowerCase(java.util.Locale.ROOT);
        this.regularFile = "file".equals(this.type);
        this.size = Math.max(0L, size);
        this.modifiedEpochMillis = Math.max(0L, modifiedEpochMillis);
        this.permissions = permissions == null ? "" : permissions.trim();
    }

    @Override
    public String toString() {
        if (directory) {
            return "DIR   " + name;
        }
        return (regularFile ? "FILE  " : "OTHER ") + name + "  (" + size + " B)";
    }
}
