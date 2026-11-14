#!/usr/bin/env bash
# 开发模式:构建并启动 "MiniPost Dev"(仅 macOS)
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# 让 GOROOT 与当前 go 命令一致(清掉环境里残留的旧 GOROOT)
. "$ROOT_DIR/scripts/goenv.sh"

MYGO="$(command -v mygo || true)"
if [[ -z "$MYGO" && -x "$(go env GOPATH)/bin/mygo" ]]; then
  MYGO="$(go env GOPATH)/bin/mygo"
fi
if [[ -z "$MYGO" ]]; then
  echo "缺少 mygo,请先安装: go install github.com/egoist/mygo/cmd/mygo@latest" >&2
  exit 1
fi

exec "$MYGO" dev "$@"
