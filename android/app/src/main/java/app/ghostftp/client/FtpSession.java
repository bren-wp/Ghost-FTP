package app.ghostftp.client;

import java.io.BufferedInputStream;
import java.io.BufferedOutputStream;
import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.Closeable;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.UUID;

import javax.net.ssl.SSLParameters;
import javax.net.ssl.SSLSocket;
import javax.net.ssl.SSLSocketFactory;

final class FtpSession implements Closeable {
    private static final int CONNECT_TIMEOUT_MS = 15000;
    private static final int READ_TIMEOUT_MS = 30000;

    private final String host;
    private final int port;
    private final boolean secure;
    private volatile Socket controlSocket;
    private volatile Socket activeDataSocket;
    private BufferedReader reader;
    private BufferedWriter writer;
    private volatile boolean connected;

    FtpSession(String host, int port, boolean secure) {
        this.host = requireHost(host);
        this.port = requirePort(port);
        this.secure = secure;
    }

    synchronized void connect(String username, String password) throws IOException {
        if (connected) {
            throw new IOException("Session is already connected.");
        }
        Socket plain = new Socket();
        plain.connect(new InetSocketAddress(host, port), CONNECT_TIMEOUT_MS);
        plain.setSoTimeout(READ_TIMEOUT_MS);
        controlSocket = plain;
        bindStreams(plain);
        expect(readReply(), 220);

        if (secure) {
            expect(command("AUTH TLS"), 234, 334);
            SSLSocket tls = wrapTls(plain);
            controlSocket = tls;
            bindStreams(tls);
        }

        String user = username == null || username.trim().isEmpty() ? "anonymous" : username.trim();
        Reply userReply = command("USER " + sanitizeArgument(user));
        if (userReply.code == 331) {
            expect(command("PASS " + sanitizeArgument(password == null ? "" : password)), 230, 202);
        } else {
            expect(userReply, 230);
        }
        expect(command("TYPE I"), 200);
        if (secure) {
            expect(command("PBSZ 0"), 200);
            expect(command("PROT P"), 200);
        }
        connected = true;
    }

    synchronized List<RemoteEntry> list(String remotePath) throws IOException {
        ensureConnected();
        String path = normalizeRemotePath(remotePath);
        Socket data = openPassiveDataSocket();
        Reply start;
        try {
            start = command("MLSD " + sanitizeArgument(path));
        } catch (IOException e) {
            releaseDataSocket(data);
            hardClose();
            throw new IOException("Directory listing could not start; the connection was closed.", e);
        }
        if (start.code != 125 && start.code != 150) {
            releaseDataSocket(data);
            throw new IOException("MLSD failed: " + start.message);
        }

        List<RemoteEntry> entries = new ArrayList<>();
        try {
            try (BufferedReader dataReader = new BufferedReader(new InputStreamReader(data.getInputStream(), StandardCharsets.UTF_8))) {
                String line;
                while ((line = dataReader.readLine()) != null) {
                    RemoteEntry entry = parseMlsd(line);
                    if (entry != null) {
                        entries.add(entry);
                    }
                }
            }
        } catch (IOException e) {
            hardClose();
            throw new IOException("Directory listing data transfer failed; the connection was closed.", e);
        } finally {
            releaseDataSocket(data);
        }

        try {
            expect(readReply(), 226, 250);
        } catch (IOException e) {
            hardClose();
            throw new IOException("Directory listing was not confirmed complete; the connection was closed.", e);
        }
        return entries;
    }

