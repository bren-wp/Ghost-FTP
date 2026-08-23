package com.byftp.client.remote;

import com.byftp.client.model.ConnectionConfig;

public final class RemoteClientFactory {
    private RemoteClientFactory() {}

    public static RemoteClient create(ConnectionConfig config) {
        return config.protocol() == ConnectionConfig.Protocol.SFTP
            ? new SftpRemoteClient(config)
            : new FtpRemoteClient(config);
    }
}
