package sshclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"brendigo.com/byftp/internal/security"
)

type Config struct {
	Host       string
	Port       int
	Username   string
	PrivateKey string
}

func findSSH() (string, error) {
	if runtime.GOOS == "windows" {
		if root := os.Getenv("WINDIR"); root != "" {
			p := filepath.Join(root, "System32", "OpenSSH", "ssh.exe")
			if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
				return p, nil
			}
		}
	}
	for _, name := range []string{"ssh", "ssh.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", errors.New("OpenSSH klijent nije pronađen")
}

func configValue(value string) (string, error) {
	if value == "" || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("SSH konfiguracija sadrži nedopuštenu vrijednost")
	}
	return `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`, nil
}

func ensureKnownHosts(path string) error {
	st, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		f, createErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if createErr != nil {
			return createErr
		}
		return f.Close()
	}
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(path) {
		return errors.New("SSH known_hosts mora biti obična lokalna datoteka bez preusmjeravanja")
	}
	return nil
}

func validatePrivateKey(path string) error {
	if path == "" {
		return nil
	}
	st, err := os.Lstat(path)
	if err != nil {
		return errors.New("privatni SSH ključ nije dostupan")
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(path) {
		return errors.New("privatni SSH ključ mora biti obična lokalna datoteka bez preusmjeravanja")
	}
	return nil
}

func writeConfig(dir string, cfg Config) (string, error) {
	if err := security.ValidateConnection("sftp", cfg.Host, cfg.Username, cfg.Port); err != nil {
		return "", err
	}
	if err := validatePrivateKey(cfg.PrivateKey); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	knownPath := filepath.Join(dir, "known_hosts")
	if err := ensureKnownHosts(knownPath); err != nil {
		return "", err
	}
	host, _ := configValue(strings.Trim(cfg.Host, "[]"))
	user, _ := configValue(cfg.Username)
	known, err := configValue(knownPath)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("Host byftp-session\n")
	b.WriteString("  HostName " + host + "\n")
	b.WriteString("  Port " + strconv.Itoa(cfg.Port) + "\n")
	b.WriteString("  User " + user + "\n")
	b.WriteString("  UserKnownHostsFile " + known + "\n")
	b.WriteString("  GlobalKnownHostsFile none\n")
	b.WriteString("  StrictHostKeyChecking ask\n")
	b.WriteString("  VerifyHostKeyDNS no\n")
	b.WriteString("  UpdateHostKeys no\n")
	b.WriteString("  BatchMode no\n")
	b.WriteString("  NumberOfPasswordPrompts 3\n")
	b.WriteString("  PasswordAuthentication yes\n")
	b.WriteString("  KbdInteractiveAuthentication yes\n")
	b.WriteString("  PubkeyAuthentication yes\n")
	b.WriteString("  HostbasedAuthentication no\n")
	b.WriteString("  GSSAPIAuthentication no\n")
	b.WriteString("  IdentityAgent none\n")
	b.WriteString("  AddKeysToAgent no\n")
	b.WriteString("  PKCS11Provider none\n")
	b.WriteString("  ProxyCommand none\n")
	b.WriteString("  ProxyJump none\n")
	b.WriteString("  KnownHostsCommand none\n")
	b.WriteString("  CanonicalizeHostname no\n")
	b.WriteString("  PermitLocalCommand no\n")
	b.WriteString("  LocalCommand none\n")
	b.WriteString("  ClearAllForwardings yes\n")
	b.WriteString("  ForwardAgent no\n")
	b.WriteString("  ForwardX11 no\n")
	b.WriteString("  RequestTTY force\n")
	b.WriteString("  ServerAliveInterval 30\n")
	b.WriteString("  ServerAliveCountMax 3\n")
	b.WriteString("  ConnectTimeout 20\n")
	b.WriteString("  IdentitiesOnly yes\n")
	if cfg.PrivateKey != "" {
		key, err := configValue(cfg.PrivateKey)
		if err != nil {
			return "", err
		}
		b.WriteString("  IdentityFile " + key + "\n")
	} else {
		b.WriteString("  IdentityFile none\n")
	}
	f, err := os.CreateTemp(dir, ".byftp-ssh-*.conf")
	if err != nil {
		return "", err
	}
	name := f.Name()
	cleanup := func() { _ = f.Close(); _ = os.Remove(name) }
	if err := f.Chmod(0600); err != nil { cleanup(); return "", err }
	if _, err := io.WriteString(f, b.String()); err != nil { cleanup(); return "", err }
	if err := f.Sync(); err != nil { cleanup(); return "", err }
	if err := f.Close(); err != nil { _ = os.Remove(name); return "", err }
	return name, nil
}

func prompt(reader *bufio.Reader, label, fallback string) (string, error) {
	if fallback != "" { fmt.Printf("%s [%s]: ", label, fallback) } else { fmt.Printf("%s: ", label) }
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 { return "", err }
	line = strings.TrimSpace(line)
	if line == "" { return fallback, nil }
	return line, nil
}

func Run(ctx context.Context, dataDir, version string) error {
	ssh, err := findSSH()
	if err != nil { return err }
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("ByFTP SSH Client %s\n", version)
	fmt.Println("Sigurni SSH terminal. Lozinku, MFA i passphrase unosi izravno OpenSSH.")
	host, err := prompt(reader, "Poslužitelj", ""); if err != nil { return err }
	portText, err := prompt(reader, "Port", "22"); if err != nil { return err }
	port, err := strconv.Atoi(portText); if err != nil || port < 1 || port > 65535 { return errors.New("port mora biti između 1 i 65535") }
	user, err := prompt(reader, "Korisničko ime", ""); if err != nil { return err }
	key, err := prompt(reader, "Privatni ključ (opcionalno)", ""); if err != nil { return err }
	cfgPath, err := writeConfig(dataDir, Config{Host: host, Port: port, Username: user, PrivateKey: strings.TrimSpace(key)})
	if err != nil { return err }
	defer os.Remove(cfgPath)
	cmd := exec.CommandContext(ctx, ssh, "-F", cfgPath, "byftp-session")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}
