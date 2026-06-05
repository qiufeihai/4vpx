package service

import (
	"context"
	"strings"
	"time"

	"4vpx/internal/domain"
	"4vpx/internal/storage/sqlite"
)

type UpdateSystemInput struct {
	ServerAddress     string
	ServerPort        int
	RealityDest       string
	RealityServerName string
	ClientFingerprint string
	RealityPrivateKey string
	RealityPublicKey  string
	RealityShortID    string
	XrayLogLevel      string
	XrayConfigPath    string
	XrayBackupPath    string
	XrayBin           string
	XrayReloadCmd     string
}

type SystemService struct {
	store    *sqlite.Store
	defaults domain.SystemConfig
}

func NewSystemService(store *sqlite.Store, defaults domain.SystemConfig) *SystemService {
	defaults.ID = 1
	if defaults.UpdatedAt.IsZero() {
		defaults.UpdatedAt = time.Now().UTC()
	}
	return &SystemService{store: store, defaults: defaults}
}

func (s *SystemService) Ensure(ctx context.Context) (domain.SystemConfig, error) {
	cfg, err := s.store.System.Get(ctx)
	if err != nil {
		return domain.SystemConfig{}, err
	}
	if cfg.ID == 0 {
		cfg = s.defaults
		cfg.ID = 1
		cfg.UpdatedAt = time.Now().UTC()
		if err := s.store.System.Upsert(ctx, cfg); err != nil {
			return domain.SystemConfig{}, err
		}
	}

	merged := s.mergeDefaults(cfg)
	if merged != cfg {
		merged.UpdatedAt = time.Now().UTC()
		if err := s.store.System.Upsert(ctx, merged); err != nil {
			return domain.SystemConfig{}, err
		}
	}
	return merged, nil
}

func (s *SystemService) Get(ctx context.Context) (domain.SystemConfig, error) {
	return s.Ensure(ctx)
}

func (s *SystemService) Update(ctx context.Context, input UpdateSystemInput) (domain.SystemConfig, error) {
	cfg, err := s.Ensure(ctx)
	if err != nil {
		return domain.SystemConfig{}, err
	}

	cfg.ServerAddress = strings.TrimSpace(input.ServerAddress)
	cfg.ServerPort = input.ServerPort
	cfg.RealityDest = strings.TrimSpace(input.RealityDest)
	cfg.RealityServerName = strings.TrimSpace(input.RealityServerName)
	cfg.ClientFingerprint = strings.TrimSpace(input.ClientFingerprint)
	cfg.RealityPrivateKey = strings.TrimSpace(input.RealityPrivateKey)
	cfg.RealityPublicKey = strings.TrimSpace(input.RealityPublicKey)
	cfg.RealityShortID = strings.TrimSpace(input.RealityShortID)
	cfg.XrayLogLevel = strings.TrimSpace(input.XrayLogLevel)
	cfg.XrayConfigPath = strings.TrimSpace(input.XrayConfigPath)
	cfg.XrayBackupPath = strings.TrimSpace(input.XrayBackupPath)
	cfg.XrayBin = strings.TrimSpace(input.XrayBin)
	cfg.XrayReloadCmd = strings.TrimSpace(input.XrayReloadCmd)
	cfg.UpdatedAt = time.Now().UTC()

	cfg = s.mergeDefaults(cfg)
	if cfg.ServerPort <= 0 {
		cfg.ServerPort = 443
	}
	if cfg.ClientFingerprint == "" {
		cfg.ClientFingerprint = "chrome"
	}
	if cfg.XrayLogLevel == "" {
		cfg.XrayLogLevel = "warning"
	}

	if err := s.store.System.Upsert(ctx, cfg); err != nil {
		return domain.SystemConfig{}, err
	}
	return cfg, nil
}

func (s *SystemService) mergeDefaults(cfg domain.SystemConfig) domain.SystemConfig {
	if cfg.ID == 0 {
		cfg.ID = 1
	}
	if strings.TrimSpace(cfg.ServerAddress) == "" {
		cfg.ServerAddress = s.defaults.ServerAddress
	}
	if cfg.ServerPort == 0 {
		cfg.ServerPort = s.defaults.ServerPort
	}
	if strings.TrimSpace(cfg.RealityDest) == "" {
		cfg.RealityDest = s.defaults.RealityDest
	}
	if strings.TrimSpace(cfg.RealityServerName) == "" {
		cfg.RealityServerName = s.defaults.RealityServerName
	}
	if strings.TrimSpace(cfg.ClientFingerprint) == "" {
		cfg.ClientFingerprint = s.defaults.ClientFingerprint
	}
	if strings.TrimSpace(cfg.RealityPrivateKey) == "" {
		cfg.RealityPrivateKey = s.defaults.RealityPrivateKey
	}
	if strings.TrimSpace(cfg.RealityPublicKey) == "" {
		cfg.RealityPublicKey = s.defaults.RealityPublicKey
	}
	if strings.TrimSpace(cfg.RealityShortID) == "" {
		cfg.RealityShortID = s.defaults.RealityShortID
	}
	if strings.TrimSpace(cfg.XrayLogLevel) == "" {
		cfg.XrayLogLevel = s.defaults.XrayLogLevel
	}
	if strings.TrimSpace(cfg.XrayConfigPath) == "" {
		cfg.XrayConfigPath = s.defaults.XrayConfigPath
	}
	if strings.TrimSpace(cfg.XrayBackupPath) == "" {
		cfg.XrayBackupPath = s.defaults.XrayBackupPath
	}
	if strings.TrimSpace(cfg.XrayBin) == "" {
		cfg.XrayBin = s.defaults.XrayBin
	}
	if strings.TrimSpace(cfg.XrayReloadCmd) == "" {
		cfg.XrayReloadCmd = s.defaults.XrayReloadCmd
	}
	if cfg.UpdatedAt.IsZero() {
		cfg.UpdatedAt = s.defaults.UpdatedAt
	}
	return cfg
}
