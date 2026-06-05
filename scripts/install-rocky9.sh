#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "$0")/.." && pwd)
INSTALL_DIR=/opt/4vpx
BIN_DIR=$INSTALL_DIR/bin
DATA_DIR=$INSTALL_DIR/data
GENERATED_DIR=$INSTALL_DIR/generated
SERVICE_FILE=/etc/systemd/system/4vpx.service
ENV_FILE=$INSTALL_DIR/.env.local
GO_MIN_VERSION=1.23.0
GO_VERSION_URL=https://go.dev/VERSION?m=text
XRAY_INSTALL_URL=https://github.com/XTLS/Xray-install/raw/main/install-release.sh
XRAY_BIN_PATH=
XRAY_SERVICE_UNIT=${XRAY_SERVICE_UNIT:-xray.service}

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
    fail "this installer only supports Linux"
  fi
  if ! command -v systemctl >/dev/null 2>&1; then
    fail "systemd is required"
  fi
}

version_ge() {
  [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

detect_go_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      printf 'amd64\n'
      ;;
    aarch64|arm64)
      printf 'arm64\n'
      ;;
    *)
      fail "unsupported CPU architecture: $(uname -m)"
      ;;
  esac
}

current_go_version() {
  if command -v go >/dev/null 2>&1; then
    go version | awk '{print $3}' | sed 's/^go//'
  fi
}

install_base_packages() {
  info "installing system packages"
  dnf install -y git tar gzip curl unzip ca-certificates firewalld policycoreutils-python-utils python3
}

install_or_upgrade_go() {
  local current
  current=$(current_go_version || true)
  if [ -n "$current" ] && version_ge "$current" "$GO_MIN_VERSION"; then
    info "using existing Go $current"
    export PATH=/usr/local/go/bin:/usr/local/bin:$PATH
    return
  fi

  local arch version tarball download_url tmp_tar
  arch=$(detect_go_arch)
  version=$(curl -fsSL "$GO_VERSION_URL" | head -n1 | tr -d '\r')
  if [ -z "$version" ]; then
    fail "failed to detect latest Go version"
  fi
  tarball=${version}.linux-${arch}.tar.gz
  download_url=https://go.dev/dl/${tarball}
  tmp_tar=/tmp/${tarball}

  info "installing ${version} for ${arch}"
  curl -fsSL "$download_url" -o "$tmp_tar"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "$tmp_tar"
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
  rm -f "$tmp_tar"

  export PATH=/usr/local/go/bin:/usr/local/bin:$PATH
  current=$(current_go_version || true)
  if [ -z "$current" ] || ! version_ge "$current" "$GO_MIN_VERSION"; then
    fail "Go installation failed"
  fi
  info "installed Go $current"
}

install_or_upgrade_xray() {
  if command -v xray >/dev/null 2>&1 && systemctl cat "$XRAY_SERVICE_UNIT" >/dev/null 2>&1; then
    XRAY_BIN_PATH=$(command -v xray)
    info "using existing Xray installation at $XRAY_BIN_PATH ($XRAY_SERVICE_UNIT)"
    return
  fi

  info "installing Xray via the official installer"
  bash -c "$(curl -fsSL "$XRAY_INSTALL_URL")" @ install

  if ! command -v xray >/dev/null 2>&1; then
    fail "Xray installation failed: xray binary not found"
  fi
  if ! systemctl cat "$XRAY_SERVICE_UNIT" >/dev/null 2>&1; then
    fail "Xray installation failed: ${XRAY_SERVICE_UNIT} not found"
  fi
  XRAY_BIN_PATH=$(command -v xray)
}

sync_project_tree() {
  mkdir -p "$INSTALL_DIR"
  if [ "$ROOT_DIR" = "$INSTALL_DIR" ]; then
    info "project already located at $INSTALL_DIR"
    return
  fi

  info "syncing project files to $INSTALL_DIR"
  mkdir -p "$INSTALL_DIR"
  tar \
    --exclude='./.git' \
    --exclude='./.trae' \
    --exclude='./bin' \
    --exclude='./data' \
    --exclude='./generated' \
    -C "$ROOT_DIR" -cf - . | tar -C "$INSTALL_DIR" -xf -
}

