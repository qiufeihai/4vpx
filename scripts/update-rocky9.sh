#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "$0")/.." && pwd)
INSTALL_DIR=/opt/4vpx
BIN_DIR=$INSTALL_DIR/bin
DATA_DIR=$INSTALL_DIR/data
GENERATED_DIR=$INSTALL_DIR/generated
SERVICE_FILE=/etc/systemd/system/4vpx.service
ENV_FILE=$INSTALL_DIR/.env.local
XRAY_SERVICE_UNIT=${XRAY_SERVICE_UNIT:-xray.service}
WITH_XRAY_RELOAD=false

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
    fail "this updater only supports Linux"
  fi
  if ! command -v systemctl >/dev/null 2>&1; then
    fail "systemd is required"
  fi
}

parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --with-xray-reload)
        WITH_XRAY_RELOAD=true
        ;;
      *)
        fail "unknown argument: $1"
        ;;
    esac
    shift
  done
}

ensure_prerequisites() {
  if ! command -v go >/dev/null 2>&1; then
    fail "go not found, please run scripts/install-rocky9.sh first"
  fi
  if [ ! -f "$ENV_FILE" ]; then
    fail "missing $ENV_FILE, please run scripts/install-rocky9.sh first"
  fi
  if ! systemctl cat 4vpx.service >/dev/null 2>&1; then
    fail "4vpx.service not found, please run scripts/install-rocky9.sh first"
  fi
  if [ "$WITH_XRAY_RELOAD" = "true" ] && ! systemctl cat "$XRAY_SERVICE_UNIT" >/dev/null 2>&1; then
    fail "${XRAY_SERVICE_UNIT} not found"
  fi
}

sync_project_tree() {
  mkdir -p "$INSTALL_DIR"
  if [ "$ROOT_DIR" = "$INSTALL_DIR" ]; then
    info "project already located at $INSTALL_DIR"
    return
  fi

  info "syncing project files to $INSTALL_DIR"
  tar \
    --exclude='./.git' \
    --exclude='./.trae' \
    --exclude='./bin' \
    --exclude='./data' \
    --exclude='./generated' \
    -C "$ROOT_DIR" -cf - . | tar -C "$INSTALL_DIR" -xf -
}

build_4vpx() {
  info "building 4vpx"
  mkdir -p "$BIN_DIR" "$DATA_DIR" "$GENERATED_DIR"
  cd "$INSTALL_DIR"
  go build -o "$BIN_DIR/4vpx" ./cmd/4vpx
  cp "$INSTALL_DIR/deploy/4vpx.service" "$SERVICE_FILE"
}

restart_services() {
  info "reloading systemd and restarting services"
  systemctl daemon-reload
  if [ "$WITH_XRAY_RELOAD" = "true" ]; then
    systemctl restart "$XRAY_SERVICE_UNIT"
  fi
  systemctl restart 4vpx
}

print_summary() {
  cat <<EOF

4vpx update finished

project path: $INSTALL_DIR
env file:     $ENV_FILE
service:      $SERVICE_FILE

next checks:
  systemctl status 4vpx --no-pager
EOF

  if [ "$WITH_XRAY_RELOAD" = "true" ]; then
    cat <<EOF
  systemctl status ${XRAY_SERVICE_UNIT} --no-pager
EOF
  fi
}

main() {
  parse_args "$@"
  require_root
  require_linux
  ensure_prerequisites
  sync_project_tree
  build_4vpx
  restart_services
  print_summary
}

main "$@"
