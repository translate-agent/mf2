package builder

import (
	"errors"
	"fmt"
	"strings"

	"go.expect.digital/mf2/parse"
)

type Builder struct {
	tree parse.AST
	err  error
}

func NewBuilder() *Builder {
	return new(Builder)
}

func (b *Builder) Build() (string, error) {
	if b.err != nil {
		return "", b.err
	}

	return b.tree.String(), nil
}

func (b *Builder) MustBuild() string {
	s, err := b.Build()
	if err != nil {
		panic(err)
	}

	return s
}

func (b *Builder) Text(s string) *Builder {
	if b.err != nil {
		return b
	}

	txt := parse.Text(s)

	switch msg := b.tree.Message.(type) {
	default:
		b.tree.Message = parse.SimpleMessage{txt}

		if strings.HasPrefix(s, ".") {
			b.tree.Message = parse.ComplexMessage{ComplexBody: parse.QuotedPattern{txt}}
		}
	case parse.SimpleMessage:
		msg = append(msg, parse.Text(s))
		b.tree.Message = msg
	case parse.ComplexMessage:
		switch v := msg.ComplexBody.(type) {
		case parse.QuotedPattern:
			v = append(v, txt)
			msg.ComplexBody = v
			b.tree.Message = msg
		case parse.Matcher:
			i := len(v.Variants) - 1
			if i < 0 {
				b.err = fmt.Errorf(`add text "%s" to "%s"`, txt, v)
				return b
			}

			v.Variants[i].QuotedPattern = append(v.Variants[i].QuotedPattern, parse.Text(s))
			msg.ComplexBody = v
		}

		b.tree.Message = msg
	}

	return b
}

// Local adds local declaration to the builder.
func (b *Builder) Local(v string, expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	local := parse.LocalDeclaration{
		Variable:   parse.Variable(v),
		Expression: expr.expression,
	}

	switch msg := b.tree.Message.(type) {
	default:
		b.tree.Message = parse.ComplexMessage{
			Declarations: []parse.Declaration{local},
			ComplexBody:  parse.QuotedPattern{},
		}
	case parse.ComplexMessage:
		msg.Declarations = append(msg.Declarations, local)
		b.tree.Message = msg
	case parse.SimpleMessage:
		b.tree.Message = parse.ComplexMessage{
			Declarations: []parse.Declaration{local},
			ComplexBody:  parse.QuotedPattern(msg),
		}
	}

	return b
}

// Input adds input declaration to the builder.
func (b *Builder) Input(expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	switch msg := b.tree.Message.(type) {
	default:
		b.tree.Message = parse.ComplexMessage{
			Declarations: []parse.Declaration{parse.InputDeclaration(expr.expression)},
		}
	case parse.SimpleMessage:
		b.tree.Message = parse.ComplexMessage{
			Declarations: []parse.Declaration{parse.InputDeclaration(expr.expression)},
			ComplexBody:  parse.QuotedPattern(msg),
		}
	case parse.ComplexMessage:
		msg.Declarations = append(msg.Declarations, parse.InputDeclaration(expr.expression))
		b.tree.Message = msg
	}

	return b
}

// Reserved adds reserved statement to the builder.
func (b *Builder) Reserved(
	keyword string,
	expression *Expression,
	reservedOrExpression ...ReservedOrExpression,
) *Builder {
	if b.err != nil {
		return b
	}

	reserved := parse.ReservedStatement{
		Keyword:     keyword,
		Expressions: []parse.Expression{expression.expression},
	}

	for _, v := range reservedOrExpression {
		switch v := v.(type) {
		case ReservedBody:
			switch x := v.(type) {
			case QuotedLiteral:
				reserved.ReservedBody = append(reserved.ReservedBody, parse.QuotedLiteral(x))
			case ReservedText:
				reserved.ReservedBody = append(reserved.ReservedBody, parse.ReservedText(x))
			}
		case *Expression:
			reserved.Expressions = append(reserved.Expressions, v.expression)
		}
	}

	switch msg := b.tree.Message.(type) {
	case parse.SimpleMessage:
	case parse.ComplexMessage:
		msg.Declarations = append(msg.Declarations, reserved)
		b.tree.Message = msg
	}

	return b
}

