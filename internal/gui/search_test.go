package gui

import (
	"testing"

	"minipost/internal/model"
)

func TestRequestHitFields(t *testing.T) {
	it := model.RequestItem{
		Name: "创建用户", Method: "POST", URL: "https://api.test/Users",
		Params:  []model.KeyValue{{Key: "page", Value: "7"}},
		Headers: []model.KeyValue{{Key: "X-Trace", Value: "abc123"}},
		Body: model.RequestBody{
			Type: "json", JSON: `{"nickname":"小明"}`, Raw: "rawcontent",
			FormUrlEncoded: []model.KeyValue{{Key: "fk", Value: "fv"}},
			FormData:       []model.FormData{{Key: "up", Value: "/secret/path.png", Type: "file"}, {Key: "txt", Value: "formtext", Type: "text"}},
		},
		Auth: model.AuthConfig{Type: "bearer", Bearer: model.BearerAuth{Token: "supersecret"}},
	}
	cases := map[string]string{
		"创建": hitName, "users": hitURL, "post": hitMethod, "page": hitParam, "abc123": hitHeader,
		"nickname": hitBody, "小明": hitBody, "rawcontent": hitBody, "fv": hitBody, "formtext": hitBody,
		"/secret/path": hitNone, // 文件路径不参与搜索
		"supersecret":  hitNone, // 鉴权信息含密钥，不参与搜索
		"nothing-here": hitNone, "": hitNone,
	}
	for q, want := range cases {
		if got := requestHit(it, q); got != want {
			t.Errorf("requestHit(%q)=%q, 想要 %q", q, got, want)
		}
	}
}

// 在侧栏搜索请求体里的内容：能找到对应请求，并在行尾标出“请求体”命中；名称命中则不标。
func TestSidebarSearchFindsBodyAndShowsHint(t *testing.T) {
	a, tt, be := zhApp(t)
	r := a.coll.Requests[1] // beta
	r.Body = model.RequestBody{Type: "json", JSON: `{"needle":"xyz"}`}
	if err := be.SaveRequestItem(r); err != nil {
		t.Fatal(err)
	}
	a.reloadCollection()

	a.side.query = "needle"
	tt.Frame()
	if _, ok := tt.Find("beta"); !ok {
		t.Fatal("请求体命中的 beta 应出现在结果里")
	}
	if _, ok := tt.Find("alpha"); ok {
		t.Fatal("alpha 不应命中")
	}
	if _, ok := tt.Find("请求体"); !ok {
		t.Fatal("应在行尾标出命中位置“请求体”")
	}

	a.side.query = "bet"
	tt.Frame()
	if _, ok := tt.Find("请求体"); ok {
		t.Fatal("名称命中时不应显示命中位置提示")
	}
}
