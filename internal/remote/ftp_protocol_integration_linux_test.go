//go:build linux

package remote

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ftpIntegrationServer is intentionally tiny and test-only. It speaks the
// subset of RFC 959/3659 that the production curl-backed adapter uses. This
// gives CI a real TCP control channel plus passive data channel without public
// test credentials, external services, Docker images, or production modules.
type ftpIntegrationServer struct {
	listener net.Listener
	user     string
	password string

	mu    sync.Mutex
	files map[string][]byte
	dirs  map[string]bool

	wg sync.WaitGroup
}

func newFTPIntegrationServer(t *testing.T, user, password string) *ftpIntegrationServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &ftpIntegrationServer{
		listener: listener,
		user:     user,
		password: password,
		files:    make(map[string][]byte),
		dirs:     map[string]bool{"/": true},
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.serveControl(conn)
			}()
		}
	}()
	return s
}

func (s *ftpIntegrationServer) close() {
	_ = s.listener.Close()
	s.wg.Wait()
}

func (s *ftpIntegrationServer) port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func ftpClean(cwd, raw string) string {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" || raw == "." {
		return cwd
	}
	var target string
	if strings.HasPrefix(raw, "/") {
		target = raw
	} else {
		target = path.Join(cwd, raw)
	}
	target = path.Clean("/" + strings.TrimLeft(target, "/"))
	if target == "." || target == "" {
		return "/"
	}
	return target
}

