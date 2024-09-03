package builder

import (
	"runtime"
	"testing"
)

func Test_Builder(t *testing.T) {
	t.Parallel()

	//nolint:lll
	for _, test := range []struct {
		name string
		b    *Builder
		want string
	}{
		{
			"simple message, empty text",
			NewBuilder().Text(""),
			"",
		},
		{
			"simple message, simple text",
			NewBuilder().Text("Hello, World!"),
			"Hello, World!",
		},
		{
			"simple message, text starts with whitespace char",
			NewBuilder().Text(" hello"),
			" hello",
		},
		{
			"simple message, text with special chars",
			NewBuilder().Text("{Hello}\\, {World}!"),
			"\\{Hello\\}\\\\, \\{World\\}!",
		},
		{
			"simple message, text with literal",
			NewBuilder().
				Text("Hello, ").
				Expr(Literal("World")).
				Text("!"),
			"Hello, { World }!",
		},
		{
			"simple message, text and expr with options",
			NewBuilder().
				Text("Hello, ").
				Expr(Var("world").
					Func("upper",
						LiteralOption("limit", 2),
						VarOption("min", "min"),
						LiteralOption("type", "integer"),
						LiteralOption("x", "y z"),
						LiteralOption("host", ""))).
				Text("!"),
			"Hello, { $world :upper limit = 2 min = $min type = integer x = |y z| host = || }!",
		},
		{
			"simple message, annotations",
			NewBuilder().
				Text("Hello, ").
				// Variable expression and private use annotation with quoted and text
				Expr(
					Var("f").
						Annotation(Caret,
							Quoted("a"),
							ReservedText("reserved")),
				).
				// Annotation expression, reserved use annotation without body
				Expr(Annotation(Exclamation)).
				// Annotation expression, reserved use annotation with escaped quoted and text
				Expr(
					Annotation(GreaterThan,
						Quoted("b|"),
						ReservedText("escaped |}"),
					),
				).
				// Literal expression with attribute and reserved use annotation with multiple texts
				Expr(
					Literal("c").
						Annotation(Ampersand,
							ReservedText("hey1 hey2"),
							ReservedText("hey3 hey4"),
							ReservedText("hey5"),
						).
						Attr(
							VarAttribute("attr1", "var"),
						),
				).
				Text("world!"),
			`Hello, { $f ^ |a| reserved }{ ! }{ > |b\|| escaped \|\} }{ c & hey1 hey2 hey3 hey4 hey5 @attr1 = $var }world!`,
		},
		{
			"complex message, period char",
			NewBuilder().Text("."),
			"{{.}}",
		},
		{
			"complex message, text starts with a period char",
			NewBuilder().Text(".ok"),
			"{{.ok}}",
		},
		{
			"complex message, text with special chars",
			NewBuilder().Local("var", Literal("greeting")),
			".local $var = { greeting }\n{{}}",
		},
		{
			"complex message, local declaration followed by expr",
			NewBuilder().Local("var", Var("greeting")).Expr(Var("var")),
			".local $var = { $greeting }\n{{{ $var }}}",
		},
		{
			"complex message all declarations",
			NewBuilder().
				Local("hostName", Var("host")).
				Local("hostName2", Annotation(Ampersand, Quoted("hey"))).
				Input(Var("input").Attr(EmptyAttribute("empty"))).
				Input(Var("input2").Func("upper")).
				Text("Beep"),
			`.local $hostName = { $host }
.local $hostName2 = { & |hey| }
.input { $input @empty }
.input { $input2 :upper }
{{Beep}}`,
		},
		{
			"complex message, matcher with multiple keys",
			NewBuilder().
				Match(
					Var("i"),
					Var("j"),
				).
				Keys(1, 2).Text("{first}").
				Keys(2, 0).Text("second ").Expr(Var("i")).
				Keys(3, 0).Expr(Literal("\\a|")).
				Keys("*", "*").Expr(Literal(1)),
			".match { $i } { $j }\n1 2 {{\\{first\\}}}\n2 0 {{second { $i }}}\n3 0 {{{ |\\\\a\\|| }}}\n* * {{{ 1 }}}",
		},
		{
			"complex message, matcher with multiple keys and local declarations",
			NewBuilder().
				Input(Var("i")).
				Local("hostName", Var("i")).
				Match(
					Var("i"),
					Var("j"),
				).
				Keys(1, 2).Text("{first}").
				Keys(2, 0).Text("second ").Expr(Var("i")).
				Keys(3, 0).Expr(Literal("\\a|")).
				Keys("*", "*").Expr(Literal(1)),
			".input { $i }\n.local $hostName = { $i }\n.match { $i } { $j }\n1 2 {{\\{first\\}}}\n2 0 {{second { $i }}}\n3 0 {{{ |\\\\a\\|| }}}\n* * {{{ 1 }}}",
		},
		{
			"attributes",
			NewBuilder().
				Text("Attributes for variable expression ").
				Expr(
					Var("i").
						Attr(
							VarAttribute("attr1", "var"),
							LiteralAttribute("attr2", "literal"),
							EmptyAttribute("empty"),
						),
				),
			"Attributes for variable expression { $i @attr1 = $var @attr2 = literal @empty }",
		},
		{
			"markup",
			NewBuilder().
				OpenMarkup(
					"open",
					LiteralOption("opt1", "val1"),
					LiteralAttribute("attr1", 1),
					VarOption("opt2", "var"),
				).
				Text(" something ").
				CloseMarkup(
					"close",
					EmptyAttribute("empty1"),
					VarAttribute("attr1", "var"),
				).
				SelfCloseMarkup(
					"selfClosing",
					LiteralAttribute("attr1", "༼ つ ◕_◕ ༽つ"),
				).
				// nested markup
				OpenMarkup("nest1").
				OpenMarkup("nest2").
				Text("nested").
				SelfCloseMarkup("nest3").
				CloseMarkup("nest2").
				CloseMarkup("nest1"),
			"{ #open opt1 = val1 opt2 = $var @attr1 = 1 } something { /close @empty1 @attr1 = $var }{ #selfClosing @attr1 = |༼ つ ◕_◕ ༽つ| /}{ #nest1 }{ #nest2 }nested{ #nest3 /}{ /nest2 }{ /nest1 }",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.b.Build()
			if err != nil {
				t.Error(err)
			}

			if test.want != got {
				t.Errorf("\nwant '%s'\ngot  '%s'", test.want, got)
			}
		})
	}
}

