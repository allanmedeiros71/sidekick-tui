package calc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// symbolicMode matches ls -l style permissions, with an optional file type.
var symbolicMode = regexp.MustCompile(`^[-dlbcps]?[r-][w-][xsS-][r-][w-][xsS-][r-][w-][xtT-]$`)

// parseMode reads an octal mode ("755", "0644", "4755") or a symbolic one
// ("rwxr-xr-x", "-rw-r--r--") into permission bits.
func parseMode(s string) (uint32, error) {
	if symbolicMode.MatchString(s) {
		if len(s) == 10 {
			s = s[1:]
		}
		var m uint32
		for i, c := range s {
			bit := uint32(1) << (8 - i)
			switch c {
			case 'r', 'w', 'x':
				m |= bit
			case 's', 't':
				m |= bit | specialBit(i)
			case 'S', 'T':
				m |= specialBit(i)
			}
		}
		return m, nil
	}
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0o"), "0")
	if s == "" {
		s = "0"
	}
	if len(s) > 4 {
		return 0, fmt.Errorf("mode %q has too many digits", s)
	}
	m, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid mode %q: use octal digits 0-7 or rwxr-xr-x", s)
	}
	return uint32(m), nil
}

// specialBit returns the setuid, setgid or sticky bit for the execute
// position i (2, 5 or 8) of a symbolic mode.
func specialBit(i int) uint32 {
	switch i {
	case 2:
		return 0o4000
	case 5:
		return 0o2000
	}
	return 0o1000
}

func symbolic(m uint32) string {
	b := []byte("---------")
	for i := 0; i < 9; i++ {
		if m&(1<<(8-i)) != 0 {
			b[i] = "rwx"[i%3]
		}
	}
	setSpecial := func(i int, bit uint32, lower, upper byte) {
		if m&bit == 0 {
			return
		}
		if b[i] == 'x' {
			b[i] = lower
		} else {
			b[i] = upper
		}
	}
	setSpecial(2, 0o4000, 's', 'S')
	setSpecial(5, 0o2000, 's', 'S')
	setSpecial(8, 0o1000, 't', 'T')
	return string(b)
}

func describe(bits uint32) string {
	var parts []string
	for i, name := range []string{"read", "write", "execute"} {
		if bits&(4>>i) != 0 {
			parts = append(parts, name)
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func chmodResult(s string) (Result, error) {
	m, err := parseMode(s)
	if err != nil {
		return Result{}, err
	}
	octal := fmt.Sprintf("%04o", m)
	lines := []Line{
		{"octal", octal},
		{"symbolic", symbolic(m)},
		{"owner", describe(m >> 6 & 7)},
		{"group", describe(m >> 3 & 7)},
		{"other", describe(m & 7)},
	}
	var special []string
	for _, sp := range []struct {
		bit  uint32
		name string
	}{{0o4000, "setuid"}, {0o2000, "setgid"}, {0o1000, "sticky"}} {
		if m&sp.bit != 0 {
			special = append(special, sp.name)
		}
	}
	if len(special) > 0 {
		lines = append(lines, Line{"special", strings.Join(special, ", ")})
	}
	lines = append(lines, Line{"command", "chmod " + strings.TrimPrefix(octal, "0")})
	return Result{Kind: "chmod", Lines: lines}, nil
}
