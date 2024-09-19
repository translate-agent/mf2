package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.expect.digital/mf2"
)

type parser struct {
	reservedVariable Variable // reservedVariable is a variable that cannot be re-declared within an expression
	declaration      string   // input or local or empty
	lexer            *lexer
	items            []item
	variables        []Variable
	pos              int
}

func (p *parser) duplicateVariable(variable Variable) error {
	if slices.Contains(p.variables, variable) {
		return fmt.Errorf("%w: %s", mf2.ErrDuplicateDeclaration, variable)
	}

	return nil
}

func (p *parser) declareVariable(variable Variable) {
	if !slices.Contains(p.variables, variable) {
		p.variables = append(p.variables, variable)
	}
}

// next returns next token if any otherwise returns error token.
func (p *parser) next() item {
	if p.pos == len(p.items)-1 {
		return mk(itemError, "no more tokens")
	}

	p.pos++

	return p.items[p.pos]
}

func (p *parser) backup() {
	p.pos--
}

// nextNonWS returns next non-whitespace token if any otherwise returns error token.
func (p *parser) nextNonWS() item {
	next := p.next()
	for ; next.typ == itemWhitespace; next = p.next() { //nolint:revive
	}

	return next
}

// peekNonWS returns next non-whitespace token.
func (p *parser) peekNonWS() item {
	for pos := p.pos + 1; pos < len(p.items)-1; pos++ {
		if itm := p.items[pos]; itm.typ != itemWhitespace {
			return itm
		}
	}

	return mk(itemError, "nothing to peek")
}

func (p *parser) current() item {
	return p.items[p.pos]
}

func (p *parser) collect() error {
	// sanity check, avoid infinite loop
	for range 1000 {
		itm := p.lexer.nextItem()
		if itm.typ == itemError {
			return itm.err
		}

		p.items = append(p.items, itm)

		if itm.typ == itemEOF {
			return nil
		}
	}

	return errors.New("too many tokens. infinite loop?")
}

// isComplexMessage returns true if first token is one of the complex message tokens.
func (p *parser) isComplexMessage() bool {
	switch p.peekNonWS().typ {
	default:
		return false
	case itemInputKeyword, itemLocalKeyword, itemMatchKeyword, itemQuotedPatternOpen:
		return true
	}
}

/*
Parse parses the input string and returns an AST tree of MessageFormat2.

Examples:

	mf2.Parse("Hello World!")
	// result
	AST{Message: SimpleMessage{Text("Hello World!")}}

	// -----------------------------------------------------------

	mf2.Parse("Hello {name}!")
	// result
	AST{
		Message: SimpleMessage{
			Text("Hello "),
			Expression{Operand: Variable("name")},
		},
	}

	// -----------------------------------------------------------

	mf2.Parse(".match {$count} 1 {{Hello world}} * {{Hello worlds}}")
	// result
	AST{
		Message: ComplexMessage{
			ComplexBody: Matcher{
				MatchStatements: []Expression{{Operand: Variable("count")}},
				Variants: []Variant{
					{
						Keys: []VariantKey{NumberLiteral(1)},
						QuotedPattern: QuotedPattern{
							Text("Hello world"),
						},
					},
					{
						Keys: []VariantKey{CatchAllKey{}},
						QuotedPattern: QuotedPattern{
							Text("Hello worlds"),
						},
					},
				},
			},
		},
	}
*/
func Parse(input string) (AST, error) {
	errorf := func(format string, err error) (AST, error) {
		// TODO(jhorsts): improve error handling, add MF2 syntax error as early as possible.
		switch {
		default:
			return AST{}, fmt.Errorf("parse MF2: %w: "+format, mf2.ErrSyntax, err)
		case errors.Is(err, mf2.ErrDuplicateDeclaration),
			errors.Is(err, mf2.ErrDuplicateOptionName),
			errors.Is(err, mf2.ErrMissingSelectorAnnotation):
			return AST{}, fmt.Errorf("parse MF2: "+format, err)
		}
	}

	p := &parser{lexer: lex(input), pos: -1}
	if err := p.collect(); err != nil {
		return errorf("%w", err)
	}

	if len(p.items) == 1 && p.items[0].typ == itemEOF {
		return AST{}, nil
	}

	parse := func() (Message, error) { return p.parseSimpleMessage() }
	if p.isComplexMessage() {
		parse = func() (Message, error) { return p.parseComplexMessage() }
	}

	message, err := parse()
	if err != nil {
		return errorf("%w", err)
	}

	if itm := p.nextNonWS(); itm.typ != itemEOF {
		return errorf("%w", unexpectedErr(itm, itemEOF))
	}

	return AST{Message: message}, nil
}

