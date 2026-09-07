package brand

import (
	"strings"
	"testing"
)

func TestRuntimeMetadataUsesOfficialPublisherDestinations(t *testing.T) {
	if Publisher != "BRENDIGO LTD" {
		t.Fatalf("publisher = %q", Publisher)
	}
	for name, value := range map[string]string{
		"website": Website,
		"support": Support,
	} {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "github.com") || strings.Contains(lower, "githubusercontent.com") {
			t.Fatalf("%s runtime metadata must not expose a GitHub destination: %q", name, value)
		}
		if !strings.HasPrefix(lower, "brendigo.com") {
			t.Fatalf("%s runtime metadata is not an official Brendigo destination: %q", name, value)
		}
		if strings.Contains(lower, "://") {
			t.Fatalf("%s runtime metadata must remain schemeless: %q", name, value)
		}
	}
}
