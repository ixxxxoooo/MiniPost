# 被其他脚本 source(POSIX sh 兼容)：让 GOROOT 与当前 go 命令保持一致。
# 环境里残留的 GOROOT(例如 g、gvm 等版本管理器写死的旧版本)会让编译器与标准库版本不匹配，
# 表现为 "compile: version go1.x does not match go tool version"。
# 注意要先 unset 再取值：GOROOT 已设置时 `go env GOROOT` 返回的就是那个错误的值。
command -v go >/dev/null 2>&1 || { echo "缺少命令: go(需要 Go 1.27 或更新版本)" >&2; exit 1; }
unset GOROOT
GOROOT="$(go env GOROOT)"
export GOROOT
