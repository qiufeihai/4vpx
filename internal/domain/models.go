package domain

import "time"

type Admin struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AdminSession struct {
	Token     string    `json:"token"`
	AdminID   int64     `json:"admin_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Notes         string    `json:"notes"`
	Enabled       bool      `json:"enabled"`
	ExpiresAt     time.Time `json:"expires_at"`
	AccessToken   string    `json:"access_token"`
	DeviceSlots   int       `json:"device_slots"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	RemainingDays int       `json:"remaining_days,omitempty"`
}

type DeviceSlot struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	SlotIndex      int        `json:"slot_index"`
	Label          string     `json:"label"`
	UUID           string     `json:"uuid"`
	Enabled        bool       `json:"enabled"`
	LastExportedAt *time.Time `json:"last_exported_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type RenewalRecord struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	Days            int       `json:"days"`
	Action          string    `json:"action"`
	BeforeExpiresAt time.Time `json:"before_expires_at"`
	AfterExpiresAt  time.Time `json:"after_expires_at"`
	Notes           string    `json:"notes"`
	Actor           string    `json:"actor"`
	CreatedAt       time.Time `json:"created_at"`
}

type SystemConfig struct {
	ID                int64     `json:"id"`
	ServerAddress     string    `json:"server_address"`
	ServerPort        int       `json:"server_port"`
	RealityDest       string    `json:"reality_dest"`
	RealityServerName string    `json:"reality_server_name"`
	ClientFingerprint string    `json:"client_fingerprint"`
	RealityPrivateKey string    `json:"reality_private_key"`
	RealityPublicKey  string    `json:"reality_public_key"`
	RealityShortID    string    `json:"reality_short_id"`
	XrayLogLevel      string    `json:"xray_loglevel"`
	XrayConfigPath    string    `json:"xray_config_path"`
	XrayBackupPath    string    `json:"xray_backup_path"`
	XrayBin           string    `json:"xray_bin"`
	XrayReloadCmd     string    `json:"xray_reload_cmd"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ExportBundle struct {
	ExportedAt   time.Time       `json:"exported_at"`
	Admins       []Admin         `json:"admins"`
	Users        []User          `json:"users"`
	DeviceSlots  []DeviceSlot    `json:"device_slots"`
	Renewals     []RenewalRecord `json:"renewals"`
	SystemConfig SystemConfig    `json:"system_config"`
}
