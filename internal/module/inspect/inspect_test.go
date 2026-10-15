package inspect

import (
	"strings"
	"testing"
	"time"
)

var refNow = time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

// get returns the value of the first line with the given label in the
// section with the given title.
func get(r Result, section, label string) string {
	for _, s := range r.Sections {
		if s.Title != section {
			continue
		}
		for _, l := range s.Lines {
			if l.Label == label {
				return l.Value
			}
		}
	}
	return ""
}

func TestEmpty(t *testing.T) {
	if r := Inspect("", refNow); len(r.Sections) != 0 {
		t.Fatalf("empty input gave %v", r)
	}
}

func TestPlainTextEncodingsAndHashes(t *testing.T) {
	r := Inspect("hello", refNow)
	if r.Kind != "text" || len(r.Sections) != 2 {
		t.Fatalf("kind %q, %d sections", r.Kind, len(r.Sections))
	}
	want := map[string]string{
		"base64":    "aGVsbG8=",
		"base64url": "aGVsbG8",
		"hex":       "68656c6c6f",
		"json":      `"hello"`,
		"length":    "5 bytes, 5 chars",
	}
	for k, v := range want {
		if got := get(r, "encode", k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	hashes := map[string]string{
		"md5":    "5d41402abc4b2a76b9719d911017c592",
		"sha1":   "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
		"sha256": "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		"crc32":  "3610a686",
	}
	for k, v := range hashes {
		if got := get(r, "hashes", k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	if got := get(Inspect("a b&é", refNow), "encode", "url"); got != "a+b%26%C3%A9" {
		t.Errorf("url = %q", got)
	}
	if got := get(Inspect("é\nx", refNow), "encode", "length"); got != "4 bytes, 3 chars, 2 lines" {
		t.Errorf("length = %q", got)
	}
}

const jwt = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
	"eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjE5MDAwMDAwMDB9." +
	"SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

func TestJWT(t *testing.T) {
	for _, in := range []string{jwt, "Bearer " + jwt} {
		r := Inspect(in, refNow)
		if r.Kind != "jwt" {
			t.Fatalf("%q: kind %q", in, r.Kind)
		}
		checks := map[string]string{
			"alg":       "HS256",
			"sub":       "1234567890",
			"status":    "valid, expires in 3 years",
			"exp":       "2030-03-17 17:46:40 UTC (in 3 years)",
			"signature": "present, not verified",
		}
		for k, v := range checks {
			if got := get(r, "jwt", k); got != v {
				t.Errorf("%s = %q, want %q", k, got, v)
			}
		}
		if p := get(r, "jwt", "payload"); !strings.Contains(p, `  "name": "John Doe",`) {
			t.Errorf("payload not pretty-printed:\n%s", p)
		}
	}
	later := time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := get(Inspect(jwt, later), "jwt", "status"); got != "expired 289 days ago" {
		t.Errorf("status = %q", got)
	}
}

func TestJWTWithoutExpiry(t *testing.T) {
	// {"alg":"none"} . {"sub":"x"} . (no signature)
	r := Inspect("eyJhbGciOiJub25lIn0.eyJzdWIiOiJ4In0.", refNow)
	if get(r, "jwt", "status") != "no expiry" || get(r, "jwt", "signature") != "none" {
		t.Errorf("got %v", r.Sections[0])
	}
}

func TestJSON(t *testing.T) {
	r := Inspect(` {"b": [1, 2], "a": {"x": null}} `, refNow)
	if r.Kind != "json" {
		t.Fatalf("kind %q", r.Kind)
	}
	if got := get(r, "json", "minified"); got != `{"b":[1,2],"a":{"x":null}}` {
		t.Errorf("minified = %q (key order must be kept)", got)
	}
	if got := get(r, "json", "shape"); got != "2 keys" {
		t.Errorf("shape = %q", got)
	}
	if got := get(r, "json", "pretty"); !strings.HasPrefix(got, "{\n  \"b\": [\n    1,") {
		t.Errorf("pretty = %q", got)
	}
	if got := get(Inspect("[1]", refNow), "json", "shape"); got != "1 item" {
		t.Errorf("shape = %q", got)
	}
	if got := get(Inspect(`{"a": 1,}`, refNow), "json", "error"); !strings.Contains(got, "at byte 9") {
		t.Errorf("error = %q", got)
	}
	if got := get(Inspect(`"a\tbé"`, refNow), "json string", "text"); got != "a\tbé" {
		t.Errorf("unquoted = %q", got)
	}
}

func TestUUID(t *testing.T) {
	tests := []struct {
		in, version, time string
	}{
		{"f47ac10b-58cc-4372-a567-0e02b2c3d479", "4 (random)", ""},
		{"{F47AC10B-58CC-4372-A567-0E02B2C3D479}", "4 (random)", ""},
		{"urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479", "4 (random)", ""},
		{"018f3c2e-7b1a-7c3d-8e4f-123456789abc", "7 (Unix time-based)", "2024-05-03 02:00:30 UTC (2 years ago)"},
		// RFC 9562 test vectors: 2022-02-22 19:22:22 UTC.
		{"C232AB00-9414-11EC-B3C8-9F6BDECED846", "1 (time-based)", "2022-02-22 19:22:22 UTC (4 years ago)"},
		{"1EC9414C-232A-6B00-B3C8-9F6BDECED846", "6 (time-based, reordered)", "2022-02-22 19:22:22 UTC (4 years ago)"},
		{"00000000-0000-0000-0000-000000000000", "nil UUID", ""},
	}
	for _, tt := range tests {
		r := Inspect(tt.in, refNow)
		if r.Kind != "uuid" {
			t.Errorf("%s: kind %q", tt.in, r.Kind)
			continue
		}
		if got := get(r, "uuid", "version"); got != tt.version {
			t.Errorf("%s: version %q, want %q", tt.in, got, tt.version)
		}
		if got := get(r, "uuid", "time"); got != tt.time {
			t.Errorf("%s: time %q, want %q", tt.in, got, tt.time)
		}
	}
	if got := get(Inspect("C232AB00-9414-11EC-B3C8-9F6BDECED846", refNow), "uuid", "node"); got != "9f:6b:de:ce:d8:46" {
		t.Errorf("node = %q", got)
	}
	if Inspect("{f47ac10b-58cc-4372-a567-0e02b2c3d479", refNow).Kind == "uuid" {
		t.Error("unbalanced brace accepted")
	}
}

func TestGenerateUUID(t *testing.T) {
	r := Inspect(" UUID ", refNow)
	for _, label := range []string{"v4", "v7"} {
		v := get(r, "new uuid", label)
		u, ok := parseUUID(v)
		if !ok {
			t.Fatalf("%s: %q is not a UUID", label, v)
		}
		if want := int(label[1] - '0'); u.version() != want || u.variant() != "RFC 9562" {
			t.Errorf("%s: version %d, variant %s", label, u.version(), u.variant())
		}
	}
	u, _ := parseUUID(get(r, "new uuid", "v7"))
	if ts, _ := u.timestamp(); !ts.Equal(refNow) {
		t.Errorf("v7 time = %v", ts)
	}
}

func TestURL(t *testing.T) {
	r := Inspect("postgres://app:s3cret@db.local:5432/my%20db?sslmode=require&tag=a+b#x", refNow)
	want := map[string]string{
		"scheme":   "postgres",
		"user":     "app",
		"password": "******",
		"host":     "db.local",
		"port":     "5432",
		"path":     "/my db",
		"?sslmode": "require",
		"?tag":     "a b",
		"fragment": "x",
	}
	for k, v := range want {
		if got := get(r, "url", k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	if Inspect("not a url", refNow).Kind != "text" {
		t.Error("plain text taken for a URL")
	}
}

func TestPercentDecoding(t *testing.T) {
	r := Inspect("a%2Fb+c%C3%A9", refNow)
	if get(r, "url decoded", "decoded") != "a/b+cé" || get(r, "url decoded", "form decoded") != "a/b cé" {
		t.Errorf("got %v", r.Sections[0])
	}
}

func TestBase64(t *testing.T) {
	tests := []struct{ in, variant, text string }{
		{"aGVsbG8gd29ybGQ=", "standard", "hello world"},
		{"aGVsbG8gd29ybGQ", "standard, unpadded", "hello world"},
		{"PDw_Pz4-", "URL-safe", "<<??>>"},
	}
	for _, tt := range tests {
		r := Inspect(tt.in, refNow)
		if r.Kind != "base64" || get(r, "base64", "variant") != tt.variant || get(r, "base64", "text") != tt.text {
			t.Errorf("%s: got %v", tt.in, r.Sections[0])
		}
	}
	// Base64 of {"a":1} also shows the JSON.
	if got := get(Inspect("eyJhIjoxfQ==", refNow), "json", "minified"); got != `{"a":1}` {
		t.Errorf("embedded json = %q", got)
	}
	// Binary data is shown in hex once the text clearly is base64.
	if got := get(Inspect("AAECAwQFBgcICQoLDA0ODw==", refNow), "base64", "hex"); got != "000102030405060708090a0b0c0d0e0f" {
		t.Errorf("hex = %q", got)
	}
	// Words that merely happen to be valid base64 are left alone.
	for _, in := range []string{"test", "abcd", "deploy12"} {
		if k := Inspect(in, refNow).Kind; k != "text" {
			t.Errorf("%s: kind %q", in, k)
		}
	}
}

func TestHex(t *testing.T) {
	r := Inspect("48656c6c6f", refNow)
	if r.Kind != "hex" || get(r, "hex", "text") != "Hello" {
		t.Errorf("got %v", r.Sections[0])
	}
	if got := get(Inspect("de:ad:be:ef", refNow), "hex", "base64"); got != "3q2+7w==" {
		t.Errorf("separated hex base64 = %q", got)
	}
	if got := get(Inspect("0x4869", refNow), "hex", "text"); got != "Hi" {
		t.Errorf("0x hex text = %q", got)
	}
	sha := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got := get(Inspect(sha, refNow), "hex", "looks like"); got != "SHA-256 digest" {
		t.Errorf("looks like = %q", got)
	}
	// Plain numbers belong to the calculator.
	if k := Inspect("2024", refNow).Kind; k != "text" {
		t.Errorf("2024: kind %q", k)
	}
}
