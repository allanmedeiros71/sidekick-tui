package calc

import (
	"strconv"
	"strings"
)

// units maps lower-case size suffixes to bytes. Bare letters (K, M, G...) are
// binary, as in dd, ls -h and most dev tooling; "KB", "MB"... are SI (powers
// of 1000) and "KiB", "MiB"... are explicitly binary. Suffixes are
// case-insensitive except bare "m", which means minutes: write "M" for MiB.
var units = map[string]int64{
	"b": 1, "byte": 1, "bytes": 1,
	"k": 1 << 10, "kib": 1 << 10, "kb": 1e3,
	"m": 1 << 20, "mib": 1 << 20, "mb": 1e6,
	"g": 1 << 30, "gib": 1 << 30, "gb": 1e9,
	"t": 1 << 40, "tib": 1 << 40, "tb": 1e12,
	"p": 1 << 50, "pib": 1 << 50, "pb": 1e15,
	"e": 1 << 60, "eib": 1 << 60, "eb": 1e18,
}

func lookupUnit(s string) (int64, bool) {
	if s == "m" {
		return 0, false
	}
	u, ok := units[strings.ToLower(s)]
	return u, ok
}

// unitName returns the canonical spelling of a unit for display.
func unitName(s string) string {
	l := strings.ToLower(s)
	switch {
	case l == "b" || l == "byte" || l == "bytes":
		return "B"
	case len(l) == 1:
		return strings.ToUpper(l) + "iB"
	case len(l) == 3: // kib
		return strings.ToUpper(l[:1]) + "iB"
	default: // kb
		return strings.ToUpper(l)
	}
}

// humanSize formats n bytes with the largest fitting unit of the given base.
func humanSize(n int64, base int64) string {
	names := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	if base == 1024 {
		names = []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	}
	neg := n < 0
	f := float64(n)
	if neg {
		f = -f
	}
	unit, name := 1.0, "B"
	for _, nm := range names {
		next := unit * float64(base)
		if f < next {
			break
		}
		unit, name = next, nm
	}
	s := formatFloat(f/unit) + " " + name
	if neg {
		s = "-" + s
	}
	return s
}

// formatFloat prints up to 2 decimals, without trailing zeros.
func formatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// convertSize handles "<expr> in <unit>" and "<expr> to <unit>".
func convertSize(exprText, unitText string) (Result, error) {
	n, err := evalExpr(exprText)
	if err != nil {
		return Result{}, err
	}
	u, ok := lookupUnit(unitText)
	if !ok {
		return Result{}, &unknownUnitError{unitText}
	}
	v := strconv.FormatFloat(float64(n.v)/float64(u), 'f', 4, 64)
	v = strings.TrimSuffix(strings.TrimRight(v, "0"), ".")
	return Result{Kind: "size", Lines: []Line{
		{"result", v + " " + unitName(unitText)},
		{"bytes", strconv.FormatInt(n.v, 10)},
		{"binary", humanSize(n.v, 1024)},
		{"SI", humanSize(n.v, 1000)},
	}}, nil
}

type unknownUnitError struct{ unit string }

func (e *unknownUnitError) Error() string {
	if e.unit == "m" {
		return `"m" is minutes: write M for MiB`
	}
	return "unknown unit " + strconv.Quote(e.unit)
}
