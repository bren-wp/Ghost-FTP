package com.byftp.client.model;

import static org.junit.Assert.*;
import org.junit.Test;

public class ConnectionConfigSecurityTest {
    @Test public void rejectsInvalidPortsAndWhitespaceHosts() {
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "bad host", "21", "u", "p", ""));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com", "0", "u", "p", ""));
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com", "65536", "u", "p", ""));
    }

    @Test public void doesNotRequireFingerprintForFtpButRequiresItForSftp() {
        assertEquals("", ConnectionConfig.create(ConnectionConfig.Protocol.FTPS_EXPLICIT, "example.com", "21", "u", "p", "").fingerprint());
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", ""));
    }
}
