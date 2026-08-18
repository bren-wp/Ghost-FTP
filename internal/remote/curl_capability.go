package remote

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const maxCurlVersionOutput = 16 << 10

// curlVersionSupportsRevokeBestEffort provjerava verziju iz prvog curl --version
// retka. Opcija --ssl-revoke-best-effort postoji od curl 7.70.0.
func curlVersionSupportsRevokeBestEffort(output string) bool {
	line := output
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) < 2 || !strings.EqualFold(fields[0], "curl") {
		return false
	}
	parts := strings.Split(fields[1], ".")
	if len(parts) < 2 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 {
		return false
	}
	patch := 0
	if len(parts) >= 3 {
		digits := parts[2]
		if i := strings.IndexFunc(digits, func(r rune) bool { return r < '0' || r > '9' }); i >= 0 {
			digits = digits[:i]
		}
		if digits == "" {
			return false
		}
		patch, err = strconv.Atoi(digits)
		if err != nil || patch < 0 {
			return false
		}
	}
	if major != 7 {
		return major > 7
	}
	if minor != 70 {
		return minor > 70
	}
	return patch >= 0
}

// curlSupportsRevokeBestEffort je lokalna capability provjera bez mrežnog
// prometa. Ako provjera zakaže ili je curl prestar, ByFTP ne dodaje nepoznatu
// opciju; Schannel tada zadržava svoj zadani revocation model.
func curlSupportsRevokeBestEffort(curlPath string) bool {
	if runtime.GOOS != "windows" || strings.TrimSpace(curlPath) == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, curlPath, "--version")
	configureToolCommand(cmd)
	cmd.WaitDelay = time.Second
	cmd.Dir = filepath.Dir(curlPath)
	cmd.Env = sanitizedToolEnv(os.Environ())
	out := newBoundedOutput(maxCurlVersionOutput)
	errOut := newBoundedOutput(maxCurlVersionOutput)
	cmd.Stdout = out
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil || ctx.Err() != nil || out.Err("curl version odgovor") != nil {
		return false
	}
	return curlVersionSupportsRevokeBestEffort(out.String())
}
