package remote

// hostKeyConstraintForScannedKey converts the key type returned by ssh-keyscan
// into the optional HostKeyAlgorithms constraint used for the actual SFTP
// session. An RSA public key is represented in known_hosts as "ssh-rsa", but
// modern OpenSSH negotiates RSA host-key signatures with rsa-sha2-512/256.
// Passing "ssh-rsa" as HostKeyAlgorithms would instead force the legacy
// RSA/SHA-1 signature algorithm. For RSA we therefore keep the pinned RSA key
// in known_hosts but leave signature negotiation to OpenSSH's modern defaults.
func hostKeyConstraintForScannedKey(scannedKeyType string) string {
	if scannedKeyType == "ssh-rsa" {
		return ""
	}
	return scannedKeyType
}
