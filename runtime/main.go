package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const version = "2.1.1-rc.8"
const runtimeKind = "compatibility-fallback"

//go:embed site/* site/assets/*
var content embed.FS

var quitOnce sync.Once
var server *http.Server

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("Ghost FTP", version)
			return
		case "--uninstall":
			if runtime.GOOS == "windows" {
				if err := uninstallWindows(); err != nil {
					showWindowsMessage("Ghost FTP", "Uninstall failed: "+err.Error())
				}
				return
			}
		case "--website":
			openExternal("https://ghostftp.com/")
			return
		}
	}

	siteFS, err := fs.Sub(content, "site")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", noCache(http.FileServer(http.FS(siteFS))))
	mux.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "token": sessionToken})
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			state, err := readPersistedState()
			if err != nil {
				writeJSONStatus(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "locale": state.Locale, "sites": state.Sites, "settings": state.Settings})
		case http.MethodPost:
			if !authorizeMutation(w, r) {
				return
			}
			var state persistedState
			if !decodeJSON(w, r, &state) {
				return
			}
			if err := writePersistedState(state); err != nil {
				writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, map[string]any{"ok": true})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		writeJSON(w, map[string]any{"ok": true})
		go shutdownSoon()
	})
	mux.HandleFunc("/api/install", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		writeJSON(w, map[string]any{"ok": false, "message": "Use GhostFTP-Setup.exe to install Ghost FTP."})
	})
	mux.HandleFunc("/api/window", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			Action string `json:"action"`
		}
		if !decodeJSON(w, r, &v) {
			return
		}
		writeJSON(w, map[string]any{"ok": handleWindowAction(v.Action)})
	})
	mux.HandleFunc("/api/fs/home", func(w http.ResponseWriter, r *http.Request) {
		home, err := os.UserHomeDir()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "path": home})
	})
	mux.HandleFunc("/api/fs/list", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("path")
		if p == "" {
			p, _ = os.UserHomeDir()
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		type item struct {
			Name     string    `json:"name"`
			Path     string    `json:"path"`
			Size     int64     `json:"size"`
			Modified time.Time `json:"modified"`
			Dir      bool      `json:"dir"`
			Mode     string    `json:"mode"`
		}
		out := make([]item, 0, len(entries))
		for _, e := range entries {
			info, er := e.Info()
			if er != nil {
				continue
			}
			out = append(out, item{Name: e.Name(), Path: filepath.Join(p, e.Name()), Size: info.Size(), Modified: info.ModTime(), Dir: e.IsDir(), Mode: info.Mode().String()})
		}
		writeJSON(w, map[string]any{"ok": true, "path": p, "items": out})
	})
	mux.HandleFunc("/api/fs/mkdir", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			Path string `json:"path"`
		}
		if !decodeJSON(w, r, &v) || strings.TrimSpace(v.Path) == "" {
			return
		}
		err := os.MkdirAll(filepath.Clean(v.Path), 0755)
		writeJSON(w, map[string]any{"ok": err == nil, "error": errString(err)})
	})
	mux.HandleFunc("/api/fs/rename", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if !decodeJSON(w, r, &v) || strings.TrimSpace(v.From) == "" || strings.TrimSpace(v.To) == "" {
			return
		}
		if dangerousDestructivePath(v.From) || dangerousDestructivePath(v.To) {
			writeJSON(w, map[string]any{"ok": false, "error": "refusing to rename a protected root path"})
			return
		}
		err := os.Rename(filepath.Clean(v.From), filepath.Clean(v.To))
		writeJSON(w, map[string]any{"ok": err == nil, "error": errString(err)})
	})

	mux.HandleFunc("/api/fs/chmod", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			Path      string `json:"path"`
			Mode      uint32 `json:"mode"`
			Recursive bool   `json:"recursive"`
		}
		if !decodeJSON(w, r, &v) || strings.TrimSpace(v.Path) == "" {
			return
		}
		if v.Mode > 0777 || (v.Recursive && dangerousDestructivePath(v.Path)) {
			writeJSON(w, map[string]any{"ok": false, "error": "unsafe permission request"})
			return
		}
		v.Path = filepath.Clean(v.Path)
		var err error
		if v.Recursive {
			err = filepath.Walk(v.Path, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				return os.Chmod(path, os.FileMode(v.Mode&0777))
			})
		} else {
			err = os.Chmod(v.Path, os.FileMode(v.Mode&0777))
		}
		writeJSON(w, map[string]any{"ok": err == nil, "error": errString(err)})
	})
	mux.HandleFunc("/api/fs/checksum", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			Path string `json:"path"`
		}
		if !decodeJSON(w, r, &v) || strings.TrimSpace(v.Path) == "" {
			return
		}
		f, err := os.Open(filepath.Clean(v.Path))
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		defer f.Close()
		h := sha256.New()
		if _, err = io.Copy(h, f); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "sha256": fmt.Sprintf("%x", h.Sum(nil))})
	})
	mux.HandleFunc("/api/fs/duplicate", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			Path string `json:"path"`
		}
		if !decodeJSON(w, r, &v) || strings.TrimSpace(v.Path) == "" {
			return
		}
		dst, err := duplicatePath(filepath.Clean(v.Path))
		writeJSON(w, map[string]any{"ok": err == nil, "path": dst, "error": errString(err)})
	})
	mux.HandleFunc("/api/fs/open", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			Path string `json:"path"`
		}
		if !decodeJSON(w, r, &v) || strings.TrimSpace(v.Path) == "" {
			return
		}
		err := openPath(filepath.Clean(v.Path))
		writeJSON(w, map[string]any{"ok": err == nil, "error": errString(err)})
	})
	mux.HandleFunc("/api/net/test", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			Host string `json:"host"`
			Port string `json:"port"`
		}
		if !decodeJSON(w, r, &v) {
			return
		}
		host := strings.TrimSpace(v.Host)
		port := strings.TrimSpace(v.Port)
		if host == "" || port == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		c, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 5*time.Second)
		if err == nil {
			_ = c.Close()
		}
		writeJSON(w, map[string]any{"ok": err == nil, "scope": "tcp-reachability-only", "error": errString(err)})
	})
	mux.HandleFunc("/api/runtime/info", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "version": version, "kind": runtimeKind, "nativeProtocols": false})
	})
	mux.HandleFunc("/api/fs/delete", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeMutation(w, r) {
			return
		}
		var v struct {
			Path string `json:"path"`
		}
		if !decodeJSON(w, r, &v) || strings.TrimSpace(v.Path) == "" {
			return
		}
		if dangerousDestructivePath(v.Path) {
			writeJSON(w, map[string]any{"ok": false, "error": "refusing to delete a protected root path"})
			return
		}
		err := os.RemoveAll(filepath.Clean(v.Path))
		writeJSON(w, map[string]any{"ok": err == nil, "error": errString(err)})
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	server = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { setSecurityHeaders(w); mux.ServeHTTP(w, r) }), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	url := "http://" + ln.Addr().String() + "/#main"
	_ = url // local application URL
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server: %v", err)
		}
	}()

	if os.Getenv("GHOSTFTP_HEADLESS") == "1" {
		fmt.Println(url)
	} else if err := launchAppWindow(url); err != nil {
		openExternal(url)
	}

	// Keep the embedded local runtime alive while the browser/app window is open.
	// It also self-terminates after a long idle safety period.
	timer := time.NewTimer(12 * time.Hour)
	<-timer.C
	shutdownSoon()
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	setSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	setSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func shutdownSoon() {
	quitOnce.Do(func() {
		time.Sleep(350 * time.Millisecond)
		if server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = server.Shutdown(ctx)
			cancel()
		}
		os.Exit(0)
	})
}