// ------------------------------Message------------------------------

func (p *parser) parseSimpleMessage() (SimpleMessage, error) {
	pattern, err := p.parsePattern()
	if err != nil {
		return SimpleMessage{}, fmt.Errorf("simple message: %w", err)
	}

	return SimpleMessage(pattern), nil
}

func (p *parser) parseComplexMessage() (ComplexMessage, error) {
	var message ComplexMessage

	errorf := func(format string, args ...any) (ComplexMessage, error) {
		return ComplexMessage{}, fmt.Errorf("complex message: "+format, args...)
	}

	// optional declarations

declarationsLoop:
	for {
		switch itm := p.nextNonWS(); itm.typ {
		default:
			p.backup()
			break declarationsLoop
		case itemInputKeyword:
			declaration, err := p.parseInputDeclaration()
			if err != nil {
				return errorf("%w", err)
			}

			message.Declarations = append(message.Declarations, declaration)
		case itemLocalKeyword:
			declaration, err := p.parseLocalDeclaration()
			if err != nil {
				return errorf("%w", err)
			}

			message.Declarations = append(message.Declarations, declaration)
		}
	}

	// complex body

	switch itm := p.nextNonWS(); itm.typ {
	default:
		err := unexpectedErr(
			itm, itemInputKeyword, itemLocalKeyword, itemMatchKeyword, itemQuotedPatternOpen)
		return errorf("%w", err)
	case itemQuotedPatternOpen:
		pattern, err := p.parsePattern()
		if err != nil {
			return errorf("%w", err)
		}

		if itm := p.next(); itm.typ != itemQuotedPatternClose {
			return errorf("%w", unexpectedErr(itm, itemQuotedPatternClose))
		}

		message.ComplexBody = QuotedPattern(pattern)
	case itemMatchKeyword:
		matcher, err := p.parseMatcher(message.Declarations)
		if err != nil {
			return errorf("%w", err)
		}

		message.ComplexBody = matcher
	case itemEOF:
		return errorf("missing complex body")
	}

	return message, nil
}

// ------------------------------Pattern------------------------------

// parsePattern parses a slice of pattern parts.
func (p *parser) parsePattern() ([]PatternPart, error) {
	var pattern []PatternPart

	errorf := func(format string, args ...any) ([]PatternPart, error) {
		return nil, fmt.Errorf("pattern: "+format, args...)
	}

	// Loop until the end, or closing pattern quote, if parsing complex message.
	for {
		switch itm := p.next(); itm.typ {
		default:
			err := unexpectedErr(itm, itemWhitespace, itemText, itemExpressionOpen, itemMarkupOpen, itemMarkupClose)
			return errorf("%w", err)
		case itemWhitespace:
			continue
		case itemQuotedPatternClose, itemEOF:
			p.backup()
			return pattern, nil
		case itemText:
			pattern = append(pattern, Text(itm.val))
		case itemExpressionOpen:
			// markup?
			if typ := p.peekNonWS().typ; typ == itemMarkupOpen || typ == itemMarkupClose {
				markup, err := p.parseMarkup()
				if err != nil {
					return errorf("%w", err)
				}

				pattern = append(pattern, markup)

				continue
			}

			expression, err := p.parseExpression()
			if err != nil {
				return errorf("%w", err)
			}

			pattern = append(pattern, expression)
		}
	}
}

// --------------------------------Markup--------------------------------

