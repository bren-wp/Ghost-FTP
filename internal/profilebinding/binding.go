package profilebinding

import (
	"strconv"
	"strings"
)

func canonicalProtocol(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

func canonicalHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	host = strings.TrimSuffix(host, ".")
	return strings.ToLower(host)
}

// EndpointKey returns the canonical identity of one network endpoint. Keeping
// this representation in one package prevents profile binding, transfer retry
// and connection lifecycle code from drifting to slightly different host rules.
func EndpointKey(protocol, host string, port int) string {
	return canonicalProtocol(protocol) + "\x00" + canonicalHost(host) + "\x00" + strconv.Itoa(port)
}

// EndpointMatches određuje pripadaju li dvije konfiguracije istom mrežnom
// endpointu. Koristi se za SFTP host-key pin i druge server-scoped podatke.
func EndpointMatches(protocolA, hostA string, portA int, protocolB, hostB string, portB int) bool {
	return EndpointKey(protocolA, hostA, portA) == EndpointKey(protocolB, hostB, portB)
}

// AccountMatches je stroža granica za spremljene login vjerodajnice.
func AccountMatches(protocolA, hostA string, portA int, usernameA, protocolB, hostB string, portB int, usernameB string) bool {
	return EndpointMatches(protocolA, hostA, portA, protocolB, hostB, portB) && usernameA == usernameB
}

// PrivateKeyPathMatches compares user-selected private-key paths with native
// platform semantics. Empty paths match only each other; callers that require a
// key to exist must additionally reject an empty path.
func PrivateKeyPathMatches(keyA, keyB string) bool {
	keyA = strings.TrimSpace(keyA)
	keyB = strings.TrimSpace(keyB)
	if keyA == "" || keyB == "" {
		return keyA == keyB
	}
	return privateKeyPathEqual(keyA, keyB)
}

// PrivateKeyMatches dodatno veže passphrase uz konkretan lokalni privatni ključ.
// Semantika usporedbe putanje mora pratiti platformu: Windows putanje su
// case-insensitive, dok na ostalim platformama koristimo strogu usporedbu kako
// passphrase nikad ne bi bio ponovno upotrijebljen za drugi case-sensitive ključ.
func PrivateKeyMatches(protocolA, hostA string, portA int, usernameA, keyA, protocolB, hostB string, portB int, usernameB, keyB string) bool {
	keyA = strings.TrimSpace(keyA)
	keyB = strings.TrimSpace(keyB)
	return keyA != "" && keyB != "" &&
		AccountMatches(protocolA, hostA, portA, usernameA, protocolB, hostB, portB, usernameB) &&
		PrivateKeyPathMatches(keyA, keyB)
}
