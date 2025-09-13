package builder

import (
	"runtime"
	"testing"

	"go.expect.digital/mf2/parse"
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
			"Hello, { $world :upper limit = |2| min = $min type = integer x = |y z| host = || }!",
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
				Local("hostName2", Literal("hey").Func("hey")).
				Input(Var("input").Attr(EmptyAttribute("empty"))).
				Input(Var("input2").Func("upper")).
				Text("Beep"),
			`.local $hostName = { $host }
.local $hostName2 = { hey :hey }
.input { $input @empty }
.input { $input2 :upper }
{{Beep}}`,
		},
		{
			"complex message, matcher with multiple keys",
			NewBuilder().
				Match(
					parse.Variable("i"),
					parse.Variable("j"),
				).
				Keys(1, 2).Text("{first}").
				Keys(2, 0).Text("second ").Expr(Var("i")).
				Keys(3, 0).Expr(Literal("\\a|")).
				Keys("*", "*").Expr(Literal(1)),
			".match $i $j\n|1| |2| {{\\{first\\}}}\n|2| |0| {{second { $i }}}\n|3| |0| {{{ |\\\\a\\|| }}}\n* * {{{ |1| }}}",
		},
		{
			"complex message, matcher with multiple keys and local declarations",
			NewBuilder().
				Input(Var("i")).
				Local("hostName", Var("i")).
				Match(
					parse.Variable("i"),
					parse.Variable("j"),
				).
				Keys(1, 2).Text("{first}").
				Keys(2, 0).Text("second ").Expr(Var("i")).
				Keys(3, 0).Expr(Literal("\\a|")).
				Keys("*", "*").Expr(Literal(1)),
			".input { $i }\n.local $hostName = { $i }\n.match $i $j\n|1| |2| {{\\{first\\}}}\n|2| |0| {{second { $i }}}\n|3| |0| {{{ |\\\\a\\|| }}}\n* * {{{ |1| }}}",
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
			"{ #open opt1 = val1 opt2 = $var @attr1 = |1| } something { /close @empty1 @attr1 = $var }{ #selfClosing @attr1 = |༼ つ ◕_◕ ༽つ| /}{ #nest1 }{ #nest2 }nested{ #nest3 /}{ /nest2 }{ /nest1 }",
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
				parse.Variable("i"),
				parse.Variable("j"),
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
