package security

import (
	"errors"
	"path"
	"strings"
)

// ValidateRemoteFilePath validates a path that must identify one concrete
// remote file. ValidateRemotePath is intentionally broader because directory
// listing/tree operations legitimately use roots such as "/" and ".".
func ValidateRemoteFilePath(p string) error {
	if err := ValidateRemotePath(p); err != nil {
		return err
	}
	if p == "" {
		return errors.New("udaljena putanja datoteke je obavezna")
	}
	normalized := strings.ReplaceAll(p, "\\", "/")
	if strings.HasSuffix(normalized, "/") {
		return errors.New("udaljena putanja datoteke ne smije završavati direktorijskim separatorom")
	}
	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == "/" {
		return errors.New("udaljena putanja mora označavati datoteku, a ne direktorij")
	}
	if err := ValidateRemoteName(path.Base(cleaned)); err != nil {
		return err
	}
	return nil
}
