package logger

import "testing"

func TestLevelFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback Level
		want     Level
	}{
		{name: "empty uses fallback", value: "", fallback: INFO, want: INFO},
		{name: "debug", value: "debug", fallback: INFO, want: DEBUG},
		{name: "warning alias", value: "warning", fallback: DEBUG, want: WARN},
		{name: "invalid uses fallback", value: "verbose", fallback: ERROR, want: ERROR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := levelFromEnv(tt.value, tt.fallback); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRedactURL(t *testing.T) {
	got := RedactURL("https://example.com/api?access_token=abc&name=alice&password=secret")
	want := "https://example.com/api?access_token=%5BREDACTED%5D&name=alice&password=%5BREDACTED%5D"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRedactRequestTargetCurl(t *testing.T) {
	got := RedactRequestTarget("curl https://example.com -H 'Cookie: sid=abc'")
	if got != "<curl command: 45 chars>" {
		t.Fatalf("expected cURL command to be summarized, got %q", got)
	}
}
