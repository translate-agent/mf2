package mf2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSimpleMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected Message
	}{
		{
			name:  "text only",
			input: "Hello, World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, World!"},
				},
			},
		},
		{
			name:  "text only with escaped chars",
			input: "Hello, \\{World!\\}",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, {World!}"},
				},
			},
		},
		{
			name:  "variable expression in the middle",
			input: "Hello, { $variable }  World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, "},
					PlaceholderPattern{
						Expression: VariableExpression{
							Variable: Variable("variable"),
						},
					},
					TextPattern{Text: "  World!"},
				},
			},
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
					TextPattern{Text: " Hello, World!"},
				},
			},
		},
		{
			name:  "variable expression at the end",
			input: "Hello, World! { $variable }",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, World! "},
					PlaceholderPattern{
						Expression: VariableExpression{
							Variable: Variable("variable"),
						},
					},
				},
			},
		},
		{
			name:  "variable expression with annotation",
			input: "Hello, { $variable :function }  World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, "},
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
					TextPattern{Text: "  World!"},
				},
			},
		},
		{
			name:  "variable expression with annotation and options",
			input: "Hello, { $variable :function option1 = -3.14 ns:option2=|value2| option3=$variable2 } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, "},
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
										Literal: UnquotedLiteral{Value: NumberLiteral{Number: -3.14}},
										Identifier: Identifier{
											Namespace: "",
											Name:      "option1",
										},
									},
									LiteralOption{
										Literal: QuotedLiteral{Value: "value2"},
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
					TextPattern{Text: " World!"},
				},
			},
		},
		{
			name:  "quoted literal expression",
			input: "Hello, { |literal| }  World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, "},
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: QuotedLiteral{Value: "literal"},
						},
					},
					TextPattern{Text: "  World!"},
				},
			},
		},
		{
			name:  "unquoted number literal expression",
			input: "Hello, { 1e3 }  World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, "},
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: UnquotedLiteral{Value: NumberLiteral{Number: 1e3}},
						},
					},
					TextPattern{Text: "  World!"},
				},
			},
		},
		{
			name:  "unquoted name literal expression", // parse converts to quoted, but that's fine
			input: "Hello, { name } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, "},
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: QuotedLiteral{Value: "name"},
						},
					},
					TextPattern{Text: " World!"},
				},
			},
		},
		{
			name:  "quoted name literal expression with annotation",
			input: "Hello, { |name| :function } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, "},
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: QuotedLiteral{Value: "name"},
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
					TextPattern{Text: " World!"},
				},
			},
		},
		{
			name:  "quoted name literal expression with annotation and options",
			input: "Hello, { |name| :function ns1:option1 = -1 ns2:option2=1 option3=|value3| } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello, "},
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: QuotedLiteral{Value: "name"},
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
										Literal: UnquotedLiteral{Value: NumberLiteral{Number: -1}},
										Identifier: Identifier{
											Namespace: "ns1",
											Name:      "option1",
										},
									},
									LiteralOption{
										Literal: UnquotedLiteral{Value: NumberLiteral{Number: +1}},
										Identifier: Identifier{
											Namespace: "ns2",
											Name:      "option2",
										},
									},
									LiteralOption{
										Literal: QuotedLiteral{Value: "value3"},
										Identifier: Identifier{
											Namespace: "",
											Name:      "option3",
										},
									},
								},
							},
						},
					},
					TextPattern{Text: " World!"},
				},
			},
		},
		{
			name:  "annotation expression",
			input: "Hello { :function } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello "},
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
					TextPattern{Text: " World!"},
				},
			},
		},
		{
			name:  "annotation expression with options and namespace",
			input: "Hello { :namespace:function namespace:option999=999 } World!",
			expected: SimpleMessage{
				Patterns: []Pattern{
					TextPattern{Text: "Hello "},
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
										Literal: UnquotedLiteral{Value: NumberLiteral{Number: 999}},
										Identifier: Identifier{
											Namespace: "namespace",
											Name:      "option999",
										},
									},
								},
							},
						},
					},
					TextPattern{Text: " World!"},
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

			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestParseComplexMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected Message
	}{
		// TODO: valid syntax: lexer stuck on infinite loop
		// {
		// 	name:  "no declarations",
		// 	input: "{{Hello, { |literal| } World!}}",
		// 	expected: ComplexMessage{
		// 		Declarations: nil,
		// 		ComplexBody: QuotedPattern{
		// 			Patterns: []Pattern{
		// 				TextPattern{Text: "Hello, "},
		// 				PlaceholderPattern{
		// 					Expression: LiteralExpression{
		// 						Literal: QuotedLiteral{Value: "literal"},
		// 					},
		// 				},
		// 				TextPattern{Text: "World! "},
		// 			},
		// 		},
		// 	},
		// },
		{
			name:  "local declaration simple text",
			input: ".local $var={2} {{Hello world}}",
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: LiteralExpression{
							Literal: UnquotedLiteral{Value: NumberLiteral{Number: 2}},
						},
					},
				},
				ComplexBody: QuotedPattern{
					Patterns: []Pattern{
						TextPattern{Text: "Hello world"},
					},
				},
			},
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
						TextPattern{Text: "Hello "},
						PlaceholderPattern{
							Expression: VariableExpression{
								Variable:   Variable("var"),
								Annotation: nil,
							},
						},
						TextPattern{Text: " world"},
					},
				},
			},
		},
		{
			name: "multiple local declaration",
			input: "" +
				".local $var = { :ns1:function opt1=1 opt2=|val2| }" +
				".local $var={2}" +
				"{{Hello { $var :ns2:function2 } world}}",
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
										Literal:    UnquotedLiteral{Value: NumberLiteral{Number: 1}},
									},
									LiteralOption{
										Identifier: Identifier{Namespace: "", Name: "opt2"},
										Literal:    QuotedLiteral{Value: "val2"},
									},
								},
							},
						},
					},
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: LiteralExpression{
							Literal:    UnquotedLiteral{Value: NumberLiteral{Number: 2}},
							Annotation: nil,
						},
					},
				},
				ComplexBody: QuotedPattern{
					Patterns: []Pattern{
						TextPattern{Text: "Hello "},
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
						TextPattern{Text: " world"},
					},
				},
			},
		},
		// Matcher
		{
			name:  "simple matcher one line",
			input: ".match { $variable :number } 1 {{Hello { $variable} world}} * {{Hello { $variable } worlds}}",
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: Matcher{
					MatchStatement: MatchStatement{
						Selectors: []Selector{
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
					},
					Variants: []Variant{
						{
							Key: LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral{Number: 1}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello "},
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern{Text: " world"},
								},
							},
						},
						{
							Key: WildcardKey{Wildcard: '*'},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello "},
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern{Text: " worlds"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "simple matcher with newline variants",
			input: `.match { $variable :number }
1 {{Hello { $variable} world}}
* {{Hello { $variable } worlds}}`,
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: Matcher{
					MatchStatement: MatchStatement{
						Selectors: []Selector{
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
					},
					Variants: []Variant{
						{
							Key: LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral{Number: 1}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello "},
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern{Text: " world"},
								},
							},
						},
						{
							Key: WildcardKey{Wildcard: '*'},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello "},
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern{Text: " worlds"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "simple matcher with newline variants in one line",
			input: `.match { $variable :number }

1 {{Hello { $variable} world}}* {{Hello { $variable } worlds}}`,
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: Matcher{
					MatchStatement: MatchStatement{
						Selectors: []Selector{
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
					},
					Variants: []Variant{
						{
							Key: LiteralKey{Literal: UnquotedLiteral{Value: NumberLiteral{Number: 1}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello "},
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern{Text: " world"},
								},
							},
						},
						{
							Key: WildcardKey{Wildcard: '*'},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello "},
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
									TextPattern{Text: " worlds"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "matcher with declarations",
			input: "" +
				".local $var1 = { male }" +
				".local $var2 = { |female| }" +
				".match { :gender }" +
				"male {{Hello sir!}}" +
				"female {{Hello madam!}}" +
				"* {{Hello { $var1 } or { $var2 }!}}",
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable:   Variable("var1"),
						Expression: LiteralExpression{Literal: QuotedLiteral{Value: "male"}},
					},
					LocalDeclaration{
						Variable:   Variable("var2"),
						Expression: LiteralExpression{Literal: QuotedLiteral{Value: "female"}},
					},
				},
				ComplexBody: Matcher{
					MatchStatement: MatchStatement{
						Selectors: []Selector{
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
					},
					Variants: []Variant{
						{
							Key: LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral{Name: "male"}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello sir!"},
								},
							},
						},
						{
							Key: LiteralKey{Literal: UnquotedLiteral{Value: NameLiteral{Name: "female"}}},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello madam!"},
								},
							},
						},
						{
							Key: WildcardKey{Wildcard: '*'},
							QuotedPattern: QuotedPattern{
								Patterns: []Pattern{
									TextPattern{Text: "Hello "},
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("var1")}},
									TextPattern{Text: " or "},
									PlaceholderPattern{Expression: VariableExpression{Variable: Variable("var2")}},
									TextPattern{Text: "!"},
								},
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

			require.Equal(t, tt.expected, actual)
		})
	}
}
