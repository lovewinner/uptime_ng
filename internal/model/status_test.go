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

func TestAcceptedStatusCodesFallback(t *testing.T) {
	got := acceptedStatusCodes("")
	if len(got) != 1 || got[0] != "200-299" {
		t.Fatalf("default=%v", got)
	}

	got = acceptedStatusCodes(`["201","418"]`)
	if len(got) != 2 || got[0] != "201" || got[1] != "418" {
		t.Fatalf("codes=%v", got)
	}
}

func TestStatusCodeRuleMatches(t *testing.T) {
	tests := []struct {
		name string
		code int
		rule string
		want bool
	}{
		{name: "trimmed single", code: 201, rule: " 201 ", want: true},
		{name: "trimmed range", code: 204, rule: " 200 - 299 ", want: true},
		{name: "outside range", code: 500, rule: "200-299", want: false},
		{name: "invalid single", code: 200, rule: "ok", want: false},
		{name: "invalid range", code: 200, rule: "200-ok", want: false},
		{name: "reversed range", code: 250, rule: "299-200", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusCodeRuleMatches(tt.code, tt.rule); got != tt.want {
				t.Fatalf("statusCodeRuleMatches(%d, %q)=%v want %v", tt.code, tt.rule, got, tt.want)
			}
		})
	}
}