type ReservedOrExpression interface{ reservedOrExpression() }

func Expr() *Expression { return new(Expression) }

func (b *Builder) Expr(expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	switch msg := b.tree.Message.(type) {
	default:
		b.tree.Message = parse.SimpleMessage{expr.expression}
	case parse.SimpleMessage:
		msg = append(msg, expr.expression)
		b.tree.Message = msg
	case parse.ComplexMessage:
		switch body := msg.ComplexBody.(type) {
		default:
			b.tree.Message = parse.ComplexMessage{
				ComplexBody: parse.QuotedPattern{expr.expression},
			}
		case parse.QuotedPattern:
			body = append(body, expr.expression)
			msg.ComplexBody = body
			b.tree.Message = msg
		case parse.Matcher:
			if len(body.Variants) == 0 {
				b.err = fmt.Errorf(`add expression "%s"`, expr.expression)
				return b
			}

			i := len(body.Variants) - 1
			body.Variants[i].QuotedPattern = append(body.Variants[i].QuotedPattern, expr.expression)
			msg.ComplexBody = body
			b.tree.Message = msg
		}
	}

	return b
}

func (b *Builder) Match(selector *Expression, selectors ...*Expression) *Builder {
	if b.err != nil {
		return b
	}

	parseSelectors := make([]parse.Expression, 0, len(selectors)+1)

	parseSelectors = append(parseSelectors, selector.expression)
	for _, v := range selectors {
		parseSelectors = append(parseSelectors, v.expression)
	}

	switch msg := b.tree.Message.(type) {
	default:
		b.tree.Message = parse.ComplexMessage{
			ComplexBody: parse.Matcher{
				Selectors: parseSelectors,
			},
		}
	case parse.SimpleMessage:
		b.err = errors.New("match cannot be added after simple message")
		return b
	case parse.ComplexMessage:
		switch body := msg.ComplexBody.(type) {
		default:
			msg.ComplexBody = parse.Matcher{
				Selectors: parseSelectors,
			}
			b.tree.Message = msg
		case parse.QuotedPattern:
			b.err = errors.New("match cannot be added after quoted pattern message")
			return b
		case parse.Matcher:
			body.Selectors = parseSelectors
			msg.ComplexBody = body
			b.tree.Message = msg
		}
	}

	return b
}

func (b *Builder) Keys(key any, keys ...any) *Builder {
	if b.err != nil {
		return b
	}

	switch msg := b.tree.Message.(type) {
	default:
		b.err = fmt.Errorf(`add keys to "%s"`, msg)
	case parse.ComplexMessage:
		switch body := msg.ComplexBody.(type) {
		default:
			b.err = fmt.Errorf(`add selectors to "%s"`, msg)
		case parse.Matcher:
			n := len(keys) + 1

			if len(body.Selectors) != n {
				b.err = errors.New("number of keys in each variant MUST match the number of selectors in the matcher")
				return b
			}

			all := make([]parse.VariantKey, 0, n)

			for _, k := range append([]any{key}, keys...) {
				var v parse.VariantKey = toLiteral(k)
				if k == "*" {
					v = parse.CatchAllKey{}
				}

				all = append(all, v)
			}

			body.Variants = append(body.Variants, parse.Variant{Keys: all})
			msg.ComplexBody = body
			b.tree.Message = msg
		}
	}

	return b
}

type Expression struct {
	expression parse.Expression
}

func (e *Expression) reservedOrExpression() {}

func Literal(v any) *Expression {
	return Expr().Literal(v)
}

func (e *Expression) Literal(v any) *Expression {
	e.expression.Operand = toLiteral(v)

	return e
}

func Var(name string) *Expression {
	return Expr().Var(name)
}

func (e *Expression) Var(name string) *Expression {
	if len(name) == 0 {
		panic("variable name cannot be empty")
	}

	e.expression.Operand = parse.Variable(name)

	return e
}

// Hack: limit to only options and attributes, instead of any.
type OptsAndAttr interface{ optsAndAttr() }

