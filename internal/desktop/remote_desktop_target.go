package desktop

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const defaultRemoteDesktopPort = 3389

func remoteDesktopTarget(value string) (string, error) {
	input := strings.TrimSpace(value)
	if input == "" {
		return "", fmt.Errorf("remote desktop server is required")
	}
	if len(input) > 512 || strings.ContainsAny(input, "\x00\r\n\t /\\@?#") {
		return "", fmt.Errorf("remote desktop server contains unsupported characters")
	}

	host := input
	port := defaultRemoteDesktopPort
	if strings.HasPrefix(input, "[") {
		end := strings.IndexByte(input, ']')
		if end <= 1 {
			return "", fmt.Errorf("invalid bracketed remote desktop host")
		}
		host = input[1:end]
		rest := input[end+1:]
		if rest != "" {
			if !strings.HasPrefix(rest, ":") || len(rest) == 1 {
				return "", fmt.Errorf("invalid remote desktop port")
			}
			parsed, err := strconv.Atoi(rest[1:])
			if err != nil {
				return "", fmt.Errorf("invalid remote desktop port")
			}
			port = parsed
		}
	} else if strings.Count(input, ":") == 1 {
		parts := strings.SplitN(input, ":", 2)
		if parts[0] == "" || parts[1] == "" {
			return "", fmt.Errorf("invalid remote desktop target")
		}
		parsed, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", fmt.Errorf("invalid remote desktop port")
		}
		host, port = parts[0], parsed
	}

	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" || len(host) > 255 || strings.ContainsAny(host, "\x00\r\n\t /\\@?#") {
		return "", fmt.Errorf("invalid remote desktop host")
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("remote desktop port must be between 1 and 65535")
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}
