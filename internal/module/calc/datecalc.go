package calc

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// span is a length of time to add to a date. Years, months and days are
// calendar units: adding months keeps the day of the month, clamped to the
// month's last day (Jan 31 + 1mo is Feb 28), and a day across a DST change
// keeps the clock time. The rest is exact.
type span struct {
	years, months, days int
	rest                time.Duration
}

func (s span) neg() span { return span{-s.years, -s.months, -s.days, -s.rest} }

func (s span) add(o span) span {
	return span{s.years + o.years, s.months + o.months, s.days + o.days, s.rest + o.rest}
}

func (s span) applyTo(t time.Time) time.Time {
	if months := s.years*12 + s.months; months != 0 {
		y, m, d := t.Date()
		first := time.Date(y, m+time.Month(months), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		last := first.AddDate(0, 1, -1).Day()
		t = first.AddDate(0, 0, min(d, last)-1)
	}
	return t.AddDate(0, 0, s.days).Add(s.rest)
}

type dateValueKind int

const (
	kindDate dateValueKind = iota
	kindSpan
	kindDiff
)

// dateValue is an operand or partial result of a date expression.
type dateValue struct {
	kind     dateValueKind
	t        time.Time // kindDate
	s        span      // kindSpan
	from, to time.Time // kindDiff: to - from
}

var (
	dateTermRe  = regexp.MustCompile(`^(?i)\d{4}-\d{2}-\d{2}([t ]\d{2}:\d{2}(:\d{2}(\.\d+)?)?(z|[+-]\d{2}:\d{2})?)?`)
	nowTermRe   = regexp.MustCompile(`^(?i)now\b`)
	todayTermRe = regexp.MustCompile(`^(?i)today\b`)
	epochRe     = regexp.MustCompile(`^@\d+`)
	spanTermRe  = regexp.MustCompile(`^(\d+(mo|[ywdhms]))+\b`)
	spanPartRe  = regexp.MustCompile(`(\d+)(mo|[ywdhms])`)
)

// evalDateExpr handles date arithmetic such as "now + 3d", "2026-01-01 - 2w"
// or "2026-12-25 - now". ok is false when s is not a date expression (no date
// operand or no operator), so the caller can try other interpretations.
func evalDateExpr(s string, now time.Time) (r Result, ok bool, err error) {
	acc, rest, isDate, matched := dateTerm(s, now)
	if !matched {
		return Result{}, false, nil
	}
	sawDate, sawOp := isDate, false
	for {
		rest = strings.TrimLeft(rest, " ")
		if rest == "" {
			break
		}
		op := rest[0]
		if op != '+' && op != '-' {
			if !sawDate {
				return Result{}, false, nil
			}
			return Result{}, true, fmt.Errorf("unexpected %q in date expression", rest)
		}
		rest = strings.TrimLeft(rest[1:], " ")
		next, after, isDate, matched := dateTerm(rest, now)
		if !matched {
			if !sawDate {
				return Result{}, false, nil
			}
			return Result{}, true, fmt.Errorf("expected a date or a duration such as 3d or 1y, got %q", rest)
		}
		sawOp, sawDate, rest = true, sawDate || isDate, after
		if acc, err = combine(acc, op, next); err != nil {
			if !sawDate {
				return Result{}, false, nil
			}
			return Result{}, true, err
		}
	}
	if !sawDate || !sawOp {
		return Result{}, false, nil
	}
	switch acc.kind {
	case kindDate:
		return dateResult(acc.t, now), true, nil
	case kindDiff:
		return diffResult(acc.from, acc.to), true, nil
	}
	return Result{}, true, errors.New("a date expression must result in a date or a difference")
}

// dateTerm reads one operand from the start of s.
func dateTerm(s string, now time.Time) (v dateValue, rest string, isDate, ok bool) {
	if m := dateTermRe.FindString(s); m != "" {
		if t, ok := parseDate(m, now.Location()); ok {
			return dateValue{kind: kindDate, t: t}, s[len(m):], true, true
		}
	}
	if m := todayTermRe.FindString(s); m != "" {
		return dateValue{kind: kindDate, t: today(now)}, s[len(m):], true, true
	}
	if m := nowTermRe.FindString(s); m != "" {
		return dateValue{kind: kindDate, t: now}, s[len(m):], true, true
	}
	if m := epochRe.FindString(s); m != "" {
		n, err := strconv.ParseInt(m[1:], 10, 64)
		if err == nil {
			t, _ := fromTimestamp(n)
			return dateValue{kind: kindDate, t: t}, s[len(m):], true, true
		}
	}
	if m := spanTermRe.FindString(s); m != "" {
		sp, err := parseSpan(m)
		if err == nil {
			return dateValue{kind: kindSpan, s: sp}, s[len(m):], false, true
		}
	}
	return dateValue{}, s, false, false
}

func parseSpan(s string) (span, error) {
	var sp span
	for _, m := range spanPartRe.FindAllStringSubmatch(s, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil || n > 1_000_000 {
			return span{}, fmt.Errorf("duration %q is too large", m[0])
		}
		switch m[2] {
		case "y":
			sp.years += n
		case "mo":
			sp.months += n
		case "w":
			sp.days += 7 * n
		case "d":
			sp.days += n
		case "h":
			sp.rest += time.Duration(n) * time.Hour
		case "m":
			sp.rest += time.Duration(n) * time.Minute
		case "s":
			sp.rest += time.Duration(n) * time.Second
		}
	}
	return sp, nil
}

func combine(a dateValue, op byte, b dateValue) (dateValue, error) {
	switch {
	case a.kind == kindDate && b.kind == kindSpan:
		if op == '-' {
			b.s = b.s.neg()
		}
		return dateValue{kind: kindDate, t: b.s.applyTo(a.t)}, nil
	case a.kind == kindSpan && b.kind == kindDate && op == '+':
		return dateValue{kind: kindDate, t: a.s.applyTo(b.t)}, nil
	case a.kind == kindSpan && b.kind == kindSpan:
		if op == '-' {
			b.s = b.s.neg()
		}
		return dateValue{kind: kindSpan, s: a.s.add(b.s)}, nil
	case a.kind == kindDate && b.kind == kindDate && op == '-':
		return dateValue{kind: kindDiff, from: b.t, to: a.t}, nil
	case a.kind == kindDate && b.kind == kindDate:
		return dateValue{}, errors.New("cannot add two dates; subtract them to get the time between")
	case a.kind == kindSpan:
		return dateValue{}, errors.New("cannot subtract a date from a duration")
	}
	return dateValue{}, errors.New("a difference between dates cannot be combined further")
}

func diffResult(from, to time.Time) Result {
	d := to.Sub(from)
	return Result{Kind: "duration", Lines: []Line{
		{"duration", calendarDuration(from, to)},
		{"seconds", strconv.FormatInt(int64(d/time.Second), 10)},
		{"hours", formatFloat(d.Hours())},
		{"days", formatFloat(d.Hours() / 24)},
		{"weeks", formatFloat(d.Hours() / 24 / 7)},
	}}
}

// calendarDuration describes to - from in whole calendar years, then days,
// hours, minutes and seconds, e.g. "1 year 10 days (1 week 3 days) 4 hours".
func calendarDuration(from, to time.Time) string {
	sign := ""
	if to.Before(from) {
		from, to, sign = to, from, "-"
	}
	years := 0
	for !from.AddDate(years+1, 0, 0).After(to) {
		years++
	}
	return sign + describeDuration(int64(years), 0, 0, to.Sub(from.AddDate(years, 0, 0)))
}

// describeDuration spells out years, months, days and an exact remainder, carrying
// whole days out of rest. Day counts of a week or more are also given in
// weeks, e.g. "168 days (24 weeks)".
func describeDuration(years, months, days int64, rest time.Duration) string {
	days += int64(rest / (24 * time.Hour))
	rest %= 24 * time.Hour
	var out []string
	add := func(n int64, name string) {
		if n == 0 {
			return
		}
		out = append(out, plural(n, name))
	}
	add(years, "year")
	add(months, "month")
	add(days, "day")
	if days >= 7 {
		weeks := plural(days/7, "week")
		if days%7 != 0 {
			weeks += " " + plural(days%7, "day")
		}
		out = append(out, "("+weeks+")")
	}
	add(int64(rest/time.Hour), "hour")
	add(int64(rest%time.Hour/time.Minute), "minute")
	add(int64(rest%time.Minute/time.Second), "second")
	if len(out) == 0 {
		return "0 seconds"
	}
	return strings.Join(out, " ")
}

func plural(n int64, name string) string {
	s := fmt.Sprintf("%d %s", n, name)
	if n != 1 {
		s += "s"
	}
	return s
}

// Durations outside a date have no calendar to follow, so a year counts as
// 365 days and a month as a twelfth of that.
const yearSeconds = 365 * 24 * 60 * 60

func (s span) seconds() float64 {
	return float64(s.years)*yearSeconds + float64(s.months)*yearSeconds/12 +
		float64(s.days)*86400 + s.rest.Seconds()
}

var timeUnits = map[string]struct {
	seconds float64
	name    string
}{
	"s":  {1, "seconds"},
	"m":  {60, "minutes"},
	"h":  {3600, "hours"},
	"d":  {86400, "days"},
	"w":  {7 * 86400, "weeks"},
	"mo": {yearSeconds / 12, "months"},
	"y":  {yearSeconds, "years"},
}

// parseBareSpan reads a whole input such as "30d" or "1h30m" as a duration.
func parseBareSpan(s string) (span, bool) {
	if spanTermRe.FindString(s) != s {
		return span{}, false
	}
	sp, err := parseSpan(s)
	return sp, err == nil
}

// spanResult shows a duration in every time unit.
func spanResult(sp span) Result {
	secs := sp.seconds()
	return Result{Kind: "duration", Lines: []Line{
		{"duration", describeDuration(int64(sp.years), int64(sp.months), int64(sp.days), sp.rest)},
		{"seconds", formatFloat(secs)},
		{"minutes", formatFloat(secs / 60)},
		{"hours", formatFloat(secs / 3600)},
		{"days", formatFloat(secs / 86400)},
		{"weeks", formatFloat(secs / (7 * 86400))},
	}}
}

// convertSpan handles "30d in w" and "90m to h". ok is false when the
// input is not a duration converted to a time unit, so "1GiB in M" and
// "3072M in g" stay size conversions.
func convertSpan(exprText, unitText string) (r Result, ok bool, err error) {
	sp, isSpan := parseBareSpan(strings.TrimSpace(exprText))
	if !isSpan {
		return Result{}, false, nil
	}
	u, isTime := timeUnits[unitText]
	if !isTime {
		if _, isSize := lookupUnit(unitText); isSize {
			return Result{}, false, nil
		}
		return Result{}, true, fmt.Errorf("unknown time unit %q: use s, m, h, d, w, mo or y", unitText)
	}
	v := strconv.FormatFloat(sp.seconds()/u.seconds, 'f', 4, 64)
	v = strings.TrimSuffix(strings.TrimRight(v, "0"), ".")
	r = spanResult(sp)
	r.Lines = append([]Line{{"result", v + " " + u.name}}, r.Lines...)
	return r, true, nil
}

// ageResult describes how old someone born on birth is today, in whole
// calendar years and months, then weeks and days.
func ageResult(input string, now time.Time) (Result, error) {
	b, rest, isDate, ok := dateTerm(strings.TrimSpace(input), now)
	if !ok || !isDate || strings.TrimSpace(rest) != "" {
		return Result{}, fmt.Errorf("age needs a date, e.g. age 1990-05-17")
	}
	birth, to := today(b.t.In(now.Location())), today(now)
	if birth.After(to) {
		return Result{}, errors.New("that date is in the future")
	}

	months := 0
	for !(span{months: months + 1}).applyTo(birth).After(to) {
		months++
	}
	anniversary := span{months: months}.applyTo(birth)
	days := int64(math.Round(to.Sub(anniversary).Hours() / 24))

	var parts []string
	if y := int64(months / 12); y > 0 {
		parts = append(parts, plural(y, "year"))
	}
	if m := int64(months % 12); m > 0 {
		parts = append(parts, plural(m, "month"))
	}
	if days >= 7 {
		parts = append(parts, "("+plural(days/7, "week")+")")
	}
	if d := days % 7; d > 0 || len(parts) == 0 {
		parts = append(parts, plural(d, "day"))
	}

	next := span{years: months/12 + 1}.applyTo(birth)
	lived := int64(math.Round(to.Sub(birth).Hours() / 24))
	return Result{Kind: "age", Lines: []Line{
		{"age", strings.Join(parts, " ")},
		{"born", birth.Format("2006-01-02 Monday")},
		{"next birthday", fmt.Sprintf("%s (%s)", next.Format("2006-01-02 Monday"), relative(next, now))},
		{"days", strconv.FormatInt(lived, 10)},
		{"weeks", formatFloat(float64(lived) / 7)},
	}}, nil
}
