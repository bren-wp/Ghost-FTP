package security

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	safeName      = regexp.MustCompile(`^[^\\/:*?"<>|\x00-\x1f]+$`)
	hostnameLabel = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
)

func ValidateName(name string) error {
	if name == "" || name != strings.TrimSpace(name) || name == "." || name == ".." || len(name) > 255 || !utf8.ValidString(name) || !safeName.MatchString(name) {
		return errors.New("neispravan naziv datoteke ili mape")
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return errors.New("neispravan naziv datoteke ili mape")
	}
	upper := strings.ToUpper(name)
	stem := upper
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	reserved := stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || stem == "CLOCK$"
	if !reserved && len(stem) == 4 {
		prefix, digit := stem[:3], stem[3]
		reserved = (prefix == "COM" || prefix == "LPT") && digit >= '1' && digit <= '9'
	}
	if reserved {
		return errors.New("naziv je rezerviran u sustavu Windows")
	}
	return nil
}

// ValidateRemoteName validates one server-side path component. Remote servers
// may support characters that Windows local filenames do not, so this is
// deliberately less restrictive than ValidateName while still blocking path
// traversal and command-stream separators.
func ValidateRemoteName(name string) error {
	if name == "" || name == "." || name == ".." || len(name) > 1024 || !utf8.ValidString(name) || strings.ContainsAny(name, "\x00\r\n/\\") {
		return errors.New("neispravan naziv stavke na poslužitelju")
	}
	return nil
}

func ValidateRemotePath(p string) error {
	if p == "" {
		return nil
	}
	if len(p) > 4096 || !utf8.ValidString(p) || strings.ContainsAny(p, "\x00\r\n") {
		return errors.New("udaljena putanja sadrži nedopuštene znakove")
	}
	for _, part := range strings.Split(strings.ReplaceAll(p, "\\", "/"), "/") {
		if part == ".." {
			return errors.New("udaljena putanja ne smije sadržavati '..'")
		}
	}
	return nil
}

func ValidateSecret(secret string) error {
	if len(secret) > 8192 || !utf8.ValidString(secret) || strings.ContainsAny(secret, "\x00\r\n") {
		return errors.New("lozinka ili passphrase sadrži nedopuštene znakove")
	}
	return nil
}

func ValidateHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" || len(host) > 253 || !utf8.ValidString(host) || strings.ContainsAny(host, "\x00\r\n\t /\\@") {
		return errors.New("neispravan poslužitelj")
	}

	// Uglate zagrade dopuštene su samo kao točan par oko IPv6 adrese.
	// strings.Trim(host, "[]") nije siguran za validaciju jer bi prihvatio
	// nepotpune ili višestruke zagrade poput "[2001:db8::1" i "[[...]]".
	if strings.ContainsAny(host, "[]") {
		if len(host) < 3 || host[0] != '[' || host[len(host)-1] != ']' || strings.Count(host, "[") != 1 || strings.Count(host, "]") != 1 {
			return errors.New("neispravan poslužitelj")
		}
		inner := host[1 : len(host)-1]
		if !strings.Contains(inner, ":") || net.ParseIP(inner) == nil {
			return errors.New("neispravan poslužitelj")
		}
		return nil
	}

	// Dvotočka izvan zagrada dopuštena je samo u sirovoj IPv6 adresi.
	if strings.Contains(host, ":") {
		if net.ParseIP(host) == nil || !strings.Contains(host, ":") {
			return errors.New("neispravan poslužitelj")
		}
		return nil
	}
	if net.ParseIP(host) != nil {
		return nil
	}
	if strings.HasSuffix(host, ".") {
		host = strings.TrimSuffix(host, ".")
	}
	labels := strings.Split(host, ".")
	if len(labels) == 0 {
		return errors.New("neispravan poslužitelj")
	}
	for _, label := range labels {
		if !hostnameLabel.MatchString(label) {
			return errors.New("neispravan poslužitelj")
		}
	}
	return nil
}

