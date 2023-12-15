package mf2

import (
	"errors"
	"fmt"
	"strconv"
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
		item := p.lexer.nextItem()
		if item.typ == itemError {
			return fmt.Errorf("got error token: %s", item.val)
		}

		p.items = append(p.items, item)

		if item.typ == itemEOF {
			return nil
		}

	}

	return errors.New("too many tokens. infinite loop ?")
}

func Parse(input string) (AST, error) {
	p := &parser{lexer: lex(input)}
	if err := p.collect(); err != nil {
		return nil, fmt.Errorf("collect tokens: %w", err)
	}

	if len(p.items) == 1 && p.items[0].typ == itemEOF {
		return nil, nil
	}

	// TODO: parse error handling
	if typ := p.items[0].typ; typ == itemKeyword || typ == itemQuotedPatternOpen {
		return p.parseComplexMessage(), nil
	}

	return p.parseSimpleMessage(), nil
}

func (p *parser) parseSimpleMessage() SimpleMessage {
	return SimpleMessage{Patterns: p.parsePatterns()}
}

func (p *parser) parseComplexMessage() ComplexMessage {
	var message ComplexMessage

	for item := p.current(); p.current().typ != itemEOF; item = p.next() {
		switch item.typ {
		case itemKeyword:
			switch item.val {
			// TODO: case ReservedKeyword:
			case "." + keywordInput:
				// TODO: implementation
			case "." + keywordLocal:
				message.Declarations = append(message.Declarations, p.parseLocalDeclaration())
			case "." + keywordMatch:
				message.ComplexBody = p.parseMatcher()

				// last possible element
				return message
			}

		case itemQuotedPatternOpen:
			message.ComplexBody = QuotedPattern{Patterns: p.parsePatterns()}

			// last possible element
			return message

			// TODO: Error handling: case unexpected token
		}
	}

	// TODO: error (loop should end with match or quoted pattern item)
	return message
}

// ------------------------------Pattern------------------------------

// parsePatterns parses a slice of patterns.
func (p *parser) parsePatterns() []Pattern {
	var pattern []Pattern

	// Loop until the end, or closing pattern quote, if parsing complex message.
	for item := p.current(); item.typ != itemEOF && item.typ != itemQuotedPatternClose; item = p.next() {
		switch item.typ {
		case itemText:
			pattern = append(pattern, TextPattern{Text: item.val})
		case itemExpressionOpen:
			pattern = append(pattern, PlaceholderPattern{Expression: p.parseExpression()})
			// TODO: Error handling: case unexpected token
		}
	}

	return pattern
}

// ------------------------------Expression------------------------------

// parseExpression chooses the correct expression to parse and then parses it.
func (p *parser) parseExpression() Expression {
	// Move to the significant token. I.e, variable, literal or function.
	for item := p.current(); p.current().typ != itemExpressionClose; item = p.next() {
		switch item.typ {
		case itemVariable:
			return p.parseVariableExpression()
		case itemLiteral:
			return p.parseLiteralExpression()
		case itemFunction:
			return AnnotationExpression{Annotation: p.parseAnnotation()}
			// TODO: Error handling: case unexpected token
		}
	}

	// TODO: error. Reason: no expression start found.
	return nil
}

func (p *parser) parseVariableExpression() VariableExpression {
	var expression VariableExpression

	for item := p.current(); p.current().typ != itemExpressionClose; item = p.next() {
		switch item.typ {
		case itemVariable:
			expression.Variable = Variable(item.val[1:])
		case itemFunction, itemPrivate, itemReserved:
			expression.Annotation = p.parseAnnotation()

			// last possible element
			return expression
			// TODO: Error handling: case unexpected token
		}
	}

	return expression
}

func (p *parser) parseLiteralExpression() LiteralExpression {
	var expression LiteralExpression

	for item := p.current(); item.typ != itemExpressionClose; item = p.next() {
		switch item.typ {
		case itemLiteral:
			expression.Literal = p.parseLiteral()
		case itemFunction:
			expression.Annotation = p.parseAnnotation()

			// return with function annotation
			return expression
			// TODO: Error handling: case unexpected token
		}
	}

	// return without function annotation
	return expression
}

