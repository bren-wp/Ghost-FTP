package com.GhostFTP.client.model;

public final class RemotePaths {
    private RemotePaths() {}

    public static String normalizeDirectory(String path) {
        if (path == null || path.isBlank() || path.equals("/")) return "/";
        if (!path.startsWith("/") || path.contains("\\") || path.indexOf('\0') >= 0 || path.contains("//")) {
            throw new IllegalArgumentException("Remote path is not canonical.");
        }
        String value = path;
        if (value.length() > 1 && value.endsWith("/")) value = value.substring(0, value.length() - 1);
        String[] parts = value.substring(1).split("/", -1);
        for (String part : parts) {
            if (part.isEmpty() || part.equals(".") || part.equals("..")) {
                throw new IllegalArgumentException("Remote path contains an unsafe component.");
            }
        }
        return value;
    }

    public static String child(String directory, String name) {
        validateName(name);
        String base = normalizeDirectory(directory);
        return base.equals("/") ? "/" + name : base + "/" + name;
    }

    public static String parent(String directory) {
        String path = normalizeDirectory(directory);
        if (path.equals("/")) return "/";
        int slash = path.lastIndexOf('/');
        return slash <= 0 ? "/" : path.substring(0, slash);
    }

    public static void validateName(String name) {
        if (name == null
            || name.isBlank()
            || !name.equals(name.trim())
            || name.equals(".")
            || name.equals("..")
            || name.contains("/")
            || name.contains("\\")
            || name.indexOf('\0') >= 0
            || name.indexOf('\r') >= 0
            || name.indexOf('\n') >= 0) {
            throw new IllegalArgumentException("Name must be one canonical remote path component.");
        }
    }
}
