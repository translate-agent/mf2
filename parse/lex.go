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
	itemCatchAllKey
	itemNumberLiteral
	itemQuotedLiteral
	itemUnquotedLiteral
	itemOption
	itemAttribute
	itemWhitespace
	itemOperator
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

// mkErrorf creates a new error item with the given format and args.
func mkErrorf(format string, args ...any) item {
	return item{typ: itemError, err: fmt.Errorf(format, args...)}
}

// lex creates a new lexer for the given input string.
func lex(input string) *lexer {
	return &lexer{
		input: input,
		line:  1,
	}
}

// lexer is a lexical analyzer for MessageFormat2.
//
// See ".message-format-wg/spec/message.abnf".
type lexer struct {
	input      string
	item       item     // previous item or start char with optional preceding whitespaces in simple message
	prevType   itemType // previous non-whitespace item type
	start, end int      // start and end positions of the item to be emitted
	line       int      // line number

	isFunction,
	isMarkup,
	isExpression,
	isPattern,
	isComplexMessage bool
}

// peek peeks at the next rune.
func (l *lexer) peek() rune {
	if l.end < 0 || len(l.input) <= l.end { // isSliceInBounds()
		return eof
	}

	r, _ := utf8.DecodeRuneInString(l.input[l.end:])

	return r
}

// next returns the next rune.
func (l *lexer) next() rune {
	if l.end < 0 || len(l.input) <= l.end { // isSliceInBounds()
		return eof
	}

	r, n := utf8.DecodeRuneInString(l.input[l.end:])
	l.end += n

	if r == '\n' {
		l.line++
	}

	return r
}

// backup backs up the current position in the input string.
func (l *lexer) backup() {
	r, n := utf8.DecodeLastRuneInString(l.input[:l.end])

	if r == '\n' {
		l.line--
	}

	l.end -= n
}

// nextItem returns the next item in the input string.
func (l *lexer) nextItem() item {
	l.item = mk(itemEOF, "")

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
	case l.end == 0:
		state = lexStart
	}

	for {
		if state := state(l); state == nil {
			return l.item
		}
	}
}

// emitItem emits the given item and returns the next state function.
func (l *lexer) emitItem(i item) stateFn {
	l.start = l.end
	l.item = i

	if i.typ != itemWhitespace && i.typ != itemEOF {
		l.prevType = i.typ
	}

	return nil
}

// emit emits the item to be emitted.
func (l *lexer) emit(typ itemType) stateFn {
	return l.emitItem(mk(typ, l.val()))
}

// val returns the value of the item to be emitted.
func (l *lexer) val() string {
	if 0 <= l.start && l.end <= len(l.input) && l.start <= l.end { // IsSliceInBounds()
		return l.input[l.start:l.end]
	}

	return ""
}

// emitErrorf emits the error and returns the next state function.
func (l *lexer) emitErrorf(s string, args ...any) stateFn {
	return l.emitItem(mkErrorf(s, args...))
}

// stateFn is a function that returns the next state function.
type stateFn func(*lexer) stateFn

// lexStart is the state function to lex the start of the MF2.
func lexStart(l *lexer) stateFn {
	complexItem := func() stateFn {
		l.isComplexMessage = true
		l.backup()

		if l.start < l.end {
			return l.emit(itemWhitespace)
		}

		return lexComplexMessage(l)
	}

	simpleItem := func() stateFn {
		return lexPattern(l)
	}

	for {
		r := l.next()

		switch {
		default:
			return l.emitErrorf(`unexpected start char "%c"`, r)
		case isWhitespace(r):
		case isSimpleStart(r):
			return simpleItem()
		case r == '\\':
			l.backup()

			return simpleItem()
		case r == '.':
			return complexItem()
		case r == '{':
			if l.peek() == '{' {
				return complexItem()
			}

			// expression in simple message

			l.backup()

			return lexPattern(l)
		case r == eof:
			if l.start < l.end {
				return l.emit(itemText)
			}

			return nil
		}
	}
}

