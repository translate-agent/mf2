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
	itemUnknown itemType = iota
	itemError
	itemEOF
	itemVariable
	itemFunction
	itemExpressionOpen
	itemExpressionClose
	itemMarkupOpen
	itemMarkupClose
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
	itemAttribute
	itemWhitespace
	itemOperator
	itemPrivateStart
	itemReservedStart
	itemReservedText
)

// String returns a string representation of the item type.
func (t itemType) String() string {
	switch t {
	case itemUnknown:
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
	case itemMarkupOpen:
		return "markup open"
	case itemMarkupClose:
		return "markup close"
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
	case itemAttribute:
		return "attribute"
	case itemQuotedLiteral:
		return "quoted literal"
	case itemQuotedPatternClose:
		return "quoted pattern close"
	case itemQuotedPatternOpen:
		return "quoted pattern open"
	case itemUnquotedLiteral:
		return "unquoted literal"
	case itemVariable:
		return "variable"
	case itemWhitespace:
		return "whitespace"
	case itemPrivateStart:
		return "private start"
	case itemReservedStart:
		return "reserved start"
	case itemReservedText:
		return "reserved text"
	}

	return "<invalid type>"
}

// Keywords.
const (
	keywordMatch = "match"
	keywordLocal = "local"
	keywordInput = "input"
)

// item is an item returned by the lexer.
type item struct {
	err error
	val string
	typ itemType
}

func (i item) String() string {
	v := i.val
	if i.typ == itemError {
		v = i.err.Error()
	}

	return i.typ.String() + ` token "` + v + `"`
}

// mk creates a new item with the given type and value.
func mk(typ itemType, val string) item {
	return item{typ: typ, val: val}
}

// mkErr creates a new error item with the given format and args.
func mkErr(format string, args ...any) item {
	return item{typ: itemError, err: fmt.Errorf(format, args...)}
}

// lex creates a new lexer for the given input string.
func lex(input string) *lexer {
	return &lexer{
		input:            input,
		line:             1,
		isComplexMessage: len(input) > 0 && input[:1] == "." || len(input) > 1 && input[:2] == "{{",
	}
}

// lexer is a lexical analyzer for MessageFormat2.
//
// See ".message-format-wg/spec/message.abnf".
type lexer struct {
	input     string
	item      item
	prevType  itemType // prev non-whitespace
	pos, line int

	isFunction,
	isMarkup,
	isReservedBody,
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

	state := lexPattern

	// Sorted by children first - expression can be inside pattern but pattern
	// cannot be inside expression. And so on.
	switch {
	case l.isExpression:
		state = lexExpr
	case l.isPattern:
		state = lexPattern
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
		l.prevType = i.typ
	}

	return nil
}

// emitErrorf emits the error and returns the next state function.
func (l *lexer) emitErrorf(s string, args ...any) stateFn {
	return l.emitItem(mkErr(s, args...))
}

// stateFn is a function that returns the next state function.
type stateFn func(*lexer) stateFn

// lexPattern is the state function for lexing patterns.
func lexPattern(l *lexer) stateFn {
	// var s string
	sb := new(strings.Builder)

	for {
		r := l.next()

		// cases sorted based on the frequency of rune occurrence
		switch {
		default:
			l.backup()
			l.isPattern = false

			return l.emitItem(mk(itemText, sb.String()))
		case r == '\\':
			next := l.next()
			if !isEscapedChar(next) {
				return l.emitErrorf("unexpected escaped char in pattern: %s", string(next))
			}

			sb.WriteRune(next)

			// s += string(next)
		case r == '{':
			if l.peek() == '{' { // complex message without declarations
				l.backup()
				return lexComplexMessage(l)
			}

			l.backup()

			if sb.Len() > 0 {
				l.isExpression = true

				return l.emitItem(mk(itemText, sb.String()))
			}

			return lexExpr(l)
		case !l.isComplexMessage && sb.Len() == 0 && r == '.':
			l.backup()

			return lexComplexMessage(l)
		case r == '}':
			if l.peek() != '}' { // pattern end in complex message?
				return l.emitErrorf("unescaped } in pattern")
			}

			l.backup()
			l.isPattern = false

			if sb.Len() > 0 {
				return l.emitItem(mk(itemText, sb.String()))
			}

			return lexComplexMessage(l)
		case !l.isComplexMessage && sb.Len() == 0 && isSimpleStart(r),
			isText(r) && (l.isComplexMessage || sb.Len() >= 1):
			sb.WriteRune(r)
		case r == eof:
			if sb.Len() > 0 {
				return l.emitItem(mk(itemText, sb.String()))
			}

			return nil
		}
	}
}

