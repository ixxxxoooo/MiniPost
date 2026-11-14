#!/bin/sh
# 提交与发版前的本地检查（项目不使用 CI）：格式 → go vet → staticcheck → 竞态测试。
# 用法：
#   scripts/check.sh           全部检查
#   scripts/check.sh -short    跳过 -race，只跑普通测试（更快）
set -eu
cd "$(dirname "$0")/.."
. scripts/goenv.sh

echo "==> gofmt"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "以下文件需要 gofmt：" >&2
  echo "$unformatted" >&2
  exit 1
fi

echo "==> go vet"
go vet ./...

echo "==> staticcheck"
go tool staticcheck ./...

if [ "${1:-}" = "-short" ]; then
  echo "==> go test"
  go test ./...
else
  echo "==> go test -race"
  go test -race ./...
fi

echo "全部检查通过"
