package client

import "testing"

// 默认自动下载更新（NoAutoUpdate 为 false），关闭后能持久化并重新读出。
func TestSettingsNoAutoUpdateRoundTrip(t *testing.T) {
	st := NewSettingsStore(t.TempDir())
	if st.Load().NoAutoUpdate {
		t.Fatal("默认应自动下载更新")
	}
	s := st.Load()
	s.NoAutoUpdate = true
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	if !st.Load().NoAutoUpdate {
		t.Fatal("关闭自动下载应被保存")
	}
}
