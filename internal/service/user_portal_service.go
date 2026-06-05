package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"4vpx/internal/domain"
	"4vpx/internal/xray"
)

type PortalDevice struct {
	Slot       domain.DeviceSlot
	VLESSURI   string
	MihomoYAML string
}

type DeviceExportView struct {
	User   domain.User
	Device PortalDevice
	System domain.SystemConfig
}

type PortalView struct {
	User    domain.User
	Devices []PortalDevice
	Renewal []domain.RenewalRecord
	System  domain.SystemConfig
}

type UserPortalService struct {
	users    *UserService
	devices  *DeviceService
	renewals *RenewalService
	system   *SystemService
	renderer *xray.Renderer
}

func NewUserPortalService(users *UserService, devices *DeviceService, renewals *RenewalService, system *SystemService, renderer *xray.Renderer) *UserPortalService {
	return &UserPortalService{
		users:    users,
		devices:  devices,
		renewals: renewals,
		system:   system,
		renderer: renderer,
	}
}

func (s *UserPortalService) GetByToken(ctx context.Context, token string, now time.Time) (PortalView, error) {
	user, err := s.users.GetByAccessToken(ctx, token, now)
	if err != nil {
		return PortalView{}, err
	}
	return s.buildView(ctx, user, now)
}

func (s *UserPortalService) GetByUserID(ctx context.Context, userID int64, now time.Time) (PortalView, error) {
	user, err := s.users.Get(ctx, userID, now)
	if err != nil {
		return PortalView{}, err
	}
	return s.buildView(ctx, user, now)
}

func (s *UserPortalService) GetDeviceByTokenAndSlot(ctx context.Context, token string, slotIndex int, now time.Time) (DeviceExportView, error) {
	user, err := s.users.GetByAccessToken(ctx, token, now)
	if err != nil {
		return DeviceExportView{}, err
	}
	slot, err := s.devices.store.DeviceSlots.GetByUserAndSlotIndex(ctx, user.ID, slotIndex)
	if err != nil {
		return DeviceExportView{}, err
	}
	systemCfg, err := s.system.Get(ctx)
	if err != nil {
		return DeviceExportView{}, err
	}

	record := xray.DeviceRecord{User: user, Slot: slot}
	clientConfig, err := s.renderer.RenderDeviceClientConfig(systemCfg, record)
	if err != nil {
		return DeviceExportView{}, fmt.Errorf("render device %d: %w", slot.SlotIndex, err)
	}
	return DeviceExportView{
		User: user,
		Device: PortalDevice{
			Slot:       slot,
			VLESSURI:   strings.TrimSpace(clientConfig.VLESSURI),
			MihomoYAML: clientConfig.MihomoYAML,
		},
		System: systemCfg,
	}, nil
}

func (s *UserPortalService) buildView(ctx context.Context, user domain.User, now time.Time) (PortalView, error) {
	slots, err := s.devices.ListByUserID(ctx, user.ID)
	if err != nil {
		return PortalView{}, err
	}
	renewals, err := s.renewals.ListByUserID(ctx, user.ID)
	if err != nil {
		return PortalView{}, err
	}
	systemCfg, err := s.system.Get(ctx)
	if err != nil {
		return PortalView{}, err
	}

	devices := make([]PortalDevice, 0, len(slots))
	for _, slot := range slots {
		record := xray.DeviceRecord{User: user, Slot: slot}
		clientConfig, err := s.renderer.RenderDeviceClientConfig(systemCfg, record)
		if err != nil {
			return PortalView{}, fmt.Errorf("render device %d: %w", slot.SlotIndex, err)
		}
		devices = append(devices, PortalDevice{
			Slot:       slot,
			VLESSURI:   strings.TrimSpace(clientConfig.VLESSURI),
			MihomoYAML: clientConfig.MihomoYAML,
		})
	}

	return PortalView{
		User:    user,
		Devices: devices,
		Renewal: renewals,
		System:  systemCfg,
	}, nil
}
