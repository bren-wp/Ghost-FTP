package remote

import "testing"

func TestCurlVersionSupportsRevokeBestEffort(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"before feature", "curl 7.69.1 (Windows) libcurl/7.69.1 Schannel", false},
		{"feature baseline", "curl 7.70.0 (Windows) libcurl/7.70.0 Schannel", true},
		{"newer seven", "curl 7.83.1 (Windows) libcurl/7.83.1 Schannel", true},
		{"modern eight", "curl 8.0.1 (Windows) libcurl/8.0.1 Schannel", true},
		{"newer major", "curl 10.1.0 libcurl/10.1.0 Schannel", true},
		{"malformed", "curl unknown Schannel", false},
		{"not curl", "other 8.0.1", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := curlVersionSupportsRevokeBestEffort(tc.output); got != tc.want {
				t.Fatalf("curlVersionSupportsRevokeBestEffort(%q) = %v, want %v", tc.output, got, tc.want)
			}
		})
	}
}
