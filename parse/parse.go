package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type parser struct {
	lexer *lexer
	items []item
	pos   int
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
	for ; next.typ == itemWhitespace; next = p.next() {
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
			return fmt.Errorf("got error token: %s", itm.val)
		}

		p.items = append(p.items, itm)

		if itm.typ == itemEOF {
			return nil
		}
	}

	return errors.New("too many tokens. infinite loop ?")
}

// isComplexMessage returns true if first token is one of the complex message tokens.
func (p *parser) isComplexMessage() bool {
	switch p.items[0].typ {
	default:
		return false
	case itemInputKeyword, itemLocalKeyword, itemMatchKeyword, itemReservedKeyword, itemQuotedPatternOpen:
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
	p := &parser{lexer: lex(input), pos: -1}
	if err := p.collect(); err != nil {
		return AST{}, fmt.Errorf("collect tokens: %w", err)
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
		return AST{}, fmt.Errorf("parse message `%s`: %w", input, err)
	}

	ast := AST{Message: message}
	if err := ast.validate(); err != nil {
		return AST{}, fmt.Errorf("validate AST: %w", err)
	}

	return ast, nil
}

// ------------------------------Message------------------------------

func (p *parser) parseSimpleMessage() (SimpleMessage, error) {
	pattern, err := p.parsePattern()
	if err != nil {
		return SimpleMessage{}, fmt.Errorf("parse pattern: %w", err)
	}

	return SimpleMessage(pattern), nil
}

func (p *parser) parseComplexMessage() (ComplexMessage, error) {
	var message ComplexMessage

	for {
		switch itm := p.nextNonWS(); itm.typ {
		// Ending tokens
		default:
			return ComplexMessage{},
				unexpectedErr(itm, itemInputKeyword, itemLocalKeyword, itemReservedKeyword, itemMatchKeyword, itemQuotedPatternOpen)
		case itemError:
			return ComplexMessage{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemEOF:
			return message, nil
		// Non-ending tokens
		case itemInputKeyword:
			declaration, err := p.parseInputDeclaration()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse input declaration: %w", err)
			}

			message.Declarations = append(message.Declarations, declaration)
		case itemLocalKeyword:
			declaration, err := p.parseLocalDeclaration()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse local declaration: %w", err)
			}

			message.Declarations = append(message.Declarations, declaration)
		case itemReservedKeyword:
			declaration, err := p.parseReservedStatement()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse reserved statement: %w", err)
			}

			message.Declarations = append(message.Declarations, declaration)
		case itemMatchKeyword:
			matcher, err := p.parseMatcher()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse matcher: %w", err)
			}

			message.ComplexBody = matcher
		case itemQuotedPatternOpen:
			pattern, err := p.parsePattern()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse pattern: %w", err)
			}

			message.ComplexBody = QuotedPattern(pattern)
		}
	}
}

// ------------------------------Pattern------------------------------

// parsePattern parses a slice of pattern parts.
func (p *parser) parsePattern() ([]PatternPart, error) {
	var pattern []PatternPart

	// Loop until the end, or closing pattern quote, if parsing complex message.
	for itm := p.next(); itm.typ != itemEOF && itm.typ != itemQuotedPatternClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return nil, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemText:
			pattern = append(pattern, Text(itm.val))
		case itemExpressionOpen:
			// HACK: Find if it's a markup or expression, if it's markup, let the markup case handle it.
			if typ := p.peekNonWS().typ; typ == itemMarkupOpen || typ == itemMarkupClose {
				continue
			}

			expression, err := p.parseExpression()
			if err != nil {
				return nil, fmt.Errorf("parse expression: %w", err)
			}

			pattern = append(pattern, expression)
		case itemMarkupOpen, itemMarkupClose:
			markup, err := p.parseMarkup()
			if err != nil {
				return nil, fmt.Errorf("parse markup: %w", err)
			}

			pattern = append(pattern, markup)
		// bad tokens
		default:
			return nil, unexpectedErr(itm, itemWhitespace, itemText, itemExpressionOpen, itemMarkupOpen, itemMarkupClose)
		}
	}

	return pattern, nil
}

// --------------------------------Markup--------------------------------

func (p *parser) parseMarkup() (Markup, error) {
	var markup Markup

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return Markup{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemMarkupOpen:
			markup.Typ = Open
			markup.Identifier = p.parseIdentifier()

		case itemMarkupClose:
			if markup.Typ == Unspecified {
				markup.Typ = Close
				markup.Identifier = p.parseIdentifier()
			} else {
				markup.Typ = SelfClose
			}

		case itemOption:
			option, err := p.parseOption()
			if err != nil {
				return Markup{}, fmt.Errorf("parse option: %w", err)
			}

			markup.Options = append(markup.Options, option)

		case itemAttribute:
			attribute, err := p.parseAttribute()
			if err != nil {
				return Markup{}, fmt.Errorf("parse attribute: %w", err)
			}

			markup.Attributes = append(markup.Attributes, attribute)
		default:
			return Markup{}, unexpectedErr(itm, itemWhitespace, itemMarkupOpen, itemMarkupClose, itemOption, itemAttribute)
		}
	}

	return markup, nil
}

