package calc

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"unicode"
)

// number is an integer value produced by the expression evaluator. sized
// records whether any operand carried a size unit (KiB, MB...), which makes
// the result worth showing as a human-readable size.
type number struct {
	v     int64
	sized bool
}

type tokenKind int

const (
	tokNum tokenKind = iota
	tokOp
	tokLParen
	tokRParen
	tokEOF
)

type token struct {
	kind tokenKind
	op   string
	num  number
	pos  int
}

// operators, longest first so "**" and "<<" win over "*" and "<".
var operators = []string{"**", "<<", ">>", "+", "-", "*", "/", "%", "&", "|", "^", "~"}

// evalExpr evaluates an integer expression with C-like operators, Python-like
// precedence for ** and unary minus, and literals in decimal, 0x, 0b and 0o,
// optionally followed by a size unit (4KiB, 1.5 GB).
func evalExpr(s string) (number, error) {
	toks, err := lex(s)
	if err != nil {
		return number{}, err
	}
	p := &parser{toks: toks}
	n, err := p.parseBinary(1)
	if err != nil {
		return number{}, err
	}
	if t := p.peek(); t.kind != tokEOF {
		return number{}, fmt.Errorf("unexpected %q at %d", tokenText(t), t.pos+1)
	}
	return n, nil
}

func tokenText(t token) string {
	switch t.kind {
	case tokOp:
		return t.op
	case tokLParen:
		return "("
	case tokRParen:
		return ")"
	case tokNum:
		return fmt.Sprint(t.num.v)
	}
	return "end of input"
}

func lex(s string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '(':
			toks = append(toks, token{kind: tokLParen, pos: i})
			i++
		case c == ')':
			toks = append(toks, token{kind: tokRParen, pos: i})
			i++
		case c >= '0' && c <= '9' || c == '.':
			n, next, err := lexNumber(s, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{kind: tokNum, num: n, pos: i})
			i = next
		default:
			op := ""
			for _, o := range operators {
				if strings.HasPrefix(s[i:], o) {
					op = o
					break
				}
			}
			if op == "" {
				return nil, fmt.Errorf("unexpected %q at %d", rune(c), i+1)
			}
			toks = append(toks, token{kind: tokOp, op: op, pos: i})
			i += len(op)
		}
	}
	return append(toks, token{kind: tokEOF, pos: len(s)}), nil
}

// lexNumber reads a literal starting at s[i] and returns it with the index
// just past it, including any trailing size unit.
func lexNumber(s string, i int) (number, int, error) {
	start := i
	base, digits := 10, "0123456789_"
	if len(s) > i+1 && s[i] == '0' {
		switch s[i+1] {
		case 'x', 'X':
			base, digits = 16, "0123456789abcdefABCDEF_"
		case 'b', 'B':
			base, digits = 2, "01_"
		case 'o', 'O':
			base, digits = 8, "01234567_"
		}
		if base != 10 {
			i += 2
		}
	}
	bodyStart := i
	for i < len(s) && strings.IndexByte(digits, s[i]) >= 0 {
		i++
	}
	if base == 10 && i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && strings.IndexByte(digits, s[i]) >= 0 {
			i++
		}
	}
	body := strings.ReplaceAll(s[bodyStart:i], "_", "")
	if body == "" || body == "." {
		return number{}, 0, fmt.Errorf("malformed number at %d", start+1)
	}
	if base == 10 && len(body) > 1 && body[0] == '0' && body[1] != '.' {
		return number{}, 0, fmt.Errorf("ambiguous leading zero in %q: write 0o%s for octal", s[start:i], body[1:])
	}

	// Optional size unit, possibly after spaces: "4KiB", "1.5 GB".
	j := i
	for j < len(s) && s[j] == ' ' {
		j++
	}
	k := j
	for k < len(s) && unicode.IsLetter(rune(s[k])) {
		k++
	}
	unit, sized := int64(1), false
	if k > j {
		u, ok := lookupUnit(s[j:k])
		if !ok {
			if s[j:k] == "m" {
				return number{}, 0, errors.New(`"m" is minutes: write M for MiB`)
			}
			return number{}, 0, fmt.Errorf("unknown unit %q", s[j:k])
		}
		unit, sized, i = u, true, k
	}

	r := new(big.Rat)
	if base == 10 {
		if _, ok := r.SetString(body); !ok {
			return number{}, 0, fmt.Errorf("malformed number %q", body)
		}
	} else {
		n, ok := new(big.Int).SetString(body, base)
		if !ok {
			return number{}, 0, fmt.Errorf("malformed number %q", s[start:i])
		}
		// Hex, binary and octal literals up to 64 bits are bit patterns:
		// 0xFFFFFFFFFFFFFFFF is -1, as in a two's complement register.
		if n.BitLen() > 64 {
			return number{}, 0, errOverflow
		}
		r.SetInt(n)
		if !sized && n.BitLen() == 64 {
			return number{v: int64(n.Uint64())}, i, nil
		}
	}
	if !r.IsInt() && !sized {
		return number{}, 0, fmt.Errorf("fractions need a size unit (e.g. 1.5GiB); the calculator works on integers")
	}
	r.Mul(r, new(big.Rat).SetInt64(unit))
	v := new(big.Int).Quo(r.Num(), r.Denom()) // truncate fractional bytes
	if !v.IsInt64() {
		return number{}, 0, errOverflow
	}
	return number{v: v.Int64(), sized: sized}, i, nil
}