// lexPattern is the state function for lexing patterns.
func lexPattern(l *lexer) stateFn {
	sb := new(strings.Builder)
	if l.start == 0 { // whitespace at the start of the MF2 if any
		sb.WriteString(l.val())
	}

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
				return l.emitErrorf(`unexpected escaped char "%c" in pattern`, next)
			}

			sb.WriteRune(next)
		case r == '{':
			l.backup()

			if sb.Len() > 0 {
				l.isExpression = true

				return l.emitItem(mk(itemText, sb.String()))
			}

			return lexExpr(l)
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
		case isText(r):
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
			return l.emitErrorf(`bad character "%c" in complex message`, r)
		case r == '.':
			input := l.input[l.end:]

			switch {
			default:
				return l.emitErrorf(`invalid keyword`)
			case strings.HasPrefix(input, keywordLocal):
				l.start++ // skip .
				l.end += len(keywordLocal)

				return l.emit(itemLocalKeyword)
			case strings.HasPrefix(input, keywordInput):
				l.start++ // skip .
				l.end += len(keywordInput)

				return l.emit(itemInputKeyword)
			case strings.HasPrefix(input, keywordMatch):
				l.start++ // skip .
				l.end += len(keywordMatch)

				return l.emit(itemMatchKeyword)
			}
		case r == variablePrefix:
			l.start++ // skip $
			return lexName(l, itemVariable)
		case isWhitespace(r):
			return lexWhitespace(l)
		case r == '=':
			return l.emit(itemOperator)
		case r == '{':
			if l.peek() == '{' {
				l.next()
				l.isPattern = true

				return l.emit(itemQuotedPatternOpen)
			}

			l.backup()

			return lexExpr(l)
		case r == '}':
			if l.peek() == '}' {
				l.next()
				l.isPattern = false

				return l.emit(itemQuotedPatternClose)
			}

			return l.emitErrorf("unexpected } in complex message")
		case r == '*':
			return l.emitItem(mk(itemCatchAllKey, "*"))
		case r == '|':
			l.backup()

			return lexQuotedLiteral(l)
		case isNameStart(r):
			return lexUnquotedLiteral(l)
		case r == '-' || '0' <= r && r <= '9':
			return lexNumberLiteral(l)
		case r == eof:
			return nil
		}
	}
}

// lexExpr is the state function for lexing expressions.
func lexExpr(l *lexer) stateFn {
	switch r := l.next(); {
	default:
		return l.emitErrorf(`bad character "%c" in expression`, r)
	case r == variablePrefix:
		l.start++ // skip $
		return lexName(l, itemVariable)
	case r == '|':
		l.backup()
		return lexQuotedLiteral(l)
	case r == ':':
		l.isFunction = true
		l.start++ // skip :

		return lexIdentifier(l, itemFunction)
	case r == '@':
		l.isFunction = false
		l.start++ // skip @

		return lexIdentifier(l, itemAttribute)
	case r == '#':
		l.isMarkup = true
		l.start++ // skip #

		return lexIdentifier(l, itemMarkupOpen)
	case r == '/':
		if l.isMarkup {
			return l.emitItem(mk(itemMarkupClose, ""))
		}

		l.isMarkup = true
		l.start++ // skip /

		return lexIdentifier(l, itemMarkupClose)
	case r == '{': // expression/markup start
		l.isExpression = true

		return l.emit(itemExpressionOpen)
	case r == '}': // expression/markup end
		l.isExpression = false
		l.isFunction = false
		l.isMarkup = false

		return l.emit(itemExpressionClose)
	case isWhitespace(r):
		return lexWhitespace(l)
	case (l.isFunction || l.isMarkup) && isNameStart(r):
		l.backup()

		if l.prevType == itemOperator {
			return lexName(l, itemUnquotedLiteral)
		}

		return lexIdentifier(l, itemOption)
	case r == '=':
		return l.emit(itemOperator)
	case isNameStart(r):
		return lexUnquotedLiteral(l)
	case r == '-' || '0' <= r && r <= '9':
		return lexNumberLiteral(l)
	case r == eof:
		return l.emitErrorf("unexpected eof in expression")
	}
}

