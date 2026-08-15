package remote

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSFTPCommandStream(t *testing.T) {
	got, err := buildSFTPCommandStream([]string{`ls -la "."`, `mkdir "public_html"`})
	if err != nil {
		t.Fatal(err)
	}
	want := "ls -la \".\"\nmkdir \"public_html\"\nquit\n"
	if got != want {
		t.Fatalf("unexpected stream:\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildSFTPCommandStreamRejectsInjection(t *testing.T) {
	for _, bad := range []string{"ls\nrm x", "ls\rquit", "ls\x00quit"} {
		if _, err := buildSFTPCommandStream([]string{bad}); err == nil {
			t.Fatalf("expected command %q to be rejected", bad)
		}
	}
}

func TestSFTPCommandArgsKeepAskPassCompatibleBatchSemantics(t *testing.T) {
	s := &SFTP{host: "example.test", port: 22, sshConfig: `C:\\data\\ssh.conf`, sessionHost: "byftp-session"}
	args := s.commandArgs()
	batchNo, batchFlag := -1, -1
	for i, arg := range args {
		if arg == "-oBatchMode=no" {
			batchNo = i
		}
		if arg == "-b" && i+1 < len(args) && args[i+1] == "-" {
			batchFlag = i
		}
	}
	if batchNo < 0 || batchFlag < 0 {
		t.Fatalf("missing controlled batch arguments: %#v", args)
	}
	if batchNo > batchFlag {
		t.Fatalf("BatchMode=no must precede -b so AskPass remains usable: %#v", args)
	}
}

func TestSFTPCloseRemovesSessionKnownHosts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(path, []byte("host key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(filepath.Dir(path), "session.conf")
	if err := os.WriteFile(config, []byte("Host test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &SFTP{knownHosts: path, sshConfig: config}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{path, config} {
		if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("session artifact still exists %s: %v", file, err)
		}
	}
}
