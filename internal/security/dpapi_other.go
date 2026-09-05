//go:build linux

package security

import (
	"crypto/rand"
	"crypto/subtle"
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
	linuxSecretPrefix   = "linux-secret-v1:"
	linuxSecretMaxBytes = 64 << 10
	linuxSecretTTL      = 2 * time.Hour
	linuxSecretTokenLen = 32
)

type linuxSecretEntry struct {
	value   []byte
	expires time.Time
}

type linuxSecretBrokerState struct {
	mu       sync.Mutex
	listener *net.UnixListener
	dir      string
	socket   string
	entries  map[string]*linuxSecretEntry
}

var linuxSecretBroker linuxSecretBrokerState

func purgeExpiredLinuxSecretsLocked(now time.Time) {
	for token, entry := range linuxSecretBroker.entries {
		if entry == nil || now.After(entry.expires) {
			if entry != nil {
				WipeBytes(entry.value)
			}
			delete(linuxSecretBroker.entries, token)
		}
	}
}

func ensureLinuxSecretBroker() (string, error) {
	linuxSecretBroker.mu.Lock()
	defer linuxSecretBroker.mu.Unlock()
	if linuxSecretBroker.listener != nil {
		return linuxSecretBroker.socket, nil
	}

	dir, err := os.MkdirTemp("", "ghostftp-secret-")
	if err != nil {
		return "", err
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		cleanup()
		return "", err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		cleanup()
		return "", errors.New("Linux secret broker directory is not private")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || int(stat.Uid) != os.Geteuid() {
		cleanup()
		return "", errors.New("Linux secret broker directory has an invalid owner")
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
	linuxSecretBroker.listener = listener
	linuxSecretBroker.dir = dir
	linuxSecretBroker.socket = socket
	linuxSecretBroker.entries = make(map[string]*linuxSecretEntry)
	go serveLinuxSecrets(listener)
	return socket, nil
}

func unixPeerUID(conn *net.UnixConn) (int, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return -1, err
	}
	uid := -1
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err != nil {
			controlErr = err
			return
		}
		uid = int(cred.Uid)
	}); err != nil {
		return -1, err
	}
	if controlErr != nil {
		return -1, controlErr
	}
	return uid, nil
}

func serveLinuxSecrets(listener *net.UnixListener) {
	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			return
		}
		go handleLinuxSecretRequest(conn)
	}
}

func handleLinuxSecretRequest(conn *net.UnixConn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(4 * time.Second))
	uid, err := unixPeerUID(conn)
	if err != nil || uid != os.Geteuid() {
		return
	}
	var tokenRaw [linuxSecretTokenLen]byte
	if _, err := io.ReadFull(conn, tokenRaw[:]); err != nil {
		return
	}
	token := hex.EncodeToString(tokenRaw[:])

	linuxSecretBroker.mu.Lock()
	purgeExpiredLinuxSecretsLocked(time.Now())
	entry := linuxSecretBroker.entries[token]
	var secret []byte
	if entry != nil {
		decoded, decodeErr := hex.DecodeString(token)
		stored, storedErr := hex.DecodeString(token)
		if decodeErr == nil && storedErr == nil && subtle.ConstantTimeCompare(decoded, stored) == 1 {
			secret = append([]byte(nil), entry.value...)
			entry.expires = time.Now().Add(linuxSecretTTL)
		}
	}
	linuxSecretBroker.mu.Unlock()
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

func newLinuxSecretToken() ([]byte, string, error) {
	raw := make([]byte, linuxSecretTokenLen)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}
	return raw, hex.EncodeToString(raw), nil
}

func ProtectBytes(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	if len(data) > linuxSecretMaxBytes {
		return "", errors.New("Linux runtime secret is too large")
	}
	socket, err := ensureLinuxSecretBroker()
	if err != nil {
		return "", err
	}
	rawToken, token, err := newLinuxSecretToken()
	if err != nil {
		return "", err
	}
	defer WipeBytes(rawToken)

	linuxSecretBroker.mu.Lock()
	purgeExpiredLinuxSecretsLocked(time.Now())
	if len(linuxSecretBroker.entries) >= runtimeSecretCapacity {
		linuxSecretBroker.mu.Unlock()
		return "", errors.New("Linux runtime secret capacity reached")
	}
	linuxSecretBroker.entries[token] = &linuxSecretEntry{
		value:   append([]byte(nil), data...),
		expires: time.Now().Add(linuxSecretTTL),
	}
	linuxSecretBroker.mu.Unlock()

	encodedSocket := base64.RawURLEncoding.EncodeToString([]byte(socket))
	return linuxSecretPrefix + encodedSocket + "." + token, nil
}

func ProtectString(value string) (string, error) {
	return ProtectBytes([]byte(value))
}

func parseLinuxSecretBlob(encoded string) (string, []byte, error) {
	if !strings.HasPrefix(encoded, linuxSecretPrefix) {
		return "", nil, errors.New("persistent DPAPI credentials are unavailable on Linux")
	}
	parts := strings.SplitN(strings.TrimPrefix(encoded, linuxSecretPrefix), ".", 2)
	if len(parts) != 2 || len(parts[1]) != linuxSecretTokenLen*2 {
		return "", nil, errors.New("Linux runtime secret token is malformed")
	}
	socketRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(socketRaw) == 0 || len(socketRaw) > 4096 {
		return "", nil, errors.New("Linux runtime secret socket is malformed")
	}
	socket := string(socketRaw)
	if !filepath.IsAbs(socket) || strings.ContainsAny(socket, "\x00\r\n") {
		return "", nil, errors.New("Linux runtime secret socket is invalid")
	}
	token, err := hex.DecodeString(parts[1])
	if err != nil || len(token) != linuxSecretTokenLen {
		return "", nil, errors.New("Linux runtime secret token is invalid")
	}
	return socket, token, nil
}

func UnprotectBytes(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	socket, token, err := parseLinuxSecretBlob(encoded)
	if err != nil {
		return nil, err
	}
	defer WipeBytes(token)
	conn, err := net.DialTimeout("unix", socket, 3*time.Second)
	if err != nil {
		return nil, errors.New("Linux runtime secret broker is unavailable")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(4 * time.Second))
	if _, err := conn.Write(token); err != nil {
		return nil, err
	}
	var header [4]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return nil, errors.New("Linux runtime secret broker rejected the request")
	}
	n := int(binary.BigEndian.Uint32(header[:]))
	if n < 1 || n > linuxSecretMaxBytes {
		return nil, errors.New("Linux runtime secret broker returned an invalid length")
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

func WipeBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
