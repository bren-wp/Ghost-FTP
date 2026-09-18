package external

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
)

var ErrUntrustedURL = errors.New("untrusted external URL")

func OpenReleasePage(value string) error {
	return open(value, []string{"github.com"})
}

func OpenPremiumPage() error {
	return open(brand.PremiumURL, []string{"ghostftp.com", "www.ghostftp.com"})
}

func OpenWebsite() error {
	return open(brand.WebsiteURL, []string{"ghostftp.com", "www.ghostftp.com"})
}

func open(value string, allowedHosts []string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return ErrUntrustedURL
	}
	host := strings.ToLower(parsed.Hostname())
	trusted := false
	for _, allowed := range allowedHosts {
		if host == allowed {
			trusted = true
			break
		}
	}
	if !trusted {
		return ErrUntrustedURL
	}
	if parsed.User != nil {
		return ErrUntrustedURL
	}

	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", parsed.String())
	case "darwin":
		command = exec.Command("open", parsed.String())
	case "linux":
		command = exec.Command("xdg-open", parsed.String())
	default:
		return fmt.Errorf("open external URL: unsupported platform %s", runtime.GOOS)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("open external URL: %w", err)
	}
	return nil
}

func TrustedPremiumURL() string { return brand.PremiumURL }
func TrustedReleaseURL() string { return brand.ReleaseURL }
