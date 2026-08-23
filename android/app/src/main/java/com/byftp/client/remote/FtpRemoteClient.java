package com.byftp.client.remote;

import com.byftp.client.model.ConnectionConfig;
import com.byftp.client.model.RemoteEntry;
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
    private final ConnectionConfig config;
    private FTPClient client;

    public FtpRemoteClient(ConnectionConfig config) { this.config = config; }

    @Override public void connect() throws Exception {
        FTPClient next;
        if (config.protocol() == ConnectionConfig.Protocol.FTPS_EXPLICIT || config.protocol() == ConnectionConfig.Protocol.FTPS_IMPLICIT) {
            FTPSClient ftps = new FTPSClient(config.protocol() == ConnectionConfig.Protocol.FTPS_IMPLICIT);
            ftps.setEndpointCheckingEnabled(true);
            next = ftps;
        } else {
            next = new FTPClient();
        }
        next.setConnectTimeout(15_000);
        next.setDefaultTimeout(15_000);
        next.setDataTimeout(Duration.ofSeconds(30));
        next.setControlEncoding("UTF-8");
        try {
            next.connect(config.host(), config.port());
            if (!FTPReply.isPositiveCompletion(next.getReplyCode())) throw new IOException("FTP server rejected the connection.");
            next.setSoTimeout(30_000);
            if (!next.login(config.username(), config.password())) throw new IOException("FTP login failed.");
            if (next instanceof FTPSClient ftps) {
                ftps.execPBSZ(0);
                ftps.execPROT("P");
            }
            next.enterLocalPassiveMode();
            if (!next.setFileType(FTP.BINARY_FILE_TYPE)) throw new IOException("Server refused binary transfer mode.");
            next.sendCommand("OPTS", "UTF8 ON");
            client = next;
        } catch (Exception error) {
            try { if (next.isConnected()) next.disconnect(); } catch (IOException ignored) {}
            throw error;
        }
    }

    @Override public List<RemoteEntry> list(String directory) throws Exception {
        FTPFile[] files = requireClient().listFiles(directory);
        List<RemoteEntry> result = new ArrayList<>();
        for (FTPFile file : files) {
            String name = file.getName();
            if (name == null || name.equals(".") || name.equals("..") || name.contains("/") || name.contains("\\")) continue;
            long modified = file.getTimestamp() == null ? 0L : file.getTimestamp().getTimeInMillis();
            result.add(new RemoteEntry(name, file.isDirectory(), Math.max(0L, file.getSize()), modified));
        }
        result.sort(Comparator.comparing(RemoteEntry::directory).reversed().thenComparing(RemoteEntry::name, String.CASE_INSENSITIVE_ORDER));
        return result;
    }

    @Override public void upload(String remotePath, InputStream source) throws Exception {
        if (!requireClient().storeFile(remotePath, source)) throw new IOException("Upload failed: " + safeReply());
    }

    @Override public void download(String remotePath, OutputStream destination) throws Exception {
        if (!requireClient().retrieveFile(remotePath, destination)) throw new IOException("Download failed: " + safeReply());
    }

    @Override public void mkdir(String remotePath) throws Exception {
        if (!requireClient().makeDirectory(remotePath)) throw new IOException("Create directory failed: " + safeReply());
    }

    @Override public void delete(String remotePath, boolean directory) throws Exception {
        boolean ok = directory ? requireClient().removeDirectory(remotePath) : requireClient().deleteFile(remotePath);
        if (!ok) throw new IOException("Delete failed: " + safeReply());
    }

    @Override public void rename(String from, String to) throws Exception {
        if (!requireClient().rename(from, to)) throw new IOException("Rename failed: " + safeReply());
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
        FTPClient current = client;
        client = null;
        if (current == null) return;
        try { if (current.isConnected()) current.logout(); } catch (Exception ignored) {}
        try { if (current.isConnected()) current.disconnect(); } catch (Exception ignored) {}
    }
}
