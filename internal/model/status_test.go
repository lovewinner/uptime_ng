package model

import "testing"

func TestCheckStatusCodeUsesJSONParser(t *testing.T) {
	tests := []struct {
		name string
		code int
		raw  string
		want bool
	}{
		{name: "range", code: 204, raw: `["200-299"]`, want: true},
		{name: "single", code: 418, raw: `["200","418"]`, want: true},
		{name: "invalid json falls back", code: 204, raw: `not-json`, want: true},
		{name: "outside", code: 500, raw: `["200-299"]`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckStatusCode(tt.code, tt.raw); got != tt.want {
				t.Fatalf("CheckStatusCode(%d, %q)=%v want %v", tt.code, tt.raw, got, tt.want)
			}
		})
	}
}
