package brand

import "testing"

func TestDisplayVersion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "first public release", in: "0.0.1", want: "0.0.1"},
		{name: "later zero-major release", in: "0.9.7", want: "0.9.7"},
		{name: "one-major release", in: "1.0.0", want: "1.0.0"},
		{name: "later release", in: "1.4.2", want: "1.4.2"},
		{name: "trim whitespace", in: " 0.0.2\n", want: "0.0.2"},
		{name: "development fallback", in: "", want: "dev"},
		{name: "whitespace development fallback", in: " \t\n", want: "dev"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DisplayVersion(test.in); got != test.want {
				t.Fatalf("DisplayVersion(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}
