package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

type legacyUninstallerProof struct {
	path    string
	digest  string
	warning string
	owned   bool
}

func sameCanonicalLegacyCommand(raw, expectedPath string) bool {
	command := strings.TrimSpace(raw)
	if len(command) >= 2 && command[0] == '"' && command[len(command)-1] == '"' {
		command = command[1 : len(command)-1]
	}
	if command == "" || !filepath.IsAbs(command) || !filepath.IsAbs(expectedPath) {
		return false
	}

	actual := filepath.Clean(command)
	expected := filepath.Clean(expectedPath)
	return strings.EqualFold(actual, expected)
}

func captureLegacyUninstallerProof(dir string, snapshot registrySnapshot) legacyUninstallerProof {
	legacyPath := filepath.Join(dir, "Uninstall.exe")
	registered, ok := snapshot.stringValue(uninstallKey, "UninstallString")
	if !ok || !sameCanonicalLegacyCommand(registered, legacyPath) {
		if _, err := os.Lstat(legacyPath); err == nil {
			return legacyUninstallerProof{
				warning: "A legacy Uninstall.exe file was preserved because the previous Installed Apps registration did not prove Ghost FTP ownership.",
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return legacyUninstallerProof{
				warning: "The legacy Uninstall.exe path could not be checked safely and was not removed.",
			}
		}
		return legacyUninstallerProof{}
	}

	digest, err := platform.VerifiedRegularFileSHA256(legacyPath)
	if errors.Is(err, os.ErrNotExist) {
		return legacyUninstallerProof{}
	}
	if err != nil {
		return legacyUninstallerProof{
			warning: "A registered legacy Uninstall.exe file was preserved because its file identity could not be verified safely.",
		}
	}
	return legacyUninstallerProof{path: legacyPath, digest: digest, owned: true}
}

func cleanupLegacyUninstaller(proof legacyUninstallerProof) string {
	if proof.warning != "" {
		return "\n\n" + proof.warning
	}
	if !proof.owned || proof.path == "" || proof.digest == "" {
		return ""
	}

	removed, err := platform.RemoveVerifiedRegularFileMatchingSHA256(proof.path, proof.digest)
	if err != nil {
		return "\n\nThe legacy Uninstall.exe file was preserved because it changed after ownership verification or could not be removed safely."
	}
	if !removed {
		return ""
	}
	return ""
}
