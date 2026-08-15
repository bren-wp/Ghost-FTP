package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"brendigo.com/byftp/internal/brand"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
)

//go:embed all:payload
var payload embed.FS

const maxPayloadFileSize = 128 << 20
const maxPayloadManifestSize = 64 << 10

func readPayloadFiles() ([]byte, []byte, error) {
	data, err := payload.ReadFile("payload/payload.zip")
	if err != nil {
		return nil, nil, errors.New("komprimirani instalacijski payload nije dostupan")
	}
	return parsePayload(data)
}

func parsePayload(data []byte) ([]byte, []byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, errors.New("instalacijski payload nije ispravan ZIP")
	}
	files := map[string][]byte{}
	var manifestData []byte
	for _, f := range zr.File {
		switch f.Name {
		case "ByFTP.exe", "Uninstall.exe":
			if _, exists := files[f.Name]; exists {
				return nil, nil, errors.New("instalacijski payload sadrži dupliciranu datoteku")
			}
			if f.UncompressedSize64 == 0 || f.UncompressedSize64 > maxPayloadFileSize {
				return nil, nil, errors.New("instalacijski payload ima neispravnu veličinu")
			}
			b, err := readZipEntry(f, int(f.UncompressedSize64))
			if err != nil {
				return nil, nil, err
			}
			files[f.Name] = b
		case "manifest.json":
			if manifestData != nil {
				return nil, nil, errors.New("instalacijski payload sadrži duplicirani manifest")
			}
			if f.UncompressedSize64 == 0 || f.UncompressedSize64 > maxPayloadManifestSize {
				return nil, nil, errors.New("instalacijski manifest ima neispravnu veličinu")
			}
			manifestData, err = readZipEntry(f, int(f.UncompressedSize64))
			if err != nil {
				return nil, nil, err
			}
		default:
			return nil, nil, errors.New("instalacijski payload sadrži neočekivanu datoteku")
		}
	}
	app, appOK := files["ByFTP.exe"]
	un, unOK := files["Uninstall.exe"]
	if !appOK || !unOK || manifestData == nil {
		return nil, nil, errors.New("instalacijski payload ne sadrži sve obavezne datoteke")
	}
	if err := validatePayloadManifest(manifestData, files); err != nil {
		return nil, nil, err
	}
	return app, un, nil
}

func readZipEntry(f *zip.File, size int) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	b := make([]byte, size)
	n, readErr := io.ReadFull(r, b)
	closeErr := r.Close()
	if readErr != nil || n != len(b) {
		return nil, errors.New("instalacijski payload je nepotpun")
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return b, nil
}

type payloadManifest struct {
	Schema int `json:"schema"`
	Files  []struct {
		Name   string `json:"name"`
		Size   int    `json:"size"`
		SHA256 string `json:"sha256"`
	} `json:"files"`
}

func validatePayloadManifest(data []byte, files map[string][]byte) error {
	var manifest payloadManifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return errors.New("instalacijski manifest nije ispravan")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("instalacijski manifest sadrži višak podataka")
	}
	if manifest.Schema != 1 || len(manifest.Files) != 2 {
		return errors.New("instalacijski manifest nije podržan")
	}
	seen := map[string]bool{}
	for _, item := range manifest.Files {
		content, ok := files[item.Name]
		if !ok || seen[item.Name] || (item.Name != "ByFTP.exe" && item.Name != "Uninstall.exe") {
			return errors.New("instalacijski manifest ne odgovara paketu")
		}
		seen[item.Name] = true
		digest := fmt.Sprintf("%x", sha256.Sum256(content))
		if item.Size != len(content) || !strings.EqualFold(item.SHA256, digest) {
			return errors.New("provjera integriteta instalacijskog paketa nije uspjela")
		}
	}
	if !seen["ByFTP.exe"] || !seen["Uninstall.exe"] {
		return errors.New("instalacijski manifest nije potpun")
	}
	return nil
}