// ------------------------------Expression------------------------------

func (p *parser) parseExpression() (Expression, error) {
	var (
		expr Expression
		err  error
	)

	// optional operand - literal or variable

	switch itm := p.nextNonWS(); itm.typ {
	default:
		return Expression{},
			unexpectedErr(itm,
				itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
				itemFunction, itemPrivateStart, itemReservedStart, itemExpressionClose)
	case itemVariable:
		expr.Operand = Variable(itm.val)
	case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
		if expr.Operand, err = p.parseLiteral(); err != nil {
			return Expression{}, fmt.Errorf("parse literal: %w", err)
		}
	case itemFunction, itemPrivateStart, itemReservedStart:
		p.backup()
	case itemExpressionClose: // empty expression
		return expr, nil
	}

	if p.peekNonWS().typ == itemExpressionClose { // expression with operand only
		p.nextNonWS()
		return expr, nil
	}

	// ensure whitespace follows operand before annotation
	if expr.Operand != nil {
		if itm := p.next(); itm.typ != itemWhitespace {
			return Expression{}, unexpectedErr(itm, itemWhitespace)
		}
	}

	// parse annotation

	switch itm := p.next(); itm.typ {
	default:
		return Expression{}, unexpectedErr(itm, itemFunction, itemPrivateStart, itemReservedStart, itemAttribute)
	case itemFunction:
		if expr.Annotation, err = p.parseFunction(); err != nil {
			return Expression{}, fmt.Errorf("parse function: %w", err)
		}
	case itemReservedStart:
		if expr.Annotation, err = p.parseReservedAnnotation(); err != nil {
			return Expression{}, fmt.Errorf("parse private use annotation: %w", err)
		}
	case itemPrivateStart:
		if expr.Annotation, err = p.parsePrivateUseAnnotation(); err != nil {
			return Expression{}, fmt.Errorf("parse reserved annotation: %w", err)
		}
	case itemAttribute:
		if expr.Operand == nil {
			return Expression{}, errors.New("variable, literal or annotation is required before attribute")
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
		return Expression{}, unexpectedErr(itm, itemWhitespace)
	}

	// parse attributes

	if expr.Attributes, err = p.parseAttributes(); err != nil {
		return Expression{}, fmt.Errorf("parse attributes: %w", err)
	}

	if itm := p.nextNonWS(); itm.typ != itemExpressionClose {
		return Expression{}, unexpectedErr(itm, itemExpressionClose)
	}

	return expr, nil
}

// ------------------------------Annotation------------------------------

func (p *parser) parseFunction() (Function, error) {
	function := Function{Identifier: p.parseIdentifier()}

	if p.peekNonWS().typ == itemExpressionClose {
		return function, nil
	}

	// parse options

	for {
		if itm := p.next(); itm.typ != itemWhitespace {
			return Function{}, unexpectedErr(itm, itemWhitespace)
		}

		switch itm := p.next(); itm.typ {
		default:
			return Function{}, unexpectedErr(itm, itemOption, itemExpressionClose, itemAttribute)
		case itemOption:
			option, err := p.parseOption()
			if err != nil {
				return Function{}, fmt.Errorf("parse option: %w", err)
			}

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

func (p *parser) parsePrivateUseAnnotation() (PrivateUseAnnotation, error) {
	annotation := PrivateUseAnnotation{Start: rune(p.current().val[0])}

	switch itm := p.next(); itm.typ {
	default:
		return PrivateUseAnnotation{},
			unexpectedErr(itm, itemWhitespace, itemReservedText, itemQuotedLiteral, itemExpressionClose)
	case itemWhitespace: // noop
	case itemReservedText, itemQuotedLiteral:
		p.backup()
	case itemExpressionClose:
		p.backup()

		return annotation, nil
	}

	var err error

	if annotation.ReservedBody, err = p.parseReservedBody(); err != nil {
		return PrivateUseAnnotation{}, fmt.Errorf("parse reserve body: %w", err)
	}

	return annotation, nil
}

func (p *parser) parseReservedAnnotation() (ReservedAnnotation, error) {
	annotation := ReservedAnnotation{Start: rune(p.current().val[0])}

	switch itm := p.next(); itm.typ {
	default:
		return ReservedAnnotation{}, unexpectedErr(itm, itemWhitespace, itemExpressionClose)
	case itemWhitespace: // noop
	case itemReservedText, itemQuotedLiteral:
		p.backup()
	case itemExpressionClose:
		p.backup()

		return annotation, nil
	}

	var err error

	if annotation.ReservedBody, err = p.parseReservedBody(); err != nil {
		return ReservedAnnotation{}, fmt.Errorf("parse reserve body: %w", err)
	}

	return annotation, nil
}

func (p *parser) parseReservedBody() ([]ReservedBody, error) {
	var parts []ReservedBody

	for {
		switch itm := p.next(); itm.typ {
		default:
			return nil,
				unexpectedErr(itm, itemWhitespace, itemReservedText, itemQuotedLiteral, itemAttribute, itemExpressionClose)
		case itemWhitespace: // noop
		case itemReservedText:
			parts = append(parts, ReservedText(itm.val))
		case itemQuotedLiteral:
			parts = append(parts, QuotedLiteral(itm.val))
		case itemAttribute: // end of reserved body, attributes are next
			p.backup()
			p.backup()

			return parts, nil
		case itemExpressionClose:
			p.backup()

			return parts, nil
		}
	}
}

// ------------------------------Declaration------------------------------

func (p *parser) parseLocalDeclaration() (LocalDeclaration, error) {
	next := p.next()
	if next.typ != itemWhitespace {
		return LocalDeclaration{}, unexpectedErr(next, itemWhitespace)
	}

	if next = p.next(); next.typ != itemVariable {
		return LocalDeclaration{}, unexpectedErr(next, itemVariable)
	}

	declaration := LocalDeclaration{Variable: Variable(next.val)}

	if next = p.nextNonWS(); next.typ != itemOperator {
		return LocalDeclaration{}, unexpectedErr(next, itemOperator)
	}

	if next = p.nextNonWS(); next.typ != itemExpressionOpen {
		return LocalDeclaration{}, unexpectedErr(next, itemExpressionOpen)
	}

	expression, err := p.parseExpression()
	if err != nil {
		return LocalDeclaration{}, fmt.Errorf("parse expression: %w", err)
	}

	declaration.Expression = expression

	return declaration, nil
}

func (p *parser) parseInputDeclaration() (InputDeclaration, error) {
	next := p.nextNonWS()
	if next.typ != itemExpressionOpen {
		return InputDeclaration{}, unexpectedErr(next, itemExpressionOpen)
	}

	expression, err := p.parseExpression()
	if err != nil {
		return InputDeclaration{}, fmt.Errorf("parse expression: %w", err)
	}

	if _, ok := expression.Operand.(Variable); !ok {
		return InputDeclaration{}, fmt.Errorf("input declaration must have a variable as operand, got %T", expression.Operand)
	}

	return InputDeclaration(expression), nil
}

func (p *parser) parseReservedStatement() (ReservedStatement, error) {
	statement := ReservedStatement{Keyword: p.current().val}

	for {
		switch itm := p.nextNonWS(); itm.typ {
		// Ending tokens
		default:
			return ReservedStatement{}, unexpectedErr(itm, itemReservedText, itemQuotedLiteral, itemExpressionOpen)
		case itemError:
			return ReservedStatement{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemReservedKeyword, itemInputKeyword, itemLocalKeyword, // Another declaration
			itemQuotedPatternOpen, itemMatchKeyword: // End of declarations
			p.backup()
			return statement, nil
		// Non-ending tokens
		case itemReservedText:
			statement.ReservedBody = append(statement.ReservedBody, ReservedText(itm.val))
		case itemQuotedLiteral:
			statement.ReservedBody = append(statement.ReservedBody, QuotedLiteral(itm.val))
		case itemExpressionOpen:
			expression, err := p.parseExpression()
			if err != nil {
				return ReservedStatement{}, fmt.Errorf("parse expression: %w", err)
			}

			statement.Expressions = append(statement.Expressions, expression)
		}
	}
}

// ---------------------------------------------------------------------

func (p *parser) parseMatcher() (Matcher, error) {
	var matcher Matcher

	for itm := p.next(); itm.typ != itemEOF; itm = p.next() {
		switch itm.typ {
		case itemError:
			return Matcher{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemExpressionOpen:
			expression, err := p.parseExpression()
			if err != nil {
				return Matcher{}, fmt.Errorf("parse expression: %w", err)
			}

			matcher.MatchStatements = append(matcher.MatchStatements, expression)
		case itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			keys, err := p.parseVariantKeys()
			if err != nil {
				return Matcher{}, fmt.Errorf("parse variant keys: %w", err)
			}

			pattern, err := p.parsePattern()
			if err != nil {
				return Matcher{}, fmt.Errorf("parse pattern: %w", err)
			}

			matcher.Variants = append(matcher.Variants, Variant{Keys: keys, QuotedPattern: QuotedPattern(pattern)})
		// bad tokens
		default:
			return Matcher{},
				unexpectedErr(itm,
					itemWhitespace, itemExpressionOpen, itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral)
		}
	}

	p.backup()

	return matcher, nil
}

func (p *parser) parseVariantKeys() ([]VariantKey, error) {
	var keys []VariantKey

	for itm := p.current(); itm.typ != itemQuotedPatternOpen; itm = p.next() {
		switch itm.typ {
		case itemError:
			return nil, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemCatchAllKey:
			keys = append(keys, CatchAllKey{})
		case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			literal, err := p.parseLiteral()
			if err != nil {
				return nil, fmt.Errorf("parse literal: %w", err)
			}

			keys = append(keys, literal)
		// bad tokens
		default:
			return nil,
				unexpectedErr(itm, itemWhitespace, itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral)
		}
	}

	return keys, nil
}

func (p *parser) parseOption() (Option, error) {
	option := Option{Identifier: p.parseIdentifier()}

	// Next token must be an operator.
	if next := p.nextNonWS(); next.typ != itemOperator {
		return Option{}, unexpectedErr(next, itemOperator)
	}

	var err error

	// Next after operator must be a variable or literal.
	switch next := p.nextNonWS(); next.typ {
	default:
		err = unexpectedErr(next, itemVariable, itemQuotedLiteral, itemUnquotedLiteral, itemNumberLiteral)
	case itemError:
		err = fmt.Errorf("got error token: '%s'", next.val)
	case itemVariable:
		option.Value = Variable(next.val)
	case itemQuotedLiteral, itemUnquotedLiteral, itemNumberLiteral:
		option.Value, err = p.parseLiteral()
	}

	return option, err
}

func (p *parser) parseAttributes() ([]Attribute, error) {
	var attributes []Attribute

	for {
		if len(attributes) > 0 {
			if itm := p.next(); itm.typ != itemWhitespace {
				return nil, unexpectedErr(itm, itemWhitespace)
			}
		}

		switch itm := p.next(); itm.typ {
		default:
			return nil, unexpectedErr(itm, itemAttribute, itemExpressionClose)
		case itemAttribute:
			attribute, err := p.parseAttribute()
			if err != nil {
				return nil, fmt.Errorf("parse attribute: %w", err)
			}

			attributes = append(attributes, attribute)
		}

		if p.peekNonWS().typ == itemExpressionClose {
			return attributes, nil
		}
	}
}

func (p *parser) parseAttribute() (Attribute, error) {
	attribute := Attribute{Identifier: p.parseIdentifier()}

	switch itm := p.peekNonWS(); itm.typ {
	default:
		return Attribute{}, unexpectedErr(itm, itemAttribute, itemOperator, itemExpressionClose)
	case itemOperator:
		p.nextNonWS() // skip it
	case itemExpressionClose, itemAttribute:
		return attribute, nil
	}

	var err error

	switch itm := p.nextNonWS(); itm.typ {
	default:
		return Attribute{}, unexpectedErr(itm, itemVariable, itemQuotedLiteral, itemUnquotedLiteral, itemNumberLiteral)
	case itemVariable:
		attribute.Value = Variable(itm.val)
	case itemQuotedLiteral, itemUnquotedLiteral, itemNumberLiteral:
		if attribute.Value, err = p.parseLiteral(); err != nil {
			return Attribute{}, fmt.Errorf("parse literal: %w", err)
		}
	}

	return attribute, nil
}

func (p *parser) parseLiteral() (Literal, error) {
	switch itm := p.current(); itm.typ {
	default:
		return nil, unexpectedErr(itm, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral)
	case itemNumberLiteral:
		var num float64
		if err := json.Unmarshal([]byte(itm.val), &num); err != nil {
			return nil, fmt.Errorf("parse number literal: %w", err)
		}

		return NumberLiteral(num), nil
	case itemQuotedLiteral:
		return QuotedLiteral(p.current().val), nil
	case itemUnquotedLiteral:
		return NameLiteral(p.current().val), nil
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
//
// TODO(jhorsts): exposed fields should not use private types.
type UnexpectedTokenError struct {
	Expected []itemType
	Actual   item
}

func (u UnexpectedTokenError) Error() string {
	if len(u.Expected) == 0 {
		return fmt.Sprintf("expected no items, got '%s'", u.Actual)
	}

	r := u.Expected[0].String()
	for _, typ := range u.Expected[1:] {
		r += ", " + typ.String()
	}

	return fmt.Sprintf("expected item: %s, got %s", r, u.Actual)
}

func unexpectedErr(actual item, expected ...itemType) UnexpectedTokenError {
	return UnexpectedTokenError{
		Actual:   actual,
		Expected: expected,
	}
}