func BenchmarkBuildMatch(b *testing.B) {
	var s string

	for range b.N {
		s, _ = NewBuilder().
			Input(Var("i")).
			Local("hostName", Var("i")).
			Match(
				Var("i"),
				Var("j"),
			).
			Keys(1, 2).Text("{first}").
			Keys(2, 0).Text("second ").Expr(Var("i")).
			Keys(3, 0).Expr(Literal("\\a|")).
			Keys("*", "*").Expr(Literal(1)).
			Build()
	}

	runtime.KeepAlive(s)
}

func BenchmarkBuildMarkup(b *testing.B) {
	var s string

	for range b.N {
		s, _ = NewBuilder().
			OpenMarkup(
				"open",
				LiteralOption("opt1", "val1"),
				LiteralAttribute("attr1", 1),
				VarOption("opt2", "var"),
			).
			Text(" something ").
			CloseMarkup(
				"close",
				EmptyAttribute("empty1"),
				VarAttribute("attr1", "var"),
			).
			SelfCloseMarkup(
				"selfClosing",
				LiteralAttribute("attr1", "༼ つ ◕_◕ ༽つ"),
			).
			// nested markup
			OpenMarkup("nest1").
			OpenMarkup("nest2").
			Text("nested").
			SelfCloseMarkup("nest3").
			CloseMarkup("nest2").
			CloseMarkup("nest1").
			Build()
	}

	runtime.KeepAlive(s)
}
