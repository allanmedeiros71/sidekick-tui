// Package inspect is the text inspector module: it recognizes what a piece
// of text is (JWT, JSON, UUID, URL, base64, hex) and shows it decoded, along
// with the text's encodings and hashes. Inspect is a pure function; Model
// wraps it in a Bubble Tea UI.
package inspect

import (
	"fmt"
	"strings"
	"time"
)

// Line is one labelled value, e.g. {"sha256", "2c26b4..."}. Values may span
// several lines (pretty-printed JSON).
type Line struct {
	Label, Value string
}

// Section groups the lines of one interpretation of the input.
type Section struct {
	Title string
	Lines []Line
}

// Result is everything Inspect found. Kind names the main interpretation
// (jwt, json, uuid, url, base64, hex, text).
type Result struct {
	Kind     string
	Sections []Section
}

// Lines returns every line of every section, in display order.
func (r Result) Lines() []Line {
	var out []Line
	for _, s := range r.Sections {
		out = append(out, s.Lines...)
	}
	return out
}

// Inspect describes input. "uuid" generates fresh UUIDs; anything else is
// decoded as the first matching kind (JWT, UUID, JSON, URL, percent-encoded
// text, hex, base64) and is always followed by its encodings and hashes. now
// is the reference for relative times and its location is the local zone.
func Inspect(input string, now time.Time) Result {
	if input == "" {
		return Result{}
	}
	if strings.EqualFold(strings.TrimSpace(input), "uuid") {
		return generateUUIDs(now)
	}
	r := Result{Kind: "text"}
	s := strings.TrimSpace(input)
	detectors := []struct {
		kind string
		fn   func(string, time.Time) []Section
	}{
		{"jwt", jwtSections},
		{"uuid", uuidSections},
		{"json", jsonSections},
		{"url", urlSections},
		{"url", percentSections},
		{"hex", hexSections},
		{"base64", base64Sections},
	}
	for _, d := range detectors {
		if secs := d.fn(s, now); secs != nil {
			r.Kind = d.kind
			r.Sections = secs
			break
		}
	}
	r.Sections = append(r.Sections, encodeSection(input), hashSection(input))
	return r
}

// relative describes t from now's point of view, e.g. "in 3 days".
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

func formatTime(t, now time.Time) string {
	return fmt.Sprintf("%s (%s)", t.In(now.Location()).Format("2006-01-02 15:04:05 MST"), relative(t, now))
}
