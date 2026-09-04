package com.GhostFTP.client.model;

import static org.junit.Assert.*;
import org.junit.Test;

public class ConnectionConfigSecurityTest {
    private static final String VALID_SHA256 = "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA";

    @Test public void rejectsInvalidPortsAndWhitespaceHosts() {
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "bad host", "21", "u", "p", ""));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com", "0", "u", "p", ""));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com", "65536", "u", "p", ""));
    }

    @Test public void rejectsRawEndpointAndCredentialControlCharactersBeforeTrimming() {
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com\r\n", "21", "u", "p", ""));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com", "21\r\n", "u", "p", ""));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com", "21", "user\r\n", "p", ""));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com", "21", "user", "p\nextra", ""));
    }

    @Test public void doesNotRequireFingerprintForFtpButRequiresItForSftp() {
        assertEquals("", ConnectionConfig.create(ConnectionConfig.Protocol.FTPS_EXPLICIT, "example.com", "21", "u", "p", "").fingerprint());
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", ""));
    }

    @Test public void validatesAndCanonicalizesSftpSha256Fingerprint() {
        assertEquals(VALID_SHA256, ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", VALID_SHA256 + "=").fingerprint());
        assertEquals(VALID_SHA256, ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", VALID_SHA256.substring(7)).fingerprint());
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", "SHA256:not-base64***"));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", "SHA256:AAAA"));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", "SHA256:AAAA=BBBB"));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", VALID_SHA256 + "\r\n"));
    }
}
