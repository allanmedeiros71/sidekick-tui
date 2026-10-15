// Package calc is the developer calculator module: integer expressions in any
// base with bitwise operators, Unix timestamps, chmod modes, CIDR blocks and
// byte sizes. Eval is a pure function; Model wraps it in a Bubble Tea UI.
package calc

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// Line is one labelled value of a result, e.g. {"hex", "0xFF"}.
type Line struct {
	Label, Value string
}

// Result is what an input evaluates to, as labelled lines. Kind names the
// interpretation that was chosen (expr, size, epoch, chmod, cidr, ip).
type Result struct {
	Kind  string
	Lines []Line
}

// Summary is the first value of the result, used in the history list.
func (r Result) Summary() string {
	if len(r.Lines) == 0 {
		return ""
	}
	return r.Lines[0].Value
}

// Eval interprets input, trying in order: "now", "today", chmod modes, date
// arithmetic ("now + 3d", "2026-12-25 - now"), "@timestamp",
// CIDR blocks and IP addresses, dates, "<duration|size> in <unit>"
// conversions, integer expressions and finally bare durations ("30d"). now is the reference for relative times and
// its location is used as the local time zone.
func Eval(input string, now time.Time) (Result, error) {
	s := strings.TrimSpace(input)
	lower := strings.ToLower(s)
	if r, ok, err := evalDateExpr(s, now); ok {
		return r, err
	}
	switch {
	case s == "":
		return Result{}, nil
	case lower == "now":
		return dateResult(now, now), nil
	case lower == "today":
		return dateResult(today(now), now), nil
	case strings.HasPrefix(lower, "chmod "):
		return chmodResult(strings.TrimSpace(s[len("chmod "):]))
	case symbolicMode.MatchString(s):
		return chmodResult(s)
	case strings.HasPrefix(s, "@"):
		n, err := evalExpr(s[1:])
		if err != nil {
			return Result{}, err
		}
		return timestampResult(n.v, now), nil
	}
	if p, err := netip.ParsePrefix(s); err == nil {
		return cidrResult(p), nil
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return ipResult(a), nil
	}
	if t, ok := parseDate(s, now.Location()); ok {
		return dateResult(t, now), nil
	}
	for _, kw := range []string{" in ", " to "} {
		if i := strings.LastIndex(lower, kw); i >= 0 {
			unit := strings.TrimSpace(s[i+len(kw):])
			if r, ok, err := convertSpan(s[:i], unit); ok {
				return r, err
			}
			return convertSize(s[:i], unit)
		}
	}
	n, err := evalExpr(s)
	if err != nil {
		// "30d" or "1h30m" is not an integer expression but is a duration.
		if sp, ok := parseBareSpan(s); ok {
			return spanResult(sp), nil
		}
		return Result{}, err
	}
	return numberResult(n, now), nil
}

func numberResult(n number, now time.Time) Result {
	v := n.v
	lines := []Line{
		{"dec", strconv.FormatInt(v, 10)},
		{"hex", "0x" + strings.ToUpper(strconv.FormatUint(uint64(v), 16))},
		{"oct", "0o" + strconv.FormatUint(uint64(v), 8)},
		{"bin", groupBinary(uint64(v))},
	}
	if v < 0 {
		lines = append(lines, Line{"unsigned", strconv.FormatUint(uint64(v), 10)})
	}
	if n.sized {
		lines = append(lines,
			Line{"size", humanSize(v, 1024)},
			Line{"size SI", humanSize(v, 1000)})
	}
	if v >= 32 && v < 127 {
		lines = append(lines, Line{"char", fmt.Sprintf("%q", rune(v))})
	}
	// Numbers in the range of recent Unix timestamps get a date hint.
	if (v >= 1e9 && v < 1e10) || (v >= 1e12 && v < 1e13) {
		t, unit := fromTimestamp(v)
		lines = append(lines, Line{"as epoch", fmt.Sprintf("%s (%s, %s)", t.UTC().Format(time.RFC3339), unit, relative(t, now))})
	}
	return Result{Kind: "expr", Lines: lines}
}

// groupBinary prints x in binary, in groups of 4 bits.
func groupBinary(x uint64) string {
	s := strconv.FormatUint(x, 2)
	if pad := (4 - len(s)%4) % 4; pad > 0 {
		s = strings.Repeat("0", pad) + s
	}
	var b strings.Builder
	for i := 0; i < len(s); i += 4 {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s[i : i+4])
	}
	return b.String()
}
