package com.ghostftp.client.model;

import java.util.Base64;
import java.util.Locale;

public final class ConnectionConfig {
    public enum Protocol {
        FTP("FTP", 21),
        FTPS_EXPLICIT("FTPS (explicit)", 21),
        FTPS_IMPLICIT("FTPS (implicit)", 990),
        SFTP("SFTP", 22);

        private final String label;
        private final int defaultPort;

        Protocol(String label, int defaultPort) {
            this.label = label;
            this.defaultPort = defaultPort;
        }

        public int defaultPort() { return defaultPort; }
        @Override public String toString() { return label; }
    }

    private final Protocol protocol;
    private final String host;
    private final int port;
    private final String username;
    private final String password;
    private final String fingerprint;

    private ConnectionConfig(Protocol protocol, String host, int port, String username, String password, String fingerprint) {
        this.protocol = protocol;
        this.host = host;
        this.port = port;
        this.username = username;
        this.password = password;
        this.fingerprint = fingerprint;
    }

    public static ConnectionConfig create(Protocol protocol, String rawHost, String rawPort, String rawUsername, String rawPassword, String rawFingerprint) {
        if (protocol == null) throw new IllegalArgumentException("Protocol is required.");
        rejectControlCharacters(rawHost, "Host");
        rejectControlCharacters(rawPort, "Port");
        rejectControlCharacters(rawUsername, "Username");
        rejectControlCharacters(rawPassword, "Password");
        rejectControlCharacters(rawFingerprint, "SFTP fingerprint");

        String host = normalizeHost(rawHost);
        String username = rawUsername == null ? "" : rawUsername.trim();
        if (username.isEmpty()) throw new IllegalArgumentException("Username is required.");
        String password = rawPassword == null ? "" : rawPassword;
        int port = parsePort(rawPort, protocol.defaultPort());
        String fingerprint = normalizeFingerprint(rawFingerprint);
        if (protocol == Protocol.SFTP && fingerprint.isEmpty()) {
            throw new IllegalArgumentException("SFTP requires an expected SHA-256 host-key fingerprint.");
        }
        return new ConnectionConfig(protocol, host, port, username, password, fingerprint);
    }

    static String normalizeHost(String rawHost) {
        rejectControlCharacters(rawHost, "Host");
        String host = rawHost == null ? "" : rawHost.trim();
        if (host.isEmpty()) throw new IllegalArgumentException("Host is required.");
        String lower = host.toLowerCase(Locale.ROOT);
        if (lower.contains("://") || host.contains("/") || host.contains("\\") || host.indexOf('\0') >= 0) {
            throw new IllegalArgumentException("Enter a host name or IP address, not a URL or path.");
        }
        for (int i = 0; i < host.length(); i++) {
            if (Character.isWhitespace(host.charAt(i))) throw new IllegalArgumentException("Host cannot contain whitespace.");
        }
        if (host.startsWith("[") && host.endsWith("]") && host.length() > 2) {
            host = host.substring(1, host.length() - 1);
        }
        return host;
    }

    static String normalizeFingerprint(String raw) {
        rejectControlCharacters(raw, "SFTP fingerprint");
        String value = raw == null ? "" : raw.trim();
        if (value.isEmpty()) return "";
        if (value.regionMatches(true, 0, "SHA256:", 0, 7)) value = value.substring(7);
        if (value.isEmpty()) {
            throw new IllegalArgumentException("SFTP fingerprint must be an OpenSSH SHA256 fingerprint.");
        }

        int firstPadding = value.indexOf('=');
        if (firstPadding >= 0) {
            for (int i = firstPadding; i < value.length(); i++) {
                if (value.charAt(i) != '=') throw new IllegalArgumentException("SFTP fingerprint has invalid Base64 padding.");
            }
            value = value.substring(0, firstPadding);
        }
        if (value.isEmpty()) throw new IllegalArgumentException("SFTP fingerprint must be an OpenSSH SHA256 fingerprint.");

        String padded = value;
        int remainder = padded.length() % 4;
        if (remainder == 1) throw new IllegalArgumentException("SFTP fingerprint is not valid Base64.");
        if (remainder > 0) padded += "=".repeat(4 - remainder);

        byte[] digest;
        try {
            digest = Base64.getDecoder().decode(padded);
        } catch (IllegalArgumentException error) {
            throw new IllegalArgumentException("SFTP fingerprint is not valid Base64.");
        }
        if (digest.length != 32) throw new IllegalArgumentException("SFTP fingerprint must contain a 32-byte SHA-256 digest.");
        return "SHA256:" + Base64.getEncoder().withoutPadding().encodeToString(digest);
    }

    private static void rejectControlCharacters(String value, String field) {
        if (value == null) return;
        if (value.indexOf('\0') >= 0 || value.indexOf('\r') >= 0 || value.indexOf('\n') >= 0) {
            throw new IllegalArgumentException(field + " contains an unsafe control character.");
        }
    }

    private static int parsePort(String rawPort, int fallback) {
        rejectControlCharacters(rawPort, "Port");
        String value = rawPort == null ? "" : rawPort.trim();
        if (value.isEmpty()) return fallback;
        try {
            int port = Integer.parseInt(value);
            if (port < 1 || port > 65535) throw new NumberFormatException();
            return port;
        } catch (NumberFormatException ex) {
            throw new IllegalArgumentException("Port must be between 1 and 65535.");
        }
    }

    public Protocol protocol() { return protocol; }
    public String host() { return host; }
    public int port() { return port; }
    public String username() { return username; }
    public String password() { return password; }
    public String fingerprint() { return fingerprint; }
}
