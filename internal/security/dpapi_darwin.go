//go:build darwin

package security

/*
#include <sys/types.h>
#include <unistd.h>

static int ghostftp_getpeereid(int fd, unsigned int *uid) {
	uid_t euid;
	gid_t egid;
	if (getpeereid(fd, &euid, &egid) != 0) {
		return -1;
	}
	*uid = (unsigned int)euid;
	return 0;
}
*/
import "C"

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	darwinSecretPrefix   = "darwin-secret-v1:"
	darwinSecretMaxBytes = 64 << 10
	darwinSecretTTL      = 2 * time.Hour
	darwinSecretTokenLen = 32
	darwinSecretLimit    = 256
)

type darwinSecretEntry struct {
	value   []byte
	expires time.Time
}

type darwinSecretBrokerState struct {
	mu       sync.Mutex
	listener *net.UnixListener
	dir      string
	socket   string
	entries  map[string]*darwinSecretEntry
}

var darwinSecretBroker darwinSecretBrokerState

func PersistentSecretStorageAvailable() bool { return true }

func purgeExpiredDarwinSecretsLocked(now time.Time) {
	for token, entry := range darwinSecretBroker.entries {
		if entry == nil || now.After(entry.expires) {
			if entry != nil {
				WipeBytes(entry.value)
			}
			delete(darwinSecretBroker.entries, token)
		}
	}
}

func ensureDarwinSecretBroker() (string, error) {
	darwinSecretBroker.mu.Lock()
	defer darwinSecretBroker.mu.Unlock()
	if darwinSecretBroker.listener != nil {
		return darwinSecretBroker.socket, nil
	}
	dir, err := os.MkdirTemp("", "ghostftp-secret-")
	if err != nil {
		return "", err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := os.Chmod(dir, 0o700); err != nil {
		cleanup()
		return "", err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		cleanup()
		return "", errors.New("macOS secret broker directory is not private")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		cleanup()
		return "", errors.New("macOS secret broker directory has an invalid owner")
	}
	socket := filepath.Join(dir, "askpass.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		cleanup()
		return "", err
	}
	if err := os.Chmod(socket, 0o600); err != nil {
		_ = listener.Close()
		cleanup()
		return "", err
	}
	darwinSecretBroker.listener = listener
	darwinSecretBroker.dir = dir
	darwinSecretBroker.socket = socket
	darwinSecretBroker.entries = make(map[string]*darwinSecretEntry)
	go serveDarwinSecrets(listener)
	return socket, nil
}

func darwinPeerUID(conn *net.UnixConn) (int, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return -1, err
	}
	uid := -1
	var peerErr error
	if err := raw.Control(func(fd uintptr) {
		var euid C.uint
		if C.ghostftp_getpeereid(C.int(fd), &euid) != 0 {
			peerErr = errors.New("macOS peer credential lookup failed")
			return
		}
		uid = int(euid)
	}); err != nil {
		return -1, err
	}
	if peerErr != nil {
		return -1, peerErr
	}
	return uid, nil
}

func serveDarwinSecrets(listener *net.UnixListener) {
	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			return
		}
		go handleDarwinSecretRequest(conn)
	}
}

func handleDarwinSecretRequest(conn *net.UnixConn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(4 * time.Second))
	uid, err := darwinPeerUID(conn)
	if err != nil || uid != os.Geteuid() {
		return
	}
	var rawToken [darwinSecretTokenLen]byte
	if _, err := io.ReadFull(conn, rawToken[:]); err != nil {
		return
	}
	token := hex.EncodeToString(rawToken[:])
	darwinSecretBroker.mu.Lock()
	purgeExpiredDarwinSecretsLocked(time.Now())
	entry := darwinSecretBroker.entries[token]
	var secret []byte
	if entry != nil {
		secret = append([]byte(nil), entry.value...)
		entry.expires = time.Now().Add(darwinSecretTTL)
	}
	darwinSecretBroker.mu.Unlock()
	if len(secret) == 0 {
		return
	}
	defer WipeBytes(secret)
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(secret)))
	if _, err := conn.Write(header[:]); err != nil {
		return
	}
	_, _ = conn.Write(secret)
}

