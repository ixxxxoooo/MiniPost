package model

import "testing"

// 当前值优先，当前值为空时使用初始值。
func TestVariableEffective(t *testing.T) {
	cases := []struct {
		v    Variable
		want string
	}{
		{Variable{Value: "cur", InitialValue: "init"}, "cur"},
		{Variable{Value: "", InitialValue: "init"}, "init"},
		{Variable{Value: "cur"}, "cur"},
		{Variable{}, ""},
	}
	for _, c := range cases {
		if got := c.v.Effective(); got != c.want {
			t.Fatalf("Effective(%+v) = %q，期望 %q", c.v, got, c.want)
		}
	}
}
