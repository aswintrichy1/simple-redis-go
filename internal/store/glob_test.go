package store

import "testing"

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern, str string
		want         bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"h?llo", "hello", true},
		{"h?llo", "hallo", true},
		{"h?llo", "hllo", false},
		{"h[ae]llo", "hello", true},
		{"h[ae]llo", "hallo", true},
		{"h[ae]llo", "hillo", false},
		{"h[^e]llo", "hallo", true},
		{"h[^e]llo", "hello", false},
		{"h[a-c]llo", "hbllo", true},
		{"h[a-c]llo", "hdllo", false},
		{"hello", "hello", true},
		{"hello", "world", false},
		{"he*o", "hello", true},
		{"he*o", "heo", true},
		{"*llo", "hello", true},
		{"user:*", "user:1", true},
		{"user:*", "other", false},
		{`hel\*`, "hel*", true},
		{`hel\*`, "hello", false},
		{"", "", true},
		{"", "x", false},
	}
	for _, tt := range tests {
		if got := matchPattern(tt.pattern, tt.str); got != tt.want {
			t.Errorf("matchPattern(%q,%q) = %v, want %v", tt.pattern, tt.str, got, tt.want)
		}
	}
}
