package service

import (
	"context"
	"testing"
	"time"
)

func TestUserServiceListFilteredAppliesQueryStatusAndExpiry(t *testing.T) {
	store := newTestStore(t)
	userService := NewUserService(store)

	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	users := []CreateUserInput{
		{Name: "alpha", Notes: "vip", Enabled: true, ExpiresAt: now.AddDate(0, 0, 2), DeviceSlots: 1},
		{Name: "beta", Notes: "trial", Enabled: true, ExpiresAt: now.AddDate(0, 0, 10), DeviceSlots: 1},
		{Name: "gamma", Notes: "vip", Enabled: false, ExpiresAt: now.AddDate(0, 0, 1), DeviceSlots: 1},
		{Name: "delta", Notes: "expired", Enabled: true, ExpiresAt: now.AddDate(0, 0, -1), DeviceSlots: 1},
	}
	for _, input := range users {
		if _, err := userService.Create(context.Background(), input); err != nil {
			t.Fatalf("create user %q: %v", input.Name, err)
		}
	}

	page, err := userService.ListFiltered(context.Background(), UserListFilter{
		Query:    "vip",
		Status:   "enabled",
		Expiry:   "expiring_3d",
		Page:     1,
		PageSize: 20,
	}, now)
	if err != nil {
		t.Fatalf("ListFiltered() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1", page.Total)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "alpha" {
		t.Fatalf("items = %+v, want only alpha", page.Items)
	}
}

func TestUserServiceListFilteredPaginates(t *testing.T) {
	store := newTestStore(t)
	userService := NewUserService(store)

	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	for i := 1; i <= 5; i++ {
		if _, err := userService.Create(context.Background(), CreateUserInput{
			Name:        "user-" + string(rune('0'+i)),
			Enabled:     true,
			ExpiresAt:   now.AddDate(0, 0, i),
			DeviceSlots: 1,
		}); err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
	}

	page, err := userService.ListFiltered(context.Background(), UserListFilter{
		Page:     2,
		PageSize: 2,
	}, now)
	if err != nil {
		t.Fatalf("ListFiltered() error = %v", err)
	}
	if page.Total != 5 {
		t.Fatalf("total = %d, want 5", page.Total)
	}
	if page.TotalPages != 3 {
		t.Fatalf("total pages = %d, want 3", page.TotalPages)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(page.Items))
	}
	if page.Page != 2 {
		t.Fatalf("page = %d, want 2", page.Page)
	}
}
