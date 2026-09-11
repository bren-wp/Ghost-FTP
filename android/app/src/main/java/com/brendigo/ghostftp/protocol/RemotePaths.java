package com.brendigo.ghostftp.protocol;

final class RemotePaths {
    private RemotePaths() {}

    static String navigationPath(String value) {
        String path = value == null ? "" : value.trim();
        if (path.isEmpty()) {
            path = ".";
        }
        validateSingleLine(path, 4096);
        return path;
    }

    static String leafName(String value) {
        String name = value == null ? "" : value.trim();
        validateSingleLine(name, 1024);
        if (name.isEmpty() || name.equals(".") || name.equals("..") || name.indexOf('/') >= 0) {
            throw new IllegalArgumentException("Remote item name is invalid.");
        }
        return name;
    }

    private static void validateSingleLine(String value, int maxLength) {
        if (value.length() > maxLength || value.indexOf('\0') >= 0 || value.indexOf('\r') >= 0 || value.indexOf('\n') >= 0) {
            throw new IllegalArgumentException("Remote path contains unsupported data.");
        }
    }
}
