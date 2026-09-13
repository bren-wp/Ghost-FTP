package app.ghostftp.client;

final class RemoteEntry {
    final String name;
    final boolean directory;
    final long size;
    final long modifiedEpochMillis;
    final String permissions;

    RemoteEntry(String name, boolean directory, long size) {
        this(name, directory, size, 0L, "");
    }

    RemoteEntry(String name, boolean directory, long size, long modifiedEpochMillis, String permissions) {
        this.name = name;
        this.directory = directory;
        this.size = Math.max(0L, size);
        this.modifiedEpochMillis = Math.max(0L, modifiedEpochMillis);
        this.permissions = permissions == null ? "" : permissions.trim();
    }

    @Override
    public String toString() {
        return directory ? "DIR   " + name : "FILE  " + name + "  (" + size + " B)";
    }
}