read_env_value() {
  local key=$1
  local file=$2
  if [ ! -f "$file" ]; then
    return
  fi
  grep -E "^${key}=" "$file" | head -n1 | cut -d= -f2- || true
}

prompt_value() {
  local label=$1
  local default_value=$2
  local secret=${3:-false}
  local input

  if [ "$secret" = "true" ]; then
    if [ -n "$default_value" ]; then
      read -r -s -p "$label [press Enter to keep current/default]: " input
    else
      read -r -s -p "$label: " input
    fi
    printf '\n'
  else
    if [ -n "$default_value" ]; then
      read -r -p "$label [$default_value]: " input
    else
      read -r -p "$label: " input
    fi
  fi

  if [ -n "$input" ]; then
    printf '%s\n' "$input"
    return
  fi
  printf '%s\n' "$default_value"
}

generate_reality_keypair() {
  local output private_key public_key
  output=$("$XRAY_BIN_PATH" x25519 2>&1 | tr -d '\r')
  private_key=$(printf '%s\n' "$output" | sed -nE 's/^[[:space:]]*([Pp]rivate[[:space:]]*[Kk]ey|PrivateKey):[[:space:]]*//p' | head -n1)
  public_key=$(printf '%s\n' "$output" | sed -nE 's/^[[:space:]]*([Pp]ublic[[:space:]]*[Kk]ey|PublicKey|Password[[:space:]]*\(PublicKey\)):[[:space:]]*//p' | head -n1)
  if [ -z "$private_key" ] || [ -z "$public_key" ]; then
    return 1
  fi
  printf '%s\n%s\n' "$private_key" "$public_key"
}

derive_reality_public_key() {
  local private_key=$1 output public_key
  [ -n "$private_key" ] || return 1

  output=$("$XRAY_BIN_PATH" x25519 -i "$private_key" 2>&1 | tr -d '\r')
  public_key=$(printf '%s\n' "$output" | sed -nE 's/^[[:space:]]*([Pp]ublic[[:space:]]*[Kk]ey|PublicKey|Password[[:space:]]*\(PublicKey\)):[[:space:]]*//p' | head -n1)
  [ -n "$public_key" ] || return 1
  printf '%s\n' "$public_key"
}

read_existing_reality_config() {
  local config_path=$1
  [ -f "$config_path" ] || return 1
  command -v python3 >/dev/null 2>&1 || return 1

  python3 - "$config_path" <<'PY'
import json
import sys

path = sys.argv[1]

try:
    with open(path, "r", encoding="utf-8") as f:
        data = json.load(f)
except Exception:
    raise SystemExit(1)

for inbound in data.get("inbounds", []):
    stream_settings = inbound.get("streamSettings") or {}
    reality_settings = stream_settings.get("realitySettings") or {}
    if not reality_settings:
        continue

    server_names = reality_settings.get("serverNames") or []
    short_ids = reality_settings.get("shortIds") or []

    print(reality_settings.get("dest", ""))
    print(server_names[0] if server_names else "")
    print(reality_settings.get("privateKey", ""))
    print(short_ids[0] if short_ids else "")
    raise SystemExit(0)

raise SystemExit(1)
PY
}

generate_short_id() {
  head -c 8 /dev/urandom | od -An -tx1 | tr -d ' \n'
}

