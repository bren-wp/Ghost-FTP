package s3client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
}

type Item struct {
	Name   string
	Size   int64
	Prefix bool
}

type Client struct {
	cfg  Config
	http *http.Client
	now  func() time.Time
}

func New(cfg Config) (*Client, error) {
	cfg.Endpoint = strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.AccessKey = strings.TrimSpace(cfg.AccessKey)
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	if cfg.Endpoint == "" || cfg.Region == "" || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return nil, errors.New("S3 endpoint, regija, access key, secret key i bucket su obavezni")
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !loopbackHTTP(u)) {
		return nil, errors.New("S3 endpoint mora biti HTTPS URL bez korisnika, queryja i fragmenta")
	}
	if strings.ContainsAny(cfg.Bucket, "/\\\x00\r\n") || cfg.Bucket == "." || cfg.Bucket == ".." {
		return nil, errors.New("neispravan S3 bucket")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment
	return &Client{
		cfg: cfg,
		http: &http.Client{Timeout: 90 * time.Second, Transport: transport},
		now: time.Now,
	}, nil
}

func loopbackHTTP(u *url.URL) bool {
	if u == nil || u.Scheme != "http" { return false }
	host := u.Hostname()
	if host == "localhost" { return true }
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func hmacSHA256(key []byte, text string) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(text))
	return h.Sum(nil)
}

func hashHex(data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}

func encodePath(value string) string {
	parts := strings.Split(strings.TrimPrefix(value, "/"), "/")
	for i := range parts { parts[i] = url.PathEscape(parts[i]) }
	return "/" + strings.Join(parts, "/")
}

func canonicalQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values { keys = append(keys, k) }
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vals := append([]string(nil), values[k]...)
		sort.Strings(vals)
		for _, v := range vals { parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v)) }
	}
	return strings.ReplaceAll(strings.Join(parts, "&"), "+", "%20")
}

func (c *Client) objectURL(key string, query url.Values) (*url.URL, error) {
	base, err := url.Parse(c.cfg.Endpoint)
	if err != nil { return nil, err }
	path := strings.TrimSuffix(base.Path, "/") + "/" + c.cfg.Bucket
	if key != "" { path += "/" + strings.TrimPrefix(key, "/") }
	base.Path = path
	base.RawPath = encodePath(path)
	base.RawQuery = canonicalQuery(query)
	return base, nil
}

func (c *Client) signedRequest(ctx context.Context, method, key string, query url.Values, body []byte, extra http.Header) (*http.Request, error) {
	u, err := c.objectURL(key, query)
	if err != nil { return nil, err }
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil { return nil, err }
	for k, vals := range extra { for _, v := range vals { req.Header.Add(k, v) } }
	now := c.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	payloadHash := hashHex(body)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	signedNames := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	for k := range extra {
		lk := strings.ToLower(k)
		if lk == "host" || lk == "authorization" || lk == "x-amz-date" || lk == "x-amz-content-sha256" { continue }
		signedNames = append(signedNames, lk)
	}
	sort.Strings(signedNames)
	seen := map[string]bool{}
	canonicalHeaders := strings.Builder{}
	finalNames := make([]string, 0, len(signedNames))
	for _, name := range signedNames {
		if seen[name] { continue }
		seen[name] = true
		value := ""
		if name == "host" { value = req.URL.Host } else { value = strings.Join(strings.Fields(req.Header.Get(name)), " ") }
		canonicalHeaders.WriteString(name+":"+value+"\n")
		finalNames = append(finalNames, name)
	}
	signedHeaderNames := strings.Join(finalNames, ";")
	canonicalURI := req.URL.EscapedPath()
	canonicalRequest := strings.Join([]string{
		method, canonicalURI, req.URL.RawQuery, canonicalHeaders.String(), signedHeaderNames, payloadHash,
	}, "\n")
	scope := date + "/" + c.cfg.Region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + hashHex([]byte(canonicalRequest))
	kDate := hmacSHA256([]byte("AWS4"+c.cfg.SecretKey), date)
	kRegion := hmacSHA256(kDate, c.cfg.Region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+c.cfg.AccessKey+"/"+scope+", SignedHeaders="+signedHeaderNames+", Signature="+signature)
	return req, nil
}

