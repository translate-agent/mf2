package parse

import (
	"runtime"
	"testing"
)

func Test_lex(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		input string // MessageFormat2 formatted string
		want  []item
	}{
		{
			name:  "empty simple message",
			input: "",
			want:  []item{mk(itemEOF, "")},
		},
		{
			name:  "whitespace simple message",
			input: " ",
			want:  []item{mk(itemText, " "), mk(itemEOF, "")},
		},
		{
			name:  "escaped characters",
			input: `\\ \} \{ \|`,
			want: []item{
				mk(itemText, `\ } { |`),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "unescaped }",
			input: `}`,
			want: []item{
				mkErr(`unexpected start char "}"`),
			},
		},
		{
			name:  "function",
			input: "{:rand seed=1 log:level=$log lag:k=v o = $k @attr1=val1 @attr2}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemFunction, "rand"),
				mk(itemWhitespace, " "),
				mk(itemOption, "seed"),
				mk(itemOperator, "="),
				mk(itemNumberLiteral, "1"),
				mk(itemWhitespace, " "),
				mk(itemOption, "log:level"),
				mk(itemOperator, "="),
				mk(itemVariable, "log"),
				mk(itemWhitespace, " "),
				mk(itemOption, "lag:k"),
				mk(itemOperator, "="),
				mk(itemUnquotedLiteral, "v"),
				mk(itemWhitespace, " "),
				mk(itemOption, "o"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemVariable, "k"),
				mk(itemWhitespace, " "),
				mk(itemAttribute, "attr1"),
				mk(itemOperator, "="),
				mk(itemUnquotedLiteral, "val1"),
				mk(itemWhitespace, " "),
				mk(itemAttribute, "attr2"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "function syntax error",
			input: "{:func:}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mkErr(`invalid function name "func:"`),
			},
		},
		{
			name:  "bad placeholder",
			input: "{:}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mkErr("missing function name"),
			},
		},
		{
			name:  "variable",
			input: "{$count :math:round}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "count"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "math:round"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "text with variable",
			input: "Hello, {$guest}!",
			want: []item{
				mk(itemText, "Hello, "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "guest"),
				mk(itemExpressionClose, "}"),
				mk(itemText, "!"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "empty quoted literal",
			input: "{||}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemQuotedLiteral, ""),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "quoted literal",
			input: "{|\\| is escaped| :uppercase}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemQuotedLiteral, "| is escaped"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "uppercase"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "number literal",
			input: "{-1.9e+10 :odd}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemNumberLiteral, "-1.9e+10"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "odd"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "unquoted literal",
			input: "{hello :uppercase}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemUnquotedLiteral, "hello"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "uppercase"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "invalid unquoted literal",
			input: "{hello+world}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mkErr(`invalid unquoted literal "hello+world"`),
			},
		},
		{
			name:  "reserved annotations",
			input: `{!a}{%b c}{* \{}{+txt |quoted| txt}{ <r }{>}{?a b c}{ ~ a b c }text`,
			want: []item{
				// 1
				mk(itemExpressionOpen, "{"),
				mk(itemReservedStart, "!"),
				mk(itemReservedText, "a"),
				mk(itemExpressionClose, "}"),
				// 2
				mk(itemExpressionOpen, "{"),
				mk(itemReservedStart, "%"),
				mk(itemReservedText, "b"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "c"),
				mk(itemExpressionClose, "}"),
				// 3
				mk(itemExpressionOpen, "{"),
				mk(itemReservedStart, "*"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "{"),
				mk(itemExpressionClose, "}"),
				// 4
				mk(itemExpressionOpen, "{"),
				mk(itemReservedStart, "+"),
				mk(itemReservedText, "txt"),
				mk(itemWhitespace, " "),
				mk(itemQuotedLiteral, "quoted"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "txt"),
				mk(itemExpressionClose, "}"),
				// 5
				mk(itemExpressionOpen, "{"),
				mk(itemWhitespace, " "),
				mk(itemReservedStart, "<"),
				mk(itemReservedText, "r"),
				mk(itemWhitespace, " "),
				mk(itemExpressionClose, "}"),
				// 6
				mk(itemExpressionOpen, "{"),
				mk(itemReservedStart, ">"),
				mk(itemExpressionClose, "}"),
				// 7
				mk(itemExpressionOpen, "{"),
				mk(itemReservedStart, "?"),
				mk(itemReservedText, "a"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "b"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "c"),
				mk(itemExpressionClose, "}"),
				// 8
				mk(itemExpressionOpen, "{"),
				mk(itemWhitespace, " "),
				mk(itemReservedStart, "~"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "a"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "b"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "c"),
				mk(itemWhitespace, " "),
				mk(itemExpressionClose, "}"),
				//
				mk(itemText, "text"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "private use annotations",
			input: "{ ^ .body }{&|body| a}{^ \\|body \\}}{&hey}",
			want: []item{
				// 1
				mk(itemExpressionOpen, "{"),
				mk(itemWhitespace, " "),
				mk(itemPrivateStart, "^"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, ".body"),
				mk(itemWhitespace, " "),
				mk(itemExpressionClose, "}"),
				// 2
				mk(itemExpressionOpen, "{"),
				mk(itemPrivateStart, "&"),
				mk(itemQuotedLiteral, "body"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "a"),
				mk(itemExpressionClose, "}"),
				// 3
				mk(itemExpressionOpen, "{"),
				mk(itemPrivateStart, "^"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "|body"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "}"),
				mk(itemExpressionClose, "}"),
				// 4 Without whitespace
				mk(itemExpressionOpen, "{"),
				mk(itemPrivateStart, "&"),
				mk(itemReservedText, "hey"),
				mk(itemExpressionClose, "}"),
				//
				mk(itemEOF, ""),
			},
		},
		{
			name:  "local declaration",
			input: ".local $hostName = {$host} .local $h = {|host| :func @a=1}",
			want: []item{
				// .local $hostName = {$host}
				mk(itemLocalKeyword, "local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "hostName"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "host"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .local $h = {|host| :func @a=1}
				mk(itemLocalKeyword, "local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "h"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemQuotedLiteral, "host"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "func"),
				mk(itemWhitespace, " "),
				mk(itemAttribute, "a"),
				mk(itemOperator, "="),
				mk(itemNumberLiteral, "1"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "input declaration",
			input: ".input {$host} .input {$user :func @a} .input {$num ^private}",
			want: []item{
				// .input {$host}
				mk(itemInputKeyword, "input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "host"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .input {$user :func @a}
				mk(itemInputKeyword, "input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "user"),
				mk(itemWhitespace, " "),
				mk(itemFunction, "func"),
				mk(itemWhitespace, " "),
				mk(itemAttribute, "a"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .input {$num ^private}
				mk(itemInputKeyword, "input"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "num"),
				mk(itemWhitespace, " "),
				mk(itemPrivateStart, "^"),
				mk(itemReservedText, "private"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		//nolint:dupword
		{
			name: "reserved statement",
			input: ".first {$var} " +
				".second body1 |body2| {|quoted| ^exprBody} " +
				".third ho ho ho {$var !reserved} {:func @a} {2} {{.}}",
			want: []item{
				// .first {$var} // 1 expression
				mk(itemReservedKeyword, "first"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "var"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .second body1 |body2| {|quoted| ^exprBody} // reservedBody + expression
				mk(itemReservedKeyword, "second"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "body1"),
				mk(itemWhitespace, " "),
				mk(itemQuotedLiteral, "body2"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemQuotedLiteral, "quoted"),
				mk(itemWhitespace, " "),
				mk(itemPrivateStart, "^"),
				mk(itemReservedText, "exprBody"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// .third ho ho ho {$var !reserved} {:func @a} {2} // reservedBody + 3 expressions
				mk(itemReservedKeyword, "third"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "ho"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "ho"),
				mk(itemWhitespace, " "),
				mk(itemReservedText, "ho"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "var"),
				mk(itemWhitespace, " "),
				mk(itemReservedStart, "!"),
				mk(itemReservedText, "reserved"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemFunction, "func"),
				mk(itemWhitespace, " "),
				mk(itemAttribute, "a"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemNumberLiteral, "2"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				//
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemText, "."),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "matcher",
			input: ".match {$n} 0 {{no apples}} 1 {{{$n} apple}} * {{{$n} apples}}",
			want: []item{
				mk(itemMatchKeyword, "match"),
				mk(itemWhitespace, " "),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "n"),
				mk(itemExpressionClose, "}"),
				mk(itemWhitespace, " "),
				// 0 {{no apples}}
				mk(itemNumberLiteral, "0"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemText, "no apples"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, " "),
				// 1 {{{$n} apple}}
				mk(itemNumberLiteral, "1"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "n"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " apple"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, " "),
				// * {{{$n} apples}}
				mk(itemCatchAllKey, "*"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "n"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " apples"),
				mk(itemQuotedPatternClose, "}}"),

				mk(itemEOF, ""),
			},
		},
		{
			name:  "complex message with unexpected }",
			input: "{{}}}",
			want: []item{
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemQuotedPatternClose, "}}"),
				mkErr("unexpected } in complex message"),
			},
		},
		{
			name:  "complex message without declaration",
			input: "{{Hello, {|literal|} World!}}",
			want: []item{
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemText, "Hello, "),
				mk(itemExpressionOpen, "{"),
				mk(itemQuotedLiteral, "literal"),
				mk(itemExpressionClose, "}"),
				mk(itemText, " World!"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "variable with _",
			input: "{$csv_filename}",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemVariable, "csv_filename"),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "variable with whitespace",
			input: "{ $csv_filename }",
			want: []item{
				mk(itemExpressionOpen, "{"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "csv_filename"),
				mk(itemWhitespace, " "),
				mk(itemExpressionClose, "}"),
				mk(itemEOF, ""),
			},
		},
		{
			name: "markup",
			input: `{#button}Submit{/button}
{#img alt=|Cancel| @hello=world @goodbye /}
{ #nest1}{#nest2}text{#nest3/}{/nest2}{/nest1 a=b}`,
			want: []item{
				// 1. simple open-close
				mk(itemExpressionOpen, "{"),
				mk(itemMarkupOpen, "button"),
				mk(itemExpressionClose, "}"),
				mk(itemText, "Submit"),
				mk(itemExpressionOpen, "{"),
				mk(itemMarkupClose, "button"),
				mk(itemExpressionClose, "}"),
				mk(itemText, "\n"),

				// 2. self-closing + options + attributes
				mk(itemExpressionOpen, "{"),
				mk(itemMarkupOpen, "img"),
				mk(itemWhitespace, " "),
				mk(itemOption, "alt"),
				mk(itemOperator, "="),
				mk(itemQuotedLiteral, "Cancel"),
				mk(itemWhitespace, " "),
				mk(itemAttribute, "hello"),
				mk(itemOperator, "="),
				mk(itemUnquotedLiteral, "world"),
				mk(itemWhitespace, " "),
				mk(itemAttribute, "goodbye"),
				mk(itemWhitespace, " "),
				mk(itemMarkupClose, ""),
				mk(itemExpressionClose, "}"),
				mk(itemText, "\n"),

				// 3. nested
				mk(itemExpressionOpen, "{"),
				mk(itemWhitespace, " "),
				mk(itemMarkupOpen, "nest1"),
				mk(itemExpressionClose, "}"),
				mk(itemExpressionOpen, "{"),
				mk(itemMarkupOpen, "nest2"),
				mk(itemExpressionClose, "}"),
				mk(itemText, "text"),
				mk(itemExpressionOpen, "{"),
				mk(itemMarkupOpen, "nest3"),
				mk(itemMarkupClose, ""),
				mk(itemExpressionClose, "}"),
				mk(itemExpressionOpen, "{"),
				mk(itemMarkupClose, "nest2"),
				mk(itemExpressionClose, "}"),
				mk(itemExpressionOpen, "{"),
				mk(itemMarkupClose, "nest1"),
				mk(itemWhitespace, " "),
				mk(itemOption, "a"),
				mk(itemOperator, "="),
				mk(itemUnquotedLiteral, "b"),
				mk(itemExpressionClose, "}"),
				//
				mk(itemEOF, ""),
			},
		},
		{
			name:  "head and tail whitespaces",
			input: "  {{}}  ",
			want: []item{
				mk(itemWhitespace, "  "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, "  "),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "whitespace with declarations",
			input: "\t.local $foo =bar {{}}\n",
			want: []item{
				mk(itemWhitespace, "\t"),
				mk(itemLocalKeyword, "local"),
				mk(itemWhitespace, " "),
				mk(itemVariable, "foo"),
				mk(itemWhitespace, " "),
				mk(itemOperator, "="),
				mk(itemUnquotedLiteral, "bar"),
				mk(itemWhitespace, " "),
				mk(itemQuotedPatternOpen, "{{"),
				mk(itemQuotedPatternClose, "}}"),
				mk(itemWhitespace, "\n"),
				mk(itemEOF, ""),
			},
		},
		{
			name:  "no whitespace in simple message, unless inside expression",
			input: "  { |simple| }  ",
			want: []item{
				mk(itemText, "  "),
				mk(itemExpressionOpen, "{"),
				mk(itemWhitespace, " "),
				mk(itemQuotedLiteral, "simple"),
				mk(itemWhitespace, " "),
				mk(itemExpressionClose, "}"),
				mk(itemText, "  "),
				mk(itemEOF, ""),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assertItems(t, test.want, lex(test.input))
		})
	}
}

func assertItems(t *testing.T, want []item, l *lexer) {
	t.Helper()

	got := make([]lexer, 0, len(want))

	// collect all lexer states

	for {
		itm := l.nextItem()
		got = append(got, *l)

		if itm.typ == itemEOF || itm.typ == itemError {
			break
		}
	}

	// asserts

	if len(want) != len(got) {
		t.Errorf("want %d items, got %d", len(want), len(got))
	}

	for i, wantItem := range want {
		if i >= len(got) {
			t.Fatalf("want %v, got nothing", wantItem)
		}

		gotItem := got[i].item

		if wantItem.typ == itemError {
			if wantItem.err == nil || gotItem.err == nil || wantItem.err.Error() != gotItem.err.Error() {
				t.Errorf(`want error '%v', got '%v'`, wantItem.err, gotItem.err)
			}

			return
		}

		if wantItem != gotItem {
			t.Errorf(`want '%v', got '%v'`, wantItem, gotItem)

			for j := range i + 1 {
				logItem(t, want[j], got[j])
			}
		}
	}
}

func logItem(t *testing.T, want item, l lexer) {
	t.Helper()

	f := func(b bool) string {
		if b {
			return "✓"
		}

		return " "
	}

	wantVal := want.val
	if want.typ == itemError {
		wantVal = want.err.Error()
	}

	val := l.item.val
	if l.item.typ == itemError {
		val = l.item.err.Error()
	}

	t.Logf("c%s p%s e%s f%s r%s m%s %-30s e%s(%s) a%s(%s)\n",
		f(l.isComplexMessage), f(l.isPattern), f(l.isExpression), f(l.isFunction), f(l.isReservedBody), f(l.isMarkup),
		"'"+l.input[l.pos:]+"'", "'"+wantVal+"'", want.typ, "'"+val+"'", l.item.typ)
}

func BenchmarkLex(b *testing.B) {
	var itm item

	for range b.N {
		lexer := lex("  .input {$foo :number} .local $bar = {$foo} .match {$bar} one {{one}} * {{other}}  ")

		for {
			itm = lexer.nextItem()
			if itm.typ == itemEOF {
				break
			}
		}
	}

	runtime.KeepAlive(itm)
}
