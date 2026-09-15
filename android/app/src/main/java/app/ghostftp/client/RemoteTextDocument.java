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
    private static final char[] HEX = "0123456789abcdef".toCharArray();

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
            throw new IOException("The server returned no file data.");
        }
        if (bytes.length > WorkspaceOps.MAX_REMOTE_EDIT_BYTES) {
            throw new IOException("This file is larger than the 1 MiB editing limit.");
        }
        for (byte value : bytes) {
            if (value == 0) {
                throw new IOException("This file appears to be binary and cannot be edited as text.");
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
            throw new IOException("This file is not valid UTF-8 text and cannot be edited safely.", e);
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
            throw new IOException("The edited file is larger than the 1 MiB editing limit.");
        }
        return encoded;
    }

    static void requireUnchanged(byte[] latest, String baselineSha256) throws IOException {
        String baseline = baselineSha256 == null ? "" : baselineSha256.trim();
        if (baseline.isEmpty() || !constantTimeEquals(sha256(latest), baseline)) {
            throw new IOException("The remote file changed after you opened it. Reload before saving so newer server changes are not overwritten.");
        }
    }

    static String sha256(byte[] bytes) throws IOException {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hashed = digest.digest(bytes == null ? new byte[0] : bytes);
            char[] hex = new char[hashed.length * 2];
            for (int i = 0; i < hashed.length; i++) {
                int value = hashed[i] & 0xff;
                hex[i * 2] = HEX[value >>> 4];
                hex[i * 2 + 1] = HEX[value & 0x0f];
            }
            return new String(hex);
        } catch (NoSuchAlgorithmException e) {
            throw new IOException("Secure change detection is unavailable on this device.", e);
        }
    }

    private static LineEnding detectLineEnding(String text) throws IOException {
        boolean sawLf = false;
        boolean sawCrLf = false;
        boolean sawCr = false;
        for (int i = 0; i < text.length(); i++) {
            char ch = text.charAt(i);
            if (ch == '\r') {
                if (i + 1 < text.length() && text.charAt(i + 1) == '\n') {
                    sawCrLf = true;
                    i++;
                } else {
                    sawCr = true;
                }
            } else if (ch == '\n') {
                sawLf = true;
            }
        }
        int styles = (sawLf ? 1 : 0) + (sawCrLf ? 1 : 0) + (sawCr ? 1 : 0);
        if (styles > 1) {
            throw new IOException("This file uses mixed line endings. Normalize it before editing to avoid unintended changes.");
        }
        if (sawCrLf) return LineEnding.CRLF;
        if (sawCr) return LineEnding.CR;
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
