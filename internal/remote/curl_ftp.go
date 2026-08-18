package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
)

type CurlFTP struct {
	protocol       string
	host           string
	username       string
	passwordBlob   string
	port           int
	connectTimeout int
	curl           string
	mlsdState      atomic.Int32
}

func NewCurlFTP(protocol, host string, port int, username, password string) (*CurlFTP, error) {
	return newCurlFTPWithProtectedSecret(protocol, host, port, username, password, "", 15)
}

func newCurlFTPWithProtectedSecret(protocol, host string, port int, username, password, passwordBlob string, connectTimeout int) (*CurlFTP, error) {
	if err := security.ValidateConnection(protocol, host, username, port); err != nil {
		return nil, err
	}
	if err := security.ValidateSecret(password); err != nil {
		return nil, err
	}
	if connectTimeout < 5 || connectTimeout > 60 {
		connectTimeout = 15
	}
	p, err := findCurl()
	if err != nil {
		return nil, err
	}
	if password != "" {
		passwordBlob, err = security.ProtectRuntimeString(password)
		if err != nil {
			return nil, err
		}
	}
	return &CurlFTP{protocol: protocol, host: host, port: port, username: username, passwordBlob: passwordBlob, connectTimeout: connectTimeout, curl: p}, nil
}

func (c *CurlFTP) Protocol() string { return c.protocol }
func (c *CurlFTP) Host() string     { return c.host }
func (c *CurlFTP) Port() int        { return c.port }
func (c *CurlFTP) Close() error {
	security.ForgetRuntimeSecret(c.passwordBlob)
	c.passwordBlob = ""
	c.username = ""
	c.host = ""
	return nil
}

func cfgQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func appendCfgQuotedBytes(dst []byte, value []byte) []byte {
	dst = append(dst, '"')
	for _, b := range value {
		if b == '\\' || b == '"' {
			dst = append(dst, '\\')
		}
		dst = append(dst, b)
	}
	return append(dst, '"')
}

func appendConfigLine(dst []byte, line string) []byte {
	dst = append(dst, line...)
	return append(dst, '\n')
}

func (c *CurlFTP) baseURL(p string) string {
	scheme := "ftp"
	if c.protocol == "ftpsi" {
		scheme = "ftps"
	}
	hostport := net.JoinHostPort(strings.Trim(c.host, "[]"), strconv.Itoa(c.port))
	return fmt.Sprintf("%s://%s%s", scheme, hostport, escapeURLPath(p))
}

func (c *CurlFTP) configFor(password []byte, lines []string) []byte {
	cfg := make([]byte, 0, 512+len(password))
	for _, line := range []string{
		"silent",
		"proxy = \"\"",
		"noproxy = \"*\"",
		"show-error",
		"fail",
		"connect-timeout = " + strconv.Itoa(c.connectTimeout),
		"ftp-skip-pasv-ip",
	} {
		cfg = appendConfigLine(cfg, line)
	}
	cfg = append(cfg, "user = "...)
	credential := make([]byte, 0, len(c.username)+1+len(password))
	credential = append(credential, c.username...)
	credential = append(credential, ':')
	credential = append(credential, password...)
	cfg = appendCfgQuotedBytes(cfg, credential)
	security.WipeBytes(credential)
	cfg = append(cfg, '\n')
	if c.protocol == "ftps" {
		cfg = appendConfigLine(cfg, "ssl-reqd")
		cfg = appendConfigLine(cfg, "tlsv1.2")
		cfg = appendConfigLine(cfg, "ssl-no-revoke")
	} else if c.protocol == "ftpsi" {
		cfg = appendConfigLine(cfg, "tlsv1.2")
		cfg = appendConfigLine(cfg, "ssl-no-revoke")
	}
	for _, line := range lines {
		cfg = appendConfigLine(cfg, line)
	}
	return cfg
}

