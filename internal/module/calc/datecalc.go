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
// hours, minutes and seconds, e.g. "1 year 3 days 4 hours".
func calendarDuration(from, to time.Time) string {
	sign := ""
	if to.Before(from) {
		from, to, sign = to, from, "-"
	}
	years := 0
	for !from.AddDate(years+1, 0, 0).After(to) {
		years++
	}
	rem := to.Sub(from.AddDate(years, 0, 0))
	parts := []struct {
		n    int64
		name string
	}{
		{int64(years), "year"},
		{int64(rem / (24 * time.Hour)), "day"},
		{int64(rem % (24 * time.Hour) / time.Hour), "hour"},
		{int64(rem % time.Hour / time.Minute), "minute"},
		{int64(rem % time.Minute / time.Second), "second"},
	}
	var out []string
	for _, p := range parts {
		if p.n == 0 {
			continue
		}
		s := fmt.Sprintf("%d %s", p.n, p.name)
		if p.n != 1 {
			s += "s"
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return "0 seconds"
	}
	return sign + strings.Join(out, " ")
}
