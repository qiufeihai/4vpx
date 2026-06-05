package sqlite

import "database/sql"

func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admins (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL
        );`,
		`CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            notes TEXT NOT NULL DEFAULT '',
            enabled INTEGER NOT NULL DEFAULT 1,
            expires_at TEXT NOT NULL,
            access_token TEXT NOT NULL UNIQUE,
            device_slots INTEGER NOT NULL DEFAULT 1,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL
        );`,
		`CREATE TABLE IF NOT EXISTS admin_sessions (
            token TEXT PRIMARY KEY,
            admin_id INTEGER NOT NULL,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL,
            FOREIGN KEY(admin_id) REFERENCES admins(id) ON DELETE CASCADE
        );`,
		`CREATE TABLE IF NOT EXISTS device_slots (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            slot_index INTEGER NOT NULL,
            label TEXT NOT NULL,
            uuid TEXT NOT NULL UNIQUE,
            enabled INTEGER NOT NULL DEFAULT 1,
            last_exported_at TEXT,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL,
            UNIQUE(user_id, slot_index),
            FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
        );`,
		`CREATE TABLE IF NOT EXISTS renewal_records (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            days INTEGER NOT NULL,
            action TEXT NOT NULL,
            before_expires_at TEXT NOT NULL,
            after_expires_at TEXT NOT NULL,
            notes TEXT NOT NULL DEFAULT '',
            actor TEXT NOT NULL,
            created_at TEXT NOT NULL,
            FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
        );`,
		`CREATE TABLE IF NOT EXISTS system_configs (
            id INTEGER PRIMARY KEY CHECK (id = 1),
            server_address TEXT NOT NULL,
            server_port INTEGER NOT NULL,
            reality_dest TEXT NOT NULL,
            reality_server_name TEXT NOT NULL,
            client_fingerprint TEXT NOT NULL,
            reality_private_key TEXT NOT NULL,
            reality_public_key TEXT NOT NULL,
            reality_short_id TEXT NOT NULL,
            xray_loglevel TEXT NOT NULL,
            xray_config_path TEXT NOT NULL,
            xray_backup_path TEXT NOT NULL,
            xray_bin TEXT NOT NULL,
            xray_reload_cmd TEXT NOT NULL,
            updated_at TEXT NOT NULL
        );`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
