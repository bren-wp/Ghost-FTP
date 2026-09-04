package com.GhostFTP.client.remote;

import com.GhostFTP.client.model.ConnectionConfig;
import com.GhostFTP.client.model.RemoteEntry;
import com.GhostFTP.client.model.RemotePaths;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.EnumSet;
import java.util.List;
import net.schmizz.sshj.SSHClient;
import net.schmizz.sshj.sftp.OpenMode;
import net.schmizz.sshj.sftp.RemoteFile;
import net.schmizz.sshj.sftp.RemoteResourceInfo;
import net.schmizz.sshj.sftp.SFTPClient;

public final class SftpRemoteClient implements RemoteClient {
    private final String host;
    private final int port;
    private final String username;
    private final String fingerprint;
    private String password;
    private SSHClient ssh;
    private SFTPClient sftp;

    public SftpRemoteClient(ConnectionConfig config) {
        host = config.host();
        port = config.port();
        username = config.username();
        password = config.password();
        fingerprint = config.fingerprint();
    }

    @Override public void connect() throws Exception {
        SSHClient next = new SSHClient();
        next.setConnectTimeout(15_000);
        next.setTimeout(30_000);
        next.addHostKeyVerifier(fingerprint);
        String loginPassword = password;
        try {
            next.connect(host, port);
            next.authPassword(username, loginPassword);
            SFTPClient nextSftp = next.newSFTPClient();
            ssh = next;
            sftp = nextSftp;
        } catch (Exception error) {
            try { next.disconnect(); } catch (Exception ignored) {}
            try { next.close(); } catch (Exception ignored) {}
            throw error;
        } finally {
            password = "";
            loginPassword = "";
        }
    }

    @Override public List<RemoteEntry> list(String directory) throws Exception {
        List<RemoteEntry> result = new ArrayList<>();
        for (RemoteResourceInfo info : requireSftp().ls(directory)) {
            String name = info.getName();
            try {
                RemotePaths.validateName(name);
            } catch (IllegalArgumentException unsafeName) {
                continue;
            }
            long modified = Math.max(0L, info.getAttributes().getMtime()) * 1000L;
            result.add(new RemoteEntry(name, info.isDirectory(), Math.max(0L, info.getAttributes().getSize()), modified));
        }
        result.sort(Comparator.comparing(RemoteEntry::directory).reversed().thenComparing(RemoteEntry::name, String.CASE_INSENSITIVE_ORDER));
        return result;
    }

    @Override public void upload(String remotePath, InputStream source) throws Exception {
        try (RemoteFile file = requireSftp().open(remotePath, EnumSet.of(OpenMode.WRITE, OpenMode.CREAT, OpenMode.TRUNC))) {
            byte[] buffer = new byte[64 * 1024];
            long offset = 0;
            for (int read; (read = source.read(buffer)) >= 0;) {
                if (read == 0) continue;
                file.write(offset, buffer, 0, read);
                offset += read;
            }
        }
    }

    @Override public void download(String remotePath, OutputStream destination) throws Exception {
        try (RemoteFile file = requireSftp().open(remotePath, EnumSet.of(OpenMode.READ))) {
            byte[] buffer = new byte[64 * 1024];
            long offset = 0;
            while (true) {
                int read = file.read(offset, buffer, 0, buffer.length);
                if (read <= 0) break;
                destination.write(buffer, 0, read);
                offset += read;
            }
            destination.flush();
        }
    }

    @Override public void mkdir(String remotePath) throws Exception { requireSftp().mkdir(remotePath); }

    @Override public void delete(String remotePath, boolean directory) throws Exception {
        if (directory) requireSftp().rmdir(remotePath); else requireSftp().rm(remotePath);
    }

    @Override public void rename(String from, String to) throws Exception { requireSftp().rename(from, to); }

    private SFTPClient requireSftp() throws IOException {
        if (sftp == null) throw new IOException("Not connected.");
        return sftp;
    }

    @Override public void close() {
        password = "";
        SFTPClient currentSftp = sftp;
        SSHClient currentSsh = ssh;
        sftp = null;
        ssh = null;
        try { if (currentSftp != null) currentSftp.close(); } catch (Exception ignored) {}
        try { if (currentSsh != null) currentSsh.disconnect(); } catch (Exception ignored) {}
        try { if (currentSsh != null) currentSsh.close(); } catch (Exception ignored) {}
    }
}
