package config

import "testing"

func TestLoadDefaultsAutoPublishInterval(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret")
	t.Setenv("AUTO_PUBLISH_INTERVAL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.AutoPublishInterval.String(); got != "15m0s" {
		t.Fatalf("AutoPublishInterval = %s, want 15m0s", got)
	}
}

func TestLoadParsesAutoPublishInterval(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret")
	t.Setenv("AUTO_PUBLISH_INTERVAL", "30m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.AutoPublishInterval.String(); got != "30m0s" {
		t.Fatalf("AutoPublishInterval = %s, want 30m0s", got)
	}
}
