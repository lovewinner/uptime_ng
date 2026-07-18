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

func TestUintParam(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		want   uint
		wantOK bool
	}{
		{name: "valid", value: "25", want: 25, wantOK: true},
		{name: "invalid", value: "bad", want: 0, wantOK: false},
		{name: "zero", value: "0", want: 0, wantOK: false},
		{name: "negative", value: "-1", want: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := uintParam(tt.value)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("uintParam(%q)=(%d,%v) want (%d,%v)", tt.value, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
