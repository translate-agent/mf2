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
	if p.pos >= len(p.items) {
		// todo: error or check elsewhere
		return item{}
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
	SimpleMessage{Patterns: []Pattern{TextPattern("Hello World!")}}

	// -----------------------------------------------------------

	mf2.Parse("Hello {name}!")
	// result
	SimpleMessage{
		Patterns: []Pattern{
			TextPattern("Hello "),
			PlaceholderPattern{Expression: VariableExpression{Variable: "name"}},
		},
	}

	// -----------------------------------------------------------

	mf2.Parse(".match {$count} 1 {{Hello world}} * {{Hello worlds}}")
	// result
	ComplexMessage{
		ComplexBody: Matcher{
			MatchStatements: []Expression{
				VariableExpression{Variable: "count"},
			},
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

	return AST{Message: p.parseMessage()}, nil
}

// ------------------------------Message------------------------------

// parseMessage determines message type and then parses it accordingly.
// TODO: parse error handling.
func (p *parser) parseMessage() Message { //nolint:ireturn
	if typ := p.items[0].typ; typ == itemKeyword || typ == itemQuotedPatternOpen {
		return p.parseComplexMessage()
	}

	return p.parseSimpleMessage()
}

func (p *parser) parseSimpleMessage() SimpleMessage {
	return SimpleMessage{Patterns: p.parsePatterns()}
}

func (p *parser) parseComplexMessage() ComplexMessage {
	var declarations []Declaration

	for itm := p.current(); p.current().typ != itemEOF; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemKeyword:
			switch itm.val {
			// TODO: case ReservedKeyword:
			case "." + keywordInput:
				// TODO: implementation
			case "." + keywordLocal:
				declarations = append(declarations, p.parseLocalDeclaration())
			case "." + keywordMatch:
				// Declarations + Matcher
				return ComplexMessage{Declarations: declarations, ComplexBody: p.parseMatcher()}
			}

		case itemQuotedPatternOpen:
			// Declarations + QuotedPattern
			return ComplexMessage{Declarations: declarations, ComplexBody: QuotedPattern{Patterns: p.parsePatterns()}}
		}
	}

	// TODO: error: No complex body found.
	return ComplexMessage{}
}

// ------------------------------Pattern------------------------------

// parsePatterns parses a slice of patterns.
func (p *parser) parsePatterns() []Pattern {
	var pattern []Pattern

	// Loop until the end, or closing pattern quote, if parsing complex message.
	for itm := p.current(); itm.typ != itemEOF && itm.typ != itemQuotedPatternClose; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemText:
			pattern = append(pattern, TextPattern(itm.val))
		case itemExpressionOpen:
			pattern = append(pattern, PlaceholderPattern{Expression: p.parseExpression()})
		}
	}

	return pattern
}

// ------------------------------Expression------------------------------

// parseExpression determines expression type and then parses it accordingly.
func (p *parser) parseExpression() Expression { //nolint:ireturn
	// Move to the significant token. I.e, variable, literal or function.
	for itm := p.current(); p.current().typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemVariable:
			return p.parseVariableExpression()
		case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			return p.parseLiteralExpression()
		case itemFunction:
			return AnnotationExpression{Annotation: p.parseAnnotation()}
		}
	}

	// TODO: error. Reason: no expression start found.
	return nil
}

func (p *parser) parseVariableExpression() VariableExpression {
	var variable Variable

	for itm := p.current(); p.current().typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemVariable:
			variable = Variable(itm.val[1:]) // omit "$" prefix //TODO: Lexer should not capture variable prefix
			// TODO: error handling: this case should happen exactly once
		case itemFunction, itemPrivate, itemReserved:
			// Variable expression with annotation.
			return VariableExpression{Variable: variable, Annotation: p.parseAnnotation()}
		}
	}

	// Variable expression without annotation.
	return VariableExpression{Variable: variable}
}

func (p *parser) parseLiteralExpression() LiteralExpression {
	var literal Literal

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			literal = p.parseLiteral()
			// TODO: error handling: this case should happen exactly once
		case itemFunction:
			// Literal expression with annotation.
			return LiteralExpression{Literal: literal, Annotation: p.parseAnnotation()}
		}
	}

	// Literal expression without annotation.
	return LiteralExpression{Literal: literal}
}

