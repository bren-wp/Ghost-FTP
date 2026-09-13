package app.ghostftp.client;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertThrows;

import java.io.IOException;

import org.junit.Test;

public final class FtpSessionPassiveReplyTest {
    @Test
    public void acceptsStrictEpsvReply() throws Exception {
        assertEquals(6446, FtpSession.parseEpsvPort("229 Entering Extended Passive Mode (|||6446|)"));
    }

    @Test
    public void rejectsMalformedEpsvDelimiters() {
        assertThrows(IOException.class,
                () -> FtpSession.parseEpsvPort("229 Entering Extended Passive Mode (|||6446)"));
        assertThrows(IOException.class,
                () -> FtpSession.parseEpsvPort("229 Entering Extended Passive Mode (||!6446|)"));
        assertThrows(IOException.class,
                () -> FtpSession.parseEpsvPort("229 Entering Extended Passive Mode (|||64x6|)"));
    }

    @Test
    public void rejectsOutOfRangeEpsvPort() {
        assertThrows(IOException.class,
                () -> FtpSession.parseEpsvPort("229 Entering Extended Passive Mode (|||0|)"));
        assertThrows(IOException.class,
                () -> FtpSession.parseEpsvPort("229 Entering Extended Passive Mode (|||65536|)"));
    }

    @Test
    public void acceptsStrictPasvReply() throws Exception {
        assertEquals(50000, FtpSession.parsePasvPort("227 Entering Passive Mode (127,0,0,1,195,80)"));
    }

    @Test
    public void rejectsNegativeOrOversizedPasvOctets() {
        assertThrows(IOException.class,
                () -> FtpSession.parsePasvPort("227 Entering Passive Mode (127,0,0,1,-1,80)"));
        assertThrows(IOException.class,
                () -> FtpSession.parsePasvPort("227 Entering Passive Mode (127,0,0,1,256,1)"));
        assertThrows(IOException.class,
                () -> FtpSession.parsePasvPort("227 Entering Passive Mode (999,0,0,1,195,80)"));
    }

    @Test
    public void rejectsMalformedPasvFieldsAndZeroPort() {
        assertThrows(IOException.class,
                () -> FtpSession.parsePasvPort("227 Entering Passive Mode (127,0,0,1,abc,80)"));
        assertThrows(IOException.class,
                () -> FtpSession.parsePasvPort("227 Entering Passive Mode (127,0,0,1,0,0)"));
        assertThrows(IOException.class,
                () -> FtpSession.parsePasvPort("227 Entering Passive Mode (127,0,0,1,195)"));
    }
}
