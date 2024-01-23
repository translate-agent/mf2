package parse

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
	itemInputKeyword
	itemLocalKeyword
	itemMatchKeyword
	itemReservedKeyword
	itemCatchAllKey
	itemNumberLiteral
	itemQuotedLiteral
	itemUnquotedLiteral
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
	case itemCatchAllKey:
		return "catch all key"
	case itemEOF:
		return "eof"
	case itemError:
		return "error"
	case itemExpressionClose:
		return "expression close"
	case itemExpressionOpen:
		return "expression open"
	case itemFunction:
		return "function"
	case itemText:
		return "text"
	case itemInputKeyword:
		return "input keyword"
	case itemLocalKeyword:
		return "local keyword"
	case itemMatchKeyword:
		return "match keyword"
	case itemReservedKeyword:
		return "reserved keyword"
	case itemNumberLiteral:
		return "number literal"
	case itemOperator:
		return "operator"
	case itemOption:
		return "option"
	case itemPrivate:
		return "private"
	case itemQuotedLiteral:
		return "quoted literal"
	case itemQuotedPatternClose:
		return "quoted pattern close"
	case itemQuotedPatternOpen:
		return "quoted pattern open"
	case itemReserved:
		return "reserved"
	case itemUnquotedLiteral:
		return "unquoted literal"
	case itemVariable:
		return "variable"
	case itemWhitespace:
		return "whitespace"
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
func lex(input string) *lexer { return &lexer{input: input, line: 1} }

// lexer is a lexical analyzer for MessageFormat2.
//
// See https://github.com/unicode-org/message-format-wg/blob/7c00820a0462679eba696181c45bfadb43d2eedd/spec/message.abnf
type lexer struct {
	input      string
	item, prev item // prev non-whitespace
	pos, line  int

	isFunction,
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
func (l *lexer) emitItem(i item) stateFn {
	l.item = i

	if i.typ != itemWhitespace && i.typ != itemEOF {
		l.prev = i
	}

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

			// cases sorted based on the frequency of rune occurrence
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
					return l.emitErrorf("unexpected escaped char in pattern: %s", string(next))
				case '\\', '{', '}': // text-escape = backslash ( backslash / "{" / "}" )
					s += string(next)
				}
			case r == '{':
				if l.peek() == '{' { // complex message without declarations
					l.backup()

					return lexComplexMessage(l)
				}

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
				return lexName(l) // TODO: return lexReservedKeyword
			case strings.HasPrefix(l.input[l.pos:], keywordLocal):
				l.pos += len(keywordLocal)
				return l.emitItem(mk(itemLocalKeyword, keywordLocal))
			case strings.HasPrefix(l.input[l.pos:], keywordInput):
				l.pos += len(keywordInput)
				return l.emitItem(mk(itemInputKeyword, keywordInput))
			case strings.HasPrefix(l.input[l.pos:], keywordMatch):
				l.pos += len(keywordMatch)
				return l.emitItem(mk(itemMatchKeyword, keywordMatch))
			}
		// TODO: parse reserved-statement
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
				l.isComplexMessage = true // complex message without declarations
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
			return l.emitItem(mk(itemCatchAllKey, "*"))
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
		return l.emitErrorf("unexpected eof in expression")
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
		l.isFunction = false

		return l.emitItem(mk(itemExpressionClose, "}"))
	case isWhitespace(v):
		l.backup()
		return lexWhitespace(l)
	case l.isFunction &&
		(l.prev.typ == itemFunction ||
			l.prev.typ == itemQuotedLiteral ||
			l.prev.typ == itemUnquotedLiteral ||
			l.prev.typ == itemNumberLiteral):
		l.backup()
		return lexIdentifier(l)
	case isReservedStart(v):
		l.backup()

		return lexReserved(l, itemReserved)

	case isPrivateStart(v):
		l.backup()

		return lexReserved(l, itemPrivate)

	case v == '=':
		return l.emitItem(mk(itemOperator, "="))
	}
}

