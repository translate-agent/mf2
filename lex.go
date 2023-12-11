package mf2

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// eof is the end of file item.
const eof = -1

// itemType is the type of an item.
type itemType int

const (
	itemError itemType = iota
	itemEOF
	itemVariable
	itemFunction
	itemExpressionOpen
	itemExpressionClose
	itemQuotedPatternOpen
	itemQuotedPatternClose
	itemText
	itemKeyword
	itemLiteral
	itemOption
	itemWhitespace
	itemReserved
	itemOperator
	itemPrivate
)

// String returns a string representation of the item type.
func (t itemType) String() string {
	switch t {
	default:
		return "unknown"
	case itemError:
		return "error"
	case itemEOF:
		return "eof"
	case itemVariable:
		return "variable"
	case itemFunction:
		return "function"
	case itemExpressionOpen:
		return "expression open"
	case itemExpressionClose:
		return "expression close"
	case itemQuotedPatternOpen:
		return "quoted pattern open"
	case itemQuotedPatternClose:
		return "quoted pattern close"
	case itemText:
		return "text"
	case itemKeyword:
		return "keyword"
	case itemLiteral:
		return "literal"
	case itemOption:
		return "option"
	case itemWhitespace:
		return "whitespace"
	case itemReserved:
		return "reserved"
	case itemOperator:
		return "operator"
	case itemPrivate:
		return "private"
	}
}

// Keywords.
const (
	keywordMatch = "match"
	keywordLocal = "local"
	keywordInput = "input"
)

// item is an item returned by the lexer.
type item struct {
	val string
	typ itemType
}

// mk creates a new item with the given type and value.
func mk(typ itemType, val string) item {
	return item{typ: typ, val: val}
}

// lex creates a new lexer for the given input string.
func lex(input string) *lexer { return &lexer{input: input} }

// lexer is a lexical analyzer for MessageFormat2.
//
// See https://github.com/unicode-org/message-format-wg/blob/main/spec/syntax.md
type lexer struct {
	input     string
	item      item
	pos, line int

	isExpression,
	isPattern,
	isComplexMessage bool
}

// peek peeks at the next rune.
func (l *lexer) peek() rune {
	pos := l.pos
	r := l.next()
	l.pos = pos

	return r
}

// next returns the next rune.
func (l *lexer) next() rune {
	if len(l.input) <= l.pos {
		return eof
	}

	r, n := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += n

	if r == '\n' {
		l.line++
	}

	return r
}

// backup backs up the current position in the input string.
func (l *lexer) backup() {
	r, n := utf8.DecodeLastRuneInString(l.input[:l.pos])

	if r == '\n' {
		l.line--
	}

	l.pos -= n
}

// nextItem returns the next item in the input string.
func (l *lexer) nextItem() item {
	l.emitItem(mk(itemEOF, ""))

	state := lexPattern(true)

	// Sorted by children first - expression can be inside pattern but pattern
	// cannot be inside expression. And so on.
	switch {
	case l.isExpression:
		state = lexExpr
	case l.isPattern:
		state = lexPattern(false)
	case l.isComplexMessage:
		state = lexComplexMessage
	}

	for {
		if state := state(l); state == nil {
			return l.item
		}
	}
}

// emitItem emits the given item and returns the next state function.
func (l *lexer) emitItem(t item) stateFn {
	l.item = t

	return nil
}

// emitErrorf emits the error and returns the next state function.
func (l *lexer) emitErrorf(s string, args ...any) stateFn {
	return l.emitItem(item{typ: itemError, val: fmt.Sprintf(s, args...)})
}

// stateFn is a function that returns the next state function.
type stateFn func(*lexer) stateFn

