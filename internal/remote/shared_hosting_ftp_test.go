package remote

import "testing"

func TestFTPCommandPathUsesLoginHomeNamespace(t *testing.T) {
	tests := map[string]string{
		"/":                           ".",
		"/public_html":                "public_html",
		"/public_html/site/index.php": "public_html/site/index.php",
		"public_html/assets":          "public_html/assets",
	}
	for input, want := range tests {
		if got := ftpCommandPath(input); got != want {
			t.Fatalf("ftpCommandPath(%q)=%q want %q", input, got, want)
		}
	}
}

func TestFTPURLPathCannotSwitchToServerAbsoluteNamespace(t *testing.T) {
	tests := map[string]string{
		"":                              "/",
		"/":                             "/",
		"//":                            "/",
		"/public_html":                  "/public_html",
		"//public_html":                 "/public_html",
		"///public_html/site/index.php": "/public_html/site/index.php",
		`\\public_html\\assets`:        "/public_html/assets",
	}
	for input, want := range tests {
		if got := ftpURLPath(input); got != want {
			t.Fatalf("ftpURLPath(%q)=%q want %q", input, got, want)
		}
	}
}
