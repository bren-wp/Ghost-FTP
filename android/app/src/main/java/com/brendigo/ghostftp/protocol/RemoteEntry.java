package com.brendigo.ghostftp.protocol;

public final class RemoteEntry {
    private final String name;
    private final boolean directory;
    private final boolean symlink;
    private final long size;

    public RemoteEntry(String name, boolean directory, boolean symlink, long size) {
        this.name = name;
        this.directory = directory;
        this.symlink = symlink;
        this.size = Math.max(0L, size);
    }

    public String name() {
        return name;
    }

    public boolean isDirectory() {
        return directory;
    }

    public boolean isSymlink() {
        return symlink;
    }

    public long size() {
        return size;
    }
}