func launchAppWindow(url string) error {
	switch runtime.GOOS {
	case "windows":
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
				profile, err := os.MkdirTemp("", "ghostftp-runtime-")
				if err != nil {
					return err
				}
				cmd := exec.Command(p,
					"--app="+url,
					"--window-size=1290,852",
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
					shutdownSoon()
				}()
				return nil
			}
		}
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "linux":
		for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "brave-browser"} {
			if p, err := exec.LookPath(name); err == nil {
				profile, err := os.MkdirTemp("", "ghostftp-runtime-")
				if err != nil {
					return err
				}
				cmd := exec.Command(p,
					"--app="+url,
					"--window-size=1290,852",
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
					shutdownSoon()
				}()
				return nil
			}
		}
		if p, err := exec.LookPath("xdg-open"); err == nil {
			return exec.Command(p, url).Start()
		}
	}
	return fmt.Errorf("no supported browser runtime found")
}

func openExternal(url string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	}
}

func uninstallWindows() error {
	local := os.Getenv("LOCALAPPDATA")
	appdata := os.Getenv("APPDATA")
	user := os.Getenv("USERPROFILE")
	if local == "" || appdata == "" || user == "" {
		return fmt.Errorf("Windows user folders are unavailable")
	}
	installDir := filepath.Join(local, "Programs", "Ghost FTP")
	if exe, err := os.Executable(); err == nil && strings.EqualFold(filepath.Base(exe), "GhostFTP.exe") {
		candidate := filepath.Clean(filepath.Dir(exe))
		volumeRoot := filepath.Clean(filepath.VolumeName(candidate) + `\`)
		if candidate != "." && candidate != volumeRoot && len(candidate) > len(volumeRoot)+3 {
			installDir = candidate
		}
	}
	desktop := filepath.Join(user, "Desktop", "Ghost FTP.lnk")
	startDir := filepath.Join(appdata, "Microsoft", "Windows", "Start Menu", "Programs")
	start := filepath.Join(startDir, "Ghost FTP.lnk")
	uninstallLink := filepath.Join(startDir, "Uninstall Ghost FTP.lnk")
	_ = os.Remove(desktop)
	_ = os.Remove(start)
	_ = os.Remove(uninstallLink)
	_ = exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`, "/f").Run()
	exe, _ := os.Executable()
	cmd := fmt.Sprintf(`timeout /t 1 /nobreak >nul & rmdir /s /q "%s"`, strings.ReplaceAll(installDir, `"`, ``))
	if strings.HasPrefix(strings.ToLower(exe), strings.ToLower(installDir)) {
		_ = exec.Command("cmd", "/C", "start", "", "/min", "cmd", "/C", cmd).Start()
	} else {
		_ = os.RemoveAll(installDir)
	}
	showWindowsMessage("Ghost FTP", "Ghost FTP has been removed from this user account.")
	return nil
}

func showWindowsMessage(title, message string) {
	if runtime.GOOS != "windows" {
		return
	}
	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	script := fmt.Sprintf(`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('%s','%s') | Out-Null`, esc(message), esc(title))
	_ = exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script).Run()
}

