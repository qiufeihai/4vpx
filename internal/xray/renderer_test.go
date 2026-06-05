package xray

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"4vpx/internal/domain"
)

func TestRenderServerConfigFiltersInactiveDevices(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}

	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	cfg := domain.SystemConfig{
		ServerAddress:     "example.com",
		ServerPort:        443,
		RealityDest:       "www.microsoft.com:443",
		RealityServerName: "www.microsoft.com",
		ClientFingerprint: "chrome",
		RealityPrivateKey: "private-key",
		RealityPublicKey:  "public-key",
		RealityShortID:    "abcd1234",
		XrayLogLevel:      "warning",
	}

	devices := []DeviceRecord{
		{
			User: domain.User{Name: "alice", Enabled: true, ExpiresAt: now.Add(24 * time.Hour)},
			Slot: domain.DeviceSlot{SlotIndex: 1, Label: "phone", UUID: "uuid-active", Enabled: true},
		},
		{
			User: domain.User{Name: "alice", Enabled: true, ExpiresAt: now.Add(-time.Hour)},
			Slot: domain.DeviceSlot{SlotIndex: 2, Label: "expired", UUID: "uuid-expired", Enabled: true},
		},
		{
			User: domain.User{Name: "bob", Enabled: false, ExpiresAt: now.Add(24 * time.Hour)},
			Slot: domain.DeviceSlot{SlotIndex: 1, Label: "disabled-user", UUID: "uuid-disabled-user", Enabled: true},
		},
		{
			User: domain.User{Name: "carol", Enabled: true, ExpiresAt: now.Add(24 * time.Hour)},
			Slot: domain.DeviceSlot{SlotIndex: 1, Label: "disabled-slot", UUID: "uuid-disabled-slot", Enabled: false},
		},
	}

	output, err := renderer.RenderServerConfig(cfg, devices, now)
	if err != nil {
		t.Fatalf("RenderServerConfig() error = %v", err)
	}
	if !json.Valid(output) {
		t.Fatalf("RenderServerConfig() produced invalid JSON: %s", output)
	}

	rendered := string(output)
	if !strings.Contains(rendered, "uuid-active") {
		t.Fatalf("active uuid missing from rendered config: %s", rendered)
	}
	for _, unexpected := range []string{"uuid-expired", "uuid-disabled-user", "uuid-disabled-slot"} {
		if strings.Contains(rendered, unexpected) {
			t.Fatalf("unexpected uuid %q present in rendered config: %s", unexpected, rendered)
		}
	}

	clientConfig, err := renderer.RenderDeviceClientConfig(cfg, devices[0])
	if err != nil {
		t.Fatalf("RenderDeviceClientConfig() error = %v", err)
	}
	if !strings.Contains(clientConfig.VLESSURI, "vless://uuid-active@example.com:443") {
		t.Fatalf("unexpected vless URI: %s", clientConfig.VLESSURI)
	}
	if !strings.Contains(clientConfig.MihomoYAML, "uuid: uuid-active") {
		t.Fatalf("unexpected mihomo YAML: %s", clientConfig.MihomoYAML)
	}
	if strings.Contains(clientConfig.MihomoYAML, "proxy-groups:") {
		t.Fatalf("unexpected proxy-groups in mihomo YAML: %s", clientConfig.MihomoYAML)
	}
	if strings.Contains(clientConfig.MihomoYAML, "MATCH,PROXY") {
		t.Fatalf("unexpected legacy MATCH,PROXY rule in mihomo YAML: %s", clientConfig.MihomoYAML)
	}
}
