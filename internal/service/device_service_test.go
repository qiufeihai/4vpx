package service

import (
	"context"
	"testing"
	"time"
)

func TestDeviceServiceAdjustResetAndDisable(t *testing.T) {
	store := newTestStore(t)
	userService := NewUserService(store)
	deviceService := NewDeviceService(store)

	user, err := userService.Create(context.Background(), CreateUserInput{
		Name:        "dave",
		Enabled:     true,
		ExpiresAt:   time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
		DeviceSlots: 1,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	slots, err := deviceService.AdjustSlotCount(context.Background(), user.ID, 3)
	if err != nil {
		t.Fatalf("increase slots: %v", err)
	}
	if len(slots) != 3 {
		t.Fatalf("slot count = %d, want 3", len(slots))
	}
	if slots[2].SlotIndex != 3 {
		t.Fatalf("last slot index = %d, want 3", slots[2].SlotIndex)
	}

	beforeUUID := slots[0].UUID
	resetSlot, err := deviceService.ResetSlotUUID(context.Background(), user.ID, 1)
	if err != nil {
		t.Fatalf("reset uuid: %v", err)
	}
	if resetSlot.UUID == beforeUUID {
		t.Fatalf("uuid did not change")
	}

	disabledSlot, err := deviceService.SetSlotEnabled(context.Background(), user.ID, 2, false)
	if err != nil {
		t.Fatalf("disable slot: %v", err)
	}
	if disabledSlot.Enabled {
		t.Fatalf("slot 2 should be disabled")
	}

	slots, err = deviceService.AdjustSlotCount(context.Background(), user.ID, 1)
	if err != nil {
		t.Fatalf("decrease slots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slot count = %d, want 1", len(slots))
	}

	updatedUser, err := userService.Get(context.Background(), user.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if updatedUser.DeviceSlots != 1 {
		t.Fatalf("user device_slots = %d, want 1", updatedUser.DeviceSlots)
	}
}
