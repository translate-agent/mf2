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
				Pattern: []Pattern{
					TextPattern{Text: "Hello, World!"},
				},
			},
		},
		{
			name:  "variable expression",
			input: "Hello, { $variable }  World!",
			expected: SimpleMessage{
				Pattern: []Pattern{
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
			name:  "variable expression with annotation",
			input: "Hello, { $variable :function }  World!",
			expected: SimpleMessage{
				Pattern: []Pattern{
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
				Pattern: []Pattern{
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
										Literal: UnquotedLiteral{Value: NumberLiteral[float64]{Number: -3.14}},
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
				Pattern: []Pattern{
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
				Pattern: []Pattern{
					TextPattern{Text: "Hello, "},
					PlaceholderPattern{
						Expression: LiteralExpression{
							Literal: UnquotedLiteral{Value: NumberLiteral[float64]{Number: 1e3}},
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
				Pattern: []Pattern{
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
				Pattern: []Pattern{
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
				Pattern: []Pattern{
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
										Literal: UnquotedLiteral{Value: NumberLiteral[int64]{Number: -1}},
										Identifier: Identifier{
											Namespace: "ns1",
											Name:      "option1",
										},
									},
									LiteralOption{
										Literal: UnquotedLiteral{Value: NumberLiteral[int64]{Number: +1}},
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
				Pattern: []Pattern{
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
				Pattern: []Pattern{
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
										Literal: UnquotedLiteral{Value: NumberLiteral[int64]{Number: 999}},
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