func (p *parser) parseMarkup() (Markup, error) {
	var markup Markup

	errorf := func(format string, args ...any) (Markup, error) {
		return Markup{}, fmt.Errorf("markup: "+format, args...)
	}

	switch itm := p.nextNonWS(); itm.typ {
	default:
		err := unexpectedErr(itm, itemMarkupOpen, itemMarkupClose)
		return errorf("open item: %w", err)
	case itemMarkupOpen:
		markup.Typ = Open
	case itemMarkupClose:
		markup.Typ = Close
	}

	markup.Identifier = p.parseIdentifier()

	// options

optionsLoop:
	for {
		switch itm := p.nextNonWS(); itm.typ {
		default:
			err := unexpectedErr(itm, itemOption, itemAttribute, itemMarkupClose, itemExpressionClose)
			return errorf("options: %w", err)
		case itemOption:
			option, err := p.parseOption()
			if err != nil {
				return errorf("%w", err)
			}

			markup.Options = append(markup.Options, option)
		case itemAttribute:
			p.backup()
			break optionsLoop
		case itemExpressionClose:
			return markup, nil
		case itemMarkupClose:
			if markup.Typ == Close {
				return errorf("closing close markup")
			}

			if itm := p.next(); itm.typ != itemExpressionClose {
				return errorf("%w", unexpectedErr(itm, itemExpressionClose))
			}

			markup.Typ = SelfClose

			return markup, nil
		}
	}

	// attributes

	for {
		switch itm := p.nextNonWS(); itm.typ {
		default:
			err := unexpectedErr(itm, itemAttribute, itemMarkupClose, itemExpressionClose)
			return errorf("%w", err)
		case itemAttribute:
			attribute, err := p.parseAttribute()
			if err != nil {
				return errorf("%w", err)
			}

			markup.Attributes = append(markup.Attributes, attribute)
		case itemExpressionClose:
			return markup, nil
		case itemMarkupClose:
			if markup.Typ == Close {
				return errorf("closing close markup")
			}

			if itm := p.next(); itm.typ != itemExpressionClose {
				return errorf("%w", unexpectedErr(itm, itemExpressionClose))
			}

			markup.Typ = SelfClose

			return markup, nil
		}
	}
}

// ------------------------------Expression------------------------------

func (p *parser) parseExpression() (Expression, error) {
	var (
		expr Expression
		err  error
	)

	errorf := func(format string, args ...any) (Expression, error) { //nolint:unparam
		return Expression{}, fmt.Errorf("expression: "+format, args...)
	}

	// optional operand - literal or variable

	switch itm := p.nextNonWS(); itm.typ {
	default:
		err = unexpectedErr(itm,
			itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
			itemFunction, itemExpressionClose)

		return errorf("%w: %w", mf2.ErrBadOperand, err)
	case itemVariable:
		variable := Variable(itm.val)

		switch p.declaration {
		case "local":
			// .local $foo = {$foo}
			if variable == p.reservedVariable {
				return errorf("%w: %s", mf2.ErrDuplicateDeclaration, variable)
			}
		case "input":
			// .input {$foo} .input {$foo}
			if err = p.duplicateVariable(variable); err != nil {
				return errorf("%w", err)
			}

			p.reservedVariable = variable
			defer func() { p.reservedVariable = "" }()
		}

		p.declareVariable(variable)
		expr.Operand = variable
	case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
		if expr.Operand, err = p.parseLiteral(); err != nil {
			return errorf("%w", err)
		}
	case itemFunction:
		p.backup()
	case itemExpressionClose: // empty expression
		return errorf("missing operand or annotation")
	}

	if p.peekNonWS().typ == itemExpressionClose { // expression with operand only
		p.nextNonWS()
		return expr, nil
	}

	// ensure whitespace follows operand before annotation
	if expr.Operand != nil {
		if itm := p.next(); itm.typ != itemWhitespace {
			return errorf("between operand and annotation: %w", unexpectedErr(itm, itemWhitespace))
		}
	}

	// parse annotation

	switch itm := p.next(); itm.typ {
	default:
		return errorf("%w", unexpectedErr(itm, itemFunction, itemAttribute))
	case itemFunction:
		if expr.Annotation, err = p.parseFunction(); err != nil {
			return errorf("%w", err)
		}
	case itemAttribute:
		if expr.Operand == nil {
			return errorf("variable, literal or annotation is required before attribute")
		}

		p.backup() // attribute
		p.backup() // whitespace
	}

	if p.peekNonWS().typ == itemExpressionClose {
		p.nextNonWS()
		return expr, nil
	}

	// ensure whitespace follows annotation before attributes
	if itm := p.next(); itm.typ != itemWhitespace {
		return errorf("%w", unexpectedErr(itm, itemWhitespace))
	}

	// parse attributes

	if expr.Attributes, err = p.parseAttributes(); err != nil {
		return errorf("%w", err)
	}

	if itm := p.nextNonWS(); itm.typ != itemExpressionClose {
		return errorf("%w", unexpectedErr(itm, itemExpressionClose))
	}

	return expr, nil
}

