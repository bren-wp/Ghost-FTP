//go:build linux

package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	linuxProfileKeyFile = ".profiles.key"
	linuxProfilePrefix  = "linux-aesgcm-v1:"
	linuxProfileKeySize = 32
)

var linuxProfileAAD = []byte("Ghost FTP secure profiles v1")

func validateLinuxProfileKeyInfo(info os.FileInfo) error {
	if info == nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Linux profile key is not a safe regular file")
	}
	if info.Size() != linuxProfileKeySize {
		return errors.New("Linux profile key has an invalid size")
	}
	if info.Mode().Perm()&0077 != 0 {
		return errors.New("Linux profile key permissions are too broad")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errors.New("Linux profile key is not owned by the current user")
	}
	return nil
}

func readLinuxProfileKey(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if err := validateLinuxProfileKeyInfo(before); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if err := validateLinuxProfileKeyInfo(opened); err != nil || !os.SameFile(before, opened) {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("Linux profile key changed during secure open")
	}
	key := make([]byte, linuxProfileKeySize)
	if _, err := io.ReadFull(f, key); err != nil {
		for i := range key {
			key[i] = 0
		}
		return nil, errors.New("Linux profile key could not be read completely")
	}
	var extra [1]byte
	if n, err := f.Read(extra[:]); n != 0 || !errors.Is(err, io.EOF) {
		for i := range key {
			key[i] = 0
		}
		return nil, errors.New("Linux profile key contains trailing data")
	}
	after, err := f.Stat()
	if err != nil || !os.SameFile(opened, after) || after.Size() != linuxProfileKeySize {
		for i := range key {
			key[i] = 0
		}
		return nil, errors.New("Linux profile key changed while being read")
	}
	return key, nil
}

func createLinuxProfileKey(dir, path string) ([]byte, error) {
	key := make([]byte, linuxProfileKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		for i := range key {
			key[i] = 0
		}
		if errors.Is(err, os.ErrExist) {
			return readLinuxProfileKey(path)
		}
		return nil, err
	}
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(path)
	}
	if err := f.Chmod(0600); err != nil {
		cleanup()
		for i := range key {
			key[i] = 0
		}
		return nil, err
	}
	n, err := f.Write(key)
	if err != nil || n != len(key) {
		cleanup()
		for i := range key {
			key[i] = 0
		}
		if err != nil {
			return nil, err
		}
		return nil, io.ErrShortWrite
	}
	if err := f.Sync(); err != nil {
		cleanup()
		for i := range key {
			key[i] = 0
		}
		return nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		for i := range key {
			key[i] = 0
		}
		return nil, err
	}
	if err := syncStateDirectory(dir); err != nil {
		_ = os.Remove(path)
		for i := range key {
			key[i] = 0
		}
		return nil, err
	}
	return key, nil
}

func linuxProfileKey(dir string) ([]byte, error) {
	if strings.TrimSpace(dir) == "" || !filepath.IsAbs(dir) {
		return nil, errors.New("Linux profile storage directory is invalid")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("Linux profile storage directory is not private")
	}
	path := filepath.Join(dir, linuxProfileKeyFile)
	key, err := readLinuxProfileKey(path)
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return createLinuxProfileKey(dir, path)
}

func protectProfileData(data []byte, dir string) (string, error) {
	key, err := linuxProfileKey(dir)
	if err != nil {
		return "", err
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, data, linuxProfileAAD)
	payload := make([]byte, 0, len(nonce)+len(sealed))
	payload = append(payload, nonce...)
	payload = append(payload, sealed...)
	return linuxProfilePrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func unprotectProfileData(encoded, dir string) ([]byte, error) {
	if !strings.HasPrefix(encoded, linuxProfilePrefix) {
		// One-way migration support for the old non-secret Base64 compatibility
		// envelope. Profiles.loadSecure immediately rewrites this plaintext codec
		// into AES-GCM after a successful parse.
		return base64.StdEncoding.DecodeString(encoded)
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, linuxProfilePrefix))
	if err != nil {
		return nil, errors.New("Linux secure profile envelope is malformed")
	}
	key, err := linuxProfileKey(dir)
	if err != nil {
		return nil, err
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(payload) <= gcm.NonceSize() {
		return nil, errors.New("Linux secure profile envelope is truncated")
	}
	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, linuxProfileAAD)
	if err != nil {
		return nil, errors.New("Linux secure profile authentication failed")
	}
	return plain, nil
}

func profileDataNeedsMigration(encoded string) bool {
	return encoded != "" && !strings.HasPrefix(encoded, linuxProfilePrefix)
}
