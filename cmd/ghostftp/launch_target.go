package main

import (
	"errors"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/desktop"
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
	target, err := desktop.ParseStartupTarget(raw)
	if err != nil {
		return desktopLaunchTarget{}, errInvalidDesktopLaunch
	}
	return desktopLaunchTarget{
		Protocol: target.Protocol,
		Host:     target.Host,
		Port:     target.Port,
		Username: target.Username,
		Path:     target.Path,
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
