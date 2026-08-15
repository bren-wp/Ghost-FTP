package remote

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"brendigo.com/byftp/internal/model"
)

func TestCfgQuoteEscapesBackslashAndQuote(t *testing.T) {
	got := cfgQuote(`a\\b"c`)
	want := `"a\\\\b\"c"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRemoteJoinStaysAbsolute(t *testing.T) {
	if got := remoteJoin("/public_html", "index.php"); got != "/public_html/index.php" {
		t.Fatalf("unexpected join: %s", got)
	}
}

func TestReplaceLocalFileAtomicKeepsBackup(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "file.txt")
	part := filepath.Join(dir, "part.txt")
	if err := os.WriteFile(local, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceLocalFileAtomic(local, part, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(local)
	if err != nil || string(data) != "new" {
		t.Fatalf("new file: %q err=%v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	backups := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "file.txt.byftp-backup-") {
			backups++
			old, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil || string(old) != "old" {
				t.Fatalf("backup: %q err=%v", old, err)
			}
		}
	}
	if backups != 1 {
		t.Fatalf("backup count=%d want 1", backups)
	}
}

func TestReplaceLocalFileAtomicRemovesRollback(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "file.txt")
	part := filepath.Join(dir, "part.txt")
	if err := os.WriteFile(local, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceLocalFileAtomic(local, part, false); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".byftp-rollback-") {
			t.Fatalf("rollback left behind: %s", entry.Name())
		}
	}
}

func TestRemoteTransferNamesStayInDirectory(t *testing.T) {
	dir, base, temp, saved, err := remoteTransferNames("/public_html/site/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if dir != "/public_html/site" || base != "index.html" {
		t.Fatalf("dir=%q base=%q", dir, base)
	}
	if !strings.HasPrefix(temp, ".byftp-part-") {
		t.Fatalf("temp=%q", temp)
	}
	if !strings.HasPrefix(saved, ".byftp-rollback-") {
		t.Fatalf("saved=%q", saved)
	}
}

func TestValidateChmod(t *testing.T) {
	for _, mode := range []string{"755", "644", "0755"} {
		if err := validateChmod(mode); err != nil {
			t.Fatalf("mode %s should be valid: %v", mode, err)
		}
	}
	for _, mode := range []string{"", "12", "888", "12345", "7a5"} {
		if err := validateChmod(mode); err == nil {
			t.Fatalf("mode %s should be invalid", mode)
		}
	}
}

func TestBackupName(t *testing.T) {
	rollback := "index.html.byftp-rollback-20260814T230000Z"
	if got := backupName("index.html", rollback, false); got != rollback {
		t.Fatalf("rollback name=%q", got)
	}
	want := "index.html.byftp-backup-20260814T230000Z"
	if got := backupName("index.html", rollback, true); got != want {
		t.Fatalf("backup name=%q want=%q", got, want)
	}
}

func TestCommitRemoteTempKeepsBackup(t *testing.T) {
	var renamed [][2]string
	var deleted []string
	ops := remoteCommitOps{
		rename: func(_ context.Context, _ string, from, to string) error {
			renamed = append(renamed, [2]string{from, to})
			return nil
		},
		delete: func(_ context.Context, _ string, name string, _ bool) error {
			deleted = append(deleted, name)
			return nil
		},
	}
	items := []model.Item{{Name: "index.html"}}
	if err := commitRemoteTemp(context.Background(), items, "/www", "index.html", ".part", "index.html.byftp-rollback-stamp", true, ops); err != nil {
		t.Fatal(err)
	}
	want := [][2]string{{"index.html", "index.html.byftp-backup-stamp"}, {".part", "index.html"}}
	if !reflect.DeepEqual(renamed, want) {
		t.Fatalf("renames=%v want=%v", renamed, want)
	}
	if len(deleted) != 0 {
		t.Fatalf("unexpected deletes=%v", deleted)
	}
}

func TestCommitRemoteTempRestoresOnActivationFailure(t *testing.T) {
	var renamed [][2]string
	var deleted []string
	ops := remoteCommitOps{
		rename: func(_ context.Context, _ string, from, to string) error {
			renamed = append(renamed, [2]string{from, to})
			if from == ".part" && to == "index.html" {
				return errors.New("activate failed")
			}
			return nil
		},
		delete: func(_ context.Context, _ string, name string, _ bool) error {
			deleted = append(deleted, name)
			return nil
		},
	}
	items := []model.Item{{Name: "index.html"}}
	err := commitRemoteTemp(context.Background(), items, "/www", "index.html", ".part", "index.html.byftp-rollback-stamp", false, ops)
	if err == nil {
		t.Fatal("expected activation failure")
	}
	want := [][2]string{{"index.html", "index.html.byftp-rollback-stamp"}, {".part", "index.html"}, {"index.html.byftp-rollback-stamp", "index.html"}}
	if !reflect.DeepEqual(renamed, want) {
		t.Fatalf("renames=%v want=%v", renamed, want)
	}
	if !reflect.DeepEqual(deleted, []string{".part"}) {
		t.Fatalf("deletes=%v", deleted)
	}
}

func TestDeleteGuardLimitsDepth(t *testing.T) {
	g := &deleteGuard{}
	if err := g.step(maxRemoteDeleteDepth + 1); err == nil {
		t.Fatal("expected depth guard to reject excessive recursion")
	}
}

func TestDeleteGuardLimitsItems(t *testing.T) {
	g := &deleteGuard{items: maxRemoteDeleteItems}
	if err := g.step(0); err == nil {
		t.Fatal("expected item guard to reject excessive delete tree")
	}
}

func TestCleanupStaleSFTPArtifactsRemovesOnlyManagedFiles(t *testing.T) {
	dir := t.TempDir()
	managed := []string{
		filepath.Join(dir, "askpass-old.txt"),
		filepath.Join(dir, ".byftp-sftp-session.conf"),
		filepath.Join(dir, ".byftp-known-session.txt"),
		filepath.Join(dir, ".byftp-scan-host-test.txt"),
		filepath.Join(dir, "byftp-key-test.known_hosts"),
		filepath.Join(dir, "ssh-client.conf"),
	}
	keep := filepath.Join(dir, "keep.txt")
	for _, file := range append(append([]string{}, managed...), keep) {
		if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cleanupStaleSFTPArtifacts(dir)
	for _, file := range managed {
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Fatalf("managed stale artifact still exists %s: %v", file, err)
		}
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}

func TestRemoteTransferNamesRelativeStayInCurrentDirectory(t *testing.T) {
	dir, base, temp, saved, err := remoteTransferNames("index.html")
	if err != nil {
		t.Fatal(err)
	}
	if dir != "." || base != "index.html" {
		t.Fatalf("dir=%q base=%q", dir, base)
	}
	if !strings.HasPrefix(temp, ".byftp-part-") {
		t.Fatalf("temp=%q", temp)
	}
	if !strings.HasPrefix(saved, ".byftp-rollback-") {
		t.Fatalf("saved=%q", saved)
	}
}

func TestRecursiveDeleteTreatsSymlinkAsFile(t *testing.T) {
	removedFiles := []string{}
	removedDirs := []string{}
	ops := recursiveDeleteOps{
		list: func(_ context.Context, target string) ([]model.Item, error) {
			if target != "/www/site" {
				t.Fatalf("unexpected list target %q", target)
			}
			return []model.Item{
				{Name: "index.html"},
				{Name: "linkdir", IsDirectory: true, IsSymlink: true},
			}, nil
		},
		removeFile: func(_ context.Context, target string) error {
			removedFiles = append(removedFiles, target)
			return nil
		},
		removeDir: func(_ context.Context, target string) error {
			removedDirs = append(removedDirs, target)
			return nil
		},
	}
	if err := recursiveDelete(context.Background(), "/www", "site", true, 0, &deleteGuard{}, ops); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removedFiles, []string{"/www/site/index.html", "/www/site/linkdir"}) {
		t.Fatalf("removed files=%v", removedFiles)
	}
	if !reflect.DeepEqual(removedDirs, []string{"/www/site"}) {
		t.Fatalf("removed dirs=%v", removedDirs)
	}
}

func TestRecursiveDeleteRejectsRemoteRoot(t *testing.T) {
	ops := recursiveDeleteOps{
		list:       func(context.Context, string) ([]model.Item, error) { return nil, nil },
		removeFile: func(context.Context, string) error { return nil },
		removeDir:  func(context.Context, string) error { return nil },
	}
	if err := recursiveDelete(context.Background(), "/", ".", true, 0, &deleteGuard{}, ops); err == nil {
		t.Fatal("expected root delete to be rejected")
	}
}

func TestBoundedOutputCapsMemory(t *testing.T) {
	b := newBoundedOutput(5)
	if n, err := b.Write([]byte("abcdefgh")); err != nil || n != 8 {
		t.Fatalf("write n=%d err=%v", n, err)
	}
	if got := b.String(); got != "abcde" {
		t.Fatalf("buffer=%q", got)
	}
	if err := b.Err("odgovor"); err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestReplaceLocalFileAtomicRejectsDirectoryTarget(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "target")
	part := filepath.Join(dir, "part.txt")
	if err := os.Mkdir(local, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceLocalFileAtomic(local, part, false); err == nil {
		t.Fatal("expected directory target to be rejected")
	}
	if st, err := os.Stat(local); err != nil || !st.IsDir() {
		t.Fatalf("target directory was modified: st=%v err=%v", st, err)
	}
}

func TestCommitRemoteTempRejectsDirectoryOverwrite(t *testing.T) {
	var renamed bool
	var deleted []string
	ops := remoteCommitOps{
		rename: func(context.Context, string, string, string) error { renamed = true; return nil },
		delete: func(_ context.Context, _ string, name string, _ bool) error {
			deleted = append(deleted, name)
			return nil
		},
	}
	err := commitRemoteTemp(context.Background(), []model.Item{{Name: "site", IsDirectory: true}}, "/www", "site", ".part", ".rollback", false, ops)
	if err == nil {
		t.Fatal("expected directory overwrite rejection")
	}
	if renamed {
		t.Fatal("directory target must not be renamed")
	}
	if !reflect.DeepEqual(deleted, []string{".part"}) {
		t.Fatalf("cleanup=%v", deleted)
	}
}

func TestParseMLSDLine(t *testing.T) {
	item, ok := parseMLSDLine("type=file;size=123;modify=20260814121530; report final.txt")
	if !ok || item.Name != "report final.txt" || item.Size != 123 || item.IsDirectory || item.Modified.IsZero() {
		t.Fatalf("unexpected MLSD file parse: %#v ok=%v", item, ok)
	}
	dir, ok := parseMLSDLine("type=dir;modify=20260814121530; public_html")
	if !ok || !dir.IsDirectory || dir.Name != "public_html" {
		t.Fatalf("unexpected MLSD dir parse: %#v ok=%v", dir, ok)
	}
	if _, ok := parseMLSDLine("type=cdir;modify=20260814121530; ."); ok {
		t.Fatal("cdir should be ignored")
	}
}

func TestParseMLSDRecognizesMachineReadableListing(t *testing.T) {
	items, recognized, err := parseMLSD([]byte("type=dir;modify=20260814121530; public_html\r\ntype=file;size=42;modify=20260814121531; index.php\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !recognized || len(items) != 2 || !items[0].IsDirectory {
		t.Fatalf("unexpected MLSD parse: recognized=%v items=%#v", recognized, items)
	}
}

func TestTransferTempNamesUseUnpredictableTokens(t *testing.T) {
	_, _, tempA, savedA, err := remoteTransferNames("/public/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _, tempB, savedB, err := remoteTransferNames("/public/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if tempA == tempB || savedA == savedB {
		t.Fatalf("transfer temp names must be unique: %q %q / %q %q", tempA, savedA, tempB, savedB)
	}
	if !strings.HasPrefix(tempA, ".byftp-part-") || !strings.HasPrefix(savedA, ".byftp-rollback-") {
		t.Fatalf("unexpected transfer names: %q %q", tempA, savedA)
	}
}

func TestFTPCommandUnsupportedClassification(t *testing.T) {
	for _, msg := range []string{"500 Unknown command", "502 Command not implemented", "504 unsupported command"} {
		if !ftpCommandUnsupported(errors.New(msg)) {
			t.Fatalf("expected unsupported classification for %q", msg)
		}
	}
	for _, msg := range []string{"connection timed out", "530 Login incorrect", "425 Can't open data connection"} {
		if ftpCommandUnsupported(errors.New(msg)) {
			t.Fatalf("transient/auth error must not disable MLSD cache for %q", msg)
		}
	}
}

func TestIsRetryableConservativeClassification(t *testing.T) {
	for _, err := range []error{
		&toolError{tool: "curl", code: 7, message: "Failed to connect"},
		&toolError{tool: "curl", code: 28, message: "Operation timed out"},
		errors.New("connection reset by peer"),
	} {
		if !IsRetryable(err) {
			t.Fatalf("expected transient error to be retryable: %v", err)
		}
	}
	for _, err := range []error{
		&toolError{tool: "curl", code: 67, message: "Login denied"},
		errors.New("permission denied"),
		errors.New("host key verification failed"),
		errors.New("no such file"),
		ErrSkipped,
		context.Canceled,
	} {
		if IsRetryable(err) {
			t.Fatalf("deterministic/cancelled error must not be retried: %v", err)
		}
	}
}
