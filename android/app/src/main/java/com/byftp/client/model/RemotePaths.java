package com.byftp.client.model;

public final class RemotePaths {
    private RemotePaths() {}

    public static String normalizeDirectory(String path) {
        if (path == null || path.isBlank() || path.equals("/")) return "/";
        String p = path.replace('\\', '/');
        if (!p.startsWith("/")) p = "/" + p;
        while (p.contains("//")) p = p.replace("//", "/");
        if (p.length() > 1 && p.endsWith("/")) p = p.substring(0, p.length() - 1);
        return p;
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
        if (name == null || name.isBlank() || name.equals(".") || name.equals("..") || name.contains("/") || name.contains("\\") || name.indexOf('\0') >= 0) {
            throw new IllegalArgumentException("Name must be a single remote path component.");
        }
    }
}