    synchronized void upload(String remotePath, InputStream input, TransferCommitGate gate) throws IOException {
        ensureConnected();
        if (input == null) {
            throw new IOException("Missing upload source.");
        }
        if (gate == null) {
            throw new IOException("Missing upload cancellation lifecycle.");
        }

        String path = normalizeRemotePath(remotePath);
        String tempPath = uploadTempPath(path);
        Socket data = openPassiveDataSocket();
        Reply start;
        try {
            start = command("STOR " + sanitizeArgument(tempPath));
        } catch (IOException e) {
            releaseDataSocket(data);
            hardClose();
            throw new IOException("Upload could not start; the connection was closed.", e);
        }
        if (start.code != 125 && start.code != 150) {
            releaseDataSocket(data);
            throw new IOException("Upload rejected: " + start.message);
        }

        try {
            try (OutputStream out = new BufferedOutputStream(data.getOutputStream())) {
                copy(input, out);
                out.flush();
            }
        } catch (IOException e) {
            hardClose();
            throw new IOException("Upload data transfer failed before final commit; the connection was closed.", e);
        } finally {
            releaseDataSocket(data);
        }

        final Reply terminal;
        try {
            terminal = readReply();
            expect(terminal, 226, 250);
        } catch (IOException e) {
            hardClose();
            throw new IOException("Upload was not confirmed complete; final remote name was not committed and the connection was closed.", e);
        }

        if (!gate.markReadyToCommit()) {
            deleteRemoteBestEffort(tempPath);
            hardClose();
            throw new IOException("Upload cancelled after staging completed; final remote name was not committed.");
        }
        if (!gate.beginCommit()) {
            deleteRemoteBestEffort(tempPath);
            hardClose();
            throw new IOException("Upload cancelled before final remote-name commit.");
        }

        final Reply renameFrom;
        try {
            renameFrom = command("RNFR " + sanitizeArgument(tempPath));
        } catch (IOException e) {
            hardClose();
            throw new IOException("Upload completed to staging, but final rename could not be started; the connection was closed.", e);
        }
        if (renameFrom.code != 350) {
            deleteRemoteBestEffort(tempPath);
            throw new IOException("Server does not support safe staged upload commit: " + renameFrom.message);
        }

        final Reply renameTo;
        try {
            renameTo = command("RNTO " + sanitizeArgument(path));
        } catch (IOException e) {
            hardClose();
            throw new IOException("Upload staging rename lost connection before final commit confirmation.", e);
        }
        if (renameTo.code != 250) {
            deleteRemoteBestEffort(tempPath);
            throw new IOException("Server rejected final staged upload commit: " + renameTo.message);
        }
        gate.finish();
    }

    synchronized void download(String remotePath, OutputStream output, TransferCommitGate gate) throws IOException {
        ensureConnected();
        if (output == null) {
            throw new IOException("Missing download destination.");
        }
        if (gate == null) {
            throw new IOException("Missing download cancellation lifecycle.");
        }
        String path = normalizeRemotePath(remotePath);
        Socket data = openPassiveDataSocket();
        Reply start;
        try {
            start = command("RETR " + sanitizeArgument(path));
        } catch (IOException e) {
            releaseDataSocket(data);
            hardClose();
            throw new IOException("Download could not start; the connection was closed.", e);
        }
        if (start.code != 125 && start.code != 150) {
            releaseDataSocket(data);
            throw new IOException("Download rejected: " + start.message);
        }

        try {
            try (InputStream in = new BufferedInputStream(data.getInputStream())) {
                copy(in, output);
                output.flush();
            }
        } catch (IOException e) {
            hardClose();
            throw new IOException("Download data transfer failed before local commit; the connection was closed.", e);
        } finally {
            releaseDataSocket(data);
        }

        try {
            expect(readReply(), 226, 250);
        } catch (IOException e) {
            hardClose();
            throw new IOException("Download was not confirmed complete; local final name was not committed and the connection was closed.", e);
        }
        if (!gate.markReadyToCommit()) {
            hardClose();
            throw new IOException("Download cancelled after transfer completion; local final name was not committed.");
        }
    }

    synchronized String pwd() throws IOException {
        ensureConnected();
        Reply reply = command("PWD");
        expect(reply, 257);
        int first = reply.message.indexOf('"');
        int second = first >= 0 ? reply.message.indexOf('"', first + 1) : -1;
        if (first >= 0 && second > first) {
            return normalizeRemotePath(reply.message.substring(first + 1, second));
        }
        return "/";
    }

    boolean isConnected() {
        return connected;
    }

    TransferCommitGate.CancelDisposition cancelActiveTransfer(TransferCommitGate gate) {
        TransferCommitGate.CancelDisposition disposition = gate == null
                ? TransferCommitGate.CancelDisposition.INTERRUPT_IO
                : gate.requestCancel();
        if (disposition == TransferCommitGate.CancelDisposition.TOO_LATE
                || disposition == TransferCommitGate.CancelDisposition.ALREADY_CANCELLED) {
            return disposition;
        }
        if (disposition == TransferCommitGate.CancelDisposition.CLEANUP_STAGING) {
            closeQuietly(activeDataSocket);
            return disposition;
        }
        connected = false;
        closeQuietly(activeDataSocket);
        closeQuietly(controlSocket);
        return disposition;
    }

    void cancelActiveTransfer() {
        connected = false;
        closeQuietly(activeDataSocket);
        closeQuietly(controlSocket);
    }

    void closeCancelledTransferSession() {
        hardClose();
    }

    @Override
    public synchronized void close() {
        if (writer != null && connected) {
            try {
                command("QUIT");
            } catch (IOException ignored) {
                // Hard close below is authoritative.
            }
        }
        hardClose();
    }