// lexComplexMessage is the state function for lexing complex messages.
func lexComplexMessage(l *lexer) stateFn {
	for {
		r := l.next()

		switch {
		default:
			return l.emitErrorf("unknown character in complex message: %s", string(r))

		case r == '.':
			switch {
			default: // reserved keyword
				l.backup()
				l.isReservedBody = true

				return lexReservedKeyword(l)
			case l.input[l.pos:l.pos+len(keywordLocal)] == keywordLocal:
				l.pos += len(keywordLocal)
				return l.emitItem(mk(itemLocalKeyword, keywordLocal))
			case l.input[l.pos:l.pos+len(keywordInput)] == keywordInput:
				l.pos += len(keywordInput)
				return l.emitItem(mk(itemInputKeyword, keywordInput))
			case l.input[l.pos:l.pos+len(keywordMatch)] == keywordMatch:
				l.pos += len(keywordMatch)
				return l.emitItem(mk(itemMatchKeyword, keywordMatch))
			}
		case l.isReservedBody:
			l.backup()
			return lexReservedBody(l)
		case r == variablePrefix:
			l.backup()
			return lexVariable(l)
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

			return l.emitErrorf("unexpected } in complex message")
		case r == '*':
			return l.emitItem(mk(itemCatchAllKey, "*"))
		case r == '|':
			l.backup()

			return lexQuotedLiteral(l)
		case isName(r):
			l.backup()
			return lexUnquotedOrNumberLiteral(l)
		case r == eof:
			return nil
		}
	}
}

// lexExpr is the state function for lexing expressions.
func lexExpr(l *lexer) stateFn {
	switch v := l.next(); {
	default:
		l.backup()

		return lexUnquotedOrNumberLiteral(l)
	case l.isReservedBody:
		l.backup()
		return lexReservedBody(l)
	case v == eof:
		return l.emitErrorf("unexpected eof in expression")
	case v == variablePrefix:
		l.backup()
		return lexVariable(l)
	case v == '|':
		l.backup()
		return lexQuotedLiteral(l)
	case v == '#', // markup-open
		v == '/', // markup-close
		v == '@', // attribute
		v == ':': // function
		l.backup()

		return lexIdentifier(l)
	case v == '{': // expression/markup start
		l.isExpression = true

		return l.emitItem(mk(itemExpressionOpen, "{"))
	case v == '}': // expression/markup end
		l.isExpression = false
		l.isFunction = false
		l.isMarkup = false

		return l.emitItem(mk(itemExpressionClose, "}"))
	case isWhitespace(v):
		l.backup()
		return lexWhitespace(l)
	case (l.prevType == itemMarkupOpen || l.prevType == itemMarkupClose) ||
		(l.isFunction || l.isMarkup) &&
			(l.prevType == itemFunction ||
				l.prevType == itemQuotedLiteral ||
				l.prevType == itemUnquotedLiteral ||
				l.prevType == itemNumberLiteral ||
				l.prevType == itemVariable):
		l.backup()
		return lexIdentifier(l)
	case isReservedStart(v):
		l.isReservedBody = true

		return l.emitItem(mk(itemReservedStart, string(v)))
	case isPrivateStart(v):
		l.isReservedBody = true

		return l.emitItem(mk(itemPrivateStart, string(v)))
	case v == '=':
		return l.emitItem(mk(itemOperator, "="))
	}
}

// lexQuotedLiteral is the state function for lexing quoted literals.
func lexQuotedLiteral(l *lexer) stateFn {
	// var s string
	sb := new(strings.Builder)

	if r := l.next(); r != '|' {
		return l.emitErrorf(`unexpected opening character in quoted literal: "%s"`, string(r))
	}

	for {
		r := l.next()

		switch {
		default:
			return l.emitErrorf(`unknown character in quoted literal: "%s"`, string(r))
		case isQuoted(r):
			sb.WriteRune(r)
		case r == '|': // closing
			return l.emitItem(mk(itemQuotedLiteral, sb.String()))
		case r == '\\':
			next := l.next()

			switch next {
			default:
				return l.emitErrorf(`unexpected escaped character in quoted literal: "%s"`, string(r))
			case '\\', '|':
				sb.WriteRune(next)
			case eof:
				return l.emitErrorf("unexpected eof in quoted literal")
			}
		}
	}
}