// lexName is the state function for lexing names.
func lexName(l *lexer) stateFn {
	var typ itemType

	switch l.next() {
	case '$':
		typ = itemVariable
	case '.':
		typ = itemReservedKeyword
	default:
		typ = itemUnquotedLiteral

		l.backup() // backup to the first rune
	}

	var (
		s string // item value
		r rune   // current rune
	)

	for r = l.next(); isName(r); r = l.next() {
		s += string(r)
	}

	if r == eof {
		return l.emitErrorf("unexpected eof in name")
	}

	l.backup()

	return l.emitItem(mk(typ, s))
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
				return l.emitErrorf("unknown character in quoted literal: %s", string(r))
			case isQuoted(r):
				s += string(r)
			case r == '|':
				opening = !opening
				if !opening {
					return l.emitItem(mk(itemQuotedLiteral, s))
				}
			case r == '\\':
				next := l.next()

				switch next {
				default:
					return l.emitErrorf("unexpected escaped character in quoted literal: %s", string(r))
				case '\\', '|':
					s += string(next)
				case eof:
					return l.emitErrorf("unexpected eof in quoted literal")
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

				return l.emitItem(mk(itemNumberLiteral, s))
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
			if ns {
				return l.emitErrorf("namespace already defined in identifier: %s", s)
			}

			ns = true
			s += string(r)
		case r == eof:
			return l.emitErrorf("unexpected eof in identifier")
		case len(s) == 0:
			switch r {
			default:
				typ = itemOption
			case '-', '+', ':':
				l.isFunction = true
				typ = itemFunction
			}

			s += string(r)
		case len(s) == 1 && isNameStart(r):
			s = string(r)
		}
	}
}

func lexReserved(l *lexer, typ itemType) stateFn {
	var s string

	for {
		v := l.next()

		switch {
		default:
			return l.emitErrorf("unexpected reserved character: %s", string(v))
		case v == eof:
			return l.emitErrorf("unexpected eof in reserved")
		case v == '\\':
			v = l.next()

			if !isReservedEscape(v) {
				return l.emitErrorf("unexpected escaped character in reserved: %s", string(v))
			}

			s += string(v)

		case isReserved(v):
			s += string(v)
		case v == '|':
			l.backup()
			lexLiteral(l)

			if l.item.typ == itemError {
				return l.emitItem(l.item)
			}

			s += fmt.Sprintf("|%s|", l.item.val)
		case isWhitespace(v):
			if l.peek() == '}' {
				l.backup()

				return l.emitItem(mk(typ, s))
			}

			s += string(v)
		case v == '}':
			l.backup()

			return l.emitItem(mk(typ, s))
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
	return isNameStart(v) ||
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
// s = 1*( SP / HTAB / CR / LF / %x3000 ).
func isWhitespace(r rune) bool {
	switch r {
	default:
		return false
	case ' ', '\t', '\r', '\n', '\u3000':
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
//	reserved-char  = %x00-08        ; omit HTAB and LF
//	               / %x0B-0C        ; omit CR
//	               / %x0E-19        ; omit SP
//	               / %x21-5B        ; omit \
//	               / %x5D-7A        ; omit { | }
//	               / %x7E-2FFF      ; omit IDEOGRAPHIC SPACE
//	               / %x3001-D7FF    ; omit surrogates
//	               / %xE000-10FFFF
func isReserved(r rune) bool {
	return 0x00 <= r && r <= 0x08 || // omit HTAB and LF
		0x0B <= r && r <= 0x0C || // omit CR
		0x0E <= r && r <= 0x19 || // omit SP
		0x21 <= r && r <= 0x5B || // omit \
		0x5D <= r && r <= 0x7A || // omit { | }
		0x7E <= r && r <= 0x2FFF || // omit IDEOGRAPHIC SPACE
		0x3001 <= r && r <= 0xD7FF || // omit surrogates
		0xE000 <= r && r <= 0x10FFFF
}

// isReservedEscape returns true if r is reserved escape character.
//
// ABNF:
//
//	reserved-escape = backslash ( backslash / "{" / "|" / "}" ).
func isReservedEscape(r rune) bool {
	return r == '\\' || r == '{' || r == '|' || r == '}'
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

// isPrivateStart returns true if r is private start character.
//
// ABNF:
//
//	private-start = "^" / "&".
func isPrivateStart(r rune) bool {
	return r == '^' || r == '&'
}
