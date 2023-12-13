package mf2

import (
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
	for {
		item := p.lexer.nextItem()
		if item.typ == itemError {
			return fmt.Errorf("got error token: %s", item.val)
		}

		p.items = append(p.items, item)

		if item.typ == itemEOF {
			break
		}
	}

	return nil
}

func new(lexer *lexer) *parser {
	return &parser{lexer: lexer}
}

func Parse(input string) (AST, error) {
	p := new(lex(input))
	if err := p.collect(); err != nil {
		return nil, fmt.Errorf("collect tokens: %w", err)
	}

	if len(p.items) == 0 {
		return nil, nil
	}

	// Determine if the input is a complex or simple message.
	isFirstKeyword := p.items[0].typ == itemKeyword
	isFirstQuotedPattern := len(p.items) > 1 &&
		p.items[0].typ == itemQuotedPatternOpen &&
		p.items[1].typ == itemQuotedPatternOpen

	var message Message
	var err error

	if isFirstKeyword || isFirstQuotedPattern {
		message, err = p.parseComplexMessage()
	} else {
		message, err = p.parseSimpleMessage()
	}

	if err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}

	return message, nil
}

func (p *parser) parseSimpleMessage() (SimpleMessage, error) {
	var message SimpleMessage
	var err error

	message.Pattern, err = p.parsePattern()
	if err != nil {
		return SimpleMessage{}, fmt.Errorf("parse pattern: %w", err)
	}

	return message, nil
}

func (p *parser) parseComplexMessage() (ComplexMessage, error) {
	return ComplexMessage{}, nil
}

// ------------------------------Pattern------------------------------

// parsePattern parses a slice of patterns.
func (p *parser) parsePattern() ([]Pattern, error) {
	var pattern []Pattern

	for item := p.current(); p.current().typ != itemEOF; item = p.next() {
		switch item.typ {
		case itemText:
			pattern = append(pattern, TextPattern{Text: item.val})
		case itemExpressionOpen:
			item = p.next() // we move omit the "{"

			pattern = append(pattern, PlaceholderPattern{Expression: p.parseExpression()})
		default:
			return nil, fmt.Errorf("unexpected token: %v", item.typ)
		}
	}

	return pattern, nil
}

// ------------------------------Expression------------------------------

// parseExpression chooses the correct expression to parse and then parses it.
func (p *parser) parseExpression() Expression {
	var expression Expression

	// move to next significant token
	for p.current().typ == itemWhitespace {
		p.next()
	}

	switch p.current().typ {
	case itemVariable:
		expression = p.parseVariableExpression()
	case itemLiteral:
		expression = p.parseLiteralExpression()
	case itemFunction:
		expression = p.parseAnnotationExpression()
	}

	return expression
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
		}
	}

	// return without function annotation
	return expression
}

func (p *parser) parseAnnotationExpression() AnnotationExpression {
	return AnnotationExpression{Annotation: p.parseAnnotation()}
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

func (p *parser) parseOption() Option {
	var identifier Identifier

	for item := p.current(); item.typ != itemExpressionClose; item = p.next() {
		switch item.typ {
		case itemOption:
			identifier = p.parseIdentifier()

		case itemLiteral:
			option := LiteralOption{Literal: p.parseLiteral(), Identifier: identifier}
			p.next()

			return option
		case itemVariable:
			option := VariableOption{Variable: Variable(p.current().val[1:]), Identifier: identifier}
			p.next()

			return option
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

	// If it possible to parse the value as a integer or float then it is unquoted number literal.
	if num, err := strconv.ParseInt(p.current().val, 10, 64); err == nil {
		return UnquotedLiteral{Value: NumberLiteral[int64]{Number: num}}
	}

	if num, err := strconv.ParseFloat(p.current().val, 64); err == nil {
		return UnquotedLiteral{Value: NumberLiteral[float64]{Number: num}}
	}

	// Else it is quoted literal.
	return QuotedLiteral{Value: p.current().val}
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
	}

	return Identifier{Namespace: ns, Name: name}
}