// lexUnquotedOrNumberLiteral is the state function for lexing names.
func lexUnquotedOrNumberLiteral(l *lexer) stateFn {
	var hasPlus bool

	sb := new(strings.Builder)

	for r := l.next(); isName(r) || r == '+'; r = l.next() {
		if r == '+' {
			hasPlus = true
		}

		sb.WriteRune(r)
	}

	l.backup()

	var number float64

	if err := json.Unmarshal([]byte(sb.String()), &number); err == nil {
		return l.emitItem(mk(itemNumberLiteral, sb.String()))
	}

	// "+" is not valid unquoted literal character
	if hasPlus {
		return l.emitErrorf(`invalid unquoted literal "%s"`, sb.String())
	}

	return l.emitItem(mk(itemUnquotedLiteral, sb.String()))
}

// lexLiteral is the state function for lexing variables.
func lexVariable(l *lexer) stateFn {
	sb := new(strings.Builder)

	if r := l.next(); r != variablePrefix {
		return l.emitErrorf(`invalid variable prefix "%s"`, string(r))
	}

	for r := l.next(); isName(r); r = l.next() {
		sb.WriteRune(r)
	}

	l.backup()

	return l.emitItem(mk(itemVariable, sb.String()))
}

// lexLiteral is the state function for reserved keywords.
func lexReservedKeyword(l *lexer) stateFn {
	sb := new(strings.Builder)

	if r := l.next(); r != '.' {
		return l.emitErrorf(`invalid reserved keyword prefix "%s"`, string(r))
	}

	for r := l.next(); isName(r); r = l.next() {
		sb.WriteRune(r)
	}

	l.backup()

	return l.emitItem(mk(itemReservedKeyword, sb.String()))
}

// lexWhitespace is the state function for lexing whitespace.
func lexWhitespace(l *lexer) stateFn {
	sb := new(strings.Builder)

	for {
		r := l.next()

		switch {
		default:
			l.backup()
			return l.emitItem(mk(itemWhitespace, sb.String()))
		case r == eof:
			return l.emitItem(mk(itemWhitespace, sb.String()))
		case isWhitespace(r):
			sb.WriteRune(r)
		}
	}
}

// lexIdentifier is the state function for lexing identifiers.
func lexIdentifier(l *lexer) stateFn {
	var (
		ns  bool
		typ itemType
	)

	sb := new(strings.Builder)

	for {
		r := l.next()

		switch {
		default:
			if sb.Len() == 0 && typ != itemMarkupClose {
				return l.emitErrorf("missing %s name", typ)
			}

			if sb.Len() > 0 && sb.String()[sb.Len()-1:] == ":" {
				return l.emitErrorf(`invalid %s name "%s"`, typ, sb.String())
			}

			l.backup()

			return l.emitItem(mk(typ, sb.String()))
		case typ == itemUnknown:
			switch r {
			default:
				typ = itemOption

				sb.WriteRune(r)
			case ':':
				l.isFunction = true
				typ = itemFunction
			case '#':
				l.isMarkup = true
				typ = itemMarkupOpen
			case '/':
				l.isMarkup = true
				typ = itemMarkupClose
			case '@':
				l.isFunction = false
				typ = itemAttribute
			}
		case isName(r):
			sb.WriteRune(r)
		case sb.Len() > 0 && r == ':':
			if ns {
				return l.emitErrorf("namespace already defined in identifier: %s", sb.String())
			}

			ns = true

			sb.WriteRune(r)
		case r == eof:
			return l.emitErrorf("unexpected eof in identifier")

		case sb.Len() == 0 && isNameStart(r):
			sb.WriteRune(r)
		}
	}
}

