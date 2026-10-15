package inspect

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// jsonSections pretty-prints a JSON object or array, or unquotes a JSON
// string. Text that starts like an object or array but does not parse is
// reported with the position of the error.
func jsonSections(s string, _ time.Time) []Section {
	switch {
	case strings.HasPrefix(s, "{"), strings.HasPrefix(s, "["):
	case strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && len(s) > 1:
		var str string
		if json.Unmarshal([]byte(s), &str) != nil {
			return nil
		}
		return []Section{{"json string", []Line{{"text", str}}}}
	default:
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		msg := err.Error()
		var syn *json.SyntaxError
		if errors.As(err, &syn) {
			msg = fmt.Sprintf("%s (at byte %d)", syn.Error(), syn.Offset)
		}
		return []Section{{"json", []Line{{"error", msg}}}}
	}
	return []Section{{"json", jsonLines([]byte(s), v)}}
}

// jsonLines describes valid JSON data: its shape, pretty and minified.
func jsonLines(data []byte, v any) []Line {
	var shape string
	switch v := v.(type) {
	case map[string]any:
		shape = plural(len(v), "key")
	case []any:
		shape = plural(len(v), "item")
	}
	var pretty, compact bytes.Buffer
	_ = json.Indent(&pretty, data, "", "  ")
	_ = json.Compact(&compact, data)
	lines := []Line{}
	if shape != "" {
		lines = append(lines, Line{"shape", shape})
	}
	return append(lines, Line{"pretty", pretty.String()}, Line{"minified", compact.String()})
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// urlSections splits an absolute URL (with a scheme and a host) into its parts.
func urlSections(s string, _ time.Time) []Section {
	if strings.ContainsAny(s, " \t\n") {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil
	}
	lines := []Line{{"scheme", u.Scheme}}
	if u.User != nil {
		lines = append(lines, Line{"user", u.User.Username()})
		if p, ok := u.User.Password(); ok {
			lines = append(lines, Line{"password", strings.Repeat("*", utf8.RuneCountInString(p))})
		}
	}
	lines = append(lines, Line{"host", u.Hostname()})
	if p := u.Port(); p != "" {
		lines = append(lines, Line{"port", p})
	}
	if u.Path != "" {
		lines = append(lines, Line{"path", u.Path})
	}
	if u.RawQuery != "" {
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			lines = append(lines, Line{"query", u.RawQuery})
		}
		// Keep the parameters in the order they appear in the URL.
		seen := map[string]bool{}
		for _, part := range strings.Split(u.RawQuery, "&") {
			k, _, _ := strings.Cut(part, "=")
			key, err := url.QueryUnescape(k)
			if err != nil || seen[key] || q[key] == nil {
				continue
			}
			seen[key] = true
			for _, v := range q[key] {
				lines = append(lines, Line{"?" + key, v})
			}
		}
	}
	if u.Fragment != "" {
		lines = append(lines, Line{"fragment", u.Fragment})
	}
	return []Section{{"url", lines}}
}

var percentEscape = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)

// percentSections decodes percent-encoded text such as "a%20b%2Fc".
func percentSections(s string, _ time.Time) []Section {
	if !percentEscape.MatchString(s) {
		return nil
	}
	path, err := url.PathUnescape(s)
	if err != nil {
		return nil
	}
	lines := []Line{{"decoded", path}}
	if strings.Contains(s, "+") {
		if form, err := url.QueryUnescape(s); err == nil && form != path {
			lines = append(lines, Line{"form decoded", form})
		}
	}
	return []Section{{"url decoded", lines}}
}

var (
	base64Text = regexp.MustCompile(`^[A-Za-z0-9+/_-]+={0,2}$`)
	hexText    = regexp.MustCompile(`^(?:0[xX])?[0-9A-Fa-f]+$`)
	hexPairs   = regexp.MustCompile(`^[0-9A-Fa-f]{2}(?:([:\- ])[0-9A-Fa-f]{2})+$`)
)

// hashNames maps the length of a hex digest to the hashes that produce it.
var hashNames = map[int]string{
	8:   "CRC-32",
	32:  "MD5",
	40:  "SHA-1",
	56:  "SHA-224",
	64:  "SHA-256",
	96:  "SHA-384",
	128: "SHA-512",
}

