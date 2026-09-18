package updatecheck

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func testHTTPClient(body string, status int) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
}

func TestCheckFindsNewStableRelease(t *testing.T) {
	client := NewForTest(testHTTPClient(
		`{"tag_name":"ghostftp-v0.0.9","html_url":"https://github.com/bren-wp/Ghost-FTP/releases/tag/ghostftp-v0.0.9","draft":false,"prerelease":false}`,
		http.StatusOK,
	), "https://api.github.com/repos/bren-wp/Ghost-FTP/releases/latest")

	result, err := client.Check(context.Background(), "0.0.8")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Available || result.LatestVersion != "0.0.9" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestCheckRejectsUntrustedReleaseURL(t *testing.T) {
	client := NewForTest(testHTTPClient(
		`{"tag_name":"v0.0.9","html_url":"https://example.com/evil","draft":false,"prerelease":false}`,
		http.StatusOK,
	), "https://api.github.com/repos/bren-wp/Ghost-FTP/releases/latest")

	if _, err := client.Check(context.Background(), "0.0.8"); err != ErrUntrustedReleaseURL {
		t.Fatalf("expected ErrUntrustedReleaseURL, got %v", err)
	}
}

func TestCheckRejectsDraftAndPrerelease(t *testing.T) {
	cases := []string{
		`{"tag_name":"v0.0.9","html_url":"https://github.com/bren-wp/Ghost-FTP/releases/tag/v0.0.9","draft":true,"prerelease":false}`,
		`{"tag_name":"v0.0.9","html_url":"https://github.com/bren-wp/Ghost-FTP/releases/tag/v0.0.9","draft":false,"prerelease":true}`,
	}
	for _, body := range cases {
		client := NewForTest(testHTTPClient(body, http.StatusOK), "https://api.github.com/repos/bren-wp/Ghost-FTP/releases/latest")
		if _, err := client.Check(context.Background(), "0.0.8"); err == nil {
			t.Fatal("expected stable-release rejection")
		}
	}
}

func TestVersionComparison(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"0.0.8", "0.0.8", 0},
		{"0.0.9", "0.0.8", 1},
		{"0.1.0", "0.0.9", 1},
		{"1.0.0", "0.99.99", 1},
		{"0.0.7", "0.0.8", -1},
	} {
		a, err := parseVersion(tc.a)
		if err != nil { t.Fatal(err) }
		b, err := parseVersion(tc.b)
		if err != nil { t.Fatal(err) }
		if got := a.Compare(b); got != tc.want {
			t.Fatalf("%s compare %s = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestVersionRejectsAmbiguousOrMalformedInput(t *testing.T) {
	for _, value := range []string{"", "0.8", "v0.0.8", "00.0.8", "0.00.8", "0.0.-1", "0.0.8.1"} {
		if _, err := parseVersion(value); err == nil {
			t.Fatalf("expected invalid version: %q", value)
		}
	}
}
