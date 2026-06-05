package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppAddr           string
	AppBaseURL        string
	SessionCookieName string
	SessionSecure     bool
	AdminUsername     string
	AdminPassword     string
	SQLitePath        string
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

func Load() (Config, error) {
	cfg := Config{
		AppAddr:           getenv("APP_ADDR", ":8080"),
		AppBaseURL:        getenv("APP_BASE_URL", "http://127.0.0.1:8080"),
		SessionCookieName: getenv("SESSION_COOKIE_NAME", "admin_session"),
		SessionSecure:     getbool("SESSION_SECURE", false),
		AdminUsername:     getenv("ADMIN_USERNAME", "admin"),
		AdminPassword:     getenv("ADMIN_PASSWORD", "change-me-now"),
		SQLitePath:        getenv("SQLITE_PATH", "./data/4vpx.db"),
		ServerAddress:     getenv("SERVER_ADDRESS", ""),
		ServerPort:        getint("SERVER_PORT", 443),
		RealityDest:       getenv("REALITY_DEST", "www.microsoft.com:443"),
		RealityServerName: getenv("REALITY_SERVER_NAME", "www.microsoft.com"),
		ClientFingerprint: getenv("CLIENT_FINGERPRINT", "chrome"),
		RealityPrivateKey: getenv("REALITY_PRIVATE_KEY", ""),
		RealityPublicKey:  getenv("REALITY_PUBLIC_KEY", ""),
		RealityShortID:    getenv("REALITY_SHORT_ID", ""),
		XrayLogLevel:      getenv("XRAY_LOGLEVEL", "warning"),
		XrayConfigPath:    getenv("XRAY_CONFIG_PATH", "./generated/xray-config.json"),
		XrayBackupPath:    getenv("XRAY_BACKUP_PATH", "./generated/xray-config.backup.json"),
		XrayBin:           getenv("XRAY_BIN", ""),
		XrayReloadCmd:     getenv("XRAY_RELOAD_CMD", ""),
	}
	if strings.TrimSpace(cfg.AdminPassword) == "" {
		return Config{}, errors.New("ADMIN_PASSWORD must not be empty")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getint(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getbool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