write_env_file() {
  local existing_file=$ENV_FILE
  local app_addr_default app_base_url_default admin_user_default admin_password_default
  local sqlite_default server_address_default reality_dest_default reality_server_name_default
  local private_key_default public_key_default short_id_default xray_config_default
  local xray_backup_default xray_reload_default session_secure_default
  local app_addr app_base_url admin_username admin_password sqlite_path server_address
  local reality_dest reality_server_name private_key public_key short_id
  local xray_config_path xray_backup_path xray_reload_cmd backup_file generated_keys
  local existing_reality_config existing_reality_dest existing_reality_server_name
  local existing_reality_private_key existing_reality_short_id

  app_addr_default=$(read_env_value APP_ADDR "$existing_file")
  [ -n "$app_addr_default" ] || app_addr_default=0.0.0.0:8443

  admin_user_default=$(read_env_value ADMIN_USERNAME "$existing_file")
  [ -n "$admin_user_default" ] || admin_user_default=admin

  admin_password_default=$(read_env_value ADMIN_PASSWORD "$existing_file")

  sqlite_default=$(read_env_value SQLITE_PATH "$existing_file")
  [ -n "$sqlite_default" ] || sqlite_default=/opt/4vpx/data/4vpx.db

  server_address_default=$(read_env_value SERVER_ADDRESS "$existing_file")
  [ -n "$server_address_default" ] || server_address_default=$(curl -4 -fsSL https://api.ipify.org 2>/dev/null || true)

  xray_config_default=$(read_env_value XRAY_CONFIG_PATH "$existing_file")
  [ -n "$xray_config_default" ] || xray_config_default=/usr/local/etc/xray/config.json

  existing_reality_config=$(read_existing_reality_config "$xray_config_default" || true)
  if [ -n "$existing_reality_config" ]; then
    existing_reality_dest=$(printf '%s\n' "$existing_reality_config" | sed -n '1p')
    existing_reality_server_name=$(printf '%s\n' "$existing_reality_config" | sed -n '2p')
    existing_reality_private_key=$(printf '%s\n' "$existing_reality_config" | sed -n '3p')
    existing_reality_short_id=$(printf '%s\n' "$existing_reality_config" | sed -n '4p')
  fi

  reality_dest_default=$(read_env_value REALITY_DEST "$existing_file")
  [ -n "$reality_dest_default" ] || reality_dest_default=$existing_reality_dest
  [ -n "$reality_dest_default" ] || reality_dest_default=www.microsoft.com:443

  reality_server_name_default=$(read_env_value REALITY_SERVER_NAME "$existing_file")
  [ -n "$reality_server_name_default" ] || reality_server_name_default=$existing_reality_server_name
  [ -n "$reality_server_name_default" ] || reality_server_name_default=www.microsoft.com

  private_key_default=$(read_env_value REALITY_PRIVATE_KEY "$existing_file")
  [ -n "$private_key_default" ] || private_key_default=$existing_reality_private_key
  public_key_default=$(read_env_value REALITY_PUBLIC_KEY "$existing_file")
  if [ -z "$public_key_default" ] && [ -n "$private_key_default" ]; then
    public_key_default=$(derive_reality_public_key "$private_key_default" || true)
  fi
  if [ -z "$private_key_default" ] || [ -z "$public_key_default" ]; then
    if generated_keys=$(generate_reality_keypair); then
      private_key_default=$(printf '%s\n' "$generated_keys" | sed -n '1p')
      public_key_default=$(printf '%s\n' "$generated_keys" | sed -n '2p')
    else
      info "automatic REALITY key generation failed, please enter existing keys manually"
    fi
  fi

  short_id_default=$(read_env_value REALITY_SHORT_ID "$existing_file")
  [ -n "$short_id_default" ] || short_id_default=$existing_reality_short_id
  [ -n "$short_id_default" ] || short_id_default=$(generate_short_id)

  xray_backup_default=$(read_env_value XRAY_BACKUP_PATH "$existing_file")
  [ -n "$xray_backup_default" ] || xray_backup_default=/usr/local/etc/xray/config.json.bak

  xray_reload_default=$(read_env_value XRAY_RELOAD_CMD "$existing_file")
  [ -n "$xray_reload_default" ] || xray_reload_default="systemctl restart ${XRAY_SERVICE_UNIT}"

  session_secure_default=$(read_env_value SESSION_SECURE "$existing_file")
  [ -n "$session_secure_default" ] || session_secure_default=false

  info "collecting deployment parameters"
  app_addr=$(prompt_value "4vpx listen address" "$app_addr_default")
  server_address=$(prompt_value "Public server IP or domain" "$server_address_default")

  app_base_url_default=$(read_env_value APP_BASE_URL "$existing_file")
  if [ -z "$app_base_url_default" ]; then
    case "$app_addr" in
      127.0.0.1:*)
        app_base_url_default=http://127.0.0.1:${app_addr#127.0.0.1:}
        ;;
      0.0.0.0:*)
        app_base_url_default=http://${server_address}:${app_addr#0.0.0.0:}
        ;;
      :*)
        app_base_url_default=http://${server_address}:${app_addr#:}
        ;;
      *)
        app_base_url_default=http://${server_address}:8443
        ;;
    esac
  fi

  app_base_url=$(prompt_value "4vpx base URL" "$app_base_url_default")
  admin_username=$(prompt_value "Admin username" "$admin_user_default")
  admin_password=$(prompt_value "Admin password" "$admin_password_default" true)
  sqlite_path=$(prompt_value "SQLite path" "$sqlite_default")
  reality_dest=$(prompt_value "REALITY dest" "$reality_dest_default")
  reality_server_name=$(prompt_value "REALITY server name" "$reality_server_name_default")
  private_key=$(prompt_value "REALITY private key" "$private_key_default" true)
  public_key=$(prompt_value "REALITY public key" "$public_key_default" true)
  short_id=$(prompt_value "REALITY short id" "$short_id_default")
  xray_config_path=$(prompt_value "Xray config path" "$xray_config_default")
  xray_backup_path=$(prompt_value "Xray backup path" "$xray_backup_default")
  xray_reload_cmd=$(prompt_value "Xray reload command" "$xray_reload_default")

  if [ -z "$admin_password" ]; then
    fail "admin password must not be empty"
  fi
  if [ -z "$server_address" ]; then
    fail "server address must not be empty"
  fi
  if [ -z "$private_key" ] || [ -z "$public_key" ] || [ -z "$short_id" ]; then
    fail "REALITY fields must not be empty"
  fi

  if [ -f "$ENV_FILE" ]; then
    backup_file=${ENV_FILE}.bak.$(date +%Y%m%d%H%M%S)
    cp "$ENV_FILE" "$backup_file"
    info "existing env backed up to $backup_file"
  fi

  cat >"$ENV_FILE" <<EOF
