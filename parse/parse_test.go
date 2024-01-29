package parse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSimpleMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected Message
		input    string
	}{
		{
			name:  "text only",
			input: "Hello, World!",
			expected: SimpleMessage{
				TextPattern("Hello, World!"),
			},
		},
		{
			name:  "text only with escaped chars",
			input: "Hello, \\{World!\\}",
			expected: SimpleMessage{
				TextPattern("Hello, {World!}"),
			},
		},
		{
			name:  "variable expression in the middle",
			input: "Hello, { $variable } World!",
			expected: SimpleMessage{
				TextPattern("Hello, "),
				Expression{Operand: Variable("variable"), Typ: VariableExpression},
				TextPattern(" World!"),
			},
		},
		{
			name:  "variable expression at the start",
			input: "{ $variable } Hello, World!",
			expected: SimpleMessage{
				Expression{Operand: Variable("variable"), Typ: VariableExpression},
				TextPattern(" Hello, World!"),
			},
		},
		{
			name:  "variable expression at the end",
			input: "Hello, World! { $variable }",
			expected: SimpleMessage{
				TextPattern("Hello, World! "),
				Expression{
					Operand: Variable("variable"),
					Typ:     VariableExpression,
				},
			},
		},
		{
			name:  "variable expression with annotation",
			input: "Hello, { $variable :function }  World!",
			expected: SimpleMessage{
				TextPattern("Hello, "),
				Expression{
					Operand: Variable("variable"),
					Typ:     VariableExpression,
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
					},
				},
				TextPattern("  World!"),
			},
		},
		{
			name:  "variable expression with annotation options and attributes",
			input: "Hello, { $variable :function option1 = -3.14 ns:option2 = |value2| option3 = $variable2 @attr1 = attr1} World!", //nolint:lll
			expected: SimpleMessage{
				TextPattern("Hello, "),
				Expression{
					Operand: Variable("variable"), Typ: VariableExpression,
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
						Options: []Option{
							{
								Value: NumberLiteral(-3.14),
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
							Value:      UnquotedLiteral{Value: NameLiteral("attr1")},
							Identifier: Identifier{Name: "attr1"},
						},
					},
				},
				TextPattern(" World!"),
			},
		},
		{
			name:  "quoted literal expression",
			input: "Hello, { |literal| }  World!",
			expected: SimpleMessage{
				TextPattern("Hello, "),
				Expression{Operand: QuotedLiteral("literal"), Typ: LiteralExpression},
				TextPattern("  World!"),
			},
		},
		{
			name:  "unquoted scientific notation number literal expression",
			input: "Hello, { 1e3 }  World!",
			expected: SimpleMessage{
				TextPattern("Hello, "),
				Expression{Operand: NumberLiteral(1e3), Typ: LiteralExpression},
				TextPattern("  World!"),
			},
		},
		{
			name:  "unquoted name literal expression",
			input: "Hello, { name } World!",
			expected: SimpleMessage{
				TextPattern("Hello, "),
				Expression{Operand: NameLiteral("name"), Typ: LiteralExpression},
				TextPattern(" World!"),
			},
		},
		{
			name:  "quoted name literal expression with annotation",
			input: "Hello, { |name| :function } World!",
			expected: SimpleMessage{
				TextPattern("Hello, "),
				Expression{
					Operand: QuotedLiteral("name"), Typ: LiteralExpression,
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
					},
				},
				TextPattern(" World!"),
			},
		},
		{
			name:  "quoted name literal expression with annotation and options",
			input: "Hello, { |name| :function ns1:option1 = -1 ns2:option2 = 1 option3 = |value3| } World!",
			expected: SimpleMessage{
				TextPattern("Hello, "),
				Expression{
					Operand: QuotedLiteral("name"), Typ: LiteralExpression,
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
						Options: []Option{
							{
								Value: NumberLiteral(-1),
								Identifier: Identifier{
									Namespace: "ns1",
									Name:      "option1",
								},
							},
							{
								Value: NumberLiteral(+1),
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
				TextPattern(" World!"),
			},
		},
		{
			name:  "annotation expression",
			input: "Hello { :function } World!",
			expected: SimpleMessage{
				TextPattern("Hello "),
				Expression{
					Typ: AnnotationExpression,
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "",
							Name:      "function",
						},
					},
				},
				TextPattern(" World!"),
			},
		},
		{
			name:  "annotation expression with options and namespace",
			input: "Hello { :namespace:function namespace:option999 = 999 } World!",
			expected: SimpleMessage{
				TextPattern("Hello "),
				Expression{
					Typ: AnnotationExpression,
					Annotation: Function{
						Identifier: Identifier{
							Namespace: "namespace",
							Name:      "function",
						},
						Options: []Option{
							{
								Value: NumberLiteral(999),
								Identifier: Identifier{
									Namespace: "namespace",
									Name:      "option999",
								},
							},
						},
					},
				},
				TextPattern(" World!"),
			},
		},
		{
			name:  "markup",
			input: `It is a {#button opt1=val1 @attr1=val1 } button { /button } this is a { #br /} something else, {#ns:tag1}{#tag2}text{ #img /}{/tag2}{/ns:tag1}`, //nolint:lll
			expected: SimpleMessage{
				// 1. Open-Close markup
				TextPattern("It is a "),
				Markup{
					Typ: OpenMarkup,
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
				TextPattern(" button "),
				Markup{Typ: CloseMarkup, Identifier: Identifier{Name: "button"}},
				// 2. Self-close markup
				TextPattern(" this is a "),
				Markup{Typ: SelfCloseMarkup, Identifier: Identifier{Name: "br"}},
				TextPattern(" something else, "),
				// 3. Nested markup
				Markup{Typ: OpenMarkup, Identifier: Identifier{Namespace: "ns", Name: "tag1"}},
				Markup{Typ: OpenMarkup, Identifier: Identifier{Name: "tag2"}},
				TextPattern("text"),
				Markup{Typ: SelfCloseMarkup, Identifier: Identifier{Name: "img"}},
				Markup{Typ: CloseMarkup, Identifier: Identifier{Name: "tag2"}},
				Markup{Typ: CloseMarkup, Identifier: Identifier{Namespace: "ns", Name: "tag1"}},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := Parse(tt.input)
			require.NoError(t, err)

			// Check that AST message is equal to expected one.
			require.Equal(t, tt.expected, actual.Message)

			// Check that AST message converted back to string is equal to input.

			// Edge case: scientific notation number is converted to normal notation, hence comparison is bound to fail.
			// I.E. input string has 1e3, output string has 1000.
			if tt.name == "unquoted scientific notation number literal expression" {
				return
			}

			// If strings already match, we're done.
			// Otherwise check both sanitized strings.
			if actualStr := actual.String(); actualStr != tt.input {
				requireEqualMF2String(t, tt.input, actualStr)
			}
		})
	}
}

func TestParseComplexMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected Message
		input    string
	}{
		{
			name:  "no declarations",
			input: "{{Hello, { |literal| } World!}}",
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: QuotedPattern{
					TextPattern("Hello, "),
					Expression{Operand: QuotedLiteral("literal"), Typ: LiteralExpression},
					TextPattern(" World!"),
				},
			},
		},
		{
			name:  "local declaration simple text",
			input: ".local $var={2} {{Hello world}}",
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: Expression{
							Operand: NumberLiteral(2),
							Typ:     LiteralExpression,
						},
					},
				},
				ComplexBody: QuotedPattern{
					TextPattern("Hello world"),
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
						Expression: Expression{
							Operand: Variable("anotherVar"), Typ: VariableExpression,
							Annotation: nil,
						},
					},
				},
				ComplexBody: QuotedPattern{
					TextPattern("Hello "),
					Expression{Operand: Variable("var"), Typ: VariableExpression},
					TextPattern(" world"),
				},
			},
		},
		{
			name:  "multiple local declaration one line",
			input: ".local $var = { :ns1:function opt1 = 1 opt2 = |val2| } .local $var = { 2 } {{Hello { $var :ns2:function2 } world}}", //nolint:lll
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: Expression{
							Typ: AnnotationExpression,
							Annotation: Function{
								Identifier: Identifier{
									Namespace: "ns1",
									Name:      "function",
								},
								Options: []Option{
									{
										Identifier: Identifier{Namespace: "", Name: "opt1"},
										Value:      NumberLiteral(1),
									},
									{
										Identifier: Identifier{Namespace: "", Name: "opt2"},
										Value:      QuotedLiteral("val2"),
									},
								},
							},
						},
					},
					LocalDeclaration{
						Variable: Variable("var"),
						Expression: Expression{
							Operand:    NumberLiteral(2),
							Typ:        LiteralExpression,
							Annotation: nil,
						},
					},
				},
				ComplexBody: QuotedPattern{
					TextPattern("Hello "),
					Expression{
						Operand: Variable("var"), Typ: VariableExpression,
						Annotation: Function{
							Identifier: Identifier{
								Namespace: "ns2",
								Name:      "function2",
							},
						},
					},
					TextPattern(" world"),
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
					MatchStatements: []Expression{
						{
							Operand: Variable("variable"), Typ: VariableExpression,
							Annotation: Function{
								Identifier: Identifier{Namespace: "", Name: "number"},
							},
						},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{NumberLiteral(1)},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello "),
								Expression{Operand: Variable("variable"), Typ: VariableExpression},
								TextPattern(" world"),
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello "),
								Expression{Operand: Variable("variable"), Typ: VariableExpression},
								TextPattern(" worlds"),
							},
						},
					},
				},
			},
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
						{
							Operand: Variable("variable"), Typ: VariableExpression,
							Annotation: Function{
								Identifier: Identifier{Namespace: "", Name: "number"},
							},
						},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{NumberLiteral(1)},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello "),
								Expression{Operand: Variable("variable"), Typ: VariableExpression},
								TextPattern(" world"),
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello "),
								Expression{Operand: Variable("variable"), Typ: VariableExpression},
								TextPattern(" worlds"),
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
					MatchStatements: []Expression{
						{
							Operand: Variable("variable"), Typ: VariableExpression,
							Annotation: Function{
								Identifier: Identifier{Namespace: "", Name: "number"},
							},
						},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{NumberLiteral(1)},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello "),
								Expression{Operand: Variable("variable"), Typ: VariableExpression},
								TextPattern(" world"),
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello "),
								Expression{Operand: Variable("variable"), Typ: VariableExpression},
								TextPattern(" worlds"),
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
.match { :gender }
male {{Hello sir!}}
|female| {{Hello madam!}}
* {{Hello { $var1 } or { $var2 }!}}`,
			expected: ComplexMessage{
				Declarations: []Declaration{
					LocalDeclaration{
						Variable:   Variable("var1"),
						Expression: Expression{Operand: NameLiteral("male"), Typ: LiteralExpression},
					},
					LocalDeclaration{
						Variable:   Variable("var2"),
						Expression: Expression{Operand: QuotedLiteral("female"), Typ: LiteralExpression},
					},
				},
				ComplexBody: Matcher{
					MatchStatements: []Expression{
						{
							Typ: AnnotationExpression,
							Annotation: Function{
								Identifier: Identifier{
									Namespace: "",
									Name:      "gender",
								},
							},
						},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{NameLiteral("male")},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello sir!"),
							},
						},
						{
							Keys: []VariantKey{QuotedLiteral("female")},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello madam!"),
							},
						},
						{
							Keys: []VariantKey{CatchAllKey{}},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello "),
								Expression{Operand: Variable("var1"), Typ: VariableExpression},
								TextPattern(" or "),
								Expression{Operand: Variable("var2"), Typ: VariableExpression},
								TextPattern("!"),
							},
						},
					},
				},
			},
		},
		{
			name: "double matcher",
			//nolint:dupword
			input: `.match { $var1 } { $var2 }
yes yes {{Hello beautiful world!}}
yes no {{Hello beautiful!}}
no yes {{Hello world!}}
no no {{Hello!}}`,
			expected: ComplexMessage{
				Declarations: nil,
				ComplexBody: Matcher{
					MatchStatements: []Expression{
						{Operand: Variable("var1"), Typ: VariableExpression},
						{Operand: Variable("var2"), Typ: VariableExpression},
					},
					Variants: []Variant{
						{
							Keys: []VariantKey{
								NameLiteral("yes"),
								NameLiteral("yes"),
							},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello beautiful world!"),
							},
						},
						{
							Keys: []VariantKey{
								NameLiteral("yes"),
								NameLiteral("no"),
							},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello beautiful!"),
							},
						},
						{
							Keys: []VariantKey{
								NameLiteral("no"),
								NameLiteral("yes"),
							},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello world!"),
							},
						},
						{
							Keys: []VariantKey{
								NameLiteral("no"),
								NameLiteral("no"),
							},
							QuotedPattern: QuotedPattern{
								TextPattern("Hello!"),
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

			// Check that AST message is equal to expected one.
			require.Equal(t, tt.expected, actual.Message)

			// Check that AST message converted back to string is equal to input.

			// If strings already match, we're done.
			// Otherwise check both sanitized strings.
			if actualStr := actual.String(); actualStr != tt.input {
				requireEqualMF2String(t, tt.input, actualStr)
			}
		})
	}
}

// TestValidate tests negative cases for AST validation. Positive cases are covered by TestParse* tests.
func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ast       AST
		name      string
		errorPath string // path to the failing field (simplified to last n fields)
	}{
		{
			name:      "No message",
			ast:       AST{Message: nil},
			errorPath: "ast",
		},
		{
			// Hello, { $ } World!
			name: "Variable expression empty variable name",
			ast: AST{
				Message: SimpleMessage{
					TextPattern("Hello, "),
					Expression{Operand: Variable(""), Typ: VariableExpression},
				},
			},
			errorPath: "expression.variable",
		},
		{
			// Hello, { $variable : } World!
			name: "Variable expression with annotation empty function name",
			ast: AST{
				Message: SimpleMessage{
					TextPattern("Hello, "),
					Expression{
						Operand: Variable("variable"),
						Typ:     VariableExpression,
						Annotation: Function{
							Identifier: Identifier{
								Namespace: "",
								Name:      "",
							},
						},
					},
				},
			},
			errorPath: "function.identifier",
		},
		{
			// Hello, { } World!
			name: "Empty annotation expression",
			ast: AST{
				Message: SimpleMessage{
					TextPattern("Hello, "),
					Expression{Typ: AnnotationExpression},
					TextPattern(" World!"),
				},
			},
			errorPath: "simpleMessage.expression",
		},
		{
			// .input { $ } {{Hello, World!}}
			name: "Empty variable in input declaration",
			ast: AST{
				Message: ComplexMessage{
					Declarations: []Declaration{
						InputDeclaration{
							Expression: Expression{Operand: Variable(""), Typ: VariableExpression},
						},
					},
					ComplexBody: QuotedPattern{
						TextPattern("Hello, World!"),
					},
				},
			},
			errorPath: "inputDeclaration.expression",
		},
		{
			// .local $var = {  } {{Hello, World!}}
			name: "Empty expression in local declaration",
			ast: AST{
				Message: ComplexMessage{
					Declarations: []Declaration{
						LocalDeclaration{
							Variable:   Variable("var"),
							Expression: Expression{},
						},
					},
					ComplexBody: QuotedPattern{
						TextPattern("Hello, World!"),
					},
				},
			},
			errorPath: "complexMessage.localDeclaration",
		},
		{
			// .match { } 1 {{Hello, World!}}
			name: "Empty expression in matcher",
			ast: AST{
				Message: ComplexMessage{
					Declarations: nil,
					ComplexBody: Matcher{
						MatchStatements: nil,
						Variants: []Variant{
							{
								Keys: []VariantKey{NumberLiteral(1)},
								QuotedPattern: QuotedPattern{
									TextPattern("Hello, World!"),
								},
							},
						},
					},
				},
			},
			errorPath: "complexMessage.matcher",
		},
		{
			// .match { $variable }
			name: "Matcher without variants",
			ast: AST{
				Message: ComplexMessage{
					Declarations: nil,
					ComplexBody: Matcher{
						MatchStatements: []Expression{
							{Operand: Variable("variable"), Typ: VariableExpression},
						},
						Variants: nil,
					},
				},
			},
			errorPath: "complexMessage.matcher",
		},
		{
			// .match { $variable } {{Hello world}}
			name: "Matcher without variant key",
			ast: AST{
				Message: ComplexMessage{
					ComplexBody: Matcher{
						MatchStatements: []Expression{
							{Operand: Variable("variable"), Typ: VariableExpression},
						},
						Variants: []Variant{
							{
								Keys:          []VariantKey{},
								QuotedPattern: QuotedPattern{TextPattern("Hello world")},
							},
						},
					},
				},
			},
			errorPath: "matcher.variant",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.errorPath == "" {
				require.FailNow(t, "test.errorPath is not set")
			}

			require.ErrorContains(t, tt.ast.validate(), tt.errorPath)
		})
	}
}

// helpers

// requireEqualMF2String compares two strings, but ignores whitespace, tabs, and newlines.
func requireEqualMF2String(t *testing.T, expected, actual string) {
	t.Helper()

	r := strings.NewReplacer(
		"\n", "",
		"\t", "",
		" ", "",
	)

	require.Equal(t, r.Replace(expected), r.Replace(actual))
}
