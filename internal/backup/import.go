package backup

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"4vpx/internal/domain"
)

type Importer struct {
	db *sql.DB
}

func NewImporter(db *sql.DB) *Importer {
	return &Importer{db: db}
}

func (i *Importer) ImportJSON(ctx context.Context, data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var bundle domain.ExportBundle
	if err := dec.Decode(&bundle); err != nil {
		return fmt.Errorf("decode export bundle: %w", err)
	}
	return i.Import(ctx, bundle)
}

func (i *Importer) Import(ctx context.Context, bundle domain.ExportBundle) (err error) {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, stmt := range []string{
		`DELETE FROM renewal_records`,
		`DELETE FROM device_slots`,
		`DELETE FROM users`,
		`DELETE FROM admins`,
		`DELETE FROM system_configs`,
	} {
		if _, err = tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("reset tables: %w", err)
		}
	}

	if err = importAdmins(ctx, tx, bundle.Admins); err != nil {
		return err
	}
	if err = importUsers(ctx, tx, bundle.Users); err != nil {
		return err
	}
	if err = importDeviceSlots(ctx, tx, bundle.DeviceSlots); err != nil {
		return err
	}
	if err = importRenewals(ctx, tx, bundle.Renewals); err != nil {
		return err
	}
	if hasSystemConfig(bundle.SystemConfig) {
		if err = importSystemConfig(ctx, tx, bundle.SystemConfig); err != nil {
			return err
		}
	}

	if err = syncSQLiteSequence(ctx, tx, "admins", maxAdminID(bundle.Admins)); err != nil {
		return err
	}
	if err = syncSQLiteSequence(ctx, tx, "users", maxUserID(bundle.Users)); err != nil {
		return err
	}
	if err = syncSQLiteSequence(ctx, tx, "device_slots", maxDeviceSlotID(bundle.DeviceSlots)); err != nil {
		return err
	}
	if err = syncSQLiteSequence(ctx, tx, "renewal_records", maxRenewalID(bundle.Renewals)); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit import transaction: %w", err)
	}
	return nil
}

func importAdmins(ctx context.Context, tx *sql.Tx, admins []domain.Admin) error {
	for _, admin := range admins {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO admins (id, username, password_hash, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?)
        `,
			admin.ID,
			admin.Username,
			admin.PasswordHash,
			formatTimeOrZero(admin.CreatedAt),
			formatTimeOrZero(admin.UpdatedAt),
		); err != nil {
			return fmt.Errorf("import admin %q: %w", admin.Username, err)
		}
	}
	return nil
}

func importUsers(ctx context.Context, tx *sql.Tx, users []domain.User) error {
	for _, user := range users {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO users (
                id, name, notes, enabled, expires_at, access_token, device_slots, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        `,
			user.ID,
			user.Name,
			user.Notes,
			boolToInt(user.Enabled),
			formatTimeOrZero(user.ExpiresAt),
			user.AccessToken,
			user.DeviceSlots,
			formatTimeOrZero(user.CreatedAt),
			formatTimeOrZero(user.UpdatedAt),
		); err != nil {
			return fmt.Errorf("import user %q: %w", user.Name, err)
		}
	}
	return nil
}

func importDeviceSlots(ctx context.Context, tx *sql.Tx, slots []domain.DeviceSlot) error {
	for _, slot := range slots {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO device_slots (
                id, user_id, slot_index, label, uuid, enabled, last_exported_at, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        `,
			slot.ID,
			slot.UserID,
			slot.SlotIndex,
			slot.Label,
			slot.UUID,
			boolToInt(slot.Enabled),
			formatNullableTime(slot.LastExportedAt),
			formatTimeOrZero(slot.CreatedAt),
			formatTimeOrZero(slot.UpdatedAt),
		); err != nil {
			return fmt.Errorf("import device slot %d/%d: %w", slot.UserID, slot.SlotIndex, err)
		}
	}
	return nil
}

func importRenewals(ctx context.Context, tx *sql.Tx, renewals []domain.RenewalRecord) error {
	for _, renewal := range renewals {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO renewal_records (
                id, user_id, days, action, before_expires_at, after_expires_at, notes, actor, created_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        `,
			renewal.ID,
			renewal.UserID,
			renewal.Days,
			renewal.Action,
			formatTimeOrZero(renewal.BeforeExpiresAt),
			formatTimeOrZero(renewal.AfterExpiresAt),
			renewal.Notes,
			renewal.Actor,
			formatTimeOrZero(renewal.CreatedAt),
		); err != nil {
			return fmt.Errorf("import renewal %d: %w", renewal.ID, err)
		}
	}
	return nil
}

