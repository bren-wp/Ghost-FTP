//go:build !windows && !linux && !darwin

package remote

import (
	"errors"
	"os/exec"
)

func configureToolCommand(cmd *exec.Cmd) {}

func runToolCommand(cmd *exec.Cmd) error {
	if cmd == nil {
		return errors.New("vanjski proces nije postavljen")
	}
	return cmd.Run()
}
