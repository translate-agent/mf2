package builder

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.expect.digital/mf2/parse"
)

const (
	defaultSpacing = " "
	defaultNewline = "\n"
	varSymbol      = "$"
)

type Builder struct {
	spacing string // optional spacing [s]
	newline string
	err     error

	declarations []declaration // input, local, reserved
	selectors    []*Expression // matcher selectors
	variants     []variant     // matcher variants
	pattern      []any
}

func NewBuilder() *Builder {
	return &Builder{
		newline: defaultNewline,
		spacing: defaultSpacing,
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

	for _, decl := range b.declarations {
		s += decl.build(b.spacing) + b.newline
	}

	quotedPattern := len(b.declarations) > 0 && (len(b.variants) == 0 && len(b.selectors) == 0)

	if len(b.pattern) > 0 {
		if v, ok := b.pattern[0].(string); ok && !hasSimpleStart(v) {
			switch {
			case len(b.pattern) == 1 && v == "": // simple message with empty text
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

	for _, v := range b.pattern {
		switch v := v.(type) {
		case string:
			s += textEscape(v)
		case *Expression:
			s += v.build(b.spacing)
		case *markup:
			s += v.build(b.spacing)
		default:
			return "", fmt.Errorf("unsupported pattern type: %T", v)
		}
	}

	if quotedPattern {
		s += "}}"
	}

	if len(b.selectors) > 0 {
		s += ".match"

		for _, v := range b.selectors {
			s += b.spacing + v.build(b.spacing)
		}

		s += b.newline
	}

	for i, v := range b.variants {
		s += v.build(b.spacing)

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
		if len(b.pattern) > 0 {
			return errors.New("complex message MUST have single complex body")
		}

		if len(b.selectors) == 0 {
			return errors.New("matcher MUST have at least one selector")
		}

		if !hasCatchAllVariant(b.variants) {
			return errors.New("matcher MUST have at least one variant with all catch-all keys")
		}
	}

	if len(b.selectors) > 0 && len(b.variants) == 0 {
		return errors.New("matcher MUST have at least one variant")
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

// TODO: add to all expressions.
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
		b.pattern = append(b.pattern, s)
	}

	return b
}

type declaration struct {
	keyword      string
	operand      variable // only for local
	expressions  []Expression
	reservedBody []ReservedBody // only for reserved
}

func (d declaration) build(spacing string) string {
	s := "." + d.keyword

	switch d.keyword {
	case "local":
		s += spacing + varSymbol + string(d.operand) + spacing + "="
	case "input":
		// noop
	default: // reserved
		for _, rb := range d.reservedBody {
			switch rb := rb.(type) {
			case Quoted:
				s += spacing + printQuoted(string(rb))
			case ReservedText:
				s += spacing + reservedEscape(string(rb))
			}
		}
	}

	for _, expr := range d.expressions {
		s += spacing + expr.build(spacing)
	}

	return s
}

// Local adds local declaration to the builder.
func (b *Builder) Local(v string, expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	b.declarations = append(b.declarations, declaration{
		keyword:     "local",
		operand:     variable(v),
		expressions: []Expression{*expr},
	})

	return b
}

// Input adds input declaration to the builder.
func (b *Builder) Input(expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	b.declarations = append(b.declarations, declaration{
		keyword:     "input",
		expressions: []Expression{*expr},
	})

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

	decl := declaration{keyword: keyword, expressions: []Expression{*expression}}

	for _, v := range reservedOrExpression {
		switch v := v.(type) {
		case ReservedBody:
			decl.reservedBody = append(decl.reservedBody, v)
		case *Expression:
			decl.expressions = append(decl.expressions, *v)
		}
	}

	b.declarations = append(b.declarations, decl)

	return b
}

type ReservedOrExpression interface{ reservedOrExpression() }

func Expr() *Expression { return new(Expression) }

func (b *Builder) Expr(expr *Expression) *Builder {
	if b.err != nil {
		return b
	}

	if len(b.variants) > 0 {
		b.variants[len(b.variants)-1].pattern = append(b.variants[len(b.variants)-1].pattern, expr)
		return b
	}

	b.pattern = append(b.pattern, expr)

	return b
}

func (b *Builder) Match(selector *Expression, selectors ...*Expression) *Builder {
	if b.err != nil {
		return b
	}

	if len(b.pattern) > 0 {
		b.err = errors.New("complex message cannot be added after simple message")
		return b
	}

	b.selectors = append(b.selectors, selector)
	b.selectors = append(b.selectors, selectors...)

	return b
}

func (b *Builder) Keys(key any, keys ...any) *Builder {
	if b.err != nil {
		return b
	}

	if len(keys)+1 != len(b.selectors) {
		b.err = errors.New("number of keys in each variant MUST match the number of selectors in the matcher")
		return b
	}

	b.variants = append(b.variants, variant{keys: append([]any{key}, keys...)})

	return b
}

type variant struct {
	keys    []any
	pattern []any
}

func (v *variant) build(spacing string) string {
	var s string

	for i, k := range v.keys {
		if i > 0 {
			s += cmp.Or(spacing, defaultSpacing)
		}

		if k == "*" {
			s += "*"
		} else {
			s += printLiteral(k)
		}
	}

	s += spacing + "{{"

	for i := range v.pattern {
		switch p := v.pattern[i].(type) {
		case string:
			s += textEscape(p)
		case *Expression:
			s += p.build(spacing)
		default:
			panic(fmt.Sprintf("unsupported pattern type: %T", p))
		}
	}

	return s + "}}"
}

type literal any

type variable string

type function struct {
	name    string
	options []FuncOption
}

type Expression struct {
	operand    any // literal or variable
	annotation any // function or annotation
	attributes []attribute
}

func (Expression) reservedOrExpression() {}

func (e *Expression) build(spacing string) string {
	s := "{"

	switch v := e.operand.(type) {
	case variable:
		s += spacing + varSymbol + string(v)
	case nil:
		// noop
	case literal:
		s += spacing + printLiteral(v)
	default:
		panic(fmt.Sprintf("unsupported operand type: %T", v))
	}

	switch f := e.annotation.(type) {
	case function:
		s += spacing + ":" + f.name
		for _, opt := range f.options {
			s += opt.sprint(spacing)
		}

	case annotation:
		s += spacing + f.start.String()

		for _, v := range f.body {
			switch v := v.(type) {
			case Quoted:
				s += spacing + printQuoted(string(v))
			case ReservedText:
				s += spacing + reservedEscape(string(v))
			}
		}
	}

	// attributes

	for _, attr := range e.attributes {
		s += attr.sprint(spacing)
	}

	return s + spacing + "}"
}

func Literal(v any) *Expression {
	return Expr().Literal(v)
}

func (e *Expression) Literal(v any) *Expression {
	e.operand = v
	return e
}

func Var(name string) *Expression {
	return Expr().Var(name)
}

func (e *Expression) Var(name string) *Expression {
	if len(name) == 0 {
		panic("variable name cannot be empty")
	}

	e.operand = variable(name)

	return e
}

type markup struct {
	name       string       // required
	options    []FuncOption // optional, only allowed for open markup
	attributes []attribute  // optional
	typ        parse.MarkupType
}

func (m *markup) build(spacing string) string {
	var s string

	switch m.typ {
	case parse.Open, parse.SelfClose:
		s += "#" + m.name
	case parse.Close:
		s += "/" + m.name
	case parse.Unspecified:
		panic("unspecified markup type")
	}

	for _, opt := range m.options {
		s += opt.sprint(spacing)
	}

	for _, attr := range m.attributes {
		s += attr.sprint(spacing)
	}

	if m.typ == parse.SelfClose {
		return fmt.Sprintf("{%s%s%s/}", spacing, s, spacing)
	}

	return fmt.Sprintf("{%s%s%s}", spacing, s, spacing)
}

// Hack: limit to only options and attributes, instead of any.
type OptsAndAttr interface{ optsAndAttr() }

func (b *Builder) OpenMarkup(name string, optionsAndAttributes ...OptsAndAttr) *Builder {
	if name == "" {
		panic("markup name cannot be empty")
	}

	markup := &markup{name: name, typ: parse.Open}

	for _, v := range optionsAndAttributes {
		switch v := v.(type) {
		case FuncOption:
			markup.options = append(markup.options, v)
		case attribute:
			markup.attributes = append(markup.attributes, v)
		}
	}

	b.pattern = append(b.pattern, markup)

	return b
}

func (b *Builder) CloseMarkup(name string, attributes ...attribute) *Builder {
	if name == "" {
		panic("markup name cannot be empty")
	}

	b.pattern = append(b.pattern, &markup{name: name, typ: parse.Close, attributes: attributes})

	return b
}

func (b *Builder) SelfCloseMarkup(name string, optionsAndAttributes ...OptsAndAttr) *Builder {
	// Same as OpenMarkup, but with SelfClose type. So, we can reuse the code.
	b.OpenMarkup(name, optionsAndAttributes...)

	added := b.pattern[len(b.pattern)-1].(*markup) //nolint:forcetypeassert
	added.typ = parse.SelfClose

	return b
}

type FuncOption struct {
	operand any // literal or variable
	key     string
}

func (o *FuncOption) sprint(spacing string) string {
	var optVal string

	switch v := o.operand.(type) {
	case variable:
		optVal = "$" + string(v)
	case literal:
		optVal = printLiteral(v)
	}

	return fmt.Sprintf("%s%s%s=%s%s", spacing, o.key, spacing, spacing, optVal)
}

func VarOption(name, varName string) FuncOption {
	return FuncOption{key: name, operand: variable(varName)}
}

func LiteralOption(name string, value any) FuncOption {
	return FuncOption{key: name, operand: value}
}

func (FuncOption) optsAndAttr() {}

type ReservedBody interface{ reservedBody() }

type (
	Quoted       string
	ReservedText string
)

func (Quoted) reservedBody()         {}
func (Quoted) reservedOrExpression() {}

func (ReservedText) reservedBody()         {}
func (ReservedText) reservedOrExpression() {}

type annotation struct {
	body  []ReservedBody
	start AnnotationStart
}

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
	e.annotation = annotation{body: reservedBody, start: start}

	return e
}

func (e *Expression) Func(name string, option ...FuncOption) *Expression {
	if len(name) == 0 {
		panic("function name cannot be empty")
	}

	e.annotation = function{name: name, options: option}

	return e
}

// Attr adds attributes to the expression.
func (e *Expression) Attr(attributes ...attribute) *Expression {
	e.attributes = append(e.attributes, attributes...)

	return e
}

type attribute struct {
	value any    // optional: literal or variable
	name  string // required
}

func (a *attribute) sprint(spacing string) string {
	var attrVal string

	switch val := a.value.(type) {
	case variable:
		attrVal = "$" + string(val)
	case literal:
		attrVal = printLiteral(val)
	default: // empty attribute
		return fmt.Sprintf("%s@%s", spacing, a.name)
	}

	return fmt.Sprintf("%s@%s%s=%s%s", spacing, a.name, spacing, spacing, attrVal)
}

func (attribute) optsAndAttr() {}

func VarAttribute(name, varName string) attribute {
	return attribute{name: name, value: variable(varName)}
}

func LiteralAttribute(name string, value any) attribute {
	return attribute{name: name, value: value}
}

func EmptyAttribute(name string) attribute {
	return attribute{name: name}
}

func printLiteral(l any) string {
	switch v := l.(type) { // TODO: more liberal
	case string:
		if len(v) == 0 {
			return printQuoted(v)
		}

		for i, r := range v {
			if i == 0 && !isNameStart(r) || i > 0 && !isName(r) {
				return printQuoted(v)
			}
		}

		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64:
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}

		return string(b)
	default:
		panic(fmt.Sprintf("unsupported literal type: %T", v))
	}
}

// helpers

/*
	printQuoted escapes special characters in quoted name literal.

ABNF:
quoted-escape   = backslash ( backslash / "|" )
.
*/
func printQuoted(s string) string {
	return "|" + strings.NewReplacer("\\", "\\\\", "|", "\\|").Replace(s) + "|"
}

/*
	textEscape escapes special characters in text.

ABNF:
text-escape     = backslash ( backslash / "{" / "}" )
.
*/
func textEscape(s string) string {
	return strings.NewReplacer("\\", "\\\\", "{", "\\{", "}", "\\}").Replace(s)
}

func reservedEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `{`, `\{`, `|`, `\|`, `}`, `\}`).Replace(s)
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
