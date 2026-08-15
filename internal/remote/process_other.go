//go:build !windows

package remote

import "os/exec"

func configureToolCommand(cmd *exec.Cmd) {}
