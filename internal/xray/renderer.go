package xray

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"time"

	"4vpx/internal/domain"
)

//go:embed templates/*
var templateFS embed.FS

type DeviceRecord struct {
	User domain.User
	Slot domain.DeviceSlot
}

type DeviceClientConfig struct {
	Device     DeviceRecord
	VLESSURI   string
	MihomoYAML string
}

type Renderer struct {
	serverTpl *template.Template
	uriTpl    *template.Template
	mihomoTpl *template.Template
}

func NewRenderer() (*Renderer, error) {
	serverTpl, err := template.ParseFS(templateFS, "templates/config.server.json.tpl")
	if err != nil {
		return nil, fmt.Errorf("parse xray server template: %w", err)
	}
	uriTpl, err := template.ParseFS(templateFS, "templates/client.vless.uri.tpl")
	if err != nil {
		return nil, fmt.Errorf("parse xray uri template: %w", err)
	}
	mihomoTpl, err := template.ParseFS(templateFS, "templates/client.mihomo.yaml.tpl")
	if err != nil {
		return nil, fmt.Errorf("parse xray mihomo template: %w", err)
	}
	return &Renderer{
		serverTpl: serverTpl,
		uriTpl:    uriTpl,
		mihomoTpl: mihomoTpl,
	}, nil
}

func (r *Renderer) ActiveDevices(devices []DeviceRecord, now time.Time) []DeviceRecord {
	now = now.UTC()
	active := make([]DeviceRecord, 0, len(devices))
	for _, device := range devices {
		if device.User.Enabled && device.Slot.Enabled && device.User.ExpiresAt.After(now) {
			active = append(active, device)
		}
	}
	return active
}

func (r *Renderer) RenderServerConfig(cfg domain.SystemConfig, devices []DeviceRecord, now time.Time) ([]byte, error) {
	active := r.ActiveDevices(devices, now)

	data := struct {
		XrayLogLevel      string
		ServerPort        int
		RealityDest       string
		RealityServerName string
		RealityPrivateKey string
		RealityShortID    string
		Clients           []struct {
			UUID string
		}
	}{
		XrayLogLevel:      cfg.XrayLogLevel,
		ServerPort:        cfg.ServerPort,
		RealityDest:       cfg.RealityDest,
		RealityServerName: cfg.RealityServerName,
		RealityPrivateKey: cfg.RealityPrivateKey,
		RealityShortID:    cfg.RealityShortID,
	}
	for _, device := range active {
		data.Clients = append(data.Clients, struct{ UUID string }{UUID: device.Slot.UUID})
	}

	var raw bytes.Buffer
	if err := r.serverTpl.Execute(&raw, data); err != nil {
		return nil, fmt.Errorf("render xray server config: %w", err)
	}

	var normalized bytes.Buffer
	if err := json.Indent(&normalized, raw.Bytes(), "", "  "); err != nil {
		return nil, fmt.Errorf("validate rendered xray config: %w", err)
	}
	normalized.WriteByte('\n')
	return normalized.Bytes(), nil
}

func (r *Renderer) RenderClientURI(cfg domain.SystemConfig, device DeviceRecord) (string, error) {
	data := struct {
		UUID              string
		ServerAddress     string
		ServerPort        int
		RealityServerName string
		ClientFingerprint string
		RealityPublicKey  string
		RealityShortID    string
		Tag               string
	}{
		UUID:              device.Slot.UUID,
		ServerAddress:     cfg.ServerAddress,
		ServerPort:        cfg.ServerPort,
		RealityServerName: cfg.RealityServerName,
		ClientFingerprint: cfg.ClientFingerprint,
		RealityPublicKey:  cfg.RealityPublicKey,
		RealityShortID:    cfg.RealityShortID,
		Tag:               url.QueryEscape(deviceName(device)),
	}

	var buf bytes.Buffer
	if err := r.uriTpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render vless uri: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func (r *Renderer) RenderClientMihomoYAML(cfg domain.SystemConfig, device DeviceRecord) (string, error) {
	data := struct {
		Name              string
		UUID              string
		ServerAddress     string
		ServerPort        int
		RealityServerName string
		ClientFingerprint string
		RealityPublicKey  string
		RealityShortID    string
	}{
		Name:              deviceName(device),
		UUID:              device.Slot.UUID,
		ServerAddress:     cfg.ServerAddress,
		ServerPort:        cfg.ServerPort,
		RealityServerName: cfg.RealityServerName,
		ClientFingerprint: cfg.ClientFingerprint,
		RealityPublicKey:  cfg.RealityPublicKey,
		RealityShortID:    cfg.RealityShortID,
	}

	var buf bytes.Buffer
	if err := r.mihomoTpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render mihomo yaml: %w", err)
	}
	return strings.TrimSpace(buf.String()) + "\n", nil
}

func (r *Renderer) RenderDeviceClientConfig(cfg domain.SystemConfig, device DeviceRecord) (DeviceClientConfig, error) {
	uri, err := r.RenderClientURI(cfg, device)
	if err != nil {
		return DeviceClientConfig{}, err
	}
	mihomoYAML, err := r.RenderClientMihomoYAML(cfg, device)
	if err != nil {
		return DeviceClientConfig{}, err
	}
	return DeviceClientConfig{
		Device:     device,
		VLESSURI:   uri,
		MihomoYAML: mihomoYAML,
	}, nil
}

func deviceName(device DeviceRecord) string {
	label := strings.TrimSpace(device.Slot.Label)
	if label == "" {
		label = fmt.Sprintf("Device %d", device.Slot.SlotIndex)
	}
	userName := strings.TrimSpace(device.User.Name)
	if userName == "" {
		return label
	}
	return userName + " - " + label
}
