package desktop

import (
	"errors"
	"strconv"

	"github.com/bren-wp/by-ftp/internal/security"
)

var errInvalidConnectionPort = errors.New("port mora biti broj između 1 i 65535")

// validateRawConnectionInput keeps the user's host, username and port text
// untouched until the security layer has seen them. This prevents edge
// whitespace or protocol control characters from disappearing before
// validation while preserving backend-compatible usernames verbatim.
func validateRawConnectionInput(protocol, host, portText, username string) (int, error) {
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return 0, errInvalidConnectionPort
	}
	if err := security.ValidateConnection(protocol, host, username, port); err != nil {
		return 0, err
	}
	return port, nil
}
