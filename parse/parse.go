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
			return errors.New(itm.String())
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
	errorf := func(format string, err error) (AST, error) {
		if errors.Is(err, mf2.ErrDuplicateDeclaration) {
			return AST{}, fmt.Errorf("parse MF2: "+format, err)
		}

		// fallback to syntax error unless one of MF2 errors is returned
		return AST{}, fmt.Errorf("parse MF2: %w: "+format, mf2.ErrSyntax, err)
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

	ast := AST{Message: message}
	if err := ast.validate(); err != nil {
		return errorf("validate: %w", err)
	}

	return ast, nil
}

// ------------------------------Message------------------------------

func (p *parser) parseSimpleMessage() (SimpleMessage, error) {
	pattern, err := p.parsePattern()
	if err != nil {
		return SimpleMessage{}, fmt.Errorf("simple message: %w", err)
	}

	return SimpleMessage(pattern), nil
}

// getVariables returns all variables used in the given expression.
func getVariables(e Expression, includeOperand bool) []Variable {
	var variables []Variable

	if includeOperand {
		if variable, ok := e.Operand.(Variable); ok {
			variables = append(variables, variable)
		}
	}

	if function, ok := e.Annotation.(Function); ok {
		for _, option := range function.Options {
			if value, ok := option.Value.(Variable); ok {
				variables = append(variables, value)
			}
		}
	}

	for _, a := range e.Attributes {
		if variable, ok := a.Value.(Variable); ok {
			variables = append(variables, variable)
		}
	}

	return variables
}

type variableDeclarations []Variable

// add adds a declared variable and implicitly declares variables if they never were declared before.
func (d *variableDeclarations) add(variable Variable, variables []Variable) error {
	// a declared variable is NOT available for use in the expression variables
	if slices.Contains(variables, variable) {
		return fmt.Errorf(`%w: %s`, mf2.ErrDuplicateDeclaration, variable)
	}

	// ensure a declared variable has not been declared previously
	for _, v := range *d {
		if v == variable {
			return fmt.Errorf(`%w: %s`, mf2.ErrDuplicateDeclaration, variable)
		}
	}

	*d = append(*d, variable) // add a declared variable

	// add expression variable implicitly
	for _, v := range variables {
		if !slices.Contains(*d, v) {
			*d = append(*d, variables...)
		}
	}

	return nil
}

func (p *parser) parseComplexMessage() (ComplexMessage, error) {
	var (
		message   ComplexMessage
		variables variableDeclarations
	)

	errorf := func(format string, args ...any) (ComplexMessage, error) {
		return ComplexMessage{}, fmt.Errorf("complex message: "+format, args...)
	}

	for {
		switch itm := p.nextNonWS(); itm.typ {
		// Ending tokens
		default:
			err := unexpectedErr(
				itm, itemInputKeyword, itemLocalKeyword, itemReservedKeyword, itemMatchKeyword, itemQuotedPatternOpen)
			return errorf("%w", err)
		case itemError:
			return errorf("%s", itm)
		case itemEOF:
			return message, nil
		// Non-ending tokens
		case itemInputKeyword:
			declaration, err := p.parseInputDeclaration()
			if err != nil {
				return errorf("%w", err)
			}

			if err = variables.add( //nolint:forcetypeassert
				declaration.Operand.(Variable), // Operand is always Variable for InputDeclaration
				getVariables(Expression(declaration), false),
			); err != nil {
				return errorf("%w", err)
			}

			message.Declarations = append(message.Declarations, declaration)
		case itemLocalKeyword:
			declaration, err := p.parseLocalDeclaration()
			if err != nil {
				return errorf("%w", err)
			}

			if err = variables.add(declaration.Variable, getVariables(declaration.Expression, true)); err != nil {
				return errorf("%w", err)
			}

			message.Declarations = append(message.Declarations, declaration)
		case itemReservedKeyword:
			declaration, err := p.parseReservedStatement()
			if err != nil {
				return errorf("%w", err)
			}

			message.Declarations = append(message.Declarations, declaration)
		case itemMatchKeyword:
			matcher, err := p.parseMatcher()
			if err != nil {
				return errorf("%w", err)
			}

			message.ComplexBody = matcher
		case itemQuotedPatternOpen:
			pattern, err := p.parsePattern()
			if err != nil {
				return errorf("%w", err)
			}

			message.ComplexBody = QuotedPattern(pattern)
		}
	}
}

// ------------------------------Pattern------------------------------