func (s *ftpIntegrationServer) serveControl(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(45 * time.Second))
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	reply := func(format string, args ...any) bool {
		if _, err := fmt.Fprintf(writer, format+"\r\n", args...); err != nil {
			return false
		}
		return writer.Flush() == nil
	}
	if !reply("220 ByFTP local integration FTP ready") {
		return
	}

	cwd := "/"
	authed := false
	var dataListener net.Listener
	var renameFrom string
	defer func() {
		if dataListener != nil {
			_ = dataListener.Close()
		}
	}()

	openPassive := func() (int, error) {
		if dataListener != nil {
			_ = dataListener.Close()
			dataListener = nil
		}
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return 0, err
		}
		dataListener = l
		return l.Addr().(*net.TCPAddr).Port, nil
	}
	withData := func(action func(net.Conn) error) bool {
		if dataListener == nil {
			return reply("425 Use EPSV or PASV first")
		}
		l := dataListener
		dataListener = nil
		defer l.Close()
		if !reply("150 Opening passive data connection") {
			return false
		}
		if tcp, ok := l.(*net.TCPListener); ok {
			_ = tcp.SetDeadline(time.Now().Add(10 * time.Second))
		}
		dataConn, err := l.Accept()
		if err != nil {
			return reply("425 Cannot open data connection")
		}
		_ = dataConn.SetDeadline(time.Now().Add(10 * time.Second))
		err = action(dataConn)
		_ = dataConn.Close()
		if err != nil {
			return reply("451 Local transfer error")
		}
		return reply("226 Transfer complete")
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		command, argument, _ := strings.Cut(line, " ")
		command = strings.ToUpper(strings.TrimSpace(command))
		argument = strings.TrimSpace(argument)

		switch command {
		case "USER":
			if argument != s.user {
				if !reply("530 Invalid username") {
					return
				}
				continue
			}
			if !reply("331 Password required") {
				return
			}
		case "PASS":
			if argument != s.password {
				if !reply("530 Login incorrect") {
					return
				}
				continue
			}
			authed = true
			if !reply("230 Login successful") {
				return
			}
		case "QUIT":
			reply("221 Goodbye")
			return
		case "SYST":
			if !reply("215 UNIX Type: L8") {
				return
			}
		case "FEAT":
			if _, err := fmt.Fprint(writer, "211-Features\r\n EPSV\r\n MLST type*;size*;modify*;\r\n UTF8\r\n211 End\r\n"); err != nil || writer.Flush() != nil {
				return
			}
		case "OPTS", "CLNT", "TYPE", "NOOP":
			if !reply("200 Command OK") {
				return
			}
		case "PWD", "XPWD":
			if !reply("257 %q is current directory", cwd) {
				return
			}
		case "CWD":
			target := ftpClean(cwd, argument)
			s.mu.Lock()
			exists := s.dirs[target]
			s.mu.Unlock()
			if !exists {
				if !reply("550 Directory not found") {
					return
				}
				continue
			}
			cwd = target
			if !reply("250 Directory changed") {
				return
			}
		case "CDUP":
			cwd = path.Dir(cwd)
			if cwd == "." {
				cwd = "/"
			}
			if !reply("250 Directory changed") {
				return
			}
		case "EPSV":
			port, err := openPassive()
			if err != nil {
				if !reply("425 Passive listener failed") {
					return
				}
				continue
			}
			if !reply("229 Entering Extended Passive Mode (|||%d|)", port) {
				return
			}
		case "PASV":
			port, err := openPassive()
			if err != nil {
				if !reply("425 Passive listener failed") {
					return
				}
				continue
			}
			if !reply("227 Entering Passive Mode (127,0,0,1,%d,%d)", port/256, port%256) {
				return
			}
		case "MLSD", "LIST":
			target := cwd
			if argument != "" && !strings.HasPrefix(argument, "-") {
				target = ftpClean(cwd, argument)
			}
			if !withData(func(data net.Conn) error { return s.writeListing(data, target, command == "MLSD") }) {
				return
			}
		case "STOR":
			if !authed {
				if !reply("530 Login required") {
					return
				}
				continue
			}
			target := ftpClean(cwd, argument)
			if !withData(func(data net.Conn) error {
				payload, err := io.ReadAll(io.LimitReader(data, 8<<20))
				if err != nil {
					return err
				}
				s.mu.Lock()
				s.files[target] = append([]byte(nil), payload...)
				s.dirs[path.Dir(target)] = true
				s.mu.Unlock()
				return nil
			}) {
				return
			}
		case "RETR":
			target := ftpClean(cwd, argument)
			s.mu.Lock()
			payload, ok := s.files[target]
			payload = append([]byte(nil), payload...)
			s.mu.Unlock()
			if !ok {
				if !reply("550 File not found") {
					return
				}
				continue
			}
			if !withData(func(data net.Conn) error {
				_, err := data.Write(payload)
				return err
			}) {
				return
			}
		case "SIZE":
			target := ftpClean(cwd, argument)
			s.mu.Lock()
			payload, ok := s.files[target]
			s.mu.Unlock()
			if !ok {
				if !reply("550 File not found") {
					return
				}
				continue
			}
			if !reply("213 %d", len(payload)) {
				return
			}
		case "MDTM":
			if !reply("213 20260822200000") {
				return
			}
		case "MKD", "XMKD":
			target := ftpClean(cwd, argument)
			s.mu.Lock()
			s.dirs[target] = true
			s.mu.Unlock()
			if !reply("257 %q created", target) {
				return
			}
		case "RMD", "XRMD":
			target := ftpClean(cwd, argument)
			s.mu.Lock()
			delete(s.dirs, target)
			s.mu.Unlock()
			if !reply("250 Directory removed") {
				return
			}
		case "DELE":
			target := ftpClean(cwd, argument)
			s.mu.Lock()
			_, ok := s.files[target]
			if ok {
				delete(s.files, target)
			}
			s.mu.Unlock()
			if !ok {
				if !reply("550 File not found") {
					return
				}
				continue
			}
			if !reply("250 File deleted") {
				return
			}
		case "RNFR":
			renameFrom = ftpClean(cwd, argument)
			s.mu.Lock()
			_, fileOK := s.files[renameFrom]
			dirOK := s.dirs[renameFrom]
			s.mu.Unlock()
			if !fileOK && !dirOK {
				renameFrom = ""
				if !reply("550 Source not found") {
					return
				}
				continue
			}
			if !reply("350 Ready for RNTO") {
				return
			}
		case "RNTO":
			if renameFrom == "" {
				if !reply("503 RNFR required") {
					return
				}
				continue
			}
			target := ftpClean(cwd, argument)
			s.mu.Lock()
			if payload, ok := s.files[renameFrom]; ok {
				s.files[target] = payload
				delete(s.files, renameFrom)
			} else if s.dirs[renameFrom] {
				s.dirs[target] = true
				delete(s.dirs, renameFrom)
			}
			s.mu.Unlock()
			renameFrom = ""
			if !reply("250 Rename successful") {
				return
			}
		case "SITE":
			if !reply("200 SITE command OK") {
				return
			}
		default:
			if !reply("502 Command not implemented") {
				return
			}
		}
	}
}