// lexPattern is the state function for lexing patterns.
// When singleMessage is true it will lex single message.
// Otherwise, it lexes pattern.
func lexPattern(singleMessage bool) func(*lexer) stateFn {
	return func(l *lexer) stateFn {
		var s string

		for {
			r := l.next()

			// cases sorted based on the frequency of rune occurance
			switch {
			default:
				l.backup()
				l.isPattern = false

				return l.emitItem(mk(itemText, s))
			case singleMessage && len(s) == 0 && isSimpleStart(r),
				singleMessage && len(s) >= 1 && isText(r),
				!singleMessage && isText(r):
				s += string(r)
			case r == '\\':
				switch next := l.next(); next {
				default:
					return l.emitErrorf("unexpected escaped char: %s", string(next))
				case '\\', '{', '}': // text-escape = backslash ( backslash / "{" / "}" )
					s += string(next)
				}
			case r == '{':
				l.backup()

				if len(s) > 0 {
					l.isExpression = true

					return l.emitItem(mk(itemText, s))
				}

				return lexExpr(l)
			case len(s) == 0 && r == '.':
				l.backup()

				return lexComplexMessage(l)
			case r == eof:
				if len(s) > 0 {
					return l.emitItem(mk(itemText, s))
				}

				return nil
			}
		}
	}
}

// lexComplexMessage is the state function for lexing complex messages.
func lexComplexMessage(l *lexer) stateFn {
	for {
		r := l.next()

		switch {
		default:
			l.backup()

			return lexLiteral(l)
		case r == '.':
			l.isComplexMessage = true

			switch {
			default: // reserved keyword
				l.backup()
				return lexName(l)
			case strings.HasPrefix(l.input[l.pos:], keywordLocal):
				l.pos += len(keywordLocal)
				return l.emitItem(mk(itemKeyword, "."+keywordLocal))
			case strings.HasPrefix(l.input[l.pos:], keywordInput):
				l.pos += len(keywordInput)
				return l.emitItem(mk(itemKeyword, "."+keywordInput))
			case strings.HasPrefix(l.input[l.pos:], keywordMatch):
				l.pos += len(keywordMatch)
				return l.emitItem(mk(itemKeyword, "."+keywordMatch))
			}
		case r == '$':
			l.backup()
			return lexName(l)
		case isWhitespace(r):
			l.backup()

			return lexWhitespace(l)
		case r == '=':
			return l.emitItem(mk(itemOperator, "="))
		case r == '{':
			if l.peek() == '{' {
				l.next()
				l.isPattern = true

				return l.emitItem(mk(itemQuotedPatternOpen, "{{"))
			}

			l.backup()

			return lexExpr(l)
		case r == '}':
			if l.peek() == '}' {
				l.next()
				l.isPattern = false

				return l.emitItem(mk(itemQuotedPatternClose, "}}"))
			}
		case r == '*':
			return l.emitItem(mk(itemLiteral, "*"))
		case r == eof:
			return nil
		}
	}
}

// lexExpr is the state function for lexing expressions.
func lexExpr(l *lexer) stateFn {
	v := l.next()

	switch {
	default:
		l.backup()

		return lexLiteral(l)
	case v == eof:
		return l.emitErrorf("") // TODO: better error message
	case v == '$':
		l.backup()
		return lexName(l)
	case v == '|', v == '-' && isDigit(l.peek()):
		l.backup()
		return lexLiteral(l)
	case v == ':', v == '+', v == '-':
		l.backup()
		return lexIdentifier(l)
	case v == '{':
		l.isExpression = true

		return l.emitItem(mk(itemExpressionOpen, "{"))
	case v == '}':
		l.isExpression = false

		return l.emitItem(mk(itemExpressionClose, "}"))
	case isWhitespace(v):
		l.backup()
		return lexWhitespace(l)
	case isReservedStart(v):
		l.backup()
		return lexReserved(l)
	case v == '^', v == '&':
		// TODO(jhorsts): incomplete implementation
		return l.emitItem(mk(itemPrivate, string(v)))
	}
}

// lexName is the state function for lexing names.
func lexName(l *lexer) stateFn {
	var s string

	typ := itemLiteral

	for {
		r := l.next()

		switch {
		default:
			l.backup()

			return l.emitItem(mk(typ, s))
		case len(s) > 0 && isName(r):
			s += string(r)
		case r == eof:
			return l.emitItem(mk(typ, s))
		case len(s) == 0 && isNameStart(r):
			s = string(r)
		case len(s) == 0 && r == '$':
			s = string(r)
			typ = itemVariable
		case len(s) == 0 && r == '.':
			s = string(r)
			typ = itemKeyword

		}
	}
}

