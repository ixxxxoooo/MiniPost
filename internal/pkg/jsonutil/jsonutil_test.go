package jsonutil

import "testing"

func TestStripComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "line comment",
			input: "{\n  \"a\": 1 // 注释\n}",
			want:  "{\n  \"a\": 1 \n}",
		},
		{
			name:  "block comment",
			input: "{/* 块注释 */\"a\":1}",
			want:  "{\"a\":1}",
		},
		{
			name:  "preserves comment-like content inside string",
			input: `{"url":"http://x//y","note":"a/*b*/c"}`,
			want:  `{"url":"http://x//y","note":"a/*b*/c"}`,
		},
		{
			name:  "escaped quote inside string",
			input: `{"v":"he said \"hi\" // ok"}`,
			want:  `{"v":"he said \"hi\" // ok"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(StripComments([]byte(tt.input)))
			if got != tt.want {
				t.Fatalf("StripComments() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripComments_Empty(t *testing.T) {
	if got := StripComments(nil); len(got) != 0 {
		t.Fatalf("expected empty result, got %q", got)
	}
}
