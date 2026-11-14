package client

import (
	"reflect"
	"testing"
)

func TestFindTemplateRefs(t *testing.T) {
	vars := []Var{{Key: "baseUrl", Value: "x"}, {Key: " id ", Value: "1"}}
	in := "{{baseUrl}}/users/{{ id }}?t={{token}}&e={{ }}"
	got := FindTemplateRefs(in, vars)
	if len(got) != 3 {
		t.Fatalf("应识别 3 处引用（空白变量名忽略）: %+v", got)
	}
	if got[0].Name != "baseUrl" || !got[0].Defined || in[got[0].Start:got[0].End] != "{{baseUrl}}" {
		t.Fatalf("第一处不对: %+v", got[0])
	}
	if got[1].Name != "id" || !got[1].Defined {
		t.Fatalf("带空白的变量名应被修剪后匹配: %+v", got[1])
	}
	if got[2].Name != "token" || got[2].Defined {
		t.Fatalf("token 应为未定义: %+v", got[2])
	}
	if FindTemplateRefs("没有变量", vars) != nil {
		t.Fatal("无变量时应返回 nil")
	}
}

func TestFindTemplateRefsByteOffsetsWithChinese(t *testing.T) {
	in := "中文{{a}}"
	refs := FindTemplateRefs(in, nil)
	if len(refs) != 1 || in[refs[0].Start:refs[0].End] != "{{a}}" {
		t.Fatalf("偏移应为字节偏移: %+v", refs)
	}
}

func TestUndefinedVarNames(t *testing.T) {
	vars := []Var{{Key: "a"}}
	got := UndefinedVarNames(vars, "{{a}}{{b}}", "{{c}}{{b}}", "")
	if !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Fatalf("应去重并保持顺序: %v", got)
	}
}

func TestDraftTemplateTexts(t *testing.T) {
	d := &Draft{URL: "{{u}}"}
	d.Params = []*Row{{Enabled: true, Key: "k", Value: "{{p}}"}, {Enabled: false, Key: "off", Value: "{{ignored}}"}}
	d.Body = BodyDraft{Type: BodyJSON, JSON: `{"a":"{{j}}"}`, Raw: "{{rawIgnored}}"}
	d.Auth = AuthDraft{Type: AuthBearer, Token: "{{tok}}", Username: "{{userIgnored}}"}
	got := UndefinedVarNames(nil, d.TemplateTexts()...)
	want := []string{"u", "p", "j", "tok"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("只应统计启用项与当前类型: got %v want %v", got, want)
	}
}
