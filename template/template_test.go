package template

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.expect.digital/mf2/template/registry"
)

func Test_ExecuteSimpleMessage(t *testing.T) {
	t.Parallel()

	type args struct {
		input map[string]any
		text  string
	}

	tests := []struct {
		name     string
		args     args
		expected string
		funcs    []registry.Func // format functions to be added before executing
	}{
		{
			name: "empty message",
			args: args{
				text:  "",
				input: nil,
			},
			expected: "",
		},
		{
			name: "plain message",
			args: args{
				text:  "Hello, World!",
				input: nil,
			},
			expected: "Hello, World!",
		},
		{
			name: "variables and literals",
			args: args{
				text:  "Hello, { $name } { unquoted } { |quoted| } { 42 }!",
				input: map[string]any{"name": "World"},
			},
			expected: "Hello, World unquoted quoted 42!",
		},
		{
			name: "functions with operand",
			args: args{
				text: "Hello, { $firstName :string } your age is { $age :number style=decimal }!",
				input: map[string]any{
					"firstName": "John",
					"age":       23,
				},
			},
			expected: "Hello, John your age is 23!",
		},
		{
			name: "function without operand",
			args: args{
				text:  "Hello, { :randName }",
				input: nil,
			},
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

			template, err := New().Parse(tt.args.text)
			require.NoError(t, err)

			for _, f := range tt.funcs {
				template.AddFunc(f)
			}

			actual, err := template.Sprint(tt.args.input)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual)
		})
	}
}

func Test_ExecuteComplexMessage(t *testing.T) {
	t.Parallel()

	type args struct {
		inputs map[string]any
		text   string
	}

	tests := []struct {
		name     string
		args     args
		expected string
		funcs    []registry.Func // format functions to be added before executing
	}{
		{
			name: "complex message without declaration",
			args: args{
				text:   "{{Hello, {|literal|} World!}}",
				inputs: nil,
			},
			expected: "Hello, literal World!",
		},
		{
			name: "local declarations",
			args: args{
				text: `.local $var1 = { literalExpression }
		.local $var2 = { $anotherVar }
		.local $var3 = { :randNum }
		{{Hello, {$var1} {$var2} {$var3}!}}`,
				inputs: map[string]any{"anotherVar": "World"},
			},
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
			name: "input declaration",
			args: args{
				text:   ".input { $name :string } {{Hello, {$name}!}}",
				inputs: map[string]any{"name": 999},
			},
			expected: "Hello, 999!",
		},
		{
			name: "markup",
			args: args{
				text:   "Click {#link href=$url}here{/link} standalone {#foo/}",
				inputs: nil,
			},
			expected: "Click here standalone ",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.args.text)
			require.NoError(t, err)

			for _, f := range tt.funcs {
				template.AddFunc(f)
			}

			actual, err := template.Sprint(tt.args.inputs)
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
				i := i
				inputMap := inputMap

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

	type args struct {
		input map[string]any
		text  string
	}

	tests := []struct {
		name               string
		expectedParseErr   error
		expectedExecuteErr error
		args               args
		fn                 []registry.Func // format function to be added before executing
	}{
		{
			name: "syntax error",
			args: args{
				text:  "Hello { $name",
				input: nil,
			},
			expectedParseErr: ErrSyntax,
		},
		{
			name: "unresolved variable",
			args: args{
				text:  "Hello, { $name }!",
				input: nil,
			},
			expectedExecuteErr: ErrUnresolvedVariable,
		},
		{
			name: "unknown function",
			args: args{
				text:  "Hello, { :f }!",
				input: nil,
			},
			expectedExecuteErr: ErrUnknownFunction,
		},
		{
			name: "duplicate option name",
			args: args{
				text:  "Hello, { :number style=decimal style=percent }!",
				input: nil,
			},
			expectedExecuteErr: ErrDuplicateOptionName,
		},
		{
			name: "unsupported expression",
			args: args{
				text:  "Hello, { 12 ^private }!",
				input: nil,
			},
			expectedExecuteErr: ErrUnsupportedExpression,
		},
		{
			name: "formatting error",
			args: args{
				text:  "Hello, { :error }!",
				input: nil,
			},
			expectedExecuteErr: ErrFormatting,
			fn: []registry.Func{
				{
					Name:            "error",
					FormatSignature: &registry.Signature{},
					Fn:              func(any, map[string]any) (any, error) { return nil, errors.New("error") },
				},
			},
		},
		{
			name: "unsupported declaration",
			args: args{
				text:  ".reserved { $name } {{Hello, {$name}!}}",
				input: nil,
			},
			expectedExecuteErr: ErrUnsupportedStatement,
		},
		{
			name: "duplicate declaration",
			args: args{
				text:  ".input {$var} .input {$var} {{Redeclaration of the same variable}}",
				input: map[string]any{"var": "22"},
			},
			expectedExecuteErr: ErrDuplicateDeclaration,
		},
		{
			name: "duplicate declaration",
			args: args{
				text:  ".local $var = {$ext} .input {$var} {{Redeclaration of a local variable}}",
				input: map[string]any{"ext": "22"},
			},
			expectedExecuteErr: ErrDuplicateDeclaration,
		},
		{
			name: "Selection Error No Annotation",
			args: args{
				text:  ".match {$n} 0 {{no apples}} 1 {{apple}} * {{apples}}",
				input: map[string]any{"n": "1"},
			},
			expectedExecuteErr: ErrMissingSelectorAnnotation,
		},
		{
			name: "Selection with Reversed Annotation",
			args: args{
				text:  ".match {$count ^string} one {{Category match}} 1 {{Exact match}} *   {{Other match}}",
				input: map[string]any{"count": "1"},
			},
			expectedExecuteErr: ErrUnsupportedExpression,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.args.text)
			if tt.expectedParseErr != nil {
				require.ErrorIs(t, err, tt.expectedParseErr)
				return
			}

			for _, f := range tt.fn {
				template.AddFunc(f)
			}

			_, err = template.Sprint(tt.args.input)
			require.ErrorIs(t, err, tt.expectedExecuteErr)
		})
	}
}
