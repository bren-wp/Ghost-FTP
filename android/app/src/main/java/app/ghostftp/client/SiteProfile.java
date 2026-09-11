package app.ghostftp.client;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Locale;
import java.util.Objects;

final class SiteProfile {
    final String id;
    final String name;
    final String protocol;
    final String host;
    final int port;
    final String username;
    final String localStartTreeUri;
    final String remoteStartPath;
    final List<String> localBookmarks;
    final List<String> remoteBookmarks;

    SiteProfile(
            String id,
            String name,
            String protocol,
            String host,
            int port,
            String username,
            String localStartTreeUri,
            String remoteStartPath,
            List<String> localBookmarks,
            List<String> remoteBookmarks) {
        this.id = requireText(id, "Profile id");
        this.name = requireText(name, "Profile name");
        this.protocol = requireProtocol(protocol);
        this.host = requireText(host, "Host");
        this.port = requirePort(port);
        this.username = username == null ? "" : username.trim();
        this.localStartTreeUri = localStartTreeUri == null ? "" : localStartTreeUri.trim();
        this.remoteStartPath = normalizeRemotePath(remoteStartPath);
        this.localBookmarks = immutableUnique(localBookmarks);
        this.remoteBookmarks = immutableRemoteBookmarks(remoteBookmarks);
    }

    String identityKey() {
        return protocol + "\n" + host.toLowerCase(Locale.ROOT) + "\n" + port + "\n" + username;
    }

    boolean sameServerIdentity(SiteProfile other) {
        return other != null && identityKey().equals(other.identityKey());
    }

    SiteProfile withRemoteStateResetForIdentityChange(SiteProfile previous) {
        if (previous == null || sameServerIdentity(previous)) {
            return this;
        }
        return new SiteProfile(
                id,
                name,
                protocol,
                host,
                port,
                username,
                localStartTreeUri,
                "/",
                localBookmarks,
                Collections.emptyList());
    }

    SiteProfile withRemoteStartPath(String path) {
        return new SiteProfile(id, name, protocol, host, port, username, localStartTreeUri,
                path, localBookmarks, remoteBookmarks);
    }

    SiteProfile withLocalStartTreeUri(String uri) {
        return new SiteProfile(id, name, protocol, host, port, username, uri,
                remoteStartPath, localBookmarks, remoteBookmarks);
    }

    SiteProfile withLocalBookmark(String uri) {
        List<String> next = new ArrayList<>(localBookmarks);
        if (uri != null && !uri.trim().isEmpty() && !next.contains(uri.trim())) {
            next.add(uri.trim());
        }
        return new SiteProfile(id, name, protocol, host, port, username, localStartTreeUri,
                remoteStartPath, next, remoteBookmarks);
    }

    SiteProfile withRemoteBookmark(String path) {
        String normalized = normalizeRemotePath(path);
        List<String> next = new ArrayList<>(remoteBookmarks);
        if (!next.contains(normalized)) {
            next.add(normalized);
        }
        return new SiteProfile(id, name, protocol, host, port, username, localStartTreeUri,
                remoteStartPath, localBookmarks, next);
    }

    private static String requireText(String value, String label) {
        String text = value == null ? "" : value.trim();
        if (text.isEmpty() || text.indexOf('\0') >= 0 || text.indexOf('\r') >= 0 || text.indexOf('\n') >= 0) {
            throw new IllegalArgumentException(label + " is required.");
        }
        return text;
    }

    private static String requireProtocol(String value) {
        String protocol = value == null ? "" : value.trim().toUpperCase(Locale.ROOT);
        if (!"FTP".equals(protocol) && !"FTPS".equals(protocol)) {
            throw new IllegalArgumentException("Unsupported protocol.");
        }
        return protocol;
    }

    private static int requirePort(int value) {
        if (value < 1 || value > 65535) {
            throw new IllegalArgumentException("Port must be between 1 and 65535.");
        }
        return value;
    }

    private static String normalizeRemotePath(String value) {
        String path = value == null ? "/" : value.trim();
        if (path.isEmpty()) path = "/";
        if (path.indexOf('\0') >= 0 || path.indexOf('\r') >= 0 || path.indexOf('\n') >= 0) {
            throw new IllegalArgumentException("Remote path contains prohibited control characters.");
        }
        if (!path.startsWith("/")) path = "/" + path;
        while (path.contains("//")) path = path.replace("//", "/");
        return path;
    }

    private static List<String> immutableUnique(List<String> values) {
        List<String> result = new ArrayList<>();
        if (values != null) {
            for (String value : values) {
                String text = value == null ? "" : value.trim();
                if (!text.isEmpty() && !result.contains(text)) result.add(text);
            }
        }
        return Collections.unmodifiableList(result);
    }

    private static List<String> immutableRemoteBookmarks(List<String> values) {
        List<String> result = new ArrayList<>();
        if (values != null) {
            for (String value : values) {
                String path = normalizeRemotePath(value);
                if (!result.contains(path)) result.add(path);
            }
        }
        return Collections.unmodifiableList(result);
    }

    @Override
    public boolean equals(Object obj) {
        return obj instanceof SiteProfile && id.equals(((SiteProfile) obj).id);
    }

    @Override
    public int hashCode() {
        return Objects.hash(id);
    }

    @Override
    public String toString() {
        return name + " · " + protocol + " · " + host;
    }
}
