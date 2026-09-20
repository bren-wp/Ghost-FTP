package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const version = "2.1.1-rc.10"

//go:embed site/* site/assets/* payload/GhostFTP.exe
var bundle embed.FS

var srv *http.Server
var once sync.Once

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--silent" {
		accepted := false
		for _, arg := range os.Args[2:] {
			if arg == "--accept-license" {
				accepted = true
				break
			}
		}
		if !accepted {
			fmt.Fprintln(os.Stderr, "Ghost FTP Setup: --silent requires --accept-license")
			os.Exit(2)
		}
		if err := install(installOptions{DesktopShortcut: true, StartMenuShortcut: true, RegisterApps: true}); err != nil {
			fmt.Fprintln(os.Stderr, "Ghost FTP Setup:", err)
			os.Exit(1)
		}
		return
	}
	site, err := fs.Sub(bundle, "site")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", secure(http.FileServer(http.FS(site))))
	mux.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "token": installerToken})
	})
	mux.HandleFunc("/api/install", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeInstallerMutation(w, r) {
			return
		}
		opts := installOptions{DesktopShortcut: true, StartMenuShortcut: true, RegisterApps: true}
		if !decodeInstallerJSON(w, r, &opts) {
			return
		}
		err := install(opts)
		if err != nil {
			writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Installation failed: " + err.Error()})
			return
		}
		msg := "Ghost FTP installed successfully."
		if opts.LaunchAfter {
			msg = "Ghost FTP installed successfully. Launching Ghost FTP…"
		}
		writeJSON(w, map[string]any{"ok": true, "message": msg})
		if opts.LaunchAfter {
			go launchInstalled(opts.InstallDir)
		}
	})
	mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeInstallerMutation(w, r) {
			return
		}
		writeJSON(w, map[string]any{"ok": true})
		go shutdown()
	})
	mux.HandleFunc("/api/window", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeInstallerMutation(w, r) {
			return
		}
		var body struct {
			Action string `json:"action"`
		}
		if !decodeInstallerJSON(w, r, &body) {
			return
		}
		writeJSON(w, map[string]any{"ok": handleWindowAction(body.Action)})
	})
	mux.HandleFunc("/api/browse-folder", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeInstallerMutation(w, r) {
			return
		}
		path, err := chooseInstallFolder()
		if err != nil || path == "" {
			writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "path": path, "error": errString(err)})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "path": path, "error": ""})
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { setInstallerSecurityHeaders(w); mux.ServeHTTP(w, r) }), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 60 * time.Second}
	url := "http://" + ln.Addr().String() + "/"
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server: %v", err)
		}
	}()
	if os.Getenv("GHOSTFTP_SETUP_HEADLESS") == "1" {
		fmt.Println(url)
	} else if err := openWindow(url); err != nil {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	<-time.After(30 * time.Minute)
	shutdown()
}

func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setInstallerSecurityHeaders(w)
		next.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, v any) {
	setInstallerSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	setInstallerSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func shutdown() {
	once.Do(func() {
		time.Sleep(350 * time.Millisecond)
		if srv != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = srv.Shutdown(ctx)
			cancel()
		}
		os.Exit(0)
	})
}

type installOptions struct {
	InstallDir        string `json:"installDir"`
	DesktopShortcut   bool   `json:"desktopShortcut"`
	StartMenuShortcut bool   `json:"startMenuShortcut"`
	RegisterApps      bool   `json:"registerApps"`
	LaunchAfter       bool   `json:"launchAfter"`
}

