package app.ghostftp.client;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.assertTrue;

import org.junit.Test;

public final class FtpSessionMlsdMetadataTest {
    @Test
    public void regularFileMetadataIsExplicitlyClassified() {
        RemoteEntry entry = FtpSession.parseMlsd(
                "type=file;size=12;modify=20260914030000;unix.mode=0755; script.sh");

        assertEquals("script.sh", entry.name);
        assertFalse(entry.directory);
        assertTrue(entry.regularFile);
        assertEquals("file", entry.type);
        assertEquals("0755", entry.permissions);
        assertTrue(entry.modifiedEpochMillis > 0L);
    }

    @Test
    public void symlinkLikeEntryIsNotTreatedAsRegularFile() {
        RemoteEntry entry = FtpSession.parseMlsd(
                "type=os.unix=slink;size=7;modify=20260914030000;unix.mode=0777; current");

        assertEquals("current", entry.name);
        assertFalse(entry.directory);
        assertFalse(entry.regularFile);
        assertEquals("os.unix=slink", entry.type);
    }

    @Test
    public void directoryRemainsDirectoryAndNotRegularFile() {
        RemoteEntry entry = FtpSession.parseMlsd(
                "type=dir;modify=20260914030000;unix.mode=0750; config");

        assertTrue(entry.directory);
        assertFalse(entry.regularFile);
        assertEquals("dir", entry.type);
    }

    @Test
    public void unsafeChildPathSeparatorsAreRejected() {
        assertNull(FtpSession.parseMlsd("type=file;size=1; sub/file.txt"));
        assertNull(FtpSession.parseMlsd("type=file;size=1; sub\\file.txt"));
        assertNull(FtpSession.parseMlsd("type=file;size=1; ../escape.txt"));
    }

    @Test
    public void controlCharactersAreRejectedFromChildNames() {
        assertNull(FtpSession.parseMlsd("type=file;size=1; bad\u0001name.txt"));
        assertNull(FtpSession.parseMlsd("type=file;size=1; bad\u007fname.txt"));
    }

    @Test
    public void safeUnicodeChildNameRemainsSupported() {
        RemoteEntry entry = FtpSession.parseMlsd(
                "type=file;size=2;modify=20260914030000; račun-✓.txt");

        assertEquals("račun-✓.txt", entry.name);
        assertTrue(entry.regularFile);
    }
}
