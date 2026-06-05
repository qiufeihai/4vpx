#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 2 ]; then
  echo "usage: $0 <username> <password>" >&2
  exit 1
fi

ROOT_DIR=$(cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

export ADMIN_USERNAME="$1"
export ADMIN_PASSWORD="$2"

go run ./cmd/4vpx >/tmp/4vpx-init-admin.log 2>&1 &
PID=$!
sleep 2
kill "$PID" >/dev/null 2>&1 || true
wait "$PID" >/dev/null 2>&1 || true

echo "admin bootstrap attempted"
