package app.ghostftp.client;

import static org.junit.Assert.assertArrayEquals;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertThrows;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;

import org.junit.Test;

public final class RemoteTextDocumentTest {
    @Test
    public void decodePreservesCrLfStyleAndStrictUtf8() throws Exception {
        byte[] bytes = "alpha\r\nbeta\r\n".getBytes(StandardCharsets.UTF_8);
        RemoteTextDocument.Snapshot snapshot = RemoteTextDocument.decode(bytes);

        assertEquals("alpha\r\nbeta\r\n", snapshot.text);
        assertEquals(RemoteTextDocument.LineEnding.CRLF, snapshot.lineEnding);
        assertEquals(RemoteTextDocument.sha256(bytes), snapshot.sha256);
    }

    @Test
    public void sha256MatchesKnownDigest() throws Exception {
        assertEquals(
                "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
                RemoteTextDocument.sha256("abc".getBytes(StandardCharsets.UTF_8)));
    }

    @Test
    public void encodeForSaveRestoresOriginalLineEnding() throws Exception {
        byte[] encoded = RemoteTextDocument.encodeForSave("alpha\nbeta\n", RemoteTextDocument.LineEnding.CRLF);
        assertArrayEquals("alpha\r\nbeta\r\n".getBytes(StandardCharsets.UTF_8), encoded);
    }

    @Test
    public void decodeRejectsMixedLineEndingsInsteadOfNormalizingThem() {
        assertThrows(IOException.class, () -> RemoteTextDocument.decode(
                "alpha\r\nbeta\ngamma\rdelta".getBytes(StandardCharsets.UTF_8)));
    }

    @Test
    public void decodeRejectsBinaryMalformedAndOversizedContent() {
        assertThrows(IOException.class, () -> RemoteTextDocument.decode(new byte[]{'a', 0, 'b'}));
        assertThrows(IOException.class, () -> RemoteTextDocument.decode(new byte[]{(byte) 0xc3, (byte) 0x28}));

        byte[] oversized = new byte[WorkspaceOps.MAX_REMOTE_EDIT_BYTES + 1];
        Arrays.fill(oversized, (byte) 'a');
        assertThrows(IOException.class, () -> RemoteTextDocument.decode(oversized));
    }

    @Test
    public void requireUnchangedFailsClosedOnRemoteConflict() throws Exception {
        byte[] original = "first".getBytes(StandardCharsets.UTF_8);
        byte[] changed = "second".getBytes(StandardCharsets.UTF_8);
        String baseline = RemoteTextDocument.sha256(original);

        RemoteTextDocument.requireUnchanged(original, baseline);
        assertThrows(IOException.class, () -> RemoteTextDocument.requireUnchanged(changed, baseline));
        assertThrows(IOException.class, () -> RemoteTextDocument.requireUnchanged(original, ""));
    }

    @Test
    public void encodeRejectsEditedTextAboveBound() {
        char[] chars = new char[WorkspaceOps.MAX_REMOTE_EDIT_BYTES + 1];
        Arrays.fill(chars, 'a');
        assertThrows(IOException.class, () -> RemoteTextDocument.encodeForSave(
                new String(chars), RemoteTextDocument.LineEnding.LF));
    }
}
