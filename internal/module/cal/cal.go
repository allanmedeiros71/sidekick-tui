// Package cal is the calendar: a month grid with ISO week numbers, moved
// with the arrow keys and able to jump to dates such as "today + 90d".
package cal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/allanmedeiros71/sidekick-tui/internal/module/calc"
)

// Dates are kept as civil dates: midnight UTC, so adding days never trips
// over a daylight saving change.

// civil returns t's calendar date in its own location, as a civil date.
func civil(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// addMonths moves d by n months, keeping the day of the month clamped to the
// target month's last day (Jan 31 + 1 month is Feb 28), as the calculator
// does.
func addMonths(d time.Time, n int) time.Time {
	first := time.Date(d.Year(), d.Month()+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1).Day()
	return first.AddDate(0, 0, min(d.Day(), last)-1)
}

// daysBetween counts calendar days from a to b (both civil dates).
func daysBetween(a, b time.Time) int {
	return int(b.Sub(a).Hours() / 24)
}

var (
	monthRe = regexp.MustCompile(`^(\d{4})-(\d{1,2})$`)
	yearRe  = regexp.MustCompile(`^\d{4}$`)
)

// Parse reads a date to jump to: a month ("2026-12", the 1st), a year
// ("2027", January 1st), anything the calculator takes as a date ("today",
// "2026-12-25", "@1700000000", "today + 90d"), or a duration starting with
// + or - ("+90d", "-2w"), counted from the date from.
func Parse(s string, now, from time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("type a date such as 2026-12-25, 2026-12, today + 90d or +2w")
	}
	if m := monthRe.FindStringSubmatch(s); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		if mo < 1 || mo > 12 {
			return time.Time{}, fmt.Errorf("month %d is out of range", mo)
		}
		return time.Date(y, time.Month(mo), 1, 0, 0, 0, 0, time.UTC), nil
	}
	if yearRe.MatchString(s) {
		y, _ := strconv.Atoi(s)
		return time.Date(y, time.January, 1, 0, 0, 0, 0, time.UTC), nil
	}
	if s[0] == '+' || s[0] == '-' {
		// Evaluate in from's terms; civil dates are midnight UTC.
		s = from.Format("2006-01-02") + " " + s
		now = now.In(time.UTC)
	}
	t, err := calc.ParseDate(s, now)
	if err != nil {
		return time.Time{}, err
	}
	return civil(t), nil
}

// styles are what a month is drawn with. The zero value draws plain text.
type styles struct {
	header, weekNum, day, weekend, today, cursor lipgloss.Style
}

// gridWidth is the width of a rendered month: the week column and 7 days.
const gridWidth = 3 + 7*3

// month renders the month holding cursor as a grid of weeks starting on
// Monday, each row led by its ISO week number. The grid always has six week
// rows so its height does not change while moving between months.
func month(cursor, today time.Time, st styles) string {
	first := time.Date(cursor.Year(), cursor.Month(), 1, 0, 0, 0, 0, time.UTC)
	// Back up to the Monday on or before the 1st.
	start := first.AddDate(0, 0, -((int(first.Weekday()) + 6) % 7))

	title := fmt.Sprintf("%s %d", cursor.Month(), cursor.Year())
	pad := max(0, (gridWidth-len(title))/2)
	rows := []string{
		st.header.Render(fmt.Sprintf("%-*s", gridWidth, strings.Repeat(" ", pad)+title)),
		st.weekNum.Render("wk ") + st.weekNum.Render(" Mo Tu We Th Fr Sa Su"),
	}
	for w := range 6 {
		monday := start.AddDate(0, 0, 7*w)
		var b strings.Builder
		if monday.Month() == first.Month() || monday.AddDate(0, 0, 6).Month() == first.Month() {
			_, wk := monday.ISOWeek()
			b.WriteString(st.weekNum.Render(fmt.Sprintf("%2d ", wk)))
		} else {
			b.WriteString(st.day.Render("   "))
		}
		for i := range 7 {
			d := monday.AddDate(0, 0, i)
			b.WriteString(st.day.Render(" "))
			if d.Month() != first.Month() {
				b.WriteString(st.day.Render("  "))
				continue
			}
			style := st.day
			switch {
			case d.Equal(cursor):
				style = st.cursor
			case d.Equal(today):
				style = st.today
			case i >= 5:
				style = st.weekend
			}
			b.WriteString(style.Render(fmt.Sprintf("%2d", d.Day())))
		}
		rows = append(rows, b.String())
	}
	return strings.Join(rows, "\n")
}

// describe says where d falls: weekday, ISO week, day of the year and how far
// it is from today.
func describe(d, today time.Time) string {
	_, wk := d.ISOWeek()
	yearDays := time.Date(d.Year(), 12, 31, 0, 0, 0, 0, time.UTC).YearDay()
	return fmt.Sprintf("%s %s · week %d · day %d of %d · %s",
		d.Weekday(), d.Format("2006-01-02"), wk, d.YearDay(), yearDays, relative(daysBetween(today, d)))
}

func relative(n int) string {
	switch {
	case n == 0:
		return "today"
	case n == 1:
		return "tomorrow"
	case n == -1:
		return "yesterday"
	case n > 0:
		return fmt.Sprintf("in %d days", n)
	}
	return fmt.Sprintf("%d days ago", -n)
}

// Print renders the month holding the date s, for "sidekick cal <date>":
// the grid as plain text and a line describing the date.
func Print(s string, now time.Time) (string, error) {
	today := civil(now)
	d, err := Parse(s, now, today)
	if err != nil {
		return "", err
	}
	lines := strings.Split(month(d, today, styles{}), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	grid := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	return grid + "\n\n" + describe(d, today), nil
}
