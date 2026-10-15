package calc

import (
	"fmt"
	"math"
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

// today returns the start of now's day in its location.
func today(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

func dateResult(t time.Time, now time.Time) Result {
	lines := []Line{
		{"date", t.In(now.Location()).Format("2006-01-02 Monday")},
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

// relative describes t from now's point of view, e.g. "3 days ago". A t at
// local midnight, such as a plain date or "today + 3d", is described in
// calendar days: "tomorrow", "in 3 days".
func relative(t, now time.Time) string {
	if lt := t.In(now.Location()); lt.Equal(today(lt)) {
		days := int(math.Round(lt.Sub(today(now)).Hours() / 24))
		switch {
		case days == 0:
			return "today"
		case days == 1:
			return "tomorrow"
		case days == -1:
			return "yesterday"
		case days > 1 && days < 365:
			return fmt.Sprintf("in %d days", days)
		case days < -1 && days > -365:
			return fmt.Sprintf("%d days ago", -days)
		}
	}
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
