package client

import (
	"path/filepath"
	"testing"
	"time"
)

func TestJarAbsorbAndHeaderMatching(t *testing.T) {
	jar := NewJar(filepath.Join(t.TempDir(), "cookies.json"))
	jar.Absorb("https://api.example.com/v1", map[string][]string{
		"Set-Cookie": {"sid=1; Path=/v1; Secure", "theme=dark; Domain=.example.com", "bad"},
	})
	if got := jar.Header("https://api.example.com/v1/x"); got != "sid=1; theme=dark" {
		t.Fatalf("header: %q", got)
	}
	if got := jar.Header("http://api.example.com/v1/x"); got != "theme=dark" {
		t.Fatalf("insecure should drop Secure cookie: %q", got)
	}
	if got := jar.Header("https://evilexample.com/"); got != "" {
		t.Fatalf("suffix match must respect dot boundary: %q", got)
	}
	if got := jar.Header("https://www.example.com/other"); got != "theme=dark" {
		t.Fatalf("parent domain cookie should match subdomain: %q", got)
	}
}

func TestJarExpiredSetCookieRemovesExisting(t *testing.T) {
	jar := NewJar("")
	jar.Absorb("https://a.test/", map[string][]string{"Set-Cookie": {"sid=1"}})
	jar.Absorb("https://a.test/", map[string][]string{"Set-Cookie": {"sid=; Max-Age=-10"}})
	if len(jar.All()) != 0 {
		t.Fatalf("cookie should be removed: %+v", jar.All())
	}
}

func TestJarPersistsAndDropsExpired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cookies.json")
	jar := NewJar(path)
	jar.Add(Cookie{Domain: "a.test", Name: "keep", Value: "1", Enabled: true})
	jar.Add(Cookie{Domain: "a.test", Name: "old", Value: "1", Enabled: true,
		Expires: time.Now().Add(-time.Hour).UTC().Format(time.RFC1123)})
	reloaded := NewJar(path)
	all := reloaded.All()
	if len(all) != 1 || all[0].Name != "keep" || all[0].Path != "/" {
		t.Fatalf("reloaded: %+v", all)
	}
}

func TestJarImportHeader(t *testing.T) {
	jar := NewJar("")
	n := jar.ImportHeader("Cookie: a=1; b=2; Path=/x; HttpOnly", "https://Example.com:8080/p", "")
	if n != 2 || len(jar.All()) != 2 || jar.All()[0].Domain != "example.com" {
		t.Fatalf("import: %d %+v", n, jar.All())
	}
	if jar.ImportHeader("a=1", "", "/") != 0 {
		t.Fatal("empty domain should import nothing")
	}
}

func TestSettingsStoreRoundTripAndNormalize(t *testing.T) {
	store := NewSettingsStore(t.TempDir())
	if got := store.Load(); got.Theme != "system" || !got.FollowRedirects || got.MaxResponseSizeMB != 50 {
		t.Fatalf("defaults: %+v", got)
	}
	s := DefaultSettings()
	s.FontSize, s.RequestTimeoutMs, s.AlwaysSaveUnsaved, s.AlwaysDiscardUnsaved, s.Theme = 99, -5, true, true, "neon"
	if err := store.Save(s); err != nil {
		t.Fatal(err)
	}
	got := store.Load()
	if got.FontSize != 16 || got.RequestTimeoutMs != 0 || got.AlwaysDiscardUnsaved || got.Theme != "system" {
		t.Fatalf("normalized: %+v", got)
	}
}

// 配色方案：默认雾蓝；合法值原样保存；非法值（含旧版设置文件里缺失的字段）回落为雾蓝。
func TestSettingsColorScheme(t *testing.T) {
	store := NewSettingsStore(t.TempDir())
	if got := store.Load(); got.ColorScheme != "mist" {
		t.Fatalf("默认配色应为 mist，实际 %q", got.ColorScheme)
	}
	for _, want := range []string{"mist", "ink", "jade"} {
		s := DefaultSettings()
		s.ColorScheme = want
		if err := store.Save(s); err != nil {
			t.Fatal(err)
		}
		if got := store.Load(); got.ColorScheme != want {
			t.Fatalf("配色 %q 保存后读回 %q", want, got.ColorScheme)
		}
	}
	s := DefaultSettings()
	s.ColorScheme = "neon"
	if err := store.Save(s); err != nil {
		t.Fatal(err)
	}
	if got := store.Load(); got.ColorScheme != "mist" {
		t.Fatalf("非法配色应回落为 mist，实际 %q", got.ColorScheme)
	}
}

func TestSessionStoreRoundTrip(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	if got := store.Load(); len(got.Projects) != 0 {
		t.Fatalf("empty: %+v", got)
	}
	s := Session{Projects: map[string]ProjectSession{
		"p1": {Active: 1, Tabs: []SavedTab{{RequestID: "r1"}, {RequestID: "r2"}}},
	}}
	if err := store.Save(s); err != nil {
		t.Fatal(err)
	}
	got := store.Load().Projects["p1"]
	if got.Active != 1 || len(got.Tabs) != 2 || got.Tabs[1].RequestID != "r2" {
		t.Fatalf("session: %+v", got)
	}
}
