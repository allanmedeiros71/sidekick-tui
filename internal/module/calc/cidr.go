package calc

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"net/netip"
	"strconv"
	"strings"
)

func cidrResult(p netip.Prefix) Result {
	if p.Addr().Is4() {
		return cidr4(p)
	}
	return cidr6(p)
}

func cidr4(p netip.Prefix) Result {
	bits := p.Bits()
	addr := u32(p.Addr())
	mask := uint32(0)
	if bits > 0 {
		mask = ^uint32(0) << (32 - bits)
	}
	network, broadcast := addr&mask, addr|^mask
	total := uint64(1) << (32 - bits)

	first, last, hosts := network+1, broadcast-1, total-2
	switch bits {
	case 32:
		first, last, hosts = network, network, 1
	case 31: // point-to-point link, RFC 3021
		first, last, hosts = network, broadcast, 2
	}

	lines := []Line{
		{"network", fmt.Sprintf("%s/%d", ip4(network), bits)},
		{"netmask", ip4(mask)},
		{"wildcard", ip4(^mask)},
		{"broadcast", ip4(broadcast)},
		{"first host", ip4(first)},
		{"last host", ip4(last)},
		{"hosts", strconv.FormatUint(hosts, 10)},
		{"addresses", strconv.FormatUint(total, 10)},
	}
	if addr != network {
		lines = append(lines, Line{"note", fmt.Sprintf("host bits set: %s is inside %s/%d", p.Addr(), ip4(network), bits)})
	}
	return Result{Kind: "cidr", Lines: lines}
}

func cidr6(p netip.Prefix) Result {
	bits := p.Bits()
	network := p.Masked()
	a := network.Addr().As16()
	for i := bits; i < 128; i++ {
		a[i/8] |= 1 << (7 - i%8)
	}
	last := netip.AddrFrom16(a)
	count := new(big.Int).Lsh(big.NewInt(1), uint(128-bits))
	lines := []Line{
		{"network", network.String()},
		{"first", network.Addr().String()},
		{"last", last.String()},
		{"addresses", fmt.Sprintf("2^%d (%s)", 128-bits, count)},
	}
	if p.Addr() != network.Addr() {
		lines = append(lines, Line{"note", fmt.Sprintf("host bits set: %s is inside %s", p.Addr(), network)})
	}
	return Result{Kind: "cidr", Lines: lines}
}

func ipResult(a netip.Addr) Result {
	var lines []Line
	if a.Is4() {
		n := u32(a)
		b := a.As4()
		octets := make([]string, 4)
		for i, o := range b {
			octets[i] = fmt.Sprintf("%08b", o)
		}
		lines = []Line{
			{"decimal", strconv.FormatUint(uint64(n), 10)},
			{"hex", fmt.Sprintf("0x%08X", n)},
			{"binary", strings.Join(octets, ".")},
		}
	} else {
		lines = []Line{{"expanded", a.StringExpanded()}}
	}
	return Result{Kind: "ip", Lines: append(lines, Line{"scope", scope(a)})}
}

func scope(a netip.Addr) string {
	switch {
	case a.IsLoopback():
		return "loopback"
	case a.IsPrivate():
		return "private"
	case a.IsLinkLocalUnicast():
		return "link-local"
	case a.IsMulticast():
		return "multicast"
	case a.IsUnspecified():
		return "unspecified"
	case a.IsGlobalUnicast():
		return "global"
	}
	return "other"
}

func u32(a netip.Addr) uint32 {
	b := a.As4()
	return binary.BigEndian.Uint32(b[:])
}

func ip4(n uint32) string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], n)
	return netip.AddrFrom4(b).String()
}
