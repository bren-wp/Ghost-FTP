package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
)

const maxResponseBytes = 128 * 1024

var (
	ErrInvalidCurrentVersion = errors.New("invalid current version")
	ErrInvalidReleaseVersion = errors.New("invalid release version")
	ErrUntrustedReleaseURL   = errors.New("untrusted release URL")
)

type Result struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	Available      bool
}

type Client struct {
	httpClient *http.Client
	apiURL     string
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return errors.New("too many redirects")
				}
				if req.URL.Scheme != "https" {
					return errors.New("update redirect must use https")
				}
				return nil
			},
		},
		apiURL: brand.ReleaseAPIURL,
	}
}

func NewForTest(client *http.Client, apiURL string) *Client {
	return &Client{httpClient: client, apiURL: apiURL}
}

func (c *Client) Check(ctx context.Context, currentVersion string) (Result, error) {
	current, err := parseVersion(currentVersion)
	if err != nil {
		return Result{}, ErrInvalidCurrentVersion
	}
	if c == nil || c.httpClient == nil {
		return Result{}, errors.New("update client is unavailable")
	}
	api, err := url.Parse(c.apiURL)
	if err != nil || api.Scheme != "https" || api.Hostname() == "" {
		return Result{}, errors.New("invalid update endpoint")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Ghost-FTP/"+currentVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("check updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("check updates: http %d", resp.StatusCode)
	}

	var payload struct {
		TagName    string `json:"tag_name"`
		HTMLURL    string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes))
	if err := decoder.Decode(&payload); err != nil {
		return Result{}, fmt.Errorf("check updates: decode: %w", err)
	}
	if payload.Draft || payload.Prerelease {
		return Result{}, errors.New("latest release is not a stable public release")
	}

	latestText := normalizeTag(payload.TagName)
	latest, err := parseVersion(latestText)
	if err != nil {
		return Result{}, ErrInvalidReleaseVersion
	}
	releaseURL, err := trustedReleaseURL(payload.HTMLURL)
	if err != nil {
		return Result{}, err
	}

	return Result{
		CurrentVersion: current.String(),
		LatestVersion:  latest.String(),
		ReleaseURL:     releaseURL,
		Available:      latest.Compare(current) > 0,
	}, nil
}

type version struct {
	major int
	minor int
	patch int
}

func parseVersion(value string) (version, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return version{}, errors.New("version must contain three numeric parts")
	}
	values := [3]int{}
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0' && part != "0") {
			return version{}, errors.New("invalid numeric version part")
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 999999 {
			return version{}, errors.New("invalid numeric version part")
		}
		values[i] = n
	}
	return version{major: values[0], minor: values[1], patch: values[2]}, nil
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func (v version) Compare(other version) int {
	if v.major != other.major {
		if v.major < other.major {
			return -1
		}
		return 1
	}
	if v.minor != other.minor {
		if v.minor < other.minor {
			return -1
		}
		return 1
	}
	if v.patch < other.patch {
		return -1
	}
	if v.patch > other.patch {
		return 1
	}
	return 0
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "ghostftp-v")
	tag = strings.TrimPrefix(tag, "v")
	return tag
}

func trustedReleaseURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" {
		return "", ErrUntrustedReleaseURL
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "github.com" {
		return "", ErrUntrustedReleaseURL
	}
	if !strings.HasPrefix(parsed.EscapedPath(), "/bren-wp/Ghost-FTP/releases/") {
		return "", ErrUntrustedReleaseURL
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