// ------------------------------Annotation------------------------------

// parseAnnotation determines annotation type and then parses it accordingly.
func (p *parser) parseAnnotation() Annotation { //nolint:ireturn
	//nolint:exhaustive
	switch p.current().typ {
	default:
		// TODO: error. Reason: unexpected token
		return nil
	case itemFunction:
		return p.parseFunctionAnnotation()
	case itemPrivate:
		return p.parsePrivateUseAnnotation()
	case itemReserved:
		return p.parseReservedAnnotation()
	}
}

func (p *parser) parseFunctionAnnotation() FunctionAnnotation {
	var annotation FunctionAnnotation

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		//nolint:exhaustive
		switch itm.typ {
		case itemFunction:
			annotation.Function = p.parseFunction()
		case itemOption:
			annotation.Options = append(annotation.Options, p.parseOption())
		}
	}

	return annotation
}

func (p *parser) parsePrivateUseAnnotation() PrivateUseAnnotation {
	// TODO: implement
	return PrivateUseAnnotation{}
}

func (p *parser) parseReservedAnnotation() ReservedAnnotation {
	// TODO: implement
	return ReservedAnnotation{}
}

// ------------------------------Declaration------------------------------

func (p *parser) parseLocalDeclaration() LocalDeclaration {
	var variable Variable

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemVariable:
			variable = Variable(itm.val[1:]) // omit "$" prefix //TODO: Lexer should not capture variable prefix
			// TODO: error handling: this case should happen exactly once
		case itemExpressionOpen:
			return LocalDeclaration{Variable: variable, Expression: p.parseExpression()}
		}
	}

	// TODO: error. Reason: no expression found.
	return LocalDeclaration{}
}

// ---------------------------------------------------------------------

func (p *parser) parseMatcher() Matcher {
	var matcher Matcher

	for itm := p.current(); itm.typ != itemEOF; itm = p.next() {
		// TODO: Error handling: unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemExpressionOpen:
			matcher.MatchStatements = append(matcher.MatchStatements, p.parseExpression())
		case itemCatchAllKey, itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			matcher.Variants = append(matcher.Variants, Variant{
				Key: p.parseVariantKey(),
				QuotedPattern: QuotedPattern{
					Patterns: p.parsePatterns(),
				},
			})
		}
	}

	return matcher
}

func (p *parser) parseVariantKey() VariantKey { //nolint:ireturn
	if p.current().typ == itemCatchAllKey {
		return CatchAllKey{}
	}

	return LiteralKey{Literal: p.parseLiteral()}
}

func (p *parser) parseOption() Option { //nolint:ireturn
	var identifier Identifier

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemOption:
			identifier = p.parseIdentifier()

		case itemNumberLiteral, itemQuotedLiteral, itemUnquotedLiteral:
			return LiteralOption{Literal: p.parseLiteral(), Identifier: identifier}

		case itemVariable:
			return VariableOption{
				Variable:   Variable(p.current().val[1:]), // omit "$" prefix //TODO: Lexer should not capture variable prefix
				Identifier: identifier,
			}
		}
	}

	// todo: error. Reason: value is missing for option
	return nil
}

func (p *parser) parseLiteral() Literal { //nolint:ireturn
	// TODO: Error handling: unexpected token
	//nolint:exhaustive
	switch itm := p.current(); itm.typ {
	case itemNumberLiteral:
		var num float64
		if err := json.Unmarshal([]byte(itm.val), &num); err != nil {
			// TODO: Return error instead of panic
			panic(err)
		}

		return UnquotedLiteral{Value: NumberLiteral(num)}
	case itemQuotedLiteral:
		return QuotedLiteral(p.current().val)
	case itemUnquotedLiteral:
		return UnquotedLiteral{Value: NameLiteral(p.current().val)}
	}

	// TODO: error. Reason: unexpected token
	return nil
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
	// TODO: error handling: case unexpected length
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
