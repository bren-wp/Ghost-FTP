package remote

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func curlBandwidthConfigLine(limitBytesPerSecond int64) string {
	if limitBytesPerSecond <= 0 {
		return ""
	}
	return "limit-rate = " + cfgQuote(strconv.FormatInt(limitBytesPerSecond, 10))
}

func (c *CurlFTP) runTransfer(ctx context.Context, limitBytesPerSecond int64, lines []string) ([]byte, error) {
	line := curlBandwidthConfigLine(limitBytesPerSecond)
	if line == "" {
		return c.run(ctx, lines)
	}
	limited := make([]string, 0, len(lines)+1)
	limited = append(limited, lines...)
	limited = append(limited, line)
	return c.run(ctx, limited)
}

func sftpLimitKbitPerSecond(limitBytesPerSecond int64) int64 {
	if limitBytesPerSecond <= 0 {
		return 0
	}
	// OpenSSH sftp -l accepts Kbit/s. Floor instead of round-up so the native
	// transport ceiling never exceeds the scheduler's per-slot byte budget.
	limitKbit := limitBytesPerSecond * 8 / 1000
	if limitKbit < 1 {
		return 1
	}
	return limitKbit
}

func (s *SFTP) commandArgsWithBandwidth(limitBytesPerSecond int64) []string {
	args := s.commandArgs()
	limitKbit := sftpLimitKbitPerSecond(limitBytesPerSecond)
	if limitKbit <= 0 || len(args) == 0 {
		return args
	}
	// commandArgs deliberately keeps the host as the last argument. Insert the
	// native -l option before it so all strict host-key and forwarding controls
	// remain unchanged and the host can never be interpreted as option data.
	limited := make([]string, 0, len(args)+2)
	limited = append(limited, args[:len(args)-1]...)
	limited = append(limited, "-l", strconv.FormatInt(limitKbit, 10))
	limited = append(limited, args[len(args)-1])
	return limited
}

func (s *SFTP) runTransfer(ctx context.Context, limitBytesPerSecond int64, commands ...string) (string, error) {
	if limitBytesPerSecond <= 0 {
		return s.run(ctx, commands...)
	}
	input, err := buildSFTPCommandStream(commands)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, s.sftp, s.commandArgsWithBandwidth(limitBytesPerSecond)...)
	configureToolCommand(cmd)
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdin = strings.NewReader(input)
	cmd.Dir = filepath.Dir(s.sftp)
	cmd.Env, err = s.askpassEnvironment()
	if err != nil {
		return "", err
	}
	out := newBoundedOutput(maxCommandStdout)
	er := newBoundedOutput(maxCommandStderr)
	cmd.Stdout = out
	cmd.Stderr = er
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		if overflowErr := er.Err("dijagnostički odgovor"); overflowErr != nil {
			return "", overflowErr
		}
		msg := strings.TrimSpace(er.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", newToolError("sftp", err, msg)
	}
	if err := out.Err("odgovor"); err != nil {
		return "", err
	}
	return out.String(), nil
}