// ------------------------------Annotation------------------------------

func (p *parser) parseFunction() (Function, error) {
	function := Function{Identifier: p.parseIdentifier()}

	if p.peekNonWS().typ == itemExpressionClose {
		return function, nil
	}

	errorf := func(format string, args ...any) (Function, error) { //nolint:unparam
		return Function{}, fmt.Errorf("function: "+format, args...)
	}

	// parse options
	opts := make(map[string]struct{})

	for {
		if itm := p.next(); itm.typ != itemWhitespace {
			return errorf("%w", unexpectedErr(itm, itemWhitespace))
		}

		switch itm := p.next(); itm.typ {
		default:
			return errorf("%w", unexpectedErr(itm, itemOption, itemExpressionClose, itemAttribute))
		case itemOption:
			option, err := p.parseOption()
			if err != nil {
				return errorf("%w", err)
			}

			if _, ok := opts[option.Identifier.String()]; ok {
				return errorf("%w", mf2.ErrDuplicateOptionName)
			}

			opts[option.Identifier.String()] = struct{}{}

			function.Options = append(function.Options, option)
		case itemAttribute: // end of function, attributes are next
			p.backup()
			p.backup() // whitespace

			return function, nil
		}

		if p.peekNonWS().typ == itemExpressionClose {
			return function, nil
		}
	}
}

// ------------------------------Declaration------------------------------

func (p *parser) parseLocalDeclaration() (LocalDeclaration, error) {
	errorf := func(err error) (LocalDeclaration, error) { //nolint:unparam
		return LocalDeclaration{}, fmt.Errorf("local declaration: %w", err)
	}

	p.declaration = "local"
	defer func() { p.declaration = "" }()

	next := p.next()
	if next.typ != itemWhitespace {
		return errorf(unexpectedErr(next, itemWhitespace))
	}

	if next = p.next(); next.typ != itemVariable {
		return errorf(unexpectedErr(next, itemVariable))
	}

	variable := Variable(next.val)
	if err := p.duplicateVariable(variable); err != nil {
		return errorf(err)
	}

	p.reservedVariable = variable
	defer func() { p.reservedVariable = "" }()

	p.declareVariable(variable)

	declaration := LocalDeclaration{Variable: variable}

	if next = p.nextNonWS(); next.typ != itemOperator {
		return errorf(unexpectedErr(next, itemOperator))
	}

	if next = p.nextNonWS(); next.typ != itemExpressionOpen {
		return errorf(unexpectedErr(next, itemExpressionOpen))
	}

	expression, err := p.parseExpression()
	if err != nil {
		return errorf(err)
	}

	declaration.Expression = expression

	return declaration, nil
}

func (p *parser) parseInputDeclaration() (InputDeclaration, error) {
	errorf := func(format string, args ...any) (InputDeclaration, error) { //nolint:unparam
		return InputDeclaration{}, fmt.Errorf("input declaration: "+format, args...)
	}

	p.declaration = "input"
	defer func() { p.declaration = "" }()

	next := p.nextNonWS()
	if next.typ != itemExpressionOpen {
		return errorf("%w", unexpectedErr(next, itemExpressionOpen))
	}

	expression, err := p.parseExpression()
	if err != nil {
		return errorf("%w", err)
	}

	return InputDeclaration(expression), nil
}

// ---------------------------------------------------------------------

// isFallback returns true if all keys are "*".
func isFallback(keys []VariantKey) bool {
	fallbackVariant := CatchAllKey{}

	for _, key := range keys {
		if key != fallbackVariant {
			return false
		}
	}

	return true
}

func hasAnnotation(variable Variable, declarations []Declaration) bool {
	for _, v := range declarations {
		switch t := v.(type) {
		case InputDeclaration:
			if t.Operand != variable {
				continue
			}

			return t.Annotation != nil
		case LocalDeclaration:
			if t.Variable != variable {
				continue
			}

			if t.Expression.Annotation != nil {
				return true
			}

			if u, ok := t.Expression.Operand.(Variable); ok {
				return hasAnnotation(u, declarations)
			}
		}
	}

	return false
}