func ValidateConnection(protocol, host, username string, port int) error {
	switch protocol {
	case "ftp", "ftps", "ftpsi", "sftp":
	default:
		return errors.New("nepodržan protokol")
	}
	if err := ValidateHost(host); err != nil {
		return err
	}
	if username == "" || len(username) > 1024 || !utf8.ValidString(username) || strings.ContainsAny(username, "\x00\r\n") {
		return errors.New("neispravno korisničko ime")
	}
	if port < 1 || port > 65535 {
		return errors.New("port mora biti između 1 i 65535")
	}
	return nil
}

func SafeLocalChild(base, name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	childAbs, err := filepath.Abs(filepath.Join(baseAbs, name))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseAbs, childAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("putanja izlazi iz dopuštenog direktorija")
	}
	return childAbs, nil
}

// EnsureLocalWithinRoot verifies that target stays under root both lexically and
// through existing filesystem links. This prevents a remote-controlled name
// from escaping a chosen download directory through a pre-existing symlink or
// junction nested below that directory. The root itself may be a user-selected
// link; only links beneath it are rejected.
func EnsureLocalWithinRoot(root, target string) error {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("lokalna putanja izlazi iz odabrane mape")
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return errors.New("odabrana lokalna mapa nije dostupna")
	}
	rootReal, err = filepath.Abs(rootReal)
	if err != nil {
		return err
	}
	current := rootAbs
	if rel != "." {
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			if part == "" || part == "." {
				continue
			}
			current = filepath.Join(current, part)
			lst, statErr := os.Lstat(current)
			if errors.Is(statErr, os.ErrNotExist) {
				break
			}
			if statErr != nil {
				return statErr
			}
			if lst.Mode()&os.ModeSymlink != 0 {
				return errors.New("lokalna putanja sadrži simboličku poveznicu")
			}
			if isReparsePoint(current) {
				return errors.New("lokalna putanja sadrži preusmjeravanje datotečnog sustava")
			}
			real, evalErr := filepath.EvalSymlinks(current)
			if evalErr != nil {
				return evalErr
			}
			real, evalErr = filepath.Abs(real)
			if evalErr != nil {
				return evalErr
			}
			realRel, relErr := filepath.Rel(rootReal, real)
			if relErr != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
				return errors.New("lokalna putanja napušta odabranu mapu kroz preusmjeravanje datotečnog sustava")
			}
		}
	}
	return nil
}

// EnsureNoRedirectPath verifies that every existing component below root is a
// normal filesystem object, not a symlink/junction/reparse point. The trusted
// root itself is intentionally allowed to be a Windows Known Folder redirect;
// ByFTP-owned descendants may not redirect elsewhere.
func EnsureNoRedirectPath(root, target string) error {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("putanja nije unutar dopuštene korisničke mape")
	}
	current := rootAbs
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		st, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil {
			return statErr
		}
		if st.Mode()&os.ModeSymlink != 0 || isReparsePoint(current) {
			return errors.New("ByFTP putanja ne smije biti preusmjerena simboličkom poveznicom ili junctionom")
		}
	}
	return nil
}

// EnsureNoRedirectDirectory creates missing descendants of root one component at
// a time and verifies every existing/created component is a real directory, not
// a symlink/junction/reparse point. It is intended for ByFTP-owned state/session
// directories where redirecting files into another location would weaken privacy.
func EnsureNoRedirectDirectory(root, target string) error {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("putanja nije unutar dopuštene korisničke mape")
	}
	current := rootAbs
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		st, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			if mkErr := os.Mkdir(current, 0700); mkErr != nil && !errors.Is(mkErr, os.ErrExist) {
				return mkErr
			}
			st, statErr = os.Lstat(current)
		}
		if statErr != nil {
			return statErr
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || isReparsePoint(current) {
			return errors.New("ByFTP putanja mora biti obična lokalna mapa bez preusmjeravanja")
		}
	}
	return nil
}
