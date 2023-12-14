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

	message.Patterns, err = p.parsePatterns()
	if err != nil {
		return SimpleMessage{}, fmt.Errorf("parse pattern: %w", err)
	}

	return message, nil
}

func (p *parser) parseComplexMessage() (ComplexMessage, error) {
	var message ComplexMessage

	for item := p.current(); p.current().typ != itemEOF; item = p.next() {
		switch item.typ {
		case itemKeyword:
			switch item.val {
			case "." + keywordInput:
				// todo: implement
			case "." + keywordLocal:
				message.Declarations = append(message.Declarations, p.parseLocalDeclaration())
			case "." + keywordMatch:
				message.ComplexBody = p.parseMatcher()

				// last possible element
				return message, nil
			}

		case itemQuotedPatternOpen:
			patterns, err := p.parsePatterns()
			if err != nil {
				return ComplexMessage{}, fmt.Errorf("parse pattern: %w", err)
			}

			message.ComplexBody = QuotedPattern{Patterns: patterns}

			// last possible element
			return message, nil
		}
	}

	// todo: error no quoted pattern found
	return message, nil
}

// ------------------------------Pattern------------------------------

// parsePatterns parses a slice of patterns.
func (p *parser) parsePatterns() ([]Pattern, error) {
	var pattern []Pattern

	for item := p.current(); item.typ != itemEOF && item.typ != itemQuotedPatternClose; item = p.next() {
		switch item.typ {
		case itemText:
			pattern = append(pattern, TextPattern{Text: item.val})
		case itemExpressionOpen:
			item = p.next() // we move omit the "{"

			pattern = append(pattern, PlaceholderPattern{Expression: p.parseExpression()})
		}
	}

	return pattern, nil
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
			return p.parseAnnotationExpression()
		}
	}

	// todo: error. Reason: no expression start found.
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
		}
	}

	// todo: error. Reason: no expression found.
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
			matcher.Variants = append(matcher.Variants, p.parseVariant())
		}
	}

	return matcher
}

func (p *parser) parseVariant() Variant {
	key := p.parseVariantKey()

	patterns, err := p.parsePatterns()
	if err != nil {
		panic(err)
	}

	pattern := QuotedPattern{Patterns: patterns}

	return Variant{Key: key, QuotedPattern: pattern}
}

func (p *parser) parseVariantKey() VariantKey {
	value := p.current().val
	if value == "*" {
		return WildcardKey{Wildcard: '*'}
	}

	num, err := strAsNumber(value)
	if err != nil {
		return LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral{Name: value}}}
	}

	var literal Literal

	switch num := num.(type) {
	case int64:
		literal = UnquotedLiteral{Value: NumberLiteral[int64]{Number: num}}
	case float64:
		literal = UnquotedLiteral{Value: NumberLiteral[float64]{Number: num}}
	}

	return LiteralKey{Literal: literal}
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

	num, err := strAsNumber(p.current().val)
	if err != nil {
		return QuotedLiteral{Value: p.current().val}
	}

	var literal Literal

	switch num := num.(type) {
	case int64:
		literal = UnquotedLiteral{Value: NumberLiteral[int64]{Number: num}}
	case float64:
		literal = UnquotedLiteral{Value: NumberLiteral[float64]{Number: num}}
	}

	return literal

	// // If it possible to parse the value as a integer or float then it is unquoted number literal.
	// if num, err := strconv.ParseInt(p.current().val, 10, 64); err == nil {
	// 	return UnquotedLiteral{Value: NumberLiteral[int64]{Number: num}}
	// }

	// if num, err := strconv.ParseFloat(p.current().val, 64); err == nil {
	// 	return UnquotedLiteral{Value: NumberLiteral[float64]{Number: num}}
	// }

	// // Else it is quoted literal.
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

// helpers

func strAsNumber(s string) (interface{}, error) {
	if num, err := strconv.ParseInt(s, 10, 64); err == nil {
		return num, nil
	}

	if num, err := strconv.ParseFloat(s, 64); err == nil {
		return num, nil
	}

	return nil, errors.New("not a number")
}