func ProtectBytes(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	if len(data) > darwinSecretMaxBytes {
		return "", errors.New("macOS runtime secret is too large")
	}
	socket, err := ensureDarwinSecretBroker()
	if err != nil {
		return "", err
	}
	raw := make([]byte, darwinSecretTokenLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	defer WipeBytes(raw)
	token := hex.EncodeToString(raw)
	darwinSecretBroker.mu.Lock()
	defer darwinSecretBroker.mu.Unlock()
	purgeExpiredDarwinSecretsLocked(time.Now())
	if len(darwinSecretBroker.entries) >= darwinSecretLimit {
		return "", errors.New("macOS runtime secret capacity reached")
	}
	darwinSecretBroker.entries[token] = &darwinSecretEntry{value: append([]byte(nil), data...), expires: time.Now().Add(darwinSecretTTL)}
	encodedSocket := base64.RawURLEncoding.EncodeToString([]byte(socket))
	return darwinSecretPrefix + encodedSocket + "." + token, nil
}

func ProtectString(value string) (string, error) { return ProtectBytes([]byte(value)) }

func parseDarwinSecretBlob(encoded string) (string, []byte, string, error) {
	if !strings.HasPrefix(encoded, darwinSecretPrefix) {
		return "", nil, "", errors.New("persistent protected credentials are unavailable on macOS")
	}
	parts := strings.SplitN(strings.TrimPrefix(encoded, darwinSecretPrefix), ".", 2)
	if len(parts) != 2 || len(parts[1]) != darwinSecretTokenLen*2 {
		return "", nil, "", errors.New("macOS runtime secret token is malformed")
	}
	socketRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(socketRaw) == 0 || len(socketRaw) > 4096 {
		return "", nil, "", errors.New("macOS runtime secret socket is malformed")
	}
	socket := string(socketRaw)
	if !filepath.IsAbs(socket) || strings.ContainsAny(socket, "\x00\r\n") {
		return "", nil, "", errors.New("macOS runtime secret socket is invalid")
	}
	token, err := hex.DecodeString(parts[1])
	if err != nil || len(token) != darwinSecretTokenLen {
		return "", nil, "", errors.New("macOS runtime secret token is invalid")
	}
	return socket, token, parts[1], nil
}

func UnprotectBytes(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	socket, token, _, err := parseDarwinSecretBlob(encoded)
	if err != nil {
		return nil, err
	}
	defer WipeBytes(token)
	conn, err := net.DialTimeout("unix", socket, 3*time.Second)
	if err != nil {
		return nil, errors.New("macOS runtime secret broker is unavailable")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(4 * time.Second))
	if _, err := conn.Write(token); err != nil {
		return nil, err
	}
	var header [4]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return nil, errors.New("macOS runtime secret broker rejected the request")
	}
	n := int(binary.BigEndian.Uint32(header[:]))
	if n < 1 || n > darwinSecretMaxBytes {
		return nil, errors.New("macOS runtime secret broker returned an invalid length")
	}
	secret := make([]byte, n)
	if _, err := io.ReadFull(conn, secret); err != nil {
		WipeBytes(secret)
		return nil, err
	}
	return secret, nil
}

func UnprotectString(encoded string) (string, error) {
	secret, err := UnprotectBytes(encoded)
	if err != nil {
		return "", err
	}
	defer WipeBytes(secret)
	return string(secret), nil
}

func ForgetProtectedSecret(encoded string) {
	if encoded == "" {
		return
	}
	socket, token, tokenHex, err := parseDarwinSecretBlob(encoded)
	if err != nil {
		return
	}
	WipeBytes(token)
	darwinSecretBroker.mu.Lock()
	defer darwinSecretBroker.mu.Unlock()
	if socket != darwinSecretBroker.socket {
		return
	}
	entry := darwinSecretBroker.entries[tokenHex]
	delete(darwinSecretBroker.entries, tokenHex)
	if entry != nil {
		WipeBytes(entry.value)
	}
}

func WipeBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
