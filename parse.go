package mf2

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

func (p *parser) next() item {
	if p.pos == len(p.items)-1 {
		return mk(itemError, "no more tokens")
	}

	p.pos++

	return p.items[p.pos]
}

func (p *parser) current() item {
	return p.items[p.pos]
}

func (p *parser) collect() error {
	// sanity check, avoid infinite loop
	for i := 0; i < 1000; i++ {
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

/*
Parse parses the input string and returns an AST tree of MessageFormat2.
Empty input string returns a SimpleMessage with no patterns.

Examples:

	mf2.Parse("Hello World!")
	// result
	AST{Message: SimpleMessage{Patterns: []Pattern{TextPattern("Hello World!")}}}

	// -----------------------------------------------------------

	mf2.Parse("Hello {name}!")
	// result
	AST{
		Message: SimpleMessage{
			Patterns: []Pattern{
				TextPattern("Hello "),
				PlaceholderPattern{Expression: VariableExpression{Variable: "name"}},
			},
		},
	}

	// -----------------------------------------------------------

	mf2.Parse(".match {$count} 1 {{Hello world}} * {{Hello worlds}}")
	// result
	AST{
		Message: ComplexMessage{
			ComplexBody: Matcher{
				MatchStatements: []Expression{VariableExpression{Variable: "count"}},
				Variants: []Variant{
					{
						Key: LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral(1)}},
						QuotedPattern: QuotedPattern{
							Patterns: []Pattern{TextPattern("Hello world")},
						},
					},
					{
						Key: WildcardKey{},
						QuotedPattern: QuotedPattern{
							Patterns: []Pattern{TextPattern("Hello worlds")},
						},
					},
				},
			},
		},
	}
*/
func Parse(input string) (AST, error) {
	p := &parser{lexer: lex(input)}
	if err := p.collect(); err != nil {
		return AST{}, fmt.Errorf("collect tokens: %w", err)
	}

	if len(p.items) == 1 && p.items[0].typ == itemEOF {
		return AST{}, nil
	}

	message, err := p.parseMessage()
	if err != nil {
		return AST{}, fmt.Errorf("parse message: %w", err)
	}

	ast := AST{Message: message}
	if err := ast.validate(); err != nil {
		return AST{}, fmt.Errorf("validate AST: %w", err)
	}

	return ast, nil
}

// ------------------------------Message------------------------------

// parseMessage parses message by its type.
func (p *parser) parseMessage() (Message, error) { //nolint:ireturn
	if p.isComplexMessage() {
		message, err := p.parseComplexMessage()
		if err != nil {
			return nil, fmt.Errorf("parse complex message: %w", err)
		}

		return message, nil
	}

	message, err := p.parseSimpleMessage()
	if err != nil {
		return nil, fmt.Errorf("parse simple message: %w", err)
	}

	return message, nil
}

func (p *parser) parseSimpleMessage() (SimpleMessage, error) {
	patterns, err := p.parsePatterns()
	if err != nil {
		return SimpleMessage{}, fmt.Errorf("parse patterns: %w", err)
	}

	return SimpleMessage{Patterns: patterns}, nil
}

func (p *parser) parseComplexMessage() (ComplexMessage, error) {
	var declarations []Declaration

	for itm := p.current(); p.current().typ != itemEOF; itm = p.next() {
		switch itm.typ {
		case itemError:
			return ComplexMessage{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue

		case itemInputKeyword, itemLocalKeyword, itemReservedKeyword: // Declarations
			declaration, err := p.parseDeclaration()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse declaration: %w", err)
			}

			declarations = append(declarations, declaration)

		case itemMatchKeyword: // Matcher
			p.next() // skip keyword

			matcher, err := p.parseMatcher()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse matcher: %w", err)
			}

			return ComplexMessage{Declarations: declarations, ComplexBody: matcher}, nil

		case itemQuotedPatternOpen: // Zero or more Declarations + QuotedPattern
			p.next() // skip opening quote

			patterns, err := p.parsePatterns()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse patterns: %w", err)
			}

			return ComplexMessage{Declarations: declarations, ComplexBody: QuotedPattern{Patterns: patterns}}, nil
		// bad tokens
		case itemEOF, itemVariable, itemFunction, itemExpressionOpen,
			itemExpressionClose, itemQuotedPatternClose, itemText, itemCatchAllKey,
			itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
			itemOption, itemReserved, itemOperator, itemPrivate:
			err := UnexpectedTokenError{
				Expected: []itemType{
					itemWhitespace, itemInputKeyword, itemLocalKeyword,
					itemMatchKeyword, itemReservedKeyword, itemQuotedPatternOpen,
				},
				Actual: itm.typ,
			}

			return ComplexMessage{}, err
		}
	}

	return ComplexMessage{}, errors.New("no complex body found")
}

