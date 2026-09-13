package app.ghostftp.client;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertThrows;

import java.io.IOException;

import org.junit.Test;

public final class FtpSessionRemoteFileOperationTest {
    @Test
    public void mutableRemotePathNormalizesRelativePaths() throws Exception {
        assertEquals("/public_html/assets", FtpSession.requireMutableRemotePath("public_html//assets"));
    }

    @Test
    public void mutableRemotePathRejectsServerRootAndDotSegmentAliases() {
        assertThrows(IOException.class, () -> FtpSession.requireMutableRemotePath("/"));
        assertThrows(IOException.class, () -> FtpSession.requireMutableRemotePath(""));
        assertThrows(IOException.class, () -> FtpSession.requireMutableRemotePath("/."));
        assertThrows(IOException.class, () -> FtpSession.requireMutableRemotePath("/child/.."));
        assertThrows(IOException.class, () -> FtpSession.requireMutableRemotePath("../outside"));
        assertThrows(IOException.class, () -> FtpSession.requireMutableRemotePath("/safe/./item"));
    }

    @Test
    public void mutableRemotePathRejectsControlCharacterInjection() {
        assertThrows(IOException.class, () -> FtpSession.requireMutableRemotePath("/safe\r\nDELE /other"));
        assertThrows(IOException.class, () -> FtpSession.requireMutableRemotePath("/safe\u0000other"));
    }

    @Test
    public void chmodAcceptsOnlyThreeOrFourDigitOctalModes() throws Exception {
        assertEquals("755", FtpSession.normalizeChmodMode(" 755 "));
        assertEquals("0644", FtpSession.normalizeChmodMode("0644"));

        assertThrows(IOException.class, () -> FtpSession.normalizeChmodMode(""));
        assertThrows(IOException.class, () -> FtpSession.normalizeChmodMode("99"));
        assertThrows(IOException.class, () -> FtpSession.normalizeChmodMode("888"));
        assertThrows(IOException.class, () -> FtpSession.normalizeChmodMode("07555"));
        assertThrows(IOException.class, () -> FtpSession.normalizeChmodMode("7x5"));
    }
}
