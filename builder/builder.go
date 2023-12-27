package builder

import (
	"fmt"
	"strconv"
)

type Builder struct {
	newline, spacing string
	err              error

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
		switch i {
		case len(b.selectors) - 1:
			s += selector.String()
		default:
			s += selector.String() + b.spacing
		}
	}

	if len(b.selectors) > 0 {
		s += b.newline
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

	expr.spacing = b.spacing

	b.locals = append(b.locals, local{variable: variable(v), expr: expr})

	return b
}

func (b *Builder) Input(expr *Expression) *Builder {
	if b.err != nil {
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

	if len(b.variants) > 0 {
		b.variants[len(b.variants)-1].pattern = append(b.variants[len(b.variants)-1].pattern, e)
	} else {
		b.patterns = append(b.patterns, e)
	}

	return b
}

func (b *Builder) Match(selectors ...*Expression) *Builder {
	if b.err != nil {
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
		switch k := v.keys[i].(type) {
		case int:
			s += strconv.Itoa(k)
		case string:
			s += k
		}

		s += " "
	}

	s += "{{"

	for i := range v.pattern {
		if i != 0 {
			s += " "
		}

		switch p := v.pattern[i].(type) {
		case string:
			s += p
		case *Expression:
			s += p.String()
		}
	}

	return s + "}}"
}

type local struct {
	expr     *Expression
	variable variable
}

type literal string

type variable string

type function struct {
	name    string
	options []option
}

type Expression struct {
	spacing   string
	operand   any // literal or variable
	functions []function
}

func (e *Expression) String() string {
	s := "{" + e.spacing

	switch v := e.operand.(type) {
	case variable:
		s += string(v)
	case literal:
		s += printLiteral(v)
	}

	for _, f := range e.functions {
		s += " " + f.name

		for _, o := range f.options {
			s += " " + o.key + e.spacing + "=" + e.spacing + fmt.Sprint(o.operand)
		}
	}

	return s + e.spacing + "}"
}

func Expr() *Expression { return new(Expression) }

func (e *Expression) Literal(v string) *Expression {
	e.operand = literal(v)
	return e
}

func (e *Expression) Var(v string) *Expression {
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
	switch name[0] {
	default:
		panic(fmt.Sprintf("function MUST start with :, + or -: %s", name))
	case ':', '+', '-':
		e.functions = append(e.functions, function{name: name, options: option})
		return e
	}
}

// TODO: escape characters according to MF2.
func printLiteral(s literal) string {
	return "|" + string(s) + "|"
}
