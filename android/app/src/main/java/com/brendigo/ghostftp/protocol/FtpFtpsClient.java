package com.brendigo.ghostftp.protocol;

import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.Closeable;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.Reader;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;
import javax.net.ssl.SSLParameters;
import javax.net.ssl.SSLSocket;
import javax.net.ssl.SSLSocketFactory;

public final class FtpFtpsClient implements RemoteClient {
    private static final int CONNECT_TIMEOUT_MS = 15_000;
    private static final int IO_TIMEOUT_MS = 45_000;
    private static final int MAX_REPLY_LINES = 128;
    private static final int MAX_LIST_ENTRIES = 10_000;

    private Socket control;
    private Reader controlReader;
    private BufferedWriter controlWriter;
    private ConnectionSpec spec;
    private InetAddress peerAddress;
    private boolean secure;
    private String currentPath = "/";

    @Override
    public void connect(ConnectionSpec connectionSpec, TrustPrompt ignored) throws Exception {
        if (connectionSpec.protocol() == ConnectionSpec.Protocol.SFTP) {
            throw new IllegalArgumentException("SFTP requires the SSH client.");
        }
        close();
        spec = connectionSpec;
        secure = connectionSpec.protocol() == ConnectionSpec.Protocol.FTPS;

        Socket plain = new Socket();
        plain.connect(new InetSocketAddress(connectionSpec.host(), connectionSpec.port()), CONNECT_TIMEOUT_MS);
        plain.setSoTimeout(IO_TIMEOUT_MS);
        peerAddress = plain.getInetAddress();
        setControlSocket(plain);
        expect(readReply(), 220);

        if (secure) {
            Reply auth = command("AUTH TLS");
            expect(auth, 234, 334);
            SSLSocket tls = wrapTls(plain, connectionSpec.host(), connectionSpec.port());
            setControlSocket(tls);
            expect(command("PBSZ 0"), 200);
            expect(command("PROT P"), 200);
        }

        Reply user = command("USER " + safeCommandValue(connectionSpec.username()));
        if (user.code == 331 || user.code == 332) {
            char[] password = connectionSpec.passwordCopy();
            try {
                expect(secretCommand("PASS ", password), 230, 202);
            } finally {
                Arrays.fill(password, '\0');
            }
        } else {
            expect(user, 230);
        }
        expect(command("TYPE I"), 200);
        command("OPTS UTF8 ON");
        currentPath = readPwd();
    }

    private void setControlSocket(Socket socket) throws IOException {
        control = socket;
        controlReader = new BufferedReader(new InputStreamReader(socket.getInputStream(), StandardCharsets.UTF_8));
        controlWriter = new BufferedWriter(new OutputStreamWriter(socket.getOutputStream(), StandardCharsets.UTF_8));
    }

    private SSLSocket wrapTls(Socket socket, String host, int port) throws IOException {
        SSLSocketFactory factory = (SSLSocketFactory) SSLSocketFactory.getDefault();
        SSLSocket tls = (SSLSocket) factory.createSocket(socket, host, port, true);
        SSLParameters parameters = tls.getSSLParameters();
        parameters.setEndpointIdentificationAlgorithm("HTTPS");
        tls.setSSLParameters(parameters);
        tls.setSoTimeout(IO_TIMEOUT_MS);
        tls.startHandshake();
        return tls;
    }

    @Override
    public List<RemoteEntry> list(String path) throws Exception {
        requireConnected();
        String requested = RemotePaths.navigationPath(path);
        expect(command("CWD " + safeCommandValue(requested)), 250);
        currentPath = readPwd();

        try {
            List<String> lines = dataLines("MLSD");
            List<RemoteEntry> entries = new ArrayList<>();
            for (String line : lines) {
                RemoteEntry entry = parseMlsd(line);
                if (entry != null) {
                    entries.add(entry);
                }
            }
            sort(entries);
            return entries;
        } catch (FtpException error) {
            if (!error.isUnsupportedCommand()) {
                throw error;
            }
            List<RemoteEntry> entries = new ArrayList<>();
            for (String name : dataLines("NLST")) {
                if (!name.isEmpty() && !name.equals(".") && !name.equals("..")) {
                    entries.add(new RemoteEntry(name, false, false, 0));
                }
            }
            sort(entries);
            return entries;
        }
    }

