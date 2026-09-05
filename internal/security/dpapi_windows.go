//go:build windows

package security

import (
	"encoding/base64"
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var crypt32 = syscall.NewLazyDLL("crypt32.dll")
var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var cryptProtectData = crypt32.NewProc("CryptProtectData")
var cryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
var localFree = kernel32.NewProc("LocalFree")

func PersistentSecretStorageAvailable() bool { return true }

// Windows DPAPI ciphertext is persistent by design. There is no process-owned
// broker entry to forget when a session closes.
func ForgetProtectedSecret(string) {}

func blob(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
}

func wipeBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
	runtime.KeepAlive(data)
}

func bytesFromBlob(b dataBlob) []byte {
	if b.cbData == 0 || b.pbData == nil {
		return nil
	}
	return append([]byte(nil), unsafe.Slice(b.pbData, int(b.cbData))...)
}

func clearAndFreeBlob(b dataBlob) {
	if b.pbData == nil {
		return
	}
	if b.cbData != 0 {
		wipeBytes(unsafe.Slice(b.pbData, int(b.cbData)))
	}
	localFree.Call(uintptr(unsafe.Pointer(b.pbData)))
}

// WipeBytes clears a mutable buffer as soon as a transient credential is no
// longer needed. It cannot erase immutable Go strings, therefore secret paths
// prefer byte slices when possible.
func WipeBytes(data []byte) { wipeBytes(data) }

func UnprotectBytes(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	defer wipeBytes(raw)
	in := blob(raw)
	var out dataBlob
	r, _, e := cryptUnprotectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0x1, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, e
	}
	defer clearAndFreeBlob(out)
	if out.pbData == nil {
		return nil, errors.New("DPAPI nije vratio podatke")
	}
	return bytesFromBlob(out), nil
}

func ProtectBytes(value []byte) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	raw := append([]byte(nil), value...)
	defer wipeBytes(raw)
	in := blob(raw)
	var out dataBlob
	r, _, e := cryptProtectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0x1, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return "", e
	}
	defer clearAndFreeBlob(out)
	protected := bytesFromBlob(out)
	defer wipeBytes(protected)
	return base64.StdEncoding.EncodeToString(protected), nil
}

func ProtectString(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	raw := []byte(value)
	defer wipeBytes(raw)
	return ProtectBytes(raw)
}

func UnprotectString(encoded string) (string, error) {
	plain, err := UnprotectBytes(encoded)
	if err != nil {
		return "", err
	}
	defer wipeBytes(plain)
	return string(plain), nil
}
