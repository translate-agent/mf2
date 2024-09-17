package parse

import "testing"

func TestExpression_String(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		want string
		expr Expression
	}{
		{"{}", Expression{}},
		{"{ a }", Expression{Operand: NameLiteral("a")}},
		{"{ :f }", Expression{Annotation: Function{Identifier: Identifier{Name: "f"}}}},
	} {
		t.Run(test.want, func(t *testing.T) {
			t.Parallel()

			if s := test.expr.String(); s != test.want {
				t.Errorf("want '%s', got '%s'", test.want, s)
			}
		})
	}
}

func TestMarkup_String(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		want   string
		markup Markup
	}{
		{"{ #b }", Markup{Typ: Open, Identifier: Identifier{Name: "b"}}},
		{"{ /b }", Markup{Typ: Close, Identifier: Identifier{Name: "b"}}},
		{"{ #b /}", Markup{Typ: SelfClose, Identifier: Identifier{Name: "b"}}},
	} {
		t.Run(test.want, func(t *testing.T) {
			t.Parallel()

			if s := test.markup.String(); s != test.want {
				t.Errorf("want '%s', got '%s'", test.want, s)
			}
		})
	}
}

func BenchmarkComplexMessage_String(b *testing.B) {
	//nolint:dupword
	tree, err := Parse(".input {$foo :number} .input {$bar :number} .match $foo $bar one one {{one { $foo } one}} one * {{one other}} * * {{other}}") //nolint:lll
	if err != nil {
		b.Error(err)
	}

	var result string

	for range b.N {
		result = tree.String()
	}

	_ = result
}
