package desktop

import "testing"

func TestRemoteParentKeepsSFTPHomeRelative(t *testing.T) {
	cases := map[string]string{
		".":                ".",
		"public_html":      ".",
		"public_html/site": "public_html",
		"/":                "/",
		"/www/site":        "/www",
	}
	for in, want := range cases {
		if got := remoteParent(in); got != want {
			t.Fatalf("remoteParent(%q)=%q want %q", in, got, want)
		}
	}
}

func TestOptionalRemotePathPreservesEmptyForProtocolDefault(t *testing.T) {
	if got := optionalRemotePath("   "); got != "" {
		t.Fatalf("got %q want empty", got)
	}
	if got := optionalRemotePath(`public_html\\site`); got != "public_html/site" {
		t.Fatalf("got %q", got)
	}
}
