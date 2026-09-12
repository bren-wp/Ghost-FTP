package i18n

import (
	"strings"
	"testing"
)

func TestSFTPTrustGuidanceCoversEverySupportedLanguage(t *testing.T) {
	languages := Languages()
	if len(sftpTrustGuidance) != len(languages) {
		t.Fatalf("SFTP trust guidance has %d languages; runtime exposes %d", len(sftpTrustGuidance), len(languages))
	}

	const fingerprint = "SHA256:GhostFTPTrustBoundaryTest"
	for _, language := range languages {
		guidance, ok := sftpTrustGuidance[language.Code]
		if !ok {
			t.Errorf("missing SFTP trust guidance for %s", language.Code)
			continue
		}
		if strings.TrimSpace(guidance.body) == "" || strings.TrimSpace(guidance.terminal) == "" {
			t.Errorf("empty SFTP trust guidance for %s", language.Code)
			continue
		}

		body := T(language.Code, "sftp.trust_body", fingerprint)
		if !strings.Contains(body, fingerprint) {
			t.Errorf("SFTP trust body for %s does not preserve the exact fingerprint", language.Code)
		}
		if strings.Contains(body, "%!") || strings.Contains(body, "%s") {
			t.Errorf("SFTP trust body for %s has an invalid formatting result: %q", language.Code, body)
		}

		terminal := T(language.Code, "terminal.trust")
		if terminal == "terminal.trust" || strings.TrimSpace(terminal) == "" {
			t.Errorf("terminal trust guidance for %s is unavailable", language.Code)
		}
	}
}

func TestSFTPTrustGuidanceExplicitlyRequiresIndependentVerification(t *testing.T) {
	english := T("en", "sftp.trust_body", "SHA256:test")
	if !strings.Contains(english, "independent trusted channel") || !strings.Contains(english, "matches exactly") {
		t.Fatalf("English SFTP trust guidance lost the independent-channel or exact-match requirement: %q", english)
	}

	croatian := T("hr", "sftp.trust_body", "SHA256:test")
	if !strings.Contains(croatian, "neovisnog pouzdanog kanala") || !strings.Contains(croatian, "potpuno podudara") {
		t.Fatalf("Croatian SFTP trust guidance lost the independent-channel or exact-match requirement: %q", croatian)
	}
}

func TestSFTPTrustGuidanceDoesNotChangeCatalogShape(t *testing.T) {
	if err := ValidateCatalogs(); err != nil {
		t.Fatalf("catalog validation failed after SFTP trust guidance hardening: %v", err)
	}
}
