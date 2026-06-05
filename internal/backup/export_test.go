package backup

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"4vpx/internal/domain"
	sqliteStore "4vpx/internal/storage/sqlite"
)

func TestExportImportJSONRoundTrip(t *testing.T) {
	ctx := context.Background()

	sourceDB, err := sqliteStore.Open(filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatalf("open source db: %v", err)
	}
	defer sourceDB.Close()

	store := sqliteStore.NewStore(sourceDB)
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	exportedAt := now.Add(time.Hour)

	admin, err := store.Admins.Create(ctx, domain.Admin{
		Username:     "admin",
		PasswordHash: "hashed-password",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	user, err := store.Users.Create(ctx, domain.User{
		Name:        "alice",
		Notes:       "vip",
		Enabled:     true,
		ExpiresAt:   now.Add(7 * 24 * time.Hour),
		AccessToken: "token-1",
		DeviceSlots: 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	slot, err := store.DeviceSlots.Create(ctx, domain.DeviceSlot{
		UserID:         user.ID,
		SlotIndex:      1,
		Label:          "phone",
		UUID:           "uuid-1",
		Enabled:        true,
		LastExportedAt: &exportedAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatalf("create device slot: %v", err)
	}

	renewal, err := store.Renewals.Create(ctx, domain.RenewalRecord{
		UserID:          user.ID,
		Days:            7,
		Action:          "renew_7_days",
		BeforeExpiresAt: user.ExpiresAt,
		AfterExpiresAt:  user.ExpiresAt.Add(7 * 24 * time.Hour),
		Notes:           "manual",
		Actor:           "admin",
		CreatedAt:       now,
	})
	if err != nil {
		t.Fatalf("create renewal: %v", err)
	}

	err = store.System.Upsert(ctx, domain.SystemConfig{
		ID:                1,
		ServerAddress:     "example.com",
		ServerPort:        443,
		RealityDest:       "www.microsoft.com:443",
		RealityServerName: "www.microsoft.com",
		ClientFingerprint: "chrome",
		RealityPrivateKey: "private-key",
		RealityPublicKey:  "public-key",
		RealityShortID:    "abcd1234",
		XrayLogLevel:      "warning",
		XrayConfigPath:    "/etc/xray/config.json",
		XrayBackupPath:    "/etc/xray/config.json.bak",
		XrayBin:           "xray",
		XrayReloadCmd:     "systemctl reload xray",
		UpdatedAt:         now,
	})
	if err != nil {
		t.Fatalf("upsert system config: %v", err)
	}

	exporter := NewExporter(sourceDB)
	exporter.now = func() time.Time { return exportedAt }

	data, err := exporter.ExportJSON(ctx)
	if err != nil {
		t.Fatalf("ExportJSON() error = %v", err)
	}
	if !strings.Contains(string(data), "\"device_slots\"") {
		t.Fatalf("export payload missing device slots: %s", data)
	}

	targetDB, err := sqliteStore.Open(filepath.Join(t.TempDir(), "target.db"))
	if err != nil {
		t.Fatalf("open target db: %v", err)
	}
	defer targetDB.Close()

	importer := NewImporter(targetDB)
	if err := importer.ImportJSON(ctx, data); err != nil {
		t.Fatalf("ImportJSON() error = %v", err)
	}

	bundle, err := NewExporter(targetDB).Export(ctx)
	if err != nil {
		t.Fatalf("Export() after import error = %v", err)
	}

	if len(bundle.Admins) != 1 || bundle.Admins[0].ID != admin.ID {
		t.Fatalf("unexpected admins after import: %+v", bundle.Admins)
	}
	if len(bundle.Users) != 1 || bundle.Users[0].AccessToken != user.AccessToken {
		t.Fatalf("unexpected users after import: %+v", bundle.Users)
	}
	if len(bundle.DeviceSlots) != 1 || bundle.DeviceSlots[0].UUID != slot.UUID {
		t.Fatalf("unexpected device slots after import: %+v", bundle.DeviceSlots)
	}
	if len(bundle.Renewals) != 1 || bundle.Renewals[0].ID != renewal.ID {
		t.Fatalf("unexpected renewals after import: %+v", bundle.Renewals)
	}
	if bundle.SystemConfig.ServerAddress != "example.com" {
		t.Fatalf("unexpected system config after import: %+v", bundle.SystemConfig)
	}
}
