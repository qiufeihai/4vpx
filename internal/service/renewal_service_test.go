package service

import (
	"context"
	"testing"
	"time"

	"4vpx/internal/storage/sqlite"
)

func TestRenewalServiceRenewDaysFromCurrentExpiry(t *testing.T) {
	store := newTestStore(t)
	userService := NewUserService(store)
	renewalService := NewRenewalService(store)

	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	user, err := userService.Create(context.Background(), CreateUserInput{
		Name:        "alice",
		Enabled:     true,
		ExpiresAt:   now.Add(72 * time.Hour),
		DeviceSlots: 1,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	updatedUser, record, err := renewalService.RenewDays(context.Background(), user.ID, 7, "admin", "manual", now)
	if err != nil {
		t.Fatalf("renew days: %v", err)
	}

	want := user.ExpiresAt.AddDate(0, 0, 7)
	if !updatedUser.ExpiresAt.Equal(want) {
		t.Fatalf("expires_at = %s, want %s", updatedUser.ExpiresAt, want)
	}
	if record.Days != 7 {
		t.Fatalf("record days = %d, want 7", record.Days)
	}
	if !record.BeforeExpiresAt.Equal(user.ExpiresAt) {
		t.Fatalf("before expires_at = %s, want %s", record.BeforeExpiresAt, user.ExpiresAt)
	}
}

func TestRenewalServiceRenewDaysFromNowWhenExpired(t *testing.T) {
	store := newTestStore(t)
	userService := NewUserService(store)
	renewalService := NewRenewalService(store)

	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	user, err := userService.Create(context.Background(), CreateUserInput{
		Name:        "bob",
		Enabled:     true,
		ExpiresAt:   now.Add(-48 * time.Hour),
		DeviceSlots: 1,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	updatedUser, record, err := renewalService.RenewDays(context.Background(), user.ID, 7, "admin", "recover", now)
	if err != nil {
		t.Fatalf("renew days: %v", err)
	}

	want := now.AddDate(0, 0, 7)
	if !updatedUser.ExpiresAt.Equal(want) {
		t.Fatalf("expires_at = %s, want %s", updatedUser.ExpiresAt, want)
	}
	if !record.BeforeExpiresAt.Equal(user.ExpiresAt) {
		t.Fatalf("before expires_at = %s, want %s", record.BeforeExpiresAt, user.ExpiresAt)
	}
	if !record.AfterExpiresAt.Equal(want) {
		t.Fatalf("after expires_at = %s, want %s", record.AfterExpiresAt, want)
	}
}

func TestRenewalServiceRenewMonthClampsMonthEnd(t *testing.T) {
	store := newTestStore(t)
	userService := NewUserService(store)
	renewalService := NewRenewalService(store)

	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	user, err := userService.Create(context.Background(), CreateUserInput{
		Name:        "carol",
		Enabled:     true,
		ExpiresAt:   now.Add(-24 * time.Hour),
		DeviceSlots: 1,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	updatedUser, record, err := renewalService.RenewMonth(context.Background(), user.ID, "admin", "monthly", now)
	if err != nil {
		t.Fatalf("renew month: %v", err)
	}

	want := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	if !updatedUser.ExpiresAt.Equal(want) {
		t.Fatalf("expires_at = %s, want %s", updatedUser.ExpiresAt, want)
	}
	if record.Days != 28 {
		t.Fatalf("record days = %d, want 28", record.Days)
	}
}

func newTestStore(t *testing.T) *sqlite.Store {
	t.Helper()

	db, err := sqlite.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlite.NewStore(db)
}
