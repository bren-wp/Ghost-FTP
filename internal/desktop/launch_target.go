package desktop

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const maxLaunchTargetLength = 4096

var (
	errInvalidLaunchTarget = errors.New("invalid Ghost FTP launch target")
	launchTargetMu         sync.Mutex
	initialLaunchTarget    *LaunchTarget
)

// LaunchTarget contains only non-secret connection metadata that may be
// transferred from the browser extension into the desktop client. Passwords,
// passphrases and other credential material are intentionally not represented.
type LaunchTarget struct {
	Protocol string
	Host     string
	Port     string
	Username string
	Path     string
}

func containsControlCharacters(value string) bool {
	for _, r := range value {
		if r <= 0x1f || r == 0x7f {
			return true
		}
	}
	return false
}

func launchDefaultPort(protocol string) string {
	switch protocol {
	case "sftp":
		return "22"
	case "ftp", "ftps":
		return "21"
	default:
		return ""
	}
}

func singleLaunchValue(values url.Values, key string, required bool) (string, error) {
	entries, ok := values[key]
	if !ok {
		if required {
			return "", errInvalidLaunchTarget
		}
		return "", nil
	}
	if len(entries) != 1 {
		return "", errInvalidLaunchTarget
	}
	value := entries[0]
	if containsControlCharacters(value) {
		return "", errInvalidLaunchTarget
	}
	return value, nil
}

// ParseLaunchTarget validates the custom protocol payload used by the browser
// extensions. The accepted query is deliberately tiny and allowlisted so the
// launcher can never become a credential transport or an arbitrary argument
// channel.
func ParseLaunchTarget(raw string) (LaunchTarget, error) {
	if raw == "" || len(raw) > maxLaunchTargetLength || containsControlCharacters(raw) {
		return LaunchTarget{}, errInvalidLaunchTarget
	}

	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "ghostftp") || u.Opaque != "" {
		return LaunchTarget{}, errInvalidLaunchTarget
	}
	if u.User != nil || u.Fragment != "" || u.RawFragment != "" {
		return LaunchTarget{}, errInvalidLaunchTarget
	}
	if !strings.EqualFold(u.Hostname(), "open") || u.Port() != "" || (u.Path != "" && u.Path != "/") {
		return LaunchTarget{}, errInvalidLaunchTarget
	}

	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return LaunchTarget{}, errInvalidLaunchTarget
	}
	allowed := map[string]bool{
		"protocol": true,
		"host":     true,
		"port":     true,
		"username": true,
		"path":     true,
	}
	for key := range values {
		if !allowed[key] {
			return LaunchTarget{}, errInvalidLaunchTarget
		}
	}

	protocol, err := singleLaunchValue(values, "protocol", true)
	if err != nil {
		return LaunchTarget{}, err
	}
	protocol = strings.ToLower(protocol)
	if protocol != "ftp" && protocol != "ftps" && protocol != "sftp" {
		return LaunchTarget{}, errInvalidLaunchTarget
	}

	host, err := singleLaunchValue(values, "host", true)
	if err != nil || host == "" || len(host) > 253 {
		return LaunchTarget{}, errInvalidLaunchTarget
	}
	username, err := singleLaunchValue(values, "username", false)
	if err != nil || len(username) > 1024 {
		return LaunchTarget{}, errInvalidLaunchTarget
	}
	port, err := singleLaunchValue(values, "port", false)
	if err != nil {
		return LaunchTarget{}, errInvalidLaunchTarget
	}
	if port == "" {
		port = launchDefaultPort(protocol)
	}
	if len(port) > 5 {
		return LaunchTarget{}, errInvalidLaunchTarget
	}
	path, err := singleLaunchValue(values, "path", false)
	if err != nil || len(path) > 4096 {
		return LaunchTarget{}, errInvalidLaunchTarget
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		return LaunchTarget{}, errInvalidLaunchTarget
	}

	validatedPort, err := validateRawConnectionInput(protocol, host, port, username)
	if err != nil || strconv.Itoa(validatedPort) != port {
		return LaunchTarget{}, errInvalidLaunchTarget
	}

	return LaunchTarget{
		Protocol: protocol,
		Host:     host,
		Port:     port,
		Username: username,
		Path:     path,
	}, nil
}

// ConfigureInitialLaunchTarget parses and stores one startup target. It is
// consumed exactly once by the desktop UI after persisted profile/settings
// state has loaded, ensuring the explicit browser launch wins over stale UI
// state without ever persisting credentials.
func ConfigureInitialLaunchTarget(raw string) error {
	target, err := ParseLaunchTarget(raw)
	if err != nil {
		return err
	}
	launchTargetMu.Lock()
	initialLaunchTarget = &target
	launchTargetMu.Unlock()
	return nil
}

func takeInitialLaunchTarget() (LaunchTarget, bool) {
	launchTargetMu.Lock()
	defer launchTargetMu.Unlock()
	if initialLaunchTarget == nil {
		return LaunchTarget{}, false
	}
	target := *initialLaunchTarget
	initialLaunchTarget = nil
	return target, true
}
