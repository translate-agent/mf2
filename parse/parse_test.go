package parse

import (
	"runtime"
	"strings"
	"testing"
)

func TestParseSimpleMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  Message
		input string
	}{
		{
			name:  "text only",
			input: "Hello, World!",
			want: SimpleMessage{
				Text("Hello, World!"),
			},
		},
		{
			name:  "text only with escaped chars",
			input: "Hello, \\{World!\\}",
			want: SimpleMessage{
				Text("Hello, {World!}"),
			},
		},
		{
			name:  "variable expression in the middle",
			input: "Hello, { $variable } World!",
			want: SimpleMessage{
				Text("Hello, "),
				Expression{Operand: Variable("variable")},
				Text(" World!"),
			},
		},
		{
			name:  "variable expression at the start",
			input: "{ $variable } Hello, World!",
			want: SimpleMessage{
				Expression{Operand: Variable("variable")},
				Text(" Hello, World!"),
			},
		},
		{
			name:  "variable expression at the end",
			input: "Hello, World! { $variable }",
			want: SimpleMessage{
				Text("Hello, World! "),
				Expression{Operand: Variable("variable")},
			},
		},
		{
			name:  "variable expression with annotation",
			input: "Hello, { $variable :function }  World!",
			want: SimpleMessage{
				Text("Hello, "),
				Expression{
					Operand: Variable("variable"),
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
					},
				},
				Text("  World!"),
			},
		},
		{
			name:  "variable expression with annotation options and attributes",
			input: "Hello, { $variable :function option1 = -3.14 ns:option2 = |value2| option3 = $variable2 @attr1 = attr1} World!", //nolint:lll
			want: SimpleMessage{
				Text("Hello, "),
				Expression{
					Operand: Variable("variable"),
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
						Options: []Option{
							{
								Value: NumberLiteral("-3.14"),
								Identifier: Identifier{
									Namespace: "",
									Name:      "option1",
								},
							},
							{
								Value: QuotedLiteral("value2"),
								Identifier: Identifier{
									Namespace: "ns",
									Name:      "option2",
								},
							},
							{
								Value: Variable("variable2"),
								Identifier: Identifier{
									Namespace: "",
									Name:      "option3",
								},
							},
						},
					},
					Attributes: []Attribute{
						{
							Value:      NameLiteral("attr1"),
							Identifier: Identifier{Name: "attr1"},
						},
					},
				},
				Text(" World!"),
			},
		},
		{
			name:  "quoted literal expression",
			input: "Hello, { |literal| }  World!",
			want: SimpleMessage{
				Text("Hello, "),
				Expression{Operand: QuotedLiteral("literal")},
				Text("  World!"),
			},
		},
		{
			name:  "unquoted scientific notation number literal expression",
			input: "Hello, { 1e3 }  World!",
			want: SimpleMessage{
				Text("Hello, "),
				Expression{Operand: NumberLiteral("1e3")},
				Text("  World!"),
			},
		},
		{
			name:  "unquoted name literal expression",
			input: "Hello, { name } World!",
			want: SimpleMessage{
				Text("Hello, "),
				Expression{Operand: NameLiteral("name")},
				Text(" World!"),
			},
		},
		{
			name:  "quoted name literal expression with annotation",
			input: "Hello, { |name| :function } World!",
			want: SimpleMessage{
				Text("Hello, "),
				Expression{
					Operand: QuotedLiteral("name"),
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
					},
				},
				Text(" World!"),
			},
		},
		{
			name:  "quoted name literal expression with annotation and options",
			input: "Hello, { |name| :function ns1:option1 = -1 ns2:option2 = 1 option3 = |value3| } World!",
			want: SimpleMessage{
				Text("Hello, "),
				Expression{
					Operand: QuotedLiteral("name"),
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
						Options: []Option{
							{
								Value: NumberLiteral("-1"),
								Identifier: Identifier{
									Namespace: "ns1",
									Name:      "option1",
								},
							},
							{
								Value: NumberLiteral("1"),
								Identifier: Identifier{
									Namespace: "ns2",
									Name:      "option2",
								},
							},
							{
								Value: QuotedLiteral("value3"),
								Identifier: Identifier{
									Namespace: "",
									Name:      "option3",
								},
							},
						},
					},
				},
				Text(" World!"),
			},
		},
		{
			name:  "function expression",
			input: "Hello { :function } World!",
			want: SimpleMessage{
				Text("Hello "),
				Expression{
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
					},
				},
				Text(" World!"),
			},
		},
		{
			name:  "function expression with options and namespace",
			input: "Hello { :namespace:function namespace:option999 = 999 } World!",
			want: SimpleMessage{
				Text("Hello "),
				Expression{
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "namespace",
							Name:      "function",
						},
						Options: []Option{
							{
								Value: NumberLiteral("999"),
								Identifier: Identifier{
									Namespace: "namespace",
									Name:      "option999",
								},
							},
						},
					},
				},
				Text(" World!"),
			},
		},
		{
			name:  "markup",
			input: `It is a {#button opt1=val1 @attr1=val1 } button { /button } this is a { #br /} something else, {#ns:tag1}{#tag2}text{ #img /}{/tag2}{/ns:tag1}`, //nolint:lll
			want: SimpleMessage{
				// 1. Open-Close markup
				Text("It is a "),
				Markup{
					Typ: Open,
					Identifier: Identifier{
						Namespace: "",
						Name:      "button",
					},
					Options: []Option{
						{
							Value: NameLiteral("val1"),
							Identifier: Identifier{
								Name: "opt1",
							},
						},
					},
					Attributes: []Attribute{
						{
							Value:      NameLiteral("val1"),
							Identifier: Identifier{Name: "attr1"},
						},
					},
				},
				Text(" button "),
				Markup{Typ: Close, Identifier: Identifier{Name: "button"}},
				// 2. Self-close markup
				Text(" this is a "),
				Markup{Typ: SelfClose, Identifier: Identifier{Name: "br"}},
				Text(" something else, "),
				// 3. Nested markup
				Markup{Typ: Open, Identifier: Identifier{Namespace: "ns", Name: "tag1"}},
				Markup{Typ: Open, Identifier: Identifier{Name: "tag2"}},
				Text("text"),
				Markup{Typ: SelfClose, Identifier: Identifier{Name: "img"}},
				Markup{Typ: Close, Identifier: Identifier{Name: "tag2"}},
				Markup{Typ: Close, Identifier: Identifier{Namespace: "ns", Name: "tag1"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(test.input)
			if err != nil {
				t.Error(err)
				return
			}

			// Check that AST message is equal to expected one.
			if test.want.String() != got.Message.String() {
				t.Errorf("want '%v', got '%v'", test.want, got.Message)
			}

			// Check that AST message converted back to string is equal to input.

			// Edge case: scientific notation number is converted to normal notation, hence comparison is bound to fail.
			// I.E. input string has 1e3, output string has 1000.
			if test.name == "unquoted scientific notation number literal expression" {
				return
			}

			// If strings already match, we're done.
			// Otherwise check both sanitized strings.
			if got := got.String(); got != test.input {
				requireEqualMF2String(t, test.input, got)
			}
		})
	}
}

func TestParseComplexMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  Message
		input string
	}{
		{
			name:  "no declarations",
			input: "{{Hello, { |literal| } World!}}",
			want: ComplexMessage{
				Declarations: nil,
				ComplexBody: QuotedPattern{
					Text("Hello, "),
					Expression{Operand: QuotedLiteral("literal")},
					Text(" World!"),
				},
			},
		},
		{
			name:  "all declarations",
			input: `.input{$input :number @a} .local $local1={1} {{Text}}`,
			want: ComplexMessage{
				Declarations: []Declaration{
					// .input{$input :number @a}
					InputDeclaration{
						Operand:    Variable("input"),
						Annotation: Function{Identifier: Identifier{Name: "number"}},
						Attributes: []Attribute{{Identifier: Identifier{Name: "a"}}},
					},
					// .local $local1={1}
					LocalDeclaration{
						Variable:   Variable("local1"),
						Expression: Expression{Operand: NumberLiteral("1")},
					},
				},
				ComplexBody: QuotedPattern{Text("Text")},
			},
		},
		// Matcher
		{
			name:  "simple matcher one line",
			input: ".input { $variable :number } .match $variable 1 {{Hello { $variable} world}} * {{Hello { $variable } worlds}}", //nolint:lll
			want: ComplexMessage{
				Declarations: []Declaration{
					InputDeclaration{
						Operand: Variable("variable"),
						Annotation: Function{
							Identifier: Identifier{Namespace: "", Name: "number"},
						},
					},
				},
				ComplexBody: Matcher{
					Selectors: []Variable{
						Variable("variable"),
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{NumberLiteral("1")},
							QuotedPattern: QuotedPattern{
								Text("Hello "),
								Expression{Operand: Variable("variable")},
								Text(" world"),
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								Text("Hello "),
								Expression{Operand: Variable("variable")},
								Text(" worlds"),
							},
						},
					},
				},
			},
		},
		{
			name: "simple matcher with newline variants",
			input: `.input { $variable :number } .match $variable
1 {{Hello { $variable } world}}
* {{Hello { $variable } worlds}}`,
			want: ComplexMessage{
				Declarations: []Declaration{
					InputDeclaration{
						Operand: Variable("variable"),
						Annotation: Function{
							Identifier: Identifier{Namespace: "", Name: "number"},
						},
					},
				},
				ComplexBody: Matcher{
					Selectors: []Variable{
						Variable("variable"),
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{NumberLiteral("1")},
							QuotedPattern: QuotedPattern{
								Text("Hello "),
								Expression{Operand: Variable("variable")},
								Text(" world"),
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								Text("Hello "),
								Expression{Operand: Variable("variable")},
								Text(" worlds"),
							},
						},
					},
				},
			},
		},
		{
			name: "simple matcher with newline variants in one line",
			input: `.input { $variable :number } .match $variable

1 {{Hello { $variable} world}}* {{Hello { $variable } worlds}}`,
			want: ComplexMessage{
				Declarations: []Declaration{
					InputDeclaration{
						Operand: Variable("variable"),
						Annotation: Function{
							Identifier: Identifier{Namespace: "", Name: "number"},
						},
					},
				},
				ComplexBody: Matcher{
					Selectors: []Variable{
						Variable("variable"),
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{NumberLiteral("1")},
							QuotedPattern: QuotedPattern{
								Text("Hello "),
								Expression{Operand: Variable("variable")},
								Text(" world"),
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								Text("Hello "),
								Expression{Operand: Variable("variable")},
								Text(" worlds"),
							},
						},
					},
				},
			},
		},
		{
			name: "matcher with declarations",
			input: `.local $var1 = { male }
.local $var2 = { |female| }
.local $var3 = { :gender }
.match $var3
male {{Hello sir!}}
|female| {{Hello madam!}}
* {{Hello { $var1 } or { $var2 }!}}`,
			want: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable:   Variable("var1"),
						Expression: Expression{Operand: NameLiteral("male")},
					},
					LocalDeclaration{
						Variable:   Variable("var2"),
						Expression: Expression{Operand: QuotedLiteral("female")},
					},
					LocalDeclaration{
						Variable:   Variable("var3"),
						Expression: Expression{Annotation: Function{Identifier: Identifier{Namespace: "", Name: "gender"}}},
					},
				},
				ComplexBody: Matcher{
					Selectors: []Variable{
						Variable("var3"),
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{NameLiteral("male")},
							QuotedPattern: QuotedPattern{
								Text("Hello sir!"),
							},
						},
						{
							Keys: []VariantKey{QuotedLiteral("female")},
							QuotedPattern: QuotedPattern{
								Text("Hello madam!"),
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								Text("Hello "),
								Expression{Operand: Variable("var1")},
								Text(" or "),
								Expression{Operand: Variable("var2")},
								Text("!"),
							},
						},
					},
				},
			},
		},
		{
			name: "double matcher",
			//nolint:dupword
			input: `.input {$var1 :string}
.input {$var2 :string}
.match $var1 $var2
yes yes {{Hello beautiful world!}}
yes no {{Hello beautiful!}}
no yes {{Hello world!}}
* * {{Hello!}}`,
			want: ComplexMessage{
				Declarations: []Declaration{
					InputDeclaration{
						Operand:    Variable("var1"),
						Annotation: Function{Identifier: Identifier{Name: "string"}},
					},
					InputDeclaration{
						Operand:    Variable("var2"),
						Annotation: Function{Identifier: Identifier{Name: "string"}},
					},
				},
				ComplexBody: Matcher{
					Selectors: []Variable{
						Variable("var1"),
						Variable("var2"),
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{
								NameLiteral("yes"),
								NameLiteral("yes"),
							},
							QuotedPattern: QuotedPattern{
								Text("Hello beautiful world!"),
							},
						},
						{
							Keys: []VariantKey{
								NameLiteral("yes"),
								NameLiteral("no"),
							},
							QuotedPattern: QuotedPattern{
								Text("Hello beautiful!"),
							},
						},
						{
							Keys: []VariantKey{
								NameLiteral("no"),
								NameLiteral("yes"),
							},
							QuotedPattern: QuotedPattern{
								Text("Hello world!"),
							},
						},
						{
							Keys: []VariantKey{
								CatchAllKey{},
								CatchAllKey{},
							},
							QuotedPattern: QuotedPattern{
								Text("Hello!"),
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(test.input)
			if err != nil {
				t.Error(err)
			}

			// Check that AST message is equal to expected one.
			if test.want.String() != got.Message.String() {
				t.Errorf("want '%s', got '%s'", test.want, got.Message.String())
			}

			// Check that AST message converted back to string is equal to input.

			// If strings already match, we're done.
			// Otherwise check both sanitized strings.
			if got := got.String(); got != test.input {
				requireEqualMF2String(t, test.input, got)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		in, wantErr string
	}{
		{
			in:      ".local $foo={|foo| :x} .local $bar ={|bar| :x} .match $foo $bar ** {{foo}}",
			wantErr: "parse MF2: syntax error: complex message: matcher: variant keys: missing space between keys * and *",
		},
		{
			in:      ".local $foo= {|foo| :x} .local $bar = {|bar| :x} .match $foo $bar *1 {{foo}}",
			wantErr: "parse MF2: syntax error: complex message: matcher: variant keys: missing space between keys * and 1",
		},
		{
			in:      ".input {$foo} .input {$foo} {{ }}",
			wantErr: `parse MF2: complex message: input declaration: expression: duplicate declaration: $foo`,
		},
		{
			in:      ".input {$foo} .match $foo * {{}}",
			wantErr: `parse MF2: complex message: matcher: missing selector annotation`,
		},
		{
			in:      "Hello, { :number style=decimal style=percent }!",
			wantErr: `parse MF2: simple message: pattern: expression: function: duplicate option name`,
		},
	} {
		t.Run(test.in, func(t *testing.T) {
			t.Parallel()

			if _, err := Parse(test.in); err == nil || err.Error() != test.wantErr {
				t.Errorf("want '%s', got '%s'", test.wantErr, err)
			}
		})
	}
}

// helpers

// requireEqualMF2String compares two strings, but ignores whitespace, tabs, and newlines.
func requireEqualMF2String(t *testing.T, want, got string) {
	t.Helper()

	r := strings.NewReplacer(
		"\n", "",
		"\t", "",
		" ", "",
	)

	if r.Replace(want) != r.Replace(got) {
		t.Errorf("want '%s', got '%s'", want, got)
	}
}

func BenchmarkParse(b *testing.B) {
	var tree AST

	for range b.N {
		tree, _ = Parse(`  .input {$foo :number} .local $bar = {$foo} .match $bar one {{\|one\|}} * {{\|other\|}}  `)
	}

	runtime.KeepAlive(tree)
}