//nolint:gocognit
func (p *parser) parseMatcher(declarations []Declaration) (Matcher, error) {
	var matcher Matcher

	errorf := func(format string, args ...any) (Matcher, error) {
		return Matcher{}, fmt.Errorf("matcher: "+format, args...)
	}

	// parse one or more selectors

selectorsLoop:
	for {
		itm := p.next()
		if itm.typ != itemWhitespace {
			return errorf("missing whitespace before selector: %w", unexpectedErr(itm, itemWhitespace))
		}

		switch itm := p.next(); itm.typ {
		default:
			p.backup()
			break selectorsLoop
		case itemEOF:
			return errorf("%w", unexpectedErr(itm))
		case itemVariable:
			if !hasAnnotation(Variable(itm.val), declarations) {
				return errorf("%w", mf2.ErrMissingSelectorAnnotation)
			}

			matcher.Selectors = append(matcher.Selectors, Variable(itm.val))
		}
	}

	if v := p.current(); v.typ != itemWhitespace {
		// there should be a whitespace between selectors and variants
		return errorf("missing whitespace between selectors and variants: %w", mf2.ErrSyntax)
	}

	// parse one or more variants

	for {
		switch itm := p.nextNonWS(); itm.typ {
		default:
			err := unexpectedErr(itm, itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral)
			return errorf("%w", err)
		case itemEOF:
			p.backup()

			// fallback variant is required
			for i := range matcher.Variants {
				if isFallback(matcher.Variants[i].Keys) {
					return matcher, nil
				}
			}

			return errorf("%w", mf2.ErrMissingFallbackVariant)
		case itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			keys, err := p.parseVariantKeys()
			if err != nil {
				return errorf("%w", err)
			}

			if len(keys) != len(matcher.Selectors) {
				return errorf("%w: %d selectors and %d keys", mf2.ErrVariantKeyMismatch, len(matcher.Selectors), len(keys))
			}

			pattern, err := p.parsePattern()
			if err != nil {
				return errorf("%w", err)
			}

			if itm := p.next(); itm.typ != itemQuotedPatternClose {
				return errorf("variant pattern: %w", unexpectedErr(itm, itemExpressionClose))
			}

			matcher.Variants = append(matcher.Variants, Variant{Keys: keys, QuotedPattern: QuotedPattern(pattern)})
		}
	}
}

func (p *parser) parseVariantKeys() ([]VariantKey, error) {
	var (
		keys   []VariantKey
		spaced bool // all keys must be separated by space
	)

	errorf := func(format string, args ...any) ([]VariantKey, error) {
		return nil, fmt.Errorf("variant keys: "+format, args...)
	}

	for itm := p.current(); itm.typ != itemQuotedPatternOpen; itm = p.next() {
		switch itm.typ {
		default:
			err := unexpectedErr(itm, itemWhitespace, itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral)
			return errorf("%w", err)
		case itemWhitespace:
			spaced = true
			continue
		case itemCatchAllKey:
			if !spaced && len(keys) > 0 {
				return errorf("missing space between keys %v and *", keys[len(keys)-1])
			}

			keys = append(keys, CatchAllKey{})
			spaced = false
		case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			if !spaced && len(keys) > 0 {
				return errorf("missing space between keys %v and %s", keys[len(keys)-1], itm.val)
			}

			literal, err := p.parseLiteral()
			if err != nil {
				return errorf("%w", err)
			}

			keys = append(keys, literal)
			spaced = false
		}
	}

	return keys, nil
}

