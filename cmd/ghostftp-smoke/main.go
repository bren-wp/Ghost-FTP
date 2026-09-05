package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

const (
	connectTimeout  = 75 * time.Second
	operationTimeout = 60 * time.Second
	transferTimeout = 2 * time.Minute
)

type liveConfig struct {
	Protocol string
	Host     string
	Port     int
	Username string
	Password string
	Write    bool
	Verbose  bool
}

func defaultSmokePort(protocol string) int {
	switch protocol {
	case "sftp":
		return 22
	case "ftpsi":
		return 990
	default:
		return 21
	}
}

func loadConfig() (liveConfig, error) {
	cfg := liveConfig{
		Protocol: strings.ToLower(strings.TrimSpace(os.Getenv("GHOSTFTP_TEST_PROTOCOL"))),
		Host:     strings.TrimSpace(os.Getenv("GHOSTFTP_TEST_HOST")),
		Username: strings.TrimSpace(os.Getenv("GHOSTFTP_TEST_USER")),
		Password: os.Getenv("GHOSTFTP_TEST_PASSWORD"),
		Write:    os.Getenv("GHOSTFTP_TEST_WRITE") == "1",
		Verbose:  os.Getenv("GHOSTFTP_TEST_VERBOSE") == "1",
	}
	// Remove inherited credential material before starting any child transport
	// process. The ConnectionConfig below is the only intentional in-process
	// copy needed by the typed Engine API.
	_ = os.Unsetenv("GHOSTFTP_TEST_PASSWORD")
	if cfg.Protocol == "" {
		cfg.Protocol = "ftps"
	}
	switch cfg.Protocol {
	case "ftp", "ftps", "ftpsi", "sftp":
	default:
		return liveConfig{}, errors.New("GHOSTFTP_TEST_PROTOCOL must be ftp, ftps, ftpsi or sftp")
	}
	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
		return liveConfig{}, errors.New("GHOSTFTP_TEST_HOST, GHOSTFTP_TEST_USER and GHOSTFTP_TEST_PASSWORD are required")
	}
	cfg.Port = defaultSmokePort(cfg.Protocol)
	if raw := strings.TrimSpace(os.Getenv("GHOSTFTP_TEST_PORT")); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return liveConfig{}, errors.New("GHOSTFTP_TEST_PORT is invalid")
		}
		cfg.Port = port
	}
	return cfg, nil
}

func randomToken() (string, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func waitJob(engine *api.Engine, id string) error {
	deadline := time.NewTimer(transferTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			_ = engine.CancelTransfer(id)
			return errors.New("transfer timed out")
		case <-ticker.C:
			for _, job := range engine.Transfers() {
				if job.ID != id {
					continue
				}
				switch job.Status {
				case "done":
					return nil
				case "failed", "cancelled", "skipped":
					return fmt.Errorf("transfer ended with status %s", job.Status)
				}
			}
		}
	}
}

