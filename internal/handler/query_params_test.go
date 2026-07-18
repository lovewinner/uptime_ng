package handler

import "testing"

func TestPositiveIntParam(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback int
		want     int
	}{
		{name: "valid", value: "25", fallback: 50, want: 25},
		{name: "invalid", value: "bad", fallback: 50, want: 50},
		{name: "zero", value: "0", fallback: 50, want: 50},
		{name: "negative", value: "-1", fallback: 50, want: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := positiveIntParam(tt.value, tt.fallback); got != tt.want {
				t.Fatalf("positiveIntParam(%q, %d)=%d want %d", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}
