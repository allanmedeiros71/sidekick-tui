package calc

import (
	"strings"
	"testing"
)

func TestDateArithmetic(t *testing.T) {
	tests := []struct{ in, utc string }{
		{"now + 3d", "2026-10-18T12:00:00Z"},
		{"now+1y2d", "2027-10-17T12:00:00Z"},
		{"now - 1h30m", "2026-10-15T10:30:00Z"},
		{"2026-01-01 - 2w", "2025-12-18T00:00:00Z"},
		{"2024-02-29 + 1y", "2025-03-01T00:00:00Z"},
		{"@1700000000 + 1h", "2023-11-14T23:13:20Z"},
		{"1d + now", "2026-10-16T12:00:00Z"},
		{"2026-01-01 13:30 + 45s", "2026-01-01T13:30:45Z"},
		{"now + 1d - 2h", "2026-10-16T10:00:00Z"},
	}
	for _, tt := range tests {
		r := mustEval(t, tt.in)
		if got := get(r, "UTC"); got != tt.utc {
			t.Errorf("Eval(%q) UTC = %s, want %s", tt.in, got, tt.utc)
		}
	}
}

func TestDateDifference(t *testing.T) {
	tests := []struct{ in, duration, days string }{
		{"2026-12-25 - now", "70 days 12 hours", "70.5"},
		{"now - 2026-12-25", "-70 days 12 hours", "-70.5"},
		{"2027-10-15 - 2025-10-14", "2 years 1 day", "731"},
		{"now - now", "0 seconds", "0"},
	}
	for _, tt := range tests {
		r := mustEval(t, tt.in)
		if r.Kind != "duration" {
			t.Fatalf("Eval(%q).Kind = %s", tt.in, r.Kind)
		}
		if get(r, "duration") != tt.duration || get(r, "days") != tt.days {
			t.Errorf("Eval(%q) = %q / %q days, want %q / %q", tt.in, get(r, "duration"), get(r, "days"), tt.duration, tt.days)
		}
	}
	if got := get(mustEval(t, "2026-10-16 - now"), "seconds"); got != "43200" {
		t.Errorf("seconds = %s", got)
	}
}

func TestDateArithmeticErrors(t *testing.T) {
	tests := []struct{ in, msg string }{
		{"now + now", "cannot add two dates"},
		{"now + 3x", "expected a date or a duration"},
		{"3d - now", "cannot subtract a date"},
		{"now - now + 1d", "cannot be combined"},
	}
	for _, tt := range tests {
		_, err := Eval(tt.in, refNow)
		if err == nil || !strings.Contains(err.Error(), tt.msg) {
			t.Errorf("Eval(%q) error = %v, want it to mention %q", tt.in, err, tt.msg)
		}
	}
}

func TestDateArithmeticLeavesOtherInputsAlone(t *testing.T) {
	tests := []struct{ in, kind string }{
		{"1m + 2m", "expr"}, // sizes, not minutes: no date operand
		{"2026-01-01", "epoch"},
		{"now", "epoch"},
		{"@(1 + 2)", "epoch"},
		{"2024 - 1 - 2", "expr"},
	}
	for _, tt := range tests {
		if r := mustEval(t, tt.in); r.Kind != tt.kind {
			t.Errorf("Eval(%q).Kind = %s, want %s", tt.in, r.Kind, tt.kind)
		}
	}
}
