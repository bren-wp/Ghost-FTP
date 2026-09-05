package remote

import "testing"

func TestNormalizePermissionDisplay(t *testing.T) {
	for input, want := range map[string]string{
		"-rw-r--r--": "-rw-r--r--",
		"drwxr-xr-x": "drwxr-xr-x",
		"lrwxrwxrwx": "lrwxrwxrwx",
		"0755":       "0755",
		"644":        "644",
		"adfrw":      "",
		"0999":       "",
		"rwxr-xr-x":  "",
		"":            "",
	} {
		if got := normalizePermissionDisplay(input); got != want {
			t.Fatalf("normalizePermissionDisplay(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestListAndMLSDPermissions(t *testing.T) {
	item, ok := parseListLine("-rw-r----- 1 user group 123 Jan 02 03:04 report.txt")
	if !ok || item.Permissions != "-rw-r-----" {
		t.Fatalf("LIST permissions not preserved: %#v, ok=%v", item, ok)
	}

	item, ok = parseMLSDLine("type=file;size=123;unix.mode=0640;modify=20260102030405; report.txt")
	if !ok || item.Permissions != "0640" {
		t.Fatalf("MLSD unix.mode not preserved: %#v, ok=%v", item, ok)
	}

	item, ok = parseMLSDLine("type=file;size=123;perm=adfrw; report.txt")
	if !ok || item.Permissions != "" {
		t.Fatalf("MLSD perm capability must not be presented as file mode: %#v, ok=%v", item, ok)
	}
}