func (c *CurlFTP) run(ctx context.Context, lines []string) ([]byte, error) {
	password, err := security.UnprotectRuntimeBytes(c.passwordBlob)
	if err != nil {
		return nil, errors.New("vjerodajnicu aktivne veze nije moguće otključati")
	}
	defer security.WipeBytes(password)
	cfg := c.configFor(password, lines)
	defer security.WipeBytes(cfg)
	cmd := exec.CommandContext(ctx, c.curl, "-q", "--config", "-")
	configureToolCommand(cmd)
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdin = bytes.NewReader(cfg)
	cmd.Dir = filepath.Dir(c.curl)
	cmd.Env = sanitizedToolEnv(os.Environ())
	out := newBoundedOutput(maxCommandStdout)
	er := newBoundedOutput(maxCommandStderr)
	cmd.Stdout = out
	cmd.Stderr = er
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if overflowErr := er.Err("dijagnostički odgovor"); overflowErr != nil {
			return nil, overflowErr
		}
		msg := strings.TrimSpace(er.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, newToolError("curl", err, msg)
	}
	if err := out.Err("odgovor"); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func parseMLSDLine(line string) (model.Item, bool) {
	line = strings.TrimRight(line, "\r\n")
	sep := strings.IndexByte(line, ' ')
	if sep <= 0 || sep+1 >= len(line) {
		return model.Item{}, false
	}
	factsPart := line[:sep]
	name := line[sep+1:]
	if name == "" || name == "." || name == ".." {
		return model.Item{}, false
	}
	facts := map[string]string{}
	for _, fact := range strings.Split(factsPart, ";") {
		if fact == "" {
			continue
		}
		kv := strings.SplitN(fact, "=", 2)
		if len(kv) != 2 {
			continue
		}
		facts[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
	}
	typeValue, ok := facts["type"]
	if !ok {
		return model.Item{}, false
	}
	typeLower := strings.ToLower(typeValue)
	if typeLower == "cdir" || typeLower == "pdir" {
		return model.Item{}, false
	}
	item := model.Item{Name: name}
	item.IsDirectory = typeLower == "dir"
	item.IsSymlink = strings.Contains(typeLower, "slink") || strings.Contains(typeLower, "symlink")
	if raw := facts["size"]; raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n >= 0 {
			item.Size = n
		}
	}
	if raw := facts["modify"]; len(raw) >= 14 {
		if tm, err := time.Parse("20060102150405", raw[:14]); err == nil {
			item.Modified = tm.UTC()
		}
	}
	return item, true
}

func parseMLSD(data []byte) ([]model.Item, bool, error) {
	items := make([]model.Item, 0, 128)
	recognized := false
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, ";") && strings.Contains(strings.ToLower(line), "type=") {
			recognized = true
		}
		if item, ok := parseMLSDLine(line); ok {
			items = append(items, item)
			if len(items) > maxDirectoryItems {
				return nil, true, errors.New("mapa sadrži previše stavki za siguran prikaz")
			}
		}
	}
	sortItems(items)
	return items, recognized, nil
}

func ftpCommandUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"500 ", "500-", "502 ", "502-", "504 ", "504-",
		"unknown command", "command not understood", "not implemented", "unsupported command",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func (c *CurlFTP) List(ctx context.Context, p string) ([]model.Item, error) {
	if err := security.ValidateRemotePath(p); err != nil {
		return nil, err
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	urlLine := "url = " + cfgQuote(c.baseURL(p))
	if c.mlsdState.Load() != -1 {
		out, err := c.run(ctx, []string{urlLine, `request = "MLSD"`})
		if err == nil {
			items, recognized, parseErr := parseMLSD(out)
			if parseErr != nil {
				return nil, parseErr
			}
			if recognized || len(strings.TrimSpace(string(out))) == 0 {
				c.mlsdState.Store(1)
				return items, nil
			}
		} else if ftpCommandUnsupported(err) {
			c.mlsdState.Store(-1)
		}
	}
	out, err := c.run(ctx, []string{urlLine})
	if err != nil {
		return nil, err
	}
	var items []model.Item
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
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

func (c *CurlFTP) quote(ctx context.Context, cmds ...string) error {
	lines := []string{"url = " + cfgQuote(c.baseURL("/"))}
	for _, q := range cmds {
		if strings.ContainsAny(q, "\x00\r\n") {
			return errors.New("neispravna FTP naredba")
		}
		lines = append(lines, "quote = "+cfgQuote(q))
	}
	_, err := c.run(ctx, lines)
	return err
}

func (c *CurlFTP) Mkdir(ctx context.Context, base, name string) error {
	if err := security.ValidateRemoteName(name); err != nil {
		return err
	}
	return c.quote(ctx, "MKD "+remoteJoin(base, name))
}

func (c *CurlFTP) Rename(ctx context.Context, base, oldName, newName string) error {
	if err := security.ValidateRemoteName(oldName); err != nil {
		return err
	}
	if err := security.ValidateRemoteName(newName); err != nil {
		return err
	}
	return c.quote(ctx, "RNFR "+remoteJoin(base, oldName), "RNTO "+remoteJoin(base, newName))
}

func (c *CurlFTP) Delete(ctx context.Context, base, name string, isDir bool) error {
	ops := recursiveDeleteOps{
		list: c.List,
		removeFile: func(ctx context.Context, target string) error {
			return c.quote(ctx, "DELE "+target)
		},
		removeDir: func(ctx context.Context, target string) error {
			return c.quote(ctx, "RMD "+target)
		},
	}
	return recursiveDelete(ctx, base, name, isDir, 0, &deleteGuard{}, ops)
}

func (c *CurlFTP) Chmod(ctx context.Context, base, name, mode string) error {
	if err := security.ValidateRemoteName(name); err != nil {
		return err
	}
	if err := validateChmod(mode); err != nil {
		return err
	}
	return c.quote(ctx, "SITE CHMOD "+mode+" "+remoteJoin(base, name))
}

func (c *CurlFTP) Upload(ctx context.Context, local, remotePath string, options TransferOptions) error {
	if err := security.ValidateRemotePath(remotePath); err != nil {
		return err
	}
	st, err := os.Lstat(local)
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(local) {
		return errors.New("simboličke poveznice i junctione nije dopušteno prenositi")
	}
	if st.IsDir() {
		return errors.New("upload očekuje datoteku")
	}
	dir, base, tempName, savedName, err := remoteTransferNames(remotePath)
	if err != nil {
		return err
	}
	items, err := c.List(ctx, dir)
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
	lines := []string{"url = " + cfgQuote(c.baseURL(tempPath)), "upload-file = " + cfgQuote(local), "speed-time = 30", "speed-limit = 1"}
	if _, err = c.run(ctx, lines); err != nil {
		cleanupCtx, cancel := cleanupContext()
		_ = c.Delete(cleanupCtx, dir, tempName, false)
		cancel()
		return err
	}
	return commitRemoteTemp(ctx, items, dir, base, tempName, savedName, options.KeepBackup, remoteCommitOps{rename: c.Rename, delete: c.Delete})
}

func (c *CurlFTP) Download(ctx context.Context, remotePath, local string, options TransferOptions) error {
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
	lines := []string{"url = " + cfgQuote(c.baseURL(remotePath)), "output = " + cfgQuote(part), "speed-time = 30", "speed-limit = 1"}
	if _, err := c.run(ctx, lines); err != nil {
		_ = os.Remove(part)
		return err
	}
	if err := validateDownloadedPart(part); err != nil {
		_ = os.Remove(part)
		return err
	}
	return replaceLocalFileAtomic(local, part, options.KeepBackup)
}

func replaceLocalFileAtomic(local, part string, keepBackup bool) error {
	if st, err := os.Lstat(local); err == nil {
		if st.IsDir() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(local) {
			_ = os.Remove(part)
			return errors.New("ciljna putanja nije obična datoteka")
		}
		kind := "rollback"
		preserveBase := false
		if keepBackup {
			kind = "backup"
			preserveBase = true
		}
		bak, err := localTransferSibling(local, kind, preserveBase)
		if err != nil {
			_ = os.Remove(part)
			return err
		}
		if err = platform.RenameNoReplace(local, bak); err != nil {
			_ = os.Remove(part)
			return err
		}
		if err = platform.RenameNoReplace(part, local); err != nil {
			restoreErr := platform.RenameNoReplace(bak, local)
			_ = os.Remove(part)
			if restoreErr != nil {
				return fmt.Errorf("nova datoteka nije aktivirana, a izvorna nije mogla biti vraćena; sigurnosna kopija ostala je na %s: %w", bak, restoreErr)
			}
			return err
		}
		if !keepBackup {
			_ = os.Remove(bak)
		}
		return nil
	} else if !os.IsNotExist(err) {
		_ = os.Remove(part)
		return err
	}
	return platform.RenameNoReplace(part, local)
}
