package brand

import (
	"strings"
	"testing"
)

func TestRuntimeMetadataUsesOfficialProductAndPublisherDestinations(t *testing.T) {
	if Publisher != "BRENDIGO LTD" {
		t.Fatalf("publisher = %q", Publisher)
	}
	if Website != "ghostftp.com" {
		t.Fatalf("product website = %q", Website)
	}
	if AuthorWebsite != "brendigo.com" {
		t.Fatalf("author website = %q", AuthorWebsite)
	}
	if Support != "brendigo.com/kontakt" {
		t.Fatalf("support = %q", Support)
	}

	for name, value := range map[string]string{
		"product website": Website,
		"author website":  AuthorWebsite,
		"support":         Support,
	} {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "github.com") || strings.Contains(lower, "githubusercontent.com") {
			t.Fatalf("%s runtime metadata must not expose a GitHub destination: %q", name, value)
		}
		if strings.Contains(lower, "://") {
			t.Fatalf("%s runtime metadata must remain schemeless: %q", name, value)
		}
	}
}
