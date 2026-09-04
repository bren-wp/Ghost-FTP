package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateHost(t *testing.T) {
	valid := []string{"ftp.example.com", "localhost", "127.0.0.1", "2001:db8::1", "[2001:db8::1]"}
	for _, host := range valid {
		if err := ValidateHost(host); err != nil {
			t.Fatalf("expected valid host %q: %v", host, err)
		}
	}
	invalid := []string{"", "bad host", "https://example.com", "user@example.com", "example.com/path", "-bad.example", "bad-.example", "bad\n.example"}
	for _, host := range invalid {
		if err := ValidateHost(host); err == nil {
			t.Fatalf("expected invalid host %q", host)
		}
	}
}

func TestValidateSecretRejectsControlLineBreaks(t *testing.T) {
	for _, s := range []string{"bad\nsecret", "bad\rsecret", "bad\x00secret"} {
		if err := ValidateSecret(s); err == nil {
			t.Fatalf("expected rejected secret %q", s)
		}
	}
	if err := ValidateSecret("normal password with spaces !@#$%"); err != nil {
		t.Fatalf("unexpected rejection: %v", err)
	}
}

func TestValidateRemotePathTraversal(t *testing.T) {
	for _, p := range []string{"../etc", "/public_html/../secret", `\\server\\..\\secret`} {
		if err := ValidateRemotePath(p); err == nil {
			t.Fatalf("expected traversal rejection: %q", p)
		}
	}
	for _, p := range []string{"/", "/public_html", "/public_html/site/file.php"} {
		if err := ValidateRemotePath(p); err != nil {
			t.Fatalf("unexpected path rejection %q: %v", p, err)
		}
	}
}

func TestSafeLocalChild(t *testing.T) {
	base := t.TempDir()
	if _, err := SafeLocalChild(base, "file.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := SafeLocalChild(base, ".."); err == nil {
		t.Fatal("expected parent traversal rejection")
	}
}

func TestValidateNameRejectsWindowsReservedAndAmbiguousNames(t *testing.T) {
	invalid := []string{
		"CON", "con.txt", "PRN", "AUX.log", "NUL", "COM1", "com9.txt", "LPT1", "LPT9.doc", "CLOCK$",
		"trailing.", "trailing ", " leading.txt", "..", `a/b`, `a\\b`,
	}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) unexpectedly succeeded", name)
		}
	}
	valid := []string{"normal.txt", "COM10.txt", "LPT10", "moja datoteka.txt", "čćž.txt"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q): %v", name, err)
		}
	}
}

func TestEnsureLocalWithinRootRejectsNestedSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	if err := EnsureLocalWithinRoot(root, filepath.Join(link, "remote.txt")); err == nil {
		t.Fatal("nested symlink escape unexpectedly accepted")
	}
}

func TestEnsureLocalWithinRootAllowsNewChild(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "new-folder", "file.txt")
	if err := EnsureLocalWithinRoot(root, target); err != nil {
		t.Fatalf("safe new child rejected: %v", err)
	}
}

func TestValidateRemoteNameAllowsServerCharactersButBlocksTraversal(t *testing.T) {
	for _, name := range []string{"normal.txt", "colon:name.txt", "question?.txt", "pipe|name", "čćž.txt"} {
		if err := ValidateRemoteName(name); err != nil {
			t.Errorf("ValidateRemoteName(%q): %v", name, err)
		}
	}
	for _, name := range []string{"", ".", "..", "a/b", `a\b`, "bad\nname", "bad\x00name"} {
		if err := ValidateRemoteName(name); err == nil {
			t.Errorf("ValidateRemoteName(%q) unexpectedly succeeded", name)
		}
	}
}

func TestRemoveTreeNoFollowDoesNotTraverseSymlink(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	outside := t.TempDir()
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	outsideFile := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(outsideFile, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "local.txt"), []byte("remove"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveTreeNoFollow(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outsideFile); err != nil {
		t.Fatalf("symlink target was touched: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("root still exists: %v", err)
	}
}

func TestEnsureNoRedirectPathRejectsNestedSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "GhostFTP")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := EnsureNoRedirectPath(root, filepath.Join(link, "GhostFTP")); err == nil {
		t.Fatal("expected nested redirect to be rejected")
	}
}

func TestEnsureNoRedirectPathAllowsMissingNormalDescendants(t *testing.T) {
	root := t.TempDir()
	if err := EnsureNoRedirectPath(root, filepath.Join(root, "GhostFTP", "GhostFTP")); err != nil {
		t.Fatalf("normal missing descendants should be allowed: %v", err)
	}
}

func TestEnsureNoRedirectDirectoryCreatesNormalDescendants(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "GhostFTP", "GhostFTP", "known_hosts")
	if err := EnsureNoRedirectDirectory(root, target); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(target)
	if err != nil || !st.IsDir() {
		t.Fatalf("secure directory not created: st=%v err=%v", st, err)
	}
}

func TestEnsureNoRedirectDirectoryRejectsNestedSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "GhostFTP")); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	if err := EnsureNoRedirectDirectory(root, filepath.Join(root, "GhostFTP", "GhostFTP")); err == nil {
		t.Fatal("redirected directory unexpectedly accepted")
	}
}
