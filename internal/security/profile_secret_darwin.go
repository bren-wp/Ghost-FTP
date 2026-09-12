//go:build darwin

package security

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static CFMutableDictionaryRef ghostftp_profile_key_query(void) {
    CFMutableDictionaryRef query = CFDictionaryCreateMutable(
        kCFAllocatorDefault, 0,
        &kCFTypeDictionaryKeyCallBacks,
        &kCFTypeDictionaryValueCallBacks);
    if (query == NULL) return NULL;

    CFStringRef service = CFStringCreateWithCString(
        kCFAllocatorDefault,
        "com.brendigo.ghostftp.profile-master-key",
        kCFStringEncodingUTF8);
    CFStringRef account = CFStringCreateWithCString(
        kCFAllocatorDefault,
        "default-v1",
        kCFStringEncodingUTF8);
    if (service == NULL || account == NULL) {
        if (service != NULL) CFRelease(service);
        if (account != NULL) CFRelease(account);
        CFRelease(query);
        return NULL;
    }

    CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(query, kSecAttrService, service);
    CFDictionarySetValue(query, kSecAttrAccount, account);
    CFRelease(service);
    CFRelease(account);
    return query;
}

static OSStatus ghostftp_profile_key_copy(unsigned char **out, CFIndex *outLen) {
    if (out == NULL || outLen == NULL) return errSecParam;
    *out = NULL;
    *outLen = 0;

    CFMutableDictionaryRef query = ghostftp_profile_key_query();
    if (query == NULL) return errSecAllocate;
    CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
    CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);

    CFTypeRef result = NULL;
    OSStatus status = SecItemCopyMatching(query, &result);
    CFRelease(query);
    if (status != errSecSuccess) return status;
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
    CFRelease(result);
    *out = copy;
    *outLen = length;
    return errSecSuccess;
}

static OSStatus ghostftp_profile_key_add(const unsigned char *bytes, CFIndex length) {
    if (bytes == NULL || length <= 0) return errSecParam;
    CFMutableDictionaryRef query = ghostftp_profile_key_query();
    if (query == NULL) return errSecAllocate;
    CFDataRef data = CFDataCreate(kCFAllocatorDefault, bytes, length);
    if (data == NULL) {
        CFRelease(query);
        return errSecAllocate;
    }
    CFDictionarySetValue(query, kSecValueData, data);
    // The key is available only while the user session is unlocked and is not
    // synchronised to iCloud or another device.
    CFDictionarySetValue(query, kSecAttrAccessible, kSecAttrAccessibleWhenUnlockedThisDeviceOnly);
    OSStatus status = SecItemAdd(query, NULL);
    CFRelease(data);
    CFRelease(query);
    return status;
}

static int ghostftp_status_not_found(OSStatus status) { return status == errSecItemNotFound; }
static int ghostftp_status_duplicate(OSStatus status) { return status == errSecDuplicateItem; }
static int ghostftp_status_success(OSStatus status) { return status == errSecSuccess; }
static void ghostftp_profile_key_free(void *p) { free(p); }
*/
import "C"

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

const (
	darwinPersistentProfilePrefix = "darwin-keychain-aesgcm-v1:"
	darwinProfileMasterKeySize    = 32
	darwinPersistentMaxBytes      = 4 << 20
	persistentProfileSecretAAD    = "profile-secret-v1"
)

func keychainStatusError(action string, status C.OSStatus) error {
	return fmt.Errorf("macOS Keychain %s failed (status %d)", action, int32(status))
}

func copyDarwinProfileMasterKey() ([]byte, error) {
	var ptr *C.uchar
	var length C.CFIndex
	status := C.ghostftp_profile_key_copy(&ptr, &length)
	if C.ghostftp_status_success(status) == 0 {
		return nil, keychainStatusError("read", status)
	}
	if ptr == nil || int64(length) != darwinProfileMasterKeySize {
		if ptr != nil {
			C.ghostftp_profile_key_free(unsafe.Pointer(ptr))
		}
		return nil, errors.New("macOS Keychain profile key has an invalid size")
	}
	defer C.ghostftp_profile_key_free(unsafe.Pointer(ptr))
	key := C.GoBytes(unsafe.Pointer(ptr), C.int(length))
	if len(key) != darwinProfileMasterKeySize {
		WipeBytes(key)
		return nil, errors.New("macOS Keychain profile key could not be read completely")
	}
	return key, nil
}

