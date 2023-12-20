package mf2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSimpleMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		expected    Message
		input       string
		convertBack bool // if true, convert the expected message back to string and compare with input.
	}{
		{
			name:  "text only",
			input: "Hello, World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "text only with escaped chars",
			input: "Hello, \\{World!\\}",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, {World!}"),
				},
			},
			convertBack: true,
		},
		{
			name:  "variable expression in the middle",
			input: "Hello, { $variable } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, "),
					PlaceholderPattern{
						Expression: VariableExpression{
							Variable: Variable("variable"),
						},
					},
					TextPattern(" World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "variable expression at the start",
			input: "{ $variable } Hello, World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					PlaceholderPattern{
						Expression: VariableExpression{
							Variable: Variable("variable"),
						},
					},
					TextPattern(" Hello, World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "variable expression at the end",
			input: "Hello, World! { $variable }",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, World! "),
					PlaceholderPattern{
						Expression: VariableExpression{
							Variable: Variable("variable"),
						},
					},
				},
			},
			convertBack: true,
		},
		{
			name:  "variable expression with annotation",
			input: "Hello, { $variable :function }  World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, "),
					PlaceholderPattern{
						Expression: VariableExpression{
							Variable: Variable("variable"),
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix: ':',
									Identifier: Identifier{
										Namespace: "",
										Name:      "function",
									},
								},
							},
						},
					},
					TextPattern("  World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "variable expression with annotation and options",
			input: "Hello, { $variable :function option1 = -3.14 ns:option2 = |value2| option3 = $variable2 } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, "),
					PlaceholderPattern{
						Expression: VariableExpression{
							Variable: Variable("variable"),
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix: ':',
									Identifier: Identifier{
										Namespace: "",
										Name:      "function",
									},
								},
								Options: []Option{
									LiteralOption{
										Literal: UnquotedLiteral{Value: NumberLiteral(-3.14)},
										Identifier: Identifier{
											Namespace: "",
											Name:      "option1",
										},
									},
									LiteralOption{
										Literal: QuotedLiteral("value2"),
										Identifier: Identifier{
											Namespace: "ns",
											Name:      "option2",
										},
									},
									VariableOption{
										Variable: Variable("variable2"),
										Identifier: Identifier{
											Namespace: "",
											Name:      "option3",
										},
									},
								},
							},
						},
					},
					TextPattern(" World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "quoted literal expression",
			input: "Hello, { |literal| }  World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, "),
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: QuotedLiteral("literal"),
						},
					},
					TextPattern("  World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "unquoted number literal expression",
			input: "Hello, { 1e3 }  World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, "),
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: UnquotedLiteral{Value: NumberLiteral(1e3)},
						},
					},
					TextPattern("  World!"),
				},
			},
			convertBack: false, // will fail as 1e3 will be converted to 1000
		},
		{
			name:  "unquoted name literal expression",
			input: "Hello, { name } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, "),
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: UnquotedLiteral{Value: NameLiteral("name")},
						},
					},
					TextPattern(" World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "quoted name literal expression with annotation",
			input: "Hello, { |name| :function } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, "),
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: QuotedLiteral("name"),
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix: ':',
									Identifier: Identifier{
										Namespace: "",
										Name:      "function",
									},
								},
							},
						},
					},
					TextPattern(" World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "quoted name literal expression with annotation and options",
			input: "Hello, { |name| :function ns1:option1 = -1 ns2:option2 = 1 option3 = |value3| } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello, "),
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: QuotedLiteral("name"),
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix: ':',
									Identifier: Identifier{
										Namespace: "",
										Name:      "function",
									},
								},
								Options: []Option{
									LiteralOption{
										Literal: UnquotedLiteral{Value: NumberLiteral(-1)},
										Identifier: Identifier{
											Namespace: "ns1",
											Name:      "option1",
										},
									},
									LiteralOption{
										Literal: UnquotedLiteral{Value: NumberLiteral(+1)},
										Identifier: Identifier{
											Namespace: "ns2",
											Name:      "option2",
										},
									},
									LiteralOption{
										Literal: QuotedLiteral("value3"),
										Identifier: Identifier{
											Namespace: "",
											Name:      "option3",
										},
									},
								},
							},
						},
					},
					TextPattern(" World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "annotation expression",
			input: "Hello { :function } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello "),
					PlaceholderPattern{
						Expression: AnnotationExpression{
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix: ':',
									Identifier: Identifier{
										Namespace: "",
										Name:      "function",
									},
								},
							},
						},
					},
					TextPattern(" World!"),
				},
			},
			convertBack: true,
		},
		{
			name:  "annotation expression with options and namespace",
			input: "Hello { :namespace:function namespace:option999 = 999 } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern("Hello "),
					PlaceholderPattern{
						Expression: AnnotationExpression{
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix: ':',
									Identifier: Identifier{
										Namespace: "namespace",
										Name:      "function",
									},
								},
								Options: []Option{
									LiteralOption{
										Literal: UnquotedLiteral{Value: NumberLiteral(999)},
										Identifier: Identifier{
											Namespace: "namespace",
											Name:      "option999",
										},
									},
								},
							},
						},
					},
					TextPattern(" World!"),
				},
			},
			convertBack: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := Parse(tt.input)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual.Message)

			if tt.convertBack {
				require.Equal(t, tt.input, actual.String())
			}
		})
	}
}

func TestParseComplexMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		expected    Message
		input       string
		convertBack bool // if true, convert the expected message back to string and compare with input.
	}{
		{
			name:  "no declarations",
			input: "{{Hello, { |literal| } World!}}",
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: QuotedPattern{
					Patterns: []Pattern{
						TextPattern("Hello, "),
						PlaceholderPattern{
							Expression: LiteralExpression{
								Literal: QuotedLiteral("literal"),
							},
						},
						TextPattern(" World!"),
					},
				},
			},
			convertBack: true,
		},
		{
			name:  "local declaration simple text",
			input: ".local $var={2} {{Hello world}}",
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: LiteralExpression{
							Literal: UnquotedLiteral{Value: NumberLiteral(2)},
						},
					},
				},
				ComplexBody: QuotedPattern{
					Patterns: []Pattern{
						TextPattern("Hello world"),
					},
				},
			},
			convertBack: false, // will fail, as input is one line, but output will be multiline
		},
		{
			name:  "local declaration and expressions",
			input: ".local $var = { $anotherVar } {{Hello { $var } world}}",
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: VariableExpression{
							Variable:   Variable("anotherVar"),
							Annotation: nil,
						},
					},
				},
				ComplexBody: QuotedPattern{
					Patterns: []Pattern{
						TextPattern("Hello "),
						PlaceholderPattern{
							Expression: VariableExpression{
								Variable:   Variable("var"),
								Annotation: nil,
							},
						},
						TextPattern(" world"),
					},
				},
			},
			convertBack: false, // will fail, as input is one line, but output will be multiline
		},
		{
			name:  "multiple local declaration one line",
			input: ".local $var = { :ns1:function opt1 = 1 opt2 = |val2| } .local $var = { 2 } {{Hello { $var :ns2:function2 } world}}", //nolint:lll
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: AnnotationExpression{
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix: ':',
									Identifier: Identifier{
										Namespace: "ns1",
										Name:      "function",
									},
								},
								Options: []Option{
									LiteralOption{
										Identifier: Identifier{Namespace: "", Name: "opt1"},
										Literal:    UnquotedLiteral{Value: NumberLiteral(1)},
									},
									LiteralOption{
										Identifier: Identifier{Namespace: "", Name: "opt2"},
										Literal:    QuotedLiteral("val2"),
									},
								},
							},
						},
					},
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: LiteralExpression{
							Literal:    UnquotedLiteral{Value: NumberLiteral(2)},
							Annotation: nil,
						},
					},
				},
				ComplexBody: QuotedPattern{
					Patterns: []Pattern{
						TextPattern("Hello "),
						PlaceholderPattern{
							Expression: VariableExpression{
								Variable: Variable("var"),
								Annotation: FunctionAnnotation{
									Function: Function{
										Prefix: ':',
										Identifier: Identifier{
											Namespace: "ns2",
											Name:      "function2",
										},
									},
								},
							},
						},
						TextPattern(" world"),
					},
				},
			},
			convertBack: false, // will fail, as input is one line, but output will be multiline
		},
		// Matcher
		{
			name:  "simple matcher one line",
			input: ".match { $variable :number } 1 {{Hello { $variable} world}} * {{Hello { $variable } worlds}}",
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: Matcher{
					MatchStatements: []Expression{
						VariableExpression{
							Variable: Variable("variable"),
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix:     ':',
									Identifier: Identifier{Namespace: "", Name: "number"},
								},
							},
						},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral(1)}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello "),
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern(" world"),
								},
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello "),
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern(" worlds"),
								},
							},
						},
					},
				},
			},
			convertBack: false, // will fail, as input is one line, but output will be multiline
		},
		{
			name: "simple matcher with newline variants",
			input: `.match { $variable :number }
1 {{Hello { $variable } world}}
* {{Hello { $variable } worlds}}`,
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: Matcher{
					MatchStatements: []Expression{
						VariableExpression{
							Variable: Variable("variable"),
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix:     ':',
									Identifier: Identifier{Namespace: "", Name: "number"},
								},
							},
						},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral(1)}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello "),
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern(" world"),
								},
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello "),
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern(" worlds"),
								},
							},
						},
					},
				},
			},
			convertBack: true,
		},
		{
			name: "simple matcher with newline variants in one line",
			input: `.match { $variable :number }

1 {{Hello { $variable} world}}* {{Hello { $variable } worlds}}`,
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: Matcher{
					MatchStatements: []Expression{
						VariableExpression{
							Variable: Variable("variable"),
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix:     ':',
									Identifier: Identifier{Namespace: "", Name: "number"},
								},
							},
						},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral(1)}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello "),
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern(" world"),
								},
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello "),
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern(" worlds"),
								},
							},
						},
					},
				},
			},
			convertBack: false, // will fail, as input chaotically formatted
		},
		{
			name: "matcher with declarations",
			input: `.local $var1 = { male }
.local $var2 = { |female| }
.match { :gender }
male {{Hello sir!}}
|female| {{Hello madam!}}
* {{Hello { $var1 } or { $var2 }!}}`,
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable:   Variable("var1"),
						Expression: LiteralExpression{Literal: UnquotedLiteral{Value: NameLiteral("male")}},
					},
					LocalDeclaration{
						Variable:   Variable("var2"),
						Expression: LiteralExpression{Literal: QuotedLiteral("female")},
					},
				},
				ComplexBody: Matcher{
					MatchStatements: []Expression{
						AnnotationExpression{
							Annotation: FunctionAnnotation{
								Function: Function{
									Prefix: ':', Identifier: Identifier{
										Namespace: "",
										Name:      "gender",
									},
								},
							},
						},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("male")}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello sir!"),
								},
							},
						},
						{
							Keys: []VariantKey{LiteralKey{Literal: QuotedLiteral("female")}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello madam!"),
								},
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern("Hello "),
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("var1")}},
									TextPattern(" or "),
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("var2")}},
									TextPattern("!"),
								},
							},
						},
					},
				},
			},
			convertBack: true,
		},
		{
			name: "double matcher",
			//nolint:dupword
			input: `.match {$var1} {$var2}
yes yes {{Hello beautiful world!}}
yes no {{Hello beautiful!}}
no yes {{Hello world!}}
no no {{Hello!}}`,
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: Matcher{
					MatchStatements: []Expression{
						VariableExpression{Variable: Variable("var1")},
						VariableExpression{Variable: Variable("var2")},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{
								LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("yes")}},
								LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("yes")}},
							},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{TextPattern("Hello beautiful world!")},
							},
						},
						{
							Keys: []VariantKey{
								LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("yes")}},
								LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("no")}},
							},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{TextPattern("Hello beautiful!")},
							},
						},
						{
							Keys: []VariantKey{
								LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("no")}},
								LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("yes")}},
							},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{TextPattern("Hello world!")},
							},
						},
						{
							Keys: []VariantKey{
								LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("no")}},
								LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral("no")}},
							},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{TextPattern("Hello!")},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := Parse(tt.input)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual.Message)

			if tt.convertBack {
				require.Equal(t, tt.input, actual.String())
			}
		})
	}
}
