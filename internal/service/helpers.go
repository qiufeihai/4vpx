package service

import (
	"fmt"
	"time"

	"4vpx/internal/domain"
	"4vpx/internal/security"
)

func newDeviceSlot(userID int64, slotIndex int, now time.Time) (domain.DeviceSlot, error) {
	uuid, err := security.NewUUIDLike()
	if err != nil {
		return domain.DeviceSlot{}, err
	}

	return domain.DeviceSlot{
		UserID:    userID,
		SlotIndex: slotIndex,
		Label:     fmt.Sprintf("Device %d", slotIndex),
		UUID:      uuid,
		Enabled:   true,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
	}, nil
}

func remainingDays(expiresAt, now time.Time) int {
	expiresAt = expiresAt.UTC()
	now = now.UTC()
	if !expiresAt.After(now) {
		return 0
	}

	d := expiresAt.Sub(now)
	days := int(d / (24 * time.Hour))
	if d%(24*time.Hour) != 0 {
		days++
	}
	return days
}

func effectiveRenewalBase(expiresAt, now time.Time) time.Time {
	expiresAt = expiresAt.UTC()
	now = now.UTC()
	if expiresAt.After(now) {
		return expiresAt
	}
	return now
}

func wholeDaysBetween(start, end time.Time) int {
	if !end.After(start) {
		return 0
	}

	d := end.Sub(start)
	days := int(d / (24 * time.Hour))
	if d%(24*time.Hour) != 0 {
		days++
	}
	return days
}

func addMonthsClamped(t time.Time, months int) time.Time {
	t = t.UTC()
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	targetMonthStart := time.Date(year, month+time.Month(months), 1, hour, min, sec, t.Nanosecond(), time.UTC)
	lastDay := time.Date(targetMonthStart.Year(), targetMonthStart.Month()+1, 0, hour, min, sec, t.Nanosecond(), time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetMonthStart.Year(), targetMonthStart.Month(), day, hour, min, sec, t.Nanosecond(), time.UTC)
}