func (p *parser) parseOption() (Option, error) {
	option := Option{Identifier: p.parseIdentifier()}
	errorf := func(format string, args ...any) (Option, error) {
		return Option{}, fmt.Errorf("option: "+format, args...)
	}

	// Next token must be an operator.
	if next := p.nextNonWS(); next.typ != itemOperator {
		return errorf("%w", unexpectedErr(next, itemOperator))
	}

	// Next after operator must be a variable or literal.
	switch next := p.nextNonWS(); next.typ {
	default:
		err := unexpectedErr(next, itemVariable, itemQuotedLiteral, itemUnquotedLiteral, itemNumberLiteral)
		return errorf("%w", err)
	case itemVariable:
		variable := Variable(next.val)
		if variable == p.reservedVariable {
			return errorf("%w: %s", mf2.ErrDuplicateDeclaration, variable)
		}

		p.declareVariable(variable)
		option.Value = variable
	case itemQuotedLiteral, itemUnquotedLiteral, itemNumberLiteral:
		var err error
		if option.Value, err = p.parseLiteral(); err != nil {
			return errorf("%w", err)
		}
	}

	return option, nil
}

func (p *parser) parseAttributes() ([]Attribute, error) {
	var attributes []Attribute

	errorf := func(format string, args ...any) ([]Attribute, error) {
		return nil, fmt.Errorf("attribute at %d: "+format, append([]any{len(attributes)}, args...)...)
	}

	for {
		if len(attributes) > 0 {
			if itm := p.next(); itm.typ != itemWhitespace {
				return errorf("%w", unexpectedErr(itm, itemWhitespace))
			}
		}

		itm := p.next()
		if itm.typ != itemAttribute {
			return errorf("%w", unexpectedErr(itm, itemAttribute, itemExpressionClose))
		}

		attribute, err := p.parseAttribute()
		if err != nil {
			return errorf("%w", err)
		}

		attributes = append(attributes, attribute)

		if p.peekNonWS().typ == itemExpressionClose {
			return attributes, nil
		}
	}
}

func (p *parser) parseAttribute() (Attribute, error) {
	attribute := Attribute{Identifier: p.parseIdentifier()}
	errorf := func(format string, args ...any) (Attribute, error) {
		return Attribute{}, fmt.Errorf("attribute: "+format, args...)
	}

	switch itm := p.peekNonWS(); itm.typ {
	default:
		return errorf("%w", unexpectedErr(itm, itemAttribute, itemOperator, itemExpressionClose))
	case itemOperator:
		p.nextNonWS() // skip it
	case itemExpressionClose, itemAttribute:
		return attribute, nil
	}

	var err error

	switch itm := p.nextNonWS(); itm.typ {
	default:
		return errorf("%w", unexpectedErr(itm, itemVariable, itemQuotedLiteral, itemUnquotedLiteral, itemNumberLiteral))
	case itemVariable:
		variable := Variable(itm.val)
		p.declareVariable(variable)
		attribute.Value = variable
	case itemQuotedLiteral, itemUnquotedLiteral, itemNumberLiteral:
		if attribute.Value, err = p.parseLiteral(); err != nil {
			return errorf("%w", err)
		}
	}

	return attribute, nil
}

func (p *parser) parseLiteral() (Literal, error) {
	switch itm := p.current(); itm.typ {
	default:
		err := unexpectedErr(itm, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral)
		return nil, fmt.Errorf("literal: %w", err)
	case itemNumberLiteral:
		var num float64
		if err := json.Unmarshal([]byte(itm.val), &num); err != nil {
			return nil, fmt.Errorf("number literal: %w", err)
		}

		return NumberLiteral(itm.val), nil
	case itemQuotedLiteral:
		return QuotedLiteral(itm.val), nil
	case itemUnquotedLiteral:
		return NameLiteral(itm.val), nil
	}
}

func (p *parser) parseIdentifier() Identifier {
	split := strings.Split(p.current().val, ":") // namespace:name

	if len(split) == 1 {
		return Identifier{Name: split[0]}
	}

	return Identifier{Namespace: split[0], Name: split[1]}
}

// UnexpectedTokenError is returned when parser encounters unexpected token.
// It contains information about expected token types and actual token type.
type UnexpectedTokenError struct {
	expected []itemType
	actual   item
}

func (u UnexpectedTokenError) Error() string {
	if len(u.expected) == 0 {
		return "want no items, got " + u.actual.String()
	}

	r := `"` + u.expected[0].String() + `"`
	for _, typ := range u.expected[1:] {
		r += `, "` + typ.String() + `"`
	}

	return "want item " + r + `, got ` + u.actual.String()
}

func unexpectedErr(actual item, expected ...itemType) UnexpectedTokenError {
	return UnexpectedTokenError{
		actual:   actual,
		expected: expected,
	}
}
