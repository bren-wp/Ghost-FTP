package com.ghostftp.client.model;

public record RemoteEntry(String name, boolean directory, long size, long modifiedMillis) {
    public RemoteEntry {
        if (name == null || name.isEmpty() || name.equals(".") || name.equals("..") || name.contains("/") || name.contains("\\")) {
            throw new IllegalArgumentException("Invalid remote entry name.");
        }
    }

    public String displayLabel() {
        if (directory) return "📁  " + name;
        return "📄  " + name + "  (" + humanSize(size) + ")";
    }

    private static String humanSize(long bytes) {
        if (bytes < 1024) return bytes + " B";
        double value = bytes;
        String[] units = {"KB", "MB", "GB", "TB"};
        for (String unit : units) {
            value /= 1024.0;
            if (value < 1024.0) return String.format(java.util.Locale.ROOT, "%.1f %s", value, unit);
        }
        return bytes + " B";
    }
}