    @Override
    public String currentPath() throws Exception {
        requireConnected();
        return currentPath;
    }

    @Override
    public void upload(InputStream source, String remoteName) throws Exception {
        requireConnected();
        String name = RemotePaths.leafName(remoteName);
        withData("STOR " + safeCommandValue(name), socket -> {
            try (OutputStream output = socket.getOutputStream()) {
                copy(source, output);
                output.flush();
            }
            return null;
        });
    }

    @Override
    public void download(RemoteEntry entry, OutputStream target) throws Exception {
        requireConnected();
        if (entry == null || entry.isDirectory() || entry.isSymlink()) {
            throw new IllegalArgumentException("Only an ordinary remote file can be downloaded.");
        }
        String name = RemotePaths.leafName(entry.name());
        withData("RETR " + safeCommandValue(name), socket -> {
            try (InputStream input = socket.getInputStream()) {
                copy(input, target);
                target.flush();
            }
            return null;
        });
    }

    @Override
    public void mkdir(String name) throws Exception {
        requireConnected();
        expect(command("MKD " + safeCommandValue(RemotePaths.leafName(name))), 257, 250);
    }

    @Override
    public void delete(RemoteEntry entry) throws Exception {
        requireConnected();
        if (entry == null || entry.isSymlink()) {
            throw new IllegalArgumentException("Symlink deletion is blocked.");
        }
        String name = RemotePaths.leafName(entry.name());
        if (entry.isDirectory()) {
            expect(command("RMD " + safeCommandValue(name)), 250);
        } else {
            expect(command("DELE " + safeCommandValue(name)), 250);
        }
    }

    @Override
    public boolean isConnected() {
        return control != null && control.isConnected() && !control.isClosed();
    }

    private String readPwd() throws Exception {
        Reply reply = command("PWD");
        expect(reply, 257);
        String text = reply.message;
        int first = text.indexOf('"');
        int last = text.lastIndexOf('"');
        if (first >= 0 && last > first) {
            return text.substring(first + 1, last).replace("\"\"", "\"");
        }
        return currentPath;
    }

    private List<String> dataLines(String ftpCommand) throws Exception {
        return withData(ftpCommand, socket -> {
            List<String> lines = new ArrayList<>();
            try (Reader reader = new BufferedReader(new InputStreamReader(socket.getInputStream(), StandardCharsets.UTF_8))) {
                while (lines.size() < MAX_LIST_ENTRIES) {
                    String line = readLineBounded(reader, 16_384);
                    if (line == null) {
                        break;
                    }
                    lines.add(line);
                }
                if (lines.size() >= MAX_LIST_ENTRIES && readLineBounded(reader, 16_384) != null) {
                    throw new IOException("Server listing exceeds the Android safety limit.");
                }
            }
            return lines;
        });
    }

    private <T> T withData(String ftpCommand, DataOperation<T> operation) throws Exception {
        Socket data = openPassiveSocket();
        Reply preliminary;
        try {
            writeCommand(ftpCommand);
            preliminary = readReply();
        } catch (Exception error) {
            closeQuietly(data);
            throw error;
        }
        if (preliminary.code != 125 && preliminary.code != 150) {
            closeQuietly(data);
            throw new FtpException(preliminary.code, preliminary.message);
        }
        try {
            if (secure) {
                data = wrapTls(data, spec.host(), data.getPort());
            }
            T result = operation.run(data);
            closeQuietly(data);
            expect(readReply(), 226, 250);
            return result;
        } catch (Exception error) {
            closeQuietly(data);
            hardClose();
            throw error;
        }
    }

    private Socket openPassiveSocket() throws Exception {
        Reply epsv = command("EPSV");
        int port;
        if (epsv.code == 229) {
            port = parseEpsvPort(epsv.message);
        } else {
            Reply pasv = command("PASV");
            expect(pasv, 227);
            port = parsePasvPort(pasv.message);
        }
        Socket data = new Socket();
        data.connect(new InetSocketAddress(peerAddress, port), CONNECT_TIMEOUT_MS);
        data.setSoTimeout(IO_TIMEOUT_MS);
        return data;
    }

