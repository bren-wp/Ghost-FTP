package com.byftp.client.model;

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

    public static ConnectionConfig create(Protocol protocol, String rawHost, String rawPort, String rawUsername, String password, String rawFingerprint) {
        if (protocol == null) throw new IllegalArgumentException("Protocol is required.");
        String host = normalizeHost(rawHost);
        String username = rawUsername == null ? "" : rawUsername.trim();
        if (username.isEmpty()) throw new IllegalArgumentException("Username is required.");
        int port = parsePort(rawPort, protocol.defaultPort());
        String fingerprint = normalizeFingerprint(rawFingerprint);
        if (protocol == Protocol.SFTP && fingerprint.isEmpty()) {
            throw new IllegalArgumentException("SFTP requires an expected SHA-256 host-key fingerprint.");
        }
        return new ConnectionConfig(protocol, host, port, username, password == null ? "" : password, fingerprint);
    }

    static String normalizeHost(String rawHost) {
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
        String fingerprint = raw == null ? "" : raw.trim();
        if (fingerprint.isEmpty()) return "";
        if (!fingerprint.regionMatches(true, 0, "SHA256:", 0, 7)) {
            fingerprint = "SHA256:" + fingerprint;
        }
        return "SHA256:" + fingerprint.substring(7).replace("=", "");
    }

    private static int parsePort(String rawPort, int fallback) {
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
