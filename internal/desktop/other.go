//go:build !windows

package desktop

import (
	"brendigo.com/byftp/internal/api"
	"errors"
)

func Run(_ *api.Engine, _ string) error {
	return errors.New("ByFTP desktop sučelje dostupno je samo na Windowsu")
}