    private static int parseEpsvPort(String message) throws IOException {
        int open = message.indexOf('(');
        int close = message.indexOf(')', open + 1);
        if (open < 0 || close < 0 || close <= open + 4) {
            throw new IOException("Invalid EPSV response.");
        }
        String body = message.substring(open + 1, close);
        char delimiter = body.charAt(0);
        int second = body.indexOf(delimiter, 1);
        int third = body.indexOf(delimiter, second + 1);
        int fourth = body.indexOf(delimiter, third + 1);
        if (second < 0 || third < 0 || fourth < 0) {
            throw new IOException("Invalid EPSV response.");
        }
        return parsePort(body.substring(third + 1, fourth));
    }

    private static int parsePasvPort(String message) throws IOException {
        int open = message.indexOf('(');
        int close = message.indexOf(')', open + 1);
        if (open < 0 || close < 0) {
            throw new IOException("Invalid PASV response.");
        }
        String[] parts = message.substring(open + 1, close).split(",");
        if (parts.length != 6) {
            throw new IOException("Invalid PASV response.");
        }
        int high = parseByte(parts[4]);
        int low = parseByte(parts[5]);
        return high * 256 + low;
    }

    private static int parseByte(String value) throws IOException {
        try {
            int parsed = Integer.parseInt(value.trim());
            if (parsed < 0 || parsed > 255) {
                throw new IOException("Invalid PASV response.");
            }
            return parsed;
        } catch (NumberFormatException error) {
            throw new IOException("Invalid PASV response.", error);
        }
    }

    private static int parsePort(String value) throws IOException {
        try {
            int parsed = Integer.parseInt(value.trim());
            if (parsed < 1 || parsed > 65535) {
                throw new IOException("Invalid passive port.");
            }
            return parsed;
        } catch (NumberFormatException error) {
            throw new IOException("Invalid passive port.", error);
        }
    }

    private RemoteEntry parseMlsd(String line) {
        int split = line.indexOf(' ');
        if (split <= 0 || split + 1 >= line.length()) {
            return null;
        }
        String facts = line.substring(0, split);
        String name = line.substring(split + 1);
        if (name.equals(".") || name.equals("..") || name.isEmpty()) {
            return null;
        }
        String type = "";
        long size = 0;
        for (String fact : facts.split(";")) {
            int equals = fact.indexOf('=');
            if (equals <= 0) {
                continue;
            }
            String key = fact.substring(0, equals).toLowerCase(Locale.ROOT);
            String value = fact.substring(equals + 1);
            if (key.equals("type")) {
                type = value.toLowerCase(Locale.ROOT);
            } else if (key.equals("size")) {
                try {
                    size = Long.parseLong(value);
                } catch (NumberFormatException ignored) {
                    size = 0;
                }
            }
        }
        if (type.equals("cdir") || type.equals("pdir")) {
            return null;
        }
        boolean symlink = type.contains("slink") || type.contains("symlink");
        boolean directory = type.equals("dir") && !symlink;
        return new RemoteEntry(name, directory, symlink, size);
    }

    private static void sort(List<RemoteEntry> entries) {
        entries.sort(Comparator
                .comparing(RemoteEntry::isDirectory).reversed()
                .thenComparing(entry -> entry.name().toLowerCase(Locale.ROOT))
                .thenComparing(RemoteEntry::name));
    }

    private Reply command(String command) throws Exception {
        writeCommand(command);
        return readReply();
    }

    private Reply secretCommand(String prefix, char[] secret) throws Exception {
        requireConnected();
        for (char value : secret) {
            if (value == '\0' || value == '\r' || value == '\n') {
                throw new IllegalArgumentException("Secret contains unsupported control data.");
            }
        }
        controlWriter.write(prefix);
        controlWriter.write(secret);
        controlWriter.write("\r\n");
        controlWriter.flush();
        return readReply();
    }

