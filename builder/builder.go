package builder

import (
	"errors"
	"fmt"
	"strconv"
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

	if msg, ok := b.tree.Message.(parse.ComplexMessage); ok && msg.ComplexBody == nil {
		msg.ComplexBody = parse.QuotedPattern{}
		b.tree.Message = msg
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
		switch body := msg.ComplexBody.(type) {
		default:
			msg.ComplexBody = parse.QuotedPattern{txt}
			b.tree.Message = msg
		case parse.QuotedPattern:
			body = append(body, txt)
			msg.ComplexBody = body
			b.tree.Message = msg
		case parse.Matcher:
			i := len(body.Variants) - 1
			if i < 0 {
				b.err = fmt.Errorf(`add text "%s" to "%s"`, txt, body)
				return b
			}

			body.Variants[i].QuotedPattern = append(body.Variants[i].QuotedPattern, parse.Text(s))
			msg.ComplexBody = body
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
		}
	case parse.SimpleMessage:
		b.tree.Message = parse.ComplexMessage{
			Declarations: []parse.Declaration{local},
			ComplexBody:  parse.QuotedPattern(msg),
		}
	case parse.ComplexMessage:
		msg.Declarations = append(msg.Declarations, local)
		b.tree.Message = msg
	}

	return b
}

// Input adds input declaration to the builder.
func (b *Builder) Input(expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	input := parse.InputDeclaration(expr.expression)

	switch msg := b.tree.Message.(type) {
	default:
		b.tree.Message = parse.ComplexMessage{
			Declarations: []parse.Declaration{input},
		}
	case parse.SimpleMessage:
		b.tree.Message = parse.ComplexMessage{
			Declarations: []parse.Declaration{input},
			ComplexBody:  parse.QuotedPattern(msg),
		}
	case parse.ComplexMessage:
		msg.Declarations = append(msg.Declarations, input)
		b.tree.Message = msg
	}

	return b
}

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
			msg.ComplexBody = parse.QuotedPattern{expr.expression}
			b.tree.Message = msg
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

func (b *Builder) Match(selector parse.Variable, selectors ...parse.Variable) *Builder {
	if b.err != nil {
		return b
	}

	parseSelectors := append([]parse.Variable{selector}, selectors...)

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

	keys = append([]any{key}, keys...)

	switch msg := b.tree.Message.(type) {
	default:
		b.err = fmt.Errorf(`add keys to "%s"`, msg)
	case parse.ComplexMessage:
		switch body := msg.ComplexBody.(type) {
		default:
			b.err = fmt.Errorf(`add selectors to "%s"`, msg)
		case parse.Matcher:
			n := len(keys)

			if len(body.Selectors) != n {
				b.err = errors.New("number of keys in each variant MUST match the number of selectors in the matcher")
				return b
			}

			all := make([]parse.VariantKey, 0, n)

			for _, k := range keys {
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

func (b *Builder) CloseMarkup(name string, attributes ...Attribute) *Builder {
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
		case Attribute:
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

type Quoted parse.QuotedLiteral

type ReservedText parse.ReservedText

func (l Quoted) reservedBody() {}

func (t ReservedText) reservedBody() {}

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

// Annotation adds Private Use annotation to the expression.
func (e *Expression) Annotation(start AnnotationStart, reservedBody ...ReservedBody) *Expression {
	annotation := parse.PrivateUseAnnotation{
		Start: []rune(start.String())[0],
	}

	for _, reservedBodyPart := range reservedBody {
		switch v := reservedBodyPart.(type) {
		case Quoted:
			annotation.ReservedBody = append(annotation.ReservedBody, parse.QuotedLiteral(v))
		case ReservedText:
			annotation.ReservedBody = append(annotation.ReservedBody, parse.ReservedText(v))
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

// Attr adds attributes to the expression.
func (e *Expression) Attr(attributes ...Attribute) *Expression {
	for _, v := range attributes {
		e.expression.Attributes = append(e.expression.Attributes, parse.Attribute(v))
	}

	return e
}

type Attribute parse.Attribute

func (Attribute) optsAndAttr() {}

func VarAttribute(name, varName string) Attribute {
	return Attribute(parse.Attribute{Identifier: parse.Identifier{Name: name}, Value: parse.Variable(varName)})
}

func LiteralAttribute(name string, value any) Attribute {
	return Attribute(parse.Attribute{Identifier: parse.Identifier{Name: name}, Value: toLiteral(value)})
}

func EmptyAttribute(name string) Attribute {
	return Attribute(parse.Attribute{Identifier: parse.Identifier{Name: name}})
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
		return parse.NumberLiteral(strconv.Itoa(v))
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
