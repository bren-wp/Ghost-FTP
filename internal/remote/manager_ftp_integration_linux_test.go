//go:build linux

package remote

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

// This test goes through the same Manager.Connect -> initial List -> Operation
// lifecycle used by the desktop application. The loopback server speaks a real
// FTP control/data protocol, so a green test proves more than adapter setup or
// mocked state transitions.
func TestManagerConnectsAndUsesRealFTPProtocolSession(t *testing.T) {
	if _, err := findCurl(); err != nil {
		t.Skipf("system curl unavailable: %v", err)
	}
	server := newFTPIntegrationServer(t, "GhostFTP-manager", "manager-password")
	defer server.close()

	m := NewManager(nil, nil, t.TempDir(), "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := m.Connect(ctx, "", model.ConnectionConfig{
		Protocol: "ftp",
		Host:     "127.0.0.1",
		Port:     server.port(),
		Username: "GhostFTP-manager",
		Password: "manager-password",
	}, "", false)
	if err != nil {
		t.Fatalf("manager connect: %v", err)
	}
	if !result.Connected || result.RequiresTrust {
		t.Fatalf("unexpected connect result: %#v", result)
	}
	cfg, connected := m.Config()
	if !connected {
		t.Fatal("manager did not expose connected state after successful probe")
	}
	if cfg.Protocol != "ftp" || cfg.Host != "127.0.0.1" || cfg.Port != server.port() || cfg.Username != "GhostFTP-manager" {
		t.Fatalf("public connection config mismatch: %#v", cfg)
	}
	if cfg.Password != "" || cfg.Passphrase != "" {
		t.Fatal("manager retained plaintext connection credentials in public config")
	}

	session, opCtx, release, err := m.Operation(ctx)
	if err != nil {
		t.Fatalf("operation after connect: %v", err)
	}
	items, err := session.List(opCtx, "/")
	release()
	if err != nil {
		t.Fatalf("list through connected manager: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("new test server root is not empty: %#v", items)
	}
	if identity, err := m.ConnectionIdentity(); err != nil || identity == "" {
		t.Fatalf("connection identity after connect = %q, %v", identity, err)
	}

	if err := m.Disconnect(ctx); err != nil {
		t.Fatalf("disconnect: %v", err)
	}
	if _, connected := m.Config(); connected {
		t.Fatal("manager remained connected after disconnect")
	}
	if _, _, _, err := m.Operation(context.Background()); err == nil {
		t.Fatal("operation was admitted after disconnect")
	}
}

func TestManagerRejectsWrongFTPPasswordBeforePublishingSession(t *testing.T) {
	if _, err := findCurl(); err != nil {
		t.Skipf("system curl unavailable: %v", err)
	}
	server := newFTPIntegrationServer(t, "GhostFTP-manager", "correct-password")
	defer server.close()
	m := NewManager(nil, nil, t.TempDir(), "")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := m.Connect(ctx, "", model.ConnectionConfig{
		Protocol: "ftp",
		Host:     "127.0.0.1",
		Port:     server.port(),
		Username: "GhostFTP-manager",
		Password: "wrong-password",
	}, "", false)
	if err == nil {
		t.Fatalf("wrong password unexpectedly connected: %#v", result)
	}
	if _, connected := m.Config(); connected {
		t.Fatal("failed authentication leaked a connected manager state")
	}
	if _, _, _, opErr := m.Operation(context.Background()); opErr == nil {
		t.Fatal("failed authentication left an operational session")
	}
}

func TestExplicitFTPSDoesNotDowngradeToPlainFTP(t *testing.T) {
	if _, err := findCurl(); err != nil {
		t.Skipf("system curl unavailable: %v", err)
	}
	server := newFTPIntegrationServer(t, "GhostFTP-ftps", "password")
	defer server.close()
	m := NewManager(nil, nil, t.TempDir(), "")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := m.Connect(ctx, "", model.ConnectionConfig{
		Protocol: "ftps",
		Host:     "127.0.0.1",
		Port:     server.port(),
		Username: "GhostFTP-ftps",
		Password: "password",
	}, "", false)
	if err == nil {
		t.Fatal("explicit FTPS unexpectedly accepted a plaintext-only FTP endpoint")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "ssl") && !strings.Contains(lower, "tls") && !strings.Contains(lower, "auth") && !strings.Contains(lower, "502") {
		t.Logf("FTPS downgrade rejection returned platform-specific diagnostic: %v", err)
	}
	if _, connected := m.Config(); connected {
		t.Fatal("failed FTPS negotiation published a connected state")
	}
}
