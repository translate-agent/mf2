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
		return item{}
	}

	p.pos++

	return p.items[p.pos]
}

func (p *parser) current() item {
	if p.pos >= len(p.items) {
		return item{}
	}

	return p.items[p.pos]
}

func (p *parser) collect() error {
	for {
		item := p.lexer.nextItem()
		if item.typ == itemError {
			return errors.New("got error token")
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

	for current := p.current(); current.typ != itemEOF; current = p.next() {
		switch current.typ {
		case itemText:
			pattern = append(pattern, p.parseTextPattern())
		case itemExpressionOpen:
			closeIdx := index(p.items[p.pos:], item{typ: itemExpressionClose, val: "}"}) + 1
			openIdx := p.pos

			pattern = append(pattern, p.parsePlaceholderPattern(openIdx, closeIdx))
			p.pos = closeIdx
		default:
			return nil, fmt.Errorf("unexpected token: %v", p.current())
		}
	}

	return pattern, nil
}

// parsePattern parses a single text pattern.
func (p *parser) parseTextPattern() TextPattern {
	return TextPattern{Text: p.current().val}
}

// parsePattern parses a single placeholder(expression) pattern.
func (p *parser) parsePlaceholderPattern(start, end int) PlaceholderPattern {
	return PlaceholderPattern{Expression: p.parseExpression(start, end)}
}

// ------------------------------Expression------------------------------

func (p *parser) parseExpression(start, end int) Expression {
	var expression Expression

	typ := p.items[start+1].typ
	p.pos++
	if typ == itemWhitespace {
		p.pos++
		typ = p.items[start+2].typ
	}

	switch typ {
	case itemVariable:
		expression = p.parseVariableExpression(start, end)
	case itemLiteral:
		expression = p.parseLiteralExpression(end)
	case itemFunction:
		expression = p.parseAnnotationExpression(end)
	}

	return expression
}

func (p *parser) parseVariableExpression(start, end int) VariableExpression {
	var expression VariableExpression

	for p.pos < end {
		item := p.current()
		switch item.typ {
		case itemVariable:
			expression.Variable = Variable(item.val[1:])
		case itemFunction:
			expression.Annotation = p.parseAnnotation(end)
		}

		p.pos++
	}

	return expression
}

func (p *parser) parseLiteralExpression(end int) LiteralExpression {
	var expression LiteralExpression

	for p.pos < end {
		item := p.current()
		switch item.typ {
		case itemLiteral:
			expression.Literal = p.parseLiteral()
		case itemFunction:
			expression.Annotation = p.parseAnnotation(end)
		}

		p.pos++
	}

	return expression
}

func (p *parser) parseAnnotationExpression(end int) AnnotationExpression {
	return AnnotationExpression{Annotation: p.parseAnnotation(end)}
}

// ------------------------------Annotation------------------------------

func (p *parser) parseAnnotation(end int) Annotation {
	var annotation Annotation

	switch p.current().typ {
	case itemFunction:
		annotation = p.parseFunctionAnnotation(end)
	case itemPrivate:
		annotation = p.parsePrivateUseAnnotation()
	case itemReserved:
		annotation = p.parseReservedAnnotation()
	}

	return annotation
}

func (p *parser) parseFunctionAnnotation(end int) FunctionAnnotation {
	var annotation FunctionAnnotation

	for p.pos < end {
		item := p.current()
		switch item.typ {
		case itemFunction:
			annotation.Function = p.parseFunction()
		case itemOption:
			annotation.Options = append(annotation.Options, p.parseOption(end))
		}

		p.pos++
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

func (p *parser) parseOption(end int) Option {
	var option Option

	var identifier Identifier

	for p.pos < end {
		item := p.current()
		switch item.typ {
		case itemOption:
			identifier.Namespace, identifier.Name = getNamespaceWithName(item.val)

		case itemLiteral:
			option := LiteralOption{Literal: p.parseLiteral(), Identifier: identifier}
			p.pos++

			return option
		case itemVariable:
			option := VariableOption{Variable: p.parseVariable(), Identifier: identifier}
			p.pos++

			return option
		}

		p.pos++
	}

	return option
}

func (p *parser) parseLiteral() Literal {
	value := p.current().val

	if strings.HasPrefix(value, "$") {
		return UnquotedLiteral{Value: NameLiteral{Name: value[1:]}}
	}

	if num, err := strconv.ParseInt(value, 10, 64); err == nil {
		return UnquotedLiteral{Value: NumberLiteral[int64]{Number: num}}
	}

	if num, err := strconv.ParseFloat(value, 64); err == nil {
		return UnquotedLiteral{Value: NumberLiteral[float64]{Number: num}}
	}

	return QuotedLiteral{Value: value}
}

func (p *parser) parseVariable() Variable {
	return Variable(p.current().val[1:])
}

func (p *parser) parseFunction() Function {
	value := p.current().val

	ns, name := getNamespaceWithName(value[1:])

	return Function{
		Node:   nil,
		Prefix: rune(value[0]),
		Identifier: Identifier{
			Namespace: ns,
			Name:      name,
		},
	}
}

// helpers

func index[S []E, E comparable](s S, v E) int {
	for i, e := range s {
		if e == v {
			return i
		}
	}
	return -1
}

func getNamespaceWithName(value string) (*string, string) {
	full := strings.Split(value, ":")

	var ns *string

	name := full[0]
	if len(full) == 2 {
		ns = &full[0]
		name = full[1]
	}

	return ns, name
}
