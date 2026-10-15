package calc

import (
	"strings"
	"testing"
	"time"
)

var refNow = time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

// get returns the value of the line with the given label, or "" if absent.
func get(r Result, label string) string {
	for _, l := range r.Lines {
		if l.Label == label {
			return l.Value
		}
	}
	return ""
}

func mustEval(t *testing.T, in string) Result {
	t.Helper()
	r, err := Eval(in, refNow)
	if err != nil {
		t.Fatalf("Eval(%q): %v", in, err)
	}
	return r
}

func TestExpressions(t *testing.T) {
	tests := []struct{ in, dec string }{
		{"42", "42"},
		{"0xff + 1", "256"},
		{"0b1010 | 0b0101", "15"},
		{"0o755", "493"},
		{"1_000_000", "1000000"},
		{"1 << 10", "1024"},
		{"0xF0 & 0x3C", "48"},
		{"0xF0 ^ 0xFF", "15"},
		{"~0", "-1"},
		{"2 + 3 * 4", "14"},
		{"(2 + 3) * 4", "20"},
		{"1 | 2 + 4", "7"},
		{"-2 ** 2", "-4"},
		{"2 ** 3 ** 2", "512"},
		{"2 ** 62", "4611686018427387904"},
		{"1 << 63", "-9223372036854775808"},
		{"-7 / 2", "-3"},
		{"-7 % 3", "-1"},
		{"256 >> 4", "16"},
		{"0xFFFFFFFFFFFFFFFF", "-1"},
		{"4KiB", "4096"},
		{"4 KiB * 2", "8192"},
		{"1.5GiB", "1610612736"},
		{"2MB", "2000000"},
		{"1k", "1024"},
	}
	for _, tt := range tests {
		r := mustEval(t, tt.in)
		if r.Kind != "expr" {
			t.Errorf("Eval(%q).Kind = %q, want expr", tt.in, r.Kind)
		}
		if got := get(r, "dec"); got != tt.dec {
			t.Errorf("Eval(%q) dec = %s, want %s", tt.in, got, tt.dec)
		}
	}
}

func TestNumberFormats(t *testing.T) {
	r := mustEval(t, "255")
	for label, want := range map[string]string{"hex": "0xFF", "oct": "0o377", "bin": "1111 1111"} {
		if got := get(r, label); got != want {
			t.Errorf("255 %s = %q, want %q", label, got, want)
		}
	}
	if get(r, "size") != "" {
		t.Error("plain numbers should not show a size")
	}

	r = mustEval(t, "-1")
	if got := get(r, "hex"); got != "0xFFFFFFFFFFFFFFFF" {
		t.Errorf("-1 hex = %s", got)
	}
	if got := get(r, "unsigned"); got != "18446744073709551615" {
		t.Errorf("-1 unsigned = %s", got)
	}

	if got := get(mustEval(t, "65"), "char"); got != "'A'" {
		t.Errorf("65 char = %s", got)
	}
	if got := get(mustEval(t, "1.5GiB"), "size"); got != "1.5 GiB" {
		t.Errorf("1.5GiB size = %s", got)
	}
	if got := get(mustEval(t, "1700000000"), "as epoch"); !strings.HasPrefix(got, "2023-11-14T22:13:20Z (seconds") {
		t.Errorf("1700000000 as epoch = %s", got)
	}
}

func TestExpressionErrors(t *testing.T) {
	tests := []struct{ in, msg string }{
		{"1 / 0", "division by zero"},
		{"2 ** 64", "overflow"},
		{"9223372036854775807 + 1", "overflow"},
		{"1 << 64", "out of range"},
		{"1.5", "fractions need a size unit"},
		{"0755", "0o755"},
		{"(1 + 2", "missing )"},
		{"1 +", "incomplete"},
		{"3 apples", "unknown unit"},
		{"1 $ 2", "unexpected"},
		{"2 3", "unexpected"},
	}
	for _, tt := range tests {
		_, err := Eval(tt.in, refNow)
		if err == nil || !strings.Contains(err.Error(), tt.msg) {
			t.Errorf("Eval(%q) error = %v, want it to mention %q", tt.in, err, tt.msg)
		}
	}
}

