package com.GhostFTP.client.remote;

import com.GhostFTP.client.model.ConnectionConfig;
import com.GhostFTP.client.model.RemoteEntry;
import com.GhostFTP.client.model.RemotePaths;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.time.Duration;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import org.apache.commons.net.ftp.FTP;
import org.apache.commons.net.ftp.FTPClient;
import org.apache.commons.net.ftp.FTPFile;
import org.apache.commons.net.ftp.FTPReply;
import org.apache.commons.net.ftp.FTPSClient;

public final class FtpRemoteClient implements RemoteClient {
    private final ConnectionConfig.Protocol protocol;
    private final String host;
    private final int port;
    private final String username;
    private String password;
    private FTPClient client;
    private String loginRoot = "";

    public FtpRemoteClient(ConnectionConfig config) {
        protocol = config.protocol();
        host = config.host();
        port = config.port();
        username = config.username();
        password = config.password();
    }

    @Override public void connect() throws Exception {
        FTPClient next;
        if (protocol == ConnectionConfig.Protocol.FTPS_EXPLICIT || protocol == ConnectionConfig.Protocol.FTPS_IMPLICIT) {
            FTPSClient ftps = new FTPSClient(protocol == ConnectionConfig.Protocol.FTPS_IMPLICIT);
            // null explicitly selects the Android/JVM platform trust manager; endpoint checking verifies the hostname.
            ftps.setTrustManager(null);
            ftps.setEndpointCheckingEnabled(true);
            next = ftps;
        } else {
            next = new FTPClient();
        }
        next.setConnectTimeout(15_000);
        next.setDefaultTimeout(15_000);
        next.setDataTimeout(Duration.ofSeconds(30));
        next.setControlEncoding("UTF-8");
        String loginPassword = password;
        try {
            next.connect(host, port);
            if (!FTPReply.isPositiveCompletion(next.getReplyCode())) throw new IOException("FTP server rejected the connection.");
            next.setSoTimeout(30_000);
            if (!next.login(username, loginPassword)) throw new IOException("FTP login failed.");
            if (next instanceof FTPSClient ftps) {
                ftps.execPBSZ(0);
                ftps.execPROT("P");
            }
            next.enterLocalPassiveMode();
            if (!next.setFileType(FTP.BINARY_FILE_TYPE)) throw new IOException("Server refused binary transfer mode.");
            next.sendCommand("OPTS", "UTF8 ON");

            // Treat the UI root as the account's login directory. Shared-hosting FTP
            // servers often expose a virtual/chroot namespace and may reject or escape
            // the account namespace when clients force an unrelated absolute '/'.
            loginRoot = normalizeLoginRoot(next.printWorkingDirectory());
            client = next;
        } catch (Exception error) {
            try { if (next.isConnected()) next.disconnect(); } catch (IOException ignored) {}
            throw error;
        } finally {
            password = "";
            loginPassword = "";
        }
    }

    @Override public List<RemoteEntry> list(String directory) throws Exception {
        FTPFile[] files = requireClient().listFiles(mapLoginRelativePath(loginRoot, directory));
        List<RemoteEntry> result = new ArrayList<>();
        for (FTPFile file : files) {
            String name = file.getName();
            try {
                RemotePaths.validateName(name);
            } catch (IllegalArgumentException unsafeName) {
                continue;
            }
            long modified = file.getTimestamp() == null ? 0L : file.getTimestamp().getTimeInMillis();
            result.add(new RemoteEntry(name, file.isDirectory(), Math.max(0L, file.getSize()), modified));
        }
        result.sort(Comparator.comparing(RemoteEntry::directory).reversed().thenComparing(RemoteEntry::name, String.CASE_INSENSITIVE_ORDER));
        return result;
    }

    @Override public void upload(String remotePath, InputStream source) throws Exception {
        if (!requireClient().storeFile(mapLoginRelativePath(loginRoot, remotePath), source)) throw new IOException("Upload failed: " + safeReply());
    }

