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

// TODO: godoc.
func Parse(input string) (AST, error) { //nolint:ireturn
	p := &parser{lexer: lex(input)}
	if err := p.collect(); err != nil {
		return nil, fmt.Errorf("collect tokens: %w", err)
	}

	if len(p.items) == 1 && p.items[0].typ == itemEOF {
		return SimpleMessage{}, nil
	}

	return p.parseMessage(), nil
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
	var message ComplexMessage

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
		case itemLiteral:
			return p.parseLiteralExpression()
		case itemFunction:
			return AnnotationExpression{Annotation: p.parseAnnotation()}
		}
	}

	// TODO: error. Reason: no expression start found.
	return nil
}

func (p *parser) parseVariableExpression() VariableExpression {
	var expression VariableExpression

	for itm := p.current(); p.current().typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemVariable:
			expression.Variable = Variable(itm.val[1:])
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

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
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

// ------------------------------Annotation------------------------------

// parseAnnotation determines annotation type and then parses it accordingly.
func (p *parser) parseAnnotation() Annotation { //nolint:ireturn
	var annotation Annotation

	//nolint:exhaustive
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
	var declaration LocalDeclaration

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: case unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemVariable:
			declaration.Variable = Variable(itm.val[1:])
		case itemExpressionOpen:
			declaration.Expression = p.parseExpression()

			// last possible element
			return declaration
		}
	}

	// TODO: error. Reason: no expression found.
	return declaration
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
		case itemLiteral:
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
	val := p.current().val
	if val == "*" {
		return WildcardKey('*')
	}

	var key VariantKey

	// If the literal is not a number then it is a unquoted literal. Otherwise it is a number literal.
	// TODO: Would be nice if lexer could distinguish between quoted and unquoted literals
	// . e.g. two item types: itemQuotedLiteral and itemUnquotedLiteral, instead of itemLiteral.
	// That means, for now it always be a unquoted literal if it is not a number. But it could be a quoted literal too.
	key = LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral(val)}}
	if num := float64(0); json.Unmarshal([]byte(val), &num) == nil {
		key = LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral(num)}}
	}

	return key
}

func (p *parser) parseOption() Option { //nolint:ireturn
	var identifier Identifier

	for itm := p.current(); itm.typ != itemExpressionClose; itm = p.next() {
		// TODO: Error handling: unexpected token
		//nolint:exhaustive
		switch itm.typ {
		case itemOption:
			identifier = p.parseIdentifier()

		case itemLiteral:
			option := LiteralOption{Literal: p.parseLiteral(), Identifier: identifier}

			return option
		case itemVariable:
			option := VariableOption{Variable: Variable(p.current().val[1:]), Identifier: identifier}

			return option
		}
	}

	// todo: error. Reason: value is missing for option
	return nil
}

func (p *parser) parseLiteral() Literal { //nolint:ireturn
	val := p.current().val

	// If there is prefix "$" then it is unquoted name literal.
	if strings.HasPrefix(val, "$") {
		return UnquotedLiteral{Value: NameLiteral(p.current().val[1:])}
	}

	var literal Literal

	// If the literal is not a number then it is a quoted literal. Otherwise it is a number literal.
	// TODO: Would be nice if lexer could distinguish between quoted and unquoted literals.
	// e.g. two item types: itemQuotedLiteral and itemUnquotedLiteral, instead of itemLiteral.
	// That means, for now it always be a quoted literal if it is not a number. But it could be a unquoted literal too.
	literal = QuotedLiteral(val)
	if num := float64(0); json.Unmarshal([]byte(val), &num) == nil {
		literal = UnquotedLiteral{Value: NumberLiteral(num)}
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
