package builder

import (
	"fmt"
	"strings"
)

type Builder struct {
	spacing string // optional spacing [s]
	newline string
	err     error

	locals    []local      // local declarations
	inputs    []Expression // input declarations
	selectors []Expression // matcher selectors
	variants  []variant    // matcher variants
	patterns  []any        // string or expression
}

func New() *Builder {
	return &Builder{
		newline: "\n",
		spacing: " ",
	}
}

func (b *Builder) Build() (string, error) {
	if b.err != nil {
		return "", b.err
	}

	if err := b.validate(); err != nil {
		return "", err
	}

	var s string

	for _, v := range b.inputs {
		s += ".input" + b.spacing + v.String() + b.newline
	}

	for _, v := range b.locals {
		s += ".local" + b.spacing + string(v.variable) + b.spacing + "=" + b.spacing + v.expr.String() + b.newline
	}

	quotedPattern := (len(b.inputs) > 0 || len(b.locals) > 0) && (len(b.variants) == 0 && len(b.selectors) == 0)

	if len(b.patterns) > 0 {
		if v, ok := b.patterns[0].(string); ok && !hasSimpleStart(v) {
			switch {
			case len(b.patterns) == 1 && v == "": // simple message with empty text
				// noop
			case len(v) > 0 && []rune(v)[0] == '.': // complex message
				quotedPattern = true
			default:
				return "", fmt.Errorf("simple message MUST start with a simple start character: %s", v)
			}
		}
	}

	if quotedPattern {
		s += "{{"
	}

	for _, v := range b.patterns {
		switch v := v.(type) {
		case string:
			s += textEscape(v)
		case *Expression:
			s += v.String()
		default:
			return "", fmt.Errorf("unsupported pattern type: %T", v)
		}
	}

	if quotedPattern {
		s += "}}"
	}

	for i, v := range b.selectors {
		switch i {
		case 0:
			s += ".match" + b.spacing + v.String() + " "
		case len(b.selectors) - 1: // newline after the last selector
			s += v.String() + b.newline
		default:
			s += v.String() + " "
		}
	}

	for i, v := range b.variants {
		s += v.String()

		if i != len(b.variants)-1 {
			s += b.newline
		}
	}

	return s, nil
}

func (b *Builder) MustBuild() string {
	s, err := b.Build()
	if err != nil {
		panic(err)
	}

	return s
}

func (b *Builder) validate() error {
	if len(b.variants) > 0 {
		if len(b.patterns) > 0 {
			return fmt.Errorf("complex message MUST have single complex body")
		}

		if len(b.selectors) == 0 {
			return fmt.Errorf("matcher MUST have at least one selector")
		}

		if !hasCatchAllVariant(b.variants) {
			return fmt.Errorf("matcher MUST have at least one variant with all catch-all keys")
		}
	}

	if len(b.selectors) > 0 && len(b.variants) == 0 {
		return fmt.Errorf("matcher MUST have at least one variant")
	}

	return nil
}

func (b *Builder) Newline(s string) *Builder {
	if b.err != nil {
		return b
	}

	b.newline = s

	return b
}

func (b *Builder) Spacing(s string) *Builder {
	if b.err != nil {
		return b
	}

	b.spacing = s

	return b
}

func (b *Builder) Text(s string) *Builder {
	if b.err != nil {
		return b
	}

	if len(b.variants) > 0 {
		b.variants[len(b.variants)-1].pattern = append(b.variants[len(b.variants)-1].pattern, s)
	} else {
		b.patterns = append(b.patterns, s)
	}

	return b
}

func (b *Builder) Local(v string, expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	if len(b.patterns) > 0 {
		b.err = fmt.Errorf("complex message cannot be added after simple message")
		return b
	}

	expr.spacing = b.spacing

	b.locals = append(b.locals, local{variable: variable(v), expr: expr})

	return b
}

func (b *Builder) Input(expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	if len(b.patterns) > 0 {
		b.err = fmt.Errorf("complex message cannot be added after simple message")
		return b
	}

	expr.spacing = b.spacing

	b.inputs = append(b.inputs, *expr)

	return b
}

func Expr() *Expression { return new(Expression) }

func (b *Builder) Expr(e *Expression) *Builder {
	if b.err != nil {
		return b
	}

	e.spacing = b.spacing

	if len(b.variants) > 0 {
		b.variants[len(b.variants)-1].pattern = append(b.variants[len(b.variants)-1].pattern, e)
		return b
	}

	b.patterns = append(b.patterns, e)

	return b
}

func (b *Builder) Match(selectors ...*Expression) *Builder {
	if b.err != nil {
		return b
	}

	if len(b.patterns) > 0 {
		b.err = fmt.Errorf("complex message cannot be added after simple message")
		return b
	}

	if len(selectors) == 0 {
		b.err = fmt.Errorf("match MUST have at least one selector")
		return b
	}

	for i := range selectors {
		b.selectors = append(b.selectors, *selectors[i])
	}

	return b
}

func (b *Builder) Key(keys ...any) *Builder {
	if b.err != nil {
		return b
	}

	if len(keys) != len(b.selectors) {
		b.err = fmt.Errorf("number of keys in each variant MUST match the number of selectors in the matcher")
		return b
	}

	b.variants = append(b.variants, variant{keys: keys})

	return b
}

type variant struct {
	keys    []any
	pattern []any
}