// ABNF:
//
//	reserved-body      = reserved-body-part *([s] reserved-body-part)
//	reserved-body-part = reserved-char / escaped-char / quoted
//	reserved-char      = content-char / "."
//	escaped-char       = backslash ( backslash / "{" / "|" / "}" )
//	quoted             = "|" *(quoted-char / escaped-char) "|"
func lexReservedBody(l *lexer) stateFn {
	sb := new(strings.Builder)

	for {
		switch r := l.next(); {
		case r == '{', r == '}', r == '@':
			l.backup()
			l.isReservedBody = false

			if sb.Len() == 0 {
				return lexExpr(l)
			}

			return l.emitItem(mk(itemReservedText, sb.String()))
		case isWhitespace(r):
			l.backup()

			if sb.Len() == 0 {
				return lexWhitespace(l)
			}

			return l.emitItem(mk(itemReservedText, sb.String()))
		case r == '|':
			l.backup()
			return lexQuotedLiteral(l)
		case r == '\\': // escaped character
			next := l.next()

			if !isEscapedChar(next) {
				return l.emitErrorf("unexpected escaped character in reserved body: %s", string(r))
			}

			sb.WriteRune(next)
		case isReserved(r):
			sb.WriteRune(r)
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
//	           / %xF900-FDCF / %xFDF0-FFFC / %x10000-EFFFF
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
		0xFDF0 <= r && r <= 0xFFFC ||
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

// isQuoted returns true if r is quoted character.
//
// ABNF:
//
// quoted-char = content-char / s / "." / "@" / "{" / "}".
func isQuoted(r rune) bool {
	return isContent(r) || isWhitespace(r) || r == '.' || r == '@' || r == '{' || r == '}'
}

// isWhitespace returns true if r is whitespace character.
//
// ABNF:
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
//
// ABNF:
//
//	reserved-annotation-start = "!" / "%" / "*" / "+" / "<" / ">" / "?" / "~"
func isReservedStart(r rune) bool {
	switch r {
	default:
		return false
	case '!', '%', '*', '+', '<', '>', '?', '~':
		return true
	}
}

// isReserved returns true if r is reserved character.
//
// ABNF:
//
//	reserved-char = content-char / ".".
func isReserved(r rune) bool {
	return isContent(r) || r == '.'
}

// isEscapedChar returns true if r is an escaped character.
//
// ABNF:
//
//	escaped-char = backslash ( backslash / "{" / "|" / "}" )
func isEscapedChar(r rune) bool {
	return r == '\\' || r == '{' || r == '|' || r == '}'
}

// isSimpleStart returns true if r is simple start character.
//
// ABNF:
//
//	simple-start-char = content-char / s / "@" / "|"
func isSimpleStart(r rune) bool {
	return isContent(r) || isWhitespace(r) || r == '@' || r == '|'
}

// isText returns true if r is text character.
//
// ABNF:
//
//	text-char = content-char / s / "." / "@" / "|"
func isText(r rune) bool {
	return isContent(r) || isWhitespace(r) || r == '.' || r == '@' || r == '|'
}

// isPrivateStart returns true if r is private start character.
//
// ABNF:
//
//	private-start = "^" / "&".
func isPrivateStart(r rune) bool {
	return r == '^' || r == '&'
}

// isContent returns true if r is content character.
//
// ABNF:
//
//	content-char = %x01-08       ; omit NULL (%x00), HTAB (%x09) and LF (%x0A)
//	               %x0B-0C       ; omit CR (%x0D)
//	               %x0E-1F       ; omit SP (%x20)
//	               %x21-2D       ; omit . (%x2E)
//	               %x2F-3F       ; omit @ (%x40)
//	               %x41-5B       ; omit \ (%x5C)
//	               %x5D-7A       ; omit { | } (%x7B-7D)
//	               %x7E-2FFF     ; omit IDEOGRAPHIC SPACE (%x3000)
//	               %x3001-D7FF   ; omit surrogates
//	               %xE000-10FFFF
func isContent(r rune) bool {
	return 0x01 <= r && r <= 0x08 || // omit NULL (%x00), HTAB (%x09) and LF (%x0A)
		0x0B <= r && r <= 0x0C || // omit CR (%x0D)
		0x0E <= r && r <= 0x1F || // omit SP (%x20)
		0x21 <= r && r <= 0x2D || // omit . (%x2E)
		0x2F <= r && r <= 0x3F || // omit @ (%x40)
		0x41 <= r && r <= 0x5B || // omit \ (%x5C)
		0x5D <= r && r <= 0x7A || // omit { | } (%x7B-7D)
		0x7E <= r && r <= 0x2FFF || // omit IDEOGRAPHIC SPACE (%x3000)
		0x3001 <= 3 && r <= 0xD7FF || // omit surrogates
		0xE000 <= r && r <= 0x10FFFF
}