// lexQuotedLiteral is the state function for lexing quoted literals.
func lexQuotedLiteral(l *lexer) stateFn {
	sb := new(strings.Builder)

	// discard opening quote |
	l.next()

	for {
		r := l.next()

		switch {
		default:
			return l.emitErrorf(`unknown character "%c" in quoted literal`, r)
		case isQuoted(r):
			sb.WriteRune(r)
		case r == '|': // closing
			return l.emitItem(mk(itemQuotedLiteral, sb.String()))
		case r == '\\':
			next := l.next()

			if !isEscapedChar(next) {
				return l.emitErrorf(`unexpected escaped character "%c" in quoted literal`, next)
			}

			sb.WriteRune(next)
		}
	}
}

// lexUnquotedLiteral is the state function for lexing unquoted literals.
// The first character is already lexed.
func lexUnquotedLiteral(l *lexer) stateFn {
	for {
		r := l.next()

		switch {
		default:
			l.backup()

			return l.emit(itemUnquotedLiteral)
		case isName(r): // noop
		case r == eof:
			return l.emit(itemUnquotedLiteral)
		}
	}
}

// lexNumberLiteral is the state function for lexing number literals.
// The first character is already lexed.
//
// ABNF:
//
//	number-literal   = ["-"] (%x30 / (%x31-39 *DIGIT)) ["." 1*DIGIT] [%i"e" ["-" / "+"] 1*DIGIT]
func lexNumberLiteral(l *lexer) stateFn {
	emit := func() stateFn {
		var number float64

		val := l.val()

		if err := json.Unmarshal([]byte(val), &number); err != nil {
			return l.emitErrorf(`bad number literal "%s"`, val)
		}

		return l.emit(itemNumberLiteral)
	}

	for {
		r := l.next()

		switch {
		default:
			l.backup()
			return emit()
		case '0' <= r && r <= '9', r == '.', r == 'e', r == '-', r == '+': // noop
		case r == eof:
			return emit()
		}
	}
}

// lexWhitespace is the state function for lexing whitespace.
// The first character is already lexed.
func lexWhitespace(l *lexer) stateFn {
	for {
		r := l.next()

		switch {
		default:
			l.backup()
			return l.emit(itemWhitespace)
		case isWhitespace(r):
		case r == eof:
			return l.emit(itemWhitespace)
		}
	}
}

// lexName is the state function for lexing names.
func lexName(l *lexer, typ itemType) stateFn {
	r := l.next()
	if !isNameStart(r) {
		return l.emitErrorf(`bad %s name "%s"`, typ, string(r))
	}

	for {
		if r = l.next(); !isName(r) {
			break
		}
	}

	if r != eof {
		l.backup()
	}

	return l.emit(typ)
}

// lexIdentifier is the state function for lexing identifiers.
func lexIdentifier(l *lexer, typ itemType) stateFn {
	r := l.next()
	if !isNameStart(r) {
		return l.emitErrorf(`bad %s identifier "%s"`, typ, string(r))
	}

	for {
		if r = l.next(); !isName(r) {
			break
		}
	}

	switch r {
	default:
		l.backup()

		return l.emit(typ)
	case ':': // identifier with namespace
	case eof:
		return l.emit(typ)
	}

	r = l.next()

	if !isNameStart(r) {
		return l.emitErrorf(`bad %s identifier "%s"`, typ, l.val())
	}

	for {
		if r = l.next(); !isName(r) {
			break
		}
	}

	if r != eof {
		l.backup()
	}

	return l.emit(typ)
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
		0x37F <= r && r <= 0x61B ||
		0x61D <= r && r <= 0x1FFF ||
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
	case '\u061C', '\u200E', '\u200F', '\u2066', '\u2067', '\u2068', '\u2069':
		// TODO: should we separate it into `bidi`?
		return true
	case ' ', '\t', '\r', '\n', '\u3000':
		return true
	}
}

// isEscapedChar returns true if r is an escaped character.
//
// ABNF:
//
//	escaped-char = backslash ( backslash / "{" / "|" / "}" )
func isEscapedChar(r rune) bool {
	switch r {
	default:
		return false
	case '\\', '{', '|', '}':
		return true
	}
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