// ------------------------------Pattern------------------------------

// parsePatterns parses a slice of patterns.
func (p *parser) parsePatterns() ([]Pattern, error) {
	var pattern []Pattern

	// Loop until the end, or closing pattern quote, if parsing complex message.
	for itm := p.current(); itm.typ != itemEOF && itm.typ != itemQuotedPatternClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return nil, fmt.Errorf("got error token: '%s'", itm.val)
		case itemText:
			pattern = append(pattern, TextPattern(itm.val))
		case itemExpressionOpen:
			p.next() // skip opening brace

			expression, err := p.parseExpression()
			if err != nil {
				return nil, fmt.Errorf("parse expression: %w", err)
			}

			pattern = append(pattern, PlaceholderPattern{Expression: expression})
		// bad tokens
		case itemEOF, itemVariable, itemFunction,
			itemExpressionClose, itemQuotedPatternOpen,
			itemQuotedPatternClose, itemCatchAllKey,
			itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
			itemOption, itemWhitespace, itemReserved,
			itemOperator, itemPrivate:
			return nil, UnexpectedTokenError{Expected: []itemType{itemText, itemExpressionOpen}, Actual: itm.typ}
		}
	}

	return pattern, nil
}

// ------------------------------Expression------------------------------

// parseExpression parses expression by its type.
func (p *parser) parseExpression() (Expression, error) { //nolint:ireturn
	for itm := p.current(); p.current().typ != itemExpressionClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return nil, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemVariable: // Variable expression
			expression, err := p.parseVariableExpression()
			if err != nil {
				return nil, fmt.Errorf("parse variable expression: %w", err)
			}

			return expression, nil
		case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral: // Literal expression
			expression, err := p.parseLiteralExpression()
			if err != nil {
				return nil, fmt.Errorf("parse literal expression: %w", err)
			}

			return expression, nil
		case itemFunction: // Annotation expression
			annotation, err := p.parseAnnotation()
			if err != nil {
				return nil, fmt.Errorf("parse annotation expression: %w", err)
			}

			return AnnotationExpression{Annotation: annotation}, nil
		// bad tokens
		case itemEOF, itemExpressionOpen, itemExpressionClose,
			itemQuotedPatternOpen, itemQuotedPatternClose,
			itemText, itemCatchAllKey,
			itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemOption, itemReserved,
			itemOperator, itemPrivate:
			err := UnexpectedTokenError{
				Expected: []itemType{
					itemWhitespace, itemVariable, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral, itemFunction,
				},
				Actual: itm.typ,
			}

			return nil, err
		}
	}

	return nil, errors.New("no expression start found")
}

func (p *parser) parseVariableExpression() (VariableExpression, error) {
	var (
		variable      Variable
		foundVariable bool // flag to check if variable is already found
	)

	for itm := p.current(); p.current().typ != itemExpressionClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return VariableExpression{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemVariable:
			if foundVariable {
				return VariableExpression{}, errors.New("expression contains more than one variable")
			}

			foundVariable = true
			variable = Variable(itm.val[1:]) // omit "$" prefix //TODO: Lexer should not capture variable prefix
		case itemFunction, itemPrivate, itemReserved:
			// Variable expression with annotation.
			annotation, err := p.parseAnnotation()
			if err != nil {
				return VariableExpression{}, fmt.Errorf("parse annotation: %w", err)
			}

			return VariableExpression{Variable: variable, Annotation: annotation}, nil
		// bad tokens
		case itemEOF, itemExpressionOpen, itemExpressionClose,
			itemQuotedPatternOpen, itemQuotedPatternClose,
			itemText, itemCatchAllKey,
			itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
			itemOption, itemOperator:
			err := UnexpectedTokenError{
				Expected: []itemType{itemWhitespace, itemVariable, itemFunction, itemPrivate, itemReserved},
				Actual:   itm.typ,
			}

			return VariableExpression{}, err
		}
	}

	// Variable expression without annotation.
	return VariableExpression{Variable: variable}, nil
}

