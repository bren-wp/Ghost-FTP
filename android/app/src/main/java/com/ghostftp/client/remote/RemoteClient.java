package com.ghostftp.client.remote;

import com.ghostftp.client.model.RemoteEntry;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.List;

public interface RemoteClient extends AutoCloseable {
    void connect() throws Exception;
    List<RemoteEntry> list(String directory) throws Exception;
    void upload(String remotePath, InputStream source) throws Exception;
    void download(String remotePath, OutputStream destination) throws Exception;
    void mkdir(String remotePath) throws Exception;
    void delete(String remotePath, boolean directory) throws Exception;
    void rename(String from, String to) throws Exception;
    @Override void close();
}
