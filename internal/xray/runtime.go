package xray

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"4vpx/internal/domain"
)

type PublishResult struct {
	ConfigPath    string
	BackupPath    string
	ActiveClients int
	Reloaded      bool
}

type Runtime struct {
	renderer        *Renderer
	now             func() time.Time
	validateTimeout time.Duration
	reloadTimeout   time.Duration
}

func NewRuntime(renderer *Renderer) *Runtime {
	return &Runtime{
		renderer:        renderer,
		now:             func() time.Time { return time.Now().UTC() },
		validateTimeout: 10 * time.Second,
		reloadTimeout:   10 * time.Second,
	}
}

func (r *Runtime) Publish(ctx context.Context, cfg domain.SystemConfig, devices []DeviceRecord) (PublishResult, error) {
	if strings.TrimSpace(cfg.XrayConfigPath) == "" {
		return PublishResult{}, fmt.Errorf("xray config path is empty")
	}

	now := r.now()
	rendered, err := r.renderer.RenderServerConfig(cfg, devices, now)
	if err != nil {
		return PublishResult{}, err
	}

	backupPath := strings.TrimSpace(cfg.XrayBackupPath)
	if backupPath == "" {
		backupPath = cfg.XrayConfigPath + ".bak"
	}

	if err := os.MkdirAll(filepath.Dir(cfg.XrayConfigPath), 0o755); err != nil {
		return PublishResult{}, fmt.Errorf("mkdir xray config dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return PublishResult{}, fmt.Errorf("mkdir xray backup dir: %w", err)
	}

	tempPath := tempConfigPath(cfg.XrayConfigPath)
	if err := os.WriteFile(tempPath, rendered, 0o644); err != nil {
		return PublishResult{}, fmt.Errorf("write xray temp config: %w", err)
	}
	defer os.Remove(tempPath)

	if err := r.validate(ctx, cfg.XrayBin, tempPath); err != nil {
		return PublishResult{}, err
	}

	hadPreviousConfig := fileExists(cfg.XrayConfigPath)
	if hadPreviousConfig {
		if err := copyFile(cfg.XrayConfigPath, backupPath); err != nil {
			return PublishResult{}, fmt.Errorf("backup xray config: %w", err)
		}
	}

	if err := os.Rename(tempPath, cfg.XrayConfigPath); err != nil {
		return PublishResult{}, fmt.Errorf("promote xray config: %w", err)
	}

	result := PublishResult{
		ConfigPath:    cfg.XrayConfigPath,
		BackupPath:    backupPath,
		ActiveClients: len(r.renderer.ActiveDevices(devices, now)),
	}

	if strings.TrimSpace(cfg.XrayReloadCmd) == "" {
		return result, nil
	}
	if err := r.reload(ctx, cfg.XrayReloadCmd); err != nil {
		if hadPreviousConfig {
			rollbackErr := copyFile(backupPath, cfg.XrayConfigPath)
			if rollbackErr == nil {
				rollbackErr = r.reload(ctx, cfg.XrayReloadCmd)
			}
			if rollbackErr != nil {
				return result, fmt.Errorf("reload xray failed: %w; rollback failed: %v", err, rollbackErr)
			}
		}
		return result, fmt.Errorf("reload xray failed: %w", err)
	}

	result.Reloaded = true
	return result, nil
}

func (r *Runtime) validate(ctx context.Context, xrayBin string, configPath string) error {
	xrayBin = strings.TrimSpace(xrayBin)
	if xrayBin == "" {
		return nil
	}

	validateCtx, cancel := context.WithTimeout(ctx, r.validateTimeout)
	defer cancel()

	cmd := exec.CommandContext(validateCtx, xrayBin, "run", "-test", "-config", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("validate xray config: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (r *Runtime) reload(ctx context.Context, reloadCmd string) error {
	reloadCtx, cancel := context.WithTimeout(ctx, r.reloadTimeout)
	defer cancel()

	cmd := exec.CommandContext(reloadCtx, "/bin/sh", "-c", reloadCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func tempConfigPath(configPath string) string {
	ext := filepath.Ext(configPath)
	if ext == "" {
		return configPath + ".tmp.json"
	}
	base := strings.TrimSuffix(configPath, ext)
	return base + ".tmp" + ext
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