func (p *parser) parseLiteralExpression() (LiteralExpression, error) {
	var (
		literal      Literal
		foundLiteral bool // flag to check if literal is already found
	)

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return LiteralExpression{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			if foundLiteral {
				return LiteralExpression{}, errors.New("expression contains more than one literal")
			}

			foundLiteral = true

			var err error

			literal, err = p.parseLiteral()
			if err != nil {
				return LiteralExpression{}, fmt.Errorf("parse literal: %w", err)
			}
		case itemFunction:
			// Literal expression with annotation.
			annotation, err := p.parseAnnotation()
			if err != nil {
				return LiteralExpression{}, fmt.Errorf("parse annotation: %w", err)
			}

			return LiteralExpression{Literal: literal, Annotation: annotation}, nil
		// bad tokens
		case itemEOF, itemVariable,
			itemExpressionOpen, itemExpressionClose,
			itemQuotedPatternOpen, itemQuotedPatternClose,
			itemText, itemCatchAllKey,
			itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemOption, itemReserved,
			itemOperator, itemPrivate:
			err := UnexpectedTokenError{
				Expected: []itemType{itemWhitespace, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral, itemFunction},
				Actual:   itm.typ,
			}

			return LiteralExpression{}, err
		}
	}

	// Literal expression without annotation.
	return LiteralExpression{Literal: literal}, nil
}

// ------------------------------Annotation------------------------------

// parseAnnotation parses annotation by its type.
func (p *parser) parseAnnotation() (Annotation, error) { //nolint:ireturn
	switch p.current().typ {
	case itemFunction:
		annotation, err := p.parseFunctionAnnotation()
		if err != nil {
			return nil, fmt.Errorf("parse function annotation: %w", err)
		}

		return annotation, nil
	case itemPrivate:
		annotation, err := p.parsePrivateUseAnnotation()
		if err != nil {
			return nil, fmt.Errorf("parse private use annotation: %w", err)
		}

		return annotation, nil
	case itemReserved:
		annotation, err := p.parseReservedAnnotation()
		if err != nil {
			return nil, fmt.Errorf("parse reserved annotation: %w", err)
		}

		return annotation, nil
		// bad tokens
	case itemError, itemEOF, itemVariable,
		itemExpressionOpen, itemExpressionClose,
		itemQuotedPatternOpen, itemQuotedPatternClose,
		itemText, itemCatchAllKey,
		itemInputKeyword, itemLocalKeyword,
		itemMatchKeyword, itemReservedKeyword,
		itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
		itemOption, itemWhitespace, itemOperator:
		err := UnexpectedTokenError{
			Expected: []itemType{itemFunction, itemPrivate, itemReserved},
			Actual:   p.current().typ,
		}

		return nil, err
	}

	return nil, fmt.Errorf("unknown token: %s", p.current().typ)
}

func (p *parser) parseFunctionAnnotation() (FunctionAnnotation, error) {
	var annotation FunctionAnnotation

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return FunctionAnnotation{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemFunction:
			annotation.Function = p.parseFunction()
		case itemOption:
			// Function with options
			option, err := p.parseOption()
			if err != nil {
				return FunctionAnnotation{}, fmt.Errorf("parse option: %w", err)
			}

			annotation.Options = append(annotation.Options, option)
		// bad tokens
		case itemEOF, itemVariable,
			itemExpressionOpen, itemExpressionClose,
			itemQuotedPatternOpen, itemQuotedPatternClose,
			itemText, itemCatchAllKey,
			itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
			itemReserved, itemOperator, itemPrivate:
			err := UnexpectedTokenError{
				Expected: []itemType{itemWhitespace, itemFunction, itemOption},
				Actual:   itm.typ,
			}

			return FunctionAnnotation{}, err
		}
	}

	// Function without options
	return annotation, nil
}

func (p *parser) parsePrivateUseAnnotation() (PrivateUseAnnotation, error) {
	// TODO: implement
	return PrivateUseAnnotation{}, errors.New("not implemented")
}

func (p *parser) parseReservedAnnotation() (ReservedAnnotation, error) {
	// TODO: implement
	return ReservedAnnotation{}, errors.New("not implemented")
}

// ------------------------------Declaration------------------------------

