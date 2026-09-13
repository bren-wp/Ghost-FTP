package app.ghostftp.client;

import java.io.IOException;
import java.nio.ByteBuffer;
import java.nio.CharBuffer;
import java.nio.charset.CharacterCodingException;
import java.nio.charset.CodingErrorAction;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;

final class RemoteTextDocument {
    enum LineEnding {
        LF,
        CRLF,
        CR
    }

    static final class Snapshot {
        final String text;
        final String sha256;
        final LineEnding lineEnding;

        Snapshot(String text, String sha256, LineEnding lineEnding) {
            this.text = text;
            this.sha256 = sha256;
            this.lineEnding = lineEnding;
        }
    }

    private RemoteTextDocument() {
    }

    static Snapshot decode(byte[] bytes) throws IOException {
        if (bytes == null) {
            throw new IOException("Remote Edit received no file data.");
        }
        if (bytes.length > WorkspaceOps.MAX_REMOTE_EDIT_BYTES) {
            throw new IOException("Remote Edit supports text files up to 1 MiB.");
        }
        for (byte value : bytes) {
            if (value == 0) {
                throw new IOException("Remote Edit rejected a binary file containing NUL bytes.");
            }
        }

        final String text;
        try {
            CharBuffer decoded = StandardCharsets.UTF_8.newDecoder()
                    .onMalformedInput(CodingErrorAction.REPORT)
                    .onUnmappableCharacter(CodingErrorAction.REPORT)
                    .decode(ByteBuffer.wrap(bytes));
            text = decoded.toString();
        } catch (CharacterCodingException e) {
            throw new IOException("Remote Edit supports strict UTF-8 text only.", e);
        }
        return new Snapshot(text, sha256(bytes), detectLineEnding(text));
    }

    static byte[] encodeForSave(String editorText, LineEnding lineEnding) throws IOException {
        String value = editorText == null ? "" : editorText;
        String canonical = value.replace("\r\n", "\n").replace('\r', '\n');
        String separator;
        switch (lineEnding == null ? LineEnding.LF : lineEnding) {
            case CRLF:
                separator = "\r\n";
                break;
            case CR:
                separator = "\r";
                break;
            case LF:
            default:
                separator = "\n";
                break;
        }
        byte[] encoded = canonical.replace("\n", separator).getBytes(StandardCharsets.UTF_8);
        if (encoded.length > WorkspaceOps.MAX_REMOTE_EDIT_BYTES) {
            throw new IOException("Edited text exceeds the 1 MiB Remote Edit safety limit.");
        }
        return encoded;
    }

    static void requireUnchanged(byte[] latest, String baselineSha256) throws IOException {
        String baseline = baselineSha256 == null ? "" : baselineSha256.trim();
        if (baseline.isEmpty() || !constantTimeEquals(sha256(latest), baseline)) {
            throw new IOException("Remote file changed since it was opened. Reload before saving to avoid overwriting newer data.");
        }
    }

    static String sha256(byte[] bytes) throws IOException {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hashed = digest.digest(bytes == null ? new byte[0] : bytes);
            StringBuilder hex = new StringBuilder(hashed.length * 2);
            for (byte value : hashed) {
                hex.append(String.format(java.util.Locale.ROOT, "%02x", value & 0xff));
            }
            return hex.toString();
        } catch (NoSuchAlgorithmException e) {
            throw new IOException("SHA-256 is unavailable.", e);
        }
    }

    private static LineEnding detectLineEnding(String text) {
        if (text.contains("\r\n")) {
            return LineEnding.CRLF;
        }
        if (text.indexOf('\r') >= 0) {
            return LineEnding.CR;
        }
        return LineEnding.LF;
    }

    private static boolean constantTimeEquals(String left, String right) {
        if (left.length() != right.length()) {
            return false;
        }
        int diff = 0;
        for (int i = 0; i < left.length(); i++) {
            diff |= left.charAt(i) ^ right.charAt(i);
        }
        return diff == 0;
    }
}
