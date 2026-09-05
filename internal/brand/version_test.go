package brand

import "testing"

func TestDisplayVersion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "beta baseline", in: "0.1.0", want: "0.1.0 Beta"},
		{name: "later beta", in: "0.9.7", want: "0.9.7 Beta"},
		{name: "first stable", in: "1.0.0", want: "1.0.0"},
		{name: "later stable", in: "1.4.2", want: "1.4.2"},
		{name: "development fallback", in: "", want: "dev"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DisplayVersion(test.in); got != test.want {
				t.Fatalf("DisplayVersion(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}