// ------------------------------Annotation------------------------------

// parseAnnotation choose the correct annotation to parse and then parses it.
func (p *parser) parseAnnotation() Annotation {
	var annotation Annotation

	switch p.current().typ {
	case itemFunction:
		annotation = p.parseFunctionAnnotation()
	case itemPrivate:
		annotation = p.parsePrivateUseAnnotation()
	case itemReserved:
		annotation = p.parseReservedAnnotation()
	}

	return annotation
}

func (p *parser) parseFunctionAnnotation() FunctionAnnotation {
	var annotation FunctionAnnotation

	for item := p.current(); item.typ != itemExpressionClose; item = p.next() {
		switch item.typ {
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
	var declaration LocalDeclaration

	for item := p.current(); item.typ != itemExpressionClose; item = p.next() {
		switch item.typ {
		case itemVariable:
			declaration.Variable = Variable(item.val[1:])
		case itemExpressionOpen:
			declaration.Expression = p.parseExpression()

			// last possible element
			return declaration
			// TODO: Error handling: case unexpected token
		}
	}

	// TODO: error. Reason: no expression found.
	return declaration
}

// ---------------------------------------------------------------------

func (p *parser) parseMatcher() Matcher {
	var matcher Matcher

	for item := p.current(); item.typ != itemEOF; item = p.next() {
		switch item.typ {
		case itemExpressionOpen:
			matcher.MatchStatement.Selectors = append(matcher.MatchStatement.Selectors, p.parseExpression())
		case itemLiteral:
			matcher.Variants = append(matcher.Variants, Variant{
				Key: p.parseVariantKey(),
				QuotedPattern: QuotedPattern{
					Patterns: p.parsePatterns(),
				},
			})
			// TODO: Error handling: unexpected token
		}
	}

	return matcher
}

func (p *parser) parseVariantKey() VariantKey {
	value := p.current().val
	if value == "*" {
		return WildcardKey{Wildcard: '*'}
	}

	var key VariantKey

	// Try to parse the literal as a number, if it fails then it is a name literal.
	switch num := strAsNumber(value).(type) {
	case nil:
		key = LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral{Name: value}}}
	case int64:
		key = LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral[int64]{Number: num}}}
	case float64:
		key = LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral[float64]{Number: num}}}
	}

	return key
}

func (p *parser) parseOption() Option {
	var identifier Identifier

	for item := p.current(); item.typ != itemExpressionClose; item = p.next() {
		switch item.typ {
		case itemOption:
			identifier = p.parseIdentifier()

		case itemLiteral:
			option := LiteralOption{Literal: p.parseLiteral(), Identifier: identifier}

			return option
		case itemVariable:
			option := VariableOption{Variable: Variable(p.current().val[1:]), Identifier: identifier}

			return option

			// TODO: Error handling: unexpected token
		}
	}

	// todo: error. Reason: value is missing for option
	return nil
}

func (p *parser) parseLiteral() Literal {
	// If there is prefix "$" then it is unquoted name literal.
	if strings.HasPrefix(p.current().val, "$") {
		return UnquotedLiteral{Value: NameLiteral{Name: p.current().val[1:]}}
	}

	var literal Literal

	// Try to parse the literal as a number, if nil then it is a quoted literal, otherwise it is a number literal.
	switch num := strAsNumber(p.current().val).(type) {
	case nil:
		literal = QuotedLiteral{Value: p.current().val}
	case int64:
		literal = UnquotedLiteral{Value: NumberLiteral[int64]{Number: num}}
	case float64:
		literal = UnquotedLiteral{Value: NumberLiteral[float64]{Number: num}}
	}

	return literal
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
		// TODO: error handling: case unexpected length
	}

	return Identifier{Namespace: ns, Name: name}
}

// helpers

// strAsNumber tries to parse a string as a number.
// Returns int64 or float64 if successful, otherwise returns nil.
func strAsNumber(s string) interface{} {
	if num, err := strconv.ParseInt(s, 10, 64); err == nil {
		return num
	}

	if num, err := strconv.ParseFloat(s, 64); err == nil {
		return num
	}

	return nil
}
