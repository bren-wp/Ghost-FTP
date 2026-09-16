package main

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

var errInvalidDesktopLaunch = errors.New("invalid Ghost FTP launch target")

type desktopLaunchTarget struct {
	Protocol string
	Host     string
	Port     int
	Username string
	Path     string
}

func parseDesktopLaunchTarget(raw string) (desktopLaunchTarget, error) {
	if raw == "" || len(raw) > 4096 || strings.IndexFunc(raw, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return desktopLaunchTarget{}, errInvalidDesktopLaunch
	}

	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "ghostftp") || u.User != nil || u.Fragment != "" {
		return desktopLaunchTarget{}, errInvalidDesktopLaunch
	}
	if !strings.EqualFold(u.Host, "connect") && !strings.EqualFold(u.Host, "open") {
		return desktopLaunchTarget{}, errInvalidDesktopLaunch
	}

	q := u.Query()
	for key, values := range q {
		switch key {
		case "protocol", "host", "port", "username", "path":
		default:
			return desktopLaunchTarget{}, errInvalidDesktopLaunch
		}
		if len(values) != 1 {
			return desktopLaunchTarget{}, errInvalidDesktopLaunch
		}
	}

	protocol := strings.ToLower(q.Get("protocol"))
	if protocol != "ftp" && protocol != "ftps" && protocol != "sftp" {
		return desktopLaunchTarget{}, errInvalidDesktopLaunch
	}

	host := q.Get("host")
	if err := security.ValidateHost(host); err != nil {
		return desktopLaunchTarget{}, errInvalidDesktopLaunch
	}

	port := 0
	if portText := q.Get("port"); portText != "" {
		port, err = strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return desktopLaunchTarget{}, errInvalidDesktopLaunch
		}
	}

	username := q.Get("username")
	if len(username) > 1024 || strings.ContainsAny(username, "\r\n\x00") {
		return desktopLaunchTarget{}, errInvalidDesktopLaunch
	}
	path := q.Get("path")
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") || security.ValidateRemotePath(path) != nil {
		return desktopLaunchTarget{}, errInvalidDesktopLaunch
	}

	return desktopLaunchTarget{
		Protocol: protocol,
		Host:     host,
		Port:     port,
		Username: username,
		Path:     path,
	}, nil
}

func desktopLaunchArgument(args []string) (desktopLaunchTarget, bool, error) {
	for _, arg := range args {
		if strings.HasPrefix(strings.ToLower(arg), "ghostftp://") {
			target, err := parseDesktopLaunchTarget(arg)
			return target, true, err
		}
	}
	return desktopLaunchTarget{}, false, nil
}