func (s *ftpIntegrationServer) writeListing(w io.Writer, dir string, mlsd bool) error {
	dir = ftpClean("/", dir)
	s.mu.Lock()
	defer s.mu.Unlock()
	type entry struct {
		name string
		dir  bool
		size int
	}
	entries := make([]entry, 0)
	prefix := strings.TrimSuffix(dir, "/") + "/"
	if dir == "/" {
		prefix = "/"
	}
	seenDirs := make(map[string]bool)
	for file, payload := range s.files {
		if !strings.HasPrefix(file, prefix) {
			continue
		}
		rest := strings.TrimPrefix(file, prefix)
		if rest == "" || strings.Contains(rest, "/") {
			continue
		}
		entries = append(entries, entry{name: rest, size: len(payload)})
	}
	for candidate := range s.dirs {
		if candidate == dir || !strings.HasPrefix(candidate, prefix) {
			continue
		}
		rest := strings.TrimPrefix(candidate, prefix)
		if rest == "" || strings.Contains(rest, "/") || seenDirs[rest] {
			continue
		}
		seenDirs[rest] = true
		entries = append(entries, entry{name: rest, dir: true})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	for _, item := range entries {
		if mlsd {
			typeName := "file"
			if item.dir {
				typeName = "dir"
			}
			if _, err := fmt.Fprintf(w, "type=%s;size=%d;modify=20260822200000; %s\r\n", typeName, item.size, item.name); err != nil {
				return err
			}
			continue
		}
		mode := "-rw-r--r--"
		if item.dir {
			mode = "drwxr-xr-x"
		}
		if _, err := fmt.Fprintf(w, "%s 1 owner group %d Aug 22 20:00 %s\r\n", mode, item.size, item.name); err != nil {
			return err
		}
	}
	return nil
}

func TestCurlFTPRealProtocolWorkflow(t *testing.T) {
	if _, err := findCurl(); err != nil {
		t.Skipf("system curl unavailable: %v", err)
	}
	server := newFTPIntegrationServer(t, "byftp-test", "byftp-test-password")
	defer server.close()

	client, err := NewCurlFTP("ftp", "127.0.0.1", server.port(), "byftp-test", "byftp-test-password")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	items, err := client.List(ctx, "/")
	if err != nil {
		t.Fatalf("initial list: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("initial list = %d items, want empty", len(items))
	}

	if err := client.Mkdir(ctx, "/", "integration-dir"); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	items, err = client.List(ctx, "/")
	if err != nil {
		t.Fatalf("list after mkdir: %v", err)
	}
	foundDir := false
	for _, item := range items {
		if item.Name == "integration-dir" && item.IsDirectory {
			foundDir = true
		}
	}
	if !foundDir {
		t.Fatalf("created directory not visible in MLSD: %#v", items)
	}
	if err := client.Delete(ctx, "/", "integration-dir", true); err != nil {
		t.Fatalf("remove empty directory: %v", err)
	}

	payload := []byte("ByFTP real FTP integration payload\nline two\n")
	localSource := pathForTestTemp(t, "upload-source.txt")
	if err := os.WriteFile(localSource, payload, 0600); err != nil {
		t.Fatal(err)
	}
	if err := client.Upload(ctx, localSource, "/alpha.txt", TransferOptions{}); err != nil {
		t.Fatalf("upload: %v", err)
	}

	items, err = client.List(ctx, "/")
	if err != nil {
		t.Fatalf("list after upload: %v", err)
	}
	foundFile := false
	for _, item := range items {
		if item.Name == "alpha.txt" && !item.IsDirectory && item.Size == int64(len(payload)) {
			foundFile = true
		}
	}
	if !foundFile {
		t.Fatalf("uploaded file not visible with expected size: %#v", items)
	}

	if err := client.Rename(ctx, "/", "alpha.txt", "beta.txt"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	downloadTarget := pathForTestTemp(t, "download-target.txt")
	if err := client.Download(ctx, "/beta.txt", downloadTarget, TransferOptions{}); err != nil {
		t.Fatalf("download: %v", err)
	}
	downloaded, err := os.ReadFile(downloadTarget)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(downloaded, payload) {
		t.Fatalf("downloaded payload mismatch: got %q want %q", downloaded, payload)
	}

	if err := client.Delete(ctx, "/", "beta.txt", false); err != nil {
		t.Fatalf("delete: %v", err)
	}
	items, err = client.List(ctx, "/")
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	if len(items) != 0 {
		names := make([]string, 0, len(items))
		for _, item := range items {
			names = append(names, item.Name)
		}
		t.Fatalf("remote cleanup incomplete: %s", strings.Join(names, ", "))
	}
}

func pathForTestTemp(t *testing.T, name string) string {
	t.Helper()
	return t.TempDir() + string(os.PathSeparator) + name
}

func TestFTPIntegrationServerRejectsWrongPassword(t *testing.T) {
	if _, err := findCurl(); err != nil {
		t.Skipf("system curl unavailable: %v", err)
	}
	server := newFTPIntegrationServer(t, "byftp-test", "correct-password")
	defer server.close()
	client, err := NewCurlFTP("ftp", "127.0.0.1", server.port(), "byftp-test", "wrong-password")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	_, err = client.List(ctx, "/")
	if err == nil {
		t.Fatal("wrong password unexpectedly established a usable FTP session")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "530") && !strings.Contains(strings.ToLower(err.Error()), "login") {
		t.Fatalf("wrong password error does not preserve authentication signal: %v", err)
	}
}

func TestFTPIntegrationServerPortIsLoopback(t *testing.T) {
	server := newFTPIntegrationServer(t, "u", "p")
	defer server.close()
	host, portText, err := net.SplitHostPort(server.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if host != "127.0.0.1" {
		t.Fatalf("integration server escaped loopback: %q", host)
	}
	if port, err := strconv.Atoi(portText); err != nil || port <= 0 {
		t.Fatalf("invalid loopback port %q", portText)
	}
}