func (c *Client) do(ctx context.Context, method, key string, query url.Values, body []byte, extra http.Header) ([]byte, error) {
	req, err := c.signedRequest(ctx, method, key, query, body, extra)
	if err != nil { return nil, err }
	resp, err := c.http.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, 16<<20)
	data, err := io.ReadAll(limited)
	if err != nil { return nil, err }
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("S3 zahtjev nije uspio (%s): %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}

type listResult struct {
	Contents []struct {
		Key  string `xml:"Key"`
		Size int64  `xml:"Size"`
	} `xml:"Contents"`
	Prefixes []struct { Prefix string `xml:"Prefix"` } `xml:"CommonPrefixes"`
	NextContinuationToken string `xml:"NextContinuationToken"`
	IsTruncated bool `xml:"IsTruncated"`
}

func (c *Client) List(ctx context.Context, prefix string) ([]Item, error) {
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") { prefix += "/" }
	var out []Item
	token := ""
	for pages := 0; pages < 1000; pages++ {
		q := url.Values{"list-type": {"2"}, "delimiter": {"/"}, "prefix": {prefix}, "max-keys": {"1000"}}
		if token != "" { q.Set("continuation-token", token) }
		data, err := c.do(ctx, http.MethodGet, "", q, nil, nil)
		if err != nil { return nil, err }
		var result listResult
		if err := xml.Unmarshal(data, &result); err != nil { return nil, errors.New("S3 odgovor nije valjan XML") }
		for _, p := range result.Prefixes {
			name := strings.TrimSuffix(strings.TrimPrefix(p.Prefix, prefix), "/")
			if name != "" { out = append(out, Item{Name: name, Prefix: true}) }
		}
		for _, obj := range result.Contents {
			name := strings.TrimPrefix(obj.Key, prefix)
			if name != "" && !strings.Contains(name, "/") { out = append(out, Item{Name: name, Size: obj.Size}) }
		}
		if !result.IsTruncated { break }
		if result.NextContinuationToken == "" { return nil, errors.New("S3 odgovor je označen kao nepotpun bez continuation tokena") }
		token = result.NextContinuationToken
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Prefix != out[j].Prefix { return out[i].Prefix }
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (c *Client) Put(ctx context.Context, localPath, key string) error {
	st, err := os.Lstat(localPath)
	if err != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 { return errors.New("lokalna datoteka nije obična datoteka") }
	if st.Size() > 5<<30 { return errors.New("S3 upload veći od 5 GiB zahtijeva multipart podršku") }
	data, err := os.ReadFile(localPath)
	if err != nil { return err }
	_, err = c.do(ctx, http.MethodPut, strings.TrimPrefix(key, "/"), nil, data, http.Header{"Content-Type": {"application/octet-stream"}})
	return err
}

func (c *Client) Get(ctx context.Context, key, localPath string) error {
	data, err := c.do(ctx, http.MethodGet, strings.TrimPrefix(key, "/"), nil, nil, nil)
	if err != nil { return err }
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0700); err != nil { return err }
	f, err := os.CreateTemp(dir, ".byftp-s3-*.part")
	if err != nil { return err }
	tmp := f.Name()
	cleanup := func() { _ = f.Close(); _ = os.Remove(tmp) }
	if err := f.Chmod(0600); err != nil { cleanup(); return err }
	if _, err := f.Write(data); err != nil { cleanup(); return err }
	if err := f.Sync(); err != nil { cleanup(); return err }
	if err := f.Close(); err != nil { _ = os.Remove(tmp); return err }
	if err := os.Rename(tmp, localPath); err != nil { _ = os.Remove(tmp); return err }
	return nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.do(ctx, http.MethodDelete, strings.TrimPrefix(key, "/"), nil, nil, nil)
	return err
}

func (c *Client) Mkdir(ctx context.Context, prefix string) error {
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix == "" { return errors.New("S3 prefix je prazan") }
	if !strings.HasSuffix(prefix, "/") { prefix += "/" }
	_, err := c.do(ctx, http.MethodPut, prefix, nil, nil, http.Header{"Content-Type": {"application/x-directory"}})
	return err
}

func (c *Client) Rename(ctx context.Context, oldKey, newKey string) error {
	oldKey, newKey = strings.TrimPrefix(oldKey, "/"), strings.TrimPrefix(newKey, "/")
	if oldKey == "" || newKey == "" { return errors.New("S3 ključ je prazan") }
	source := "/" + c.cfg.Bucket + "/" + strings.Join(strings.Split(oldKey, "/"), "/")
	if _, err := c.do(ctx, http.MethodPut, newKey, nil, nil, http.Header{"x-amz-copy-source": {source}}); err != nil { return err }
	return c.Delete(ctx, oldKey)
}
