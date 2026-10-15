package calc

import (
	"fmt"
	"strconv"
	"time"
)

// dateLayouts are tried in order when the input looks like a date. Layouts
// without a zone are read in the local time zone.
var dateLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
}

func parseDate(s string, loc *time.Location) (time.Time, bool) {
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// fromTimestamp interprets n as a Unix timestamp, guessing its unit from its
// magnitude: seconds, milliseconds, microseconds or nanoseconds.
func fromTimestamp(n int64) (time.Time, string) {
	abs := n
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs < 1e11:
		return time.Unix(n, 0), "seconds"
	case abs < 1e14:
		return time.UnixMilli(n), "milliseconds"
	case abs < 1e17:
		return time.UnixMicro(n), "microseconds"
	default:
		return time.Unix(0, n), "nanoseconds"
	}
}

func timestampResult(n int64, now time.Time) Result {
	t, unit := fromTimestamp(n)
	lines := []Line{{"unit", unit}}
	return Result{Kind: "epoch", Lines: append(lines, timeLines(t, now)...)}
}

func dateResult(t time.Time, now time.Time) Result {
	lines := []Line{
		{"epoch", strconv.FormatInt(t.Unix(), 10)},
		{"epoch ms", strconv.FormatInt(t.UnixMilli(), 10)},
	}
	return Result{Kind: "epoch", Lines: append(lines, timeLines(t, now)...)}
}

func timeLines(t, now time.Time) []Line {
	return []Line{
		{"UTC", t.UTC().Format(time.RFC3339)},
		{"local", t.In(now.Location()).Format("2006-01-02 15:04:05 MST")},
		{"relative", relative(t, now)},
	}
}

// relative describes t from now's point of view, e.g. "3 days ago".
func relative(t, now time.Time) string {
	d := t.Sub(now)
	future := d > 0
	if !future {
		d = -d
	}
	if d < time.Second {
		return "now"
	}
	steps := []struct {
		unit time.Duration
		name string
	}{
		{365 * 24 * time.Hour, "year"},
		{24 * time.Hour, "day"},
		{time.Hour, "hour"},
		{time.Minute, "minute"},
		{time.Second, "second"},
	}
	var s string
	for _, st := range steps {
		if d >= st.unit {
			n := int64(d / st.unit)
			s = fmt.Sprintf("%d %s", n, st.name)
			if n != 1 {
				s += "s"
			}
			break
		}
	}
	if future {
		return "in " + s
	}
	return s + " ago"
}
