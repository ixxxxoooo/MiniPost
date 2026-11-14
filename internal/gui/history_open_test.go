package gui

import (
	"strings"
	"testing"

	"minipost/internal/model"
)

// 新记录带 RequestID：直接打开对应请求。
func TestHistoryOpenByRequestID(t *testing.T) {
	a, _, _ := zhApp(t)
	target := a.coll.Requests[1]
	a.openFromHistory(model.HistoryEntry{ID: "h1", RequestID: target.ID, Method: "GET", URL: "https://whatever"})
	if len(a.tabs) != 1 || a.tabs[0].draft.ID != target.ID {
		t.Fatalf("应打开 %s，实际 %d 个标签", target.ID, len(a.tabs))
	}
	if len(a.toastMsgs) != 0 {
		t.Fatalf("不应提示，实际 %v", a.toastMsgs)
	}
}

// 旧记录没有 RequestID：按方法 + 地址匹配集合里的请求，不再误报“已删除”。
func TestHistoryOpenLegacyEntryByMethodAndURL(t *testing.T) {
	a, _, _ := zhApp(t)
	target := a.coll.Requests[1]
	a.openFromHistory(model.HistoryEntry{ID: "h1", Method: "get", URL: target.URL})
	if len(a.tabs) != 1 || a.tabs[0].draft.ID != target.ID {
		t.Fatalf("旧记录应按方法+地址打开 %s，实际 %d 个标签", target.ID, len(a.tabs))
	}
	if len(a.toastMsgs) != 0 {
		t.Fatalf("不应提示，实际 %v", a.toastMsgs)
	}
}

// 请求确实被删除（ID 失效或地址对不上）时仍提示“已被删除”。
func TestHistoryOpenMissingRequestToasts(t *testing.T) {
	cases := map[string]model.HistoryEntry{
		"ID 已失效":   {ID: "h1", RequestID: "gone", Method: "GET", URL: "https://example.com/alpha"},
		"旧记录无匹配":   {ID: "h2", Method: "GET", URL: "https://nowhere.test/x"},
		"旧记录地址为空":  {ID: "h3", Method: "GET"},
		"旧记录方法不一致": {ID: "h4", Method: "DELETE", URL: "https://example.com/alpha"},
	}
	for name, e := range cases {
		t.Run(name, func(t *testing.T) {
			a, _, _ := zhApp(t)
			a.openFromHistory(e)
			if len(a.tabs) != 0 {
				t.Fatalf("不应打开标签，实际 %d 个", len(a.tabs))
			}
			if len(a.toastMsgs) != 1 || !strings.Contains(a.toastMsgs[0].msg, "已被删除") {
				t.Fatalf("应提示已被删除，实际 %v", a.toastMsgs)
			}
		})
	}
}

// 发送时要把集合里的请求 ID 带给后端，历史才能回溯到原请求。
func TestSendCarriesSourceID(t *testing.T) {
	a, tt, be := zhApp(t)
	target := a.coll.Requests[0]
	a.openRequest(target.ID)
	tt.Frame()
	if err := tt.Click("发送"); err != nil {
		t.Fatal(err)
	}
	if len(be.sent) != 1 || be.sent[0].SourceID != target.ID {
		t.Fatalf("发送应携带 SourceID=%s，实际 %+v", target.ID, be.sent)
	}
	if be.sent[0].RequestID == "" || be.sent[0].RequestID == target.ID {
		t.Fatalf("RequestID 仍应是每次发送的随机 ID，实际 %q", be.sent[0].RequestID)
	}
}
