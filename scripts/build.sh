#!/usr/bin/env bash
# 构建 MiniPost(仅 macOS,最低系统版本见 mygo.json 的 macos.minimumSystemVersion)
# 用法:
#   ./scripts/build.sh                      # 构建当前架构的 .app 与 .dmg
#   ./scripts/build.sh -skip-dmg            # 只构建 .app
#   ./scripts/build.sh -platform darwin/universal
# 产物输出到 build/ 目录。
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "MiniPost 仅支持在 macOS 上构建。" >&2
  exit 1
fi

command -v go >/dev/null 2>&1 || { echo "缺少命令: go(需要 Go 1.27 或更新版本)" >&2; exit 1; }

# 优先使用 PATH 中的 mygo,其次使用 go install 的默认位置
MYGO="$(command -v mygo || true)"
if [[ -z "$MYGO" && -x "$(go env GOPATH)/bin/mygo" ]]; then
  MYGO="$(go env GOPATH)/bin/mygo"
fi
if [[ -z "$MYGO" ]]; then
  echo "缺少 mygo,请先安装: go install github.com/egoist/mygo/cmd/mygo@latest" >&2
  exit 1
fi

# 保证 GOROOT 与当前 go 命令一致,避免多版本 Go 并存时编译器与标准库不匹配
GOROOT="$(go env GOROOT)"
export GOROOT

exec "$MYGO" build "$@"
