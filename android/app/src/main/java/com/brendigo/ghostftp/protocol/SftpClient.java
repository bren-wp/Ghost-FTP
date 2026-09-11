package com.brendigo.ghostftp.protocol;

import android.content.Context;
import com.jcraft.jsch.Channel;
import com.jcraft.jsch.ChannelSftp;
import com.jcraft.jsch.HostKey;
import com.jcraft.jsch.HostKeyRepository;
import com.jcraft.jsch.JSch;
import com.jcraft.jsch.JSchException;
import com.jcraft.jsch.Session;
import com.jcraft.jsch.SftpATTRS;
import com.jcraft.jsch.UserInfo;
import java.io.File;
import java.io.InputStream;
import java.io.OutputStream;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Base64;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;
import java.util.Vector;

public final class SftpClient implements RemoteClient {
    private static final int CONNECT_TIMEOUT_MS = 15_000;
    private static final int MAX_LIST_ENTRIES = 10_000;

    private final Context context;
    private Session session;
    private ChannelSftp channel;
    private ConnectionSpec spec;

    public SftpClient(Context context) {
        this.context = context.getApplicationContext();
    }

    @Override
    public void connect(ConnectionSpec connectionSpec, TrustPrompt trustPrompt) throws Exception {
        if (connectionSpec.protocol() != ConnectionSpec.Protocol.SFTP) {
            throw new IllegalArgumentException("SFTP client requires the SFTP protocol.");
        }
        if (trustPrompt == null) {
            throw new IllegalArgumentException("SFTP trust confirmation is required.");
        }
        close();
        spec = connectionSpec;

        File knownHostsFile = new File(context.getFilesDir(), "known_hosts");
        if (!knownHostsFile.exists() && !knownHostsFile.createNewFile()) {
            throw new IllegalStateException("Unable to create the app-private SFTP known-hosts file.");
        }

        JSch jsch = new JSch();
        jsch.setKnownHosts(knownHostsFile.getAbsolutePath());
        HostKeyRepository delegate = jsch.getHostKeyRepository();
        PinnedRepository repository = new PinnedRepository(delegate);
        jsch.setHostKeyRepository(repository);

        Session next = jsch.getSession(connectionSpec.username(), connectionSpec.host(), connectionSpec.port());
        next.setConfig("StrictHostKeyChecking", "ask");
        next.setConfig("PreferredAuthentications", "password");
        next.setServerAliveInterval(15_000);
        next.setServerAliveCountMax(2);
        char[] password = connectionSpec.passwordCopy();
        byte[] passwordBytes = utf8(password);
        Arrays.fill(password, '\0');
        try {
            next.setPassword(passwordBytes);
            next.setUserInfo(new HostTrustUserInfo(repository, trustPrompt));
            next.connect(CONNECT_TIMEOUT_MS);
        } finally {
            Arrays.fill(passwordBytes, (byte) 0);
        }

        Channel opened = next.openChannel("sftp");
        opened.connect(CONNECT_TIMEOUT_MS);
        session = next;
        channel = (ChannelSftp) opened;
    }

    @Override
    public List<RemoteEntry> list(String path) throws Exception {
        requireConnected();
        channel.cd(RemotePaths.navigationPath(path));
        Vector<?> raw = channel.ls(".");
        if (raw.size() > MAX_LIST_ENTRIES) {
            throw new IllegalStateException("Server listing exceeds the Android safety limit.");
        }
        List<RemoteEntry> entries = new ArrayList<>(raw.size());
        for (Object value : raw) {
            if (!(value instanceof ChannelSftp.LsEntry)) {
                continue;
            }
            ChannelSftp.LsEntry entry = (ChannelSftp.LsEntry) value;
            String name = entry.getFilename();
            if (name.equals(".") || name.equals("..")) {
                continue;
            }
            SftpATTRS attrs = entry.getAttrs();
            entries.add(new RemoteEntry(name, attrs.isDir() && !attrs.isLink(), attrs.isLink(), attrs.getSize()));
        }
        entries.sort(Comparator
                .comparing(RemoteEntry::isDirectory).reversed()
                .thenComparing(entry -> entry.name().toLowerCase(Locale.ROOT))
                .thenComparing(RemoteEntry::name));
        return entries;
    }

    @Override
    public String currentPath() throws Exception {
        requireConnected();
        return channel.pwd();
    }

    @Override
    public void upload(InputStream source, String remoteName) throws Exception {
        requireConnected();
        channel.put(source, RemotePaths.leafName(remoteName), ChannelSftp.OVERWRITE);
    }

    @Override
    public void download(RemoteEntry entry, OutputStream target) throws Exception {
        requireConnected();
        if (entry == null || entry.isDirectory() || entry.isSymlink()) {
            throw new IllegalArgumentException("Only an ordinary remote file can be downloaded.");
        }
        channel.get(RemotePaths.leafName(entry.name()), target);
        target.flush();
    }

