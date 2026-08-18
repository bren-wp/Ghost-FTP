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
	"hash"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"brendigo.com/byftp/internal/platform"
)

const (
	maxControlResponse = 16 << 20
	maxSinglePutBytes   = int64(5_000_000_000)
)

type Config struct {
	Endpoint     string
	Region       string
	AccessKey    string
	SecretKey    string
	SessionToken string
	Bucket       string
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
	if strings.ContainsAny(cfg.AccessKey, "\x00\r\n") || strings.ContainsAny(cfg.SecretKey, "\x00\r\n") || strings.ContainsAny(cfg.SessionToken, "\x00\r\n") {
		return nil, errors.New("S3 vjerodajnica sadrži nedopuštene znakove")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Privatnosno fail-closed: S3 ide izravno na endpoint koji je korisnik
	// upisao. Ne nasljeđujemo HTTP(S)_PROXY koji bi endpoint/objektne putanje
	// mogao poslati posredniku bez jasne postavke u ByFTP-u.
	transport.Proxy = nil
	transport.ResponseHeaderTimeout = 60 * time.Second
	return &Client{cfg: cfg, http: &http.Client{Transport: transport}, now: time.Now}, nil
}

func loopbackHTTP(u *url.URL) bool {
	if u == nil || u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
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

func hashReader(r io.Reader) (string, error) {
	var h hash.Hash = sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func awsURIEncode(value string, encodeSlash bool) string {
	const hexUpper = "0123456789ABCDEF"
	var b strings.Builder
	for _, c := range []byte(value) {
		unreserved := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~'
		if unreserved || (c == '/' && !encodeSlash) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hexUpper[c>>4])
		b.WriteByte(hexUpper[c&0x0f])
	}
	return b.String()
}

func encodePath(value string) string {
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return awsURIEncode(value, false)
}

func canonicalQuery(values url.Values) string {
	pairs := make([]string, 0)
	for key, vals := range values {
		encodedKey := awsURIEncode(key, true)
		if len(vals) == 0 {
			pairs = append(pairs, encodedKey+"=")
			continue
		}
		for _, value := range vals {
			pairs = append(pairs, encodedKey+"="+awsURIEncode(value, true))
		}
	}
	// AWS zahtijeva sortiranje nakon URI enkodiranja.
	sort.Strings(pairs)
	return strings.Join(pairs, "&")
}

func (c *Client) objectURL(key string, query url.Values) (*url.URL, error) {
	base, err := url.Parse(c.cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	plainPath := strings.TrimSuffix(base.Path, "/") + "/" + c.cfg.Bucket
	if key != "" {
		plainPath += "/" + strings.TrimPrefix(key, "/")
	}
	base.Path = plainPath
	base.RawPath = encodePath(plainPath)
	base.RawQuery = canonicalQuery(query)
	return base, nil
}

func (c *Client) signedRequestReader(ctx context.Context, method, key string, query url.Values, body io.Reader, payloadHash string, contentLength int64, extra http.Header) (*http.Request, error) {
	u, err := c.objectURL(key, query)
	if err != nil {
		return nil, err
	}
	if body == nil {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}
	for k, vals := range extra {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	now := c.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if c.cfg.SessionToken != "" {
		req.Header.Set("x-amz-security-token", c.cfg.SessionToken)
	}

	signedNames := []string{"host"}
	for k := range req.Header {
		lk := strings.ToLower(k)
		if lk != "authorization" {
			signedNames = append(signedNames, lk)
		}
	}
	sort.Strings(signedNames)
	seen := map[string]bool{}
	var canonicalHeaders strings.Builder
	finalNames := make([]string, 0, len(signedNames))
	for _, name := range signedNames {
		if seen[name] {
			continue
		}
		seen[name] = true
		value := ""
		if name == "host" {
			value = req.URL.Host
		} else {
			values := req.Header.Values(name)
			for i := range values {
				values[i] = strings.Join(strings.Fields(values[i]), " ")
			}
			value = strings.Join(values, ",")
		}
		canonicalHeaders.WriteString(name + ":" + value + "\n")
		finalNames = append(finalNames, name)
	}
	signedHeaderNames := strings.Join(finalNames, ";")
	canonicalRequest := strings.Join([]string{
		method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders.String(),
		signedHeaderNames,
		payloadHash,
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

func (c *Client) signedRequest(ctx context.Context, method, key string, query url.Values, body []byte, extra http.Header) (*http.Request, error) {
	return c.signedRequestReader(ctx, method, key, query, bytes.NewReader(body), hashHex(body), int64(len(body)), extra)
}

func readControlResponse(resp *http.Response) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxControlResponse+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxControlResponse {
		return nil, errors.New("S3 kontrolni odgovor je prevelik")
	}
	return data, nil
}

func responseError(resp *http.Response) error {
	data, _ := readControlResponse(resp)
	text := strings.TrimSpace(string(data))
	if len(text) > 4096 {
		text = text[:4096]
	}
	if text == "" {
		text = "poslužitelj nije vratio dodatno objašnjenje"
	}
	return fmt.Errorf("S3 zahtjev nije uspio (%s): %s", resp.Status, text)
}

func (c *Client) do(ctx context.Context, method, key string, query url.Values, body []byte, extra http.Header) ([]byte, error) {
	req, err := c.signedRequest(ctx, method, key, query, body, extra)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, responseError(resp)
	}
	return readControlResponse(resp)
}

type listResult struct {
	Contents []struct {
		Key  string `xml:"Key"`
		Size int64  `xml:"Size"`
	} `xml:"Contents"`
	Prefixes []struct {
		Prefix string `xml:"Prefix"`
	} `xml:"CommonPrefixes"`
	NextContinuationToken string `xml:"NextContinuationToken"`
	IsTruncated           bool   `xml:"IsTruncated"`
}

func (c *Client) List(ctx context.Context, prefix string) ([]Item, error) {
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	var out []Item
	token := ""
	for pages := 0; pages < 1000; pages++ {
		q := url.Values{"list-type": {"2"}, "delimiter": {"/"}, "prefix": {prefix}, "max-keys": {"1000"}}
		if token != "" {
			q.Set("continuation-token", token)
		}
		data, err := c.do(ctx, http.MethodGet, "", q, nil, nil)
		if err != nil {
			return nil, err
		}
		var result listResult
		if err := xml.Unmarshal(data, &result); err != nil {
			return nil, errors.New("S3 odgovor nije valjan XML")
		}
		for _, p := range result.Prefixes {
			name := strings.TrimSuffix(strings.TrimPrefix(p.Prefix, prefix), "/")
			if name != "" {
				out = append(out, Item{Name: name, Prefix: true})
			}
		}
		for _, obj := range result.Contents {
			name := strings.TrimPrefix(obj.Key, prefix)
			if name != "" && !strings.Contains(name, "/") {
				out = append(out, Item{Name: name, Size: obj.Size})
			}
		}
		if !result.IsTruncated {
			break
		}
		if result.NextContinuationToken == "" {
			return nil, errors.New("S3 odgovor je označen kao nepotpun bez continuation tokena")
		}
		token = result.NextContinuationToken
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Prefix != out[j].Prefix {
			return out[i].Prefix
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (c *Client) objectExists(ctx context.Context, key string) (bool, error) {
	req, err := c.signedRequest(ctx, http.MethodHead, strings.TrimPrefix(key, "/"), nil, nil, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	return false, responseError(resp)
}

func (c *Client) Put(ctx context.Context, localPath, key string) error {
	st, err := os.Lstat(localPath)
	if err != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("lokalna datoteka nije obična datoteka")
	}
	if st.Size() > maxSinglePutBytes {
		return errors.New("S3 upload veći od 5 GB zahtijeva multipart podršku")
	}
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	payloadHash, err := hashReader(f)
	if err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	req, err := c.signedRequestReader(ctx, http.MethodPut, strings.TrimPrefix(key, "/"), nil, f, payloadHash, st.Size(), http.Header{"Content-Type": {"application/octet-stream"}})
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp)
	}
	_, err = io.Copy(io.Discard, io.LimitReader(resp.Body, maxControlResponse+1))
	return err
}

func (c *Client) Get(ctx context.Context, key, localPath string) error {
	req, err := c.signedRequest(ctx, http.MethodGet, strings.TrimPrefix(key, "/"), nil, nil, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp)
	}
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".byftp-s3-*.part")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}
	if err := f.Chmod(0600); err != nil {
		cleanup()
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		cleanup()
		return err
	}
	if err := f.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := platform.RenameNoReplace(tmp, localPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.do(ctx, http.MethodDelete, strings.TrimPrefix(key, "/"), nil, nil, nil)
	return err
}

func (c *Client) Mkdir(ctx context.Context, prefix string) error {
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix == "" {
		return errors.New("S3 prefix je prazan")
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	_, err := c.do(ctx, http.MethodPut, prefix, nil, nil, http.Header{"Content-Type": {"application/x-directory"}})
	return err
}

func (c *Client) Rename(ctx context.Context, oldKey, newKey string) error {
	oldKey, newKey = strings.TrimPrefix(oldKey, "/"), strings.TrimPrefix(newKey, "/")
	if oldKey == "" || newKey == "" || oldKey == newKey {
		return errors.New("neispravan S3 izvor ili odredište")
	}
	exists, err := c.objectExists(ctx, newKey)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("odredišni S3 objekt već postoji")
	}
	// S3 nema atomski rename. ByFTP radi server-side CopyObject pa DeleteObject
	// tek nakon uspješnog copyja; provjera odredišta smanjuje rizik prepisivanja,
	// ali konkurentna izmjena između HEAD i COPY ostaje ograničenje S3 API-ja.
	source := encodePath("/" + c.cfg.Bucket + "/" + oldKey)
	if _, err := c.do(ctx, http.MethodPut, newKey, nil, nil, http.Header{"x-amz-copy-source": {source}}); err != nil {
		return err
	}
	return c.Delete(ctx, oldKey)
}
