//go:build linux

package desktop

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestLinuxInfoOverlayLayoutStaysInsideMinimumWindow(t *testing.T) {
	for _, kind := range []linuxInfoOverlayKind{linuxInfoOverlayConnection, linuxInfoOverlayAbout} {
		layout := buildLinuxInfoOverlayLayout(premiumMinWidth, premiumMinHeight, kind)
		if layout.panel.left < 0 || layout.panel.top < 0 ||
			layout.panel.right > premiumMinWidth || layout.panel.bottom > premiumMinHeight {
			t.Fatalf("overlay %d outside minimum window: %+v", kind, layout.panel)
		}
		if !layout.panel.contains(layout.close.left, layout.close.top) ||
			!layout.panel.contains(layout.close.right-1, layout.close.bottom-1) {
			t.Fatalf("overlay %d close button outside panel: panel=%+v close=%+v", kind, layout.panel, layout.close)
		}
	}
}

func TestLinuxConnectionInfoIsUsefulAndPrivacySafe(t *testing.T) {
	u := &linuxDesktop{
		version:       "9.9.9",
		language:      "en",
		protocol:      "ftps",
		host:          "private.example.test",
		username:      "private-user",
		password:      "private-password",
		keyPath:       "/home/user/.ssh/private-key",
		passphrase:    "private-passphrase",
		connected:     true,
		remoteCurrent: "/public_html",
		transferJobs: []model.TransferJob{
			{Status: "queued"},
			{Status: "done"},
		},
	}
	joined := strings.Join(linuxConnectionInfoLines(u), "\n")
	for _, want := range []string{
		"FTPS",
		"Transfer Queue: 1",
		"No telemetry or tracking",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("connection info missing %q:\n%s", want, joined)
		}
	}
	u.infoOverlay = linuxInfoOverlayConnection
	_, heading := u.linuxInfoOverlayTitleAndHeading()
	if heading != brand.ProductName+" 9.9.9" {
		t.Fatalf("connection info heading = %q", heading)
	}

	for _, secret := range []string{
		u.host,
		u.username,
		u.password,
		u.keyPath,
		u.passphrase,
		u.remoteCurrent,
	} {
		if strings.Contains(joined, secret) {
			t.Fatalf("connection info leaked secret/private connection detail %q:\n%s", secret, joined)
		}
	}
}

func TestLinuxAboutInfoUsesPublicProductMetadata(t *testing.T) {
	u := &linuxDesktop{version: "9.9.9", language: "en"}
	joined := strings.Join(linuxAboutInfoLines(u), "\n")
	for _, want := range []string{
		brand.ProductName + " 9.9.9",
		"FTP / FTPS / SFTP",
		brand.Website,
		"No telemetry or tracking",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("about info missing %q:\n%s", want, joined)
		}
	}
}

func TestLinuxWrapForUIKeepsUTF8AndBoundedLines(t *testing.T) {
	for _, input := range []string{
		"Čuvanje sigurnih postavki bez telemetrije",
		"连接信息和隐私保护不应溢出固定宽度信息面板",
	} {
		lines := linuxWrapForUI(input, 14)
		if len(lines) < 2 {
			t.Fatalf("expected wrapped lines for %q, got %#v", input, lines)
		}
		for _, line := range lines {
			if !utf8.ValidString(line) {
				t.Fatalf("invalid UTF-8 wrapped line: %q", line)
			}
			if utf8.RuneCountInString(line) > 14 {
				t.Fatalf("wrapped line exceeds limit: %q", line)
			}
		}
	}
}

func TestLinuxInfoOverlayLifecycleBlocksConflictingSettingsSurface(t *testing.T) {
	u := &linuxDesktop{language: "en"}
	u.openLinuxInfoOverlay(linuxInfoOverlayAbout)
	if u.infoOverlay != linuxInfoOverlayAbout {
		t.Fatalf("info overlay = %d, want about", u.infoOverlay)
	}
	u.closeLinuxInfoOverlay()
	if u.infoOverlay != linuxInfoOverlayNone {
		t.Fatalf("info overlay did not close: %d", u.infoOverlay)
	}
	u.settingsOpen = true
	u.openLinuxInfoOverlay(linuxInfoOverlayConnection)
	if u.infoOverlay != linuxInfoOverlayNone {
		t.Fatalf("info overlay opened over settings: %d", u.infoOverlay)
	}
}