    @Override public void download(String remotePath, OutputStream destination) throws Exception {
        if (!requireClient().retrieveFile(mapLoginRelativePath(loginRoot, remotePath), destination)) throw new IOException("Download failed: " + safeReply());
    }

    @Override public void mkdir(String remotePath) throws Exception {
        if (!requireClient().makeDirectory(mapLoginRelativePath(loginRoot, remotePath))) throw new IOException("Create directory failed: " + safeReply());
    }

    @Override public void delete(String remotePath, boolean directory) throws Exception {
        String path = mapLoginRelativePath(loginRoot, remotePath);
        boolean ok = directory ? requireClient().removeDirectory(path) : requireClient().deleteFile(path);
        if (!ok) throw new IOException("Delete failed: " + safeReply());
    }

    @Override public void rename(String from, String to) throws Exception {
        if (!requireClient().rename(mapLoginRelativePath(loginRoot, from), mapLoginRelativePath(loginRoot, to))) {
            throw new IOException("Rename failed: " + safeReply());
        }
    }

    static String mapLoginRelativePath(String rawLoginRoot, String uiPath) {
        String root = normalizeLoginRoot(rawLoginRoot);
        if (uiPath == null || uiPath.isBlank()) throw new IllegalArgumentException("Remote path is required.");
        if (!uiPath.startsWith("/")) throw new IllegalArgumentException("Remote UI path must start with '/'.");
        if (uiPath.indexOf('\0') >= 0 || uiPath.indexOf('\\') >= 0 || uiPath.contains("//")) {
            throw new IllegalArgumentException("Remote UI path is not canonical.");
        }
        if (uiPath.equals("/")) return root.isEmpty() ? "." : root;

        String[] parts = uiPath.substring(1).split("/", -1);
        StringBuilder relative = new StringBuilder();
        for (String part : parts) {
            if (part.isEmpty() || part.equals(".") || part.equals("..")) {
                throw new IllegalArgumentException("Remote UI path contains an unsafe component.");
            }
            if (relative.length() > 0) relative.append('/');
            relative.append(part);
        }

        if (root.isEmpty() || root.equals(".")) return relative.toString();
        if (root.equals("/")) return "/" + relative;
        return root + "/" + relative;
    }

    private static String normalizeLoginRoot(String raw) {
        if (raw == null) return "";
        if (raw.indexOf('\0') >= 0 || raw.indexOf('\r') >= 0 || raw.indexOf('\n') >= 0) {
            throw new IllegalArgumentException("FTP server returned a noncanonical login directory.");
        }
        String root = raw.trim().replace('\\', '/');
        if (root.isEmpty() || root.equals(".")) return "";
        if (root.contains("//")) {
            throw new IllegalArgumentException("FTP server returned a noncanonical login directory.");
        }
        while (root.length() > 1 && root.endsWith("/")) root = root.substring(0, root.length() - 1);
        if (root.equals("/")) return root;

        String check = root.startsWith("/") ? root.substring(1) : root;
        for (String part : check.split("/", -1)) {
            if (part.isEmpty() || part.equals(".") || part.equals("..")) {
                throw new IllegalArgumentException("FTP server returned an unsafe login directory.");
            }
        }
        return root;
    }

    private FTPClient requireClient() throws IOException {
        if (client == null || !client.isConnected()) throw new IOException("Not connected.");
        return client;
    }

    private String safeReply() {
        String reply = client == null ? "" : client.getReplyString();
        if (reply == null) return "server rejected the operation";
        reply = reply.replace('\r', ' ').replace('\n', ' ').trim();
        return reply.length() > 240 ? reply.substring(0, 240) : reply;
    }

    @Override public void close() {
        password = "";
        FTPClient current = client;
        client = null;
        loginRoot = "";
        if (current == null) return;
        try { if (current.isConnected()) current.logout(); } catch (Exception ignored) {}
        try { if (current.isConnected()) current.disconnect(); } catch (Exception ignored) {}
    }
}
