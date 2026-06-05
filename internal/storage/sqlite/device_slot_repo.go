package sqlite

import (
	"context"
	"fmt"

	"4vpx/internal/domain"
)

type DeviceSlotRepository struct {
	q DBTX
}

func (r *DeviceSlotRepository) List(ctx context.Context) ([]domain.DeviceSlot, error) {
	rows, err := r.q.QueryContext(ctx, `
        SELECT id, user_id, slot_index, label, uuid, enabled, last_exported_at, created_at, updated_at
        FROM device_slots
        ORDER BY user_id ASC, slot_index ASC, id ASC
    `)
	if err != nil {
		return nil, fmt.Errorf("list all device slots: %w", err)
	}
	defer rows.Close()

	slots := make([]domain.DeviceSlot, 0)
	for rows.Next() {
		slot, err := scanDeviceSlot(rows)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all device slots: %w", err)
	}
	return slots, nil
}

func (r *DeviceSlotRepository) Create(ctx context.Context, slot domain.DeviceSlot) (domain.DeviceSlot, error) {
	res, err := r.q.ExecContext(ctx, `
        INSERT INTO device_slots (
            user_id, slot_index, label, uuid, enabled, last_exported_at, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `,
		slot.UserID,
		slot.SlotIndex,
		slot.Label,
		slot.UUID,
		boolToInt(slot.Enabled),
		formatNullableTime(slot.LastExportedAt),
		formatTime(slot.CreatedAt),
		formatTime(slot.UpdatedAt),
	)
	if err != nil {
		return domain.DeviceSlot{}, fmt.Errorf("insert device slot: %w", err)
	}

	slot.ID, err = res.LastInsertId()
	if err != nil {
		return domain.DeviceSlot{}, fmt.Errorf("device slot last insert id: %w", err)
	}
	return slot, nil
}

func (r *DeviceSlotRepository) Update(ctx context.Context, slot domain.DeviceSlot) error {
	res, err := r.q.ExecContext(ctx, `
        UPDATE device_slots
        SET label = ?, uuid = ?, enabled = ?, last_exported_at = ?, updated_at = ?
        WHERE user_id = ? AND slot_index = ?
    `,
		slot.Label,
		slot.UUID,
		boolToInt(slot.Enabled),
		formatNullableTime(slot.LastExportedAt),
		formatTime(slot.UpdatedAt),
		slot.UserID,
		slot.SlotIndex,
	)
	if err != nil {
		return fmt.Errorf("update device slot: %w", err)
	}
	return ensureRowsAffected(res)
}

func (r *DeviceSlotRepository) GetByUserAndSlotIndex(ctx context.Context, userID int64, slotIndex int) (domain.DeviceSlot, error) {
	return scanDeviceSlot(r.q.QueryRowContext(ctx, `
        SELECT id, user_id, slot_index, label, uuid, enabled, last_exported_at, created_at, updated_at
        FROM device_slots
        WHERE user_id = ? AND slot_index = ?
    `, userID, slotIndex))
}

func (r *DeviceSlotRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.DeviceSlot, error) {
	rows, err := r.q.QueryContext(ctx, `
        SELECT id, user_id, slot_index, label, uuid, enabled, last_exported_at, created_at, updated_at
        FROM device_slots
        WHERE user_id = ?
        ORDER BY slot_index ASC
    `, userID)
	if err != nil {
		return nil, fmt.Errorf("list device slots: %w", err)
	}
	defer rows.Close()

	slots := make([]domain.DeviceSlot, 0)
	for rows.Next() {
		slot, err := scanDeviceSlot(rows)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device slots: %w", err)
	}
	return slots, nil
}

func (r *DeviceSlotRepository) DeleteByUserAndSlotIndex(ctx context.Context, userID int64, slotIndex int) error {
	res, err := r.q.ExecContext(ctx, `
        DELETE FROM device_slots
        WHERE user_id = ? AND slot_index = ?
    `, userID, slotIndex)
	if err != nil {
		return fmt.Errorf("delete device slot: %w", err)
	}
	return ensureRowsAffected(res)
}
