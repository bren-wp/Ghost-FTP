package com.byftp.client.model;

import static org.junit.Assert.*;
import org.junit.Test;

public class ConnectionConfigTest {
    @Test public void defaultsPortsByProtocol() {
        assertEquals(21, ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "example.com", "", "u", "p", "").port());
        assertEquals(990, ConnectionConfig.create(ConnectionConfig.Protocol.FTPS_IMPLICIT, "example.com", "", "u", "p", "").port());
        assertEquals(22, ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "", "u", "p", "SHA256:abc").port());
    }

    @Test public void sftpRequiresFingerprint() {
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "example.com", "22", "u", "p", ""));
    }

    @Test public void rejectsUrlInsteadOfHost() {
        assertThrows(IllegalArgumentException.class, () -> ConnectionConfig.create(ConnectionConfig.Protocol.FTP, "ftp://example.com/path", "21", "u", "p", ""));
    }

    @Test public void normalizesBracketedIpv6AndFingerprint() {
        ConnectionConfig c = ConnectionConfig.create(ConnectionConfig.Protocol.SFTP, "[2001:db8::1]", "22", "user", "pw", "abc=");
        assertEquals("2001:db8::1", c.host());
        assertEquals("SHA256:abc", c.fingerprint());
    }
}
