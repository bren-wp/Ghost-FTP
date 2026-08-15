package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizedToolEnvRemovesExternalNetworkControls(t *testing.T) {
	in := []string{
		"PATH=C:\\Windows",
		"HTTP_PROXY=http://proxy.invalid",
		"https_proxy=http://proxy.invalid",
		"FTP_PROXY=http://proxy.invalid",
		"ALL_PROXY=socks5://proxy.invalid",
		"NO_PROXY=localhost",
		"SSLKEYLOGFILE=C:\\temp\\tls.keys",
		"CURL_SSL_BACKEND=openssl",
		"CURL_CA_BUNDLE=C:\\temp\\ca.pem",
		"SSH_ASKPASS=C:\\evil.exe",
		"SSH_AUTH_SOCK=C:\\agent.sock",
		"SSH_SK_PROVIDER=C:\\untrusted-provider.dll",
		"DISPLAY=external",
		"KEEP=value",
	}
	got := sanitizedToolEnv(in)
	joined := strings.ToLower(strings.Join(got, "\n"))
	for _, forbidden := range []string{
		"http_proxy=", "https_proxy=", "ftp_proxy=", "all_proxy=", "no_proxy=",
		"sslkeylogfile=", "curl_ssl_backend=", "curl_ca_bundle=",
		"ssh_askpass=", "ssh_auth_sock=", "ssh_sk_provider=", "display=",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("proxy variable survived: %s in %q", forbidden, got)
		}
	}
	if !strings.Contains(strings.Join(got, "\n"), "KEEP=value") {
		t.Fatalf("unrelated environment variable was removed: %q", got)
	}
}

func TestCurlBaseConfigForcesDirectConnection(t *testing.T) {
	c := &CurlFTP{protocol: "ftp", username: "user"}
	cfg := string(c.configFor([]byte("secret"), nil))
	for _, required := range []string{`proxy = ""`, `noproxy = "*"`} {
		if !strings.Contains(cfg, required) {
			t.Fatalf("missing direct-connection guard %q in %q", required, cfg)
		}
	}
}

func TestSFTPCommandArgsHideConnectionMetadataAndDisableExternalRouting(t *testing.T) {
	dir := t.TempDir()
	known := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(known, []byte("host ssh-ed25519 key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(dir, "private key")
	if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	config, alias, err := createSSHSessionConfig(dir, "files.example.test", 22, "private-user", key, known, "ssh-ed25519", 15)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(config)
	s := &SFTP{host: "files.example.test", port: 22, knownHosts: known, sshConfig: config, sessionHost: alias}
	args := s.commandArgs()
	joined := strings.Join(args, "\n")
	for _, forbidden := range []string{"files.example.test", "private-user", key, known} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("SFTP command line leaks connection metadata %q in %#v", forbidden, args)
		}
	}
	for _, required := range []string{
		"-F\n" + config,
		"-oProxyCommand=none",
		"-oProxyJump=none",
		"-oIdentityAgent=none",
		"-oPKCS11Provider=none",
		"-oKnownHostsCommand=none",
		"-oPermitLocalCommand=no",
		"-oClearAllForwardings=yes",
		"-oForwardAgent=no",
		"-oForwardX11=no",
		"-oStrictHostKeyChecking=yes",
		"-oGlobalKnownHostsFile=none",
		"-oVerifyHostKeyDNS=no",
		"-oUpdateHostKeys=no",
		"-oIdentitiesOnly=yes",
		alias,
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing SFTP privacy/security guard %q in %#v", required, args)
		}
	}
}

func TestFindCurlPrefersWindowsSystemBinary(t *testing.T) {
	system32 := filepath.Join(t.TempDir(), "System32")
	if err := os.MkdirAll(system32, 0700); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(system32, "curl.exe")
	if err := os.WriteFile(want, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	oldSystemDirectory := systemDirectory
	systemDirectory = func() (string, error) { return system32, nil }
	t.Cleanup(func() { systemDirectory = oldSystemDirectory })
	got, err := findCurl()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("findCurl=%q want trusted system path %q", got, want)
	}
}

func TestFindOpenSSHPreferWindowsSystemBinary(t *testing.T) {
	system32 := filepath.Join(t.TempDir(), "System32")
	sshDir := filepath.Join(system32, "OpenSSH")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(sshDir, "sftp.exe")
	if err := os.WriteFile(want, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	oldSystemDirectory := systemDirectory
	systemDirectory = func() (string, error) { return system32, nil }
	t.Cleanup(func() { systemDirectory = oldSystemDirectory })
	got, err := findOpenSSH("sftp.exe")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("findOpenSSH=%q want trusted system path %q", got, want)
	}
}

func TestSSHSessionConfigIsPrivateDirectAndScoped(t *testing.T) {
	dir := t.TempDir()
	known := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(known, []byte("host ssh-ed25519 key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config, alias, err := createSSHSessionConfig(dir, "files.example.test", 2222, `domain\\user`, "", known, "ssh-ed25519", 15)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(config)
	if filepath.Dir(config) != dir || !strings.HasPrefix(filepath.Base(config), ".byftp-sftp-") {
		t.Fatalf("session config escaped managed directory: %q", config)
	}
	b, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, required := range []string{
		"Host " + alias,
		"HostName files.example.test",
		"Port 2222",
		"ProxyCommand none",
		"ProxyJump none",
		"GlobalKnownHostsFile none",
		"VerifyHostKeyDNS no",
		"UpdateHostKeys no",
		"IdentityAgent none",
		"PermitLocalCommand no",
		"ClearAllForwardings yes",
		"IdentityFile none",
		"HostKeyAlgorithms ssh-ed25519",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("managed SSH session config missing %q: %q", required, text)
		}
	}
}

func TestFTPSConfigDisablesExternalRevocationFetch(t *testing.T) {
	for _, protocol := range []string{"ftps", "ftpsi"} {
		c := &CurlFTP{protocol: protocol, username: "user"}
		cfg := string(c.configFor([]byte("secret"), nil))
		for _, required := range []string{"ssl-no-revoke", "tlsv1.2"} {
			if !strings.Contains(cfg, required) {
				t.Fatalf("%s config missing FTPS security option %q: %q", protocol, required, cfg)
			}
		}
	}
}

func TestSFTPWithoutExplicitKeyDisablesDefaultIdentities(t *testing.T) {
	dir := t.TempDir()
	known := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(known, []byte("host ssh-ed25519 key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config, _, err := createSSHSessionConfig(dir, "files.example.test", 22, "user", "", known, "ssh-ed25519", 15)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(config)
	b, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	joined := string(b)
	for _, required := range []string{"IdentitiesOnly yes", "IdentityFile none"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("password-only SFTP may use implicit SSH identities; missing %q in %q", required, joined)
		}
	}
}

func TestConfiguredConnectionTimeoutReachesNetworkTools(t *testing.T) {
	c := &CurlFTP{protocol: "ftp", username: "user", connectTimeout: 27}
	cfg := string(c.configFor([]byte("secret"), nil))
	if !strings.Contains(cfg, "connect-timeout = 27") {
		t.Fatalf("curl config does not contain configured timeout: %q", cfg)
	}

	dir := t.TempDir()
	known := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(known, []byte("host ssh-ed25519 key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config, _, err := createSSHSessionConfig(dir, "files.example.test", 22, "user", "", known, "ssh-ed25519", 27)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(config)
	b, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "ConnectTimeout 27") {
		t.Fatalf("OpenSSH config does not contain configured timeout: %q", string(b))
	}
}
