package profilebinding

import "strings"

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	host = strings.TrimSuffix(host, ".")
	return strings.ToLower(host)
}

// EndpointMatches određuje pripadaju li dvije konfiguracije istom mrežnom
// endpointu. Koristi se za SFTP host-key pin i druge server-scoped podatke.
func EndpointMatches(protocolA, hostA string, portA int, protocolB, hostB string, portB int) bool {
	return strings.EqualFold(strings.TrimSpace(protocolA), strings.TrimSpace(protocolB)) &&
		normalizeHost(hostA) == normalizeHost(hostB) &&
		portA == portB
}

// AccountMatches je stroža granica za spremljene login vjerodajnice.
func AccountMatches(protocolA, hostA string, portA int, usernameA, protocolB, hostB string, portB int, usernameB string) bool {
	return EndpointMatches(protocolA, hostA, portA, protocolB, hostB, portB) && usernameA == usernameB
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
		privateKeyPathEqual(keyA, keyB)
}