var errOverflow = errors.New("overflow: result does not fit in 64 bits")

type parser struct {
	toks []token
	i    int
}

func (p *parser) peek() token { return p.toks[p.i] }
func (p *parser) next() token { t := p.toks[p.i]; p.i++; return t }

var precedence = map[string]int{
	"|": 1, "^": 2, "&": 3, "<<": 4, ">>": 4, "+": 5, "-": 5, "*": 6, "/": 6, "%": 6,
}

// parseBinary implements precedence climbing for left-associative operators.
func (p *parser) parseBinary(minPrec int) (number, error) {
	lhs, err := p.parseUnary()
	if err != nil {
		return number{}, err
	}
	for {
		t := p.peek()
		prec, ok := precedence[t.op]
		if t.kind != tokOp || !ok || prec < minPrec {
			return lhs, nil
		}
		p.next()
		rhs, err := p.parseBinary(prec + 1)
		if err != nil {
			return number{}, err
		}
		if lhs, err = apply(t.op, lhs, rhs); err != nil {
			return number{}, err
		}
	}
}

// parseUnary binds looser than ** so that -2**2 is -4, as in Python.
func (p *parser) parseUnary() (number, error) {
	t := p.peek()
	if t.kind == tokOp && (t.op == "-" || t.op == "+" || t.op == "~") {
		p.next()
		n, err := p.parseUnary()
		if err != nil {
			return number{}, err
		}
		switch t.op {
		case "-":
			if n.v == minInt64 {
				return number{}, errOverflow
			}
			n.v = -n.v
		case "~":
			n.v = ^n.v
		}
		return n, nil
	}
	return p.parsePower()
}

func (p *parser) parsePower() (number, error) {
	base, err := p.parsePrimary()
	if err != nil {
		return number{}, err
	}
	if t := p.peek(); t.kind == tokOp && t.op == "**" {
		p.next()
		exp, err := p.parseUnary() // right-associative
		if err != nil {
			return number{}, err
		}
		return apply("**", base, exp)
	}
	return base, nil
}

func (p *parser) parsePrimary() (number, error) {
	t := p.next()
	switch t.kind {
	case tokNum:
		return t.num, nil
	case tokLParen:
		n, err := p.parseBinary(1)
		if err != nil {
			return number{}, err
		}
		if c := p.next(); c.kind != tokRParen {
			return number{}, fmt.Errorf("missing ) at %d", c.pos+1)
		}
		return n, nil
	case tokEOF:
		return number{}, errors.New("incomplete expression")
	}
	return number{}, fmt.Errorf("unexpected %q at %d", tokenText(t), t.pos+1)
}

const minInt64 = -1 << 63

func apply(op string, a, b number) (number, error) {
	out := number{sized: a.sized || b.sized}
	x, y := a.v, b.v
	switch op {
	case "&":
		out.v = x & y
	case "|":
		out.v = x | y
	case "^":
		out.v = x ^ y
	case "/", "%":
		if y == 0 {
			return number{}, errors.New("division by zero")
		}
		if x == minInt64 && y == -1 {
			return number{}, errOverflow
		}
		if op == "/" {
			out.v = x / y
		} else {
			out.v = x % y
		}
	case "<<", ">>":
		if y < 0 || y > 63 {
			return number{}, fmt.Errorf("shift count %d out of range 0..63", y)
		}
		// Shifts act on the 64-bit pattern: 1 << 63 is 0x8000000000000000.
		if op == "<<" {
			out.v = x << uint(y)
		} else {
			out.v = x >> uint(y)
		}
	default: // + - * **: computed exactly, then checked for overflow.
		bx, by := big.NewInt(x), big.NewInt(y)
		r := new(big.Int)
		switch op {
		case "+":
			r.Add(bx, by)
		case "-":
			r.Sub(bx, by)
		case "*":
			r.Mul(bx, by)
		case "**":
			if y < 0 {
				return number{}, errors.New("negative exponent")
			}
			if y > 64 && x != 0 && x != 1 && x != -1 {
				return number{}, errOverflow
			}
			r.Exp(bx, by, nil)
		}
		if !r.IsInt64() {
			return number{}, errOverflow
		}
		out.v = r.Int64()
	}
	return out, nil
}