func darwinProfileMasterKey() ([]byte, error) {
	key, err := copyDarwinProfileMasterKey()
	if err == nil {
		return key, nil
	}

	var probe *C.uchar
	var probeLength C.CFIndex
	status := C.ghostftp_profile_key_copy(&probe, &probeLength)
	if probe != nil {
		C.ghostftp_profile_key_free(unsafe.Pointer(probe))
	}
	if C.ghostftp_status_not_found(status) == 0 {
		return nil, err
	}

	candidate := make([]byte, darwinProfileMasterKeySize)
	if _, err := rand.Read(candidate); err != nil {
		return nil, err
	}
	status = C.ghostftp_profile_key_add((*C.uchar)(unsafe.Pointer(&candidate[0])), C.CFIndex(len(candidate)))
	if C.ghostftp_status_success(status) != 0 {
		return candidate, nil
	}
	WipeBytes(candidate)
	if C.ghostftp_status_duplicate(status) != 0 {
		return copyDarwinProfileMasterKey()
	}
	return nil, keychainStatusError("create", status)
}

func persistentProfileCipher(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// ProtectPersistentProfileBytes encrypts durable profile material with a random
// AES-GCM nonce. Only the 256-bit wrapping key is kept in the user Keychain;
// ciphertext remains in Ghost FTP's private application data directory.
func ProtectPersistentProfileBytes(data []byte, purpose string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	if len(data) > darwinPersistentMaxBytes || purpose == "" || len(purpose) > 128 {
		return "", errors.New("macOS persistent profile value is invalid")
	}
	key, err := darwinProfileMasterKey()
	if err != nil {
		return "", err
	}
	defer WipeBytes(key)
	aead, err := persistentProfileCipher(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nil, nonce, data, []byte(purpose))
	payload := make([]byte, 0, len(nonce)+len(sealed))
	payload = append(payload, nonce...)
	payload = append(payload, sealed...)
	defer WipeBytes(payload)
	return darwinPersistentProfilePrefix + base64.RawStdEncoding.EncodeToString(payload), nil
}

// UnprotectPersistentProfileBytes authenticates both ciphertext and its purpose,
// so a stored password blob cannot be replayed as the profile-file envelope.
func UnprotectPersistentProfileBytes(encoded, purpose string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	if !strings.HasPrefix(encoded, darwinPersistentProfilePrefix) || purpose == "" || len(purpose) > 128 {
		return nil, errors.New("macOS persistent profile value has an unsupported format")
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(encoded, darwinPersistentProfilePrefix))
	if err != nil {
		return nil, errors.New("macOS persistent profile value is malformed")
	}
	defer WipeBytes(payload)
	key, err := darwinProfileMasterKey()
	if err != nil {
		return nil, err
	}
	defer WipeBytes(key)
	aead, err := persistentProfileCipher(key)
	if err != nil {
		return nil, err
	}
	if len(payload) <= aead.NonceSize() {
		return nil, errors.New("macOS persistent profile value is truncated")
	}
	plain, err := aead.Open(nil, payload[:aead.NonceSize()], payload[aead.NonceSize():], []byte(purpose))
	if err != nil {
		return nil, errors.New("macOS persistent profile authentication failed")
	}
	return plain, nil
}

// PersistentProfileSecretToRuntime is the only bridge from durable profile
// storage into the short-lived process authentication channel. Plaintext is
// wiped immediately after ProtectRuntimeBytes creates the ephemeral handle.
func PersistentProfileSecretToRuntime(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	plain, err := UnprotectPersistentProfileBytes(encoded, persistentProfileSecretAAD)
	if err != nil {
		return "", err
	}
	defer WipeBytes(plain)
	return ProtectRuntimeBytes(plain)
}
