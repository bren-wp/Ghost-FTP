package i18n

import (
	"strings"
	"testing"
)

func TestSFTPTrustGuidanceCoversEverySupportedLanguage(t *testing.T) {
	languages := Languages()
	if len(sftpTrustGuidance) != len(languages) {
		t.Fatalf("SFTP trust guidance covers %d languages; registry contains %d", len(sftpTrustGuidance), len(languages))
	}

	const fingerprint = "SHA256:GhostFTPFirstContactTest"
	for _, language := range languages {
		guidance, ok := sftpTrustGuidance[language.Code]
		if !ok {
			t.Fatalf("missing SFTP trust guidance for %s", language.Code)
		}
		if strings.Count(guidance.dialog, "%s") != 1 {
			t.Fatalf("dialog guidance for %s must contain exactly one fingerprint placeholder", language.Code)
		}
		if strings.TrimSpace(guidance.terminal) == "" || strings.Contains(guidance.terminal, "%s") {
			t.Fatalf("terminal guidance for %s is invalid", language.Code)
		}
		if catalogs[language.Code]["sftp.trust_body"] != guidance.dialog {
			t.Fatalf("dialog guidance override is not active for %s", language.Code)
		}
		if catalogs[language.Code]["terminal.trust"] != guidance.terminal {
			t.Fatalf("terminal guidance override is not active for %s", language.Code)
		}

		rendered := T(language.Code, "sftp.trust_body", fingerprint)
		if strings.Count(rendered, fingerprint) != 1 {
			t.Fatalf("rendered SFTP trust dialog for %s does not show the fingerprint exactly once", language.Code)
		}
		if strings.Contains(rendered, "%!") || strings.Contains(rendered, "%s") {
			t.Fatalf("rendered SFTP trust dialog for %s contains a formatting artifact", language.Code)
		}
	}
}

func TestDefaultSFTPTrustGuidanceRequiresIndependentVerification(t *testing.T) {
	dialog := sftpTrustGuidance[DefaultLanguage].dialog
	terminal := sftpTrustGuidance[DefaultLanguage].terminal
	for name, value := range map[string]string{"dialog": dialog, "terminal": terminal} {
		lower := strings.ToLower(value)
		for _, required := range []string{"independent", "trusted", "exact"} {
			if !strings.Contains(lower, required) {
				t.Fatalf("default %s guidance is missing security requirement %q", name, required)
			}
		}
	}
}
