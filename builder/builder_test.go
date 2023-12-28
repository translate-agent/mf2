package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Builder(t *testing.T) {
	t.Parallel()

	for _, test := range []struct { //nolint:govet
		name, expected string
		b              *Builder
	}{
		{
			"text",
			"Hello, World!",
			New().Text("Hello, World!"),
		},
		{
			"text, first character is dot",
			"\\.text",
			New().Text(".text"),
		},
		{
			"text, with special chars",
			"Hello\\\\, \\{World\\}!",
			New().Text("Hello\\, {World}!"),
		},
		{
			"text with literal",
			"Hello, { |World| }!",
			New().
				Text("Hello, ").
				Expr(Expr().Literal("World")).
				Text("!"),
		},
		{
			"text with expr",
			"Hello, { $world :upper limit = 2 }!",
			New().
				Text("Hello, ").
				Expr(Expr().Var("$world").Func(":upper", Option("limit", "2"))).
				Text("!"),
		},
		{
			"local",
			".local $hostName = { $host }\n",
			New().Local("$hostName", Expr().Var("$host")),
		},
		{
			"input",
			".input { $host }\n",
			New().Input(Expr().Var("$host")),
		},
		{
			"input and local",
			".input { $host }\n.local $hostName = { $host }\n",
			New().
				Local("$hostName", Expr().Var("$host")).
				Input(Expr().Var("$host")),
		},
		{
			"match",
			".match {$i} {$j}\n1 2 {{\\{first\\}}}\n2 0 {{second { $i }}}\n3 0 {{{ |\\\\a\\|| }}}\n* * {{{ |1| }}}\n",
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

			assert.Equal(t, test.expected, test.b.String())
		})
	}
}
