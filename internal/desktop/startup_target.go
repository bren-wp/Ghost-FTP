package desktop

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

const maxStartupTargetLength = 4096

var errInvalidStartupTarget = errors.New("invalid Ghost FTP launch target")

// StartupTarget contains only non-sensitive connection metadata supplied by an
// explicit OS protocol launch (for example from the official browser helper).
// Passwords, passphrases and private-key material are deliberately absent.
type StartupTarget struct {
	Protocol string
	Host     string
	Port     int
	Username string
	Path     string
}

var startupTargetState struct {
	sync.Mutex
	target *StartupTarget
}

func startupTargetHasControlCharacters(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool {
		return r < 0x20 || r == 0x7f
	}) >= 0
}

// ParseStartupTarget validates the allowlisted custom-protocol payload shared
// by the browser helper, a fresh desktop process and an already-running desktop
// process. It deliberately accepts no credential fields or arbitrary command
// arguments.
func ParseStartupTarget(raw string) (StartupTarget, error) {
	if raw == "" || len(raw) > maxStartupTargetLength || startupTargetHasControlCharacters(raw) {
		return StartupTarget{}, errInvalidStartupTarget
	}

	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "ghostftp") || u.Opaque != "" || u.User != nil || u.Fragment != "" || u.RawFragment != "" {
		return StartupTarget{}, errInvalidStartupTarget
	}
	if u.Port() != "" || (!strings.EqualFold(u.Hostname(), "connect") && !strings.EqualFold(u.Hostname(), "open")) {
		return StartupTarget{}, errInvalidStartupTarget
	}
	if u.Path != "" && u.Path != "/" {
		return StartupTarget{}, errInvalidStartupTarget
	}

	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return StartupTarget{}, errInvalidStartupTarget
	}
	for key, values := range q {
		switch key {
		case "protocol", "host", "port", "username", "path":
		default:
			return StartupTarget{}, errInvalidStartupTarget
		}
		if len(values) != 1 || startupTargetHasControlCharacters(values[0]) {
			return StartupTarget{}, errInvalidStartupTarget
		}
	}

	protocol := strings.ToLower(q.Get("protocol"))
	if protocol != "ftp" && protocol != "ftps" && protocol != "sftp" {
		return StartupTarget{}, errInvalidStartupTarget
	}

	host := q.Get("host")
	if err := security.ValidateHost(host); err != nil {
		return StartupTarget{}, errInvalidStartupTarget
	}

	port := 0
	if portText := q.Get("port"); portText != "" {
		port, err = strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != portText {
			return StartupTarget{}, errInvalidStartupTarget
		}
	}

	username := q.Get("username")
	if len(username) > 1024 {
		return StartupTarget{}, errInvalidStartupTarget
	}
	path := q.Get("path")
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") || security.ValidateRemotePath(path) != nil {
		return StartupTarget{}, errInvalidStartupTarget
	}

	return StartupTarget{
		Protocol: protocol,
		Host:     host,
		Port:     port,
		Username: username,
		Path:     path,
	}, nil
}

// SetStartupTarget records a one-shot target for the next desktop UI startup
// or explicit foreground handoff. The target lives only in process memory and
// is cleared as soon as a UI reads it.
func SetStartupTarget(target StartupTarget) {
	startupTargetState.Lock()
	defer startupTargetState.Unlock()
	copyTarget := target
	startupTargetState.target = &copyTarget
}

func takeStartupTarget() (StartupTarget, bool) {
	startupTargetState.Lock()
	defer startupTargetState.Unlock()
	if startupTargetState.target == nil {
		return StartupTarget{}, false
	}
	target := *startupTargetState.target
	startupTargetState.target = nil
	return target, true
}

func startupTargetPort(target StartupTarget) int {
	if target.Port > 0 {
		return target.Port
	}
	switch target.Protocol {
	case "sftp":
		return 22
	case "ftpsi":
		return 990
	default:
		return 21
	}
}
