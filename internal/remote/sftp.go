package remote

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/security"
)

type SFTP struct {
	host                               string
	port                               int
	passwordBlob, passphraseBlob       string
	knownHosts, sshConfig, sessionHost string
	exePath, sftp                      string
}

func findOpenSSH(name string) (string, error) {
	if systemDir, err := systemDirectory(); err == nil && systemDir != "" {
		p := filepath.Join(systemDir, "OpenSSH", name)
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p, nil
		}
	}
	if runtime.GOOS == "windows" {
		return "", errors.New("SFTP podrška nije dostupna u sustavu Windows")
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	return "", errors.New("SFTP komponenta nije pronađena")
}
func writePrivateTempFile(dir, pattern string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(name)
	}
	if err := f.Chmod(0600); err != nil {
		cleanup()
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		cleanup()
		return "", err
	}
	if err := f.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func cleanupStaleSFTPArtifacts(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	prefixes := []string{".byftp-sftp-", ".byftp-known-", ".byftp-scan-host-", "byftp-key-", "askpass-"}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		managed := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(name, prefix) {
				managed = true
				break
			}
		}
		if managed {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	// 2.8 and older used one persistent generic SSH config. It contains no
	// credential, but it is obsolete once every session receives its own config.
	_ = os.Remove(filepath.Join(dir, "ssh-client.conf"))
}

func scanKeyAlgorithm(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return ""
	}
	alg := fields[1]
	switch {
	case alg == "ssh-ed25519":
		return alg
	case strings.HasPrefix(alg, "ecdsa-sha2-"):
		return alg
	case alg == "ssh-rsa":
		return alg
	default:
		return ""
	}
}

func ScanFingerprint(ctx context.Context, host string, port int, tempDir string) (string, string, string, error) {
	if err := security.ValidateHost(host); err != nil {
		return "", "", "", err
	}
	if port < 1 || port > 65535 {
		return "", "", "", errors.New("neispravan SFTP port")
	}
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return "", "", "", err
	}
	scan, err := findOpenSSH("ssh-keyscan.exe")
	if err != nil {
		return "", "", "", err
	}
	keygen, err := findOpenSSH("ssh-keygen.exe")
	if err != nil {
		return "", "", "", err
	}

	// Keep the user-entered host out of both the process command line and disk.
	// OpenSSH ssh-keyscan accepts -f - to read targets directly from stdin.
	scanCtx, scanCancel := context.WithTimeout(ctx, 12*time.Second)
	defer scanCancel()
	cmd := exec.CommandContext(scanCtx, scan, "-T", "8", "-t", "ed25519,ecdsa,rsa", "-p", strconv.Itoa(port), "-f", "-")
	configureToolCommand(cmd)
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdin = strings.NewReader(host + "\n")
	cmd.Dir = filepath.Dir(scan)
	cmd.Env = sanitizedToolEnv(os.Environ())
	out := newBoundedOutput(1 << 20)
	er := newBoundedOutput(maxCommandStderr)
	cmd.Stdout = out
	cmd.Stderr = er
	if err := cmd.Run(); err != nil {
		if overflowErr := er.Err("odgovor"); overflowErr != nil {
			return "", "", "", overflowErr
		}
		return "", "", "", fmt.Errorf("nije moguće dohvatiti SFTP host ključ: %s", strings.TrimSpace(er.String()))
	}
	if err := out.Err("odgovor"); err != nil {
		return "", "", "", err
	}
	lines := []string{}
	for _, l := range strings.Split(strings.ReplaceAll(out.String(), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(l) != "" && !strings.HasPrefix(strings.TrimSpace(l), "#") {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return "", "", "", errors.New("poslužitelj nije vratio SSH host ključ")
	}

	// Bind trust to one concrete host-key algorithm. Earlier versions displayed
	// the fingerprint of the preferred key but wrote all scanned keys to
	// known_hosts, allowing the real session to negotiate a different scanned key.
	selected := lines[0]
	rank := func(line string) int {
		switch alg := scanKeyAlgorithm(line); {
		case alg == "ssh-ed25519":
			return 3
		case strings.HasPrefix(alg, "ecdsa-sha2-"):
			return 2
		case alg == "ssh-rsa":
			return 1
		default:
			return 0
		}
	}
	for _, line := range lines[1:] {
		if rank(line) > rank(selected) {
			selected = line
		}
	}
	algorithm := scanKeyAlgorithm(selected)
	if algorithm == "" {
		return "", "", "", errors.New("poslužitelj je vratio nepodržan SSH host ključ")
	}

	name, err := writePrivateTempFile(tempDir, "byftp-key-*.known_hosts", []byte(selected+"\n"))
	if err != nil {
		return "", "", "", err
	}
	defer os.Remove(name)
	keygenCtx, keygenCancel := context.WithTimeout(ctx, 8*time.Second)
	defer keygenCancel()
	cmd = exec.CommandContext(keygenCtx, keygen, "-lf", name, "-E", "sha256")
	configureToolCommand(cmd)
	cmd.WaitDelay = 5 * time.Second
	cmd.Dir = filepath.Dir(keygen)
	cmd.Env = sanitizedToolEnv(os.Environ())
	out.Reset()
	er.Reset()
	cmd.Stdout = out
	cmd.Stderr = er
	if err := cmd.Run(); err != nil {
		if overflowErr := er.Err("odgovor"); overflowErr != nil {
			return "", "", "", overflowErr
		}
		return "", "", "", fmt.Errorf("nije moguće izračunati fingerprint: %s", strings.TrimSpace(er.String()))
	}
	if err := out.Err("odgovor"); err != nil {
		return "", "", "", err
	}
	fields := strings.Fields(out.String())
	if len(fields) < 2 || !strings.HasPrefix(fields[1], "SHA256:") {
		return "", "", "", errors.New("nepoznat format SHA-256 fingerprinta")
	}
	return fields[1], selected + "\n", algorithm, nil
}

func randomSFTPAlias() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "byftp-" + hex.EncodeToString(buf), nil
}

