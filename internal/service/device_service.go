package service

import (
	"context"
	"time"

	"4vpx/internal/domain"
	"4vpx/internal/security"
	"4vpx/internal/storage/sqlite"
)

type DeviceService struct {
	store *sqlite.Store
}

func NewDeviceService(store *sqlite.Store) *DeviceService {
	return &DeviceService{store: store}
}

func (s *DeviceService) ListByUserID(ctx context.Context, userID int64) ([]domain.DeviceSlot, error) {
	return s.store.DeviceSlots.ListByUserID(ctx, userID)
}

func (s *DeviceService) AdjustSlotCount(ctx context.Context, userID int64, targetCount int) ([]domain.DeviceSlot, error) {
	if targetCount <= 0 {
		return nil, ErrInvalidDeviceSlotCount
	}

	now := time.Now().UTC()
	var updatedSlots []domain.DeviceSlot
	err := s.store.WithTx(ctx, func(tx *sqlite.Store) error {
		user, err := tx.Users.GetByID(ctx, userID)
		if err != nil {
			return err
		}
		slots, err := tx.DeviceSlots.ListByUserID(ctx, userID)
		if err != nil {
			return err
		}

		currentCount := len(slots)
		if currentCount < targetCount {
			for slotIndex := currentCount + 1; slotIndex <= targetCount; slotIndex++ {
				slot, err := newDeviceSlot(userID, slotIndex, now)
				if err != nil {
					return err
				}
				createdSlot, err := tx.DeviceSlots.Create(ctx, slot)
				if err != nil {
					return err
				}
				slots = append(slots, createdSlot)
			}
		}
		if currentCount > targetCount {
			for slotIndex := currentCount; slotIndex > targetCount; slotIndex-- {
				if err := tx.DeviceSlots.DeleteByUserAndSlotIndex(ctx, userID, slotIndex); err != nil {
					return err
				}
			}
			slots = slots[:targetCount]
		}

		user.DeviceSlots = targetCount
		user.UpdatedAt = now
		if err := tx.Users.Update(ctx, user); err != nil {
			return err
		}

		updatedSlots = slots
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updatedSlots, nil
}

func (s *DeviceService) ResetSlotUUID(ctx context.Context, userID int64, slotIndex int) (domain.DeviceSlot, error) {
	slot, err := s.store.DeviceSlots.GetByUserAndSlotIndex(ctx, userID, slotIndex)
	if err != nil {
		return domain.DeviceSlot{}, err
	}

	uuid, err := security.NewUUIDLike()
	if err != nil {
		return domain.DeviceSlot{}, err
	}
	slot.UUID = uuid
	slot.UpdatedAt = time.Now().UTC()

	if err := s.store.DeviceSlots.Update(ctx, slot); err != nil {
		return domain.DeviceSlot{}, err
	}
	return slot, nil
}

func (s *DeviceService) SetSlotEnabled(ctx context.Context, userID int64, slotIndex int, enabled bool) (domain.DeviceSlot, error) {
	slot, err := s.store.DeviceSlots.GetByUserAndSlotIndex(ctx, userID, slotIndex)
	if err != nil {
		return domain.DeviceSlot{}, err
	}
	slot.Enabled = enabled
	slot.UpdatedAt = time.Now().UTC()

	if err := s.store.DeviceSlots.Update(ctx, slot); err != nil {
		return domain.DeviceSlot{}, err
	}
	return slot, nil
}

func (s *DeviceService) MarkExported(ctx context.Context, userID int64, slotIndex int, exportedAt time.Time) (domain.DeviceSlot, error) {
	slot, err := s.store.DeviceSlots.GetByUserAndSlotIndex(ctx, userID, slotIndex)
	if err != nil {
		return domain.DeviceSlot{}, err
	}
	exportedAt = exportedAt.UTC()
	slot.LastExportedAt = &exportedAt
	slot.UpdatedAt = time.Now().UTC()

	if err := s.store.DeviceSlots.Update(ctx, slot); err != nil {
		return domain.DeviceSlot{}, err
	}
	return slot, nil
}
