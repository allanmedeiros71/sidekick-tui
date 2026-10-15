package cal

import (
	"strings"
	"testing"
	"time"
)

// refNow is a Thursday in ISO week 42.
var refNow = time.Date(2026, 10, 15, 14, 30, 0, 0, time.Local)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestParse(t *testing.T) {
	from := date(2026, 3, 31)
	tests := []struct {
		in   string
		want time.Time
	}{
		{"2026-12-25", date(2026, 12, 25)},
		{"2026-12", date(2026, 12, 1)},
		{"2027-2", date(2027, 2, 1)},
		{"2030", date(2030, 1, 1)},
		{"today", date(2026, 10, 15)},
		{"Today + 90d", date(2027, 1, 13)},
		{"2026-01-31 + 1mo", date(2026, 2, 28)},
		{"+2w", date(2026, 4, 14)},
		{"-1mo", date(2026, 2, 28)},
		{"+1y", date(2027, 3, 31)},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in, refNow, from)
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.in, err)
			continue
		}
		if !got.Equal(tt.want) {
			t.Errorf("Parse(%q) = %s, want %s", tt.in, got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
		}
	}
	for _, in := range []string{"", "2026-13", "tomorrow", "90d", "today + 1x", "2026-12-25 - today"} {
		if _, err := Parse(in, refNow, from); err == nil {
			t.Errorf("Parse(%q) should fail", in)
		}
	}
}

func TestAddMonthsClamps(t *testing.T) {
	if got := addMonths(date(2024, 1, 31), 1); !got.Equal(date(2024, 2, 29)) {
		t.Errorf("Jan 31 + 1 month = %s", got.Format("2006-01-02"))
	}
	if got := addMonths(date(2024, 2, 29), 12); !got.Equal(date(2025, 2, 28)) {
		t.Errorf("Feb 29 + 1 year = %s", got.Format("2006-01-02"))
	}
}

func TestMonthGrid(t *testing.T) {
	got := month(date(2026, 10, 15), date(2026, 10, 15), styles{})
	rows := strings.Split(got, "\n")
	if len(rows) != 8 {
		t.Fatalf("want a title, a header and 6 weeks, got %d rows:\n%s", len(rows), got)
	}
	for i, want := range map[int]string{
		0: "      October 2026      ",
		1: "wk  Mo Tu We Th Fr Sa Su",
		2: "40            1  2  3  4",
		4: "42  12 13 14 15 16 17 18",
		6: "44  26 27 28 29 30 31   ",
		7: "                        ",
	} {
		if rows[i] != want {
			t.Errorf("row %d = %q, want %q", i, rows[i], want)
		}
	}
	// Week 53 of 2026 leads January 2027.
	if row := strings.Split(month(date(2027, 1, 1), date(2026, 10, 15), styles{}), "\n")[2]; !strings.HasPrefix(row, "53 ") {
		t.Errorf("first week of January 2027 = %q, want ISO week 53", row)
	}
}

func TestDescribe(t *testing.T) {
	today := date(2026, 10, 15)
	tests := map[time.Time]string{
		today:              "Thursday 2026-10-15 · week 42 · day 288 of 365 · today",
		date(2026, 10, 16): "Friday 2026-10-16 · week 42 · day 289 of 365 · tomorrow",
		date(2024, 12, 31): "Tuesday 2024-12-31 · week 1 · day 366 of 366 · 653 days ago",
		date(2027, 1, 13):  "Wednesday 2027-01-13 · week 2 · day 13 of 365 · in 90 days",
	}
	for d, want := range tests {
		if got := describe(d, today); got != want {
			t.Errorf("describe(%s) = %q, want %q", d.Format("2006-01-02"), got, want)
		}
	}
}

func TestPrint(t *testing.T) {
	out, err := Print("2027-02", refNow)
	if err != nil {
		t.Fatal(err)
	}
	want := `     February 2027
wk  Mo Tu We Th Fr Sa Su
 5   1  2  3  4  5  6  7
 6   8  9 10 11 12 13 14
 7  15 16 17 18 19 20 21
 8  22 23 24 25 26 27 28

Monday 2027-02-01 · week 5 · day 32 of 365 · in 109 days`
	if out != want {
		t.Errorf("Print:\n%s\nwant:\n%s", out, want)
	}
}
