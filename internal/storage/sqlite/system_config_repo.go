package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"4vpx/internal/domain"
)

type SystemConfigRepository struct {
	q DBTX
}

func (r *SystemConfigRepository) Get(ctx context.Context) (domain.SystemConfig, error) {
	row := r.q.QueryRowContext(ctx, `
        SELECT
            id,
            server_address,
            server_port,
            reality_dest,
            reality_server_name,
            client_fingerprint,
            reality_private_key,
            reality_public_key,
            reality_short_id,
            xray_loglevel,
            xray_config_path,
            xray_backup_path,
            xray_bin,
            xray_reload_cmd,
            updated_at
        FROM system_configs
        WHERE id = 1
    `)

	var cfg domain.SystemConfig
	var updatedAt string
	err := row.Scan(
		&cfg.ID,
		&cfg.ServerAddress,
		&cfg.ServerPort,
		&cfg.RealityDest,
		&cfg.RealityServerName,
		&cfg.ClientFingerprint,
		&cfg.RealityPrivateKey,
		&cfg.RealityPublicKey,
		&cfg.RealityShortID,
		&cfg.XrayLogLevel,
		&cfg.XrayConfigPath,
		&cfg.XrayBackupPath,
		&cfg.XrayBin,
		&cfg.XrayReloadCmd,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.SystemConfig{}, nil
		}
		return domain.SystemConfig{}, fmt.Errorf("get system config: %w", err)
	}
	cfg.UpdatedAt = parseTime(updatedAt)
	return cfg, nil
}

func (r *SystemConfigRepository) Upsert(ctx context.Context, cfg domain.SystemConfig) error {
	if cfg.ID == 0 {
		cfg.ID = 1
	}

	_, err := r.q.ExecContext(ctx, `
        INSERT INTO system_configs (
            id,
            server_address,
            server_port,
            reality_dest,
            reality_server_name,
            client_fingerprint,
            reality_private_key,
            reality_public_key,
            reality_short_id,
            xray_loglevel,
            xray_config_path,
            xray_backup_path,
            xray_bin,
            xray_reload_cmd,
            updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            server_address = excluded.server_address,
            server_port = excluded.server_port,
            reality_dest = excluded.reality_dest,
            reality_server_name = excluded.reality_server_name,
            client_fingerprint = excluded.client_fingerprint,
            reality_private_key = excluded.reality_private_key,
            reality_public_key = excluded.reality_public_key,
            reality_short_id = excluded.reality_short_id,
            xray_loglevel = excluded.xray_loglevel,
            xray_config_path = excluded.xray_config_path,
            xray_backup_path = excluded.xray_backup_path,
            xray_bin = excluded.xray_bin,
            xray_reload_cmd = excluded.xray_reload_cmd,
            updated_at = excluded.updated_at
    `,
		cfg.ID,
		cfg.ServerAddress,
		cfg.ServerPort,
		cfg.RealityDest,
		cfg.RealityServerName,
		cfg.ClientFingerprint,
		cfg.RealityPrivateKey,
		cfg.RealityPublicKey,
		cfg.RealityShortID,
		cfg.XrayLogLevel,
		cfg.XrayConfigPath,
		cfg.XrayBackupPath,
		cfg.XrayBin,
		cfg.XrayReloadCmd,
		formatTime(cfg.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert system config: %w", err)
	}
	return nil
}