    @Override
    public void mkdir(String name) throws Exception {
        requireConnected();
        channel.mkdir(RemotePaths.leafName(name));
    }

    @Override
    public void delete(RemoteEntry entry) throws Exception {
        requireConnected();
        if (entry == null || entry.isSymlink()) {
            throw new IllegalArgumentException("Symlink deletion is blocked.");
        }
        String name = RemotePaths.leafName(entry.name());
        if (entry.isDirectory()) {
            channel.rmdir(name);
        } else {
            channel.rm(name);
        }
    }

    @Override
    public boolean isConnected() {
        return session != null && session.isConnected() && channel != null && channel.isConnected();
    }

    private void requireConnected() {
        if (!isConnected()) {
            throw new IllegalStateException("SFTP connection is not active.");
        }
    }

    @Override
    public void close() {
        if (channel != null) {
            channel.disconnect();
            channel = null;
        }
        if (session != null) {
            session.disconnect();
            session = null;
        }
        if (spec != null) {
            spec.clearPassword();
            spec = null;
        }
    }

    private static byte[] utf8(char[] value) {
        try {
            java.nio.ByteBuffer encoded = java.nio.charset.StandardCharsets.UTF_8.newEncoder().encode(java.nio.CharBuffer.wrap(value));
            byte[] output = new byte[encoded.remaining()];
            encoded.get(output);
            return output;
        } catch (java.nio.charset.CharacterCodingException error) {
            throw new IllegalArgumentException("Password is not valid UTF-8.", error);
        }
    }

    private static final class PinnedRepository implements HostKeyRepository {
        private final HostKeyRepository delegate;
        private volatile int lastStatus = NOT_INCLUDED;
        private volatile String lastHost = "";
        private volatile byte[] lastKey = new byte[0];
        private volatile String lastType = "";

        PinnedRepository(HostKeyRepository delegate) {
            this.delegate = delegate;
        }

        @Override
        public int check(String host, byte[] key) {
            int status = delegate.check(host, key);
            lastStatus = status;
            lastHost = host == null ? "" : host;
            lastKey = key == null ? new byte[0] : Arrays.copyOf(key, key.length);
            lastType = detectType(host, key);
            return status;
        }

        String fingerprint() {
            try {
                byte[] digest = MessageDigest.getInstance("SHA-256").digest(lastKey);
                return "SHA256:" + Base64.getEncoder().withoutPadding().encodeToString(digest);
            } catch (Exception error) {
                return "SHA256:unavailable";
            }
        }

        String host() {
            return lastHost;
        }

        String type() {
            return lastType;
        }

        int status() {
            return lastStatus;
        }

        private static String detectType(String host, byte[] key) {
            try {
                return new HostKey(host == null ? "" : host, key).getType();
            } catch (JSchException error) {
                return "unknown";
            }
        }

        @Override
        public void add(HostKey hostkey, UserInfo ui) {
            delegate.add(hostkey, ui);
        }

        @Override
        public void remove(String host, String type) {
            delegate.remove(host, type);
        }

        @Override
        public void remove(String host, String type, byte[] key) {
            delegate.remove(host, type, key);
        }

        @Override
        public String getKnownHostsRepositoryID() {
            return delegate.getKnownHostsRepositoryID();
        }

        @Override
        public HostKey[] getHostKey() {
            return delegate.getHostKey();
        }

        @Override
        public HostKey[] getHostKey(String host, String type) {
            return delegate.getHostKey(host, type);
        }
    }

    private static final class HostTrustUserInfo implements UserInfo {
        private final PinnedRepository repository;
        private final TrustPrompt prompt;

        HostTrustUserInfo(PinnedRepository repository, TrustPrompt prompt) {
            this.repository = repository;
            this.prompt = prompt;
        }

        @Override
        public String getPassphrase() {
            return null;
        }

        @Override
        public String getPassword() {
            return null;
        }

        @Override
        public boolean promptPassword(String message) {
            return false;
        }

        @Override
        public boolean promptPassphrase(String message) {
            return false;
        }

        @Override
        public boolean promptYesNo(String message) {
            if (repository.status() == HostKeyRepository.CHANGED) {
                prompt.reportChangedHostKey(repository.host(), repository.fingerprint());
                return false;
            }
            if (repository.status() == HostKeyRepository.NOT_INCLUDED) {
                return prompt.confirmNewHostKey(repository.host(), repository.type(), repository.fingerprint());
            }
            return false;
        }

        @Override
        public void showMessage(String message) {
            // JSch informational messages are deliberately not persisted or logged.
        }
    }
}
