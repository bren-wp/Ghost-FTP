//go:build !windows && !linux && !darwin

package remote

import "os/exec"

func configureToolCommand(cmd *exec.Cmd) {}