func parseIdentifier(s string) (parse.Identifier, error) {
	parts := strings.Split(s, ":")
	switch len(parts) {
	default:
		return parse.Identifier{}, fmt.Errorf(`want identifier with optional namespace, got "%s"`, s)
	case 1:
		return parse.Identifier{Name: parts[0]}, nil
	case 2: //nolint:mnd
		return parse.Identifier{Namespace: parts[0], Name: parts[1]}, nil
	}
}

func (b *Builder) OpenMarkup(name string, optionsAndAttributes ...OptsAndAttr) *Builder {
	return b.markup(parse.Open, name, optionsAndAttributes)
}

func (b *Builder) CloseMarkup(name string, attributes ...attribute) *Builder {
	optsAndAttr := make([]OptsAndAttr, 0, len(attributes))
	for _, v := range attributes {
		optsAndAttr = append(optsAndAttr, v)
	}

	return b.markup(parse.Close, name, optsAndAttr)
}

func (b *Builder) SelfCloseMarkup(name string, optionsAndAttributes ...OptsAndAttr) *Builder {
	return b.markup(parse.SelfClose, name, optionsAndAttributes)
}

func (b *Builder) markup(typ parse.MarkupType, name string, optionsAndAttributes []OptsAndAttr) *Builder {
	if name == "" {
		panic("markup name cannot be empty")
	}

	identifier, err := parseIdentifier(name)
	if err != nil {
		b.err = err
		return b
	}

	markup := parse.Markup{
		Typ:        typ,
		Identifier: identifier,
	}

	for _, opt := range optionsAndAttributes {
		switch v := opt.(type) {
		case FuncOption:
			markup.Options = append(markup.Options, parse.Option(v))
		case attribute:
			markup.Attributes = append(markup.Attributes, parse.Attribute(v))
		}
	}

	switch msg := b.tree.Message.(type) {
	default:
		b.tree.Message = parse.SimpleMessage{markup}
	case parse.SimpleMessage:
		msg = append(msg, markup)
		b.tree.Message = msg
	case parse.ComplexMessage:
	}

	return b
}

type FuncOption parse.Option

func VarOption(name, varName string) FuncOption {
	identifier, err := parseIdentifier(name)
	if err != nil {
		panic(err)
	}

	return FuncOption(parse.Option{Identifier: identifier, Value: parse.Variable(varName)})
}

func LiteralOption(name string, value any) FuncOption {
	identifier, err := parseIdentifier(name)
	if err != nil {
		panic(err)
	}

	return FuncOption(parse.Option{Identifier: identifier, Value: toLiteral(value)})
}

func (FuncOption) optsAndAttr() {}

type ReservedBody interface{ reservedBody() }

type QuotedLiteral parse.QuotedLiteral

type ReservedText parse.ReservedText

func (l QuotedLiteral) reservedBody() {}

func (QuotedLiteral) reservedOrExpression() {}

func (t ReservedText) reservedBody() {}

func (ReservedText) reservedOrExpression() {}

type AnnotationStart int

// Private Use start character.
const (
	Caret     AnnotationStart = iota // ^
	Ampersand                        // &
)

// Reserved start character.
const (
	Exclamation AnnotationStart = iota + 2 // !
	Percent                                // %
	Asterisk                               // *
	Plus                                   // +
	LessThan                               // <
	GreaterThan                            // >
	Question                               // ?
	Tilde                                  // ~
)

func (a AnnotationStart) String() string {
	switch a {
	case Caret:
		return "^"
	case Ampersand:
		return "&"
	case Exclamation:
		return "!"
	case Percent:
		return "%"
	case Asterisk:
		return "*"
	case Plus:
		return "+"
	case LessThan:
		return "<"
	case GreaterThan:
		return ">"
	case Question:
		return "?"
	case Tilde:
		return "~"
	default:
		panic(fmt.Sprintf("unknown annotation start: %d", a))
	}
}

// Annotation adds Private Use or Reserved annotation to the expression.
func Annotation(start AnnotationStart, reservedBody ...ReservedBody) *Expression {
	return Expr().Annotation(start, reservedBody...)
}

