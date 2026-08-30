package com.byftp.client.model;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import java.util.List;
import org.junit.Test;

public final class SharedHostingDiagnosticsTest {
    @Test public void prefersPublicHtmlAndReportsSecureFtps() {
        SharedHostingDiagnostics got = SharedHostingDiagnostics.analyze(
            ConnectionConfig.Protocol.FTPS_EXPLICIT,
            List.of(
                new RemoteEntry("www", true, 0, 0),
                new RemoteEntry("public_html", true, 0, 0),
                new RemoteEntry("index.php", false, 12, 0)
            )
        );

        assertTrue(got.secure());
        assertEquals("account", got.rootMode());
        assertTrue(got.webRootDetected());
        assertEquals("public_html", got.webRoot());
        assertEquals(3, got.rootEntryCount());
    }

    @Test public void plainFtpRemainsVisibleAsInsecureAndFilesAreNotWebRoots() {
        SharedHostingDiagnostics got = SharedHostingDiagnostics.analyze(
            ConnectionConfig.Protocol.FTP,
            List.of(
                new RemoteEntry("public_html", false, 20, 0),
                new RemoteEntry("htdocs", true, 0, 0)
            )
        );

        assertFalse(got.secure());
        assertTrue(got.webRootDetected());
        assertEquals("htdocs", got.webRoot());
    }

    @Test public void sftpUsesHomeRootWithoutInventingWebRoot() {
        SharedHostingDiagnostics got = SharedHostingDiagnostics.analyze(
            ConnectionConfig.Protocol.SFTP,
            List.of(new RemoteEntry("backups", true, 0, 0))
        );

        assertTrue(got.secure());
        assertEquals("home", got.rootMode());
        assertFalse(got.webRootDetected());
        assertEquals("", got.webRoot());
    }
}