func TestSizeConversion(t *testing.T) {
	tests := []struct{ in, want string }{
		{"1.5GiB in MiB", "1536 MiB"},
		{"1GiB to MB", "1073.7418 MB"},
		{"4096 in k", "4 KiB"},
		{"1TB in GiB", "931.3226 GiB"},
	}
	for _, tt := range tests {
		r := mustEval(t, tt.in)
		if got := get(r, "result"); got != tt.want {
			t.Errorf("Eval(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if _, err := Eval("1GiB in parsecs", refNow); err == nil {
		t.Error("expected an unknown unit error")
	}
}

func TestEpoch(t *testing.T) {
	r := mustEval(t, "@1700000000")
	if get(r, "unit") != "seconds" || get(r, "UTC") != "2023-11-14T22:13:20Z" {
		t.Errorf("@1700000000 = %+v", r.Lines)
	}
	if got := get(r, "relative"); got != "2 years ago" {
		t.Errorf("relative = %s", got)
	}

	r = mustEval(t, "@1700000000123")
	if get(r, "unit") != "milliseconds" || get(r, "UTC") != "2023-11-14T22:13:20Z" {
		t.Errorf("@1700000000123 = %+v", r.Lines)
	}

	r = mustEval(t, "2026-10-16")
	if get(r, "epoch") != "1792108800" || get(r, "relative") != "tomorrow" {
		t.Errorf("2026-10-16 = %+v", r.Lines)
	}
	if got := get(mustEval(t, "2024-01-02T03:04:05-03:00"), "UTC"); got != "2024-01-02T06:04:05Z" {
		t.Errorf("zoned date UTC = %s", got)
	}
	if got := get(mustEval(t, "now"), "epoch"); got != "1792065600" {
		t.Errorf("now epoch = %s", got)
	}
}

func TestChmod(t *testing.T) {
	tests := []struct{ in, octal, sym string }{
		{"chmod 755", "0755", "rwxr-xr-x"},
		{"chmod 0644", "0644", "rw-r--r--"},
		{"chmod 4755", "4755", "rwsr-xr-x"},
		{"chmod 1777", "1777", "rwxrwxrwt"},
		{"chmod 2640", "2640", "rw-r-S---"},
		{"rwxr-x---", "0750", "rwxr-x---"},
		{"-rw-r--r--", "0644", "rw-r--r--"},
		{"drwxrwxrwt", "1777", "rwxrwxrwt"},
	}
	for _, tt := range tests {
		r := mustEval(t, tt.in)
		if get(r, "octal") != tt.octal || get(r, "symbolic") != tt.sym {
			t.Errorf("Eval(%q) = %s %s, want %s %s", tt.in, get(r, "octal"), get(r, "symbolic"), tt.octal, tt.sym)
		}
	}
	r := mustEval(t, "chmod 750")
	if get(r, "group") != "read, execute" || get(r, "other") != "none" || get(r, "command") != "chmod 750" {
		t.Errorf("chmod 750 = %+v", r.Lines)
	}
	for _, bad := range []string{"chmod 789", "chmod 77777", "chmod abc"} {
		if _, err := Eval(bad, refNow); err == nil {
			t.Errorf("Eval(%q): expected error", bad)
		}
	}
}

func TestCIDR(t *testing.T) {
	r := mustEval(t, "192.168.1.0/24")
	for label, want := range map[string]string{
		"network": "192.168.1.0/24", "netmask": "255.255.255.0", "wildcard": "0.0.0.255",
		"broadcast": "192.168.1.255", "first host": "192.168.1.1", "last host": "192.168.1.254",
		"hosts": "254", "addresses": "256",
	} {
		if got := get(r, label); got != want {
			t.Errorf("/24 %s = %s, want %s", label, got, want)
		}
	}
	if get(r, "note") != "" {
		t.Error("unexpected note for an aligned network")
	}

	r = mustEval(t, "10.1.2.3/8")
	if get(r, "network") != "10.0.0.0/8" || !strings.Contains(get(r, "note"), "host bits set") {
		t.Errorf("10.1.2.3/8 = %+v", r.Lines)
	}
	r = mustEval(t, "10.0.0.0/31")
	if get(r, "hosts") != "2" || get(r, "first host") != "10.0.0.0" {
		t.Errorf("/31 = %+v", r.Lines)
	}
	r = mustEval(t, "10.0.0.7/32")
	if get(r, "hosts") != "1" || get(r, "last host") != "10.0.0.7" {
		t.Errorf("/32 = %+v", r.Lines)
	}
	if got := get(mustEval(t, "0.0.0.0/0"), "addresses"); got != "4294967296" {
		t.Errorf("/0 addresses = %s", got)
	}

	r = mustEval(t, "2001:db8::/32")
	if get(r, "last") != "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff" || !strings.HasPrefix(get(r, "addresses"), "2^96 ") {
		t.Errorf("IPv6 /32 = %+v", r.Lines)
	}
}

func TestIP(t *testing.T) {
	r := mustEval(t, "10.0.0.1")
	if get(r, "decimal") != "167772161" || get(r, "hex") != "0x0A000001" || get(r, "scope") != "private" {
		t.Errorf("10.0.0.1 = %+v", r.Lines)
	}
	if got := get(mustEval(t, "::1"), "scope"); got != "loopback" {
		t.Errorf("::1 scope = %s", got)
	}
}

func TestEmptyInput(t *testing.T) {
	r, err := Eval("   ", refNow)
	if err != nil || len(r.Lines) != 0 {
		t.Errorf("empty input = %+v, %v", r, err)
	}
}
