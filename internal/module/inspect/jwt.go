package inspect

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// timeClaims are the registered JWT claims holding NumericDate values.
var timeClaims = []string{"iat", "nbf", "exp", "auth_time"}

// jwtSections decodes a JSON Web Token without verifying its signature.
func jwtSections(s string, now time.Time) []Section {
	s = strings.TrimPrefix(s, "Bearer ")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil
	}
	header, ok := jwtPart(parts[0])
	if !ok {
		return nil
	}
	var h map[string]any
	if json.Unmarshal(header, &h) != nil || (h["alg"] == nil && h["typ"] == nil) {
		return nil
	}
	payload, ok := jwtPart(parts[1])
	if !ok {
		return nil
	}

	var lines []Line
	for _, k := range []string{"alg", "typ", "kid"} {
		if v, ok := h[k].(string); ok {
			lines = append(lines, Line{k, v})
		}
	}
	claims := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	if dec.Decode(&claims) != nil {
		// Not JSON: an encrypted or nested payload. Show it as it is.
		lines = append(lines, Line{"payload", string(payload)})
		return []Section{{"jwt", lines}}
	}
	lines = append(lines, Line{"status", jwtStatus(claims, now)})
	for _, k := range timeClaims {
		if t, ok := claimTime(claims, k); ok {
			lines = append(lines, Line{k, formatTime(t, now)})
		}
	}
	for _, k := range []string{"iss", "sub", "aud"} {
		if v, ok := claims[k]; ok {
			b, _ := json.Marshal(v)
			lines = append(lines, Line{k, strings.Trim(string(b), `"`)})
		}
	}
	signature := "none"
	if parts[2] != "" {
		signature = "present, not verified"
	}
	lines = append(lines, Line{"signature", signature})

	var ph, pp bytes.Buffer
	_ = json.Indent(&ph, header, "", "  ")
	_ = json.Indent(&pp, payload, "", "  ")
	lines = append(lines, Line{"header", ph.String()}, Line{"payload", pp.String()})
	return []Section{{"jwt", lines}}
}

// jwtPart decodes one dot-separated part of a token (unpadded base64url,
// though padded input is accepted too).
func jwtPart(p string) ([]byte, bool) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(p, "="))
	return b, err == nil && len(b) > 0
}

func claimTime(claims map[string]any, k string) (time.Time, bool) {
	n, ok := claims[k].(json.Number)
	if !ok {
		return time.Time{}, false
	}
	f, err := n.Float64()
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(int64(f), 0), true
}

// jwtStatus says whether the token is usable now, judging by exp and nbf.
func jwtStatus(claims map[string]any, now time.Time) string {
	if nbf, ok := claimTime(claims, "nbf"); ok && now.Before(nbf) {
		return "not valid yet, starts " + relative(nbf, now)
	}
	exp, ok := claimTime(claims, "exp")
	switch {
	case !ok:
		return "no expiry"
	case now.Before(exp):
		return "valid, expires " + relative(exp, now)
	default:
		return "expired " + relative(exp, now)
	}
}