// hexSections decodes hex text ("48656c6c6f", "0x4869", "de:ad:be:ef"). To
// avoid claiming plain numbers and words, bare hex needs a letter a-f, and
// short bare hex must decode to readable text.
func hexSections(s string, _ time.Time) []Section {
	digits := s
	switch {
	case hexPairs.MatchString(s):
		digits = strings.NewReplacer(":", "", "-", "", " ", "").Replace(s)
	case hexText.MatchString(s):
		prefixed := strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X")
		if prefixed {
			digits = s[2:]
		} else if !strings.ContainsAny(strings.ToLower(s), "abcdef") {
			return nil
		}
	default:
		return nil
	}
	if len(digits)%2 != 0 {
		return nil
	}
	data, err := hex.DecodeString(digits)
	if err != nil {
		return nil
	}
	lines := []Line{{"bytes", strconv.Itoa(len(data))}}
	if name, ok := hashNames[len(digits)]; ok && digits == s {
		lines = append(lines, Line{"looks like", name + " digest"})
	}
	readable := printable(data)
	if !readable && digits == s && len(digits) < 8 {
		// Short words such as "cafe" or "abcd" are just words.
		return nil
	}
	if readable {
		lines = append(lines, Line{"text", string(data)})
	}
	lines = append(lines, Line{"base64", base64.StdEncoding.EncodeToString(data)})
	return []Section{{"hex", lines}}
}

// base64Sections decodes standard or URL-safe base64, padded or not. Short
// text that only happens to be valid base64 ("test", "abcd") is ignored
// unless it decodes to readable text.
func base64Sections(s string, _ time.Time) []Section {
	if len(s) < 4 || !base64Text.MatchString(s) {
		return nil
	}
	encodings := []struct {
		name string
		enc  *base64.Encoding
	}{
		{"standard", base64.StdEncoding},
		{"standard, unpadded", base64.RawStdEncoding},
		{"URL-safe", base64.URLEncoding},
		{"URL-safe, unpadded", base64.RawURLEncoding},
	}
	for _, e := range encodings {
		data, err := e.enc.DecodeString(s)
		if err != nil || len(data) == 0 {
			continue
		}
		readable := printable(data)
		strong := len(s) >= 16 || strings.ContainsAny(s, "+/=-_")
		if !readable && !strong {
			return nil
		}
		lines := []Line{{"variant", e.name}, {"bytes", strconv.Itoa(len(data))}}
		if readable {
			lines = append(lines, Line{"text", string(data)})
		} else {
			lines = append(lines, Line{"hex", hex.EncodeToString(data)})
		}
		secs := []Section{{"base64", lines}}
		var v any
		if t := bytes.TrimSpace(data); len(t) > 0 && (t[0] == '{' || t[0] == '[') && json.Unmarshal(t, &v) == nil {
			secs = append(secs, Section{"json", jsonLines(t, v)})
		}
		return secs
	}
	return nil
}

// printable reports whether data is UTF-8 text without control characters
// other than tabs and line breaks.
func printable(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	for _, r := range string(data) {
		if !unicode.IsPrint(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

// encodeSection shows the input encoded in the usual ways.
func encodeSection(s string) Section {
	var quoted bytes.Buffer
	enc := json.NewEncoder(&quoted)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	size := fmt.Sprintf("%s, %s", plural(len(s), "byte"), plural(utf8.RuneCountInString(s), "char"))
	if n := strings.Count(s, "\n"); n > 0 {
		size += ", " + plural(n+1, "line")
	}
	return Section{"encode", []Line{
		{"length", size},
		{"base64", base64.StdEncoding.EncodeToString([]byte(s))},
		{"base64url", base64.RawURLEncoding.EncodeToString([]byte(s))},
		{"url", url.QueryEscape(s)},
		{"hex", hex.EncodeToString([]byte(s))},
		{"json", strings.TrimSuffix(quoted.String(), "\n")},
	}}
}

// hashSection shows the common digests of the input, in hex.
func hashSection(s string) Section {
	b := []byte(s)
	md := md5.Sum(b)
	s1 := sha1.Sum(b)
	s256 := sha256.Sum256(b)
	s512 := sha512.Sum512(b)
	return Section{"hashes", []Line{
		{"md5", hex.EncodeToString(md[:])},
		{"sha1", hex.EncodeToString(s1[:])},
		{"sha256", hex.EncodeToString(s256[:])},
		{"sha512", hex.EncodeToString(s512[:])},
		{"crc32", fmt.Sprintf("%08x", crc32.ChecksumIEEE(b))},
	}}
}
