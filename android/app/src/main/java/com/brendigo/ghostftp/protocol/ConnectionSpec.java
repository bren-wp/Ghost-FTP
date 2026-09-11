package com.brendigo.ghostftp.protocol;

import java.net.IDN;
import java.util.Arrays;
import java.util.Locale;

public final class ConnectionSpec {
    public enum Protocol {
        FTP,
        FTPS,
        SFTP;

        public static Protocol fromLabel(String value) {
            return Protocol.valueOf(value.trim().toUpperCase(Locale.ROOT));
        }
    }

    private final Protocol protocol;
    private final String host;
    private final int port;
    private final String username;
    private final char[] password;

    public ConnectionSpec(Protocol protocol, String host, int port, String username, char[] password) {
        if (protocol == null) {
            throw new IllegalArgumentException("Protocol is required.");
        }
        this.protocol = protocol;
        this.host = normalizeHost(host);
        if (port < 1 || port > 65535) {
            throw new IllegalArgumentException("Port must be between 1 and 65535.");
        }
        this.port = port;
        this.username = requireSingleLine(username, "Username", 512);
        if (this.username.isEmpty()) {
            throw new IllegalArgumentException("Username is required.");
        }
        char[] source = password == null ? new char[0] : password;
        for (char value : source) {
            if (value == '\0' || value == '\r' || value == '\n') {
                throw new IllegalArgumentException("Password contains an unsupported control character.");
            }
        }
        this.password = Arrays.copyOf(source, source.length);
    }

    private static String normalizeHost(String value) {
        String host = requireSingleLine(value, "Host", 253);
        if (host.startsWith("[") && host.endsWith("]") && host.length() > 2) {
            host = host.substring(1, host.length() - 1);
        }
        if (host.isEmpty() || host.indexOf('/') >= 0 || host.indexOf('\\') >= 0 || host.indexOf(' ') >= 0 || host.indexOf('\t') >= 0) {
            throw new IllegalArgumentException("Host is invalid.");
        }
        if (host.indexOf(':') >= 0) {
            return host;
        }
        try {
            return IDN.toASCII(host, IDN.USE_STD3_ASCII_RULES).toLowerCase(Locale.ROOT);
        } catch (IllegalArgumentException error) {
            throw new IllegalArgumentException("Host is invalid.", error);
        }
    }

    private static String requireSingleLine(String value, String field, int maxLength) {
        String normalized = value == null ? "" : value.trim();
        if (normalized.length() > maxLength || normalized.indexOf('\0') >= 0 || normalized.indexOf('\r') >= 0 || normalized.indexOf('\n') >= 0) {
            throw new IllegalArgumentException(field + " is invalid.");
        }
        return normalized;
    }

    public Protocol protocol() {
        return protocol;
    }

    public String host() {
        return host;
    }

    public int port() {
        return port;
    }

    public String username() {
        return username;
    }

    public char[] passwordCopy() {
        return Arrays.copyOf(password, password.length);
    }

    public void clearPassword() {
        Arrays.fill(password, '\0');
    }
}
