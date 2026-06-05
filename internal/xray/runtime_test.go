package xray

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"4vpx/internal/domain"
)

func TestTempConfigPathPreservesJSONExtension(t *testing.T) {
	got := tempConfigPath("/usr/local/etc/xray/config.json")
	want := "/usr/local/etc/xray/config.tmp.json"
	if got != want {
		t.Fatalf("tempConfigPath() = %q, want %q", got, want)
	}
}

func TestTempConfigPathAddsJSONWhenMissingExtension(t *testing.T) {
	got := tempConfigPath("/usr/local/etc/xray/config")
	want := "/usr/local/etc/xray/config.tmp.json"
	if got != want {
		t.Fatalf("tempConfigPath() = %q, want %q", got, want)
	}
}

func TestPublishPreservesExistingConfigMode(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	backupPath := filepath.Join(dir, "config.json.bak")
	if err := os.WriteFile(configPath, []byte("{\"old\":true}\n"), 0o640); err != nil {
		t.Fatalf("write existing config: %v", err)
	}
	if err := os.Chmod(configPath, 0o640); err != nil {
		t.Fatalf("chmod existing config: %v", err)
	}

	runtime := NewRuntime(renderer)
	runtime.now = func() time.Time { return time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC) }

	cfg := domain.SystemConfig{
		ServerAddress:     "example.com",
		ServerPort:        443,
		RealityDest:       "www.microsoft.com:443",
		RealityServerName: "www.microsoft.com",
		ClientFingerprint: "chrome",
		RealityPrivateKey: "private-key",
		RealityPublicKey:  "public-key",
		RealityShortID:    "abcd1234",
		XrayLogLevel:      "warning",
		XrayConfigPath:    configPath,
		XrayBackupPath:    backupPath,
	}
	devices := []DeviceRecord{
		{
			User: domain.User{Name: "alice", Enabled: true, ExpiresAt: runtime.now().Add(24 * time.Hour)},
			Slot: domain.DeviceSlot{SlotIndex: 1, Label: "phone", UUID: "uuid-active", Enabled: true},
		},
	}

	if _, err := runtime.Publish(context.Background(), cfg, devices); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o640); got != want {
		t.Fatalf("config mode = %o, want %o", got, want)
	}

	rendered, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(rendered), "uuid-active") {
		t.Fatalf("rendered config missing device uuid: %s", rendered)
	}

	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if got := string(backup); got != "{\"old\":true}\n" {
		t.Fatalf("backup contents = %q, want previous config", got)
	}

	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("stat backup: %v", err)
	}
	if got, want := backupInfo.Mode().Perm(), os.FileMode(0o640); got != want {
		t.Fatalf("backup mode = %o, want %o", got, want)
	}
}