func (v *variant) String() string {
	var s string

	for i := range v.keys {
		s += printLiteral(v.keys[i], false) + " "
	}

	s += "{{"

	for i := range v.pattern {
		if i > 0 {
			s += " "
		}

		switch v := v.pattern[i].(type) {
		case string:
			s += textEscape(v)
		case *Expression:
			s += v.String()
		default:
			panic(fmt.Sprintf("unsupported pattern type: %T", v))
		}
	}

	return s + "}}"
}

type local struct {
	expr     *Expression
	variable variable
}

type literal any

type variable string

type function struct {
	name    string
	options []FuncOption
}

type Expression struct {
	spacing  string
	operand  any // literal or variable
	function function
}

func (e *Expression) String() string {
	s := "{" + e.spacing

	switch v := e.operand.(type) {
	case variable:
		s += string(v)
	case nil:
		// noop
	case literal:
		s += printLiteral(v, true)
	default:
		panic(fmt.Sprintf("unsupported operand type: %T", v))
	}

	if e.function.name != "" {
		if e.operand != nil { // literal or variable
			s += " "
		}

		s += e.function.name

		for _, o := range e.function.options {
			s += " " + o.key + e.spacing + "=" + e.spacing

			if v, ok := o.operand.(string); ok && len(v) > 0 && v[0] == '$' { // recognized as variable if string starts with $
				s += v
				continue
			}

			s += printLiteral(o.operand, true)
		}
	}

	return s + e.spacing + "}"
}

func Literal(v any) *Expression {
	return Expr().Literal(v)
}

func (e *Expression) Literal(v any) *Expression {
	e.operand = v
	return e
}

func Var(s string) *Expression {
	return Expr().Var(s)
}

func (e *Expression) Var(v string) *Expression {
	if len(v) == 0 {
		panic("variable name cannot be empty")
	}

	if v[0] != '$' {
		panic(fmt.Sprintf("variable must start with $: %s", v))
	}

	e.operand = variable(v)

	return e
}

type FuncOption struct { //nolint:govet
	key     string
	operand any // literal or variable
}

func Option(key string, operand any) FuncOption { return FuncOption{key, operand} }

func Func(name string, option ...FuncOption) *Expression {
	return Expr().Func(name, option...)
}

func (e *Expression) Func(name string, option ...FuncOption) *Expression {
	if len(name) == 0 {
		panic("function name cannot be empty")
	}

	if e.function.name != "" {
		panic("expression already has a function")
	}

	switch name[0] {
	default:
		panic(fmt.Sprintf("function MUST start with :, + or -: %s", name))
	case ':', '+', '-':
		e.function = function{name: name, options: option}
		return e
	}
}

func printLiteral(l any, quoted bool) string {
	switch v := l.(type) {
	case string:
		if quoted {
			return "|" + quotedEscape(v) + "|"
		}

		return quotedEscape(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	default:
		panic(fmt.Sprintf("unsupported literal type: %T", v))
	}
}

// helpers

/*
	quotedEscape escapes special characters in quoted name literal.

ABNF:
quoted-escape   = backslash ( backslash / "|" )
.
*/
func quotedEscape(s string) string {
	if s == "" {
		return s
	}

	var sb strings.Builder

	sb.Grow(len(s))

	for _, c := range s {
		switch c {
		case '\\', '|':
			sb.WriteRune('\\')
		}

		sb.WriteRune(c)
	}

	return sb.String()
}

/*
	textEscape escapes special characters in text.

ABNF:
text-escape     = backslash ( backslash / "{" / "}" )
.
*/
func textEscape(s string) string {
	if s == "" {
		return s
	}

	var sb strings.Builder

	sb.Grow(len(s))

	for _, c := range s {
		switch c {
		case '\\', '{', '}':
			sb.WriteRune('\\')
		}

		sb.WriteRune(c)
	}

	return sb.String()
}

// hasSimpleStart returns true if the string has a simple start.
// ABNF:
// simple-start = simple-start-char / text-escape / placeholder
// .
func hasSimpleStart(s string) bool {
	if len(s) > 0 {
		c := []rune(s)[0]

		if isSimpleStart(c) ||
			c == '{' || c == '}' || c == '\\' { // text-escape     = backslash ( backslash / "{" / "}" )
			return true
		}
	}

	return false
}

// isSimpleStart returns true if r is simple start character.
//
// ABNF:
//
//	simple-start-char = %x0-2D         ; omit .
//	                  / %x2F-5B        ; omit \
//	                  / %x5D-7A        ; omit {
//	                  / %x7C           ; omit }
//	                  / %x7E-D7FF      ; omit surrogates
//	                  / %xE000-10FFFF
func isSimpleStart(r rune) bool {
	return 0x0 <= r && r <= 0x2D || // omit .
		0x2F <= r && r <= 0x5B || // omit \
		0x5D <= r && r <= 0x7A || // omit {}
		r == 0x7C || // omit }
		0x7E <= r && r <= 0xD7FF || // omit surrogates
		0xE000 <= r && r <= 0x10FFFF
}

// hasCatchAllVariant() checks if at least variant has catch-all keys.
func hasCatchAllVariant(variants []variant) bool {
	for _, v := range variants {
		var catchAllCount int

		for _, key := range v.keys {
			if key == "*" {
				catchAllCount++
			}

			if catchAllCount == len(v.keys) {
				return true
			}
		}
	}

	return false
}
