package security

import "testing"

func TestValidateRemoteFilePathAcceptsConcreteFiles(t *testing.T) {
	for _, p := range []string{
		"index.php",
		"/public_html/index.php",
		"public_html/sub dir/file.txt",
		`public_html\assets\app.js`,
	} {
		if err := ValidateRemoteFilePath(p); err != nil {
			t.Fatalf("ValidateRemoteFilePath(%q)=%v", p, err)
		}
	}
}

func TestValidateRemoteFilePathRejectsDirectoryLikeTargets(t *testing.T) {
	for _, p := range []string{"", ".", "/", "public_html/", `/public_html\`} {
		if err := ValidateRemoteFilePath(p); err == nil {
			t.Fatalf("ValidateRemoteFilePath(%q) unexpectedly succeeded", p)
		}
	}
}

func TestValidateRemoteFilePathRetainsTraversalProtection(t *testing.T) {
	for _, p := range []string{"../file.txt", "/public_html/../file.txt"} {
		if err := ValidateRemoteFilePath(p); err == nil {
			t.Fatalf("ValidateRemoteFilePath(%q) accepted traversal", p)
		}
	}
}
