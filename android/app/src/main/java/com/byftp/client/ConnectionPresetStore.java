package com.byftp.client;

import android.content.Context;
import android.content.SharedPreferences;
import com.byftp.client.model.ConnectionConfig;

/**
 * Persists only non-secret connection metadata in the app-private preferences
 * file. Passwords and other authentication secrets are deliberately absent.
 */
final class ConnectionPresetStore {
    private static final String STORE = "byftp_connection_preset";
    private static final String KEY_PROTOCOL = "protocol";
    private static final String KEY_HOST = "host";
    private static final String KEY_PORT = "port";
    private static final String KEY_USERNAME = "username";
    private static final String KEY_FINGERPRINT = "fingerprint";

    private final SharedPreferences preferences;

    ConnectionPresetStore(Context context) {
        preferences = context.getSharedPreferences(STORE, Context.MODE_PRIVATE);
    }

    void save(ConnectionConfig config) {
        preferences.edit()
            .putString(KEY_PROTOCOL, config.protocol().name())
            .putString(KEY_HOST, config.host())
            .putInt(KEY_PORT, config.port())
            .putString(KEY_USERNAME, config.username())
            .putString(KEY_FINGERPRINT, config.fingerprint())
            .apply();
    }

    Preset load() {
        String protocolName = preferences.getString(KEY_PROTOCOL, "");
        String host = preferences.getString(KEY_HOST, "");
        String username = preferences.getString(KEY_USERNAME, "");
        String fingerprint = preferences.getString(KEY_FINGERPRINT, "");
        int port = preferences.getInt(KEY_PORT, 0);
        if (protocolName == null || host == null || username == null || fingerprint == null || port < 1 || port > 65535) {
            return null;
        }
        try {
            ConnectionConfig.Protocol protocol = ConnectionConfig.Protocol.valueOf(protocolName);
            ConnectionConfig validated = ConnectionConfig.create(
                protocol,
                host,
                Integer.toString(port),
                username,
                "",
                fingerprint
            );
            return new Preset(
                validated.protocol(),
                validated.host(),
                validated.port(),
                validated.username(),
                validated.fingerprint()
            );
        } catch (IllegalArgumentException invalid) {
            clear();
            return null;
        }
    }

    void clear() {
        preferences.edit().clear().apply();
    }

    static final class Preset {
        private final ConnectionConfig.Protocol protocol;
        private final String host;
        private final int port;
        private final String username;
        private final String fingerprint;

        Preset(ConnectionConfig.Protocol protocol, String host, int port, String username, String fingerprint) {
            this.protocol = protocol;
            this.host = host;
            this.port = port;
            this.username = username;
            this.fingerprint = fingerprint;
        }

        ConnectionConfig.Protocol protocol() { return protocol; }
        String host() { return host; }
        int port() { return port; }
        String username() { return username; }
        String fingerprint() { return fingerprint; }
    }
}
