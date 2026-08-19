package remote

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"unicode/utf8"

	"brendigo.com/byftp/internal/itemlist"
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
)

const (
	cleanupTimeout       = 5 * time.Second
	maxRemoteDeleteDepth = 64
	maxRemoteDeleteItems = 10000
	maxDirectoryItems    = 50000
)

type deleteGuard struct {
	items int
}

func (g *deleteGuard) step(depth int) error {
	if depth > maxRemoteDeleteDepth {
		return errors.New("udaljena struktura je preduboka za sigurno brisanje")
	}
	g.items++
	if g.items > maxRemoteDeleteItems {
		return errors.New("udaljena struktura ima previše stavki za jedno brisanje")
	}
	return nil
}

type toolError struct {
	tool    string
	code    int
	message string
}

func (e *toolError) Error() string { return e.message }

func newToolError(tool string, runErr error, message string) error {
	code := -1
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		code = exitErr.ExitCode()
	}
	if strings.TrimSpace(message) == "" {
		message = runErr.Error()
	}
	return &toolError{tool: tool, code: code, message: message}
}

// IsRetryable is deliberately conservative. Automatic retry is useful only for
// transport interruptions; authentication, permission, path and host-key errors
// should fail immediately instead of repeating the same destructive or expensive
// operation several times.
func IsRetryable(err error) bool {
	if err == nil || errors.Is(err, ErrSkipped) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// A cleanup failure means the previous attempt may have left a staging or
	// rollback object on the server. Retrying automatically would create a new
	// random artifact while the previous remote state is still uncertain.
	if isRemoteResidualArtifactError(err) {
		return false
	}
	var te *toolError
	if errors.As(err, &te) && te.tool == "curl" {
		switch te.code {
		case 6, 7, 18, 28, 52, 55, 56:
			return true
		}
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"permission denied", "access denied", "authentication failed", "login denied", "login incorrect",
		"host key verification failed", "fingerprint", "no such file", "not found", "not a directory",
		"is a directory", "file exists", "already exists", "invalid argument", "unsupported", "not implemented",
	} {
		if strings.Contains(msg, marker) {
			return false
		}
	}
	for _, marker := range []string{
		"connection reset", "connection timed out", "operation timed out", "connection timeout",
		"failed to connect", "couldn't connect", "could not connect", "connection refused", "connection closed",
		"broken pipe", "network is unreachable", "no route to host", "temporary failure in name resolution",
		"could not resolve", "recv failure", "send failure", "partial file", "server returned nothing",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func sanitizedToolEnv(env []string) []string {
	blocked := map[string]struct{}{
		// Never inherit proxy routing from the parent process.
		"http_proxy": {}, "https_proxy": {}, "ftp_proxy": {}, "ftps_proxy": {}, "all_proxy": {}, "no_proxy": {},
		// Never inherit curl/TLS controls that could change trust, backend selection
		// or export session secrets outside ByFTP.
		"curl_ssl_backend": {}, "curl_ca_bundle": {}, "ssl_cert_file": {}, "ssl_cert_dir": {}, "sslkeylogfile": {},
		"curl_home": {}, "xdg_config_home": {}, "netrc": {},
		// Never inherit external SSH helpers/agents. ByFTP supplies its own
		// AskPass variables only for the specific child process that needs them.
		"ssh_askpass": {}, "ssh_askpass_require": {}, "ssh_auth_sock": {}, "ssh_sk_provider": {}, "display": {},
	}
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if _, skip := blocked[strings.ToLower(key)]; skip {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func writePrivateFileAtomic(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".byftp-private-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}
	if err = f.Chmod(0600); err != nil {
		cleanup()
		return err
	}
	if _, err = f.Write(data); err != nil {
		cleanup()
		return err
	}
	if err = f.Sync(); err != nil {
		cleanup()
		return err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = platform.ReplaceFile(tmp, filePath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func remoteJoin(base, name string) string {
	if base == "" {
		base = "/"
	}
	return path.Clean(path.Join(base, name))
}

func escapeURLPath(p string) string {
	parts := strings.Split(strings.ReplaceAll(p, "\\", "/"), "/")
	for i, v := range parts {
		if v != "" {
			parts[i] = url.PathEscape(v)
		}
	}
	out := strings.Join(parts, "/")
	if !strings.HasPrefix(out, "/") {
		out = "/" + out
	}
	return out
}

func parseListingSize(raw string) int64 {
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func parseListLine(line string) (model.Item, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "total ") {
		return model.Item{}, false
	}
	f := strings.Fields(line)
	if len(f) >= 9 && (strings.HasPrefix(f[0], "d") || strings.HasPrefix(f[0], "-") || strings.HasPrefix(f[0], "l")) {
		isDirectory := strings.HasPrefix(f[0], "d")
		isSymlink := strings.HasPrefix(f[0], "l")
		name := strings.Join(f[8:], " ")
		// Samo ls -l zapis simboličke poveznice koristi "ime -> odredište".
		// Obična datoteka smije legitimno sadržavati niz " -> " u nazivu.
		if isSymlink {
			if ix := strings.Index(name, " -> "); ix >= 0 {
				name = name[:ix]
			}
		}
		if name == "." || name == ".." {
			return model.Item{}, false
		}
		return model.Item{Name: name, Size: parseListingSize(f[4]), IsDirectory: isDirectory, IsSymlink: isSymlink, Modified: time.Time{}}, true
	}
	if len(f) >= 4 && strings.Contains(f[0], "-") {
		isDir := strings.EqualFold(f[2], "<DIR>")
		name := strings.Join(f[3:], " ")
		if name == "." || name == ".." {
			return model.Item{}, false
		}
		size := int64(0)
		if !isDir {
			size = parseListingSize(f[2])
		}
		return model.Item{Name: name, Size: size, IsDirectory: isDir}, true
	}
	return model.Item{Name: line}, true
}

func sortItems(items []model.Item) {
	itemlist.Sort(items)
}

func remoteEntry(items []model.Item, name string) (model.Item, bool) {
	for _, item := range items {
		if item.Name == name {
			return item, true
		}
	}
	return model.Item{}, false
}

func randomTransferToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func truncateUTF8Bytes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(value) <= max {
		return value
	}
	b := []byte(value[:max])
	for len(b) > 0 && !utf8.Valid(b) {
		b = b[:len(b)-1]
	}
	return string(b)
}

func remoteTransferNames(remotePath string) (dir, base, tempName, savedName string, err error) {
	dir, base = path.Split(path.Clean(remotePath))
	if dir == "" {
		dir = "."
	}
	dir = path.Clean(dir)
	token, err := randomTransferToken()
	if err != nil {
		return "", "", "", "", err
	}
	tempName = ".byftp-part-" + token
	savedName = ".byftp-rollback-" + token
	return
}

func validateDownloadedPart(part string) error {
	st, err := os.Lstat(part)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(part) {
		return errors.New("preuzeta privremena datoteka nije obična lokalna datoteka")
	}
	return nil
}

func localTransferSibling(local, kind string, preserveBase bool) (string, error) {
	dir := filepath.Dir(local)
	base := filepath.Base(local)
	token, err := randomTransferToken()
	if err != nil {
		return "", err
	}
	suffix := ".byftp-" + kind + "-" + token
	name := suffix
	if preserveBase {
		prefix := truncateUTF8Bytes(base, 240-len(suffix))
		if prefix == "" {
			prefix = "datoteka"
		}
		name = prefix + suffix
	}
	return filepath.Join(dir, name), nil
}

func validateChmod(mode string) error {
	if len(mode) != 3 && len(mode) != 4 {
		return errors.New("neispravan CHMOD")
	}
	for _, r := range mode {
		if r < '0' || r > '7' {
			return errors.New("neispravan CHMOD")
		}
	}
	return nil
}

func backupName(base, rollbackName string, keepBackup bool) string {
	if !keepBackup {
		return rollbackName
	}
	token := strings.TrimPrefix(rollbackName, ".byftp-rollback-")
	if token == rollbackName {
		token = strings.TrimPrefix(rollbackName, base+".byftp-rollback-")
	}
	suffix := ".byftp-backup-" + token
	prefix := truncateUTF8Bytes(base, 240-len(suffix))
	if prefix == "" {
		prefix = "datoteka"
	}
	return prefix + suffix
}

type recursiveDeleteOps struct {
	list       func(context.Context, string) ([]model.Item, error)
	removeFile func(context.Context, string) error
	removeDir  func(context.Context, string) error
}

func recursiveDelete(ctx context.Context, base, name string, isDir bool, depth int, guard *deleteGuard, ops recursiveDeleteOps) error {
	if err := security.ValidateRemoteName(name); err != nil {
		return err
	}
	if guard == nil {
		guard = &deleteGuard{}
	}
	if err := guard.step(depth); err != nil {
		return err
	}
	target := remoteJoin(base, name)
	if target == "/" || target == "." {
		return errors.New("brisanje remote root direktorija nije dopušteno")
	}
	if !isDir {
		return ops.removeFile(ctx, target)
	}
	items, err := ops.list(ctx, target)
	if err != nil {
		return err
	}
	for _, item := range items {
		childIsDir := item.IsDirectory && !item.IsSymlink
		if err := recursiveDelete(ctx, target, item.Name, childIsDir, depth+1, guard, ops); err != nil {
			return err
		}
	}
	return ops.removeDir(ctx, target)
}

type remoteCommitOps struct {
	rename func(context.Context, string, string, string) error
	delete func(context.Context, string, string, bool) error
}

func cleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cleanupTimeout)
}

func commitRemoteTemp(
	ctx context.Context,
	items []model.Item,
	dir, base, tempName, rollbackName string,
	keepBackup bool,
	ops remoteCommitOps,
) error {
	if ops.rename == nil || ops.delete == nil {
		return errors.New("remote commit operacije nisu dostupne")
	}

	existing, existed := remoteEntry(items, base)
	if existed && (existing.IsDirectory || existing.IsSymlink) {
		err := errors.New("odredište nije obična datoteka i neće biti prepisano")
		return cleanupFailure(err, dir, tempName, ops.delete)
	}
	savedName := backupName(base, rollbackName, keepBackup)
	if existed {
		if err := ops.rename(ctx, dir, base, savedName); err != nil {
			protectErr := fmt.Errorf("nije moguće zaštititi postojeću remote datoteku: %w", err)
			return cleanupFailure(protectErr, dir, tempName, ops.delete)
		}
	}

	if err := ops.rename(ctx, dir, tempName, base); err != nil {
		activationErr := fmt.Errorf("nije moguće aktivirati prenesenu datoteku: %w", err)
		if existed {
			restoreCtx, cancel := cleanupContext()
			restoreErr := ops.rename(restoreCtx, dir, savedName, base)
			cancel()
			if restoreErr != nil {
				activationErr = fmt.Errorf("aktivacija nove datoteke nije uspjela, a vraćanje izvorne datoteke iz sigurnosne kopije %s također nije uspjelo: %w", savedName, restoreErr)
			}
		}
		return cleanupFailure(activationErr, dir, tempName, ops.delete)
	}

	if existed && !keepBackup {
		return committedCleanupFailure(nil, dir, savedName, ops.delete)
	}
	return nil
}

const (
	maxCommandStdout = 32 << 20 // directory listings and SFTP command output
	maxCommandStderr = 512 << 10
)

type boundedOutput struct {
	buf      strings.Builder
	limit    int
	overflow bool
}

func newBoundedOutput(limit int) *boundedOutput {
	return &boundedOutput{limit: limit}
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		b.overflow = len(p) > 0
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		_, _ = b.buf.Write(p[:remaining])
	}
	if remaining < len(p) {
		b.overflow = true
	}
	return len(p), nil
}

func (b *boundedOutput) String() string { return b.buf.String() }
func (b *boundedOutput) Bytes() []byte  { return []byte(b.buf.String()) }
func (b *boundedOutput) Reset() {
	b.buf.Reset()
	b.overflow = false
}
func (b *boundedOutput) Err(label string) error {
	if !b.overflow {
		return nil
	}
	return fmt.Errorf("%s poslužitelja je prevelik", label)
}
