package app.ghostftp.client;

final class RemoteEntry {
    final String name;
    final boolean directory;
    final long size;

    RemoteEntry(String name, boolean directory, long size) {
        this.name = name;
        this.directory = directory;
        this.size = size;
    }

    @Override
    public String toString() {
        return directory ? "DIR   " + name : "FILE  " + name + "  (" + size + " B)";
    }
}
