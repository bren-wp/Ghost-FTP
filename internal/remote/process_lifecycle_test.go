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

const (
	processHelperEnv          = "GhostFTP_PROCESS_HELPER"
	processChildReadyEnv      = "GhostFTP_PROCESS_CHILD_READY"
	processSurvivalTriggerEnv = "GhostFTP_PROCESS_SURVIVAL_TRIGGER"
)

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
	childReady := os.Getenv(processChildReadyEnv)
	survivalTrigger := os.Getenv(processSurvivalTriggerEnv)
	switch mode {
	case "child":
		if childReady != "" {
			if err := os.WriteFile(childReady, []byte("ready"), 0600); err != nil {
				os.Exit(14)
			}
		}
		if survivalTrigger == "" {
			os.Exit(0)
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(survivalTrigger); err == nil {
				if marker != "" {
					_ = os.WriteFile(marker, []byte("orphan"), 0600)
				}
				os.Exit(0)
			} else if !errors.Is(err, os.ErrNotExist) {
				os.Exit(15)
			}
			time.Sleep(10 * time.Millisecond)
		}
		os.Exit(0)
	case "parent":
		child := exec.Command(os.Args[0], "-test.run=TestProcessLifecycleHelper")
		child.Env = helperEnv(os.Environ(), map[string]string{
			processHelperEnv:          "child",
			"GhostFTP_PROCESS_MARKER": marker,
			"GhostFTP_PROCESS_READY":  "",
			processChildReadyEnv:       childReady,
			processSurvivalTriggerEnv:  survivalTrigger,
		})
		if err := child.Start(); err != nil {
			os.Exit(11)
		}
		if childReady != "" {
			if err := waitForProcessMarker(childReady, 3*time.Second); err != nil {
				_ = child.Process.Kill()
				_ = child.Wait()
				os.Exit(16)
			}
		}
		if ready != "" {
			if err := os.WriteFile(ready, []byte("ready"), 0600); err != nil {
				_ = child.Process.Kill()
				_ = child.Wait()
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
	childReady := filepath.Join(dir, "child-ready.txt")
	survivalTrigger := filepath.Join(dir, "survival-trigger.txt")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestProcessLifecycleHelper")
	cmd.Env = helperEnv(os.Environ(), map[string]string{
		processHelperEnv:          "parent",
		"GhostFTP_PROCESS_MARKER": marker,
		"GhostFTP_PROCESS_READY":  ready,
		processChildReadyEnv:       childReady,
		processSurvivalTriggerEnv:  survivalTrigger,
	})
	configureToolCommand(cmd)
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	// The parent publishes ready only after the descendant itself has initialized
	// and entered its survival-trigger wait. Cancellation therefore always tests
	// a real descendant process instead of racing a fixed child-side sleep.
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

	// Arm the descendant only after cancellation has completed. A child that was
	// correctly terminated cannot observe this trigger; any surviving orphan will
	// observe it and write the marker. This avoids depending on scheduler timing
	// between process launch and context cancellation under the race detector.
	if err := os.WriteFile(survivalTrigger, []byte("probe"), 0600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
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