    private void writeCommand(String command) throws Exception {
        requireConnected();
        controlWriter.write(safeCommandValue(command));
        controlWriter.write("\r\n");
        controlWriter.flush();
    }

    private static String safeCommandValue(String value) {
        if (value == null || value.indexOf('\0') >= 0 || value.indexOf('\r') >= 0 || value.indexOf('\n') >= 0) {
            throw new IllegalArgumentException("FTP command contains unsupported control data.");
        }
        return value;
    }

    private Reply readReply() throws Exception {
        requireConnected();
        String first = readLineBounded(controlReader, 8192);
        if (first == null || first.length() < 3) {
            throw new IOException("FTP server closed the control connection.");
        }
        int code;
        try {
            code = Integer.parseInt(first.substring(0, 3));
        } catch (NumberFormatException error) {
            throw new IOException("FTP server returned an invalid reply.", error);
        }
        StringBuilder message = new StringBuilder(first);
        if (first.length() > 3 && first.charAt(3) == '-') {
            String terminator = String.format(Locale.ROOT, "%03d ", code);
            for (int i = 1; i < MAX_REPLY_LINES; i++) {
                String line = readLineBounded(controlReader, 8192);
                if (line == null) {
                    throw new IOException("FTP server closed a multiline reply.");
                }
                message.append('\n').append(line);
                if (line.startsWith(terminator)) {
                    return new Reply(code, message.toString());
                }
            }
            throw new IOException("FTP server reply exceeds the safety limit.");
        }
        return new Reply(code, message.toString());
    }

    private static String readLineBounded(Reader reader, int maxChars) throws IOException {
        StringBuilder line = new StringBuilder();
        while (true) {
            int value = reader.read();
            if (value < 0) {
                return line.length() == 0 ? null : line.toString();
            }
            if (value == '\n') {
                if (line.length() > 0 && line.charAt(line.length() - 1) == '\r') {
                    line.setLength(line.length() - 1);
                }
                return line.toString();
            }
            line.append((char) value);
            if (line.length() > maxChars) {
                throw new IOException("Server line exceeds the safety limit.");
            }
        }
    }

    private static void expect(Reply reply, int... accepted) throws FtpException {
        for (int code : accepted) {
            if (reply.code == code) {
                return;
            }
        }
        throw new FtpException(reply.code, reply.message);
    }

    private void requireConnected() throws IOException {
        if (control == null || control.isClosed() || !control.isConnected()) {
            throw new IOException("FTP connection is not active.");
        }
    }

    private static void copy(InputStream input, OutputStream output) throws IOException {
        byte[] buffer = new byte[64 * 1024];
        int read;
        while ((read = input.read(buffer)) >= 0) {
            if (read > 0) {
                output.write(buffer, 0, read);
            }
        }
    }

    private void hardClose() {
        closeQuietly(control);
        control = null;
        controlReader = null;
        controlWriter = null;
        peerAddress = null;
    }

    @Override
    public void close() {
        Socket socket = control;
        if (socket != null && socket.isConnected() && !socket.isClosed()) {
            try {
                writeCommand("QUIT");
                readReply();
            } catch (Exception ignored) {
                // Transport is being closed regardless of QUIT support/state.
            }
        }
        hardClose();
        if (spec != null) {
            spec.clearPassword();
            spec = null;
        }
    }

    private static void closeQuietly(Closeable closeable) {
        if (closeable == null) {
            return;
        }
        try {
            closeable.close();
        } catch (IOException ignored) {
            // Best-effort close after the operation has already failed.
        }
    }

    private interface DataOperation<T> {
        T run(Socket socket) throws Exception;
    }

    private static final class Reply {
        final int code;
        final String message;

        Reply(int code, String message) {
            this.code = code;
            this.message = message;
        }
    }

    private static final class FtpException extends IOException {
        private final int code;

        FtpException(int code, String message) {
            super("FTP " + code + ": " + message);
            this.code = code;
        }

        boolean isUnsupportedCommand() {
            return code == 500 || code == 501 || code == 502 || code == 504;
        }
    }
}