    private Socket openPassiveDataSocket() throws IOException {
        Socket plain = null;
        try {
            Reply epsv = command("EPSV");
            int dataPort;
            if (epsv.code == 229) {
                dataPort = parseEpsvPort(epsv.message);
            } else {
                Reply pasv = command("PASV");
                expect(pasv, 227);
                dataPort = parsePasvPort(pasv.message);
            }

            plain = new Socket();
            activeDataSocket = plain;
            if (!connected) {
                throw new IOException("Transfer was cancelled before the data connection opened.");
            }

            plain.connect(new InetSocketAddress(host, dataPort), CONNECT_TIMEOUT_MS);
            plain.setSoTimeout(READ_TIMEOUT_MS);
            if (!secure) {
                if (!connected) {
                    throw new IOException("Transfer was cancelled while the data connection opened.");
                }
                return plain;
            }

            SSLSocket tls = wrapTls(plain);
            activeDataSocket = tls;
            if (!connected) {
                throw new IOException("Transfer was cancelled while the protected data channel opened.");
            }
            return tls;
        } catch (IOException e) {
            if (plain != null) {
                releaseDataSocket(plain);
            }
            hardClose();
            throw new IOException("Passive data connection setup failed; the FTP session was closed.", e);
        }
    }

    private SSLSocket wrapTls(Socket plain) throws IOException {
        SSLSocketFactory factory = (SSLSocketFactory) SSLSocketFactory.getDefault();
        SSLSocket tls = (SSLSocket) factory.createSocket(plain, host, plain.getPort(), true);
        SSLParameters parameters = tls.getSSLParameters();
        parameters.setEndpointIdentificationAlgorithm("HTTPS");
        tls.setSSLParameters(parameters);
        tls.setSoTimeout(READ_TIMEOUT_MS);
        tls.startHandshake();
        return tls;
    }

    private void bindStreams(Socket socket) throws IOException {
        reader = new BufferedReader(new InputStreamReader(socket.getInputStream(), StandardCharsets.US_ASCII));
        writer = new BufferedWriter(new OutputStreamWriter(socket.getOutputStream(), StandardCharsets.US_ASCII));
    }

    private Reply command(String command) throws IOException {
        ensureStreams();
        writer.write(command);
        writer.write("\r\n");
        writer.flush();
        return readReply();
    }

    private Reply readReply() throws IOException {
        ensureStreams();
        String first = reader.readLine();
        if (first == null || first.length() < 3) {
            throw new IOException("FTP server closed the control connection.");
        }
        int code = parseCode(first);
        StringBuilder message = new StringBuilder(first);
        if (first.length() > 3 && first.charAt(3) == '-') {
            String terminal = String.format(Locale.ROOT, "%03d ", code);
            String line;
            do {
                line = reader.readLine();
                if (line == null) {
                    throw new IOException("FTP multiline response ended unexpectedly.");
                }
                message.append('\n').append(line);
            } while (!line.startsWith(terminal));
        }
        return new Reply(code, message.toString());
    }

    private static RemoteEntry parseMlsd(String line) {
        int split = line.indexOf(' ');
        if (split <= 0 || split + 1 >= line.length()) {
            return null;
        }
        String facts = line.substring(0, split).toLowerCase(Locale.ROOT);
        String name = line.substring(split + 1).trim();
        if (name.isEmpty() || ".".equals(name) || "..".equals(name) || facts.contains("type=cdir") || facts.contains("type=pdir")) {
            return null;
        }
        boolean directory = facts.contains("type=dir");
        long size = 0L;
        for (String fact : facts.split(";")) {
            if (fact.startsWith("size=")) {
                try {
                    size = Long.parseLong(fact.substring(5));
                } catch (NumberFormatException ignored) {
                    size = 0L;
                }
            }
        }
        return new RemoteEntry(name, directory, size);
    }

    private static int parseEpsvPort(String message) throws IOException {
        int open = message.indexOf('(');
        int close = message.indexOf(')', open + 1);
        if (open < 0 || close <= open) {
            throw new IOException("Invalid EPSV response.");
        }
        String payload = message.substring(open + 1, close);
        if (payload.isEmpty()) {
            throw new IOException("Invalid EPSV response.");
        }
        char delimiter = payload.charAt(0);
        String[] parts = payload.split(java.util.regex.Pattern.quote(String.valueOf(delimiter)), -1);
        if (parts.length < 4) {
            throw new IOException("Invalid EPSV response.");
        }
        try {
            return requirePort(Integer.parseInt(parts[3]));
        } catch (NumberFormatException e) {
            throw new IOException("Invalid EPSV port.", e);
        }
    }

