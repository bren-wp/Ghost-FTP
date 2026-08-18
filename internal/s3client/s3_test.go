package s3client

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testClient(t *testing.T) *Client {
	t.Helper()
	c, err := New(Config{
		Endpoint: "https://s3.eu-central-1.amazonaws.com",
		Region: "eu-central-1",
		AccessKey: "AKIDEXAMPLE",
		SecretKey: "secret-example",
		SessionToken: "session-example",
		Bucket: "example-bucket",
	})
	if err != nil { t.Fatal(err) }
	c.now = func() time.Time { return time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC) }
	return c
}

func TestNewRejectsInsecureRemoteHTTP(t *testing.T) {
	if _, err := New(Config{Endpoint: "http://example.com", Region: "x", AccessKey: "a", SecretKey: "b", Bucket: "c"}); err == nil {
		t.Fatal("udaljeni HTTP S3 endpoint mora biti odbijen")
	}
	if _, err := New(Config{Endpoint: "http://127.0.0.1:9000", Region: "x", AccessKey: "a", SecretKey: "b", Bucket: "c"}); err != nil {
		t.Fatalf("loopback HTTP mora ostati dostupan za lokalne testne/S3 kompatibilne servise: %v", err)
	}
}

func TestSignedRequestDoesNotLeakSecret(t *testing.T) {
	c := testClient(t)
	req, err := c.signedRequest(context.Background(), http.MethodGet, "mapa/file name+.txt", url.Values{"list-type": {"2"}}, nil, nil)
	if err != nil { t.Fatal(err) }
	if strings.Contains(req.URL.String(), c.cfg.SecretKey) || strings.Contains(req.Header.Get("Authorization"), c.cfg.SecretKey) {
		t.Fatal("S3 secret key ne smije završiti u URL-u ili Authorization headeru")
	}
	if !strings.Contains(req.Header.Get("Authorization"), "Credential=AKIDEXAMPLE/") {
		t.Fatal("Authorization nema očekivani access-key credential scope")
	}
	if req.Header.Get("x-amz-security-token") != "session-example" || !strings.Contains(req.Header.Get("Authorization"), "x-amz-security-token") {
		t.Fatal("temporary session token mora biti poslan i uključen u signed headers")
	}
	if !strings.Contains(req.URL.EscapedPath(), "file%20name%2B.txt") {
		t.Fatalf("S3 ključ nije AWS URI-enkodiran: %s", req.URL.EscapedPath())
	}
}

func TestCanonicalQuerySortsAfterEncodingAndUsesPercent20(t *testing.T) {
	q := url.Values{"z": {"a b", "a+b"}, "a": {"2", "1"}, "é": {"/"}}
	got := canonicalQuery(q)
	want := "%C3%A9=%2F&a=1&a=2&z=a%20b&z=a%2Bb"
	if got != want {
		t.Fatalf("neočekivani canonical query: %q, očekivano %q", got, want)
	}
}

func TestAWSURIEncodePreservesOnlyUnreservedAndObjectSlashes(t *testing.T) {
	got := encodePath("/bucket/folder/a b+#.txt")
	if got != "/bucket/folder/a%20b%2B%23.txt" {
		t.Fatalf("neočekivani encoded path: %q", got)
	}
	if got := awsURIEncode("a/b+c d", true); got != "a%2Fb%2Bc%20d" {
		t.Fatalf("query-style URI encode nije ispravan: %q", got)
	}
}
