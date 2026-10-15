package calc

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// span is a length of time to add to a date. Years and days are calendar
// units (applied with AddDate, so a year from Feb 29 and a day across a DST
// change behave as on a calendar); the rest is exact.
type span struct {
	years, days int
	rest        time.Duration
}

func (s span) neg() span { return span{-s.years, -s.days, -s.rest} }

func (s span) add(o span) span {
	return span{s.years + o.years, s.days + o.days, s.rest + o.rest}
}

func (s span) applyTo(t time.Time) time.Time {
	return t.AddDate(s.years, 0, s.days).Add(s.rest)
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
	dateTermRe = regexp.MustCompile(`^(?i)\d{4}-\d{2}-\d{2}([t ]\d{2}:\d{2}(:\d{2}(\.\d+)?)?(z|[+-]\d{2}:\d{2})?)?`)
	nowTermRe  = regexp.MustCompile(`^(?i)now\b`)
	epochRe    = regexp.MustCompile(`^@\d+`)
	spanTermRe = regexp.MustCompile(`^(\d+[ywdhms])+\b`)
	spanPartRe = regexp.MustCompile(`(\d+)([ywdhms])`)
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
	return sign + describeDuration(int64(years), 0, to.Sub(from.AddDate(years, 0, 0)))
}

// describeDuration spells out years, days and an exact remainder, carrying
// whole days out of rest. Day counts of a week or more are also given in
// weeks, e.g. "168 days (24 weeks)".
func describeDuration(years, days int64, rest time.Duration) string {
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
// 365 days.
const yearSeconds = 365 * 24 * 60 * 60

func (s span) seconds() float64 {
	return float64(s.years)*yearSeconds + float64(s.days)*86400 + s.rest.Seconds()
}

var timeUnits = map[string]struct {
	seconds float64
	name    string
}{
	"s": {1, "seconds"},
	"m": {60, "minutes"},
	"h": {3600, "hours"},
	"d": {86400, "days"},
	"w": {7 * 86400, "weeks"},
	"y": {yearSeconds, "years"},
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
		{"duration", describeDuration(int64(sp.years), int64(sp.days), sp.rest)},
		{"seconds", formatFloat(secs)},
		{"minutes", formatFloat(secs / 60)},
		{"hours", formatFloat(secs / 3600)},
		{"days", formatFloat(secs / 86400)},
		{"weeks", formatFloat(secs / (7 * 86400))},
	}}
}

// convertSpan handles "30d in w" and "90m to h". ok is false when the
// input is not a duration converted to a time unit, so "1GiB in m" and
// "30m in g" stay size conversions.
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
		return Result{}, true, fmt.Errorf("unknown time unit %q: use s, m, h, d, w or y", unitText)
	}
	v := strconv.FormatFloat(sp.seconds()/u.seconds, 'f', 4, 64)
	v = strings.TrimSuffix(strings.TrimRight(v, "0"), ".")
	r = spanResult(sp)
	r.Lines = append([]Line{{"result", v + " " + u.name}}, r.Lines...)
	return r, true, nil
}
