package remote

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const processHelperEnv = "GhostFTP_PROCESS_HELPER"

func helperEnv(base []string, values map[string]string) []string {
	out := make([]string, 0, len(base)+len(values))
	for _, entry := range base {
		keep := true
		for key := range values {
			prefix := key + "="
			if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, entry)
		}
	}
	for key, value := range values {
		out = append(out, key+"="+value)
	}
	return out
}

func TestProcessLifecycleHelper(t *testing.T) {
	mode := os.Getenv(processHelperEnv)
	if mode == "" {
		return
	}
	marker := os.Getenv("GhostFTP_PROCESS_MARKER")
	ready := os.Getenv("GhostFTP_PROCESS_READY")
	switch mode {
	case "child":
		time.Sleep(700 * time.Millisecond)
		if marker != "" {
			_ = os.WriteFile(marker, []byte("orphan"), 0600)
		}
		os.Exit(0)
	case "parent":
		child := exec.Command(os.Args[0], "-test.run=TestProcessLifecycleHelper")
		child.Env = helperEnv(os.Environ(), map[string]string{
			processHelperEnv:          "child",
			"GhostFTP_PROCESS_MARKER": marker,
			"GhostFTP_PROCESS_READY":  "",
		})
		if err := child.Start(); err != nil {
			os.Exit(11)
		}
		if ready != "" {
			if err := os.WriteFile(ready, []byte("ready"), 0600); err != nil {
				_ = child.Process.Kill()
				os.Exit(12)
			}
		}
		_ = child.Wait()
		os.Exit(0)
	default:
		os.Exit(13)
	}
}

func waitForProcessMarker(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("helper child nije pokrenut na vrijeme")
}

func TestConfigureToolCommandCancelsDescendantProcess(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("process-tree cancellation nije produkcijski podržan na ovoj platformi")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "orphan.txt")
	ready := filepath.Join(dir, "ready.txt")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestProcessLifecycleHelper")
	cmd.Env = helperEnv(os.Environ(), map[string]string{
		processHelperEnv:          "parent",
		"GhostFTP_PROCESS_MARKER": marker,
		"GhostFTP_PROCESS_READY":  ready,
	})
	configureToolCommand(cmd)
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	if err := waitForProcessMarker(ready, 3*time.Second); err != nil {
		cancel()
		<-done
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled helper process unexpectedly succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled helper process did not terminate")
	}
	// The descendant writes only if it survives cancellation long enough. Give
	// it more time than its own delay and prove that no orphan remained alive.
	time.Sleep(950 * time.Millisecond)
	if data, err := os.ReadFile(marker); err == nil {
		t.Fatalf("descendant survived cancellation and wrote %q", data)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestConfigureToolCommandAllowsNormalCompletion(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestProcessLifecycleHelper")
	configureToolCommand(cmd)
	if err := cmd.Run(); err != nil {
		t.Fatalf("normal helper command failed: %v", err)
	}
}