APP_ADDR=${app_addr}
APP_BASE_URL=${app_base_url}
SESSION_COOKIE_NAME=admin_session
SESSION_SECURE=${session_secure_default}

ADMIN_USERNAME=${admin_username}
ADMIN_PASSWORD=${admin_password}

SQLITE_PATH=${sqlite_path}

SERVER_ADDRESS=${server_address}
SERVER_PORT=443
REALITY_DEST=${reality_dest}
REALITY_SERVER_NAME=${reality_server_name}
CLIENT_FINGERPRINT=chrome
REALITY_PRIVATE_KEY=${private_key}
REALITY_PUBLIC_KEY=${public_key}
REALITY_SHORT_ID=${short_id}

XRAY_LOGLEVEL=warning
XRAY_CONFIG_PATH=${xray_config_path}
XRAY_BACKUP_PATH=${xray_backup_path}
XRAY_BIN=${XRAY_BIN_PATH}
XRAY_RELOAD_CMD=${xray_reload_cmd}
EOF
  chmod 600 "$ENV_FILE"
}

build_and_install_4vpx() {
  info "building 4vpx"
  mkdir -p "$BIN_DIR" "$DATA_DIR" "$GENERATED_DIR"
  cd "$INSTALL_DIR"
  go build -o "$BIN_DIR/4vpx" ./cmd/4vpx
  cp "$INSTALL_DIR/deploy/4vpx.service" "$SERVICE_FILE"
}

enable_firewall() {
  info "configuring firewalld"
  systemctl enable --now firewalld
  firewall-cmd --permanent --add-port=443/tcp >/dev/null
  firewall-cmd --permanent --add-port=8443/tcp >/dev/null
  firewall-cmd --reload >/dev/null
}

enable_services() {
  info "enabling services"
  systemctl daemon-reload
  systemctl enable "$XRAY_SERVICE_UNIT" >/dev/null
  systemctl enable 4vpx >/dev/null
  systemctl restart "$XRAY_SERVICE_UNIT"
  systemctl restart 4vpx
}

print_summary() {
  local app_addr
  app_addr=$(read_env_value APP_ADDR "$ENV_FILE")
  cat <<EOF

4vpx one-click deploy finished

project path: $INSTALL_DIR
env file:     $ENV_FILE
service:      $SERVICE_FILE
panel addr:   $app_addr

next checks:
  systemctl status ${XRAY_SERVICE_UNIT} --no-pager
  systemctl status 4vpx --no-pager
  ss -lntp | grep ':443'
  ss -lntp | grep ':8443'
EOF
}

main() {
  require_root
  require_linux
  install_base_packages
  install_or_upgrade_go
  install_or_upgrade_xray
  sync_project_tree
  write_env_file
  build_and_install_4vpx
  enable_firewall
  enable_services
  print_summary
}

main "$@"