func install(opts installOptions) error {
	local := os.Getenv("LOCALAPPDATA")
	appdata := os.Getenv("APPDATA")
	user := os.Getenv("USERPROFILE")
	if local == "" || appdata == "" || user == "" {
		return fmt.Errorf("Windows user folders are unavailable")
	}
	payload, err := bundle.ReadFile("payload/GhostFTP.exe")
	if err != nil {
		return err
	}
	dir := strings.TrimSpace(opts.InstallDir)
	if dir == "" || strings.EqualFold(dir, `%LOCALAPPDATA%\Programs\Ghost FTP`) {
		dir = filepath.Join(local, "Programs", "Ghost FTP")
	}
	dir = filepath.Clean(os.ExpandEnv(dir))
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("installation folder must be an absolute path")
	}
	volumeRoot := filepath.Clean(filepath.VolumeName(dir) + `\`)
	if dir == volumeRoot || len(dir) <= len(volumeRoot)+3 {
		return fmt.Errorf("installation folder is not safe")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	probe := filepath.Join(dir, ".ghostftp-write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0600); err != nil {
		return fmt.Errorf("installation folder is not writable: %w", err)
	}
	_ = os.Remove(probe)
	exe := filepath.Join(dir, "GhostFTP.exe")
	tmp := exe + ".new"
	backup := exe + ".previous"
	if err := os.WriteFile(tmp, payload, 0755); err != nil {
		return err
	}
	_ = os.Remove(backup)
	if _, err := os.Stat(exe); err == nil {
		if err := os.Rename(exe, backup); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("preparing upgrade rollback: %w", err)
		}
	}
	if err := os.Rename(tmp, exe); err != nil {
		if _, oldErr := os.Stat(backup); oldErr == nil {
			_ = os.Rename(backup, exe)
		}
		return fmt.Errorf("committing application binary: %w", err)
	}
	hadPrevious := false
	if _, oldErr := os.Stat(backup); oldErr == nil {
		hadPrevious = true
	}
	rollback := func() {
		_ = os.Remove(exe)
		if hadPrevious {
			_ = os.Rename(backup, exe)
		}
	}
	// Product shortcuts are created through the Windows Script Host so no extra installer dependency is required.
	desktop := filepath.Join(user, "Desktop", "Ghost FTP.lnk")
	startDir := filepath.Join(appdata, "Microsoft", "Windows", "Start Menu", "Programs")
	_ = os.MkdirAll(startDir, 0755)
	start := filepath.Join(startDir, "Ghost FTP.lnk")
	uninstallLink := filepath.Join(startDir, "Uninstall Ghost FTP.lnk")
	if opts.DesktopShortcut {
		if err := shortcut(exe, desktop, ""); err != nil {
			rollback()
			return fmt.Errorf("creating desktop shortcut: %w", err)
		}
	} else {
		_ = os.Remove(desktop)
	}
	if opts.StartMenuShortcut {
		if err := shortcut(exe, start, ""); err != nil {
			rollback()
			return fmt.Errorf("creating Start Menu shortcut: %w", err)
		}
		if err := shortcut(exe, uninstallLink, "--uninstall"); err != nil {
			rollback()
			return fmt.Errorf("creating uninstall shortcut: %w", err)
		}
	} else {
		_ = os.Remove(start)
		_ = os.Remove(uninstallLink)
	}
	uninstall := fmt.Sprintf(`\"%s\" --uninstall`, exe)
	if opts.RegisterApps {
		args := [][]string{
			{"add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/v", "DisplayName", "/t", "REG_SZ", "/d", "Ghost FTP", "/f"},
			{"add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/v", "DisplayVersion", "/t", "REG_SZ", "/d", version, "/f"},
			{"add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/v", "Publisher", "/t", "REG_SZ", "/d", "Brendigo LTD / Brendigo, obrt za programiranje", "/f"},
			{"add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/v", "InstallLocation", "/t", "REG_SZ", "/d", dir, "/f"},
			{"add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/v", "DisplayIcon", "/t", "REG_SZ", "/d", exe, "/f"},
			{"add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/v", "URLInfoAbout", "/t", "REG_SZ", "/d", "https://ghostftp.com/", "/f"},
			{"add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/v", "UninstallString", "/t", "REG_SZ", "/d", uninstall, "/f"},
		}
		for _, a := range args {
			if err := exec.Command("reg", a...).Run(); err != nil {
				rollback()
				return fmt.Errorf("registering Apps & Features entry: %w", err)
			}
		}
		if err := exec.Command("reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/v", "UninstallString").Run(); err != nil {
			rollback()
			return fmt.Errorf("verifying uninstall registration: %w", err)
		}
	} else {
		_ = exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/f").Run()
	}
	_ = os.Remove(backup)
	return nil
}

func shortcut(target, path, args string) error {
	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	ps := fmt.Sprintf(`$w=New-Object -ComObject WScript.Shell;$s=$w.CreateShortcut('%s');$s.TargetPath='%s';$s.WorkingDirectory='%s';$s.Arguments='%s';$s.Description='Ghost FTP';$s.Save()`, esc(path), esc(target), esc(filepath.Dir(target)), esc(args))
	return exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps).Run()
}

func launchInstalled(requested string) {
	time.Sleep(500 * time.Millisecond)
	dir := strings.TrimSpace(requested)
	if dir == "" || strings.EqualFold(dir, `%LOCALAPPDATA%\Programs\Ghost FTP`) {
		dir = filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ghost FTP")
	}
	exe := filepath.Join(filepath.Clean(os.ExpandEnv(dir)), "GhostFTP.exe")
	_ = exec.Command(exe).Start()
}

func openWindow(url string) error {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			profile, err := os.MkdirTemp("", "ghostftp-setup-")
			if err != nil {
				return err
			}
			cmd := exec.Command(p,
				"--app="+url,
				"--window-size=820,630",
				"--user-data-dir="+profile,
				"--no-first-run",
				"--no-default-browser-check",
				"--disable-background-networking",
				"--disable-background-mode",
				"--disable-default-apps",
				"--disable-sync",
				"--disable-component-update",
				"--disable-features=TranslateUI,MediaRouter,OptimizationHints",
				"--disable-session-crashed-bubble",
			)
			if err := cmd.Start(); err != nil {
				_ = os.RemoveAll(profile)
				return err
			}
			stripNativeCaptionSoon()
			go func() {
				_ = cmd.Wait()
				_ = os.RemoveAll(profile)
				shutdown()
			}()
			return nil
		}
	}
	return fmt.Errorf("Microsoft Edge or Chrome was not found")
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