// parseDeclaration parses declaration by its type.
func (p *parser) parseDeclaration() (Declaration, error) { //nolint:ireturn
	switch p.current().typ {
	case itemInputKeyword:
		p.next() // skip keyword

		declaration, err := p.parseInputDeclaration()
		if err != nil {
			return nil, fmt.Errorf("parse input declaration: %w", err)
		}

		return declaration, nil
	case itemLocalKeyword:
		p.next() // skip keyword

		declaration, err := p.parseLocalDeclaration()
		if err != nil {
			return nil, fmt.Errorf("parse local declaration: %w", err)
		}

		return declaration, nil
	case itemReservedKeyword:
		p.next() // skip keyword

		declaration, err := p.parseReservedStatement()
		if err != nil {
			return nil, fmt.Errorf("parse reserved statement: %w", err)
		}

		return declaration, nil
	// bad tokens
	case itemError, itemEOF, itemVariable, itemFunction,
		itemExpressionOpen, itemExpressionClose,
		itemQuotedPatternOpen, itemQuotedPatternClose,
		itemText, itemMatchKeyword,
		itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
		itemOption, itemWhitespace, itemReserved,
		itemOperator, itemPrivate:
		return nil, UnexpectedTokenError{
			Expected: []itemType{itemInputKeyword, itemLocalKeyword, itemReservedKeyword},
			Actual:   p.current().typ,
		}
	}

	return nil, fmt.Errorf("unknown token: %s", p.current().typ)
}

func (p *parser) parseLocalDeclaration() (LocalDeclaration, error) {
	var (
		variable      Variable
		foundVariable bool // flag to check if variable is already found
	)

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return LocalDeclaration{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace, itemOperator:
			continue
		case itemVariable:
			if foundVariable {
				return LocalDeclaration{}, errors.New("local declaration contains more than one variable")
			}

			foundVariable = true
			variable = Variable(itm.val[1:]) // omit "$" prefix //TODO: Lexer should not capture variable prefix
		case itemExpressionOpen:
			p.next() // skip opening brace

			expression, err := p.parseExpression()
			if err != nil {
				return LocalDeclaration{}, fmt.Errorf("parse expression: %w", err)
			}

			return LocalDeclaration{Variable: variable, Expression: expression}, nil
		// bad tokens
		case itemEOF, itemFunction,
			itemExpressionClose, itemQuotedPatternOpen, itemQuotedPatternClose,
			itemText, itemCatchAllKey,
			itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
			itemOption, itemReserved, itemPrivate:
			err := UnexpectedTokenError{
				Expected: []itemType{itemWhitespace, itemOperator, itemVariable, itemExpressionOpen},
				Actual:   itm.typ,
			}

			return LocalDeclaration{}, err
		}
	}

	return LocalDeclaration{}, errors.New("no expression found start")
}

func (p *parser) parseInputDeclaration() (InputDeclaration, error) {
	// TODO: implement
	return InputDeclaration{}, errors.New("not implemented")
}

func (p *parser) parseReservedStatement() (ReservedStatement, error) {
	// TODO: implement.
	return ReservedStatement{}, errors.New("not implemented")
}

// ---------------------------------------------------------------------

func (p *parser) parseMatcher() (Matcher, error) {
	var matcher Matcher

	for itm := p.current(); itm.typ != itemEOF; itm = p.next() {
		switch itm.typ {
		case itemError:
			return Matcher{}, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace:
			continue
		case itemExpressionOpen:
			p.next() // skip opening brace

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

			p.next() // skip opening quoted pattern

			patterns, err := p.parsePatterns()
			if err != nil {
				return Matcher{}, fmt.Errorf("parse patterns: %w", err)
			}

			matcher.Variants = append(matcher.Variants, Variant{Keys: keys, QuotedPattern: QuotedPattern{Patterns: patterns}})
		// bad tokens
		case itemEOF, itemVariable, itemFunction,
			itemExpressionClose, itemQuotedPatternOpen, itemQuotedPatternClose,
			itemText, itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemOption, itemReserved,
			itemOperator, itemPrivate:
			err := UnexpectedTokenError{
				Expected: []itemType{
					itemWhitespace, itemExpressionOpen, itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
				},
				Actual: itm.typ,
			}

			return Matcher{}, err
		}
	}

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

			keys = append(keys, LiteralKey{Literal: literal})
		// bad tokens
		case itemEOF, itemVariable, itemFunction,
			itemExpressionOpen, itemExpressionClose,
			itemQuotedPatternOpen, itemQuotedPatternClose,
			itemText, itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemOption, itemReserved,
			itemOperator, itemPrivate:
			err := UnexpectedTokenError{
				Expected: []itemType{itemWhitespace, itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral},
				Actual:   itm.typ,
			}

			return nil, err
		}
	}

	return keys, nil
}