func importSystemConfig(ctx context.Context, tx *sql.Tx, cfg domain.SystemConfig) error {
	if cfg.ID == 0 {
		cfg.ID = 1
	}
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO system_configs (
            id,
            server_address,
            server_port,
            reality_dest,
            reality_server_name,
            client_fingerprint,
            reality_private_key,
            reality_public_key,
            reality_short_id,
            xray_loglevel,
            xray_config_path,
            xray_backup_path,
            xray_bin,
            xray_reload_cmd,
            updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
		cfg.ID,
		cfg.ServerAddress,
		cfg.ServerPort,
		cfg.RealityDest,
		cfg.RealityServerName,
		cfg.ClientFingerprint,
		cfg.RealityPrivateKey,
		cfg.RealityPublicKey,
		cfg.RealityShortID,
		cfg.XrayLogLevel,
		cfg.XrayConfigPath,
		cfg.XrayBackupPath,
		cfg.XrayBin,
		cfg.XrayReloadCmd,
		formatTimeOrZero(cfg.UpdatedAt),
	); err != nil {
		return fmt.Errorf("import system config: %w", err)
	}
	return nil
}

func hasSystemConfig(cfg domain.SystemConfig) bool {
	return cfg.ID != 0 ||
		cfg.ServerAddress != "" ||
		cfg.ServerPort != 0 ||
		cfg.RealityDest != "" ||
		cfg.RealityServerName != "" ||
		cfg.ClientFingerprint != "" ||
		cfg.RealityPrivateKey != "" ||
		cfg.RealityPublicKey != "" ||
		cfg.RealityShortID != "" ||
		cfg.XrayLogLevel != "" ||
		cfg.XrayConfigPath != "" ||
		cfg.XrayBackupPath != "" ||
		cfg.XrayBin != "" ||
		cfg.XrayReloadCmd != ""
}

func syncSQLiteSequence(ctx context.Context, tx *sql.Tx, table string, maxID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM sqlite_sequence WHERE name = ?`, table); err != nil {
		return fmt.Errorf("reset sqlite sequence for %s: %w", table, err)
	}
	if maxID == 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sqlite_sequence(name, seq) VALUES(?, ?)`, table, maxID); err != nil {
		return fmt.Errorf("set sqlite sequence for %s: %w", table, err)
	}
	return nil
}

func maxAdminID(admins []domain.Admin) int64 {
	var maxID int64
	for _, admin := range admins {
		if admin.ID > maxID {
			maxID = admin.ID
		}
	}
	return maxID
}

func maxUserID(users []domain.User) int64 {
	var maxID int64
	for _, user := range users {
		if user.ID > maxID {
			maxID = user.ID
		}
	}
	return maxID
}

func maxDeviceSlotID(slots []domain.DeviceSlot) int64 {
	var maxID int64
	for _, slot := range slots {
		if slot.ID > maxID {
			maxID = slot.ID
		}
	}
	return maxID
}

func maxRenewalID(renewals []domain.RenewalRecord) int64 {
	var maxID int64
	for _, renewal := range renewals {
		if renewal.ID > maxID {
			maxID = renewal.ID
		}
	}
	return maxID
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func formatNullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTimeOrZero(*t)
}

func formatTimeOrZero(t time.Time) string {
	if t.IsZero() {
		return time.Time{}.UTC().Format(time.RFC3339Nano)
	}
	return t.UTC().Format(time.RFC3339Nano)
}
