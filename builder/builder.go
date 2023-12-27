package builder

import (
	"fmt"
)

type Builder struct {
	newline, spacing string
	locals           []local
	inputs           []Expression
	patterns         []any
}

func New() *Builder {
	return &Builder{
		newline: "\n",
		spacing: " ",
	}
}

func (b *Builder) Build() (string, error) {
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
	b.newline = s
	return b
}

func (b *Builder) Spacing(s string) *Builder {
	b.spacing = s
	return b
}

func (b *Builder) Text(s string) *Builder {
	b.patterns = append(b.patterns, s)

	return b
}

func (b *Builder) Local(v string, expr *Expression) *Builder {
	expr.spacing = b.spacing

	b.locals = append(b.locals, local{variable: variable(v), expr: expr})

	return b
}

func (b *Builder) Input(expr *Expression) *Builder {
	expr.spacing = b.spacing

	b.inputs = append(b.inputs, *expr)

	return b
}

func (b *Builder) Expr(e *Expression) *Builder {
	e.spacing = b.spacing
	b.patterns = append(b.patterns, e)

	return b
}

func (b *Builder) Match(selectors ...Expression) *Builder {
	return b
}

type local struct {
	variable variable
	expr     *Expression
}

type input struct {
	variable variable
	expr     Expression
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

type option struct {
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