var version = "2.12.0"

const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ByFTP`

func stageVerified(path string, data []byte) (string, error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".byftp-install-*.tmp")
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}
	if err = f.Chmod(0755); err != nil {
		cleanup()
		return "", err
	}
	if _, err = f.Write(data); err != nil {
		cleanup()
		return "", err
	}
	if err = f.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	written, err := os.ReadFile(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if sha256.Sum256(data) != sha256.Sum256(written) {
		_ = os.Remove(tmp)
		return "", errors.New("provjera integriteta instalacijske datoteke nije uspjela")
	}
	return tmp, nil
}

func installFile(path string, data []byte) error {
	tmp, err := stageVerified(path, data)
	if err != nil {
		return err
	}
	if err := platform.ReplaceFile(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func register(appPath, uninstallPath, dir string) error {
	entries := []struct{ key, name, value string }{
		{uninstallKey, "DisplayName", brand.ProductFull},
		{uninstallKey, "DisplayVersion", version},
		{uninstallKey, "Publisher", brand.Company},
		{uninstallKey, "InstallLocation", dir},
		{uninstallKey, "DisplayIcon", appPath},
		{uninstallKey, "UninstallString", `"` + uninstallPath + `"`},
		{`Software\Microsoft\Windows\CurrentVersion\App Paths\ByFTP.exe`, "", appPath},
	}
	for _, e := range entries {
		if err := platform.SetRegistryString(e.key, e.name, e.value); err != nil {
			return fmt.Errorf("Windows registracija nije uspjela (%s): %w", e.name, err)
		}
	}
	if err := platform.SetRegistryString(uninstallKey, "InstallDate", time.Now().Format("20060102")); err != nil {
		return fmt.Errorf("Windows registracija nije uspjela (InstallDate): %w", err)
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoModify", 1); err != nil {
		return fmt.Errorf("Windows registracija nije uspjela (NoModify): %w", err)
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoRepair", 1); err != nil {
		return fmt.Errorf("Windows registracija nije uspjela (NoRepair): %w", err)
	}
	return nil
}

func main() {
	platform.HardenProcessPrivacy()
	defer func() {
		if recover() != nil {
			platform.ErrorDialog(brand.ProductFull+" — "+brand.Company, "Radnja nije dovršena", "Instalacija nije dovršena. Ponovno pokrenite računalo i pokušajte ponovno.")
		}
	}()
	content := brand.Company + "\n" + brand.Website + "\n" + brand.Support + "\n\n" +
		"ByFTP će se instalirati za vaš Windows korisnički račun i bit će dostupan iz izbornika Start."
	if !platform.ConfirmDialog(brand.ProductName+" — Instalacija", "Instalirati "+brand.ProductFull+" "+version+"?", content) {
		return
	}

	dir, err := platform.InstallDir()
	if err != nil {
		platform.ErrorDialog("ByFTP — Instalacija", "Instalacija nije moguća", "Korisnička mapa sustava Windows nije dostupna.")
		return
	}
	localAppData, err := platform.LocalAppData()
	if err != nil || security.EnsureNoRedirectDirectory(localAppData, dir) != nil {
		platform.ErrorDialog("ByFTP — Instalacija", "Instalacijska mapa nije sigurna", "Uklonite preusmjeravanje ByFTP instalacijske mape i pokušajte ponovno.")
		return
	}
	if err := ensureInstallDir(dir); err != nil {
		platform.ErrorDialog("ByFTP — Instalacija", "Instalacija nije pokrenuta", "Instalacijsku mapu nije moguće pripremiti. Provjerite slobodan prostor i dopuštenja pa pokušajte ponovno.")
		return
	}

	app, un, err := readPayloadFiles()
	if err != nil {
		platform.ErrorDialog("ByFTP — Instalacija", "Instalacijski paket nije ispravan", "Preuzmite novu kopiju instalacijskog paketa i pokušajte ponovno.")
		return
	}

	appPath := filepath.Join(dir, "ByFTP.exe")
	unPath := filepath.Join(dir, "Uninstall.exe")
	appBackup, err := backupExisting(appPath)
	if err != nil {
		platform.ErrorDialog("ByFTP — Instalacija", "Nadogradnja nije pokrenuta", "Postojeću instalaciju nije moguće pripremiti za nadogradnju. Zatvorite ByFTP i pokušajte ponovno.")
		return
	}
	unBackup, err := backupExisting(unPath)
	if err != nil {
		appBackup.cleanup()
		platform.ErrorDialog("ByFTP — Instalacija", "Nadogradnja nije pokrenuta", "Postojeću instalaciju nije moguće pripremiti za nadogradnju. Zatvorite ByFTP i pokušajte ponovno.")
		return
	}
	registryBackup, err := captureRegistrySnapshot()
	if err != nil {
		appBackup.cleanup()
		unBackup.cleanup()
		platform.ErrorDialog("ByFTP — Instalacija", "Nadogradnja nije pokrenuta", "Postojeće postavke instalacije nije moguće sigurno pripremiti. Pokušajte ponovno.")
		return
	}
	rollback := func() error {
		var errs []error
		if err := unBackup.rollback(); err != nil {
			errs = append(errs, err)
		}
		if err := appBackup.rollback(); err != nil {
			errs = append(errs, err)
		}
		if err := registryBackup.restore(); err != nil {
			errs = append(errs, err)
		}
		if !appBackup.existed() && !unBackup.existed() {
			if err := platform.RemoveShortcuts(); err != nil {
				errs = append(errs, err)
			}
			if err := platform.DeleteRegistryKey(uninstallKey); err != nil {
				errs = append(errs, err)
			}
			if err := platform.DeleteRegistryKey(`Software\Microsoft\Windows\CurrentVersion\App Paths\ByFTP.exe`); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	rollbackMessage := func() string {
		if err := rollback(); err != nil {
			return " Prethodnu instalaciju nije bilo moguće potpuno vratiti; ponovno pokrenite Windows prije novog pokušaja."
		}
		return ""
	}
	if err = installFile(appPath, app); err != nil {
		extra := rollbackMessage()
		platform.ErrorDialog("ByFTP — Instalacija", "ByFTP nije moguće instalirati", "Zatvorite pokrenuti ByFTP i pokušajte ponovno."+extra)
		return
	}
	if err = installFile(unPath, un); err != nil {
		extra := rollbackMessage()
		platform.ErrorDialog("ByFTP — Instalacija", "Instalacija nije dovršena", "Potrebne datoteke nije moguće spremiti. Pokušajte ponovno."+extra)
		return
	}
	if err = register(appPath, unPath, dir); err != nil {
		extra := rollbackMessage()
		platform.ErrorDialog("ByFTP — Instalacija", "Instalacija nije dovršena", "Windows nije uspio dovršiti postavljanje aplikacije. Pokušajte ponovno."+extra)
		return
	}
	appBackup.cleanup()
	unBackup.cleanup()
	shortcutWarning := ""
	if err = platform.CreateShortcuts(appPath); err != nil {
		shortcutWarning = "\n\nPrečac nije moguće izraditi. ByFTP možete pokrenuti iz instalacijske mape."
	}

	if platform.ConfirmDialog(
		brand.ProductName+" — Instalacija",
		"Instalacija je uspješno dovršena",
		"ByFTP je spreman za korištenje."+shortcutWarning+"\n\nPokrenuti ByFTP sada?",
	) {
		cmd := exec.Command(appPath)
		if err := cmd.Start(); err != nil {
			platform.ErrorDialog("ByFTP — Instalacija", "ByFTP je instaliran", "Aplikaciju nije moguće automatski pokrenuti. Pokrenite ByFTP iz izbornika Start.")
			return
		}
	}
}
