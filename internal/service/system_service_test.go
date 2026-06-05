package service

import (
	"context"
	"testing"
	"time"

	"4vpx/internal/domain"
)

func TestSystemServiceEnsurePersistsMergedDefaults(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	service := NewSystemService(store, domain.SystemConfig{
		ID:                1,
		ServerAddress:     "203.0.113.1",
		ServerPort:        443,
		RealityDest:       "www.microsoft.com:443",
		RealityServerName: "www.microsoft.com",
		ClientFingerprint: "chrome",
		RealityPrivateKey: "private-key",
		RealityPublicKey:  "public-key",
		RealityShortID:    "abcd1234",
		XrayLogLevel:      "warning",
		XrayConfigPath:    "/usr/local/etc/xray/config.json",
		XrayBackupPath:    "/usr/local/etc/xray/config.json.bak",
		XrayBin:           "/usr/local/bin/xray",
		XrayReloadCmd:     "systemctl restart xray.service",
		UpdatedAt:         now,
	})

	if err := store.System.Upsert(context.Background(), domain.SystemConfig{
		ID:                1,
		ServerAddress:     "203.0.113.1",
		ServerPort:        443,
		RealityDest:       "www.microsoft.com:443",
		RealityServerName: "www.microsoft.com",
		ClientFingerprint: "chrome",
		RealityShortID:    "",
		XrayLogLevel:      "warning",
		XrayConfigPath:    "/usr/local/etc/xray/config.json",
		XrayBackupPath:    "/usr/local/etc/xray/config.json.bak",
		XrayBin:           "/usr/local/bin/xray",
		XrayReloadCmd:     "systemctl restart xray.service",
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("seed system config: %v", err)
	}

	cfg, err := service.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if cfg.RealityPrivateKey != "private-key" || cfg.RealityPublicKey != "public-key" || cfg.RealityShortID != "abcd1234" {
		t.Fatalf("Ensure() did not merge defaults: %+v", cfg)
	}

	stored, err := store.System.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.RealityPrivateKey != "private-key" || stored.RealityPublicKey != "public-key" || stored.RealityShortID != "abcd1234" {
		t.Fatalf("stored config did not persist merged defaults: %+v", stored)
	}
}
