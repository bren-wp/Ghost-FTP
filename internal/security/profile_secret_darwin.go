//go:build darwin

package security

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

static CFMutableDictionaryRef ghostftp_keychain_base_query(void) {
    CFMutableDictionaryRef query = CFDictionaryCreateMutable(
        kCFAllocatorDefault,
        0,
        &kCFTypeDictionaryKeyCallBacks,
        &kCFTypeDictionaryValueCallBacks
    );
    if (query == NULL) {
        return NULL;
    }
    CFStringRef service = CFSTR("app.ghostftp.client.profile-key-v1");
    CFStringRef account = CFSTR("Ghost FTP persistent profile master key");
    CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(query, kSecAttrService, service);
    CFDictionarySetValue(query, kSecAttrAccount, account);
    return query;
}

static OSStatus ghostftp_keychain_copy(unsigned char **outBytes, CFIndex *outLength) {
    if (outBytes == NULL || outLength == NULL) {
        return errSecParam;
    }
    *outBytes = NULL;
    *outLength = 0;
    CFMutableDictionaryRef query = ghostftp_keychain_base_query();
    if (query == NULL) {
        return errSecAllocate;
    }
    CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
    CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);
    CFTypeRef result = NULL;
    OSStatus status = SecItemCopyMatching(query, &result);
    CFRelease(query);
    if (status != errSecSuccess) {
        if (result != NULL) CFRelease(result);
        return status;
    }
    if (result == NULL || CFGetTypeID(result) != CFDataGetTypeID()) {
        if (result != NULL) CFRelease(result);
        return errSecDecode;
    }
    CFDataRef data = (CFDataRef)result;
    CFIndex length = CFDataGetLength(data);
    if (length <= 0) {
        CFRelease(result);
        return errSecDecode;
    }
    unsigned char *copy = (unsigned char *)malloc((size_t)length);
    if (copy == NULL) {
        CFRelease(result);
        return errSecAllocate;
    }
    memcpy(copy, CFDataGetBytePtr(data), (size_t)length);
    *outBytes = copy;
    *outLength = length;
    CFRelease(result);
    return errSecSuccess;
}

static OSStatus ghostftp_keychain_add(const unsigned char *bytes, CFIndex length) {
    if (bytes == NULL || length <= 0) {
        return errSecParam;
    }
    CFMutableDictionaryRef query = ghostftp_keychain_base_query();
    if (query == NULL) {
        return errSecAllocate;
    }
    CFDataRef data = CFDataCreate(kCFAllocatorDefault, bytes, length);
    if (data == NULL) {
        CFRelease(query);
        return errSecAllocate;
    }
    CFDictionarySetValue(query, kSecValueData, data);
    CFDictionarySetValue(query, kSecAttrAccessible, kSecAttrAccessibleWhenUnlockedThisDeviceOnly);
    OSStatus status = SecItemAdd(query, NULL);
    CFRelease(data);
    CFRelease(query);
    return status;
}

static int ghostftp_status_not_found(OSStatus status) { return status == errSecItemNotFound; }
static int ghostftp_status_duplicate(OSStatus status) { return status == errSecDuplicateItem; }
*/
import "C"

import (
    "bytes"
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
    "unsafe"
)

const (
    darwinPersistentProfilePrefix = "darwin-keychain-aesgcm-v1:"
    darwinPersistentProfileKeyLen = 32
    darwinPersistentProfileMax    = 8 << 20
    persistentProfileSecretMarker = "profile-secret-v1\x00"
)

var darwinPersistentProfileAAD = []byte("Ghost FTP Keychain persistent profile protection v1")

func readDarwinProfileMasterKey() ([]byte, error) {
    var out *C.uchar
    var length C.CFIndex
    status := C.ghostftp_keychain_copy(&out, &length)
    if status != 0 {
        if C.ghostftp_status_not_found(status) != 0 {
            return nil, osStatusError("read", int32(status))
        }
        return nil, osStatusError("read", int32(status))
    }
    if out == nil || int64(length) != darwinPersistentProfileKeyLen {
        if out != nil {
            C.free(unsafe.Pointer(out))
        }
        return nil, errors.New("macOS Keychain profile master key has an invalid size")
    }
    defer C.free(unsafe.Pointer(out))
    key := C.GoBytes(unsafe.Pointer(out), C.int(length))
    if len(key) != darwinPersistentProfileKeyLen {
        WipeBytes(key)
        return nil, errors.New("macOS Keychain profile master key could not be read")
    }
    return key, nil
}

func osStatusError(operation string, status int32) error {
    return fmt.Errorf("macOS Keychain profile key %s failed (OSStatus %d)", operation, status)
}