func duplicatePath(src string) (string, error) {
	info, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	dir, base := filepath.Dir(src), filepath.Base(src)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	var dst string
	for i := 1; ; i++ {
		suffix := " copy"
		if i > 1 {
			suffix = fmt.Sprintf(" copy %d", i)
		}
		dst = filepath.Join(dir, stem+suffix+ext)
		if _, e := os.Stat(dst); os.IsNotExist(e) {
			break
		}
	}
	if info.IsDir() {
		return dst, copyDir(src, dst)
	}
	return dst, copyFile(src, dst, info.Mode())
}
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	_, cpErr := io.Copy(out, in)
	closeErr := out.Close()
	if cpErr != nil {
		return cpErr
	}
	return closeErr
}
func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err = os.Mkdir(dst, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		inf, er := e.Info()
		if er != nil {
			return er
		}
		if e.IsDir() {
			if er = copyDir(s, d); er != nil {
				return er
			}
		} else {
			if er = copyFile(s, d, inf.Mode()); er != nil {
				return er
			}
		}
	}
	return nil
}
func openPath(p string) error {
	info, err := os.Stat(p)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		p = filepath.Dir(p)
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", p).Start()
	case "linux":
		return exec.Command("xdg-open", p).Start()
	}
	return fmt.Errorf("opening paths is unsupported on this platform")
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func decodeURLPath(v string) string {
	if x, err := url.QueryUnescape(v); err == nil {
		return x
	}
	return v
}
