package com.byftp.client.model;

import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/** Non-secret facts derived from the already loaded remote root listing. */
public record SharedHostingDiagnostics(
    boolean secure,
    String rootMode,
    String webRoot,
    boolean webRootDetected,
    int rootEntryCount
) {
    private static final List<String> WEB_ROOT_PRIORITY = List.of(
        "public_html", "httpdocs", "htdocs", "www", "web", "html"
    );

    public SharedHostingDiagnostics {
        rootMode = rootMode == null ? "account" : rootMode;
        webRoot = webRoot == null ? "" : webRoot;
        if (!webRootDetected) webRoot = "";
        rootEntryCount = Math.max(0, rootEntryCount);
    }

    public static SharedHostingDiagnostics analyze(ConnectionConfig.Protocol protocol, List<RemoteEntry> entries) {
        if (protocol == null) throw new IllegalArgumentException("Protocol is required.");
        List<RemoteEntry> safeEntries = entries == null ? List.of() : entries;
        Map<String, String> directories = new HashMap<>();
        for (RemoteEntry entry : safeEntries) {
            if (entry == null || !entry.directory()) continue;
            String name = entry.name();
            if (name == null || name.isBlank()) continue;
            directories.put(name.toLowerCase(Locale.ROOT), name);
        }

        String webRoot = "";
        for (String candidate : WEB_ROOT_PRIORITY) {
            String actual = directories.get(candidate);
            if (actual != null) {
                webRoot = actual;
                break;
            }
        }
        boolean detected = !webRoot.isEmpty();
        return new SharedHostingDiagnostics(
            protocol != ConnectionConfig.Protocol.FTP,
            protocol == ConnectionConfig.Protocol.SFTP ? "home" : "account",
            webRoot,
            detected,
            safeEntries.size()
        );
    }
}