func darwinProfileMasterKey() ([]byte, error) {
    key, err := readDarwinProfileMasterKey()
    if err == nil {
        return key, nil
    }

    generated := make([]byte, darwinPersistentProfileKeyLen)
    if _, randErr := rand.Read(generated); randErr != nil {
        return nil, randErr
    }
    ptr := C.CBytes(generated)
    if ptr == nil {
        WipeBytes(generated)
        return nil, errors.New("macOS Keychain profile master key allocation failed")
    }
    status := C.ghostftp_keychain_add((*C.uchar)(ptr), C.CFIndex(len(generated)))
    C.free(ptr)
    if status == 0 {
        return generated, nil
    }
    WipeBytes(generated)
    if C.ghostftp_status_duplicate(status) != 0 {
        return readDarwinProfileMasterKey()
    }
    return nil, osStatusError("create", int32(status))
}

func persistentProfileAEAD() (cipher.AEAD, []byte, error) {
    key, err := darwinProfileMasterKey()
    if err != nil {
        return nil, nil, err
    }
    block, err := aes.NewCipher(key)
    if err != nil {
        WipeBytes(key)
        return nil, nil, err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        WipeBytes(key)
        return nil, nil, err
    }
    return aead, key, nil
}

// ProtectPersistentProfileBytes encrypts durable profile data with an AES-256
// key kept in the user's macOS Keychain. The key is non-synchronizing and
// accessible only while this device's Keychain is unlocked.
func ProtectPersistentProfileBytes(data []byte) (string, error) {
    if len(data) == 0 {
        return "", nil
    }
    if len(data) > darwinPersistentProfileMax {
        return "", errors.New("macOS persistent profile data is too large")
    }
    aead, key, err := persistentProfileAEAD()
    if err != nil {
        return "", err
    }
    defer WipeBytes(key)
    nonce := make([]byte, aead.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return "", err
    }
    sealed := aead.Seal(nil, nonce, data, darwinPersistentProfileAAD)
    payload := make([]byte, 0, len(nonce)+len(sealed))
    payload = append(payload, nonce...)
    payload = append(payload, sealed...)
    encoded := base64.RawURLEncoding.EncodeToString(payload)
    WipeBytes(payload)
    WipeBytes(sealed)
    return darwinPersistentProfilePrefix + encoded, nil
}

func UnprotectPersistentProfileBytes(encoded string) ([]byte, error) {
    if encoded == "" {
        return nil, nil
    }
    if !bytes.HasPrefix([]byte(encoded), []byte(darwinPersistentProfilePrefix)) {
        return nil, errors.New("macOS persistent profile data uses an unsupported format")
    }
    raw, err := base64.RawURLEncoding.DecodeString(encoded[len(darwinPersistentProfilePrefix):])
    if err != nil || len(raw) > darwinPersistentProfileMax+256 {
        WipeBytes(raw)
        return nil, errors.New("macOS persistent profile data is malformed")
    }
    defer WipeBytes(raw)
    aead, key, err := persistentProfileAEAD()
    if err != nil {
        return nil, err
    }
    defer WipeBytes(key)
    if len(raw) <= aead.NonceSize()+aead.Overhead() {
        return nil, errors.New("macOS persistent profile data is truncated")
    }
    nonce := raw[:aead.NonceSize()]
    ciphertext := raw[aead.NonceSize():]
    plain, err := aead.Open(nil, nonce, ciphertext, darwinPersistentProfileAAD)
    if err != nil {
        return nil, errors.New("macOS persistent profile data failed authentication")
    }
    return plain, nil
}

// ProtectRuntimeBytes deliberately reuses the existing short-lived, same-user
// runtime secret broker. Persistent Keychain material is never handed directly
// to ssh/curl helpers.
func ProtectRuntimeBytes(data []byte) (string, error) { return ProtectBytes(data) }

// PersistentProfileSecretToRuntime converts a saved credential into a fresh
// runtime capability. The returned blob is owned by the connection attempt and
// must be forgotten when the attempt/session is finished.
func PersistentProfileSecretToRuntime(encoded string) (string, bool, error) {
    if encoded == "" {
        return "", false, nil
    }
    plain, err := UnprotectPersistentProfileBytes(encoded)
    if err != nil {
        return "", false, err
    }
    defer WipeBytes(plain)
    marker := []byte(persistentProfileSecretMarker)
    if !bytes.HasPrefix(plain, marker) || len(plain) == len(marker) {
        return "", false, errors.New("macOS saved profile credential is malformed")
    }
    runtimeBlob, err := ProtectRuntimeBytes(plain[len(marker):])
    if err != nil {
        return "", false, err
    }
    return runtimeBlob, true, nil
}
