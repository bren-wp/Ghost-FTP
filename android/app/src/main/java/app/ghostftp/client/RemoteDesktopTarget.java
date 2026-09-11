package app.ghostftp.client;

import android.net.Uri;

final class RemoteDesktopTarget {
    private static final int DEFAULT_PORT = 3389;

    private RemoteDesktopTarget() { }

    static Uri uri(String raw) {
        return Uri.parse("rdp://" + authority(raw));
    }

    static String authority(String raw) {
        String value = raw == null ? "" : raw.trim();
        if (value.isEmpty()) throw new IllegalArgumentException("RDP server is required.");
        if (value.length() > 512 || containsUnsafe(value)) {
            throw new IllegalArgumentException("RDP server contains unsupported characters.");
        }

        String host = value;
        int port = DEFAULT_PORT;
        if (value.startsWith("[")) {
            int end = value.indexOf(']');
            if (end <= 1) throw new IllegalArgumentException("Invalid bracketed RDP server.");
            host = value.substring(1, end);
            String rest = value.substring(end + 1);
            if (!rest.isEmpty()) {
                if (!rest.startsWith(":") || rest.length() == 1) throw new IllegalArgumentException("Invalid RDP port.");
                port = parsePort(rest.substring(1));
            }
        } else if (count(value, ':') == 1) {
            int split = value.indexOf(':');
            host = value.substring(0, split);
            port = parsePort(value.substring(split + 1));
        }

        host = host.trim();
        if (host.isEmpty() || host.length() > 255 || containsUnsafe(host)) {
            throw new IllegalArgumentException("Invalid RDP server.");
        }
        String normalizedHost = host.indexOf(':') >= 0 ? "[" + host + "]" : host;
        return normalizedHost + ":" + port;
    }

    private static int parsePort(String value) {
        try {
            int port = Integer.parseInt(value);
            if (port < 1 || port > 65535) throw new NumberFormatException();
            return port;
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("RDP port must be between 1 and 65535.");
        }
    }

    private static int count(String value, char needle) {
        int count = 0;
        for (int i = 0; i < value.length(); i++) if (value.charAt(i) == needle) count++;
        return count;
    }

    private static boolean containsUnsafe(String value) {
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            if (c == '\0' || c == '\r' || c == '\n' || c == '\t' || Character.isWhitespace(c)
                    || c == '/' || c == '\\' || c == '@' || c == '?' || c == '#') return true;
        }
        return false;
    }
}