func sshConfigQuote(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

func validatePrivateKeyPath(keyPath string) error {
	if keyPath == "" {
		return nil
	}
	if len(keyPath) > 32767 || !utf8.ValidString(keyPath) || strings.ContainsAny(keyPath, "\x00\r\n") {
		return errors.New("putanja privatnog ključa nije ispravna")
	}
	st, err := os.Lstat(keyPath)
	if err != nil {
		return errors.New("privatni ključ nije dostupan")
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(keyPath) {
		return errors.New("privatni ključ mora biti obična lokalna datoteka bez preusmjeravanja")
	}
	return nil
}

func createSSHSessionConfig(dir, host string, port int, username, keyPath, knownHosts, hostKeyAlgorithm string, connectTimeout int) (string, string, error) {
	if connectTimeout < 5 || connectTimeout > 60 {
		connectTimeout = 15
	}
	if err := validatePrivateKeyPath(keyPath); err != nil {
		return "", "", err
	}
	alias, err := randomSFTPAlias()
	if err != nil {
		return "", "", err
	}
	if hostKeyAlgorithm != "" && scanKeyAlgorithm("host "+hostKeyAlgorithm+" key") == "" {
		return "", "", errors.New("nepodržan algoritam sigurnosnog ključa poslužitelja")
	}
	identity := "none"
	if keyPath != "" {
		identity = sshConfigQuote(keyPath)
	}
	lines := []string{
		"Host " + alias,
		"  HostName " + host,
		"  Port " + strconv.Itoa(port),
		"  User " + sshConfigQuote(username),
		"  UserKnownHostsFile " + sshConfigQuote(knownHosts),
		"  GlobalKnownHostsFile none",
		"  StrictHostKeyChecking yes",
		"  VerifyHostKeyDNS no",
		"  UpdateHostKeys no",
		"  BatchMode no",
		"  NumberOfPasswordPrompts 1",
		"  ConnectTimeout " + strconv.Itoa(connectTimeout),
		"  ServerAliveInterval 20",
		"  ServerAliveCountMax 2",
		"  ProxyCommand none",
		"  ProxyJump none",
		"  IdentityAgent none",
		"  PKCS11Provider none",
		"  KnownHostsCommand none",
		"  PermitLocalCommand no",
		"  ClearAllForwardings yes",
		"  ForwardAgent no",
		"  ForwardX11 no",
		"  IdentitiesOnly yes",
		"  IdentityFile " + identity,
		"  CanonicalizeHostname no",
	}
	if hostKeyAlgorithm != "" {
		lines = append(lines, "  HostKeyAlgorithms "+hostKeyAlgorithm)
	}
	configPath, err := writePrivateTempFile(dir, ".byftp-sftp-*.conf", []byte(strings.Join(lines, "\n")+"\n"))
	if err != nil {
		return "", "", err
	}
	return configPath, alias, nil
}

func NewSFTP(host string, port int, username, password, keyPath, passphrase, knownHosts, exePath string) (*SFTP, error) {
	return newSFTPWithProtectedSecrets(host, port, username, password, "", keyPath, passphrase, "", knownHosts, "", exePath, 15)
}

func newSFTPWithProtectedSecrets(host string, port int, username, password, passwordBlob, keyPath, passphrase, passphraseBlob, knownHosts, hostKeyAlgorithm, exePath string, connectTimeout int) (*SFTP, error) {
	if err := security.ValidateConnection("sftp", host, username, port); err != nil {
		return nil, err
	}
	if err := security.ValidateSecret(password); err != nil {
		return nil, err
	}
	if err := security.ValidateSecret(passphrase); err != nil {
		return nil, err
	}
	if err := validatePrivateKeyPath(keyPath); err != nil {
		return nil, err
	}
	sftp, err := findOpenSSH("sftp.exe")
	if err != nil {
		return nil, err
	}
	if password != "" {
		passwordBlob, err = security.ProtectString(password)
		if err != nil {
			return nil, err
		}
	}
	if passphrase != "" {
		passphraseBlob, err = security.ProtectString(passphrase)
		if err != nil {
			return nil, err
		}
	}
	sshConfig, sessionHost, err := createSSHSessionConfig(filepath.Dir(knownHosts), host, port, username, keyPath, knownHosts, hostKeyAlgorithm, connectTimeout)
	if err != nil {
		return nil, err
	}
	return &SFTP{
		host: host, port: port, passwordBlob: passwordBlob, passphraseBlob: passphraseBlob,
		knownHosts: knownHosts, sshConfig: sshConfig, sessionHost: sessionHost, exePath: exePath, sftp: sftp,
	}, nil
}

func (s *SFTP) Protocol() string { return "sftp" }
func (s *SFTP) Host() string     { return s.host }
func (s *SFTP) Port() int        { return s.port }
func (s *SFTP) Close() error {
	var errs []error
	for _, file := range []string{s.knownHosts, s.sshConfig} {
		if file == "" {
			continue
		}
		if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	// Drop active references to connection metadata/protected credentials once
	// the session is closed. The DPAPI blobs are not plaintext, but there is no
	// reason to retain them in the live session object after disconnect.
	s.passwordBlob = ""
	s.passphraseBlob = ""
	s.host = ""
	s.knownHosts = ""
	s.sshConfig = ""
	s.sessionHost = ""
	s.exePath = ""
	return errors.Join(errs...)
}

func sftpQuote(v string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v) + `"`
}

func (s *SFTP) askpassEnvironment() ([]string, error) {
	if s.passwordBlob == "" && s.passphraseBlob == "" {
		return sanitizedToolEnv(os.Environ()), nil
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)
	env := sanitizedToolEnv(os.Environ())
	env = append(env,
		"SSH_ASKPASS="+s.exePath,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=ByFTP",
		"BYFTP_ASKPASS_TOKEN="+token,
		"BYFTP_PASSWORD_BLOB="+s.passwordBlob,
		"BYFTP_PASSPHRASE_BLOB="+s.passphraseBlob,
	)
	return env, nil
}

func (s *SFTP) commandArgs() []string {
	// The user-entered host, username, key path and known_hosts path live only in
	// the private short-lived config. Process command lines expose only a random
	// local alias plus generic hardening switches.
	return []string{
		"-q", "-oBatchMode=no", "-b", "-", "-F", s.sshConfig,
		"-oProxyCommand=none", "-oProxyJump=none", "-oIdentityAgent=none",
		"-oPKCS11Provider=none", "-oKnownHostsCommand=none", "-oPermitLocalCommand=no",
		"-oClearAllForwardings=yes", "-oForwardAgent=no", "-oForwardX11=no",
		"-oStrictHostKeyChecking=yes", "-oGlobalKnownHostsFile=none", "-oVerifyHostKeyDNS=no", "-oUpdateHostKeys=no",
		"-oIdentitiesOnly=yes", "-oCanonicalizeHostname=no",
		s.sessionHost,
	}
}

func buildSFTPCommandStream(commands []string) (string, error) {
	var b strings.Builder
	for _, command := range commands {
		if strings.ContainsAny(command, "\x00\r\n") {
			return "", errors.New("neispravna SFTP naredba")
		}
		b.WriteString(command)
		b.WriteByte('\n')
	}
	b.WriteString("quit\n")
	return b.String(), nil
}

func (s *SFTP) run(ctx context.Context, commands ...string) (string, error) {
	input, err := buildSFTPCommandStream(commands)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, s.sftp, s.commandArgs()...)
	configureToolCommand(cmd)
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdin = strings.NewReader(input)
	cmd.Dir = filepath.Dir(s.sftp)
	cmd.Env, err = s.askpassEnvironment()
	if err != nil {
		return "", err
	}
	out := newBoundedOutput(maxCommandStdout)
	er := newBoundedOutput(maxCommandStderr)
	cmd.Stdout = out
	cmd.Stderr = er
	if err := cmd.Run(); err != nil {
		if overflowErr := er.Err("dijagnostički odgovor"); overflowErr != nil {
			return "", overflowErr
		}
		msg := strings.TrimSpace(er.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", newToolError("sftp", err, msg)
	}
	if err := out.Err("odgovor"); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (s *SFTP) List(ctx context.Context, p string) ([]model.Item, error) {
	if err := security.ValidateRemotePath(p); err != nil {
		return nil, err
	}
	out, err := s.run(ctx, "ls -la "+sftpQuote(p))
	if err != nil {
		return nil, err
	}
	var items []model.Item
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "sftp>") {
			continue
		}
		if item, ok := parseListLine(line); ok {
			items = append(items, item)
			if len(items) > maxDirectoryItems {
				return nil, errors.New("mapa sadrži previše stavki za siguran prikaz")
			}
		}
	}
	sortItems(items)
	return items, nil
}
func (s *SFTP) Mkdir(ctx context.Context, base, name string) error {
	if err := security.ValidateRemoteName(name); err != nil {
		return err
	}
	_, err := s.run(ctx, "mkdir "+sftpQuote(remoteJoin(base, name)))
	return err
}
func (s *SFTP) Rename(ctx context.Context, base, oldName, newName string) error {
	if err := security.ValidateRemoteName(oldName); err != nil {
		return err
	}
	if err := security.ValidateRemoteName(newName); err != nil {
		return err
	}
	_, err := s.run(ctx, "rename "+sftpQuote(remoteJoin(base, oldName))+" "+sftpQuote(remoteJoin(base, newName)))
	return err
}
func (s *SFTP) Delete(ctx context.Context, base, name string, isDir bool) error {
	ops := recursiveDeleteOps{
		list: s.List,
		removeFile: func(ctx context.Context, target string) error {
			_, err := s.run(ctx, "rm "+sftpQuote(target))
			return err
		},
		removeDir: func(ctx context.Context, target string) error {
			_, err := s.run(ctx, "rmdir "+sftpQuote(target))
			return err
		},
	}
	return recursiveDelete(ctx, base, name, isDir, 0, &deleteGuard{}, ops)
}

func (s *SFTP) Chmod(ctx context.Context, base, name, mode string) error {
	if err := security.ValidateRemoteName(name); err != nil {
		return err
	}
	if err := validateChmod(mode); err != nil {
		return err
	}
	_, err := s.run(ctx, "chmod "+mode+" "+sftpQuote(remoteJoin(base, name)))
	return err
}
func (s *SFTP) Upload(ctx context.Context, local, remotePath string, options TransferOptions) error {
	if err := security.ValidateRemotePath(remotePath); err != nil {
		return err
	}
	if st, err := os.Lstat(local); err != nil {
		return err
	} else if st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(local) {
		return errors.New("simboličke poveznice i junctione nije dopušteno prenositi")
	} else if st.IsDir() {
		return errors.New("upload očekuje datoteku")
	}

	dir, base, tempName, savedName, err := remoteTransferNames(remotePath)
	if err != nil {
		return err
	}
	items, err := s.List(ctx, dir)
	if err != nil {
		return fmt.Errorf("nije moguće provjeriti odredišni direktorij: %w", err)
	}
	if existing, ok := remoteEntry(items, base); ok {
		if existing.IsDirectory || existing.IsSymlink {
			return errors.New("odredište nije obična datoteka i neće biti prepisano")
		}
		if options.SkipExisting {
			return ErrSkipped
		}
	}
	tempPath := remoteJoin(dir, tempName)
	if _, err = s.run(ctx, "put "+sftpQuote(local)+" "+sftpQuote(tempPath)); err != nil {
		cleanupCtx, cancel := cleanupContext()
		_ = s.Delete(cleanupCtx, dir, tempName, false)
		cancel()
		return err
	}
	return commitRemoteTemp(ctx, items, dir, base, tempName, savedName, options.KeepBackup, remoteCommitOps{
		rename: s.Rename,
		delete: s.Delete,
	})
}
func (s *SFTP) Download(ctx context.Context, remotePath, local string, options TransferOptions) error {
	if err := security.ValidateRemotePath(remotePath); err != nil {
		return err
	}
	if options.SkipExisting {
		if st, err := os.Lstat(local); err == nil {
			if st.IsDir() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(local) {
				return errors.New("ciljna putanja nije obična datoteka")
			}
			return ErrSkipped
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
		return err
	}
	part, err := localTransferSibling(local, "part", false)
	if err != nil {
		return err
	}
	if _, err := s.run(ctx, "get "+sftpQuote(remotePath)+" "+sftpQuote(part)); err != nil {
		_ = os.Remove(part)
		return err
	}
	if err := validateDownloadedPart(part); err != nil {
		_ = os.Remove(part)
		return err
	}
	return replaceLocalFileAtomic(local, part, options.KeepBackup)
}
