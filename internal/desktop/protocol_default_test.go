package desktop

import "testing"

func TestFreshConnectionProtocolRequiresTLS(t *testing.T) {
	if defaultConnectionProtocol != "ftps" {
		t.Fatalf("fresh connection protocol=%q, want explicit FTPS", defaultConnectionProtocol)
	}
}
