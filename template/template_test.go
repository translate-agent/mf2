package template

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.expect.digital/mf2/template/registry"
)

func Test_ExecuteSimpleMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, text string
		input      map[string]any
		expected   string
		funcs      []registry.Func // format functions to be added before executing
	}{
		{
			name: "empty message",
		},
		{
			name:     "plain message",
			text:     "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "variables and literals",
			text:     "Hello, { $name } { unquoted } { |quoted| } { 42 }!",
			input:    map[string]any{"name": "World"},
			expected: "Hello, World unquoted quoted 42!",
		},
		{
			name: "functions with operand",
			text: "Hello, { $firstName :string } your age is { $age :number style=decimal }!",
			input: map[string]any{
				"firstName": "John",
				"age":       23,
			},
			expected: "Hello, John your age is 23!",
		},
		{
			name: "function without operand",
			text: "Hello, { :randName }",
			funcs: []registry.Func{
				{
					Name:            "randName",
					FormatSignature: &registry.Signature{},
					Fn:              func(_ any, _ map[string]any) (any, error) { return "John", nil },
				},
			},
			expected: "Hello, John",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.text)
			require.NoError(t, err)

			for _, f := range tt.funcs {
				template.AddFunc(f)
			}

			actual, err := template.Sprint(tt.input)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual)
		})
	}
}

func Test_ExecuteComplexMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, text string
		inputs     map[string]any
		expected   string
		funcs      []registry.Func // format functions to be added before executing
	}{
		{
			name:     "complex message without declaration",
			text:     "{{Hello, {|literal|} World!}}",
			expected: "Hello, literal World!",
		},
		{
			name: "local declarations",
			text: `.local $var1 = { literalExpression }
		.local $var2 = { $anotherVar }
		.local $var3 = { :randNum }
		{{Hello, {$var1} {$var2} {$var3}!}}`,
			inputs: map[string]any{"anotherVar": "World"},
			funcs: []registry.Func{
				{
					Name:            "randNum",
					FormatSignature: &registry.Signature{},
					Fn:              func(_ any, _ map[string]any) (any, error) { return 0, nil },
				},
			},
			expected: "Hello, literalExpression World 0!",
		},
		{
			name:     "input declaration",
			text:     ".input { $name :string } {{Hello, {$name}!}}",
			inputs:   map[string]any{"name": 999},
			expected: "Hello, 999!",
		},
		{
			name:     "markup",
			text:     "Click {#link href=$url}here{/link} standalone {#foo/}",
			expected: "Click here standalone ",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.text)
			require.NoError(t, err)

			for _, f := range tt.funcs {
				template.AddFunc(f)
			}

			actual, err := template.Sprint(tt.inputs)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual)
		})
	}
}

func Test_Matcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		inputs   []map[string]any
		expected []string
		funcs    []registry.Func
	}{
		{
			name: "matcher string",
			text: `.match { $n :string } no {{no apples}} one {{{ $n } apple}} * {{{ $n } apples}}`,
			inputs: []map[string]any{
				{"n": "no"},
				{"n": "one"},
				{"n": "many"},
			},
			expected: []string{"no apples", "one apple", "many apples"},
		},
		{
			name: "Pattern Selection with string annotation",
			//nolint:dupword
			text: ".match {$foo :string} {$bar :string} bar bar {{All bar}} foo foo {{All foo}} * * {{Otherwise}}",
			inputs: []map[string]any{
				{"foo": "foo", "bar": "bar"},
			},
			expected: []string{"Otherwise"},
		},
		{
			name:     "Pattern Selection with Multiple Variants",
			text:     ".match {$foo :string} {$bar :string} * bar {{Any and bar}}foo * {{Foo and any}} foo bar {{Foo and bar}} * * {{Otherwise}}", //nolint:lll
			inputs:   []map[string]any{{"foo": "foo", "bar": "bar"}},
			expected: []string{"Foo and bar"},
		},
		{
			name:     "Plural Format Selection",
			text:     ".match {$count :string} one {{Category match}} 1 {{Exact match}} *   {{Other match}}",
			inputs:   []map[string]any{{"count": "1"}},
			expected: []string{"Exact match"},
		},
	}

	for _, tt := range tests {
		if len(tt.inputs) != len(tt.expected) {
			t.Error("Arguments and expected results should have the same length")
		}

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.text)
			require.NoError(t, err)

			for i, inputMap := range tt.inputs {
				i, inputMap := i, inputMap

				t.Run(tt.expected[i], func(t *testing.T) {
					t.Parallel()

					actual, err := template.Sprint(inputMap)

					require.NoError(t, err)
					require.Equal(t, tt.expected[i], actual)
				})
			}
		})
	}
}

func Test_ExecuteErrors(t *testing.T) {
	t.Parallel()

	type expected struct {
		parseErr, execErr error
	}

	tests := []struct {
		name, text string
		input      map[string]any
		expected   expected
		fn         []registry.Func // format function to be added before executing
	}{
		{
			name:     "syntax error",
			text:     "Hello { $name",
			expected: expected{parseErr: ErrSyntax},
		},
		{
			name:     "unresolved variable",
			text:     "Hello, { $name }!",
			expected: expected{execErr: ErrUnresolvedVariable},
		},
		{
			name:     "unknown function",
			text:     "Hello, { :f }!",
			expected: expected{execErr: ErrUnknownFunction},
		},
		{
			name:     "duplicate option name",
			text:     "Hello, { :number style=decimal style=percent }!",
			expected: expected{execErr: ErrDuplicateOptionName},
		},
		{
			name:     "unsupported expression",
			text:     "Hello, { 12 ^private }!",
			expected: expected{execErr: ErrUnsupportedExpression},
		},
		{
			name:     "formatting error",
			text:     "Hello, { :error }!",
			expected: expected{execErr: ErrFormatting},
			fn: []registry.Func{
				{
					Name:            "error",
					FormatSignature: &registry.Signature{},
					Fn:              func(any, map[string]any) (any, error) { return nil, errors.New("error") },
				},
			},
		},
		{
			name:     "unsupported declaration",
			text:     ".reserved { $name } {{Hello, {$name}!}}",
			expected: expected{execErr: ErrUnsupportedStatement},
		},
		{
			name:     "duplicate declaration",
			text:     ".input {$var} .input {$var} {{Redeclaration of the same variable}}",
			input:    map[string]any{"var": "22"},
			expected: expected{execErr: ErrDuplicateDeclaration},
		},
		{
			name:     "duplicate declaration",
			text:     ".local $var = {$ext} .input {$var} {{Redeclaration of a local variable}}",
			input:    map[string]any{"ext": "22"},
			expected: expected{execErr: ErrDuplicateDeclaration},
		},
		{
			name:     "Selection Error No Annotation",
			text:     ".match {$n} 0 {{no apples}} 1 {{apple}} * {{apples}}",
			input:    map[string]any{"n": "1"},
			expected: expected{execErr: ErrMissingSelectorAnnotation},
		},
		{
			name:     "Selection with Reversed Annotation",
			text:     ".match {$count ^string} one {{Category match}} 1 {{Exact match}} *   {{Other match}}",
			input:    map[string]any{"count": "1"},
			expected: expected{execErr: ErrUnsupportedExpression},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.text)
			if tt.expected.parseErr != nil {
				require.ErrorIs(t, err, tt.expected.parseErr)
				return
			}

			for _, f := range tt.fn {
				template.AddFunc(f)
			}

			_, err = template.Sprint(tt.input)
			require.ErrorIs(t, err, tt.expected.execErr)
		})
	}
}
