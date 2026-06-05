package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"4vpx/internal/domain"
	sqliteStore "4vpx/internal/storage/sqlite"
)

type Exporter struct {
	store *sqliteStore.Store
	now   func() time.Time
}

func NewExporter(db *sql.DB) *Exporter {
	return &Exporter{
		store: sqliteStore.NewStore(db),
		now:   func() time.Time { return time.Now().UTC() },
	}
}

func (e *Exporter) Export(ctx context.Context) (domain.ExportBundle, error) {
	admins, err := e.store.Admins.List(ctx)
	if err != nil {
		return domain.ExportBundle{}, err
	}
	users, err := e.store.Users.List(ctx)
	if err != nil {
		return domain.ExportBundle{}, err
	}
	deviceSlots, err := e.store.DeviceSlots.List(ctx)
	if err != nil {
		return domain.ExportBundle{}, err
	}
	renewals, err := e.store.Renewals.List(ctx)
	if err != nil {
		return domain.ExportBundle{}, err
	}
	systemConfig, err := e.store.System.Get(ctx)
	if err != nil {
		return domain.ExportBundle{}, err
	}

	return domain.ExportBundle{
		ExportedAt:   e.now(),
		Admins:       admins,
		Users:        users,
		DeviceSlots:  deviceSlots,
		Renewals:     renewals,
		SystemConfig: systemConfig,
	}, nil
}

func (e *Exporter) ExportJSON(ctx context.Context) ([]byte, error) {
	bundle, err := e.Export(ctx)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal export bundle: %w", err)
	}
	return append(data, '\n'), nil
}