func stepError(step string, err error, verbose bool) error {
	if verbose {
		return fmt.Errorf("%s: %v", step, err)
	}
	return fmt.Errorf("%s failed", step)
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	defer func() { cfg.Password = "" }()

	dataDir, err := os.MkdirTemp("", "ghostftp-live-smoke-state-")
	if err != nil {
		return stepError("temporary state", err, cfg.Verbose)
	}
	defer os.RemoveAll(dataDir)
	if err := os.Chmod(dataDir, 0o700); err != nil {
		return stepError("temporary state permissions", err, cfg.Verbose)
	}
	exe, err := os.Executable()
	if err != nil {
		return stepError("executable path", err, cfg.Verbose)
	}
	engine, err := api.New(dataDir, exe)
	if err != nil {
		return stepError("engine startup", err, cfg.Verbose)
	}
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	result, err := engine.Connect(ctx, "", model.ConnectionConfig{
		Protocol: cfg.Protocol,
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
	}, "", false)
	cancel()
	cfg.Password = ""
	if err != nil {
		return stepError("connect", err, cfg.Verbose)
	}
	if result.RequiresTrust {
		engine.CancelPendingTrust()
		return errors.New("SFTP smoke requires an explicitly preconfigured trust fingerprint; refusing interactive trust")
	}
	if !result.Connected {
		return errors.New("connect did not establish a session")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = engine.Disconnect(ctx)
	}()
	fmt.Printf("GHOSTFTP_LIVE_CONNECT=PASS protocol=%s port=%d\n", cfg.Protocol, cfg.Port)

	ctx, cancel = context.WithTimeout(context.Background(), operationTimeout)
	_, err = engine.RemoteList(ctx, "/")
	cancel()
	if err != nil {
		return stepError("root listing", err, cfg.Verbose)
	}
	fmt.Println("GHOSTFTP_LIVE_LIST=PASS")
	if !cfg.Write {
		fmt.Println("GHOSTFTP_LIVE_WRITE=SKIPPED")
		fmt.Println("GHOSTFTP_LIVE_SMOKE=PASS")
		return nil
	}

	token, err := randomToken()
	if err != nil {
		return stepError("probe token", err, cfg.Verbose)
	}
	remoteDirName := ".ghostftp-smoke-" + token
	remoteDir := path.Join("/", remoteDirName)
	remoteOriginal := path.Join(remoteDir, "probe.bin")
	remoteRenamed := path.Join(remoteDir, "probe-renamed.bin")
	remoteCreated := false
	remoteFile := ""
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if remoteFile != "" {
			_ = engine.RemoteDelete(ctx, remoteDir, path.Base(remoteFile), false)
		}
		if remoteCreated {
			_ = engine.RemoteDelete(ctx, "/", remoteDirName, true)
		}
	}()

	ctx, cancel = context.WithTimeout(context.Background(), operationTimeout)
	err = engine.RemoteMkdir(ctx, "/", remoteDirName)
	cancel()
	if err != nil {
		return stepError("remote mkdir", err, cfg.Verbose)
	}
	remoteCreated = true

	payload := []byte("Ghost FTP live smoke " + token + "\n")
	localSource := filepath.Join(dataDir, "probe.bin")
	if err := os.WriteFile(localSource, payload, 0o600); err != nil {
		return stepError("local probe create", err, cfg.Verbose)
	}
	job, err := engine.AddTransfer("upload", localSource, remoteOriginal, dataDir)
	if err != nil {
		return stepError("queue upload", err, cfg.Verbose)
	}
	if err := waitJob(engine, job.ID); err != nil {
		return stepError("upload", err, cfg.Verbose)
	}
	remoteFile = remoteOriginal

	ctx, cancel = context.WithTimeout(context.Background(), operationTimeout)
	items, err := engine.RemoteList(ctx, remoteDir)
	cancel()
	if err != nil {
		return stepError("probe listing", err, cfg.Verbose)
	}
	found := false
	for _, item := range items {
		if item.Name == "probe.bin" && !item.IsDirectory {
			found = true
			break
		}
	}
	if !found {
		return errors.New("uploaded probe was not visible in remote listing")
	}

	localDownload := filepath.Join(dataDir, "download.bin")
	job, err = engine.AddTransfer("download", localDownload, remoteOriginal, dataDir)
	if err != nil {
		return stepError("queue download", err, cfg.Verbose)
	}
	if err := waitJob(engine, job.ID); err != nil {
		return stepError("download", err, cfg.Verbose)
	}
	downloaded, err := os.ReadFile(localDownload)
	if err != nil {
		return stepError("download verification", err, cfg.Verbose)
	}
	if !bytes.Equal(downloaded, payload) {
		return errors.New("downloaded probe content did not match upload")
	}

	ctx, cancel = context.WithTimeout(context.Background(), operationTimeout)
	err = engine.RemoteRename(ctx, remoteDir, "probe.bin", "probe-renamed.bin")
	cancel()
	if err != nil {
		return stepError("remote rename", err, cfg.Verbose)
	}
	remoteFile = remoteRenamed

	ctx, cancel = context.WithTimeout(context.Background(), operationTimeout)
	err = engine.RemoteDelete(ctx, remoteDir, "probe-renamed.bin", false)
	cancel()
	if err != nil {
		return stepError("remote file delete", err, cfg.Verbose)
	}
	remoteFile = ""

	ctx, cancel = context.WithTimeout(context.Background(), operationTimeout)
	err = engine.RemoteDelete(ctx, "/", remoteDirName, true)
	cancel()
	if err != nil {
		return stepError("remote directory delete", err, cfg.Verbose)
	}
	remoteCreated = false

	fmt.Println("GHOSTFTP_LIVE_UPLOAD=PASS")
	fmt.Println("GHOSTFTP_LIVE_DOWNLOAD=PASS")
	fmt.Println("GHOSTFTP_LIVE_RENAME=PASS")
	fmt.Println("GHOSTFTP_LIVE_DELETE=PASS")
	fmt.Println("GHOSTFTP_LIVE_WRITE=PASS")
	fmt.Println("GHOSTFTP_LIVE_SMOKE=PASS")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "GHOSTFTP_LIVE_SMOKE=FAIL:", err)
		os.Exit(1)
	}
}
