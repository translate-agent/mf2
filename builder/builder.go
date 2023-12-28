package builder

import (
	"fmt"
	"strings"
)

type Builder struct {
	spacing string // optional spacing [s]
	newline string
	err     error

	locals            []local
	inputs, selectors []Expression
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

	if len(b.selectors) > 0 {
		s += ".match" + b.spacing
	}

	for i, selector := range b.selectors {
		if i == len(b.selectors)-1 { // newline after the last selector
			s += selector.String() + b.newline
		} else {
			s += selector.String() + " "
		}
	}

	for _, variant := range b.variants {
		s += variant.String() + b.newline
	}

	return s, nil
}

func (b *Builder) String() string {
	s, err := b.Build()
	if err != nil {
		panic(err)
	}

	return s
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

	if b.locals == nil && b.inputs == nil && b.selectors == nil &&
		b.variants == nil && b.patterns == nil { // simple start text
		s = textEscape(s, true)
	} else {
		s = textEscape(s, false)
	}

	if len(b.variants) == 0 {
		b.patterns = append(b.patterns, s)
	} else {
		b.variants[len(b.variants)-1].pattern = append(b.variants[len(b.variants)-1].pattern, s)
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

func (b *Builder) Expr(e *Expression) *Builder {
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
	options []option
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
	case literal:
		s += printLiteral(v)
	default:
		panic(fmt.Sprintf("unsupported operand type: %T", v))
	}

	if e.function.name != "" {
		s += " " + e.function.name

		for _, o := range e.function.options {
			s += " " + o.key + e.spacing + "=" + e.spacing + fmt.Sprint(o.operand)
		}
	}

	return s + e.spacing + "}"
}

func Expr() *Expression { return new(Expression) }

func (e *Expression) Literal(v any) *Expression {
	e.operand = v
	return e
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

type option struct { //nolint:govet
	key     string
	operand any // literal or variable
}

func Option(key string, operand any) option { return option{key, operand} }

func (e *Expression) Func(name string, option ...option) *Expression {
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
		return fmt.Sprintf("|%d|", v)
	case float32, float64:
		return fmt.Sprintf("|%f|", v)
	default:
		panic(fmt.Sprintf("unsupported literal type: %T", v))
	}
}

// helpers

// quotedEscape escapes special characters in quoted name literal.
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

// textEscape escapes special characters in text.
func textEscape(s string, simpleStart bool) string {
	var sb strings.Builder

	for i, c := range s {
		switch c {
		case '\\', '{', '}':
			sb.WriteRune('\\')
		case '.':
			if i == 0 && simpleStart {
				sb.WriteRune('\\')
			}
		}

		sb.WriteRune(c)
	}

	return sb.String()
}
