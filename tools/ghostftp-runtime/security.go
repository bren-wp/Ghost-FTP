package main

import (
	"crypto/rand"
	"errors"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const maxJSONBody = 1 << 20 // 1 MiB

var sessionToken = newSessionToken()

func newSessionToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("Ghost FTP: cannot initialize local session token: %v", err))
	}
	return hex.EncodeToString(buf)
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
}

// A browser DNS-rebinding attack must not be able to re-host the loopback
// session endpoint under an attacker-controlled hostname. Apply to every
// route, including the unauthenticated one-time /api/session token bootstrap.
func allowLocalHost(w http.ResponseWriter, r *http.Request) bool {
    hostname, port, err := net.SplitHostPort(r.Host)
    if err != nil || port == "" {
        http.Error(w, "invalid local host", http.StatusForbidden)
        return false
    }
    if hostname != "127.0.0.1" && hostname != "localhost" && hostname != "::1" {
        http.Error(w, "non-loopback host rejected", http.StatusForbidden)
        return false
    }
    origin := strings.TrimSpace(r.Header.Get("Origin"))
    if origin != "" && origin != "http://"+r.Host {
        http.Error(w, "cross-origin local request rejected", http.StatusForbidden)
        return false
    }
    if site := strings.ToLower(r.Header.Get("Sec-Fetch-Site")); site == "cross-site" {
        http.Error(w, "cross-site request rejected", http.StatusForbidden)
        return false
    }
    return true
}

func authorizeMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && origin != "http://"+r.Host && origin != "https://"+r.Host {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return false
	}
	provided := r.Header.Get("X-GhostFTP-Token")
	if len(provided) != len(sessionToken) || subtle.ConstantTimeCompare([]byte(provided), []byte(sessionToken)) != 1 {
		http.Error(w, "unauthorized local request", http.StatusUnauthorized)
		return false
	}
	return true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if err == io.EOF {
			http.Error(w, "empty request", http.StatusBadRequest)
		} else {
			http.Error(w, "invalid request", http.StatusBadRequest)
		}
		return false
	}
	// Do not accept a second JSON value or non-whitespace trailing data.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid trailing request data", http.StatusBadRequest)
		return false
	}
	return true
}

// A link in *any* ancestor can redirect local destructive operations
// outside the user's selected folder, not only the final path component.
// This conservative preflight does not replace atomic OS no-follow handles.
func pathContainsSymlink(p string) bool {
    absolute, err := filepath.Abs(p)
    if err != nil { return true }
    for current := filepath.Clean(absolute); ; current = filepath.Dir(current) {
        info, statErr := os.Lstat(current)
        if statErr == nil {
            if info.Mode()&os.ModeSymlink != 0 { return true }
        } else if !errors.Is(statErr, os.ErrNotExist) {
            return true
        }
        parent := filepath.Dir(current)
        if parent == current { break }
    }
    return false
}

// Preflight all entries before touching permissions. Never follow symlinks
// within a directory tree, which would mutate targets outside the folder.
func chmodWithoutLinks(p string, mode os.FileMode, recursive bool) error {
    if pathContainsSymlink(p) { return fmt.Errorf("unsafe symbolic-link path") }
    target := filepath.Clean(p)
    if !recursive { return os.Chmod(target, mode) }
    var entries []string
    err := filepath.WalkDir(target, func(name string, item os.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if item.Type()&os.ModeSymlink != 0 || pathContainsSymlink(name) {
            return fmt.Errorf("recursive chmod contains symbolic link")
        }
        entries = append(entries, name)
        return nil
    })
    if err != nil { return err }
    for _, name := range entries {
        if pathContainsSymlink(name) { return fmt.Errorf("path changed during chmod") }
        if err := os.Chmod(name, mode); err != nil { return err }
    }
    return nil
}

func dangerousDestructivePath(p string) bool {
	p = filepath.Clean(strings.TrimSpace(p))
	if p == "" || p == "." || pathContainsSymlink(p) {
		return true
	}
	if home, err := os.UserHomeDir(); err == nil && samePath(p, home) {
		return true
	}
	if runtime.GOOS == "windows" {
		vol := filepath.VolumeName(p)
		if vol != "" && samePath(p, vol+`\`) {
			return true
		}
		return false
	}
	return p == string(filepath.Separator)
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