// parsePattern parses a slice of pattern parts.
func (p *parser) parsePattern() ([]PatternPart, error) {
	var pattern []PatternPart

	errorf := func(format string, args ...any) ([]PatternPart, error) {
		return nil, fmt.Errorf("pattern: "+format, args...)
	}

	// Loop until the end, or closing pattern quote, if parsing complex message.
	for itm := p.next(); itm.typ != itemEOF && itm.typ != itemQuotedPatternClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return errorf("%s", itm)
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
				return errorf("%w", err)
			}

			pattern = append(pattern, expression)
		case itemMarkupOpen, itemMarkupClose:
			markup, err := p.parseMarkup()
			if err != nil {
				return errorf("%w", err)
			}

			pattern = append(pattern, markup)
		// bad tokens
		default:
			err := unexpectedErr(itm, itemWhitespace, itemText, itemExpressionOpen, itemMarkupOpen, itemMarkupClose)
			return errorf("%w", err)
		}
	}

	return pattern, nil
}

// --------------------------------Markup--------------------------------

func (p *parser) parseMarkup() (Markup, error) {
	var markup Markup

	errorf := func(format string, args ...any) (Markup, error) {
		return Markup{}, fmt.Errorf("markup: "+format, args...)
	}

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return errorf("%s", itm)
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
				return errorf("%w", err)
			}

			markup.Options = append(markup.Options, option)
		case itemAttribute:
			attribute, err := p.parseAttribute()
			if err != nil {
				return errorf("%w", err)
			}

			markup.Attributes = append(markup.Attributes, attribute)
		default:
			err := unexpectedErr(itm, itemWhitespace, itemMarkupOpen, itemMarkupClose, itemOption, itemAttribute)
			return errorf("%w", err)
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

	errorf := func(format string, args ...any) (Expression, error) {
		return Expression{}, fmt.Errorf("expression: "+format, args...)
	}

	// optional operand - literal or variable

	switch itm := p.nextNonWS(); itm.typ {
	default:
		err = unexpectedErr(itm,
			itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
			itemFunction, itemPrivateStart, itemReservedStart, itemExpressionClose)

		return errorf("%w", err)
	case itemVariable:
		expr.Operand = Variable(itm.val)
	case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
		if expr.Operand, err = p.parseLiteral(); err != nil {
			return errorf("%w", err)
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
			return errorf("between operand and annotation: %w", unexpectedErr(itm, itemWhitespace))
		}
	}

	// parse annotation

	switch itm := p.next(); itm.typ {
	default:
		return errorf("%w", unexpectedErr(itm, itemFunction, itemPrivateStart, itemReservedStart, itemAttribute))
	case itemFunction:
		if expr.Annotation, err = p.parseFunction(); err != nil {
			return errorf("%w", err)
		}
	case itemReservedStart:
		if expr.Annotation, err = p.parseReservedAnnotation(); err != nil {
			return errorf("%w", err)
		}
	case itemPrivateStart:
		if expr.Annotation, err = p.parsePrivateUseAnnotation(); err != nil {
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

	errorf := func(format string, args ...any) (Function, error) {
		return Function{}, fmt.Errorf("function: "+format, args...)
	}

	// parse options

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
	errorf := func(format string, args ...any) (PrivateUseAnnotation, error) {
		return PrivateUseAnnotation{}, fmt.Errorf("private use annotation: "+format, args...)
	}

	switch itm := p.next(); itm.typ {
	default:
		err := unexpectedErr(itm, itemWhitespace, itemReservedText, itemQuotedLiteral, itemExpressionClose)
		return errorf("%w", err)
	case itemWhitespace: // noop
	case itemReservedText, itemQuotedLiteral:
		p.backup()
	case itemExpressionClose:
		p.backup()

		return annotation, nil
	}

	var err error

	if annotation.ReservedBody, err = p.parseReservedBody(); err != nil {
		return errorf("%w", err)
	}

	return annotation, nil
}

func (p *parser) parseReservedAnnotation() (ReservedAnnotation, error) {
	annotation := ReservedAnnotation{Start: rune(p.current().val[0])}
	errorf := func(format string, args ...any) (ReservedAnnotation, error) {
		return ReservedAnnotation{}, fmt.Errorf("reserved annotation: "+format, args...)
	}

	switch itm := p.next(); itm.typ {
	default:
		return errorf("%w", unexpectedErr(itm, itemWhitespace, itemExpressionClose))
	case itemWhitespace: // noop
	case itemReservedText, itemQuotedLiteral:
		p.backup()
	case itemExpressionClose:
		p.backup()

		return annotation, nil
	}

	var err error

	if annotation.ReservedBody, err = p.parseReservedBody(); err != nil {
		return errorf("%w", err)
	}

	return annotation, nil
}

func (p *parser) parseReservedBody() ([]ReservedBody, error) {
	var parts []ReservedBody

	for {
		switch itm := p.next(); itm.typ {
		default:
			err := unexpectedErr(itm, itemWhitespace, itemReservedText, itemQuotedLiteral, itemAttribute, itemExpressionClose)
			return nil, fmt.Errorf("reserved body: %w", err)
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
	errorf := func(err error) (LocalDeclaration, error) {
		return LocalDeclaration{}, fmt.Errorf("local declaration: %w", err)
	}

	next := p.next()
	if next.typ != itemWhitespace {
		return errorf(unexpectedErr(next, itemWhitespace))
	}

	if next = p.next(); next.typ != itemVariable {
		return errorf(unexpectedErr(next, itemVariable))
	}

	variable := Variable(next.val)

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
	errorf := func(format string, args ...any) (InputDeclaration, error) {
		return InputDeclaration{}, fmt.Errorf("input declaration: "+format, args...)
	}

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

func (p *parser) parseReservedStatement() (ReservedStatement, error) {
	statement := ReservedStatement{Keyword: p.current().val}
	errorf := func(format string, args ...any) (ReservedStatement, error) {
		return ReservedStatement{}, fmt.Errorf("reserved statement: "+format, args...)
	}

	for {
		switch itm := p.nextNonWS(); itm.typ {
		// Ending tokens
		default:
			return errorf("%w", unexpectedErr(itm, itemReservedText, itemQuotedLiteral, itemExpressionOpen))
		case itemError:
			return errorf("%s", itm)
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
				return errorf("%w", err)
			}

			statement.Expressions = append(statement.Expressions, expression)
		}
	}
}

// ---------------------------------------------------------------------

func (p *parser) parseMatcher() (Matcher, error) {
	var matcher Matcher

	errorf := func(format string, args ...any) (Matcher, error) {
		return Matcher{}, fmt.Errorf("matcher: "+format, args...)
	}

	for itm := p.next(); itm.typ != itemEOF; itm = p.next() {
		switch itm.typ {
		case itemError:
			return errorf("%s", itm)
		case itemWhitespace:
			continue
		case itemExpressionOpen:
			expression, err := p.parseExpression()
			if err != nil {
				return errorf("%w", err)
			}

			matcher.MatchStatements = append(matcher.MatchStatements, expression)
		case itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			keys, err := p.parseVariantKeys()
			if err != nil {
				return errorf("%w", err)
			}

			pattern, err := p.parsePattern()
			if err != nil {
				return errorf("%w", err)
			}

			matcher.Variants = append(matcher.Variants, Variant{Keys: keys, QuotedPattern: QuotedPattern(pattern)})
		// bad tokens
		default:
			err := unexpectedErr(itm, itemWhitespace, itemExpressionOpen, itemCatchAllKey,
				itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral)
			return errorf("%w", err)
		}
	}

	p.backup()

	return matcher, nil
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
		case itemError:
			return errorf("%s", itm)
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
		// bad tokens
		default:
			err := unexpectedErr(itm, itemWhitespace, itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral)
			return errorf("%w", err)
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
	case itemError:
		return errorf("%s", next)
	case itemVariable:
		option.Value = Variable(next.val)
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
		return nil, fmt.Errorf("attribute at %d: "+format, append([]any{len(attributes)}, args)...)
	}

	for {
		if len(attributes) > 0 {
			if itm := p.next(); itm.typ != itemWhitespace {
				return errorf("%w", unexpectedErr(itm, itemWhitespace))
			}
		}

		switch itm := p.next(); itm.typ {
		default:
			return errorf("%w", unexpectedErr(itm, itemAttribute, itemExpressionClose))
		case itemAttribute:
			attribute, err := p.parseAttribute()
			if err != nil {
				return errorf("%w", err)
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
		attribute.Value = Variable(itm.val)
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
		return "want no items, got " + u.Actual.String()
	}

	r := u.Expected[0].String()
	for _, typ := range u.Expected[1:] {
		r += ", " + typ.String()
	}

	return "want item " + r + ", got " + u.Actual.String()
}

func unexpectedErr(actual item, expected ...itemType) UnexpectedTokenError {
	return UnexpectedTokenError{
		Actual:   actual,
		Expected: expected,
	}
}