// lexLiteral is the state function for lexing literals.
func lexLiteral(l *lexer) stateFn {
	var s string

	switch l.peek() {
	default: // unquoted literal
		return lexName(l)
	case '|': // quoted literal
		var opening bool

		for {
			r := l.next()

			switch {
			default:
				return l.emitErrorf("unexpected end of quoted literal: %s", string(r))
			case isQuoted(r):
				s += string(r)
			case r == '|':
				opening = !opening
				if !opening {
					return l.emitItem(mk(itemLiteral, s))
				}
			case r == '\\':
				next := l.next()

				switch next {
				default:
					return l.emitErrorf("only \\ or | can be escaped in literal") // TODO: improve error message
				case '\\', '|':
					s += string(r) + string(next)

					return l.emitItem(mk(itemLiteral, s))
				case eof:
					return l.emitErrorf("unexpected eof in literal")
				}
			}
		}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9': // number literal
		for {
			r := l.next()

			switch r {
			default:
				var number float64

				if err := json.Unmarshal([]byte(s), &number); err != nil {
					return l.emitErrorf("invalid number literal: %s", s)
				}

				l.backup()

				return l.emitItem(mk(itemLiteral, s))
			case '-', '+', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.', 'e', 'E':
				s += string(r)
			}
		}
	}
}

// lexWhitespace is the state function for lexing whitespace.
func lexWhitespace(l *lexer) stateFn {
	var ws []rune

	for {
		v := l.next()

		switch {
		default:
			l.backup()
			return l.emitItem(mk(itemWhitespace, string(ws)))
		case v == eof:
			return l.emitItem(mk(itemWhitespace, string(ws)))
		case isWhitespace(v):
			ws = append(ws, v)
		}
	}
}

// lexIdentifier is the state function for lexing identifiers.
func lexIdentifier(l *lexer) stateFn {
	var (
		s   string
		ns  bool
		typ itemType
	)

	for {
		r := l.next()

		switch {
		default:
			l.backup()

			return l.emitItem(mk(typ, s))
		case len(s) > 0 && isName(r):
			s += string(r)
		case len(s) > 0 && r == ':':
			switch {
			default:
				ns = true
				s += string(r)
			case len(s) == 0:
				return l.emitErrorf("namespace not set")
			case ns:
				return l.emitErrorf("only one namespace can be present")
			}
		case r == eof:
			return l.emitErrorf("unexpected eof")
		case len(s) == 0:
			switch r {
			default:
				return l.emitErrorf("unknown identifier")
			case '-', '+', ':':
				typ = itemFunction
			}

			s += string(r)
		case len(s) == 1 && isNameStart(r):
			s = string(r)
		}
	}
}

func lexReserved(l *lexer) stateFn {
	var s string

	for {
		v := l.next()

		switch {
		default:
			return l.emitErrorf("unexpected reserved character %s", string(v)) // TODO: better error message
		case v == eof:
			return l.emitErrorf("eof") // TODO
		case v == '\\':
			v = l.next()

			switch v {
			default:
				return l.emitErrorf("unexpected reserved escaped: %s", string(v)) // TODO: better error message
			case '\\', '{', '|', '}':
				s += string(v)
			}
		case v == '|', isWhitespace(v), v == '}':
			l.backup()

			return l.emitItem(mk(itemReserved, s))
		case len(s) == 0 && isReservedStart(v), isReserved(v):
			s += string(v)
		}
	}
}

// helpers

