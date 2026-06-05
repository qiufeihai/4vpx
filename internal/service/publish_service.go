package service

import (
	"context"

	"4vpx/internal/domain"
	"4vpx/internal/storage/sqlite"
	"4vpx/internal/xray"
)

type PublishService struct {
	store   *sqlite.Store
	system  *SystemService
	runtime *xray.Runtime
}

func NewPublishService(store *sqlite.Store, system *SystemService, runtime *xray.Runtime) *PublishService {
	return &PublishService{
		store:   store,
		system:  system,
		runtime: runtime,
	}
}

func (s *PublishService) Publish(ctx context.Context) (xray.PublishResult, error) {
	cfg, err := s.system.Get(ctx)
	if err != nil {
		return xray.PublishResult{}, err
	}

	users, err := s.store.Users.List(ctx)
	if err != nil {
		return xray.PublishResult{}, err
	}
	slots, err := s.store.DeviceSlots.List(ctx)
	if err != nil {
		return xray.PublishResult{}, err
	}

	userByID := make(map[int64]domain.User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}

	devices := make([]xray.DeviceRecord, 0, len(slots))
	for _, slot := range slots {
		user, ok := userByID[slot.UserID]
		if !ok {
			continue
		}
		devices = append(devices, xray.DeviceRecord{
			User: user,
			Slot: slot,
		})
	}

	return s.runtime.Publish(ctx, cfg, devices)
}
