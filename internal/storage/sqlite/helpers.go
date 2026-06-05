package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"4vpx/internal/domain"
)

const timeLayout = time.RFC3339Nano

type scanner interface {
	Scan(dest ...any) error
}

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func formatNullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseTime(v string) time.Time {
	t, err := time.Parse(timeLayout, v)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseNullTime(v sql.NullString) *time.Time {
	if !v.Valid || v.String == "" {
		return nil
	}
	t := parseTime(v.String)
	return &t
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func intToBool(v int) bool {
	return v != 0
}

func ensureRowsAffected(res sql.Result) error {
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanAdmin(s scanner) (domain.Admin, error) {
	var admin domain.Admin
	var createdAt, updatedAt string
	if err := s.Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &createdAt, &updatedAt); err != nil {
		return domain.Admin{}, err
	}
	admin.CreatedAt = parseTime(createdAt)
	admin.UpdatedAt = parseTime(updatedAt)
	return admin, nil
}

func scanAdminSession(s scanner) (domain.AdminSession, error) {
	var session domain.AdminSession
	var createdAt, updatedAt string
	if err := s.Scan(&session.Token, &session.AdminID, &createdAt, &updatedAt); err != nil {
		return domain.AdminSession{}, err
	}
	session.CreatedAt = parseTime(createdAt)
	session.UpdatedAt = parseTime(updatedAt)
	return session, nil
}

func scanUser(s scanner) (domain.User, error) {
	var user domain.User
	var enabled int
	var expiresAt, createdAt, updatedAt string
	if err := s.Scan(
		&user.ID,
		&user.Name,
		&user.Notes,
		&enabled,
		&expiresAt,
		&user.AccessToken,
		&user.DeviceSlots,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.User{}, err
	}
	user.Enabled = intToBool(enabled)
	user.ExpiresAt = parseTime(expiresAt)
	user.CreatedAt = parseTime(createdAt)
	user.UpdatedAt = parseTime(updatedAt)
	return user, nil
}

func scanDeviceSlot(s scanner) (domain.DeviceSlot, error) {
	var slot domain.DeviceSlot
	var enabled int
	var lastExportedAt sql.NullString
	var createdAt, updatedAt string
	if err := s.Scan(
		&slot.ID,
		&slot.UserID,
		&slot.SlotIndex,
		&slot.Label,
		&slot.UUID,
		&enabled,
		&lastExportedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.DeviceSlot{}, err
	}
	slot.Enabled = intToBool(enabled)
	slot.LastExportedAt = parseNullTime(lastExportedAt)
	slot.CreatedAt = parseTime(createdAt)
	slot.UpdatedAt = parseTime(updatedAt)
	return slot, nil
}

func scanRenewalRecord(s scanner) (domain.RenewalRecord, error) {
	var record domain.RenewalRecord
	var beforeExpiresAt, afterExpiresAt, createdAt string
	if err := s.Scan(
		&record.ID,
		&record.UserID,
		&record.Days,
		&record.Action,
		&beforeExpiresAt,
		&afterExpiresAt,
		&record.Notes,
		&record.Actor,
		&createdAt,
	); err != nil {
		return domain.RenewalRecord{}, err
	}
	record.BeforeExpiresAt = parseTime(beforeExpiresAt)
	record.AfterExpiresAt = parseTime(afterExpiresAt)
	record.CreatedAt = parseTime(createdAt)
	return record, nil
}

func unexpectedNilErr(name string) error {
	return fmt.Errorf("unexpected nil %s", name)
}
