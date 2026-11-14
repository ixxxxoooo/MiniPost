package httputil

import (
	"math/rand/v2"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// 内置动态变量：写成 {{$timestamp}} 这样的形式，每次发送请求时现场求值，无需在环境里定义。
// 它们是“预请求步骤”的最小实现：常见的时间戳、随机数、UUID 这类发送前才能确定的值。
var dynamicVarPattern = regexp.MustCompile(`\{\{\s*\$([A-Za-z]+)\s*\}\}`)

// 为便于测试，时间和随机数来源可被替换。
var (
	nowFunc     = time.Now
	randIntFunc = func() int { return rand.IntN(1000) }
	uuidFunc    = func() string { return uuid.NewString() }
)

// DynamicVarNames 是支持的动态变量名（不含 $ 前缀），用于界面提示。
var DynamicVarNames = []string{"timestamp", "timestampMs", "isoTimestamp", "randomUUID", "randomInt"}

// IsDynamicVar 判断变量名（含 $ 前缀，如 "$timestamp"）是否是内置动态变量。
func IsDynamicVar(name string) bool {
	if len(name) < 2 || name[0] != '$' {
		return false
	}
	for _, n := range DynamicVarNames {
		if name[1:] == n {
			return true
		}
	}
	return false
}

// dynamicValue 求出动态变量 name（不含 $）的当前值。
func dynamicValue(name string) (string, bool) {
	switch name {
	case "timestamp":
		return strconv.FormatInt(nowFunc().Unix(), 10), true
	case "timestampMs":
		return strconv.FormatInt(nowFunc().UnixMilli(), 10), true
	case "isoTimestamp":
		return nowFunc().UTC().Format(time.RFC3339), true
	case "randomUUID":
		return uuidFunc(), true
	case "randomInt":
		return strconv.Itoa(randIntFunc()), true
	}
	return "", false
}

// resolveDynamicVars 把 input 里的 {{$name}} 替换成当前值；不认识的 $ 变量原样保留。
// 同一个变量出现多次时每次单独求值（例如两个 randomUUID 互不相同）。
func resolveDynamicVars(input string) string {
	return dynamicVarPattern.ReplaceAllStringFunc(input, func(token string) string {
		name := dynamicVarPattern.FindStringSubmatch(token)[1]
		if v, ok := dynamicValue(name); ok {
			return v
		}
		return token
	})
}
