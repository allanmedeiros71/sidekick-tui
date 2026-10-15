package inspect

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

type uuid [16]byte

var uuidText = regexp.MustCompile(`^(?i:urn:uuid:)?\{?([0-9A-Fa-f]{8})-([0-9A-Fa-f]{4})-([0-9A-Fa-f]{4})-([0-9A-Fa-f]{4})-([0-9A-Fa-f]{12})\}?$`)

func parseUUID(s string) (uuid, bool) {
	var u uuid
	m := uuidText.FindStringSubmatch(s)
	if m == nil || strings.HasPrefix(s, "{") != strings.HasSuffix(s, "}") {
		return u, false
	}
	_, err := hex.Decode(u[:], []byte(strings.Join(m[1:], "")))
	return u, err == nil
}

func (u uuid) String() string {
	h := hex.EncodeToString(u[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func (u uuid) version() int { return int(u[6] >> 4) }

var versionNames = map[int]string{
	1: "time-based",
	2: "DCE security",
	3: "name-based, MD5",
	4: "random",
	5: "name-based, SHA-1",
	6: "time-based, reordered",
	7: "Unix time-based",
	8: "custom",
}

func (u uuid) variant() string {
	switch {
	case u[8]&0x80 == 0:
		return "NCS (reserved)"
	case u[8]&0xc0 == 0x80:
		return "RFC 9562"
	case u[8]&0xe0 == 0xc0:
		return "Microsoft (reserved)"
	default:
		return "future (reserved)"
	}
}

// gregorianOffset is the number of 100 ns intervals between the UUID epoch
// (1582-10-15) and the Unix epoch.
const gregorianOffset = 0x01B21DD213814000

// timestamp returns the time embedded in version 1, 6 and 7 UUIDs.
func (u uuid) timestamp() (time.Time, bool) {
	var ticks uint64
	switch u.version() {
	case 1:
		ticks = uint64(binary.BigEndian.Uint16(u[6:8])&0x0fff)<<48 |
			uint64(binary.BigEndian.Uint16(u[4:6]))<<32 |
			uint64(binary.BigEndian.Uint32(u[0:4]))
	case 6:
		ticks = uint64(binary.BigEndian.Uint32(u[0:4]))<<28 |
			uint64(binary.BigEndian.Uint16(u[4:6]))<<12 |
			uint64(binary.BigEndian.Uint16(u[6:8])&0x0fff)
	case 7:
		ms := uint64(u[0])<<40 | uint64(u[1])<<32 | uint64(binary.BigEndian.Uint32(u[2:6]))
		return time.UnixMilli(int64(ms)), true
	default:
		return time.Time{}, false
	}
	return time.Unix(0, int64(ticks-gregorianOffset)*100), true
}

func uuidSections(s string, now time.Time) []Section {
	u, ok := parseUUID(s)
	if !ok {
		return nil
	}
	lines := []Line{{"uuid", u.String()}}
	switch u {
	case uuid{}:
		lines = append(lines, Line{"version", "nil UUID"})
	case uuid{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}:
		lines = append(lines, Line{"version", "max UUID"})
	default:
		v := u.version()
		name := versionNames[v]
		if name == "" {
			name = "unknown"
		}
		lines = append(lines, Line{"version", fmt.Sprintf("%d (%s)", v, name)}, Line{"variant", u.variant()})
		if t, ok := u.timestamp(); ok {
			lines = append(lines, Line{"time", formatTime(t, now)})
		}
		if v == 1 || v == 6 {
			lines = append(lines, Line{"node", net.HardwareAddr(u[10:16]).String()})
		}
	}
	lines = append(lines,
		Line{"upper", strings.ToUpper(u.String())},
		Line{"hex", hex.EncodeToString(u[:])},
		Line{"base64", base64.StdEncoding.EncodeToString(u[:])},
	)
	return []Section{{"uuid", lines}}
}

// newUUID returns a random version 4 UUID, or a version 7 UUID carrying
// now's Unix time in milliseconds.
func newUUID(version int, now time.Time) uuid {
	var u uuid
	_, _ = rand.Read(u[:])
	if version == 7 {
		ms := uint64(now.UnixMilli())
		u[0], u[1] = byte(ms>>40), byte(ms>>32)
		binary.BigEndian.PutUint32(u[2:6], uint32(ms))
	}
	u[6] = u[6]&0x0f | byte(version<<4)
	u[8] = u[8]&0x3f | 0x80
	return u
}

func generateUUIDs(now time.Time) Result {
	return Result{Kind: "uuid", Sections: []Section{{"new uuid", []Line{
		{"v4", newUUID(4, now).String()},
		{"v7", newUUID(7, now).String()},
		{"v4 upper", strings.ToUpper(newUUID(4, now).String())},
	}}}}
}
