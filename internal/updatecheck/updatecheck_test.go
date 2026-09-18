package updatecheck

import "testing"

func TestSimulateKeepsInstalledVersionTruthful(t *testing.T) {
	result, err := Simulate("0.0.8")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Simulated {
		t.Fatal("expected local simulation")
	}
	if result.CurrentVersion != "0.0.8" || result.DisplayVersion != "0.0.8" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.UpdateURL != "https://ghostftp.com/#download" {
		t.Fatalf("unexpected update URL: %q", result.UpdateURL)
	}
}

func TestSimulateRejectsMalformedVersion(t *testing.T) {
	for _, value := range []string{"", "0.8", "v0.0.8", "00.0.8", "0.00.8", "0.0.-1", "+0.0.8", "0.+0.8", "0.0.+8", "0.０.8", "0.0.8.1"} {
		if _, err := Simulate(value); err == nil {
			t.Fatalf("expected invalid version: %q", value)
		}
	}
}
