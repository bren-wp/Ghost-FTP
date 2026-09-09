package remote

import (
	"errors"
	"fmt"
)

func sshKeyscanFailure(runErr error, diagnostic string) error {
	if runErr == nil {
		return errors.New("nije moguće dohvatiti SFTP host ključ")
	}
	return fmt.Errorf("nije moguće dohvatiti SFTP host ključ: %w", newToolError("sftp", runErr, diagnostic))
}