// isAlpha returns true if r is alphabetic character.
func isAlpha(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

// isNameStart returns true if r is name start character.
//
// ABNF:
//
//	name-start = ALPHA / "_"
//	           / %xC0-D6 / %xD8-F6 / %xF8-2FF
//	           / %x370-37D / %x37F-1FFF / %x200C-200D
//	           / %x2070-218F / %x2C00-2FEF / %x3001-D7FF
//	           / %xF900-FDCF / %xFDF0-FFFD / %x10000-EFFFF
func isNameStart(r rune) bool {
	return isAlpha(r) ||
		r == '_' ||
		0xC0 <= r && r <= 0xD6 ||
		0xD8 <= r && r <= 0xF6 ||
		0xF8 <= r && r <= 0x2FF ||
		0x370 <= r && r <= 0x37D ||
		0x37F <= r && r <= 0x1FFF ||
		0x200C <= r && r <= 0x200D ||
		0x2070 <= r && r <= 0x218F ||
		0x2C00 <= r && r <= 0x2FEF ||
		0x3001 <= r && r <= 0xD7FF ||
		0xF900 <= r && r <= 0xFDCF ||
		0xFDF0 <= r && r <= 0xFFFD ||
		0x10000 <= r && r <= 0xEFFFF
}

// isName returns true if r is name character.
//
// ABNF:
//
//	name-char = name-start / DIGIT / "-" / "." / %xB7 / %x0300-036F / %x203F-2040.
func isName(v rune) bool {
	return isAlpha(v) ||
		'0' <= v && v <= '9' ||
		v == '-' ||
		v == '.' ||
		v == 0xB7 ||
		0x0300 <= v && v <= 0x036F ||
		0x203F <= v && v <= 2040
}

// isQuotedChar returns true if v is quoted character.
func isQuoted(r rune) bool {
	return 0x00 <= r && r <= 0x5B || // omit \
		0x5D <= r && r <= 0x7B || // omit |
		0x7D <= r && r <= 0xD7FF || // omit surrogates
		0xE000 <= r && r <= 0x10FFFF
}

// isWhitespace returns true if r is whitespace character.
//
// s = 1*( SP / HTAB / CR / LF ).
func isWhitespace(r rune) bool {
	switch r {
	default:
		return false
	case ' ', '\t', '\r', '\n':
		return true
	}
}

// isReservedStart returns true if r is the first reserved annotation character.
func isReservedStart(r rune) bool {
	switch r {
	default:
		return false
	case '!', '@', '#', '%', '*', '<', '>', '/', '?', '~':
		return true
	}
}

// isReserved returs true if r is reserved annotation character.
//
// ABNF:
//
//	reserved-char = %x00-08        ; omit HTAB and LF
//	              / %x0B-0C        ; omit CR
//	              / %x0E-19        ; omit SP
//	              / %x21-5B        ; omit \
//	              / %x5D-7A        ; omit { | }
//	              / %x7E-D7FF      ; omit surrogates
//	              / %xE000-10FFFF
func isReserved(r rune) bool {
	return 0x00 <= r && r <= 0x08 || // omit HTAB and LF
		0x0B <= r && r <= 0x0C || // omit CR
		0x0E <= r && r <= 0x19 || // omit SP
		0x21 <= r && r <= 0x5B || // omit \
		0x5D <= r && r <= 0x7A || // omit { | }
		0x7E <= r && r <= 0xD7FF || // omit surrogates
		0xE000 <= r && r <= 0x10FFFF
}

// isSimpleStart returns true if r is simple start character.
//
// ABNF:
//
//	simple-start-char = %x0-2D         ; omit .
//	                  / %x2F-5B        ; omit \
//	                  / %x5D-7A        ; omit {
//	                  / %x7C           ; omit }
//	                  / %x7E-D7FF      ; omit surrogates
//	                  / %xE000-10FFFF
func isSimpleStart(r rune) bool {
	return 0x0 <= r && r <= 0x2D || // omit .
		0x2F <= r && r <= 0x5B || // omit \
		0x5D <= r && r <= 0x7A || // omit {}
		r == 0x7C || // omit }
		0x7E <= r && r <= 0xD7FF || // omit surrogates
		0xE000 <= r && r <= 0x10FFFF
}

// isText returns true if r is text character.
//
// ABNF:
// text-char = simple-start-char / ".".
func isText(r rune) bool {
	return isSimpleStart(r) || r == '.'
}

// isDigit returns true if r is digit character.
func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}
