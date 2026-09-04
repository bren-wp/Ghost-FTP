package com.ghostftp.client.remote;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertThrows;

import org.junit.Test;

public final class FtpRemoteClientPathTest {
    @Test public void mapsUiRootToLoginWorkingDirectory() {
        assertEquals("/home/example", FtpRemoteClient.mapLoginRelativePath("/home/example/", "/"));
        assertEquals("/home/example/public_html", FtpRemoteClient.mapLoginRelativePath("/home/example", "/public_html"));
        assertEquals("/home/example/public_html/index.php", FtpRemoteClient.mapLoginRelativePath("/home/example", "/public_html/index.php"));
    }

    @Test public void fallsBackToLoginRelativePathsWhenPwdIsUnavailable() {
        assertEquals(".", FtpRemoteClient.mapLoginRelativePath(null, "/"));
        assertEquals("public_html", FtpRemoteClient.mapLoginRelativePath("", "/public_html"));
        assertEquals("public_html/index.php", FtpRemoteClient.mapLoginRelativePath(".", "/public_html/index.php"));
    }

    @Test public void preservesVirtualRootServers() {
        assertEquals("/", FtpRemoteClient.mapLoginRelativePath("/", "/"));
        assertEquals("/public_html", FtpRemoteClient.mapLoginRelativePath("/", "/public_html"));
    }

    @Test public void rejectsTraversalAndNonCanonicalUiPaths() {
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/example", "public_html"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/example", "/../etc"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/example", "/public_html/../etc"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/example", "/public_html//assets"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/example", "/public_html\\assets"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/example", "/public_html/\0secret"));
    }

    @Test public void rejectsUnsafeServerLoginDirectory() {
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/../etc", "/"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home//example", "/"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/\0example", "/"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/example\r", "/"));
        assertThrows(IllegalArgumentException.class, () -> FtpRemoteClient.mapLoginRelativePath("/home/example\n", "/"));
    }
}