    private static int parsePasvPort(String message) throws IOException {
        int open = message.indexOf('(');
        int close = message.indexOf(')', open + 1);
        if (open < 0 || close <= open) {
            throw new IOException("Invalid PASV response.");
        }
        String[] values = message.substring(open + 1, close).split(",");
        if (values.length != 6) {
            throw new IOException("Invalid PASV response.");
        }
        try {
            int high = Integer.parseInt(values[4].trim());
            int low = Integer.parseInt(values[5].trim());
            return requirePort(high * 256 + low);
        } catch (NumberFormatException e) {
            throw new IOException("Invalid PASV port.", e);
        }
    }

    static String joinRemote(String base, String name) throws IOException {
        String safeName = sanitizeArgument(name);
        String normalizedBase = normalizeRemotePath(base);
        if ("/".equals(normalizedBase)) {
            return "/" + safeName;
        }
        return normalizedBase + "/" + safeName;
    }

    static String parentRemote(String path) throws IOException {
        String normalized = normalizeRemotePath(path);
        if ("/".equals(normalized)) {
            return "/";
        }
        int slash = normalized.lastIndexOf('/');
        return slash <= 0 ? "/" : normalized.substring(0, slash);
    }

    static String normalizeRemotePath(String path) throws IOException {
        String value = path == null ? "/" : path.trim();
        if (value.isEmpty()) {
            value = "/";
        }
        if (value.indexOf('\0') >= 0 || value.indexOf('\r') >= 0 || value.indexOf('\n') >= 0) {
            throw new IOException("Remote path contains prohibited control characters.");
        }
        if (!value.startsWith("/")) {
            value = "/" + value;
        }
        while (value.contains("//")) {
            value = value.replace("//", "/");
        }
        return value;
    }

    private static String uploadTempPath(String finalPath) throws IOException {
        String parent = parentRemote(finalPath);
        return joinRemote(parent, ".ghostftp-upload-" + UUID.randomUUID() + ".part");
    }

    private void deleteRemoteBestEffort(String path) {
        if (!connected || writer == null) {
            return;
        }
        try {
            command("DELE " + sanitizeArgument(path));
        } catch (IOException ignored) {
            hardClose();
        }
    }

    private void releaseDataSocket(Socket socket) {
        closeQuietly(socket);
        if (activeDataSocket == socket) {
            activeDataSocket = null;
        }
    }

    private void hardClose() {
        connected = false;
        closeQuietly(activeDataSocket);
        activeDataSocket = null;
        closeQuietly(controlSocket);
        controlSocket = null;
        reader = null;
        writer = null;
    }

    private static String sanitizeArgument(String value) throws IOException {
        String safe = value == null ? "" : value;
        if (safe.indexOf('\0') >= 0 || safe.indexOf('\r') >= 0 || safe.indexOf('\n') >= 0) {
            throw new IOException("FTP argument contains prohibited control characters.");
        }
        return safe;
    }

    private static void copy(InputStream input, OutputStream output) throws IOException {
        byte[] buffer = new byte[64 * 1024];
        int read;
        while ((read = input.read(buffer)) != -1) {
            output.write(buffer, 0, read);
        }
    }

    private void ensureConnected() throws IOException {
        if (!connected) {
            throw new IOException("Not connected.");
        }
        ensureStreams();
    }

    private void ensureStreams() throws IOException {
        if (reader == null || writer == null) {
            throw new IOException("Control connection is not available.");
        }
    }

    private static void expect(Reply reply, int... allowed) throws IOException {
        for (int code : allowed) {
            if (reply.code == code) {
                return;
            }
        }
        throw new IOException("FTP server rejected operation: " + reply.message);
    }

    private static int parseCode(String line) throws IOException {
        try {
            return Integer.parseInt(line.substring(0, 3));
        } catch (RuntimeException e) {
            throw new IOException("Invalid FTP response: " + line, e);
        }
    }

    private static String requireHost(String host) {
        String value = host == null ? "" : host.trim();
        if (value.isEmpty() || value.indexOf('\0') >= 0 || value.indexOf('\r') >= 0 || value.indexOf('\n') >= 0) {
            throw new IllegalArgumentException("Host is required.");
        }
        return value;
    }

    private static int requirePort(int port) {
        if (port < 1 || port > 65535) {
            throw new IllegalArgumentException("Port must be between 1 and 65535.");
        }
        return port;
    }

    private static void closeQuietly(Socket socket) {
        if (socket == null) {
            return;
        }
        try {
            socket.close();
        } catch (IOException ignored) {
            // Best-effort cleanup.
        }
    }

    private static final class Reply {
        final int code;
        final String message;

        Reply(int code, String message) {
            this.code = code;
            this.message = message;
        }
    }
}
