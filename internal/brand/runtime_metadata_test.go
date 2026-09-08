package brand

import (
	"strings"
	"testing"
)

func TestRuntimeMetadataUsesGhostFTPOnly(t *testing.T) {
	if ProductName != "Ghost FTP" {
		t.Fatalf("product name = %q", ProductName)
	}
	if Company != ProductName {
		t.Fatalf("company = %q; want product identity %q", Company, ProductName)
	}
	if Website != "ghostftp.com" {
		t.Fatalf("product website = %q", Website)
	}
	if Support != Website {
		t.Fatalf("support = %q; want product website %q", Support, Website)
	}

	for name, value := range map[string]string{
		"product name":    ProductName,
		"company":         Company,
		"product website": Website,
		"support":         Support,
	} {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "brendigo") {
			t.Fatalf("%s generic runtime metadata must not expose Brendigo identity: %q", name, value)
		}
		if strings.Contains(lower, "github.com") || strings.Contains(lower, "githubusercontent.com") {
			t.Fatalf("%s runtime metadata must not expose a GitHub destination: %q", name, value)
		}
		if strings.Contains(lower, "://") {
			t.Fatalf("%s runtime metadata must remain schemeless: %q", name, value)
		}
	}
}