// Annotation adds Private Use or Reserved annotation to the expression.
func (e *Expression) Annotation(start AnnotationStart, reservedBody ...ReservedBody) *Expression {
	annotation := parse.ReservedAnnotation{
		Start: []rune(start.String())[0],
	}

	for _, v := range reservedBody {
		switch x := v.(type) {
		case QuotedLiteral:
			annotation.ReservedBody = append(annotation.ReservedBody, parse.QuotedLiteral(x))
		case ReservedText:
			annotation.ReservedBody = append(annotation.ReservedBody, parse.ReservedText(x))
		}
	}

	e.expression.Annotation = annotation

	return e
}

func (e *Expression) Func(name string, option ...FuncOption) *Expression {
	if len(name) == 0 {
		panic("function name cannot be empty")
	}

	identifier, err := parseIdentifier(name)
	if err != nil {
		panic(err)
	}

	f := parse.Function{
		Identifier: identifier,
	}

	if len(option) == 0 {
		e.expression.Annotation = f
		return e
	}

	f.Options = make([]parse.Option, 0, len(option))

	for _, v := range option {
		f.Options = append(f.Options, parse.Option(v))
	}

	e.expression.Annotation = f

	return e
}

// Attributes adds attributes to the expression.
func (e *Expression) Attributes(attributes ...attribute) *Expression {
	for _, v := range attributes {
		e.expression.Attributes = append(e.expression.Attributes, parse.Attribute(v))
	}

	return e
}

type attribute parse.Attribute

func (attribute) optsAndAttr() {}

func VarAttribute(name, varName string) attribute {
	return attribute(parse.Attribute{Identifier: parse.Identifier{Name: name}, Value: parse.Variable(varName)})
}

func LiteralAttribute(name string, value any) attribute {
	return attribute(parse.Attribute{Identifier: parse.Identifier{Name: name}, Value: toLiteral(value)})
}

func EmptyAttribute(name string) attribute {
	return attribute(parse.Attribute{Identifier: parse.Identifier{Name: name}})
}

// helpers

// isName returns true if r is name character.
//
// ABNF:
//
//	name-char = name-start / DIGIT / "-" / "." / %xB7 / %x0300-036F / %x203F-2040.
func isName(v rune) bool {
	return isAlpha(v) ||
		'0' <= v && v <= '9' ||
		v == '-' ||
		v == '.' ||
		v == 0xB7 ||
		0x0300 <= v && v <= 0x036F ||
		0x203F <= v && v <= 2040
}

// isNameStart returns true if r is name start character.
//
// ABNF:
//
//	name-start = ALPHA / "_"
//	           / %xC0-D6 / %xD8-F6 / %xF8-2FF
//	           / %x370-37D / %x37F-1FFF / %x200C-200D
//	           / %x2070-218F / %x2C00-2FEF / %x3001-D7FF
//	           / %xF900-FDCF / %xFDF0-FFFD / %x10000-EFFFF
func isNameStart(r rune) bool {
	return isAlpha(r) ||
		r == '_' ||
		0xC0 <= r && r <= 0xD6 ||
		0xD8 <= r && r <= 0xF6 ||
		0xF8 <= r && r <= 0x2FF ||
		0x370 <= r && r <= 0x37D ||
		0x37F <= r && r <= 0x1FFF ||
		0x200C <= r && r <= 0x200D ||
		0x2070 <= r && r <= 0x218F ||
		0x2C00 <= r && r <= 0x2FEF ||
		0x3001 <= r && r <= 0xD7FF ||
		0xF900 <= r && r <= 0xFDCF ||
		0xFDF0 <= r && r <= 0xFFFD ||
		0x10000 <= r && r <= 0xEFFFF
}

// isAlpha returns true if r is alphabetic character.
func isAlpha(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

func toLiteral(value any) parse.Literal {
	var s string

	switch v := value.(type) {
	default:
		s = fmt.Sprint(value)
	case int:
		return parse.NumberLiteral(float64(v))
	case string:
		s = v
	}

	if len(s) == 0 {
		return parse.QuotedLiteral("")
	}

	for i, v := range s {
		if i == 0 && !isNameStart(v) || i > 0 && !isName(v) {
			return parse.QuotedLiteral(s)
		}
	}

	return parse.NameLiteral(s)
}
