package com.GhostFTP.client.remote;

import com.GhostFTP.client.model.ConnectionConfig;

public final class RemoteClientFactory {
    private RemoteClientFactory() {}

    public static RemoteClient create(ConnectionConfig config) {
        return config.protocol() == ConnectionConfig.Protocol.SFTP
            ? new SftpRemoteClient(config)
            : new FtpRemoteClient(config);
    }
}
