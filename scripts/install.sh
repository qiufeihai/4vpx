#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

mkdir -p ./bin
go build -o ./bin/4vpx ./cmd/4vpx

echo "build complete: ./bin/4vpx"
