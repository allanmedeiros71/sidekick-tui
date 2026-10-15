package calc

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"unicode"
)

// number is a value produced by the expression evaluator. It is an exact
// rational, so 0.1 + 0.2 is 0.3 and 10/3 keeps every digit; only a
// fractional power (2 ** 0.5) falls back to floating point. sized records
// whether any operand carried a size unit (KiB, MB...), which makes the
// result worth showing as a human-readable size.
type number struct {
	r     *big.Rat
	sized bool
}

func intNumber(v int64) number { return number{r: new(big.Rat).SetInt64(v)} }

// int64 returns the value if it is a whole number that fits in 64 bits.
func (n number) int64() (int64, bool) {
	if !n.r.IsInt() || !n.r.Num().IsInt64() {
		return 0, false
	}
	return n.r.Num().Int64(), true
}

// float returns the value as a float64, rounding as needed.
func (n number) float() float64 {
	f, _ := n.r.Float64()
	return f
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

// operators, longest first so "**", "//" and "<<" win over "*", "/" and "<".
var operators = []string{"**", "//", "<<", ">>", "+", "-", "*", "/", "%", "&", "|", "^", "~"}

// evalExpr evaluates an arithmetic expression with C-like operators,
// Python-like precedence for ** and unary minus, and literals in decimal
// (with an optional fraction and exponent, as in 6.022e23), 0x, 0b and 0o,
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
		return t.num.r.RatString()
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

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// lexNumber reads a literal starting at s[i] and returns it with the index
// just past it, including any exponent and trailing size unit.
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
	mantissa := strings.ReplaceAll(s[bodyStart:i], "_", "")
	if mantissa == "" || mantissa == "." {
		return number{}, 0, fmt.Errorf("malformed number at %d", start+1)
	}
	if base == 10 && len(mantissa) > 1 && mantissa[0] == '0' && mantissa[1] != '.' {
		return number{}, 0, fmt.Errorf("ambiguous leading zero in %q: write 0o%s for octal", s[start:i], mantissa[1:])
	}

	// Exponent: "e" or "E" followed by digits, optionally signed. A bare
	// "E" is left for the EiB unit.
	exponent := ""
	if base == 10 && i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		if j < len(s) && isDigit(s[j]) {
			for j < len(s) && isDigit(s[j]) {
				j++
			}
			exponent, i = s[i:j], j
		}
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
		if len(exponent) > 6 {
			return number{}, 0, errTooLarge
		}
		if _, ok := r.SetString(mantissa + exponent); !ok {
			return number{}, 0, fmt.Errorf("malformed number %q", mantissa+exponent)
		}
	} else {
		n, ok := new(big.Int).SetString(mantissa, base)
		if !ok {
			return number{}, 0, fmt.Errorf("malformed number %q", s[start:i])
		}
		// Hex, binary and octal literals up to 64 bits are bit patterns:
		// 0xFFFFFFFFFFFFFFFF is -1, as in a two's complement register.
		if n.BitLen() > 64 {
			return number{}, 0, errOverflow
		}
		if n.BitLen() == 64 {
			n.SetInt64(int64(n.Uint64()))
		}
		r.SetInt(n)
	}
	r.Mul(r, new(big.Rat).SetInt64(unit))
	return number{r: r, sized: sized}, i, nil
}

var (
	errOverflow = errors.New("overflow: does not fit in 64 bits")
	errTooLarge = errors.New("number too large")
)

// maxBits bounds the size of exact results, so 9 ** 9 ** 9 fails quickly
// instead of exhausting memory.
const maxBits = 1 << 16

type parser struct {
	toks []token
	i    int
}

func (p *parser) peek() token { return p.toks[p.i] }
func (p *parser) next() token { t := p.toks[p.i]; p.i++; return t }

var precedence = map[string]int{
	"|": 1, "^": 2, "&": 3, "<<": 4, ">>": 4, "+": 5, "-": 5, "*": 6, "/": 6, "//": 6, "%": 6,
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
			n.r = new(big.Rat).Neg(n.r)
		case "~":
			v, ok := n.int64()
			if !ok {
				return number{}, errors.New("~ needs a whole number that fits in 64 bits")
			}
			n.r = new(big.Rat).SetInt64(^v)
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

func apply(op string, a, b number) (number, error) {
	out := number{r: new(big.Rat), sized: a.sized || b.sized}
	switch op {
	case "+":
		out.r.Add(a.r, b.r)
	case "-":
		out.r.Sub(a.r, b.r)
	case "*":
		out.r.Mul(a.r, b.r)
	case "/":
		if b.r.Sign() == 0 {
			return number{}, errors.New("division by zero")
		}
		out.r.Quo(a.r, b.r)
	case "**":
		return power(a, b)
	default: // integer-only operators, on 64-bit values
		x, okx := a.int64()
		y, oky := b.int64()
		if !okx || !oky {
			return number{}, fmt.Errorf("%s needs whole numbers that fit in 64 bits", op)
		}
		v, err := applyInt(op, x, y)
		if err != nil {
			return number{}, err
		}
		out.r.SetInt64(v)
	}
	if out.r.Num().BitLen() > maxBits || out.r.Denom().BitLen() > maxBits {
		return number{}, errTooLarge
	}
	return out, nil
}

func applyInt(op string, x, y int64) (int64, error) {
	switch op {
	case "&":
		return x & y, nil
	case "|":
		return x | y, nil
	case "^":
		return x ^ y, nil
	case "//", "%":
		if y == 0 {
			return 0, errors.New("division by zero")
		}
		if x == math.MinInt64 && y == -1 {
			return 0, errOverflow
		}
		if op == "//" {
			return x / y, nil
		}
		return x % y, nil
	case "<<", ">>":
		if y < 0 || y > 63 {
			return 0, fmt.Errorf("shift count %d out of range 0..63", y)
		}
		// Shifts act on the 64-bit pattern: 1 << 63 is 0x8000000000000000.
		if op == "<<" {
			return x << uint(y), nil
		}
		return x >> uint(y), nil
	}
	return 0, fmt.Errorf("unknown operator %s", op)
}

// power is exact for whole exponents and approximate otherwise.
func power(a, b number) (number, error) {
	out := number{r: new(big.Rat), sized: a.sized || b.sized}
	if e, ok := b.int64(); ok {
		if a.r.Sign() == 0 && e < 0 {
			return number{}, errors.New("division by zero")
		}
		num, den := new(big.Int).Abs(a.r.Num()), a.r.Denom()
		if e != 0 && int64(max(num.BitLen(), den.BitLen())-1)*abs(e) > maxBits {
			return number{}, errTooLarge
		}
		ue := big.NewInt(abs(e))
		n, d := new(big.Int).Exp(num, ue, nil), new(big.Int).Exp(den, ue, nil)
		if a.r.Sign() < 0 && e%2 != 0 {
			n.Neg(n)
		}
		if e < 0 {
			n, d = d, n
			if d.Sign() < 0 {
				n.Neg(n)
				d.Neg(d)
			}
		}
		out.r.SetFrac(n, d)
		return out, nil
	}
	f := math.Pow(a.float(), b.float())
	if math.IsNaN(f) {
		return number{}, errors.New("a negative number has no real fractional power")
	}
	if math.IsInf(f, 0) {
		return number{}, errTooLarge
	}
	out.r.SetFloat64(f)
	return out, nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
