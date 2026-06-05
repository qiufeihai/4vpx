#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR=/opt/4vpx
ENV_FILE=$INSTALL_DIR/.env.local
SERVICE_NAME=4vpx

info() {
  printf '[4vpx] %s\n' "$1"
}

fail() {
  printf '[4vpx] %s\n' "$1" >&2
  exit 1
}

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    fail "please run as root"
  fi
}

require_linux() {
  if [ "$(uname -s)" != "Linux" ]; then
    fail "this script only supports Linux"
  fi
  if ! command -v systemctl >/dev/null 2>&1; then
    fail "systemd is required"
  fi
  if ! command -v sqlite3 >/dev/null 2>&1; then
    fail "sqlite3 is required"
  fi
}

read_env_value() {
  local key=$1
  local file=$2
  if [ ! -f "$file" ]; then
    return
  fi
  grep -E "^${key}=" "$file" | head -n1 | cut -d= -f2- || true
}

upsert_env_value() {
  local key=$1
  local value=$2
  local file=$3
  local tmp

  tmp=$(mktemp)
  if [ -f "$file" ]; then
    awk -v key="$key" -v value="$value" '
      BEGIN { updated = 0 }
      index($0, key "=") == 1 {
        print key "=" value
        updated = 1
        next
      }
      { print }
      END {
        if (updated == 0) {
          print key "=" value
        }
      }
    ' "$file" >"$tmp"
  else
    printf '%s=%s\n' "$key" "$value" >"$tmp"
  fi
  mv "$tmp" "$file"
}

main() {
  local username=${1:-}
  local password=${2:-}
  local db_path backup_path

  require_root
  require_linux

  if [ -z "$username" ] || [ -z "$password" ]; then
    fail "usage: $0 <username> <password>"
  fi
  if [ ! -f "$ENV_FILE" ]; then
    fail "missing $ENV_FILE, please run scripts/install-rocky9.sh first"
  fi
  if ! systemctl cat "${SERVICE_NAME}.service" >/dev/null 2>&1; then
    fail "${SERVICE_NAME}.service not found, please run scripts/install-rocky9.sh first"
  fi

  db_path=$(read_env_value SQLITE_PATH "$ENV_FILE")
  if [ -z "$db_path" ]; then
    fail "SQLITE_PATH not found in $ENV_FILE"
  fi
  if [ ! -f "$db_path" ]; then
    fail "sqlite database not found: $db_path"
  fi

  info "writing admin credentials to $ENV_FILE"
  upsert_env_value "ADMIN_USERNAME" "$username" "$ENV_FILE"
  upsert_env_value "ADMIN_PASSWORD" "$password" "$ENV_FILE"
  chmod 600 "$ENV_FILE"

  backup_path=${db_path}.bak.$(date +%Y%m%d%H%M%S)
  info "backing up sqlite database to $backup_path"
  cp "$db_path" "$backup_path"

  info "stopping ${SERVICE_NAME}"
  systemctl stop "$SERVICE_NAME"

  info "clearing admin accounts and sessions"
  sqlite3 "$db_path" 'DELETE FROM admin_sessions; DELETE FROM admins;'

  info "starting ${SERVICE_NAME}"
  systemctl start "$SERVICE_NAME"

  cat <<EOF

4vpx admin reset finished

env file:  $ENV_FILE
db file:   $db_path
db backup: $backup_path

next checks:
  systemctl status ${SERVICE_NAME} --no-pager
  grep -E '^(ADMIN_USERNAME|ADMIN_PASSWORD)=' $ENV_FILE
EOF
}

main "$@"
