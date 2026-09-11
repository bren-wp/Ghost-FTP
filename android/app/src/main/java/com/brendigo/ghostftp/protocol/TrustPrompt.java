package com.brendigo.ghostftp.protocol;

public interface TrustPrompt {
    boolean confirmNewHostKey(String host, String algorithm, String fingerprint);
    void reportChangedHostKey(String host, String fingerprint);
}
