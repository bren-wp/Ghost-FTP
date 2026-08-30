package com.byftp.client.model;

public final class DocumentName {
    private DocumentName() {}

    public static String resolve(String providerName, String lastPathSegment) {
        if (providerName != null && !providerName.isBlank()) {
            return providerName;
        }
        if (lastPathSegment != null && !lastPathSegment.isBlank()) {
            int slash = lastPathSegment.lastIndexOf('/');
            String tail = slash >= 0 ? lastPathSegment.substring(slash + 1) : lastPathSegment;
            if (!tail.isBlank()) {
                return tail;
            }
        }
        return "upload.bin";
    }
}
