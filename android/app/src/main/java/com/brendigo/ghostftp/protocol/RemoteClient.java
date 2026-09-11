package com.brendigo.ghostftp.protocol;

import java.io.Closeable;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.List;

public interface RemoteClient extends Closeable {
    void connect(ConnectionSpec spec, TrustPrompt trustPrompt) throws Exception;
    List<RemoteEntry> list(String path) throws Exception;
    String currentPath() throws Exception;
    void upload(InputStream source, String remoteName) throws Exception;
    void download(RemoteEntry entry, OutputStream target) throws Exception;
    void mkdir(String name) throws Exception;
    void delete(RemoteEntry entry) throws Exception;
    boolean isConnected();
}
