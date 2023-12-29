package builder

import (
	"fmt"
	"strings"
)

type Builder struct {
	spacing string // optional spacing [s]
	newline string
	err     error

	quoted            *pattern
	locals            []local
	inputs, selectors []expression
	variants          []variant
	patterns          []any
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

	for _, pattern := range b.patterns {
		s += fmt.Sprint(pattern)
	}

	for i, selector := range b.selectors {
		switch i {
		case 0:
			s += ".match" + b.spacing + selector.String() + " "
		case len(b.selectors) - 1: // newline after the last selector
			s += selector.String() + b.newline
		default:
			s += selector.String() + " "
		}
	}

	for _, variant := range b.variants {
		s += variant.String() + b.newline
	}

	if b.quoted != nil {
		s += b.quoted.String()
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
	if len(b.variants) == 0 && b.quoted == nil &&
		(len(b.inputs) > 0 || len(b.locals) > 0 || len(b.selectors) > 0) {
		return fmt.Errorf("complex message MUST include complex body")
	}

	if len(b.variants) > 0 && b.quoted != nil {
		return fmt.Errorf("complex message MUST have single complex body")
	}

	if len(b.selectors) > 0 && len(b.variants) == 0 {
		return fmt.Errorf("matcher MUST have at least one variant")
	}

	if len(b.variants) > 0 && len(b.selectors) == 0 {
		return fmt.Errorf("matcher MUST have at least one selector")
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

	s = textEscape(s)

	switch b.IsEmpty() {
	case true:
		if simpleStart(s) || s == "" {
			b.patterns = append(b.patterns, s)
		} else {
			b.quoted = &pattern{elements: []any{s}}
		}
	case false:
		if len(b.variants) != 0 {
			b.variants[len(b.variants)-1].pattern = append(b.variants[len(b.variants)-1].pattern, s)
		} else {
			b.patterns = append(b.patterns, s)
		}
	}

	return b
}

type pattern struct {
	elements []any
}

func Pattern() *pattern { return new(pattern) }

func (p *pattern) String() string {
	s := "{{"

	if p != nil {
		for i := range p.elements {
			switch v := p.elements[i].(type) {
			case string:
				s += v
			case *expression:
				s += v.String()
			default:
				panic(fmt.Sprintf("unsupported pattern type: %T", v))
			}
		}
	}

	return s + "}}"
}

func (p *pattern) Text(s string) *pattern {
	p.elements = append(p.elements, textEscape(s))
	return p
}

func (p *pattern) Expr(e *expression) *pattern {
	p.elements = append(p.elements, e)
	return p
}

func (b *Builder) Quoted(pattern *pattern) *Builder {
	if b.err != nil {
		return b
	}

	if len(b.patterns) > 0 {
		b.err = fmt.Errorf("complex message cannot be added after simple message")
		return b
	}

	b.quoted = pattern

	return b
}

func (b *Builder) Local(v string, expr *expression) *Builder {
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

func (b *Builder) Input(expr *expression) *Builder {
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

func (b *Builder) Expr(e *expression) *Builder {
	if b.err != nil {
		return b
	}

	e.spacing = b.spacing

	if len(b.variants) == 0 {
		b.patterns = append(b.patterns, e)
	} else {
		b.variants[len(b.variants)-1].pattern = append(b.variants[len(b.variants)-1].pattern, e)
	}

	return b
}

func (b *Builder) Match(selectors ...*expression) *Builder {
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
	keys, pattern []any
}

func (v *variant) String() string {
	var s string

	for i := range v.keys {
		switch v := v.keys[i].(type) {
		case string:
			s += v + " "
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			s += fmt.Sprintf("%d ", v)
		case float32, float64:
			s += fmt.Sprintf("%f ", v)
		default:
			panic(fmt.Sprintf("unsupported key type: %T", v))
		}
	}

	s += "{{"

	for i := range v.pattern {
		if i != 0 {
			s += " "
		}

		switch v := v.pattern[i].(type) {
		case string:
			s += v
		case *expression:
			s += v.String()
		default:
			panic(fmt.Sprintf("unsupported pattern type: %T", v))
		}
	}

	return s + "}}"
}

type local struct {
	expr     *expression
	variable variable
}

type literal any

type variable string

type function struct {
	name    string
	options []option
}

type expression struct {
	spacing  string
	operand  any // literal or variable
	function function
}

func (e *expression) String() string {
	s := "{" + e.spacing

	switch v := e.operand.(type) {
	case variable:
		s += string(v)
	case nil:
		// noop
	case literal:
		s += printLiteral(v)
	default:
		panic(fmt.Sprintf("unsupported operand type: %T", v))
	}

	if e.function.name != "" {
		if e.operand != nil { // literal or variable
			s += " "
		}

		s += e.function.name

		for _, o := range e.function.options {
			s += " " + o.key + e.spacing + "=" + e.spacing + fmt.Sprint(o.operand)
		}
	}

	return s + e.spacing + "}"
}

func Expr() *expression { return new(expression) }

func (e *expression) Literal(v any) *expression {
	e.operand = v
	return e
}

func (e *expression) Var(v string) *expression {
	if len(v) == 0 {
		panic("variable name cannot be empty")
	}

	if v[0] != '$' {
		panic(fmt.Sprintf("variable must start with $: %s", v))
	}

	e.operand = variable(v)

	return e
}

type option struct { //nolint:govet
	key     string
	operand any // literal or variable
}

func Option(key string, operand any) option { return option{key, operand} }

func (e *expression) Func(name string, option ...option) *expression {
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

func printLiteral(l any) string {
	switch v := l.(type) {
	case string:
		v = quotedEscape(v)
		return "|" + v + "|"
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
	var sb strings.Builder

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
	var sb strings.Builder

	for _, c := range s {
		switch c {
		case '\\', '{', '}':
			sb.WriteRune('\\')
		}

		sb.WriteRune(c)
	}

	return sb.String()
}

// simpleStart returns true if the string has a simple start.
// simple-start = simple-start-char / text-escape / placeholder.
func simpleStart(s string) bool {
	for i, c := range s {
		if i > 0 {
			break
		}

		if isSimpleStart(c) || c == '\\' {
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

func (b Builder) IsEmpty() bool {
	if b.locals == nil && b.inputs == nil && b.selectors == nil &&
		b.variants == nil && b.patterns == nil && b.quoted == nil {
		return true
	}

	return false
}
