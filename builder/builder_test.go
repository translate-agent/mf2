package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Builder(t *testing.T) {
	t.Parallel()

	for _, test := range []struct { //nolint:govet
		name, expected string
		b              *Builder
	}{
		{
			"empty simple message",
			"",
			New().Text(""),
		},
		{
			"simple message, text only",
			"Hello, World!",
			New().Text("Hello, World!"),
		},
		{
			"simple message, text with special chars",
			"\\{Hello\\}\\\\, \\{World\\}!",
			New().Text("{Hello}\\, {World}!"),
		},
		{
			"simple message, text with literal",
			"Hello, { |World| }!",
			New().
				Text("Hello, ").
				Expr(Expr().Literal("World")).
				Text("!"),
		},
		{
			"simple message, text with expr",
			"Hello, { $world :upper limit = 2 }!",
			New().
				Text("Hello, ").
				Expr(Expr().Var("$world").Func(":upper", Option("limit", "2"))).
				Text("!"),
		},
		{
			"simple message, text with markup-like function",
			"Hello { +link } World { -link }",
			New().
				Text("Hello ").
				Expr(Expr().Func("+link")).
				Text(" World ").
				Expr(Expr().Func("-link")),
		},
		{
			"complex message, empty quoted pattern",
			"{{}}",
			New().Quoted(Pattern()),
		},
		{
			"complex message, text starts with period",
			"{{.ok}}",
			New().Text(".ok"),
		},
		{
			"complex message, local declaration",
			".local $hostName = { $host }\n{{{$hostName}}}",
			New().Local("$hostName", Expr().Var("$host")).Quoted(Pattern().Expr(Expr().Var("$hostName"))),
		},
		{
			"complex message, input declaration",
			".input { $host }\n{{{$host}}}",
			New().Input(Expr().Var("$host")).Quoted(Pattern().Expr(Expr().Var("$host"))),
		},
		{
			"complex message, input and local declaration",
			".input { $host }\n.local $hostName = { $host }\n{{{$host}}}",
			New().
				Local("$hostName", Expr().Var("$host")).
				Input(Expr().Var("$host")).
				Quoted(Pattern().Expr(Expr().Var("$host"))),
		},
		{
			"complex message, matcher with multiple keys",
			".match {$i} {$j}\n1 2 {{\\{first\\}}}\n2 0 {{second { $i }}}\n3 0 {{{ |\\\\a\\|| }}}\n* * {{{ 1 }}}\n",
			New().
				Match(
					Expr().Var("$i"),
					Expr().Var("$j"),
				).
				Key(1, 2).Text("{first}").
				Key(2, 0).Text("second").Expr(Expr().Var("$i")).
				Key(3, 0).Expr(Expr().Literal("\\a|")).
				Key("*", "*").Expr(Expr().Literal(1)),
		},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, test.b.MustBuild())
		})
	}
}
