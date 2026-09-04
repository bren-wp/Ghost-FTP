//go:build !windows

package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

const runtimeSecretLimit = 128

var runtimeValues = struct {
	sync.Mutex
	values map[string][]byte
}{values: make(map[string][]byte)}

func ProtectRuntimeString(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	valueBytes := []byte(value)
	runtimeValues.Lock()
	defer runtimeValues.Unlock()
	if _, exists := runtimeValues.values[token]; exists {
		WipeBytes(valueBytes)
		return "", errors.New("nije moguće izdvojiti privremenu vrijednost")
	}
	if len(runtimeValues.values) >= runtimeSecretLimit {
		WipeBytes(valueBytes)
		return "", errors.New("dosegnuto je ograničenje aktivnih privremenih tajni")
	}
	runtimeValues.values[token] = valueBytes
	return token, nil
}

func UnprotectRuntimeBytes(token string) ([]byte, error) {
	if token == "" {
		return nil, nil
	}
	runtimeValues.Lock()
	defer runtimeValues.Unlock()
	value, ok := runtimeValues.values[token]
	if !ok {
		return nil, errors.New("privremena vrijednost više nije dostupna")
	}
	return append([]byte(nil), value...), nil
}

func ForgetRuntimeSecret(token string) {
	if token == "" {
		return
	}
	runtimeValues.Lock()
	value := runtimeValues.values[token]
	delete(runtimeValues.values, token)
	runtimeValues.Unlock()
	WipeBytes(value)
}