func (p *parser) parseOption() (Option, error) { //nolint:ireturn
	var identifier Identifier

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		switch itm.typ {
		case itemError:
			return nil, fmt.Errorf("got error token: '%s'", itm.val)
		case itemWhitespace, itemOperator:
			continue
		case itemOption:
			identifier = p.parseIdentifier()

		case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			literal, err := p.parseLiteral()
			if err != nil {
				return nil, fmt.Errorf("parse literal: %w", err)
			}

			return LiteralOption{Literal: literal, Identifier: identifier}, nil

		case itemVariable:
			return VariableOption{
				Variable:   Variable(p.current().val[1:]), // omit "$" prefix //TODO: Lexer should not capture variable prefix
				Identifier: identifier,
			}, nil
		// bad tokens
		case itemEOF, itemFunction,
			itemExpressionOpen, itemExpressionClose,
			itemQuotedPatternOpen, itemQuotedPatternClose,
			itemText, itemCatchAllKey,
			itemInputKeyword, itemLocalKeyword,
			itemMatchKeyword, itemReservedKeyword,
			itemReserved, itemPrivate:
			err := UnexpectedTokenError{
				Expected: []itemType{
					itemWhitespace, itemOperator, itemOption, itemVariable,
					itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral,
				},
				Actual: itm.typ,
			}

			return nil, err
		}
	}

	return nil, errors.New("no option value found")
}

func (p *parser) parseLiteral() (Literal, error) { //nolint:ireturn
	switch itm := p.current(); itm.typ {
	case itemNumberLiteral:
		var num float64
		if err := json.Unmarshal([]byte(itm.val), &num); err != nil {
			return nil, fmt.Errorf("parse number literal: %w", err)
		}

		return UnquotedLiteral{Value: NumberLiteral(num)}, nil
	case itemQuotedLiteral:
		return QuotedLiteral(p.current().val), nil
	case itemUnquotedLiteral:
		return UnquotedLiteral{Value: NameLiteral(p.current().val)}, nil
	// bad tokens
	case itemError, itemEOF, itemVariable, itemFunction,
		itemExpressionOpen, itemExpressionClose,
		itemQuotedPatternOpen, itemQuotedPatternClose,
		itemText, itemCatchAllKey,
		itemInputKeyword, itemLocalKeyword,
		itemMatchKeyword, itemReservedKeyword,
		itemOption, itemWhitespace, itemReserved,
		itemOperator, itemPrivate:
		err := UnexpectedTokenError{
			Expected: []itemType{itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral},
			Actual:   itm.typ,
		}

		return nil, err
	}

	return nil, fmt.Errorf("unknown token: %s", p.current().typ)
}

func (p *parser) parseFunction() Function {
	return Function{Prefix: rune(p.current().val[0]), Identifier: p.parseIdentifier()}
}

func (p *parser) parseIdentifier() Identifier {
	full := strings.Split(p.current().val, ":")

	var (
		ns   string
		name string
	)

	//nolint:gomnd
	switch len(full) {
	// no namespace
	case 1:
		name = full[0]
	// namespace + name
	case 2:
		ns = full[0]
		name = full[1]
	// edge case for ":namespace:function"
	case 3:
		ns = full[1]
		name = full[2]
	}

	return Identifier{Namespace: ns, Name: name}
}

// ------------------------------Helpers------------------------------

// isComplexMessage returns true if first token is one of the complex message tokens.
func (p *parser) isComplexMessage() bool {
	//nolint:exhaustive
	switch p.items[0].typ {
	case itemInputKeyword, itemLocalKeyword, itemMatchKeyword, itemReservedKeyword, itemQuotedPatternOpen:
		return true
	}

	return false
}

// UnexpectedTokenError is returned when parser encounters unexpected token.
// It contains information about expected token types and actual token type.
type UnexpectedTokenError struct {
	Expected []itemType
	Actual   itemType
}

func (u UnexpectedTokenError) Error() string {
	if len(u.Expected) == 0 {
		return fmt.Sprintf("expected no tokens, got '%s'", u.Actual)
	}

	r := u.Expected[0].String()
	for _, typ := range u.Expected[1:] {
		r += ", " + typ.String()
	}

	return fmt.Sprintf("unexpected token: expected one of [%s], got '%s'", r, u.Actual)
}
