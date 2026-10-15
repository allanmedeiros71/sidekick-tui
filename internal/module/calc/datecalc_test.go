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
		{"2024-02-29 + 1y", "2025-02-28T00:00:00Z"},
		{"2024-02-29 + 4y", "2028-02-29T00:00:00Z"},
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
		{"2026-12-25 - now", "70 days (10 weeks) 12 hours", "70.5"},
		{"now - 2026-12-25", "-70 days (10 weeks) 12 hours", "-70.5"},
		{"2027-04-30 - 2026-04-12", "1 year 18 days (2 weeks 4 days)", "383"},
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
		{"1M + 2M", "expr"}, // MiB, not minutes
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

func TestDurationConversion(t *testing.T) {
	tests := []struct{ in, want string }{
		{"30d in w", "4.2857 weeks"},
		{"90m in h", "1.5 hours"},
		{"1y in d", "365 days"},
		{"2w to d", "14 days"},
		{"1h30m in s", "5400 seconds"},
		{"1m in s", "60 seconds"},
	}
	for _, tt := range tests {
		if got := get(mustEval(t, tt.in), "result"); got != tt.want {
			t.Errorf("Eval(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := get(mustEval(t, "1GiB in M"), "result"); got != "1024 MiB" {
		t.Errorf("1GiB in M = %q, want a size conversion", got)
	}
	if got := get(mustEval(t, "3072M in g"), "result"); got != "3 GiB" {
		t.Errorf("3072M in g = %q, want a size conversion", got)
	}
	if _, err := Eval("10d to x", refNow); err == nil || !strings.Contains(err.Error(), "unknown time unit") {
		t.Errorf("10d to x error = %v", err)
	}
}

func TestBareDuration(t *testing.T) {
	r := mustEval(t, "30d")
	if r.Kind != "duration" || get(r, "duration") != "30 days (4 weeks 2 days)" || get(r, "hours") != "720" {
		t.Errorf("30d = %+v", r.Lines)
	}
	if got := get(mustEval(t, "2w3d"), "duration"); got != "17 days (2 weeks 3 days)" {
		t.Errorf("2w3d = %s", got)
	}
	if got := get(mustEval(t, "1h30m"), "minutes"); got != "90" {
		t.Errorf("1h30m minutes = %s", got)
	}
	if got := get(mustEval(t, "48h"), "duration"); got != "2 days" {
		t.Errorf("48h = %s", got)
	}
	if got := get(mustEval(t, "30m"), "duration"); got != "30 minutes" {
		t.Errorf("30m = %s, want 30 minutes", got)
	}
	if got := get(mustEval(t, "30M"), "size"); got != "30 MiB" {
		t.Errorf("30M size = %s, want 30 MiB", got)
	}
}

func TestLowercaseMIsMinutes(t *testing.T) {
	_, err := Eval("1m + 2m", refNow)
	if err == nil || !strings.Contains(err.Error(), "write M for MiB") {
		t.Errorf("1m + 2m error = %v", err)
	}
	if _, err := Eval("1GiB in m", refNow); err == nil {
		t.Error("1GiB in m: expected an error, m is minutes")
	}
	if got := get(mustEval(t, "2MB + 1mb"), "dec"); got != "3000000" {
		t.Errorf("2MB + 1mb = %s: multi-letter units stay case-insensitive", got)
	}
}

func TestToday(t *testing.T) {
	r := mustEval(t, "today")
	if get(r, "date") != "2026-10-15 Thursday" || get(r, "UTC") != "2026-10-15T00:00:00Z" || get(r, "relative") != "today" {
		t.Errorf("today = %+v", r.Lines)
	}
	tests := []struct{ in, date, rel string }{
		{"today + 1d", "2026-10-16 Friday", "tomorrow"},
		{"today - 1d", "2026-10-14 Wednesday", "yesterday"},
		{"today + 2w", "2026-10-29 Thursday", "in 14 days"},
		{"today + 3mo", "2027-01-15 Friday", "in 92 days"},
		{"today - 1y", "2025-10-15 Wednesday", "1 year ago"},
		{"today + 1y2mo3d", "2027-12-18 Saturday", "in 1 year"},
	}
	for _, tt := range tests {
		r := mustEval(t, tt.in)
		if get(r, "date") != tt.date || get(r, "relative") != tt.rel {
			t.Errorf("Eval(%q) = %s / %s, want %s / %s", tt.in, get(r, "date"), get(r, "relative"), tt.date, tt.rel)
		}
	}
	r = mustEval(t, "2026-12-25 - today")
	if get(r, "duration") != "71 days (10 weeks 1 day)" {
		t.Errorf("2026-12-25 - today = %s", get(r, "duration"))
	}
}

func TestMonths(t *testing.T) {
	for in, want := range map[string]string{
		"2026-01-31 + 1mo":  "2026-02-28 Saturday",
		"2024-01-31 + 1mo":  "2024-02-29 Thursday",
		"2026-03-31 - 1mo":  "2026-02-28 Saturday",
		"2026-01-15 + 13mo": "2027-02-15 Monday",
		"2026-05-31 - 7mo":  "2025-10-31 Friday",
	} {
		if got := get(mustEval(t, in), "date"); got != want {
			t.Errorf("%s = %s, want %s", in, got, want)
		}
	}
	if got := get(mustEval(t, "12mo in y"), "result"); got != "1 years" {
		t.Errorf("12mo in y = %s", got)
	}
	if got := get(mustEval(t, "3mo"), "duration"); got != "3 months" {
		t.Errorf("3mo = %s", got)
	}
	if got := get(mustEval(t, "1mo in d"), "result"); got != "30.4167 days" {
		t.Errorf("1mo in d = %s", got)
	}
}

func TestAge(t *testing.T) {
	tests := []struct{ in, age, next string }{
		{"age 1971-04-30", "55 years 5 months (2 weeks) 1 day", "2027-04-30 Friday (in 197 days)"},
		{"age 2025-10-15", "1 year", "2027-10-15 Friday (in 1 year)"},
		{"age 2026-09-08", "1 month (1 week)", "2027-09-08 Wednesday (in 328 days)"},
		{"age 2026-10-14", "1 day", "2027-10-14 Thursday (in 364 days)"},
		{"age 2026-10-15", "0 days", "2027-10-15 Friday (in 1 year)"},
		{"age 2024-02-29", "2 years 7 months (2 weeks) 2 days", "2027-02-28 Sunday (in 136 days)"},
		{"age 2000-10-16", "25 years 11 months (4 weeks) 1 day", "2026-10-16 Friday (tomorrow)"},
	}
	for _, tt := range tests {
		r := mustEval(t, tt.in)
		if get(r, "age") != tt.age || get(r, "next birthday") != tt.next {
			t.Errorf("Eval(%q) = %q / %q, want %q / %q", tt.in, get(r, "age"), get(r, "next birthday"), tt.age, tt.next)
		}
	}
	if got := get(mustEval(t, "age 1971-04-30"), "days"); got != "20257" {
		t.Errorf("days lived = %s", got)
	}
	for _, bad := range []string{"age 2030-01-01", "age foo", "age 2020-01-01 + 1d"} {
		if _, err := Eval(bad, refNow); err == nil {
			t.Errorf("Eval(%q): expected an error", bad)
		}
	}
}
